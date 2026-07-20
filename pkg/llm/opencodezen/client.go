// Package opencodezen implementa el adapter de salida (driven) hacia el
// proveedor de LLM OpenCode Zen (endpoint OpenAI-compatible), usando un
// cliente HTTP estándar (sin SDK propietario).
//
// IMPORTANTE: el valor de apiKey nunca aparece en logs ni en mensajes de
// error.
package opencodezen

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	agent "github.com/HongXiangZuniga/agente-inaricards/pkg/agent"
)

// client implementa agent.LLMClient usando net/http.
type client struct {
	httpClient *http.Client
	baseURL    string
	apiKey     string
	model      string
}

// NewClient construye un cliente para el endpoint OpenAI-compatible de
// OpenCode Zen.
func NewClient(
	httpClient *http.Client,
	baseURL string,
	apiKey string,
	model string,
) agent.LLMClient {
	return &client{httpClient, baseURL, apiKey, model}
}

// CompleteChat envía el historial y las herramientas al LLM y devuelve la
// respuesta.
func (c *client) CompleteChat(ctx context.Context, req agent.LLMRequest) (agent.LLMResponse, error) {
	body, err := c.buildRequestBody(req)
	if err != nil {
		return agent.LLMResponse{}, fmt.Errorf("failed to build request body: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.baseURL+"/chat/completions",
		bytes.NewReader(body),
	)
	if err != nil {
		return agent.LLMResponse{}, fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	httpReq.Header.Set("Content-Type", "application/json")

	httpResp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return agent.LLMResponse{}, fmt.Errorf("failed to execute request: %w", err)
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode >= http.StatusBadRequest {
		limitedBody := io.LimitReader(httpResp.Body, 4096)
		respBytes, _ := io.ReadAll(limitedBody)
		return agent.LLMResponse{}, fmt.Errorf(
			"llm request failed with status %d: %s",
			httpResp.StatusCode,
			string(respBytes),
		)
	}

	var chatResp chatCompletionResponse
	if err := json.NewDecoder(httpResp.Body).Decode(&chatResp); err != nil {
		return agent.LLMResponse{}, fmt.Errorf("failed to decode response: %w", err)
	}
	if len(chatResp.Choices) == 0 {
		return agent.LLMResponse{}, fmt.Errorf("llm response contained no choices")
	}

	choice := chatResp.Choices[0]
	return agent.LLMResponse{
		Message:      c.mapMessage(choice.Message),
		FinishReason: choice.FinishReason,
	}, nil
}

// buildRequestBody serializa la petición del dominio al formato del
// proveedor.
func (c *client) buildRequestBody(req agent.LLMRequest) ([]byte, error) {
	chatReq := chatCompletionRequest{
		Model:    c.model,
		Messages: c.mapMessages(req.Messages),
		Tools:    c.mapTools(req.Tools),
	}
	body, err := json.Marshal(chatReq)
	if err != nil {
		return nil, err
	}
	return body, nil
}

// mapMessages transforma los mensajes del dominio a mensajes del proveedor.
func (c *client) mapMessages(messages []agent.Message) []chatMessage {
	result := make([]chatMessage, 0, len(messages))
	for _, msg := range messages {
		chatMsg := chatMessage{
			Role:       string(msg.Role),
			Content:    msg.Content,
			ToolCallID: msg.ToolCallID,
		}
		if len(msg.ToolCalls) > 0 {
			chatMsg.ToolCalls = c.mapToolCalls(msg.ToolCalls)
		}
		result = append(result, chatMsg)
	}
	return result
}

// mapMessage transforma un mensaje del proveedor al dominio.
func (c *client) mapMessage(msg chatMessage) agent.Message {
	return agent.Message{
		Role:       agent.Role(msg.Role),
		Content:    msg.Content,
		ToolCalls:  c.mapToolCallsBack(msg.ToolCalls),
		ToolCallID: msg.ToolCallID,
	}
}

// mapTools transforma las definiciones de herramientas del dominio.
func (c *client) mapTools(tools []agent.ToolDefinition) []chatTool {
	result := make([]chatTool, 0, len(tools))
	for _, tool := range tools {
		result = append(result, chatTool{
			Type: "function",
			Function: chatFunction{
				Name:        tool.Name,
				Description: tool.Description,
				Parameters:  tool.Parameters,
			},
		})
	}
	return result
}

// mapToolCalls transforma las invocaciones del dominio al proveedor.
func (c *client) mapToolCalls(calls []agent.ToolCall) []chatToolCall {
	result := make([]chatToolCall, 0, len(calls))
	for _, call := range calls {
		result = append(result, chatToolCall{
			ID:   call.ID,
			Type: "function",
			Function: chatFunctionCall{
				Name:      call.Name,
				Arguments: call.Arguments,
			},
		})
	}
	return result
}

// mapToolCallsBack transforma las invocaciones del proveedor al dominio.
func (c *client) mapToolCallsBack(calls []chatToolCall) []agent.ToolCall {
	result := make([]agent.ToolCall, 0, len(calls))
	for _, call := range calls {
		result = append(result, agent.ToolCall{
			ID:        call.ID,
			Name:      call.Function.Name,
			Arguments: call.Function.Arguments,
		})
	}
	return result
}
