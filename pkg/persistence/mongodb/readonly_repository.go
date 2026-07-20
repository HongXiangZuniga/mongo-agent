// Package mongodb implementa el adapter de salida (driven) hacia MongoDB.
//
// Scaffold vacío. Implementa NewReadOnlyRepository y los cuatro métodos de
// agent.ReadOnlyMongoRepository (ListCollections, DescribeCollection, Find,
// Aggregate) siguiendo la sección "4. Adapters de salida" de
// openspec/changes/add-nl-mongo-agent/tasks.md (tareas 4.1-4.6).
//
// INVARIANTE DE SEGURIDAD NO NEGOCIABLE: este archivo jamás debe invocar
// InsertOne/InsertMany/UpdateOne/UpdateMany/DeleteOne/DeleteMany/ReplaceOne/
// Drop sobre ninguna colección.
package mongodb
