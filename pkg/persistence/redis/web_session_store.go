// Package redis implementa el adapter de salida (driven) para la memoria de
// conversación por sesión, respaldado por Redis.
package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	goredis "github.com/redis/go-redis/v9"

	"github.com/HongXiangZuniga/mongo-agent/pkg/auth"
)

// webSessionDoc es el DTO JSON persistido en Redis para cada sesión web.
type webSessionDoc struct {
	Username  string `json:"username"`
	CreatedAt int64  `json:"created_at"`
}

// webSessionStore implementa auth.WebSessionStore sobre Redis. Usa un namespace
// propio (web_session:*) para no chocar con las keys de conversación
// (session:*, sessions:index).
type webSessionStore struct {
	client *goredis.Client
	ttl    time.Duration
}

// NewWebSessionStore construye el almacén de sesiones web respaldado por Redis.
// El TTL se aplica de forma uniforme a cada sesión guardada.
func NewWebSessionStore(client *goredis.Client, ttl time.Duration) auth.WebSessionStore {
	return &webSessionStore{client, ttl}
}

// key construye la clave de Redis para una sesión web.
func (s *webSessionStore) key(id string) string {
	return fmt.Sprintf("web_session:%s", id)
}

// Save serializa la sesión a JSON y la guarda con expiración TTL.
func (s *webSessionStore) Save(ctx context.Context, sessionID string, session auth.WebSession) error {
	data, err := json.Marshal(webSessionDoc{
		Username:  session.Username,
		CreatedAt: session.CreatedAt.Unix(),
	})
	if err != nil {
		return fmt.Errorf("failed to marshal web session: %w", err)
	}
	if err := s.client.Set(ctx, s.key(sessionID), data, s.ttl).Err(); err != nil {
		return fmt.Errorf("failed to save web session: %w", err)
	}
	return nil
}

// Find recupera la sesión web. Mapea la ausencia/expiración de la key
// (goredis.Nil) a auth.ErrWebSessionNotFound.
func (s *webSessionStore) Find(ctx context.Context, sessionID string) (auth.WebSession, error) {
	data, err := s.client.Get(ctx, s.key(sessionID)).Bytes()
	if err == goredis.Nil {
		return auth.WebSession{}, auth.ErrWebSessionNotFound
	}
	if err != nil {
		return auth.WebSession{}, fmt.Errorf("failed to get web session: %w", err)
	}
	var doc webSessionDoc
	if err := json.Unmarshal(data, &doc); err != nil {
		return auth.WebSession{}, fmt.Errorf("failed to unmarshal web session: %w", err)
	}
	return auth.WebSession{Username: doc.Username, CreatedAt: time.Unix(doc.CreatedAt, 0)}, nil
}

// Delete borra la sesión web (revocación del lado servidor).
func (s *webSessionStore) Delete(ctx context.Context, sessionID string) error {
	if err := s.client.Del(ctx, s.key(sessionID)).Err(); err != nil {
		return fmt.Errorf("failed to delete web session: %w", err)
	}
	return nil
}
