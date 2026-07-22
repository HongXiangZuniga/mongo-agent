// Implementa el test real (sin mock) del adapter de contraseñas bcrypt.
package test

import (
	"testing"

	"golang.org/x/crypto/bcrypt"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/HongXiangZuniga/mongo-agent/pkg/security"
)

func TestBcryptHasher_CompareMatchesAndRejects(t *testing.T) {
	hash, err := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
	require.NoError(t, err)

	hasher := security.NewBcryptHasher()

	assert.NoError(t, hasher.Compare(string(hash), "admin123"))
	assert.Error(t, hasher.Compare(string(hash), "wrong"))
}
