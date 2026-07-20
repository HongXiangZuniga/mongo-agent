// Package utils contiene utilidades compartidas entre pkg/agent y sus adapters.
package utils

import (
	"regexp"
	"strings"
	"unicode"
)

var sessionIDPattern = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

// IsValidSessionID indica si id es un identificador de sesión seguro.
func IsValidSessionID(id string) bool {
	return id != "" && sessionIDPattern.MatchString(id)
}

// SanitizeUserText elimina caracteres de control no imprimibles de text,
// preservando salto de línea (\n) y tabulador (\t). No trunca ni escapa
// el texto de ningún otro modo.
func SanitizeUserText(text string) string {
	return strings.Map(func(r rune) rune {
		if r == '\n' || r == '\t' {
			return r
		}
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, text)
}
