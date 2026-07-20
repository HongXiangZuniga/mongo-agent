//go:build integration

// Test de integración contra un cluster MongoDB real.
package integration

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/HongXiangZuniga/mongo-agent/pkg/persistence/mongodb"
)

func TestVerifyReadOnlyGuarantee_RealCluster(t *testing.T) {
	uri := os.Getenv("MONGODB_URI")
	if uri == "" {
		t.Skip("MONGODB_URI no configurada")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	require.NoError(t, err)
	defer client.Disconnect(ctx)

	dbName := os.Getenv("MONGODB_DB_NAME")
	if dbName == "" {
		t.Skip("MONGODB_DB_NAME no configurada")
	}
	db := client.Database(dbName)

	err = mongodb.VerifyReadOnlyGuarantee(ctx, db)
	assert.NoError(t, err)
}
