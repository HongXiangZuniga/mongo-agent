## ADDED Requirements

### Requirement: Responder Preguntas en Lenguaje Natural
El sistema SHALL expone un endpoint HTTP `POST /ask` que recibe una pregunta en lenguaje natural junto con un identificador de sesión, y SHALL devolver una respuesta en texto generada a partir de datos reales de MongoDB.

#### Scenario: Pregunta válida con datos existentes
- **GIVEN** un usuario autenticado envía `POST /ask` con `{"session_id": "abc123", "question": "¿Cuántas tarjetas activas hay?"}`
- **WHEN** el agente traduce la pregunta a una o más consultas de solo lectura contra MongoDB y obtiene resultados
- **THEN** el sistema responde `200 OK` con un cuerpo JSON que incluye un campo `answer` de tipo string con la respuesta en lenguaje natural
- **AND** el campo `session_id` del cuerpo de respuesta coincide con el enviado en la petición

#### Scenario: Pregunta con cuerpo inválido
- **GIVEN** un usuario autenticado envía `POST /ask` con un cuerpo JSON sin el campo `question` o con `question` vacío
- **WHEN** el sistema valida la petición
- **THEN** el sistema responde `400 Bad Request` con un mensaje de error describiendo el campo faltante
- **AND** el sistema no invoca al LLM ni a MongoDB

#### Scenario: session_id ausente
- **GIVEN** un usuario autenticado envía `POST /ask` sin el campo `session_id`
- **WHEN** el sistema valida la petición
- **THEN** el sistema genera un `session_id` nuevo (UUID) para la petición
- **AND** el sistema responde `200 OK` incluyendo el `session_id` generado en el cuerpo de la respuesta

### Requirement: Autenticación por Token de API
El sistema SHALL exigir un token de API estático en el header `Authorization` para toda petición a `POST /ask`, validado contra la variable de entorno `API_TOKEN`.

#### Scenario: Petición sin header Authorization
- **GIVEN** el servicio tiene configurada la variable de entorno `API_TOKEN`
- **WHEN** una petición a `POST /ask` no incluye el header `Authorization`
- **THEN** el sistema responde `401 Unauthorized` sin invocar al agente

#### Scenario: Petición con token inválido
- **GIVEN** el servicio tiene configurada la variable de entorno `API_TOKEN`
- **WHEN** una petición a `POST /ask` incluye un header `Authorization` cuyo valor no coincide con `API_TOKEN`
- **THEN** el sistema responde `401 Unauthorized` sin invocar al agente

#### Scenario: Arranque sin API_TOKEN configurado
- **GIVEN** la variable de entorno `API_TOKEN` no está definida
- **WHEN** el proceso del servidor arranca
- **THEN** el sistema registra un error crítico y termina el proceso sin abrir el puerto HTTP

### Requirement: Memoria de Conversación por Sesión en Redis
El sistema SHALL persistir el historial de mensajes de cada sesión (usuario, asistente, llamadas a tools y resultados de tools) en Redis, asociado a un `session_id`, con un tiempo de expiración (TTL) configurable, y SHALL recuperar dicho historial al procesar una nueva pregunta de la misma sesión.

#### Scenario: Segunda pregunta de la misma sesión usa contexto previo
- **GIVEN** una sesión `session_id="abc123"` con una pregunta y respuesta previas almacenadas en Redis
- **WHEN** el usuario envía una nueva pregunta con el mismo `session_id` que hace referencia implícita a la respuesta anterior (ej. "¿y el mes pasado?")
- **THEN** el sistema recupera el historial previo de Redis antes de invocar al LLM
- **AND** el historial recuperado se incluye como contexto en la petición al LLM

#### Scenario: Sesión expirada por TTL
- **GIVEN** una sesión cuyo TTL en Redis ya expiró
- **WHEN** el usuario envía una nueva pregunta con ese `session_id`
- **THEN** el sistema trata la petición como el inicio de una sesión nueva, sin historial previo
- **AND** el sistema responde `200 OK` normalmente

