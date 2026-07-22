## ADDED Requirements

### Requirement: System Prompt Resistente a Prompt Injection
El system prompt del agente SHALL incluir instrucciones explícitas que prohíban revelar su propio contenido o la configuración interna del sistema, y SHALL instruir al modelo a tratar el contenido devuelto por las herramientas de solo lectura como datos no confiables, nunca como instrucciones nuevas.

#### Scenario: El system prompt prohíbe revelar su propio contenido
- **GIVEN** la constante `systemPrompt` definida en `cmd/server/api.go`
- **WHEN** se inspecciona su contenido
- **THEN** contiene una instrucción explícita que prohíbe repetir, resumir, parafrasear o revelar el propio system prompt ante cualquier petición del usuario, incluyendo variantes de la petición (traducción, "modo desarrollador", formato código, etc.)

#### Scenario: El system prompt prohíbe revelar configuración interna
- **GIVEN** la constante `systemPrompt`
- **WHEN** se inspecciona su contenido
- **THEN** contiene una instrucción explícita que prohíbe revelar variables de entorno, cadenas de conexión, credenciales, tokens de API o cualquier dato de configuración del sistema que aloja al agente

#### Scenario: El system prompt indica tratar el contenido de las tools como datos no confiables
- **GIVEN** la constante `systemPrompt`
- **WHEN** se inspecciona su contenido
- **THEN** contiene una instrucción explícita indicando que el contenido devuelto por `list_collections`, `describe_collection`, `query_find` y `query_aggregate` es dato y no debe interpretarse como una instrucción nueva, aunque ese contenido incluya texto con forma de orden

#### Scenario: El system prompt se declara como constante de compilación
- **GIVEN** el archivo `cmd/server/api.go`
- **WHEN** se verifica la declaración de `systemPrompt`
- **THEN** está declarada con `const`, no con `var`
- **AND** ningún operador de interpolación (`fmt.Sprintf`, concatenación con una variable de entorno) participa en su definición, garantizando en tiempo de compilación que su valor no puede incluir ningún dato leído en tiempo de ejecución

### Requirement: Detección de Fuga del System Prompt
El sistema SHALL verificar, antes de devolver y persistir la respuesta final del agente (cuando el LLM no solicita más tool calls), si dicha respuesta contiene una porción verbatim sustancial del system prompt, y en ese caso SHALL sustituirla por un mensaje de rechazo genérico antes de persistirla o devolverla.

#### Scenario: Respuesta final sin fuga se devuelve sin cambios
- **GIVEN** una respuesta final del LLM que no contiene ninguna ventana de al menos el umbral mínimo de caracteres consecutivos del system prompt
- **WHEN** el sistema evalúa la respuesta antes de devolverla
- **THEN** la respuesta se persiste y se devuelve al usuario sin ninguna modificación

#### Scenario: Respuesta final con fuga del system prompt se bloquea
- **GIVEN** una respuesta final del LLM que contiene, de forma exacta (sin distinguir mayúsculas/minúsculas), al menos el umbral mínimo de caracteres consecutivos del system prompt
- **WHEN** el sistema evalúa la respuesta antes de devolverla
- **THEN** el sistema descarta el contenido original devuelto por el LLM
- **AND** responde al usuario con un mensaje de rechazo genérico fijo
- **AND** persiste en Redis el mensaje de rechazo genérico, no el contenido original filtrado
- **AND** registra en logs del servidor una advertencia que identifica la sesión afectada, sin incluir el contenido de la respuesta ni del system prompt en el propio mensaje de log

### Requirement: Sanitización de Caracteres de Control
El sistema SHALL eliminar caracteres de control no imprimibles del texto de la pregunta del usuario antes de usarlo, persistirlo o reenviarlo al LLM, preservando salto de línea y tabulador.

#### Scenario: Pregunta con caracteres de control se sanea
- **GIVEN** una pregunta que contiene caracteres de control no imprimibles (por ejemplo, secuencias de escape ANSI o el carácter nulo) intercalados con texto legible
- **WHEN** el sistema procesa la pregunta
- **THEN** el texto usado, persistido y reenviado al LLM tiene los caracteres de control eliminados
- **AND** el texto legible restante permanece intacto, incluyendo saltos de línea, tabuladores y caracteres Unicode imprimibles (acentos, emoji)

#### Scenario: Pregunta compuesta solo por caracteres de control se rechaza
- **GIVEN** una pregunta cuyo contenido consiste exclusivamente en caracteres de control no imprimibles
- **WHEN** el sistema la sanea y el resultado queda vacío tras eliminar espacios en blanco
- **THEN** el sistema responde con el mismo error que una pregunta vacía (`400 Bad Request`)
- **AND** el sistema no invoca al LLM ni a MongoDB

