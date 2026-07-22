// Package security implementa adapters de salida relacionados con seguridad.
// Aquí vive el verificador de contraseñas basado en bcrypt, que satisface el
// puerto auth.PasswordHasher.
package security

import (
	"golang.org/x/crypto/bcrypt"

	"github.com/HongXiangZuniga/mongo-agent/pkg/auth"
)

type bcryptHasher struct{}

// NewBcryptHasher construye un auth.PasswordHasher que verifica contraseñas
// contra hashes bcrypt.
func NewBcryptHasher() auth.PasswordHasher { return bcryptHasher{} }

func (bcryptHasher) Compare(hashedPassword, plaintext string) error {
	return bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(plaintext))
}
