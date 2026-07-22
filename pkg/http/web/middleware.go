// Package web implementa el adapter de entrada HTTP para la interfaz de
// chat basada en HTML + htmx.
package web

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/HongXiangZuniga/mongo-agent/pkg/auth"
)

// CookieAuthMiddleware protege las rutas web validando el session ID de la
// cookie contra el almacén de sesiones (Redis) en cada request. Ya no compara
// contra un secreto estático: una cookie cuyo valor no corresponde a una
// sesión activa es rechazada.
func CookieAuthMiddleware(cfg CookieConfig, sessions auth.WebSessionManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		sessionID, err := c.Cookie(cfg.CookieName)
		if err != nil || sessionID == "" {
			denyAuth(c)
			return
		}
		if _, err := sessions.Validate(c.Request.Context(), sessionID); err != nil {
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
