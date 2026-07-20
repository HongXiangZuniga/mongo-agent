// Package web implementa el adapter de entrada HTTP para la interfaz de
// chat basada en HTML + htmx.
package web

import "strings"

// Tipos de segmento de un mensaje renderizable.
const (
	segmentKindText  = "text"
	segmentKindTable = "table"
)

// messageSegment es un fragmento del contenido de un mensaje: texto plano
// o una tabla (parseada desde su notación Markdown con pipes).
type messageSegment struct {
	Kind   string
	Text   string
	Header []string
	Rows   [][]string
}

// parseMessageContent segmenta el contenido de un mensaje en texto plano y
// tablas Markdown estilo GFM (línea de cabecera con pipes, línea separadora
// con guiones, y cero o más filas). Todo el texto queda disponible como
// string plano para que html/template lo auto-escape al renderizar; las
// celdas de la tabla también son strings planos escapados por la plantilla.
// Nunca se genera HTML en esta capa.
func parseMessageContent(content string) []messageSegment {
	lines := strings.Split(content, "\n")
	segments := make([]messageSegment, 0, 1)
	textBuf := make([]string, 0, len(lines))

	flushText := func() {
		if len(textBuf) == 0 {
			return
		}
		segments = append(segments, messageSegment{
			Kind: segmentKindText,
			Text: strings.Join(textBuf, "\n"),
		})
		textBuf = textBuf[:0]
	}

	i := 0
	for i < len(lines) {
		// Una tabla empieza con una línea de cabecera con pipes seguida de
		// la línea separadora (|---|...).
		if i+1 < len(lines) && isTableRow(lines[i]) && isTableSeparator(lines[i+1]) {
			header := splitTableRow(lines[i])
			if len(header) > 0 {
				flushText()
				rows := make([][]string, 0)
				j := i + 2
				for j < len(lines) && isTableRow(lines[j]) {
					cells := splitTableRow(lines[j])
					if len(cells) > 0 {
						rows = append(rows, cells)
					}
					j++
				}
				segments = append(segments, messageSegment{
					Kind:   segmentKindTable,
					Header: header,
					Rows:   rows,
				})
				i = j
				continue
			}
		}
		textBuf = append(textBuf, lines[i])
		i++
	}
	flushText()
	return segments
}

// isTableRow indica si una línea parece una fila de tabla Markdown:
// contiene al menos un pipe y no es una línea separadora.
func isTableRow(line string) bool {
	trimmed := strings.TrimSpace(line)
	return strings.Contains(trimmed, "|") && !isTableSeparator(line)
}

// isTableSeparator indica si una línea es el separador de cabecera de una
// tabla Markdown (solo pipes, guiones, dos puntos y espacios, con al menos
// un guion).
func isTableSeparator(line string) bool {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" || !strings.Contains(trimmed, "-") {
		return false
	}
	for _, r := range trimmed {
		if r != '|' && r != '-' && r != ':' && r != ' ' {
			return false
		}
	}
	return true
}

// splitTableRow divide una fila de tabla Markdown en celdas recortadas,
// tolerando pipes opcionales al inicio y al final de la línea.
func splitTableRow(line string) []string {
	trimmed := strings.TrimSpace(line)
	trimmed = strings.TrimPrefix(trimmed, "|")
	trimmed = strings.TrimSuffix(trimmed, "|")
	parts := strings.Split(trimmed, "|")
	cells := make([]string, 0, len(parts))
	for _, part := range parts {
		cells = append(cells, strings.TrimSpace(part))
	}
	return cells
}
