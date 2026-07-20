// Scaffold vacío. Define las ToolDefinition expuestas al LLM
// (list_collections, describe_collection, query_find, query_aggregate) y el
// dispatcher DispatchToolCall que las traduce a llamadas sobre
// ReadOnlyMongoRepository, siguiendo la sección "3. Application / Caso de
// uso" de openspec/changes/add-nl-mongo-agent/tasks.md (tareas 3.1-3.5).
//
// INVARIANTE DE SEGURIDAD: DispatchToolCall debe rechazar cualquier stage
// de agregación $out/$merge antes de invocar al repositorio.
package agent
