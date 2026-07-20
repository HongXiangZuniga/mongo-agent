// Package rest implementa el adapter de entrada (driving) HTTP usando gin.
package rest

// Response es el DTO de respuesta genérico de la API REST.
type Response struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data"`
}
