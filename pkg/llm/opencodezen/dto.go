// Package opencodezen implementa el adapter de salida (driven) hacia el
// proveedor de LLM OpenCode Zen (endpoint OpenAI-compatible).
package opencodezen

// chatCompletionRequest representa el cuerpo de una petición de chat
// completions en formato OpenAI.
type chatCompletionRequest struct {
	Model    string        `json:"model"`
	Messages []chatMessage `json:"messages"`
	Tools    []chatTool    `json:"tools,omitempty"`
}

// chatMessage representa un mensaje dentro de la petición/respuesta.
type chatMessage struct {
	Role       string         `json:"role"`
	Content    string         `json:"content"`
	ToolCalls  []chatToolCall `json:"tool_calls,omitempty"`
	ToolCallID string         `json:"tool_call_id,omitempty"`
}

// chatTool describe una función disponible para el modelo.
type chatTool struct {
	Type     string       `json:"type"`
	Function chatFunction `json:"function"`
}

// chatFunction describe la firma de una función.
type chatFunction struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters"`
}

// chatToolCall representa una invocación de función devuelta por el modelo.
type chatToolCall struct {
	ID       string           `json:"id"`
	Type     string           `json:"type"`
	Function chatFunctionCall `json:"function"`
}

// chatFunctionCall contiene los argumentos de una invocación de función.
type chatFunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// chatCompletionResponse representa la respuesta del endpoint de chat
// completions.
type chatCompletionResponse struct {
	Choices []chatChoice `json:"choices"`
}

// chatChoice es una opción dentro de la respuesta.
type chatChoice struct {
	Message      chatMessage `json:"message"`
	FinishReason string      `json:"finish_reason"`
}
