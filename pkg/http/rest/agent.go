// Package rest implementa el adapter de entrada (driving) HTTP usando gin.
package rest

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/HongXiangZuniga/mongo-agent/pkg/agent"
	"github.com/HongXiangZuniga/mongo-agent/pkg/utils"
)

// AgentHandlers define los manejadores HTTP del agente.
type AgentHandlers interface {
	AskQuestion(*gin.Context)
}

// agentPort implementa AgentHandlers.
type agentPort struct {
	agentService agent.AgentService
	scrubber     *utils.SecretScrubber
}

// NewAgentHandler construye el handler HTTP del agente.
func NewAgentHandler(agentService agent.AgentService, scrubber *utils.SecretScrubber) AgentHandlers {
	return &agentPort{agentService, scrubber}
}

type askRequest struct {
	SessionID string `json:"session_id"`
	Question  string `json:"question"`
}

type askResponse struct {
	SessionID string `json:"session_id"`
	Answer    string `json:"answer"`
}

// AskQuestion recibe una pregunta en lenguaje natural y la delega al
// servicio del agente.
func (p *agentPort) AskQuestion(ctx *gin.Context) {
	const maxBodyBytes = 1 << 16 // 64 KB
	ctx.Request.Body = http.MaxBytesReader(ctx.Writer, ctx.Request.Body, maxBodyBytes)

	var req askRequest
	if err := ctx.ShouldBindJSON(&req); err != nil || req.Question == "" {
		ctx.JSON(
			http.StatusBadRequest,
			Response{
				Code:    http.StatusBadRequest,
				Message: "question is required",
			},
		)
		return
	}

	sessionID := req.SessionID
	if sessionID == "" {
		sessionID = uuid.NewString()
	} else if !utils.IsValidSessionID(sessionID) {
		ctx.JSON(
			http.StatusBadRequest,
			Response{
				Code:    http.StatusBadRequest,
				Message: "invalid session_id format",
			},
		)
		return
	}

	answer, err := p.agentService.Ask(
		ctx.Request.Context(),
		agent.Question{SessionID: sessionID, Text: req.Question},
	)
	if err != nil {
		log.Printf("[AskQuestion] error: %v", p.sanitizeLogError(err))
		p.respondError(ctx, err)
		return
	}

	ctx.JSON(
		http.StatusOK,
		Response{
			Code:    http.StatusOK,
			Message: "success",
			Data: askResponse{
				SessionID: answer.SessionID,
				Answer:    answer.Text,
			},
		},
	)
}

// respondError mapea errores de dominio a respuestas HTTP genéricas.
// Nunca expone err.Error() crudo al cliente.
func (p *agentPort) respondError(ctx *gin.Context, err error) {
	status, message := utils.HTTPStatusForError(err)
	ctx.JSON(status, Response{Code: status, Message: message})
}

// sanitizeLogError sanea un error antes de registrarlo en logs: redacta
// credenciales de MongoDB por patrón genérico y cualquier secreto conocido
// configurado en el scrubber.
func (p *agentPort) sanitizeLogError(err error) string {
	msg := utils.RedactMongoCredentials(err.Error())
	if p.scrubber != nil {
		msg = p.scrubber.Scrub(msg)
	}
	return msg
}
