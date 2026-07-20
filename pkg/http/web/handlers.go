// Package web implementa el adapter de entrada HTTP para la interfaz de
// chat basada en HTML + htmx.
package web

import (
	"context"
	"crypto/subtle"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/HongXiangZuniga/mongo-agent/pkg/agent"
	"github.com/HongXiangZuniga/mongo-agent/pkg/utils"
)

// emptyStateFallback se usa cuando el descubrimiento de colecciones falla o
// la base de datos no tiene ninguna colección disponible.
const emptyStateFallback = "Escribe algo para empezar la conversación."

// WebHandlers define los manejadores HTTP de la interfaz web.
type WebHandlers interface {
	Index(*gin.Context)
	LoginForm(*gin.Context)
	Login(*gin.Context)
	Logout(*gin.Context)
	NewTab(*gin.Context)
	SwitchTab(*gin.Context)
	SendMessage(*gin.Context)
	CloseTab(*gin.Context)
}

type webPort struct {
	agentService agent.AgentService
	cookieCfg    CookieConfig
	scrubber     *utils.SecretScrubber
}

// NewWebHandler construye el handler de la interfaz web.
func NewWebHandler(agentService agent.AgentService, cookieCfg CookieConfig, scrubber *utils.SecretScrubber) WebHandlers {
	return &webPort{agentService, cookieCfg, scrubber}
}

// Index renderiza la página completa de chat.
func (p *webPort) Index(c *gin.Context) {
	summaries, err := p.agentService.ListSessions(c.Request.Context())
	if err != nil {
		c.String(http.StatusServiceUnavailable, "servicio no disponible")
		return
	}

	var active agent.SessionSummary
	var messages []agent.Message

	if len(summaries) == 0 {
		active, err = p.agentService.CreateSession(c.Request.Context())
		if err != nil {
			c.String(http.StatusServiceUnavailable, "servicio no disponible")
			return
		}
		messages = []agent.Message{}
	} else {
		active = summaries[0]
		messages, _ = p.agentService.GetConversation(c.Request.Context(), active.SessionID)
	}

	tabSummaries := summaries
	if len(summaries) == 0 {
		tabSummaries = []agent.SessionSummary{active}
	}

	vm := pageViewModel{
		Tabs:      toTabViewModels(tabSummaries, active.SessionID),
		ActiveTab: p.chatPanel(c.Request.Context(), active.SessionID, messages, ""),
	}

	c.Status(http.StatusOK)
	c.Header("Content-Type", "text/html; charset=utf-8")
	_ = render(c.Writer, "layout", vm)
}

// LoginForm muestra el formulario de login.
func (p *webPort) LoginForm(c *gin.Context) {
	c.Status(http.StatusOK)
	c.Header("Content-Type", "text/html; charset=utf-8")
	_ = render(c.Writer, "login", struct{ Error string }{})
}

// Login valida el token y fija la cookie de autenticación.
func (p *webPort) Login(c *gin.Context) {
	token := c.PostForm("token")
	if subtle.ConstantTimeCompare([]byte(token), []byte(p.cookieCfg.APIToken)) != 1 {
		c.Status(http.StatusUnauthorized)
		c.Header("Content-Type", "text/html; charset=utf-8")
		_ = render(c.Writer, "login", struct{ Error string }{Error: "Token inválido"})
		return
	}

	c.SetSameSite(http.SameSiteStrictMode)
	c.SetCookie(
		p.cookieCfg.CookieName,
		token,
		int(p.cookieCfg.MaxAge.Seconds()),
		"/web",
		"",
		p.cookieCfg.Secure,
		true,
	)
	c.Redirect(http.StatusSeeOther, "/web")
}

// Logout borra la cookie de autenticación.
func (p *webPort) Logout(c *gin.Context) {
	c.SetSameSite(http.SameSiteStrictMode)
	c.SetCookie(
		p.cookieCfg.CookieName,
		"",
		-1,
		"/web",
		"",
		p.cookieCfg.Secure,
		true,
	)
	c.Redirect(http.StatusSeeOther, "/web/login")
}

// NewTab crea una nueva pestaña y devuelve el panel de chat + la barra OOB.
func (p *webPort) NewTab(c *gin.Context) {
	ctx := c.Request.Context()

	newSession, err := p.agentService.CreateSession(ctx)
	if err != nil {
		renderChatPanel(c, p.chatPanel(ctx, "", nil, "no se pudo crear la pestaña"))
		return
	}

	existing, _ := p.agentService.ListSessions(ctx)

	panel := p.chatPanel(ctx, newSession.SessionID, nil, "")
	allTabs := append(
		[]agent.SessionSummary{newSession},
		existing...,
	)
	tabs := toTabViewModels(allTabs, newSession.SessionID)

	renderFragmentPair(c, panel, tabs)
}

// SwitchTab cambia a una pestaña existente y devuelve el panel junto con la
// barra de pestañas actualizada (para reflejar cuál queda activa).
func (p *webPort) SwitchTab(c *gin.Context) {
	sessionID := c.Param("sessionId")
	if !utils.IsValidSessionID(sessionID) {
		c.String(http.StatusBadRequest, "invalid session id")
		return
	}

	ctx := c.Request.Context()
	messages, _ := p.agentService.GetConversation(ctx, sessionID)
	sessions, _ := p.agentService.ListSessions(ctx)
	renderFragmentPair(
		c,
		p.chatPanel(ctx, sessionID, messages, ""),
		toTabViewModels(sessions, sessionID),
	)
}

