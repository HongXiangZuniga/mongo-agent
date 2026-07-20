// Package agent contiene el dominio, los puertos de salida y el caso de uso
// del agente de preguntas en lenguaje natural sobre MongoDB.
package agent

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
)

// jsonArrayToCSV convierte un resultado JSON (array de objetos) a CSV.
//
// Las columnas son la unión de las claves de primer nivel de todos los
// documentos, en orden de primera aparición (estable). Los valores
// primitivos se serializan tal cual; los objetos o arrays anidados se
// serializan como JSON compacto dentro de la celda. Un array vacío produce
// solo la línea de cabecera vacía (sin filas).
//
// El formato CSV es significativamente más compacto en tokens que el JSON
// equivalente para resultados tabulares, lo que reduce el riesgo de agotar
// las iteraciones del bucle agéntico, y es directamente exportable por el
// usuario final.
func jsonArrayToCSV(jsonResult string) (string, error) {
	var docs []map[string]any
	if err := json.Unmarshal([]byte(jsonResult), &docs); err != nil {
		return "", fmt.Errorf("failed to parse json result: %w", err)
	}

	columns := make([]string, 0)
	seen := make(map[string]bool)
	for _, doc := range docs {
		// Orden determinista de claves dentro de cada documento para que la
		// unión sea estable entre ejecuciones con el mismo resultado.
		keys := make([]string, 0, len(doc))
		for k := range doc {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			if !seen[k] {
				seen[k] = true
				columns = append(columns, k)
			}
		}
	}

	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	if err := w.Write(columns); err != nil {
		return "", fmt.Errorf("failed to write csv header: %w", err)
	}
	for _, doc := range docs {
		row := make([]string, len(columns))
		for i, col := range columns {
			row[i] = csvCellValue(doc[col])
		}
		if err := w.Write(row); err != nil {
			return "", fmt.Errorf("failed to write csv row: %w", err)
		}
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return "", fmt.Errorf("failed to flush csv: %w", err)
	}
	return buf.String(), nil
}

// csvCellValue serializa un valor JSON deserializado a su representación de
// celda CSV: primitivos tal cual, null como vacío, y objetos/arrays como
// JSON compacto.
func csvCellValue(v any) string {
	switch value := v.(type) {
	case nil:
		return ""
	case string:
		return value
	case bool:
		return strconv.FormatBool(value)
	case float64:
		return strconv.FormatFloat(value, 'f', -1, 64)
	default:
		data, err := json.Marshal(value)
		if err != nil {
			return ""
		}
		return string(data)
	}
}
