//go:build integration

// Test de integración contra una instancia Redis real.
package integration

import (
	"context"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	goredis "github.com/redis/go-redis/v9"

	"github.com/HongXiangZuniga/mongo-agent/pkg/agent"
	redisadapter "github.com/HongXiangZuniga/mongo-agent/pkg/persistence/redis"
)

type redisTestStore struct {
	store  agent.SessionStore
	client *goredis.Client
}

func newRedisTestStore(t *testing.T) (redisTestStore, context.Context) {
	addr := os.Getenv("REDIS_ADDR")
	if addr == "" {
		t.Skip("REDIS_ADDR no configurada")
	}

	dbValue := os.Getenv("REDIS_DB")
	if dbValue == "" {
		dbValue = "0"
	}
	db, err := strconv.Atoi(dbValue)
	require.NoError(t, err)

	client := goredis.NewClient(&goredis.Options{
		Addr:     addr,
		Password: os.Getenv("REDIS_PASSWORD"),
		DB:       db,
	})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	require.NoError(t, client.Ping(ctx).Err())

	t.Cleanup(func() {
		cancel()
		client.Close()
	})

	return redisTestStore{store: redisadapter.NewSessionStore(client, time.Hour), client: client}, ctx
}

func TestSessionStore_AppendAndGetHistory_RealRedis(t *testing.T) {
	store, ctx := newRedisTestStore(t)
	sessionID := uuid.NewString()

	msg1 := agent.Message{
		Role:      agent.RoleUser,
		Content:   "hola",
		CreatedAt: time.Now(),
	}
	msg2 := agent.Message{
		Role:      agent.RoleAssistant,
		Content:   "hola, ¿en qué puedo ayudarte?",
		CreatedAt: time.Now(),
	}

	require.NoError(t, store.store.AppendMessage(ctx, sessionID, msg1))
	require.NoError(t, store.store.AppendMessage(ctx, sessionID, msg2))

	history, err := store.store.GetHistory(ctx, sessionID)
	require.NoError(t, err)
	require.Len(t, history, 2)
	assert.Equal(t, msg1.Content, history[0].Content)
	assert.Equal(t, msg2.Content, history[1].Content)

	require.NoError(t, store.store.ClearSession(ctx, sessionID))
	history, err = store.store.GetHistory(ctx, sessionID)
	require.NoError(t, err)
	assert.Empty(t, history)
}

// findSession busca una sesión por ID en el listado. Los tests de
// integración corren sobre un Redis compartido (puede contener sesiones de
// la app en desarrollo), así que las aserciones siempre son relativas a los
// IDs creados por el propio test, nunca sobre el estado global del índice.
func findSession(sessions []agent.SessionSummary, sessionID string) (agent.SessionSummary, bool) {
	for _, s := range sessions {
		if s.SessionID == sessionID {
			return s, true
		}
	}
	return agent.SessionSummary{}, false
}

func TestSessionStore_TouchSession_SetsTitleOnFirstCall(t *testing.T) {
	store, ctx := newRedisTestStore(t)
	sessionID := uuid.NewString()
	now := time.Now()
	t.Cleanup(func() { _ = store.store.ClearSession(context.Background(), sessionID) })

	require.NoError(t, store.store.TouchSession(ctx, sessionID, "Mi sesión", now))

	sessions, err := store.store.ListSessions(ctx)
	require.NoError(t, err)
	session, found := findSession(sessions, sessionID)
	require.True(t, found, "la sesión creada debe aparecer en el listado")
	assert.Equal(t, "Mi sesión", session.Title)
	assert.Equal(t, now.Unix(), session.LastActivity.Unix())
}

func TestSessionStore_TouchSession_PreservesTitleOnSubsequentCalls(t *testing.T) {
	store, ctx := newRedisTestStore(t)
	sessionID := uuid.NewString()
	now := time.Now()
	t.Cleanup(func() { _ = store.store.ClearSession(context.Background(), sessionID) })

	require.NoError(t, store.store.TouchSession(ctx, sessionID, "Primer título", now))
	require.NoError(t, store.store.TouchSession(ctx, sessionID, "Segundo título", now.Add(time.Minute)))

	sessions, err := store.store.ListSessions(ctx)
	require.NoError(t, err)
	session, found := findSession(sessions, sessionID)
	require.True(t, found)
	assert.Equal(t, "Primer título", session.Title)
	assert.Equal(t, now.Add(time.Minute).Unix(), session.LastActivity.Unix())
}

func TestSessionStore_ListSessions_OrderedByMostRecentFirst(t *testing.T) {
	store, ctx := newRedisTestStore(t)
	s1 := uuid.NewString()
	s2 := uuid.NewString()
	s3 := uuid.NewString()
	now := time.Now()
	t.Cleanup(func() {
		_ = store.store.ClearSession(context.Background(), s1)
		_ = store.store.ClearSession(context.Background(), s2)
		_ = store.store.ClearSession(context.Background(), s3)
	})

	require.NoError(t, store.store.TouchSession(ctx, s1, "Reciente", now))
	require.NoError(t, store.store.TouchSession(ctx, s2, "Medio", now.Add(-time.Minute)))
	require.NoError(t, store.store.TouchSession(ctx, s3, "Antiguo", now.Add(-2*time.Minute)))

	sessions, err := store.store.ListSessions(ctx)
	require.NoError(t, err)

	// Verifica el orden relativo de las tres sesiones creadas dentro del
	// listado global (que puede contener otras sesiones ajenas al test).
	positions := make(map[string]int, 3)
	for i, s := range sessions {
		if s.SessionID == s1 || s.SessionID == s2 || s.SessionID == s3 {
			positions[s.SessionID] = i
		}
	}
	require.Len(t, positions, 3, "las tres sesiones creadas deben aparecer en el listado")
	assert.Less(t, positions[s1], positions[s2], "la más reciente debe ir antes")
	assert.Less(t, positions[s2], positions[s3], "la intermedia debe ir antes que la más antigua")
}

func TestSessionStore_ListSessions_LazilyRemovesExpiredEntries(t *testing.T) {
	store, ctx := newRedisTestStore(t)
	sessionID := uuid.NewString()
	now := time.Now()
	t.Cleanup(func() { _ = store.store.ClearSession(context.Background(), sessionID) })

	ttlStore := redisadapter.NewSessionStore(store.client, time.Second)
	require.NoError(t, ttlStore.TouchSession(ctx, sessionID, "Expirará", now))

	time.Sleep(2 * time.Second)

	sessions, err := ttlStore.ListSessions(ctx)
	require.NoError(t, err)
	_, found := findSession(sessions, sessionID)
	assert.False(t, found, "la sesión expirada no debe aparecer en el listado")
}

func TestSessionStore_ClearSession_RemovesIndexEntry(t *testing.T) {
	store, ctx := newRedisTestStore(t)
	sessionID := uuid.NewString()
	now := time.Now()
	t.Cleanup(func() { _ = store.store.ClearSession(context.Background(), sessionID) })

	require.NoError(t, store.store.TouchSession(ctx, sessionID, "Borrar", now))
	sessions, err := store.store.ListSessions(ctx)
	require.NoError(t, err)
	_, found := findSession(sessions, sessionID)
	require.True(t, found)

	require.NoError(t, store.store.ClearSession(ctx, sessionID))
	sessions, err = store.store.ListSessions(ctx)
	require.NoError(t, err)
	_, found = findSession(sessions, sessionID)
	assert.False(t, found, "tras ClearSession la sesión no debe aparecer en el listado")
}
