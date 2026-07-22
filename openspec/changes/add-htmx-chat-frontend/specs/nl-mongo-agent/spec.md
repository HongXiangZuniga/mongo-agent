## MODIFIED Requirements

### Requirement: Memoria de Conversación por Sesión en Redis
El sistema SHALL persistir el historial de mensajes de cada sesión (usuario, asistente, llamadas a tools y resultados de tools) en Redis, asociado a un `session_id`, con un tiempo de expiración (TTL) configurable, SHALL recuperar dicho historial al procesar una nueva pregunta de la misma sesión, y SHALL mantener un índice de metadata (título derivado de la primera pregunta y marca de última actividad) por cada sesión que haya recibido al menos un mensaje, eliminando dicho índice cuando la sesión se limpia explícitamente.

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

#### Scenario: Primera pregunta de una sesión indexa su metadata (nuevo)
- **GIVEN** un `session_id` que nunca antes recibió ninguna pregunta
- **WHEN** el usuario envía su primera pregunta a ese `session_id` (vía `POST /ask` o vía la interfaz web)
- **THEN** el sistema indexa esa sesión con un título derivado del texto de la primera pregunta (truncado a `WEB_SESSION_TITLE_MAX_LENGTH` caracteres si es más largo) y una marca de última actividad igual al momento de la petición
- **AND** una llamada posterior que liste las sesiones indexadas incluye este `session_id` con ese título

#### Scenario: Preguntas posteriores actualizan la actividad sin cambiar el título (nuevo)
- **GIVEN** una sesión ya indexada con un título derivado de su primera pregunta
- **WHEN** el usuario envía una segunda o posterior pregunta a esa misma sesión
- **THEN** el sistema actualiza la marca de última actividad de esa sesión en el índice
- **AND** el título indexado permanece exactamente igual al derivado de la primera pregunta, sin ser sobrescrito

#### Scenario: Limpiar una sesión elimina también su entrada del índice (nuevo, modifica ClearSession)
- **GIVEN** una sesión indexada con historial de mensajes en Redis
- **WHEN** el sistema invoca la limpieza explícita de esa sesión
- **THEN** el sistema elimina el historial de mensajes de la sesión
- **AND** el sistema elimina también la metadata y la entrada de esa sesión del índice, de forma que deja de aparecer en listados posteriores

## ADDED Requirements

### Requirement: Gestión Explícita de Sesiones desde el Caso de Uso
El sistema SHALL exponer, a través de `AgentService`, operaciones explícitas para listar las sesiones indexadas, crear una sesión nueva sin persistirla hasta su primer mensaje, cerrar una sesión existente, y obtener el historial de una sesión filtrado para presentación (excluyendo mensajes internos de rol `tool` y `system`), delegando en todos los casos en el puerto `SessionStore` ya existente, sin que ningún adapter de entrada acceda directamente a `SessionStore`.

#### Scenario: Listar sesiones indexadas
- **WHEN** un adapter de entrada invoca la operación de listado de sesiones del caso de uso
- **THEN** el sistema devuelve las sesiones indexadas ordenadas de actividad más reciente a más antigua, cada una con su `session_id`, título y marca de última actividad

#### Scenario: Crear una sesión nueva no persiste nada hasta el primer mensaje
- **WHEN** un adapter de entrada invoca la operación de creación de sesión del caso de uso
- **THEN** el sistema genera un identificador de sesión nuevo y lo devuelve junto con un título por defecto
- **AND** el sistema no realiza ninguna escritura en el almacén de sesiones como parte de esta operación

#### Scenario: Cerrar una sesión existente
- **WHEN** un adapter de entrada invoca la operación de cierre de sesión del caso de uso con un `session_id` existente
- **THEN** el sistema delega en la limpieza explícita de `SessionStore` para ese `session_id` (ver el escenario de limpieza del índice arriba)

#### Scenario: Obtener el historial de una sesión para presentación
- **WHEN** un adapter de entrada invoca la operación de obtención de conversación del caso de uso con un `session_id`
- **THEN** el sistema devuelve únicamente los mensajes de rol `user` y los mensajes de rol `assistant` con contenido no vacío de esa sesión, en orden cronológico
- **AND** el sistema omite del resultado cualquier mensaje de rol `tool` o `system` presente en el historial completo
- **AND** el sistema omite los mensajes de rol `assistant` con contenido vacío (artefactos intermedios de tool-calling que solo portan llamadas a herramientas), por no ser contenido presentable

### Requirement: Formato de Resultado CSV en Consultas de Lectura
El sistema SHALL aceptar un parámetro opcional `format` en las herramientas `query_find` y `query_aggregate` (`"json"` por defecto, `"csv"` como alternativa), devolviendo el resultado de la consulta como CSV tabular cuando se solicita `format="csv"`, sin modificar el comportamiento por defecto en JSON.

#### Scenario: Consulta con formato CSV
- **WHEN** el agente invoca `query_find` o `query_aggregate` con el argumento `format="csv"`
- **THEN** el sistema ejecuta la misma consulta de solo lectura de siempre (mismos límites y mismas restricciones)
- **AND** el resultado devuelto al LLM es un documento CSV con una fila de cabecera (unión de las claves de primer nivel de los documentos) y una fila por documento
- **AND** los valores primitivos se serializan tal cual, los valores `null` como celda vacía, y los objetos o arrays anidados como JSON compacto dentro de la celda

#### Scenario: El formato por defecto sigue siendo JSON
- **WHEN** el agente invoca `query_find` o `query_aggregate` sin el argumento `format` (o con un valor distinto de `"csv"`)
- **THEN** el resultado devuelto al LLM es exactamente el mismo JSON que antes de introducir el parámetro
