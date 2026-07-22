//go:build integration

// Implementa los tests de integración del adapter de repositorio de usuarios
// (pkg/persistence/authmongo) contra un MongoDB real. Requiere el build tag
// `integration` y la variable AUTH_MONGODB_URI apuntando a una instancia Mongo.
package integration

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/HongXiangZuniga/mongo-agent/pkg/auth"
	"github.com/HongXiangZuniga/mongo-agent/pkg/persistence/authmongo"
)

func connectAuthDB(t *testing.T) *mongo.Database {
	t.Helper()
	uri := os.Getenv("AUTH_MONGODB_URI")
	if uri == "" {
		t.Skip("AUTH_MONGODB_URI not set; skipping authmongo integration test")
	}
	dbName := os.Getenv("AUTH_MONGODB_DB_NAME")
	if dbName == "" {
		dbName = "authdb_test"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	require.NoError(t, err)
	require.NoError(t, client.Ping(ctx, nil))

	t.Cleanup(func() {
		_ = client.Disconnect(context.Background())
	})
	return client.Database(dbName)
}

func TestFindByUsername_ReturnsSeededUser(t *testing.T) {
	db := connectAuthDB(t)
	coll := db.Collection("users_it")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, _ = coll.DeleteMany(ctx, bson.M{"username": "it-user"})
	_, err := coll.InsertOne(ctx, bson.M{"username": "it-user", "password_hash": "$2a$10$abcdefghijklmnopqrstuv"})
	require.NoError(t, err)
	t.Cleanup(func() { _, _ = coll.DeleteMany(context.Background(), bson.M{"username": "it-user"}) })

	repo := authmongo.NewUserRepository(db, "users_it", 5*time.Second)
	u, err := repo.FindByUsername(ctx, "it-user")

	require.NoError(t, err)
	assert.Equal(t, "it-user", u.Username)
	assert.Equal(t, "$2a$10$abcdefghijklmnopqrstuv", u.PasswordHash)
}

func TestFindByUsername_UnknownReturnsErrUserNotFound(t *testing.T) {
	db := connectAuthDB(t)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	repo := authmongo.NewUserRepository(db, "users_it", 5*time.Second)
	_, err := repo.FindByUsername(ctx, "definitely-not-here")

	require.Error(t, err)
	assert.ErrorIs(t, err, auth.ErrUserNotFound)
}
