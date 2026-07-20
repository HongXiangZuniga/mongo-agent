// Implementa los tests de DispatchToolCall.
package test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/HongXiangZuniga/agente-inaricards/pkg/agent"
)

func TestDispatchToolCall_UnknownTool(t *testing.T) {
	repo := &fakeMongoRepo{}
	call := agent.ToolCall{
		ID:        "call_x",
		Name:      "unknown_tool",
		Arguments: "{}",
	}

	result, isErr := agent.DispatchToolCall(context.Background(), repo, 5, 50, call)

	assert.True(t, isErr)
	assert.Contains(t, result, "unknown tool")
}

func TestDispatchToolCall_MalformedArguments(t *testing.T) {
	repo := &fakeMongoRepo{}
	call := agent.ToolCall{
		ID:        "call_x",
		Name:      agent.ToolQueryFind,
		Arguments: "{invalid json",
	}

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("DispatchToolCall panicked with: %v", r)
		}
	}()

	result, isErr := agent.DispatchToolCall(context.Background(), repo, 5, 50, call)

	assert.True(t, isErr)
	assert.Contains(t, result, "error parsing arguments")
}

func TestDispatchToolCall_RejectsOutStage(t *testing.T) {
	repo := &fakeMongoRepo{}
	call := agent.ToolCall{
		ID:        "call_x",
		Name:      agent.ToolQueryAggregate,
		Arguments: `{"collection":"src","pipeline":[{"$out":"otra_coleccion"}]}`,
	}

	result, isErr := agent.DispatchToolCall(context.Background(), repo, 5, 50, call)

	assert.True(t, isErr)
	assert.Contains(t, result, "$out")
	assert.Equal(t, 0, repo.aggregateCalls)
}

func TestDispatchToolCall_RejectsMergeStage(t *testing.T) {
	repo := &fakeMongoRepo{}
	call := agent.ToolCall{
		ID:        "call_x",
		Name:      agent.ToolQueryAggregate,
		Arguments: `{"collection":"src","pipeline":[{"$merge":{"into":"otra"}}]}`,
	}

	result, isErr := agent.DispatchToolCall(context.Background(), repo, 5, 50, call)

	assert.True(t, isErr)
	assert.Contains(t, result, "$merge")
	assert.Equal(t, 0, repo.aggregateCalls)
}

func TestDispatchToolCall_RejectsNestedOutStage(t *testing.T) {
	repo := &fakeMongoRepo{}
	call := agent.ToolCall{
		ID:        "call_x",
		Name:      agent.ToolQueryAggregate,
		Arguments: `{"collection":"src","pipeline":[{"$facet":{"a":[{"$out":"x"}]}}]}`,
	}

	result, isErr := agent.DispatchToolCall(context.Background(), repo, 5, 50, call)

	assert.True(t, isErr)
	assert.Contains(t, result, "$out")
	assert.Equal(t, 0, repo.aggregateCalls)
}

func TestDispatchToolCall_ForcesMaxResultLimit(t *testing.T) {
	repo := &fakeMongoRepo{}
	call := agent.ToolCall{
		ID:        "call_x",
		Name:      agent.ToolQueryFind,
		Arguments: `{"collection":"users","filter":{}}`,
	}

	_, isErr := agent.DispatchToolCall(context.Background(), repo, 5, 50, call)

	assert.False(t, isErr)
	assert.Equal(t, 1, repo.findCalls)
	assert.Equal(t, 50, repo.lastFindLimit)
}

func TestDispatchToolCall_FindWithCSVFormat(t *testing.T) {
	repo := &fakeMongoRepo{findResult: `[{"sku":"a-1","price":100},{"sku":"b-2","price":200}]`}
	call := agent.ToolCall{
		ID:        "call_x",
		Name:      agent.ToolQueryFind,
		Arguments: `{"collection":"products","filter":{},"format":"csv"}`,
	}

	result, isErr := agent.DispatchToolCall(context.Background(), repo, 5, 50, call)

	assert.False(t, isErr)
	assert.Equal(t, "price,sku\n100,a-1\n200,b-2\n", result)
}

func TestDispatchToolCall_AggregateWithCSVFormat(t *testing.T) {
	repo := &fakeMongoRepo{aggregateResult: `[{"total":5}]`}
	call := agent.ToolCall{
		ID:        "call_x",
		Name:      agent.ToolQueryAggregate,
		Arguments: `{"collection":"orders","pipeline":[],"format":"csv"}`,
	}

	result, isErr := agent.DispatchToolCall(context.Background(), repo, 5, 50, call)

	assert.False(t, isErr)
	assert.Equal(t, "total\n5\n", result)
}

func TestDispatchToolCall_RedactsMongoCredentialsInRepositoryError(t *testing.T) {
	repo := &fakeMongoRepo{
		findErr: errors.New("server selection error: mongodb+srv://dbuser:S3cr3t@cluster0.mongodb.net: context deadline exceeded"),
	}
	call := agent.ToolCall{
		ID:        "call_x",
		Name:      agent.ToolQueryFind,
		Arguments: `{"collection":"users","filter":{}}`,
	}

	result, isErr := agent.DispatchToolCall(context.Background(), repo, 5, 50, call)

	assert.True(t, isErr)
	assert.NotContains(t, result, "dbuser")
	assert.NotContains(t, result, "S3cr3t")
	assert.Contains(t, result, "[REDACTED]")
}

func TestDispatchToolCall_PreservesNonCredentialErrorContent(t *testing.T) {
	repo := &fakeMongoRepo{findErr: errors.New("collection not found")}
	call := agent.ToolCall{
		ID:        "call_x",
		Name:      agent.ToolQueryFind,
		Arguments: `{"collection":"users","filter":{}}`,
	}

	result, isErr := agent.DispatchToolCall(context.Background(), repo, 5, 50, call)

	assert.True(t, isErr)
	assert.Contains(t, result, "collection not found")
}

func TestDispatchToolCall_DefaultFormatIsJSON(t *testing.T) {
	repo := &fakeMongoRepo{findResult: `[{"sku":"a-1"}]`}
	call := agent.ToolCall{
		ID:        "call_x",
		Name:      agent.ToolQueryFind,
		Arguments: `{"collection":"products","filter":{}}`,
	}

	result, isErr := agent.DispatchToolCall(context.Background(), repo, 5, 50, call)

	assert.False(t, isErr)
	assert.Equal(t, `[{"sku":"a-1"}]`, result)
}
