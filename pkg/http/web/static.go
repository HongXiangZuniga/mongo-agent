// Package web implementa el adapter de entrada HTTP para la interfaz de
// chat basada en HTML + htmx.
package web

import (
	"embed"
	"net/http"

	"github.com/gin-gonic/gin"
)

//go:embed static/htmx.min.js static/app.css static/fonts/*.woff2
var staticFS embed.FS

// serveHTMXAsset sirve el asset vendored de htmx con cacheo.
func serveHTMXAsset(c *gin.Context) {
	c.Header("Cache-Control", "public, max-age=86400")
	c.Header("Content-Type", "application/javascript; charset=utf-8")
	c.FileFromFS("static/htmx.min.js", http.FS(staticFS))
}

// serveAppCSS sirve la hoja de estilos propia de la interfaz web.
func serveAppCSS(c *gin.Context) {
	c.Header("Cache-Control", "public, max-age=31536000, immutable")
	c.Header("Content-Type", "text/css; charset=utf-8")
	c.FileFromFS("static/app.css", http.FS(staticFS))
}

// serveFont sirve las fuentes tipográficas vendorizadas, validando el
// nombre contra la lista cerrada de archivos conocidos.
func serveFont(c *gin.Context) {
	filename := c.Param("filename")
	if filename != "inter-400.woff2" && filename != "inter-600.woff2" {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}
	c.Header("Cache-Control", "public, max-age=31536000, immutable")
	c.Header("Content-Type", "font/woff2")
	c.FileFromFS("static/fonts/"+filename, http.FS(staticFS))
}
