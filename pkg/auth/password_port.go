package auth

// PasswordHasher es el puerto de salida (driven) para verificar contraseñas.
type PasswordHasher interface {
	// Compare devuelve nil si plaintext corresponde a hashedPassword,
	// y un error en caso contrario.
	Compare(hashedPassword, plaintext string) error
}
