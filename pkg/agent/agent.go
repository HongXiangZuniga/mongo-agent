// Package agent contiene el dominio, los puertos de salida y el caso de uso
// del agente de preguntas en lenguaje natural sobre MongoDB.
//
// Regla de dependencia (hexagonal): este paquete NO importa gin,
// mongo-driver, redis ni ningún cliente HTTP concreto.
package agent

import "time"

// Role representa el rol emisor de un mensaje en la conversación.
type Role string

// Roles soportados por el agente y los proveedores de LLM.
const (
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleTool      Role = "tool"
	RoleSystem    Role = "system"
)

// ToolCall representa una invocación de herramienta solicitada por el LLM.
// Arguments contiene el JSON crudo de los argumentos, sin parsear.
type ToolCall struct {
	ID        string
	Name      string
	Arguments string
}

// Message es un mensaje dentro del contexto de conversación de una sesión.
// ToolCallID solo se utiliza cuando Role == RoleTool.
type Message struct {
	Role       Role
	Content    string
	ToolCalls  []ToolCall
	ToolCallID string
	CreatedAt  time.Time
}

// Question es la pregunta del usuario al agente.
type Question struct {
	SessionID string
	Text      string
}

// Answer es la respuesta final producida por el agente.
type Answer struct {
	SessionID         string
	Text              string
	ToolCallsExecuted int
}
