// Package web implementa el adapter de entrada HTTP para la interfaz de
// chat basada en HTML + htmx.
package web

import "time"

// CookieConfig define los parámetros de la cookie de autenticación web.
type CookieConfig struct {
	CookieName string
	MaxAge     time.Duration
	Secure     bool
	APIToken   string
}
