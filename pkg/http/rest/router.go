// Package rest implementa el adapter de entrada (driving) HTTP usando gin.
package rest

import (
	"crypto/subtle"
	"net/http"

	"github.com/gin-gonic/gin"
)

// NewHandler crea y configura el router gin con el endpoint POST /ask.
func NewHandler(agentHandler AgentHandlers, apiToken string) *gin.Engine {
	r := gin.Default()
	RegisterRoutes(r, agentHandler, apiToken)
	return r
}

// RegisterRoutes registra las rutas REST sobre un router gin existente.
func RegisterRoutes(r *gin.Engine, agentHandler AgentHandlers, apiToken string) {
	r.POST("/ask", TokenAuthMiddleware(apiToken), agentHandler.AskQuestion)
}

// TokenAuthMiddleware verifica que el header Authorization coincida con el
// token requerido.
func TokenAuthMiddleware(requiredToken string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if subtle.ConstantTimeCompare([]byte(c.GetHeader("Authorization")), []byte(requiredToken)) != 1 {
			c.AbortWithStatusJSON(
				http.StatusUnauthorized,
				Response{
					Code:    http.StatusUnauthorized,
					Message: "invalid or missing API token",
				},
			)
			return
		}
		c.Next()
	}
}
