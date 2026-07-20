// Tests internos para el parser de tablas Markdown del paquete web.
package web

import (
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
