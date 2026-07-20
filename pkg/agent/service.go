// Package agent contiene el dominio, los puertos de salida y el caso de uso
// del agente de preguntas en lenguaje natural sobre MongoDB.
package agent

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/HongXiangZuniga/mongo-agent/pkg/utils"
)

// systemPromptLeakMinMatchChars es el número mínimo de caracteres
// consecutivos del system prompt que deben aparecer en una respuesta final
// para considerarla una fuga.
const systemPromptLeakMinMatchChars = 60

// systemPromptLeakRefusalMessage sustituye a cualquier respuesta final en la
// que se detecte una fuga verbatim del system prompt.
const systemPromptLeakRefusalMessage = "No puedo compartir esa información."

// detectSystemPromptLeak reporta si answer contiene una ventana deslizante
// de al menos minMatchChars caracteres consecutivos de systemPrompt
// (comparación insensible a mayúsculas/minúsculas).
func detectSystemPromptLeak(answer, systemPrompt string, minMatchChars int) bool {
	promptRunes := []rune(systemPrompt)
	if len(promptRunes) < minMatchChars {
		return false
	}
	answerLower := strings.ToLower(answer)
	for i := 0; i+minMatchChars <= len(promptRunes); i++ {
		window := strings.ToLower(string(promptRunes[i : i+minMatchChars]))
		if strings.Contains(answerLower, window) {
			return true
		}
	}
	return false
}

// AgentService es el puerto de entrada (driving) del caso de uso.
type AgentService interface {
	Ask(ctx context.Context, q Question) (Answer, error)
	ListSessions(ctx context.Context) ([]SessionSummary, error)
	CreateSession(ctx context.Context) (SessionSummary, error)
	CloseSession(ctx context.Context, sessionID string) error
	GetConversation(ctx context.Context, sessionID string) ([]Message, error)
	ListAvailableCollections(ctx context.Context) ([]string, error)
}

// service implementa AgentService con un bucle de tool-calling contra el LLM.
type service struct {
	llm                   LLMClient
	mongo                 ReadOnlyMongoRepository
	sessions              SessionStore
	maxIterations         int
	requestTimeout        time.Duration
	sampleSize            int
	maxResultLimit        int
	systemPrompt          string
	sessionTitleMaxLength int
}

// NewAgentService crea una instancia de AgentService.
func NewAgentService(
	llm LLMClient,
	mongo ReadOnlyMongoRepository,
	sessions SessionStore,
	maxIterations int,
	requestTimeout time.Duration,
	sampleSize int,
	maxResultLimit int,
	systemPrompt string,
	sessionTitleMaxLength int,
) AgentService {
	return &service{
		llm,
		mongo,
		sessions,
		maxIterations,
		requestTimeout,
		sampleSize,
		maxResultLimit,
		systemPrompt,
		sessionTitleMaxLength,
	}
}

// Ask procesa una pregunta del usuario usando el LLM y las herramientas de
// solo lectura de MongoDB.
func (s *service) Ask(ctx context.Context, q Question) (Answer, error) {
	if q.Text == "" {
		return Answer{}, utils.ErrEmptyQuestion()
	}
	q.Text = utils.SanitizeUserText(q.Text)
	if strings.TrimSpace(q.Text) == "" {
		return Answer{}, utils.ErrEmptyQuestion()
	}

	askCtx, cancel := context.WithTimeout(ctx, s.requestTimeout)
	defer cancel()

	history, err := s.sessions.GetHistory(askCtx, q.SessionID)
	if err != nil {
		return Answer{}, utils.ErrSessionStoreUnavailable(err)
	}

	userMsg := Message{
		Role:      RoleUser,
		Content:   q.Text,
		CreatedAt: time.Now(),
	}
	history = append(history, userMsg)
	if err := s.sessions.AppendMessage(askCtx, q.SessionID, userMsg); err != nil {
		return Answer{}, utils.ErrSessionStoreUnavailable(err)
	}
	if err := s.sessions.TouchSession(
		askCtx,
		q.SessionID,
		deriveTitle(q.Text, s.sessionTitleMaxLength),
		time.Now(),
	); err != nil {
		return Answer{}, utils.ErrSessionStoreUnavailable(err)
	}

	// Si hay system prompt, lo antecedemos al historial enviado al LLM en
	// cada iteración sin persistirlo.
	toolCallsExecuted := 0
	tools := BuildToolDefinitions(s.sampleSize, s.maxResultLimit)

	for i := 0; i < s.maxIterations; i++ {
		llmMessages := s.prependSystemPrompt(history)
		req := LLMRequest{
			Messages: llmMessages,
			Tools:    tools,
		}
		resp, err := s.llm.CompleteChat(askCtx, req)
		if err != nil {
			if errors.Is(err, context.DeadlineExceeded) {
				return Answer{}, utils.ErrRequestTimeout()
			}
			return Answer{}, utils.ErrLLMProviderUnavailable(err)
		}

		if len(resp.Message.ToolCalls) == 0 &&
			detectSystemPromptLeak(resp.Message.Content, s.systemPrompt, systemPromptLeakMinMatchChars) {
			resp.Message.Content = systemPromptLeakRefusalMessage
			log.Printf("[Ask] possible system prompt leak detected, session=%s", q.SessionID)
		}

		if err := s.sessions.AppendMessage(askCtx, q.SessionID, resp.Message); err != nil {
			return Answer{}, utils.ErrSessionStoreUnavailable(err)
		}
		history = append(history, resp.Message)

		if len(resp.Message.ToolCalls) == 0 {
			return Answer{
				SessionID:         q.SessionID,
				Text:              resp.Message.Content,
				ToolCallsExecuted: toolCallsExecuted,
			}, nil
		}

		for _, call := range resp.Message.ToolCalls {
			result, isErr := DispatchToolCall(
				askCtx,
				s.mongo,
				s.sampleSize,
				s.maxResultLimit,
				call,
			)
			if isErr {
				result = fmt.Sprintf("error: %s", result)
			}
			toolMsg := Message{
				Role:       RoleTool,
				ToolCallID: call.ID,
				Content:    result,
				CreatedAt:  time.Now(),
			}
			if err := s.sessions.AppendMessage(askCtx, q.SessionID, toolMsg); err != nil {
				return Answer{}, utils.ErrSessionStoreUnavailable(err)
			}
			history = append(history, toolMsg)
			toolCallsExecuted++
		}
	}

	if askCtx.Err() == context.DeadlineExceeded {
		return Answer{}, utils.ErrRequestTimeout()
	}
	return Answer{}, utils.ErrToolLoopExceeded()
}