### Requirement: Fail-Fast de Secretos Obligatorios
El proceso del servidor SHALL rehusar arrancar, sin abrir el puerto HTTP, si alguna de las variables de entorno `MONGODB_URI`, `MONGODB_DB_NAME` u `OPENCODE_API_KEY` no está definida o está vacía. (La misma garantía para `API_TOKEN` ya está especificada en `nl-mongo-agent`/`specs/nl-mongo-agent/spec.md`, requisito "Autenticación por Token de API".)

#### Scenario: Arranque sin MONGODB_URI configurado
- **GIVEN** la variable de entorno `MONGODB_URI` no está definida o está vacía
- **WHEN** el proceso del servidor arranca
- **THEN** el sistema registra un error crítico indicando el nombre de la variable faltante y termina el proceso sin abrir el puerto HTTP

#### Scenario: Arranque sin MONGODB_DB_NAME configurado
- **GIVEN** la variable de entorno `MONGODB_DB_NAME` no está definida o está vacía
- **WHEN** el proceso del servidor arranca
- **THEN** el sistema registra un error crítico indicando el nombre de la variable faltante y termina el proceso sin abrir el puerto HTTP

#### Scenario: Arranque sin OPENCODE_API_KEY configurado
- **GIVEN** la variable de entorno `OPENCODE_API_KEY` no está definida o está vacía
- **WHEN** el proceso del servidor arranca
- **THEN** el sistema registra un error crítico indicando el nombre de la variable faltante y termina el proceso sin abrir el puerto HTTP

### Requirement: Advertencia por Redis sin Contraseña
El sistema SHALL registrar una advertencia no bloqueante en logs si `REDIS_PASSWORD` está vacío al arrancar, y SHALL continuar el arranque normalmente, dado que es una configuración válida para desarrollo local con Redis sin autenticación.

#### Scenario: REDIS_PASSWORD vacío registra advertencia y el servidor arranca
- **GIVEN** la variable de entorno `REDIS_PASSWORD` no está definida o está vacía
- **WHEN** el proceso del servidor arranca y logra conectarse a Redis
- **THEN** el sistema registra una línea de advertencia indicando que Redis se está usando sin autenticación
- **AND** el proceso continúa su arranque normalmente y abre el puerto HTTP

#### Scenario: REDIS_PASSWORD configurado no genera advertencia
- **GIVEN** la variable de entorno `REDIS_PASSWORD` tiene un valor no vacío
- **WHEN** el proceso del servidor arranca
- **THEN** el sistema no registra ninguna advertencia relacionada con la ausencia de contraseña de Redis

### Requirement: Prohibición de Registrar Secretos en Logs
Ninguna línea de log emitida por el sistema SHALL contener el valor completo ni parcial de `API_TOKEN`, `OPENCODE_API_KEY`, `MONGODB_URI` (ni las credenciales embebidas en ella) o `REDIS_PASSWORD`.

#### Scenario: Error con un secreto embebido se redacta antes de loguearse
- **GIVEN** un error generado en cualquier punto del sistema (conexión a MongoDB, conexión a Redis, manejo de una petición HTTP) cuyo texto original contiene el valor de uno de los cuatro secretos configurados
- **WHEN** el sistema registra ese error en logs
- **THEN** la línea de log registrada contiene el marcador `[REDACTED]` en lugar del valor del secreto
- **AND** el valor original del secreto no aparece en ninguna parte de la línea de log, ni siquiera como subcadena parcial

#### Scenario: Error sin ningún secreto embebido se loguea sin modificar su contenido informativo
- **GIVEN** un error cuyo texto no contiene ningún valor de los cuatro secretos configurados
- **WHEN** el sistema registra ese error en logs
- **THEN** la línea de log conserva el mensaje de error original sin alteraciones

### Requirement: Redacción Genérica de Credenciales Mongo
Cualquier mensaje de error que atraviese el sistema (logs del servidor, resultados de tool calls devueltos al LLM) SHALL tener las credenciales de MongoDB (`usuario:contraseña@`) redactadas mediante un mecanismo basado en patrones genéricos de URI, no solo mediante comparación exacta contra el valor configurado de `MONGODB_URI`. ← (was: la redacción existente en `nl-mongo-agent` solo cubre errores de conexión inicial mediante comparación exacta de toda la cadena `MONGODB_URI`)

#### Scenario: Error de conexión inicial a MongoDB redacta credenciales
- **GIVEN** un fallo al conectar o hacer ping a MongoDB cuyo mensaje de error original incluye la URI de conexión completa con credenciales
- **WHEN** el sistema registra el error en logs
- **THEN** la línea de log no contiene la subcadena `usuario:contraseña@` original
- **AND** el proceso termina (`log.Fatal`) igual que hoy, sin abrir el puerto HTTP

