// Este archivo contiene utilidades de redacción de secretos. Ninguna función
// aquí debe registrar, loguear ni devolver el valor crudo de un secreto.
package utils

import (
	"regexp"
	"strings"
)

// SecretScrubber redacta por coincidencia exacta un conjunto de valores
// secretos conocidos (cargados una sola vez desde variables de entorno)
// de cualquier texto antes de que se registre en logs.
type SecretScrubber struct {
	secrets []string
}

// NewSecretScrubber construye un SecretScrubber a partir de los valores de
// secretos dados, ignorando los que estén vacíos.
func NewSecretScrubber(secrets ...string) *SecretScrubber {
	nonEmpty := make([]string, 0, len(secrets))
	for _, secret := range secrets {
		if secret != "" {
			nonEmpty = append(nonEmpty, secret)
		}
	}
	return &SecretScrubber{secrets: nonEmpty}
}

// Scrub reemplaza cada aparición de un secreto conocido en text por
// "[REDACTED]". Si text no contiene ningún secreto, se devuelve sin cambios.
func (s *SecretScrubber) Scrub(text string) string {
	if s == nil {
		return text
	}
	result := text
	for _, secret := range s.secrets {
		if secret == "" {
			continue
		}
		result = strings.ReplaceAll(result, secret, "[REDACTED]")
	}
	return result
}

var mongoCredentialsPattern = regexp.MustCompile(`([a-zA-Z][a-zA-Z0-9+.-]*://)([^:/@\s]+):([^@/\s]+)@`)

// RedactMongoCredentials reemplaza cualquier subcadena con forma
// "esquema://usuario:contraseña@" (p. ej. mongodb://user:pass@host,
// mongodb+srv://user:pass@cluster, o cualquier URI con user:pass@) por
// "esquema://[REDACTED]:[REDACTED]@", sin necesidad de conocer el valor
// exacto de la URI configurada. Es un mecanismo de defensa en profundidad
// independiente y complementario a SecretScrubber.
func RedactMongoCredentials(text string) string {
	return mongoCredentialsPattern.ReplaceAllString(text, "${1}[REDACTED]:[REDACTED]@")
}
