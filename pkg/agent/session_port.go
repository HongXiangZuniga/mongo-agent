// Package agent contiene el dominio, los puertos de salida y el caso de uso
// del agente de preguntas en lenguaje natural sobre MongoDB.
package agent

import (
	"context"
	"time"
)

// SessionSummary resume una sesión de conversación indexada.
type SessionSummary struct {
	SessionID    string
	Title        string
	LastActivity time.Time
}

// SessionStore es el puerto de salida para la memoria de conversación por
// sesión.
type SessionStore interface {
	AppendMessage(ctx context.Context, sessionID string, msg Message) error
	GetHistory(ctx context.Context, sessionID string) ([]Message, error)
	ClearSession(ctx context.Context, sessionID string) error
	ListSessions(ctx context.Context) ([]SessionSummary, error)
	TouchSession(ctx context.Context, sessionID string, title string, at time.Time) error
}
