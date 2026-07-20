# agente-inaricards

Agente de IA que recibe preguntas en lenguaje natural vía HTTP, las traduce a
consultas MongoDB **exclusivamente de lectura** mediante un bucle de
tool-calling contra un LLM (OpenCode Zen), mantiene memoria de conversación
por sesión en Redis, y devuelve una respuesta en texto.

Este repositorio es un proyecto de portafolio. Toda la planificación técnica
(propuesta, diseño, specs y tareas de implementación) vive en
[`openspec/changes/add-nl-mongo-agent/`](./openspec/changes/add-nl-mongo-agent/):

- `proposal.md` — qué cambia y por qué.
- `design.md` — arquitectura hexagonal completa (diagrama, decisiones,
  contratos de puertos, estrategia de testing).
- `specs/nl-mongo-agent/spec.md` — requisitos y escenarios observables.
- `tasks.md` — plan de implementación desglosado en tareas atómicas.

> Estado: **scaffold**. El código en `pkg/`, `cmd/` y `test/` todavía no
> contiene lógica de negocio; se implementa siguiendo `tasks.md` al pie de la
> letra.

## Arquitectura (resumen)

Hexagonal / Ports & Adapters, replicando las convenciones de
[`CrudGoExample`](https://github.com/HongXiangZuniga/CrudGoExample):

```
pkg/agent/            dominio + ports de salida (LLMClient, ReadOnlyMongoRepository, SessionStore) + caso de uso
pkg/persistence/mongodb/  adapter de salida: acceso a MongoDB, SOLO lectura
pkg/persistence/redis/    adapter de salida: memoria de conversación por sesión
pkg/llm/opencodezen/      adapter de salida: cliente HTTP hacia OpenCode Zen
pkg/http/rest/            adapter de entrada: API HTTP (gin)
pkg/utils/                errores de dominio compartidos
cmd/server/               composition root (wiring explícito por constructor)
test/                     tests unitarios (dominio/aplicación con fakes) e integración (adapters reales)
```

Ver el diagrama completo en
[`design.md`](./openspec/changes/add-nl-mongo-agent/design.md).

## Garantía de solo lectura

El agente **nunca** puede escribir, actualizar o borrar datos en MongoDB.
Esto se garantiza en 4 capas independientes (detalladas en `design.md`):

1. El usuario de MongoDB configurado en `MONGODB_URI` tiene rol `read` en el
   clúster de Atlas.
2. El puerto `ReadOnlyMongoRepository` no declara ningún método de escritura.
3. El adapter valida y rechaza pipelines de agregación con stages de
   escritura (`$out`, `$merge`).
4. Al arrancar, el proceso ejecuta una verificación activa
   (`VerifyReadOnlyGuarantee`) que confirma que un intento de escritura de
   prueba es rechazado por MongoDB; si no lo es, el proceso se niega a
   arrancar.

## Variables de entorno

Ver [`.env.example`](./.env.example) para la lista completa. Resumen:

| Variable | Descripción |
|---|---|
| `PORT` | Puerto HTTP del servidor. |
| `API_TOKEN` | Token estático requerido en el header `Authorization`. |
| `MONGODB_URI` | Cadena de conexión a MongoDB Atlas (usuario de solo lectura). |
| `MONGODB_DB_NAME` | Base de datos a consultar. |
| `MONGODB_QUERY_TIMEOUT_SECONDS` | Timeout por operación contra MongoDB. |
| `MONGODB_MAX_RESULT_LIMIT` | Límite máximo de documentos devueltos por consulta. |
| `MONGO_SAMPLE_SIZE` | Tamaño de muestra al describir una colección. |
| `OPENCODE_API_KEY` | API key de OpenCode Zen. |
| `OPENCODE_BASE_URL` | Base URL del endpoint OpenAI-compatible de OpenCode Zen. |
| `OPENCODE_MODEL` | Modelo a usar (formato `opencode/<model-id>`). |
| `AGENT_MAX_TOOL_ITERATIONS` | Máximo de iteraciones de tool-calling por pregunta. |
| `AGENT_REQUEST_TIMEOUT_SECONDS` | Timeout total por petición. |
| `REDIS_ADDR` | Dirección del servidor Redis. |
| `REDIS_PASSWORD` | Password de Redis (vacío en desarrollo local). |
| `REDIS_DB` | Índice de base Redis. |
| `SESSION_TTL_SECONDS` | TTL del historial de conversación por sesión. |

## Correr localmente

```bash
cp .env.example .env
# completar .env con credenciales reales (nunca commitear este archivo)
make run
```

`make run` levanta Redis vía `docker-compose` y corre el servidor Go
localmente. MongoDB es el clúster de Atlas remoto (no requiere contenedor
local).

## Ejemplo de uso

```bash
curl -X POST http://localhost:8080/ask \
  -H "Authorization: <api_token>" \
  -H "Content-Type: application/json" \
  -d '{"session_id": "demo-1", "question": "¿Cuántos documentos hay en total?"}'
```

## Tests

```bash
make unit-test         # dominio + aplicación (fakes) + HTTP (httptest)
make integration-test  # adapters reales: requiere MONGODB_URI/REDIS_ADDR/OPENCODE_API_KEY
```

## Créditos y alcance

Estructura de carpetas inspirada en
[`CrudGoExample`](https://github.com/HongXiangZuniga/CrudGoExample) (propio,
público). Ningún código de este repositorio proviene de proyectos internos
o propietarios de terceros — ver la nota de alcance/legal en `design.md`.
