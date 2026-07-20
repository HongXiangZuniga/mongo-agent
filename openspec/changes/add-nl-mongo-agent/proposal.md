## Why

Hoy no existe una forma de que una persona no técnica obtenga respuestas sobre los datos almacenados en el clúster MongoDB de Inaricards sin escribir consultas o pedírselo a alguien que sepa consultar la base de datos. Se necesita un agente de IA que traduzca preguntas en lenguaje natural a consultas MongoDB de forma automática, manteniendo memoria de la conversación, y sin poner en riesgo la integridad de los datos (el agente jamás debe poder escribir, actualizar o borrar documentos).

## What Changes

**[Pregunta en lenguaje natural vía API HTTP]**
- From: No existe ninguna interfaz para consultar los datos en lenguaje natural.
- To: Un endpoint `POST /ask` recibe una pregunta en texto libre y un identificador de sesión, y devuelve una respuesta en texto generada a partir de los datos reales de MongoDB.
- Reason: Es el punto de entrada mínimo necesario para que el agente sea usable como demo de portafolio y como servicio HTTP consumible por cualquier cliente.
- Impact: Non-breaking (API nueva). Afecta a `pkg/http/rest` y `cmd/server`.

**[Autenticación simple del endpoint]**
- From: No hay control de acceso.
- To: Toda petición a `POST /ask` debe incluir un token estático en el header `Authorization`, validado contra la variable de entorno `API_TOKEN` (mismo patrón usado en el proyecto de referencia `CrudGoExample`).
- Reason: Evitar que el endpoint quede abierto públicamente sin ningún control, dado que dispara llamadas a un LLM de pago y a la base de datos real.
- Reason: Se conserva sencillo (token estático, no OAuth) porque el alcance es un proyecto de portafolio, no un sistema multiusuario con roles.
- Impact: Non-breaking (nueva capa de middleware). Cualquier cliente del API deberá enviar el header `Authorization`.

**[Memoria de conversación por sesión respaldada en Redis]**
- From: No hay concepto de sesión ni historial; cada pregunta sería independiente.
- To: Cada petición incluye un `session_id`; el historial de mensajes (usuario, asistente, tool calls y resultados de tools) se persiste en Redis con un TTL configurable, y se recupera en cada nueva pregunta de la misma sesión para dar contexto al LLM.
- Reason: Permite conversaciones de varios turnos ("¿y en marzo?") sin acoplar el estado al proceso HTTP (stateless a nivel de proceso, statefull a nivel de sesión vía Redis), lo que además permite escalar horizontalmente el servicio.
- Impact: Non-breaking (capacidad nueva). Introduce una nueva dependencia de infraestructura (Redis) declarada en `docker-compose.yml` y como puerto/adapter dedicado.

**[Descubrimiento dinámico del esquema de MongoDB]**
- From: No aplica (no existe agente).
- To: El agente no tiene hardcodeado ningún esquema de negocio; en su lugar, dispone de herramientas (`list_collections`, `describe_collection`) que el LLM puede invocar para descubrir en tiempo real qué colecciones y campos existen, mediante muestreo de documentos reales.
- Reason: El esquema real de negocio de Inaricards es información propietaria; este proyecto de portafolio debe ser genérico y no debe versionar ni asumir un esquema fijo de negocio.
- Impact: Non-breaking. Afecta al diseño del puerto de salida hacia MongoDB (`ReadOnlyMongoRepository`) y a la definición de tools expuestas al LLM.

**[Garantía de solo lectura sobre MongoDB]**
- From: No aplica (no existe agente).
- To: El agente SOLO puede ejecutar operaciones de lectura (`find`, `aggregate` sin stages de escritura como `$out`/`$merge`, `listCollections`). Ninguna ruta de código expone una operación de escritura/borrado, el puerto de salida hacia Mongo no declara ningún método de escritura, y el usuario de MongoDB configurado vía `MONGODB_URI` debe tener rol `read` en el clúster. Adicionalmente, al arrancar el servicio se ejecuta una verificación activa que confirma que un intento de escritura de prueba es rechazado por Mongo.
- Reason: Es la invariante de seguridad más crítica del proyecto: un agente de IA con acceso de escritura no controlado a una base de datos de producción es un riesgo inaceptable.
- Impact: Non-breaking, pero es una restricción dura del diseño. Afecta al puerto `ReadOnlyMongoRepository`, al adapter `pkg/persistence/mongodb`, y a la configuración de infraestructura (rol del usuario de Mongo Atlas).

**[Abstracción del proveedor de LLM vía puerto]**
- From: No aplica (no existe agente).
- To: El servicio de aplicación depende únicamente de un puerto `LLMClient` (interfaz Go). El adapter concreto inicial habla con OpenCode Zen (endpoint OpenAI-compatible `https://opencode.ai/zen/v1/chat/completions`, autenticación Bearer vía `OPENCODE_API_KEY`, modelos con prefijo `opencode/<model-id>`) usando un cliente HTTP simple, sin SDK propietario.
- Reason: Mantener el dominio y la aplicación libres de cualquier SDK de proveedor concreto, permitiendo cambiar de proveedor de LLM en el futuro sin tocar la lógica de negocio (principio de arquitectura hexagonal).
- Impact: Non-breaking. Nueva dependencia externa (OpenCode Zen) aislada completamente en `pkg/llm/opencodezen`.

**[Límites del bucle agéntico]**
- From: No aplica (no existe agente).
- To: El bucle de tool-calling del agente se detiene forzosamente tras un máximo configurable de iteraciones (`AGENT_MAX_TOOL_ITERATIONS`, default 6) y un timeout total configurable por petición (`AGENT_REQUEST_TIMEOUT_SECONDS`, default 30s), devolviendo un error controlado si se exceden.
- Reason: Evitar loops infinitos de tool-calling, costos de LLM descontrolados y peticiones HTTP colgadas indefinidamente.
- Impact: Non-breaking. Afecta al servicio de aplicación (`pkg/agent/service.go`).

## Capabilities

### New Capabilities
- `nl-mongo-agent`: agente que recibe preguntas en lenguaje natural vía HTTP, mantiene memoria de conversación en Redis, traduce las preguntas a consultas MongoDB de solo lectura mediante un bucle de tool-calling contra un LLM (OpenCode Zen), y devuelve una respuesta en texto, garantizando en todas las capas que nunca se ejecuta una operación de escritura sobre la base de datos.

### Modified Capabilities
(ninguna; este es un proyecto nuevo sin specs previos)

## Impact

- Código nuevo: `pkg/agent` (dominio, ports, caso de uso), `pkg/persistence/mongodb` (adapter de solo lectura), `pkg/persistence/redis` (adapter de sesión), `pkg/llm/opencodezen` (adapter LLM), `pkg/http/rest` (adapter HTTP de entrada), `pkg/utils`, `cmd/server` (composition root).
- Infraestructura nueva: contenedor Redis en `docker-compose.yml`; usuario de MongoDB Atlas con rol `read` (ya existente, gestionado fuera de este repo).
- Dependencias externas nuevas: OpenCode Zen (LLM), Redis (sesión), MongoDB Atlas (lectura).
- Sin impacto en sistemas existentes: es un repositorio nuevo, sin usuarios previos.
