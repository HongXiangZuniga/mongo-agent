// Package mongodb implementa el adapter de salida (driven) hacia MongoDB.
//
// INVARIANTE DE SEGURIDAD NO NEGOCIABLE: este archivo jamás invoca
// InsertOne/InsertMany/UpdateOne/UpdateMany/DeleteOne/DeleteMany/ReplaceOne/
// Drop sobre ninguna colección.
package mongodb

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	agent "github.com/HongXiangZuniga/agente-inaricards/pkg/agent"
	"github.com/HongXiangZuniga/agente-inaricards/pkg/utils"
)

// storage implementa agent.ReadOnlyMongoRepository.
type storage struct {
	db           *mongo.Database
	queryTimeout time.Duration
}

// NewReadOnlyRepository construye un repositorio de solo lectura de MongoDB.
func NewReadOnlyRepository(db *mongo.Database, queryTimeout time.Duration) agent.ReadOnlyMongoRepository {
	return &storage{db, queryTimeout}
}

// ListCollections devuelve los nombres de las colecciones de la base de datos.
func (s *storage) ListCollections(ctx context.Context) ([]agent.CollectionInfo, error) {
	ctx, cancel := context.WithTimeout(ctx, s.queryTimeout)
	defer cancel()

	names, err := s.db.ListCollectionNames(ctx, bson.M{})
	if err != nil {
		return nil, fmt.Errorf("failed to list collections: %w", err)
	}

	collections := make([]agent.CollectionInfo, 0, len(names))
	for _, name := range names {
		collections = append(collections, agent.CollectionInfo{Name: name})
	}
	return collections, nil
}

// DescribeCollection devuelve un resumen de los campos observados en una
// muestra de documentos de la colección.
func (s *storage) DescribeCollection(
	ctx context.Context,
	collection string,
	sampleSize int,
) ([]agent.FieldSample, error) {
	ctx, cancel := context.WithTimeout(ctx, s.queryTimeout)
	defer cancel()

	names, err := s.db.ListCollectionNames(ctx, bson.M{})
	if err != nil {
		return nil, fmt.Errorf("failed to list collections: %w", err)
	}
	if !containsString(names, collection) {
		return nil, utils.ErrCollectionNotFound(collection)
	}

	pipeline := mongo.Pipeline{
		bson.D{{Key: "$sample", Value: bson.D{{Key: "size", Value: sampleSize}}}},
	}
	cursor, err := s.db.Collection(collection).Aggregate(ctx, pipeline)
	if err != nil {
		if isNamespaceNotFound(err) {
			return nil, utils.ErrCollectionNotFound(collection)
		}
		return nil, fmt.Errorf("failed to describe collection: %w", err)
	}
	defer cursor.Close(ctx)

	samples := map[string]agent.FieldSample{}
	for cursor.Next(ctx) {
		var doc bson.M
		if err := cursor.Decode(&doc); err != nil {
			continue
		}
		for field, value := range doc {
			observed := samples[field]
			observed.Field = field
			goType := fmt.Sprintf("%T", value)
			if !containsString(observed.Types, goType) {
				observed.Types = append(observed.Types, goType)
			}
			if observed.ExampleValue == "" {
				observed.ExampleValue = fmt.Sprintf("%v", value)
			}
			samples[field] = observed
		}
	}

	result := make([]agent.FieldSample, 0, len(samples))
	for _, sample := range samples {
		result = append(result, sample)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Field < result[j].Field
	})
	return result, nil
}

// Find ejecuta find() sobre la colección con el filtro y proyección dados.
func (s *storage) Find(
	ctx context.Context,
	collection string,
	filterJSON string,
	projectionJSON string,
	limit int,
) (string, error) {
	filter := bson.M{}
	if filterJSON == "" {
		filterJSON = "{}"
	}
	if err := bson.UnmarshalExtJSON([]byte(filterJSON), true, &filter); err != nil {
		return "", fmt.Errorf("invalid filter: %w", err)
	}

	var projection bson.M
	if projectionJSON != "" {
		projection = bson.M{}
		if err := bson.UnmarshalExtJSON([]byte(projectionJSON), true, &projection); err != nil {
			return "", fmt.Errorf("invalid projection: %w", err)
		}
	}

	ctx, cancel := context.WithTimeout(ctx, s.queryTimeout)
	defer cancel()

	findOptions := options.Find().SetLimit(int64(limit))
	if projectionJSON != "" {
		findOptions.SetProjection(projection)
	}

	cursor, err := s.db.Collection(collection).Find(ctx, filter, findOptions)
	if err != nil {
		if isNamespaceNotFound(err) {
			return "[]", nil
		}
		return "", fmt.Errorf("failed to execute find: %w", err)
	}
	defer cursor.Close(ctx)

	return cursorToJSONString(ctx, cursor)
}

// Aggregate ejecuta un pipeline de agregación sobre la colección.
func (s *storage) Aggregate(
	ctx context.Context,
	collection string,
	pipelineJSON string,
	limit int,
) (string, error) {
	var stages []bson.M
	if err := bson.UnmarshalExtJSON([]byte(pipelineJSON), true, &stages); err != nil {
		return "", fmt.Errorf("invalid pipeline: %w", err)
	}
	if !hasLimitStage(stages) {
		stages = append(stages, bson.M{"$limit": limit})
	}

	ctx, cancel := context.WithTimeout(ctx, s.queryTimeout)
	defer cancel()

	cursor, err := s.db.Collection(collection).Aggregate(ctx, stages)
	if err != nil {
		if isNamespaceNotFound(err) {
			return "[]", nil
		}
		return "", fmt.Errorf("failed to execute aggregate: %w", err)
	}
	defer cursor.Close(ctx)

	return cursorToJSONString(ctx, cursor)
}

// cursorToJSONString decodifica todos los documentos de un cursor y los
// serializa como JSON. Garantiza "[]" para un resultado vacío.
func cursorToJSONString(ctx context.Context, cursor *mongo.Cursor) (string, error) {
	docs := make([]bson.M, 0)
	if err := cursor.All(ctx, &docs); err != nil {
		return "", fmt.Errorf("failed to decode cursor: %w", err)
	}
	data, err := json.Marshal(docs)
	if err != nil {
		return "", fmt.Errorf("failed to marshal results: %w", err)
	}
	return string(data), nil
}

// isNamespaceNotFound detecta errores de namespace inexistente.
func isNamespaceNotFound(err error) bool {
	if err == nil {
		return false
	}
	lower := strings.ToLower(err.Error())
	return strings.Contains(lower, "namespace not found") ||
		strings.Contains(lower, "ns not found")
}

// hasLimitStage indica si el pipeline ya termina con un stage $limit.
func hasLimitStage(stages []bson.M) bool {
	if len(stages) == 0 {
		return false
	}
	last := stages[len(stages)-1]
	_, ok := last["$limit"]
	return ok
}

// containsString indica si slice contiene value.
func containsString(slice []string, value string) bool {
	for _, item := range slice {
		if item == value {
			return true
		}
	}
	return false
}
