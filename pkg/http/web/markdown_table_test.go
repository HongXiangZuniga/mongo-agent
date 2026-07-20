// Tests internos para el parser de tablas Markdown del paquete web.
package web

import (
	"encoding/base64"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseMessageContent_PlainTextOnly(t *testing.T) {
	segments := parseMessageContent("hola\n¿en qué puedo ayudarte?")

	require.Len(t, segments, 1)
	assert.Equal(t, segmentKindText, segments[0].Kind)
	assert.Equal(t, "hola\n¿en qué puedo ayudarte?", segments[0].Text)
}

func TestParseMessageContent_TableBetweenText(t *testing.T) {
	content := "Estos son los resultados:\n| sku | ventas |\n|---|---|\n| a-1 | 20 |\n| b-2 | 15 |\nFin del informe."

	segments := parseMessageContent(content)

	require.Len(t, segments, 3)
	assert.Equal(t, segmentKindText, segments[0].Kind)
	assert.Equal(t, "Estos son los resultados:", segments[0].Text)

	assert.Equal(t, segmentKindTable, segments[1].Kind)
	assert.Equal(t, []string{"sku", "ventas"}, segments[1].Header)
	require.Len(t, segments[1].Rows, 2)
	assert.Equal(t, []string{"a-1", "20"}, segments[1].Rows[0])
	assert.Equal(t, []string{"b-2", "15"}, segments[1].Rows[1])

	assert.Equal(t, segmentKindText, segments[2].Kind)
	assert.Equal(t, "Fin del informe.", segments[2].Text)
}

func TestParseMessageContent_TableWithoutBodyRows(t *testing.T) {
	content := "| a | b |\n|---|---|"

	segments := parseMessageContent(content)

	require.Len(t, segments, 1)
	assert.Equal(t, segmentKindTable, segments[0].Kind)
	assert.Equal(t, []string{"a", "b"}, segments[0].Header)
	assert.Empty(t, segments[0].Rows)
}

func TestParseMessageContent_PipeWithoutSeparatorIsNotTable(t *testing.T) {
	content := "usa a | b para comparar\notra línea normal"

	segments := parseMessageContent(content)

	require.Len(t, segments, 1)
	assert.Equal(t, segmentKindText, segments[0].Kind)
	assert.Equal(t, content, segments[0].Text)
}

func TestParseMessageContent_MultipleTables(t *testing.T) {
	content := "| a |\n|---|\n| 1 |\ntexto intermedio\n| b |\n|---|\n| 2 |"

	segments := parseMessageContent(content)

	require.Len(t, segments, 3)
	assert.Equal(t, segmentKindTable, segments[0].Kind)
	assert.Equal(t, segmentKindText, segments[1].Kind)
	assert.Equal(t, "texto intermedio", segments[1].Text)
	assert.Equal(t, segmentKindTable, segments[2].Kind)
}

func TestParseMessageContent_SeparatorWithColons(t *testing.T) {
	content := "| a | b |\n|:---|---:|\n| 1 | 2 |"

	segments := parseMessageContent(content)

	require.Len(t, segments, 1)
	assert.Equal(t, segmentKindTable, segments[0].Kind)
	require.Len(t, segments[0].Rows, 1)
}

func decodeCSVDataURL(t *testing.T, dataURL string) string {
	t.Helper()
	const prefix = "data:text/csv;charset=utf-8;base64,"
	require.True(t, strings.HasPrefix(dataURL, prefix), "unexpected data URL: %s", dataURL)
	decoded, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(dataURL, prefix))
	require.NoError(t, err)
	return string(decoded)
}

func TestParseMessageContent_CSVFenceBecomesDownloadSegment(t *testing.T) {
	content := "Aquí tienes el resultado en CSV:\n\n```csv\nname,total\na,10\nb,20\n```\n\nEso es todo."

	segments := parseMessageContent(content)

	require.Len(t, segments, 3)
	assert.Equal(t, segmentKindText, segments[0].Kind)
	assert.Equal(t, "Aquí tienes el resultado en CSV:\n", segments[0].Text)

	csvSeg := segments[1]
	assert.Equal(t, segmentKindCSV, csvSeg.Kind)
	assert.Equal(t, "resultado.csv", csvSeg.CSVFilename)
	assert.Equal(t, 2, csvSeg.CSVRowCount)
	assert.Equal(t, "name,total\na,10\nb,20", decodeCSVDataURL(t, string(csvSeg.CSVDataURL)))

	assert.Equal(t, segmentKindText, segments[2].Kind)
	assert.Equal(t, "\nEso es todo.", segments[2].Text)
}

func TestParseMessageContent_CSVFenceCaseInsensitiveLanguageTag(t *testing.T) {
	content := "```CSV\na,b\n1,2\n```"

	segments := parseMessageContent(content)

	require.Len(t, segments, 1)
	assert.Equal(t, segmentKindCSV, segments[0].Kind)
}

func TestParseMessageContent_UnclosedCSVFenceIsPlainText(t *testing.T) {
	content := "```csv\na,b\n1,2"

	segments := parseMessageContent(content)

	require.Len(t, segments, 1)
	assert.Equal(t, segmentKindText, segments[0].Kind)
	assert.Equal(t, content, segments[0].Text)
}

func TestParseMessageContent_NonCSVFenceIsPlainText(t *testing.T) {
	content := "```json\n{\"a\":1}\n```"

	segments := parseMessageContent(content)

	require.Len(t, segments, 1)
	assert.Equal(t, segmentKindText, segments[0].Kind)
	assert.Equal(t, content, segments[0].Text)
}