// prependSystemPrompt antepone el system prompt a history en cada request
// al LLM. El mensaje system nunca se persiste en Redis.
func (s *service) prependSystemPrompt(history []Message) []Message {
	if s.systemPrompt == "" {
		return history
	}
	messages := make([]Message, 0, len(history)+1)
	messages = append(messages, Message{
		Role:      RoleSystem,
		Content:   s.systemPrompt,
		CreatedAt: time.Now(),
	})
	messages = append(messages, history...)
	return messages
}

// deriveTitle genera un título para la sesión a partir del texto de la
// pregunta, respetando el límite en runes.
func deriveTitle(text string, maxLength int) string {
	trimmed := strings.TrimSpace(text)
	runes := []rune(trimmed)
	if len(runes) == 0 {
		return "Nueva conversación"
	}
	if len(runes) <= maxLength {
		return trimmed
	}
	return string(runes[:maxLength]) + "…"
}

// ListSessions delega en el almacén de sesiones.
func (s *service) ListSessions(ctx context.Context) ([]SessionSummary, error) {
	return s.sessions.ListSessions(ctx)
}

// CreateSession genera un nuevo session_id sin persistirlo todavía.
func (s *service) CreateSession(ctx context.Context) (SessionSummary, error) {
	sessionID := uuid.NewString()
	return SessionSummary{
		SessionID:    sessionID,
		Title:        "Nueva conversación",
		LastActivity: time.Now(),
	}, nil
}

// CloseSession delega en el almacén de sesiones.
func (s *service) CloseSession(ctx context.Context, sessionID string) error {
	return s.sessions.ClearSession(ctx, sessionID)
}

// GetConversation devuelve los mensajes visibles para la UI (user y
// assistant), omitiendo tool y system. Los mensajes assistant con Content
// vacío (artefactos intermedios de tool-calling, que solo portan ToolCalls)
// también se omiten por no ser contenido presentable.
func (s *service) GetConversation(ctx context.Context, sessionID string) ([]Message, error) {
	history, err := s.sessions.GetHistory(ctx, sessionID)
	if err != nil {
		return nil, utils.ErrSessionStoreUnavailable(err)
	}

	messages := make([]Message, 0, len(history))
	for _, msg := range history {
		if msg.Role == RoleUser {
			messages = append(messages, msg)
			continue
		}
		if msg.Role == RoleAssistant && msg.Content != "" {
			messages = append(messages, msg)
		}
	}
	return messages, nil
}

// ListAvailableCollections expone los nombres de las colecciones
// descubiertas en MongoDB, sin pasar por el LLM. Se usa para mostrarle al
// usuario qué datos existen antes de que haga su primera pregunta.
func (s *service) ListAvailableCollections(ctx context.Context) ([]string, error) {
	infos, err := s.mongo.ListCollections(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list collections: %w", err)
	}
	names := make([]string, 0, len(infos))
	for _, info := range infos {
		names = append(names, info.Name)
	}
	return names, nil
}