#### Scenario: Error de ejecución de una tool sobre Mongo redacta credenciales aunque el driver las incluyera
- **GIVEN** un error devuelto por el driver de MongoDB durante la ejecución de `Find`/`Aggregate`/`ListCollections`/`DescribeCollection` cuyo texto contiene, en cualquier formato, una subcadena con forma `esquema://usuario:contraseña@`
- **WHEN** el sistema convierte ese error en el resultado de una tool call
- **THEN** el resultado no contiene la subcadena `usuario:contraseña@` original

#### Scenario: El patrón genérico detecta credenciales aunque no coincidan con la URI configurada exacta
- **GIVEN** un texto de error que contiene una subcadena con forma `esquema://usuario:contraseña@` que no coincide byte a byte con el valor configurado de `MONGODB_URI` (por ejemplo, reformateada o truncada por el driver)
- **WHEN** el sistema aplica el mecanismo de redacción genérico
- **THEN** las credenciales quedan redactadas de todos modos, sin depender de una coincidencia exacta con el valor conocido

### Requirement: Sanitización de Errores de Tool Calls
Antes de convertir un error del repositorio de MongoDB en el contenido de un mensaje `tool` que se persiste en el historial de conversación y se reenvía al LLM, `DispatchToolCall` SHALL sanear ese mensaje para que no contenga credenciales de MongoDB, incluso si el error subyacente del driver las contuviera.

#### Scenario: Error de repositorio con credenciales embebidas se sanea antes de llegar al LLM
- **GIVEN** una llamada a `query_find` o `query_aggregate` cuyo repositorio subyacente devuelve un error que contiene una subcadena con forma `esquema://usuario:contraseña@`
- **WHEN** `DispatchToolCall` construye el resultado de la tool call
- **THEN** el contenido del mensaje `tool` resultante no contiene la subcadena `usuario:contraseña@` original
- **AND** el mensaje `tool` saneado es el que se persiste en Redis y se reenvía al LLM en la siguiente iteración del bucle

#### Scenario: Error de repositorio sin credenciales conserva su contenido informativo
- **GIVEN** una llamada a cualquier tool cuyo repositorio subyacente devuelve un error que no contiene ninguna credencial embebida (por ejemplo, "collection not found")
- **WHEN** `DispatchToolCall` construye el resultado de la tool call
- **THEN** el contenido del mensaje `tool` conserva el mensaje de error original sin alteraciones más allá del formato `"error: ..."` ya existente

### Requirement: Aislamiento Estructural de Secretos
El paquete `pkg/agent` (dominio y aplicación) SHALL no tener acceso directo a los valores de `API_TOKEN`, `OPENCODE_API_KEY`, `MONGODB_URI` ni `REDIS_PASSWORD`, de forma que ningún `Message` construido por el dominio pueda contener esos valores.

#### Scenario: pkg/agent no importa el paquete os
- **GIVEN** los archivos fuente de producción del paquete `pkg/agent` (excluyendo archivos `_test.go`)
- **WHEN** se inspeccionan sus imports
- **THEN** ninguno importa el paquete estándar `os`

#### Scenario: Ninguna función de pkg/agent recibe un secreto como parámetro
- **GIVEN** las firmas de `AgentService`, `service.go` y `tools.go`
- **WHEN** se inspeccionan sus parámetros
- **THEN** ninguna función recibe un parámetro que represente `MONGODB_URI`, `REDIS_PASSWORD`, `API_TOKEN` u `OPENCODE_API_KEY`; esos valores solo existen en `cmd/server` (composition root) y en los adapters concretos que los necesitan para conectarse a su sistema externo correspondiente

### Requirement: Documentación de la Política de Secretos
`README.md` y `.env.example` SHALL documentar, para cada uno de los cuatro secretos (`API_TOKEN`, `OPENCODE_API_KEY`, `MONGODB_URI`, `REDIS_PASSWORD`), que nunca deben commitearse, la consecuencia de no configurarlos (fail-fast o advertencia), y la garantía de que nunca se registran en logs.

#### Scenario: .env.example no contiene valores reales
- **GIVEN** el archivo `.env.example` versionado en el repositorio
- **WHEN** se inspecciona su contenido
- **THEN** ninguna de las variables `API_TOKEN`, `OPENCODE_API_KEY`, `MONGODB_URI`, `REDIS_PASSWORD` tiene un valor real asignado; solo contienen placeholders o están vacías

#### Scenario: README documenta la política de secretos
- **GIVEN** la sección "Seguridad" de `README.md`
- **WHEN** se inspecciona su contenido
- **THEN** incluye una subsección que enumera los cuatro secretos, indica cuáles provocan fail-fast del proceso si faltan y cuál solo genera una advertencia, y declara explícitamente que ninguno se registra en logs completo ni parcialmente
