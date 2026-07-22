package auth

import "context"

// Authenticator es el puerto de entrada (driving) del caso de uso de login.
type Authenticator interface {
	Authenticate(ctx context.Context, username, password string) (User, error)
}
