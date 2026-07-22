package auth

import (
	"context"
	"errors"
	"time"
)

// WebSession es el registro de una sesión de autenticación web. Modela quién
// inició sesión (Username) y cuándo (CreatedAt). Es un concepto de auth,
// distinto de la sesión de conversación del agente (agent.SessionStore).
type WebSession struct {
	Username  string
	CreatedAt time.Time
}

// WebSessionStore es el puerto de salida (driven) para persistir sesiones web.
// La política de expiración (TTL) la aplica el adapter que lo implementa (no se
// pasa en cada Save), igual que el store de conversaciones fija el TTL en su
// constructor.
type WebSessionStore interface {
	Save(ctx context.Context, sessionID string, session WebSession) error
	Find(ctx context.Context, sessionID string) (WebSession, error)
	Delete(ctx context.Context, sessionID string) error
}

// ErrWebSessionNotFound lo devuelve el store cuando la sesión no existe o expiró.
var ErrWebSessionNotFound = errors.New("web session not found")