#### Scenario: Persistencia tras cada turno
- **GIVEN** una petición válida procesada completamente (incluyendo tool calls intermedios)
- **WHEN** el agente genera la respuesta final
- **THEN** el sistema añade a Redis el mensaje del usuario, los mensajes intermedios de tool calls/resultados, y la respuesta final del asistente, en ese orden
- **AND** el sistema restablece el TTL de la sesión en Redis a `SESSION_TTL_SECONDS`

### Requirement: Descubrimiento Dinámico del Esquema de MongoDB
El sistema SHALL permitir que el agente descubra en tiempo de ejecución qué colecciones y campos existen en la base de datos configurada, mediante tools invocables por el LLM, sin depender de ningún esquema de negocio hardcodeado en el código o en configuración versionada.

#### Scenario: El LLM lista las colecciones disponibles
- **GIVEN** el agente no tiene contexto previo sobre el esquema en la sesión actual
- **WHEN** el LLM invoca la tool `list_collections`
- **THEN** el sistema devuelve al LLM la lista de nombres de colecciones existentes en `MONGODB_DB_NAME`

#### Scenario: El LLM describe una colección
- **WHEN** el LLM invoca la tool `describe_collection` con `{"collection": "cards"}`
- **THEN** el sistema toma una muestra de hasta `MONGO_SAMPLE_SIZE` documentos reales de esa colección
- **AND** el sistema devuelve al LLM la lista de campos observados junto con los tipos de dato detectados y un valor de ejemplo por campo

#### Scenario: Colección inexistente
- **WHEN** el LLM invoca `describe_collection` con el nombre de una colección que no existe en la base de datos
- **THEN** el sistema devuelve al LLM un resultado de tool con `is_error=true` y un mensaje indicando que la colección no existe
- **AND** el sistema no interrumpe el bucle de tool-calling por este error

### Requirement: Ejecución de Consultas MongoDB Exclusivamente de Lectura
El sistema SHALL garantizar que ninguna operación ejecutada contra MongoDB, en ninguna capa, pueda escribir, actualizar, reemplazar o borrar documentos, ni ejecutar comandos administrativos destructivos.

#### Scenario: Tool de consulta solo permite find y aggregate de lectura
- **WHEN** el LLM invoca la tool `query_find` o `query_aggregate` con parámetros válidos
- **THEN** el sistema ejecuta exclusivamente una operación `find` o `aggregate` de lectura contra MongoDB
- **AND** el sistema no expone ninguna tool capaz de invocar `insertOne`, `updateOne`, `deleteOne`, `deleteMany`, `replaceOne`, `drop` ni comandos administrativos de escritura

#### Scenario: Pipeline de agregación con stage de escritura es rechazado
- **WHEN** el LLM invoca `query_aggregate` con un pipeline que contiene un stage `$out` o `$merge`
- **THEN** el sistema rechaza la ejecución antes de enviarla al driver de MongoDB
- **AND** el sistema devuelve al LLM un resultado de tool con `is_error=true` indicando que ese stage no está permitido

#### Scenario: Verificación activa de solo lectura al arrancar
- **GIVEN** el servicio arranca con las credenciales configuradas en `MONGODB_URI`
- **WHEN** el proceso ejecuta la verificación de arranque contra MongoDB
- **THEN** el sistema intenta una escritura de prueba controlada en una colección canario y espera que MongoDB la rechace por permisos insuficientes
- **AND** si la escritura de prueba tiene éxito, el sistema registra un error crítico y el proceso termina sin abrir el puerto HTTP

#### Scenario: Límite de resultados devueltos
- **WHEN** una tool de consulta (`query_find` o `query_aggregate`) se ejecuta sin especificar `limit`, o especificando un `limit` mayor al máximo permitido
- **THEN** el sistema aplica el límite máximo configurado (`MONGODB_MAX_RESULT_LIMIT`) a la consulta ejecutada contra MongoDB

### Requirement: Límites del Bucle de Tool-Calling
El sistema SHALL limitar el número de iteraciones de tool-calling por pregunta y el tiempo total de procesamiento por petición, ambos configurables mediante variables de entorno.

