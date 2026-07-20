// Package web implementa el adapter de entrada HTTP para la interfaz de
// chat basada en HTML + htmx.
package web

import (
	"crypto/subtle"
	"net/http"

	"github.com/gin-gonic/gin"
)

// CookieAuthMiddleware protege las rutas web verificando la cookie de
// autenticación.
func CookieAuthMiddleware(cfg CookieConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		token, err := c.Cookie(cfg.CookieName)
		if err != nil || subtle.ConstantTimeCompare([]byte(token), []byte(cfg.APIToken)) != 1 {
			denyAuth(c)
			return
		}
		c.Next()
	}
}

// denyAuth rechaza una petición no autenticada, adaptando la respuesta a
// peticiones htmx vs peticiones normales.
func denyAuth(c *gin.Context) {
	if c.GetHeader("HX-Request") == "true" {
		c.Header("HX-Redirect", "/web/login")
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}
	c.Redirect(http.StatusSeeOther, "/web/login")
	c.Abort()
}
