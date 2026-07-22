// Package web implementa el adapter de entrada HTTP para la interfaz de
// chat basada en HTML + htmx.
package web

import "time"

// CookieConfig define los parámetros de la cookie de autenticación web. La
// cookie transporta un session ID opaco (validado contra Redis); ya no
// transporta ningún secreto estático.
type CookieConfig struct {
	CookieName string
	MaxAge     time.Duration
	Secure     bool
}
