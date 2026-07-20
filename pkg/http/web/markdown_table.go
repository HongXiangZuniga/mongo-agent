// Package web implementa el adapter de entrada HTTP para la interfaz de
// chat basada en HTML + htmx.
package web

import (
	"encoding/base64"
	"html/template"
	"strings"
)

// Tipos de segmento de un mensaje renderizable.
const (
	segmentKindText  = "text"
	segmentKindTable = "table"
	segmentKindCSV   = "csv"
)

// csvAttachmentFilename es el nombre de descarga fijo para los bloques CSV
// devueltos por el agente (no depende de datos del usuario ni del LLM).
const csvAttachmentFilename = "resultado.csv"

// messageSegment es un fragmento del contenido de un mensaje: texto plano,
// una tabla (parseada desde su notación Markdown con pipes), o un bloque CSV
// (parseado desde una cerca de código ```csv) ofrecido como descarga.
type messageSegment struct {
	Kind        string
	Text        string
	Header      []string
	Rows        [][]string
	CSVFilename string
	CSVRowCount int
	CSVDataURL  template.URL
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
		// Un bloque CSV empieza con una cerca ```csv y termina con una
		// cerca ``` en su propia línea.
		if isCSVFenceStart(lines[i]) {
			end := findCSVFenceEnd(lines, i+1)
			if end != -1 {
				flushText()
				csvText := strings.Join(lines[i+1:end], "\n")
				segments = append(segments, newCSVSegment(csvText))
				i = end + 1
				continue
			}
		}
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

// isCSVFenceStart indica si line es la línea de apertura de un bloque de
// código ```csv (insensible a mayúsculas/minúsculas en el identificador de
// lenguaje).
func isCSVFenceStart(line string) bool {
	trimmed := strings.TrimSpace(line)
	if !strings.HasPrefix(trimmed, "```") {
		return false
	}
	lang := strings.ToLower(strings.TrimSpace(strings.TrimPrefix(trimmed, "```")))
	return lang == "csv"
}

// findCSVFenceEnd busca, a partir de from, la línea que cierra una cerca de
// código (``` sola en su línea) y devuelve su índice, o -1 si no la
// encuentra (cerca sin cerrar: el contenido se trata como texto plano).
func findCSVFenceEnd(lines []string, from int) int {
	for i := from; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "```" {
			return i
		}
	}
	return -1
}

// newCSVSegment construye un segmento de descarga CSV a partir del
// contenido crudo entre las cercas de código. El archivo se ofrece como un
// data URI en base64 (sin endpoint ni estado en el servidor); nunca se usa
// interpolación directa que dependa del filtro de URLs de html/template.
func newCSVSegment(csvText string) messageSegment {
	return messageSegment{
		Kind:        segmentKindCSV,
		CSVFilename: csvAttachmentFilename,
		CSVRowCount: csvDataRowCount(csvText),
		CSVDataURL:  csvDataURL(csvText),
	}
}

// csvDataRowCount cuenta las filas de datos de un CSV (líneas no vacías
// menos la cabecera).
func csvDataRowCount(csvText string) int {
	lines := strings.Split(strings.TrimRight(csvText, "\n"), "\n")
	nonEmpty := 0
	for _, l := range lines {
		if strings.TrimSpace(l) != "" {
			nonEmpty++
		}
	}
	if nonEmpty == 0 {
		return 0
	}
	return nonEmpty - 1
}

// csvDataURL codifica csvText como un data URI base64 de tipo text/csv. Se
// tipa como template.URL (en vez de string) para que html/template lo trate
// como una URL ya vetada y no la reescriba a "#ZgotmplZ": el esquema "data:"
// no está en la lista de esquemas seguros por defecto del autoescaper, pero
// aquí el contenido lo construimos nosotros mismos con codificación base64
// (nunca interpolando texto crudo del LLM sin codificar en la URL).
func csvDataURL(csvText string) template.URL {
	encoded := base64.StdEncoding.EncodeToString([]byte(csvText))
	return template.URL("data:text/csv;charset=utf-8;base64," + encoded)
}
