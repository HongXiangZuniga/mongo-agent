// Implementa los tests del caso de uso de autenticación (pkg/auth), usando
// fakes de los puertos de salida (sin infraestructura real).
package test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/HongXiangZuniga/mongo-agent/pkg/auth"
)

type fakeUserRepo struct {
	user auth.User
	err  error
}

func (f fakeUserRepo) FindByUsername(ctx context.Context, username string) (auth.User, error) {
	return f.user, f.err
}

type fakeHasher struct {
	err error
}

func (f fakeHasher) Compare(hashedPassword, plaintext string) error {
	return f.err
}

func TestAuthenticate_ValidCredentialsReturnsUser(t *testing.T) {
	svc := auth.NewService(
		fakeUserRepo{user: auth.User{Username: "admin", PasswordHash: "h"}},
		fakeHasher{err: nil},
	)

	u, err := svc.Authenticate(context.Background(), "admin", "admin123")

	require.NoError(t, err)
	assert.Equal(t, "admin", u.Username)
	assert.Equal(t, "h", u.PasswordHash)
}

func TestAuthenticate_WrongPasswordReturnsInvalidCredentials(t *testing.T) {
	svc := auth.NewService(
		fakeUserRepo{user: auth.User{Username: "admin", PasswordHash: "h"}},
		fakeHasher{err: errors.New("mismatch")},
	)

	_, err := svc.Authenticate(context.Background(), "admin", "wrong")

	require.Error(t, err)
	assert.True(t, errors.Is(err, auth.ErrInvalidCredentials))
}

func TestAuthenticate_UnknownUserReturnsInvalidCredentials(t *testing.T) {
	svc := auth.NewService(
		fakeUserRepo{err: auth.ErrUserNotFound},
		fakeHasher{err: nil},
	)

	_, err := svc.Authenticate(context.Background(), "ghost", "whatever")

	require.Error(t, err)
	// Hacia afuera se devuelve ErrInvalidCredentials, NUNCA ErrUserNotFound,
	// para no revelar si falló el usuario o la contraseña.
	assert.True(t, errors.Is(err, auth.ErrInvalidCredentials))
	assert.False(t, errors.Is(err, auth.ErrUserNotFound))
}
