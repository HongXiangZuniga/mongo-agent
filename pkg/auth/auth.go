// Package auth contiene el dominio, los puertos y el caso de uso del login de usuario/contraseña. No importa infraestructura: no depende de drivers de base de datos, del verificador de contraseñas ni de variables de entorno; esas dependencias entran por adapters externos que implementan sus puertos.
package auth

import "errors"

// User es un usuario del sistema de login. PasswordHash almacena el hash de
// la contraseña (el algoritmo de hashing lo aporta el adapter PasswordHasher).
type User struct {
	Username     string
	PasswordHash string
}

// ErrInvalidCredentials se devuelve cuando usuario o contraseña no son válidos.
// No distingue entre "usuario inexistente" y "contraseña incorrecta".
var ErrInvalidCredentials = errors.New("invalid credentials")
