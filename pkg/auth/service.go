package auth

import "context"

type service struct {
	users  UserRepository
	hasher PasswordHasher
}

// NewService construye el caso de uso de autenticación a partir de sus puertos
// de salida: el repositorio de usuarios y el verificador de contraseñas.
func NewService(users UserRepository, hasher PasswordHasher) Authenticator {
	return &service{users: users, hasher: hasher}
}

// Authenticate valida usuario/contraseña. Devuelve siempre ErrInvalidCredentials
// tanto si el usuario no existe como si la contraseña es incorrecta, para no
// revelar cuál de los dos falló.
func (s *service) Authenticate(ctx context.Context, username, password string) (User, error) {
	u, err := s.users.FindByUsername(ctx, username)
	if err != nil {
		return User{}, ErrInvalidCredentials
	}
	if err := s.hasher.Compare(u.PasswordHash, password); err != nil {
		return User{}, ErrInvalidCredentials
	}
	return u, nil
}
