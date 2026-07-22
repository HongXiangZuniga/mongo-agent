//go:build integration

// Test de integración del adapter Redis de sesión web contra un Redis real.
package integration

import (
	"context"
	"errors"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	goredis "github.com/redis/go-redis/v9"

	"github.com/HongXiangZuniga/mongo-agent/pkg/auth"
	redisadapter "github.com/HongXiangZuniga/mongo-agent/pkg/persistence/redis"
)

func TestWebSessionStore_SaveFindDeleteRoundTrip(t *testing.T) {
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

	store := redisadapter.NewWebSessionStore(client, time.Minute)
	sessionID := "test-web-session-" + strconv.FormatInt(time.Now().UnixNano(), 10)
	t.Cleanup(func() { _ = store.Delete(context.Background(), sessionID) })

	require.NoError(t, store.Save(ctx, sessionID, auth.WebSession{Username: "admin", CreatedAt: time.Now()}))

	found, err := store.Find(ctx, sessionID)
	require.NoError(t, err)
	assert.Equal(t, "admin", found.Username)

	require.NoError(t, store.Delete(ctx, sessionID))

	_, err = store.Find(ctx, sessionID)
	assert.True(t, errors.Is(err, auth.ErrWebSessionNotFound))
}