#### Scenario: Respuesta directa sin necesidad de tools
- **WHEN** el LLM responde a la primera iteración sin solicitar ninguna tool call
- **THEN** el sistema devuelve esa respuesta como resultado final sin iteraciones adicionales

#### Scenario: Se excede el máximo de iteraciones
- **GIVEN** `AGENT_MAX_TOOL_ITERATIONS` está configurado con un valor N
- **WHEN** el LLM solicita tool calls en N iteraciones consecutivas sin producir una respuesta final
- **THEN** el sistema detiene el bucle tras la iteración N
- **AND** el sistema responde `502 Bad Gateway` (o código de error equivalente) con un mensaje indicando que no se pudo completar la respuesta dentro del límite de iteraciones

#### Scenario: Se excede el timeout total de la petición
- **GIVEN** `AGENT_REQUEST_TIMEOUT_SECONDS` está configurado con un valor T
- **WHEN** el procesamiento completo de una pregunta (incluyendo llamadas al LLM y a MongoDB) supera T segundos
- **THEN** el sistema cancela el procesamiento en curso
- **AND** el sistema responde `504 Gateway Timeout` con un mensaje de error controlado

### Requirement: Abstracción del Proveedor de LLM mediante Puerto
El sistema SHALL definir un puerto de salida (`LLMClient`) del que dependa el caso de uso del agente, de forma que la lógica de negocio no dependa directamente de ningún SDK propietario de un proveedor de LLM. El adapter concreto inicial SHALL comunicarse con OpenCode Zen mediante HTTP compatible con OpenAI.

#### Scenario: Petición exitosa al LLM
- **WHEN** el caso de uso invoca al puerto `LLMClient` con un historial de mensajes y la lista de tools disponibles
- **THEN** el adapter de OpenCode Zen envía una petición HTTP `POST` a `{OPENCODE_BASE_URL}/chat/completions` con header `Authorization: Bearer {OPENCODE_API_KEY}` y el modelo configurado en `OPENCODE_MODEL`
- **AND** el adapter traduce la respuesta HTTP al tipo de dominio `LLMResponse` sin exponer tipos específicos de OpenCode Zen fuera del paquete adapter

#### Scenario: El proveedor de LLM responde con error o no disponible
- **WHEN** la petición HTTP al proveedor de LLM falla (error de red, timeout, o código de estado >= 500)
- **THEN** el adapter devuelve un error al caso de uso
- **AND** el caso de uso responde a la petición HTTP original con `502 Bad Gateway` y un mensaje de error controlado, sin exponer detalles internos de la petición al proveedor

### Requirement: Manejo de Errores y Disponibilidad de Dependencias
El sistema SHALL responder con códigos y mensajes de error controlados cuando alguna dependencia externa (MongoDB, Redis, LLM) no esté disponible, sin exponer detalles internos sensibles (credenciales, stack traces) en la respuesta HTTP.

#### Scenario: MongoDB no disponible durante una consulta
- **WHEN** una tool de consulta falla porque MongoDB no responde o rechaza la conexión
- **THEN** el sistema registra el error internamente con detalle técnico
- **AND** el sistema devuelve al LLM un resultado de tool con `is_error=true` y un mensaje genérico, permitiendo que el LLM decida cómo continuar dentro del límite de iteraciones

#### Scenario: Redis no disponible al iniciar una sesión
- **WHEN** el sistema no puede conectarse a Redis para leer o escribir el historial de una sesión
- **THEN** el sistema registra el error internamente
- **AND** el sistema responde `503 Service Unavailable` a la petición HTTP, sin proceder a invocar al LLM

#### Scenario: Ningún error interno se filtra al cliente HTTP
- **WHEN** ocurre cualquier error no controlado durante el procesamiento de `POST /ask`
- **THEN** el sistema responde con un código de error HTTP apropiado (5xx) y un mensaje genérico en el cuerpo JSON
- **AND** el mensaje de error devuelto al cliente HTTP no incluye credenciales, URIs de conexión, ni stack traces
