// Package web implementa el adapter de entrada HTTP para la interfaz de
// chat basada en HTML + htmx.
package web

import "github.com/gin-gonic/gin"

// RegisterRoutes registra las rutas de la interfaz web sobre un router gin
// existente.
func RegisterRoutes(r *gin.Engine, h WebHandlers, cfg CookieConfig) {
	r.GET("/web/static/htmx.min.js", serveHTMXAsset)
	r.GET("/web/static/app.css", serveAppCSS)
	r.GET("/web/static/fonts/:filename", serveFont)
	r.GET("/web/login", h.LoginForm)
	r.POST("/web/login", h.Login)
	r.POST("/web/logout", h.Logout)

	authed := r.Group("/web")
	authed.Use(CookieAuthMiddleware(cfg))
	authed.GET("", h.Index)
	authed.POST("/tabs", h.NewTab)
	authed.GET("/tabs/:sessionId", h.SwitchTab)
	authed.POST("/tabs/:sessionId/messages", h.SendMessage)
	authed.DELETE("/tabs/:sessionId", h.CloseTab)
}
