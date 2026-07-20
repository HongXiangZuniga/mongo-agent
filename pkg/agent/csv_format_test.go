// Tests internos para la conversión JSON → CSV del paquete agent.
package agent

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJSONArrayToCSV_FlatObjects(t *testing.T) {
	input := `[{"sku":"a-1","price":100},{"sku":"b-2","price":250.5}]`

	csvOut, err := jsonArrayToCSV(input)
	require.NoError(t, err)
	assert.Equal(t, "price,sku\n100,a-1\n250.5,b-2\n", csvOut)
}

func TestJSONArrayToCSV_UnionOfColumns(t *testing.T) {
	input := `[{"a":1},{"b":"x","a":2}]`

	csvOut, err := jsonArrayToCSV(input)
	require.NoError(t, err)
	assert.Equal(t, "a,b\n1,\n2,x\n", csvOut)
}

func TestJSONArrayToCSV_NestedValuesAsCompactJSON(t *testing.T) {
	input := `[{"name":"order-1","items":[{"qty":2}],"ok":true,"note":null}]`

	csvOut, err := jsonArrayToCSV(input)
	require.NoError(t, err)
	assert.Equal(t, "items,name,note,ok\n\"[{\"\"qty\"\":2}]\",order-1,,true\n", csvOut)
}

func TestJSONArrayToCSV_EmptyArray(t *testing.T) {
	csvOut, err := jsonArrayToCSV(`[]`)
	require.NoError(t, err)
	assert.Equal(t, "\n", csvOut)
}

func TestJSONArrayToCSV_InvalidJSON(t *testing.T) {
	_, err := jsonArrayToCSV(`{not an array}`)
	require.Error(t, err)
}
