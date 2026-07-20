// Package redis implementa el adapter de salida (driven) para la memoria de
// conversación por sesión, respaldado por Redis.
package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	goredis "github.com/redis/go-redis/v9"

	agent "github.com/HongXiangZuniga/agente-inaricards/pkg/agent"
)

const sessionsIndexKey = "sessions:index"

// store implementa agent.SessionStore sobre Redis.
type store struct {
	client *goredis.Client
	ttl    time.Duration
}

// NewSessionStore construye un almacén de sesiones respaldado por Redis.
func NewSessionStore(client *goredis.Client, ttl time.Duration) agent.SessionStore {
	return &store{client, ttl}
}

// sessionKey construye la clave de Redis para los mensajes de una sesión.
func (s *store) sessionKey(sessionID string) string {
	return fmt.Sprintf("session:%s:messages", sessionID)
}

// metaKey construye la clave de Redis para la metadata de una sesión.
func (s *store) metaKey(sessionID string) string {
	return fmt.Sprintf("session:%s:meta", sessionID)
}

// AppendMessage añade un mensaje al historial de la sesión.
func (s *store) AppendMessage(ctx context.Context, sessionID string, msg agent.Message) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	key := s.sessionKey(sessionID)
	_, err = s.client.Pipelined(ctx, func(pipe goredis.Pipeliner) error {
		pipe.RPush(ctx, key, data)
		pipe.Expire(ctx, key, s.ttl)
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to append message: %w", err)
	}
	return nil
}

// GetHistory devuelve el historial de mensajes de una sesión en orden
// cronológico.
func (s *store) GetHistory(ctx context.Context, sessionID string) ([]agent.Message, error) {
	key := s.sessionKey(sessionID)
	items, err := s.client.LRange(ctx, key, 0, -1).Result()
	if err != nil {
		if err == goredis.Nil {
			return []agent.Message{}, nil
		}
		return nil, fmt.Errorf("failed to get history: %w", err)
	}

	messages := make([]agent.Message, 0, len(items))
	for _, item := range items {
		var msg agent.Message
		if err := json.Unmarshal([]byte(item), &msg); err != nil {
			return nil, fmt.Errorf("failed to unmarshal message: %w", err)
		}
		messages = append(messages, msg)
	}
	return messages, nil
}

// TouchSession registra o actualiza la metadata de una sesión en el índice.
func (s *store) TouchSession(
	ctx context.Context,
	sessionID string,
	title string,
	at time.Time,
) error {
	_, err := s.client.Pipelined(ctx, func(pipe goredis.Pipeliner) error {
		pipe.HSetNX(ctx, s.metaKey(sessionID), "title", title)
		pipe.HSet(ctx, s.metaKey(sessionID), "last_activity", at.Unix())
		pipe.Expire(ctx, s.metaKey(sessionID), s.ttl)
		pipe.ZAdd(ctx, sessionsIndexKey, goredis.Z{Score: float64(at.Unix()), Member: sessionID})
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to touch session: %w", err)
	}
	return nil
}

// ListSessions devuelve las sesiones indexadas ordenadas por actividad más
// reciente primero.
func (s *store) ListSessions(ctx context.Context) ([]agent.SessionSummary, error) {
	ids, err := s.client.ZRevRange(ctx, sessionsIndexKey, 0, -1).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to list session ids: %w", err)
	}
	if len(ids) == 0 {
		return []agent.SessionSummary{}, nil
	}

	cmds := make(map[string]*goredis.MapStringStringCmd, len(ids))
	_, err = s.client.Pipelined(ctx, func(pipe goredis.Pipeliner) error {
		for _, id := range ids {
			cmds[id] = pipe.HGetAll(ctx, s.metaKey(id))
		}
		return nil
	})
	if err != nil && err != goredis.Nil {
		return nil, fmt.Errorf("failed to list sessions: %w", err)
	}

	summaries := make([]agent.SessionSummary, 0, len(ids))
	expired := make([]string, 0)
	for _, id := range ids {
		meta, _ := cmds[id].Result()
		if len(meta) == 0 {
			expired = append(expired, id)
			continue
		}

		lastActivity := time.Time{}
		if raw, ok := meta["last_activity"]; ok {
			if parsed, parseErr := strconv.ParseInt(raw, 10, 64); parseErr == nil {
				lastActivity = time.Unix(parsed, 0)
			}
		}
		summaries = append(summaries, agent.SessionSummary{
			SessionID:    id,
			Title:        meta["title"],
			LastActivity: lastActivity,
		})
	}

	if len(expired) > 0 {
		// La limpieza del índice es perezosa; no debe fallar la lectura.
		_ = s.client.ZRem(ctx, sessionsIndexKey, toAnySlice(expired)...).Err()
	}

	return summaries, nil
}

// ClearSession elimina el historial y la metadata de una sesión.
func (s *store) ClearSession(ctx context.Context, sessionID string) error {
	_, err := s.client.Pipelined(ctx, func(pipe goredis.Pipeliner) error {
		pipe.Del(ctx, s.sessionKey(sessionID))
		pipe.Del(ctx, s.metaKey(sessionID))
		pipe.ZRem(ctx, sessionsIndexKey, sessionID)
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to clear session: %w", err)
	}
	return nil
}

// toAnySlice convierte []string a []any para APIs variádicas.
func toAnySlice(ids []string) []any {
	result := make([]any, len(ids))
	for i, id := range ids {
		result[i] = id
	}
	return result
}