// SendMessage envía un mensaje al agente y actualiza panel + barra.
func (p *webPort) SendMessage(c *gin.Context) {
	sessionID := c.Param("sessionId")
	if !utils.IsValidSessionID(sessionID) {
		c.String(http.StatusBadRequest, "invalid session id")
		return
	}

	message := strings.TrimSpace(c.PostForm("message"))
	if message == "" {
		messages, _ := p.agentService.GetConversation(c.Request.Context(), sessionID)
		renderChatPanel(c, p.chatPanel(c.Request.Context(), sessionID, messages, "el mensaje no puede estar vacío"))
		return
	}

	_, err := p.agentService.Ask(c.Request.Context(), agent.Question{
		SessionID: sessionID,
		Text:      message,
	})
	if err != nil {
		log.Printf("[SendMessage] error: %v", p.sanitizeLogError(err))
		messages, _ := p.agentService.GetConversation(c.Request.Context(), sessionID)
		renderChatPanel(c, p.chatPanel(c.Request.Context(), sessionID, messages, "no se pudo procesar el mensaje"))
		return
	}

	messages, _ := p.agentService.GetConversation(c.Request.Context(), sessionID)
	sessions, _ := p.agentService.ListSessions(c.Request.Context())
	renderFragmentPair(
		c,
		p.chatPanel(c.Request.Context(), sessionID, messages, ""),
		toTabViewModels(sessions, sessionID),
	)
}

// CloseTab cierra la pestaña solicitada y activa la siguiente más reciente.
func (p *webPort) CloseTab(c *gin.Context) {
	sessionID := c.Param("sessionId")
	if !utils.IsValidSessionID(sessionID) {
		c.String(http.StatusBadRequest, "invalid session id")
		return
	}

	ctx := c.Request.Context()
	_ = p.agentService.CloseSession(ctx, sessionID)

	remaining, _ := p.agentService.ListSessions(ctx)

	var active agent.SessionSummary
	var messages []agent.Message

	if len(remaining) > 0 {
		active = remaining[0]
		messages, _ = p.agentService.GetConversation(ctx, active.SessionID)
	} else {
		active, _ = p.agentService.CreateSession(ctx)
		messages = []agent.Message{}
	}

	tabs := remaining
	if len(tabs) == 0 {
		tabs = []agent.SessionSummary{active}
	}

	renderFragmentPair(
		c,
		p.chatPanel(ctx, active.SessionID, messages, ""),
		toTabViewModels(tabs, active.SessionID),
	)
}

// chatPanel construye un chatPanelViewModel. Si no hay error y la sesión no
// tiene mensajes, añade un hint de bienvenida basado en el autodescubrimiento
// de colecciones de MongoDB, para que el usuario sepa qué puede preguntar sin
// gastar una llamada al LLM.
func (p *webPort) chatPanel(ctx context.Context, sessionID string, messages []agent.Message, errMsg string) chatPanelViewModel {
	panel := chatPanelViewModel{
		SessionID: sessionID,
		Messages:  toMessageViewModels(messages),
		Error:     errMsg,
	}
	if errMsg == "" && len(messages) == 0 {
		panel.EmptyStateHint = p.emptyStateHint(ctx)
	}
	return panel
}

// emptyStateHint expone las colecciones descubiertas en MongoDB como texto de
// bienvenida. Si el descubrimiento falla o no hay colecciones, usa un texto
// genérico — nunca expone el error subyacente al cliente.
func (p *webPort) emptyStateHint(ctx context.Context) string {
	names, err := p.agentService.ListAvailableCollections(ctx)
	if err != nil || len(names) == 0 {
		return emptyStateFallback
	}
	return fmt.Sprintf(
		"La base de datos tiene disponibles estas colecciones: %s. Pregúntame algo sobre sus datos.",
		strings.Join(names, ", "),
	)
}

// sanitizeLogError sanea un error antes de registrarlo en logs: redacta
// credenciales de MongoDB por patrón genérico y cualquier secreto conocido
// configurado en el scrubber.
func (p *webPort) sanitizeLogError(err error) string {
	msg := utils.RedactMongoCredentials(err.Error())
	if p.scrubber != nil {
		msg = p.scrubber.Scrub(msg)
	}
	return msg
}

// renderChatPanel responde con el fragmento de panel de chat.
func renderChatPanel(c *gin.Context, panel chatPanelViewModel) {
	c.Status(http.StatusOK)
	c.Header("Content-Type", "text/html; charset=utf-8")
	_ = render(c.Writer, "chat_panel", panel)
}

// renderFragmentPair responde con el panel de chat y la barra de pestañas
// OOB concatenados.
func renderFragmentPair(c *gin.Context, panel chatPanelViewModel, tabs []tabViewModel) {
	c.Status(http.StatusOK)
	c.Header("Content-Type", "text/html; charset=utf-8")
	_ = render(c.Writer, "chat_panel", panel)
	_ = render(c.Writer, "tab_bar_oob", tabs)
}
