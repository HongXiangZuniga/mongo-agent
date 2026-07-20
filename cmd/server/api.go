// Composition root del servicio. Scaffold vacío.
//
// Implementa main()/init()/initMongo()/initRedis() y el wiring explícito
// por constructor de todos los adapters (mongodb, redis, opencodezen, rest)
// y el caso de uso (agent.NewAgentService), siguiendo la sección
// "6. Infraestructura / Wiring" de
// openspec/changes/add-nl-mongo-agent/tasks.md (tareas 6.1-6.6).
//
// INVARIANTE DE SEGURIDAD NO NEGOCIABLE: main() debe llamar a
// mongodb.VerifyReadOnlyGuarantee antes de aceptar tráfico HTTP, y debe
// hacer log.Fatal si esa verificación falla.
package main
