// Implementa los tests del caso de uso de sesión web (pkg/auth), usando un
// fake del puerto de salida WebSessionStore (sin Redis real).
package test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/HongXiangZuniga/mongo-agent/pkg/auth"
)

type fakeWebSessionStore struct {
	savedID      string
	savedSession auth.WebSession
	saveCalls    int

	findSession auth.WebSession
	findErr     error
	findCalls   int

	deletedID   string
	deleteCalls int
}

func (f *fakeWebSessionStore) Save(ctx context.Context, sessionID string, session auth.WebSession) error {
	f.saveCalls++
	f.savedID = sessionID
	f.savedSession = session
	return nil
}

func (f *fakeWebSessionStore) Find(ctx context.Context, sessionID string) (auth.WebSession, error) {
	f.findCalls++
	return f.findSession, f.findErr
}

func (f *fakeWebSessionStore) Delete(ctx context.Context, sessionID string) error {
	f.deleteCalls++
	f.deletedID = sessionID
	return nil
}

func TestWebSession_CreateStoresAndReturnsOpaqueID(t *testing.T) {
	store := &fakeWebSessionStore{}
	mgr := auth.NewWebSessionManager(store)

	id1, err := mgr.Create(context.Background(), "admin")
	require.NoError(t, err)
	assert.NotEmpty(t, id1)
	assert.Equal(t, "admin", store.savedSession.Username)
	assert.Equal(t, id1, store.savedID)

	id2, err := mgr.Create(context.Background(), "admin")
	require.NoError(t, err)
	assert.NotEmpty(t, id2)
	assert.NotEqual(t, id1, id2, "cada Create debe generar un ID distinto")
}

func TestWebSession_ValidateUnknownReturnsErr(t *testing.T) {
	store := &fakeWebSessionStore{findErr: auth.ErrWebSessionNotFound}
	mgr := auth.NewWebSessionManager(store)

	_, err := mgr.Validate(context.Background(), "abc")
	require.Error(t, err)
	assert.True(t, errors.Is(err, auth.ErrWebSessionNotFound))
	assert.Equal(t, 1, store.findCalls)

	// Un session ID vacío se rechaza sin tocar el store.
	_, err = mgr.Validate(context.Background(), "")
	require.Error(t, err)
	assert.True(t, errors.Is(err, auth.ErrWebSessionNotFound))
	assert.Equal(t, 1, store.findCalls, "Validate(\"\") no debe invocar Find")
}

func TestWebSession_RevokeDeletes(t *testing.T) {
	store := &fakeWebSessionStore{}
	mgr := auth.NewWebSessionManager(store)

	require.NoError(t, mgr.Revoke(context.Background(), "id1"))
	assert.Equal(t, 1, store.deleteCalls)
	assert.Equal(t, "id1", store.deletedID)

	// Un session ID vacío no invoca Delete.
	require.NoError(t, mgr.Revoke(context.Background(), ""))
	assert.Equal(t, 1, store.deleteCalls, "Revoke(\"\") no debe invocar Delete")
}
