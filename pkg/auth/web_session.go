package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"time"
)

// WebSessionManager es el puerto de entrada (driving) para las sesiones web.
// El adapter web (handlers + middleware) lo consume para crear, validar y
// revocar sesiones sin conocer el almacenamiento subyacente.
type WebSessionManager interface {
	Create(ctx context.Context, username string) (sessionID string, err error)
	Validate(ctx context.Context, sessionID string) (WebSession, error)
	Revoke(ctx context.Context, sessionID string) error
}

type webSessionService struct {
	store WebSessionStore
	now   func() time.Time
}

// NewWebSessionManager construye el caso de uso de sesiones web sobre un
// WebSessionStore. El reloj se inyecta como campo para poder testearlo.
func NewWebSessionManager(store WebSessionStore) WebSessionManager {
	return &webSessionService{store: store, now: func() time.Time { return time.Now().UTC() }}
}

// Create genera un session ID opaco de 256 bits (hex, 64 caracteres) con
// crypto/rand, lo persiste asociado al username y su timestamp, y lo devuelve.
func (s *webSessionService) Create(ctx context.Context, username string) (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	sessionID := hex.EncodeToString(buf)
	if err := s.store.Save(ctx, sessionID, WebSession{Username: username, CreatedAt: s.now()}); err != nil {
		return "", err
	}
	return sessionID, nil
}

// Validate devuelve ErrWebSessionNotFound si el session ID es vacío; en otro
// caso delega en el store (que también mapea la ausencia/expiración a ese
// error).
func (s *webSessionService) Validate(ctx context.Context, sessionID string) (WebSession, error) {
	if sessionID == "" {
		return WebSession{}, ErrWebSessionNotFound
	}
	return s.store.Find(ctx, sessionID)
}

// Revoke borra la sesión del store. Un session ID vacío no es error (no hay
// nada que revocar).
func (s *webSessionService) Revoke(ctx context.Context, sessionID string) error {
	if sessionID == "" {
		return nil
	}
	return s.store.Delete(ctx, sessionID)
}
