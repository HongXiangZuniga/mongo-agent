// Package utils contiene errores de dominio compartidos entre pkg/agent y
// sus adapters.
//
// IMPORTANTE: ninguna de estas funciones interpola credenciales
// (URIs, tokens, API keys) en el mensaje de error.
package utils

import (
	"errors"
	"fmt"
	"net/http"
)

var (
	sentinelEmptyQuestion    = errors.New("question is required")
	sentinelToolLoopExceeded = errors.New(
		"agent did not produce a final answer within the maximum number of tool-call iterations",
	)
	sentinelRequestTimeout          = errors.New("request exceeded the maximum allowed processing time")
	sentinelLLMProviderUnavailable  = errors.New("llm provider unavailable")
	sentinelSessionStoreUnavailable = errors.New("session store unavailable")
)

// ErrEmptyQuestion indica que no se recibió texto de pregunta.
func ErrEmptyQuestion() error {
	return sentinelEmptyQuestion
}

// ErrToolLoopExceeded indica que el agente no logró una respuesta final en
// el número máximo de iteraciones permitidas.
func ErrToolLoopExceeded() error {
	return sentinelToolLoopExceeded
}

// ErrRequestTimeout indica que la solicitud superó el tiempo máximo de
// procesamiento.
func ErrRequestTimeout() error {
	return sentinelRequestTimeout
}

// ErrCollectionNotFound indica que la colección consultada no existe.
func ErrCollectionNotFound(name string) error {
	return fmt.Errorf("collection %q not found", name)
}

// ErrDisallowedPipelineStage indica que el pipeline contiene un stage no
// permitido para un agente de solo lectura.
func ErrDisallowedPipelineStage(stage string) error {
	return fmt.Errorf("aggregation stage %q is not allowed: read-only agent", stage)
}

// ErrSessionStoreUnavailable envuelve un error del almacén de sesiones.
func ErrSessionStoreUnavailable(cause error) error {
	if cause == nil {
		return sentinelSessionStoreUnavailable
	}
	return fmt.Errorf("%w: %w", sentinelSessionStoreUnavailable, cause)
}

// ErrLLMProviderUnavailable envuelve un error del proveedor de LLM.
func ErrLLMProviderUnavailable(cause error) error {
	if cause == nil {
		return sentinelLLMProviderUnavailable
	}
	return fmt.Errorf("%w: %w", sentinelLLMProviderUnavailable, cause)
}

// IsLLMProviderUnavailable indica si err, o alguno de sus errores envueltos,
// fue producido por ErrLLMProviderUnavailable.
func IsLLMProviderUnavailable(err error) bool {
	return errors.Is(err, sentinelLLMProviderUnavailable)
}

// IsSessionStoreUnavailable indica si err, o alguno de sus errores envueltos,
// fue producido por ErrSessionStoreUnavailable.
func IsSessionStoreUnavailable(err error) bool {
	return errors.Is(err, sentinelSessionStoreUnavailable)
}

// ErrMongoUserNotReadOnly indica que el usuario de MongoDB pudo escribir,
// violando la garantía de solo lectura.
func ErrMongoUserNotReadOnly() error {
	return errors.New(
		"configured mongodb user is NOT read-only: a write operation unexpectedly succeeded",
	)
}

// HTTPStatusForError mapea errores de dominio a códigos de estado HTTP y
// mensajes seguros para el cliente.
func HTTPStatusForError(err error) (int, string) {
	switch {
	case errors.Is(err, ErrToolLoopExceeded()):
		return http.StatusBadGateway, "agent failed to finalize a response"
	case errors.Is(err, ErrRequestTimeout()):
		return http.StatusGatewayTimeout, "request timed out"
	case IsLLMProviderUnavailable(err):
		return http.StatusBadGateway, "llm provider unavailable"
	default:
		return http.StatusServiceUnavailable, "service temporarily unavailable"
	}
}
