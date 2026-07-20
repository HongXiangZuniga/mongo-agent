// Package web implementa el adapter de entrada HTTP para la interfaz de
// chat basada en HTML + htmx.
package web

import (
	"embed"
	"html/template"
	"io"
)

//go:embed templates/*.html
var templateFS embed.FS

var tmpl = template.Must(template.ParseFS(templateFS, "templates/*.html"))

// render ejecuta una plantilla por nombre sobre el writer.
func render(w io.Writer, name string, data any) error {
	return tmpl.ExecuteTemplate(w, name, data)
}
