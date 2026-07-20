// Package agent contiene el dominio, los puertos de salida y el caso de uso
// del agente de preguntas en lenguaje natural sobre MongoDB.
package agent

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/HongXiangZuniga/agente-inaricards/pkg/utils"
)

// Nombres de las herramientas expuestas al LLM.
const (
	ToolListCollections    = "list_collections"
	ToolDescribeCollection = "describe_collection"
	ToolQueryFind          = "query_find"
	ToolQueryAggregate     = "query_aggregate"
)

// BuildToolDefinitions construye las cuatro herramientas de solo lectura
// disponibles para el agente.
func BuildToolDefinitions(sampleSize int, maxResultLimit int) []ToolDefinition {
	return []ToolDefinition{
		{
			Name:        ToolListCollections,
			Description: "Lista todas las colecciones disponibles en la base de datos. Solo lectura.",
			Parameters: map[string]any{
				"type":       "object",
				"properties": map[string]any{},
				"required":   []string{},
			},
		},
		{
			Name: ToolDescribeCollection,
			Description: fmt.Sprintf(
				"Describe los campos de una colección inspeccionando una muestra de %d documentos. Solo lectura.",
				sampleSize,
			),
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"collection": map[string]any{
						"type":        "string",
						"description": "Nombre de la colección a describir",
					},
				},
				"required": []string{"collection"},
			},
		},
		{
			Name: ToolQueryFind,
			Description: fmt.Sprintf(
				"Ejecuta una consulta de solo lectura find() sobre una colección. "+
					"Opcionalmente limita los resultados (máximo %d). "+
					"Con format=csv el resultado se devuelve como CSV tabular "+
					"(más compacto para resultados con muchas filas). Solo lectura.",
				maxResultLimit,
			),
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"collection": map[string]any{
						"type":        "string",
						"description": "Nombre de la colección",
					},
					"filter": map[string]any{
						"type":        "object",
						"description": "Filtro BSON/JSON para find()",
					},
					"projection": map[string]any{
						"type":        "object",
						"description": "Proyección opcional de campos",
					},
					"limit": map[string]any{
						"type":        "integer",
						"description": fmt.Sprintf("Límite de resultados (máximo %d)", maxResultLimit),
					},
					"format": map[string]any{
						"type":        "string",
						"enum":        []string{"json", "csv"},
						"description": "Formato del resultado: json (por defecto) o csv (tabular, compacto)",
					},
				},
				"required": []string{"collection", "filter"},
			},
		},
		{
			Name: ToolQueryAggregate,
			Description: fmt.Sprintf(
				"Ejecuta un pipeline de agregación de solo lectura sobre una colección. "+
					"Opcionalmente limita los resultados (máximo %d). "+
					"Con format=csv el resultado se devuelve como CSV tabular "+
					"(más compacto para resultados con muchas filas). "+
					"NO permite stages $out ni $merge. Solo lectura.",
				maxResultLimit,
			),
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"collection": map[string]any{
						"type":        "string",
						"description": "Nombre de la colección",
					},
					"pipeline": map[string]any{
						"type":        "array",
						"description": "Pipeline de agregación como array de stages",
						"items":       map[string]any{"type": "object"},
					},
					"limit": map[string]any{
						"type":        "integer",
						"description": fmt.Sprintf("Límite de resultados (máximo %d)", maxResultLimit),
					},
					"format": map[string]any{
						"type":        "string",
						"enum":        []string{"json", "csv"},
						"description": "Formato del resultado: json (por defecto) o csv (tabular, compacto)",
					},
				},
				"required": []string{"collection", "pipeline"},
			},
		},
	}
}

