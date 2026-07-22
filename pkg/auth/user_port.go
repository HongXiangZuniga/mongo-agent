package auth

import (
	"context"
	"errors"
)

// UserRepository es el puerto de salida (driven) hacia la base de usuarios.
// INVARIANTE: solo lee usuarios; jamás escribe, actualiza ni borra.
type UserRepository interface {
	FindByUsername(ctx context.Context, username string) (User, error)
}

// ErrUserNotFound lo devuelve el repositorio cuando no existe el usuario.
var ErrUserNotFound = errors.New("user not found")
