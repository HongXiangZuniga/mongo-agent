//go:build integration

// Test de integración contra el proveedor real de OpenCode Zen.
package integration

import (
	"context"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/HongXiangZuniga/agente-inaricards/pkg/agent"
	"github.com/HongXiangZuniga/agente-inaricards/pkg/llm/opencodezen"
)

func TestCompleteChat_RealOpenCodeZen(t *testing.T) {
	apiKey := os.Getenv("OPENCODE_API_KEY")
	if apiKey == "" {
		t.Skip("OPENCODE_API_KEY no configurada")
	}

	baseURL := os.Getenv("OPENCODE_BASE_URL")
	if baseURL == "" {
		baseURL = "https://opencode.ai/zen/v1"
	}
	model := os.Getenv("OPENCODE_MODEL")
	if model == "" {
		model = "deepseek-v4-flash-free"
	}

	client := opencodezen.NewClient(
		&http.Client{Timeout: 60 * time.Second},
		baseURL,
		apiKey,
		model,
	)

	req := agent.LLMRequest{
		Messages: []agent.Message{
			{
				Role:      agent.RoleSystem,
				Content:   "Eres un asistente conciso.",
				CreatedAt: time.Now(),
			},
			{
				Role:      agent.RoleUser,
				Content:   "Responde solo: ok",
				CreatedAt: time.Now(),
			},
		},
	}

	resp, err := client.CompleteChat(context.Background(), req)

	require.NoError(t, err)
	assert.NotEmpty(t, resp.Message.Content)
}