// DispatchToolCall ejecuta la tool indicada por call sobre repo y devuelve
// el resultado serializado como JSON junto con una bandera de error.
// Nunca produce panic ante JSON malformado ni ante tool desconocidas.
func DispatchToolCall(
	ctx context.Context,
	repo ReadOnlyMongoRepository,
	sampleSize int,
	maxResultLimit int,
	call ToolCall,
) (string, bool) {
	var (
		result  string
		repoErr error
	)

	switch call.Name {
	case ToolListCollections:
		var collections []CollectionInfo
		collections, repoErr = repo.ListCollections(ctx)
		result, _ = mustMarshalJSON(collections)

	case ToolDescribeCollection:
		args, err := parseArgs(call.Arguments)
		if err != nil {
			return fmt.Sprintf("error parsing arguments: %v", err), true
		}
		collection, ok := args["collection"].(string)
		if !ok {
			return "missing required argument: collection", true
		}
		var samples []FieldSample
		samples, repoErr = repo.DescribeCollection(ctx, collection, sampleSize)
		result, _ = mustMarshalJSON(samples)

	case ToolQueryFind:
		args, err := parseArgs(call.Arguments)
		if err != nil {
			return fmt.Sprintf("error parsing arguments: %v", err), true
		}
		collection, ok := args["collection"].(string)
		if !ok {
			return "missing required argument: collection", true
		}
		filter, ok := mustMarshalJSON(args["filter"])
		if !ok {
			return "missing required argument: filter", true
		}
		projection, _ := mustMarshalJSON(args["projection"])
		limit := effectiveLimit(args, maxResultLimit)
		format := requestedFormat(args)
		result, repoErr = repo.Find(ctx, collection, filter, projection, limit)
		if repoErr == nil && format == "csv" {
			result, err = jsonArrayToCSV(result)
			if err != nil {
				return fmt.Sprintf("error converting result to csv: %v", err), true
			}
		}

	case ToolQueryAggregate:
		args, err := parseArgs(call.Arguments)
		if err != nil {
			return fmt.Sprintf("error parsing arguments: %v", err), true
		}
		collection, ok := args["collection"].(string)
		if !ok {
			return "missing required argument: collection", true
		}
		pipeline, ok := args["pipeline"].([]any)
		if !ok {
			return "missing required argument: pipeline", true
		}
		if disallowed, stage := hasDisallowedStage(pipeline); disallowed {
			return utils.ErrDisallowedPipelineStage(stage).Error(), true
		}
		pipelineJSON, _ := json.Marshal(args["pipeline"])
		limit := effectiveLimit(args, maxResultLimit)
		format := requestedFormat(args)
		result, repoErr = repo.Aggregate(ctx, collection, string(pipelineJSON), limit)
		if repoErr == nil && format == "csv" {
			result, err = jsonArrayToCSV(result)
			if err != nil {
				return fmt.Sprintf("error converting result to csv: %v", err), true
			}
		}

	default:
		return fmt.Sprintf("unknown tool: %s", call.Name), true
	}

	if repoErr != nil {
		return utils.RedactMongoCredentials(fmt.Sprintf("repository error: %v", repoErr)), true
	}
	return result, false
}

// parseArgs parsea JSON crudo en un mapa de argumentos.
func parseArgs(raw string) (map[string]any, error) {
	args := make(map[string]any)
	if err := json.Unmarshal([]byte(raw), &args); err != nil {
		return nil, err
	}
	return args, nil
}

// effectiveLimit normaliza el límite recibido a [1, maxResultLimit].
func effectiveLimit(args map[string]any, maxResultLimit int) int {
	raw, ok := args["limit"]
	if !ok {
		return maxResultLimit
	}
	limit, err := numberToInt(raw)
	if err != nil || limit <= 0 || limit > maxResultLimit {
		return maxResultLimit
	}
	return limit
}

// requestedFormat devuelve el formato de resultado solicitado ("json" por
// defecto; "csv" solo si se indica explícitamente).
func requestedFormat(args map[string]any) string {
	if format, ok := args["format"].(string); ok && format == "csv" {
		return "csv"
	}
	return "json"
}

// numberToInt convierte un número deserializado desde JSON (float64) a int.
func numberToInt(v any) (int, error) {
	switch n := v.(type) {
	case float64:
		return int(n), nil
	case int:
		return n, nil
	case int64:
		return int(n), nil
	default:
		return 0, fmt.Errorf("invalid limit type")
	}
}

// hasDisallowedStage detecta si el pipeline contiene stages de escritura
// $out o $merge, incluyendo aquellos anidados en sub-pipelines.
func hasDisallowedStage(pipeline []any) (bool, string) {
	for _, stage := range pipeline {
		if disallowed, stageName := stageHasDisallowedKey(stage); disallowed {
			return true, stageName
		}
	}
	return false, ""
}

// stageHasDisallowedKey inspecciona un stage individual y recursivamente sus
// sub-pipelines anidados.
func stageHasDisallowedKey(stage any) (bool, string) {
	m, ok := stage.(map[string]any)
	if !ok {
		return false, ""
	}
	for key, value := range m {
		if key == "$out" {
			return true, "$out"
		}
		if key == "$merge" {
			return true, "$merge"
		}
		if nested, ok := value.([]any); ok {
			if disallowed, stageName := hasDisallowedStage(nested); disallowed {
				return true, stageName
			}
		}
		if nestedMap, ok := value.(map[string]any); ok {
			for _, inner := range nestedMap {
				if innerSlice, ok := inner.([]any); ok {
					if disallowed, stageName := hasDisallowedStage(innerSlice); disallowed {
						return true, stageName
					}
				}
			}
		}
	}
	return false, ""
}

// mustMarshalJSON serializa v a JSON. Si v es nil o el marshaling falla,
// devuelve "{}" y false.
func mustMarshalJSON(v any) (string, bool) {
	if v == nil {
		return "", false
	}
	data, err := json.Marshal(v)
	if err != nil {
		return "{}", false
	}
	return string(data), true
}
