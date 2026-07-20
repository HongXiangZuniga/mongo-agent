// Package agent contiene el dominio, los puertos de salida y el caso de uso
// del agente de preguntas en lenguaje natural sobre MongoDB.
package agent

import "context"

// ToolDefinition describe una herramienta expuesta al LLM siguiendo un
// esquema JSON Schema de tipo objeto.
type ToolDefinition struct {
	Name        string
	Description string
	Parameters  map[string]any
}

// LLMRequest encapsula el historial de mensajes y las herramientas disponibles.
type LLMRequest struct {
	Messages []Message
	Tools    []ToolDefinition
}

// LLMResponse contiene la respuesta del proveedor de LLM.
type LLMResponse struct {
	Message      Message
	FinishReason string
}

// LLMClient es el puerto de salida hacia un proveedor de LLM.
type LLMClient interface {
	CompleteChat(ctx context.Context, req LLMRequest) (LLMResponse, error)
}
