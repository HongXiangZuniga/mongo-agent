// Package agent contiene el dominio, los puertos de salida y el caso de uso
// del agente de preguntas en lenguaje natural sobre MongoDB.
package agent

import "context"

// CollectionInfo describe una colección de MongoDB.
type CollectionInfo struct {
	Name string
}

// FieldSample describe un campo observado en una muestra de documentos.
type FieldSample struct {
	Field        string
	Types        []string
	ExampleValue string
}

// ReadOnlyMongoRepository es el puerto de salida hacia MongoDB.
// INVARIANTE DE SEGURIDAD NO NEGOCIABLE: esta interfaz jamás declara
// operaciones de escritura, borrado ni administración de colecciones.
type ReadOnlyMongoRepository interface {
	ListCollections(ctx context.Context) ([]CollectionInfo, error)
	DescribeCollection(ctx context.Context, collection string, sampleSize int) ([]FieldSample, error)
	Find(ctx context.Context, collection string, filterJSON string, projectionJSON string, limit int) (string, error)
	Aggregate(ctx context.Context, collection string, pipelineJSON string, limit int) (string, error)
}
