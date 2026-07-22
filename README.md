# mongo-agent

> Agente conversacional en lenguaje natural sobre MongoDB, de **solo lectura**
> y con **autodescubrimiento de esquema** — proyecto de portafolio que
> demuestra arquitectura hexagonal en Go, hardening de seguridad de un agente
> LLM con tool-calling, y una interfaz de chat servida sin build de frontend
> (htmx).

[![Go](https://img.shields.io/badge/Go-1.25-00ADD8?logo=go&logoColor=white)](https://go.dev)
[![Architecture](https://img.shields.io/badge/architecture-hexagonal-blueviolet)](#arquitectura-resumen)
[![Tests](https://img.shields.io/badge/tests-unit%20%2B%20integration-brightgreen)](#tests)
[![Status](https://img.shields.io/badge/status-portfolio%20project-orange)](#creditos-y-alcance)

## Que es esto

El objetivo de este repositorio es documentar y demostrar mi proceso de
desarrollo actual: especificar cada feature con
[OpenSpec](https://github.com/Fission-AI/OpenSpec) — propuesta, arquitectura,
requisitos con escenarios verificables y un plan de tareas con criterios de
"listo" ejecutables por comando — *antes* de escribir código, y usar esa
especificación (no un prompt suelto) para dirigir la implementación asistida
por IA. Yo diseño la arquitectura, tomo las decisiones técnicas y defino los
criterios de aceptación; el código se genera contra ese contrato y cada
resultado se valida con tests automatizados, no a ojo. Este repositorio es
evidencia de ese proceso completo — incluyendo el hardening de seguridad
explícito para un agente que expone un LLM con tool-calling a usuarios
externos — no solo del binario final.
Todo esto bajo el agentes y subagentes, encargados de objetivos sencillos y acotados.

`mongo-agent` recibe preguntas en español por HTTP o por una interfaz
web de chat, **descubre por sí mismo el esquema de una base MongoDB** (sin un
modelo de colecciones hardcodeado) y responde consultándola en tiempo real —
siempre en modo lectura. Internamente es un bucle de tool-calling contra un
LLM (OpenCode Zen): el modelo decide qué herramientas de solo lectura invocar
(`list_collections`, `describe_collection`, `query_find`, `query_aggregate`)
antes de contestar, y el agente mantiene memoria de conversación por sesión en
Redis.


### Lo que demuestra este proyecto

- **Arquitectura hexagonal real, no solo de nombre**: el dominio
  (`pkg/agent`) no importa `gin`, `mongo-driver`, `redis` ni `os` — verificado
  con tests y comprobaciones estáticas (`grep`), no solo por convención. Ver
  [Arquitectura](#arquitectura-resumen).
- **Seguridad de un agente LLM expuesto a usuarios externos**: defensa contra
  *prompt injection* directo e indirecto (datos adversariales dentro de los
  propios documentos de Mongo), detección de fuga del system prompt en la
  respuesta final, redacción de secretos en logs por dos mecanismos
  independientes, y aislamiento estructural de credenciales fuera del
  dominio. Ver [Seguridad](#seguridad).
- **Autodescubrimiento de esquema**: el agente no tiene un modelo de datos
  hardcodeado; explora `list_collections`/`describe_collection` antes de
  decidir qué consultar, así que responde sobre cualquier colección sin
  cambios de código.
- **Garantía de solo lectura en 4 capas independientes**, desde el rol del
  usuario de MongoDB Atlas hasta una verificación activa al arrancar que
  intenta escribir un documento canario y exige que falle. Ver
  [Garantía de solo lectura](#garantia-de-solo-lectura).
- **Frontend sin build step**: interfaz de chat con pestañas multi-sesión,
  servida como HTML+htmx desde el mismo binario Go — sin Node.js, sin
  bundler, sin JavaScript framework. Ver
  [Interfaz web de chat](#interfaz-web-de-chat).
- **Proceso spec-first para dirigir desarrollo asistido por IA**: cada
  feature (`openspec/changes/*/proposal.md`, `design.md`, `spec.md`,
  `tasks.md`) está diseñada y documentada por mí *antes* de escribir código,
  con criterios de "listo" verificables por comando — esa especificación,
  no un prompt suelto, es lo que guía la implementación. Ver
  [Créditos y alcance](#creditos-y-alcance).
- **Testing en las tres capas**: unitarios con fakes en dominio/aplicación,
  tests HTTP con `httptest`, e integración contra MongoDB/Redis/LLM reales.
  Ver [Tests](#tests).

Toda la planificación técnica (propuesta, diseño, specs y tareas de
implementación) vive en `openspec/changes/`:

- [`add-nl-mongo-agent/`](./openspec/changes/add-nl-mongo-agent/) — el agente
  base: bucle de tool-calling, garantía de solo lectura, API REST.
- [`add-htmx-chat-frontend/`](./openspec/changes/add-htmx-chat-frontend/) —
  la interfaz web de chat multi-sesión.
- [`harden-agent-security/`](./openspec/changes/harden-agent-security/) —
  defensa contra prompt injection y gestión verificable de secretos.

Cada change incluye `proposal.md` (qué cambia y por qué), `design.md`
(arquitectura, decisiones, contratos de puertos), `specs/*/spec.md`
(requisitos y escenarios observables) y `tasks.md` (plan de implementación
desglosado en tareas atómicas).

> Estado: **implementado**. El código en `pkg/`, `cmd/` y `test/` contiene la
> lógica de negocio completa, tests unitarios y de integración. Ver el
> `tasks.md` de cada change para el detalle de cada tarea completada.

## Tabla de Contenidos

- [Que es esto](#que-es-esto)
- [Arquitectura (resumen)](#arquitectura-resumen)
- [Garantia de solo lectura](#garantia-de-solo-lectura)
- [Variables de entorno](#variables-de-entorno)
- [Correr localmente](#correr-localmente)
- [Uso](#uso)
- [Endpoints](#endpoints)
- [Interfaz web de chat](#interfaz-web-de-chat)
- [Seguridad](#seguridad)
- [Docker](#docker)
- [Tests](#tests)
- [Creditos y alcance](#creditos-y-alcance)

## Arquitectura (resumen)

Hexagonal / Ports & Adapters, replicando las convenciones de
[`CrudGoExample`](https://github.com/HongXiangZuniga/CrudGoExample):

```
pkg/agent/                dominio + ports de salida (LLMClient, ReadOnlyMongoRepository, SessionStore) + caso de uso
pkg/persistence/mongodb/  adapter de salida: acceso a MongoDB, SOLO lectura
pkg/persistence/redis/    adapter de salida: memoria de conversacion por sesion
pkg/llm/opencodezen/      adapter de salida: cliente HTTP hacia OpenCode Zen
pkg/http/rest/            adapter de entrada: API HTTP (gin)
pkg/http/web/             adapter de entrada: interfaz de chat HTML+htmx
pkg/utils/                errores de dominio compartidos
cmd/server/               composition root (wiring explicito por constructor)
test/                     tests unitarios (dominio/aplicacion con fakes) e integracion (adapters reales)
```

Ver el diagrama completo en
[`design.md`](./openspec/changes/add-nl-mongo-agent/design.md).

**Regla de dependencia**: `pkg/agent` no importa `gin`, `mongo-driver`, `redis`
ni ningun cliente HTTP concreto. Los adapters (`pkg/persistence/*`,
`pkg/llm/*`) importan `pkg/agent` para implementar sus interfaces, nunca al
reves. `pkg/http/rest` importa `pkg/agent` pero no al reves.
`cmd/server/api.go` es el unico lugar que conoce todos los paquetes concretos
y los conecta via constructor injection explicito.

## Garantia de solo lectura

El agente **nunca** puede escribir, actualizar o borrar datos en MongoDB.
Esto se garantiza en 4 capas independientes (detalladas en `design.md`):

1. **Infraestructura**: el usuario de MongoDB configurado en `MONGODB_URI`
   tiene rol `read` en el cluster de Atlas.
2. **Contrato del puerto**: `ReadOnlyMongoRepository` no declara ningun metodo
   de escritura. El compilador de Go impide invocar operaciones de escritura
   desde `pkg/agent`.
3. **Validacion del adapter**: `Aggregate` rechaza pipelines con stages de
   escritura (`$out`, `$merge`, incluso anidados en sub-pipelines) antes de
   llamar al driver.
4. **Verificacion activa de arranque**: al iniciar, el proceso ejecuta
   `VerifyReadOnlyGuarantee` que intenta insertar un documento canario y
   espera que MongoDB lo rechace. Si la escritura tiene exito, el proceso se
   niega a arrancar (`log.Fatal`).

## Variables de entorno

Ver [`.env.example`](./.env.example) para la lista completa. Resumen:

| Variable | Descripcion | Default | Requerida |
|---|---|---|---|
| `PORT` | Puerto HTTP del servidor | `8080` | No |
| `API_TOKEN` | Token estatico requerido en el header `Authorization` | — | **Si** |
| `MONGODB_URI` | Cadena de conexion a MongoDB Atlas (usuario de solo lectura) | — | **Si** |
| `MONGODB_DB_NAME` | Base de datos a consultar | — | **Si** |
| `MONGODB_QUERY_TIMEOUT_SECONDS` | Timeout por operacion contra MongoDB | `10` | No |
| `MONGODB_MAX_RESULT_LIMIT` | Limite maximo de documentos devueltos por consulta | `50` | No |
| `MONGO_SAMPLE_SIZE` | Tamano de muestra al describir una coleccion | `5` | No |
| `OPENCODE_API_KEY` | API key de OpenCode Zen | — | **Si** |
| `OPENCODE_BASE_URL` | Base URL del endpoint OpenAI-compatible de OpenCode Zen | `https://opencode.ai/zen/v1` | No |
| `OPENCODE_MODEL` | Modelo a usar: el `<model-id>` de `GET {OPENCODE_BASE_URL}/models`, sin prefijo. Los modelos con sufijo `-free` no consumen saldo. | `deepseek-v4-flash-free` | No |
| `AGENT_MAX_TOOL_ITERATIONS` | Maximo de iteraciones de tool-calling por pregunta | `6` | No |
| `AGENT_REQUEST_TIMEOUT_SECONDS` | Timeout total por peticion | `30` | No |
| `REDIS_ADDR` | Direccion del servidor Redis | `localhost:6379` | No |
| `REDIS_PASSWORD` | Password de Redis (vacio en desarrollo local) | `""` | No |
| `REDIS_DB` | Indice de base Redis | `0` | No |
| `SESSION_TTL_SECONDS` | TTL del historial de conversacion por sesion | `3600` | No |
| `AUTH_MONGODB_URI` | Cadena de conexion a la base de usuarios del login (instancia MongoDB **local**, separada del cluster Atlas de solo lectura del agente). En docker-compose el host es `mongo-auth`. | — | **Si** |
| `AUTH_MONGODB_DB_NAME` | Base de datos de usuarios del login | — | **Si** |
| `AUTH_MONGODB_USERS_COLLECTION` | Coleccion de usuarios en la base de login | `users` | No |

> La base de usuarios del login (`AUTH_MONGODB_*`) es una instancia MongoDB
> **local y dedicada**, completamente separada del cluster MongoDB Atlas de
> solo lectura del agente (`MONGODB_URI`). Solo la conexion del agente se
> verifica con `mongodb.VerifyReadOnlyGuarantee`; la base de usuarios nunca se
> mezcla con esa garantia.

## Correr localmente

### Requisitos

- Go 1.25+
- Docker (para Redis via `docker-compose`)
- Cuenta en MongoDB Atlas con usuario de solo lectura
- API key de OpenCode Zen

### Pasos

```bash
# 1. Clonar el repositorio
git clone git@github.com:HongXiangZuniga/mongo-agent.git
cd mongo-agent

# 2. Copiar y completar variables de entorno
cp .env.example .env
# Editar .env con credenciales reales (nunca commitear este archivo)

# 3. Levantar Redis (via docker-compose) y ejecutar el servidor
make run
```

`make run` levanta Redis en segundo plano con `docker-compose up -d` y luego
ejecuta `go run cmd/server/api.go`. MongoDB es el cluster de Atlas remoto (no
requiere contenedor local).

### Llegar y montar con docker-compose

Para levantar **toda la topologia con un solo comando** (Redis + base de
usuarios MongoDB ya seedeada + API Go), sin instalar nada localmente salvo
Docker:

```bash
# 1. Completar .env con los secretos externos:
#    API_TOKEN, MONGODB_URI (Atlas), MONGODB_DB_NAME, OPENCODE_API_KEY.
#    (Las variables AUTH_MONGODB_* las sobreescribe docker-compose hacia el
#     servicio mongo-auth; no necesitas tocarlas para el arranque integral.)
cp .env.example .env

# 2. Levantar todo:
docker-compose up --build   # o: make compose-up
```

`docker-compose up --build` construye la API desde `build/docker/Dockerfile` y
levanta tres servicios enlazados en la misma red:

- `redis` — memoria de conversacion por sesion (con healthcheck).
- `mongo-auth` — MongoDB **local** de usuarios del login; en su primera
  inicializacion ejecuta `build/mongo-auth/seed.js` (montado en
  `/docker-entrypoint-initdb.d`) y precarga el usuario de prueba (con healthcheck).
- `api` — el binario Go; **arranca solo cuando Redis y mongo-auth reportan
  estado saludable**. Alcanza sus dependencias por nombre de servicio
  (`redis:6379`, `mongodb://mongo-auth:27017`), no por `localhost`.

Tras el arranque, abre `http://localhost:8080/web`, seras redirigido al login
y podras entrar con el [usuario de prueba](#usuario-de-prueba-login-web).

El seed solo se ejecuta en la **primera** inicializacion del volumen
`mongo_auth_data`. Para reejecutarlo desde cero: `docker-compose down -v` y
luego `docker-compose up --build`.

#### Usuario de prueba (login web)

La base de usuarios se precarga (via seed) con un unico usuario de prueba,
pensado **solo para la POC**:

- **Usuario:** `admin`
- **Contraseña:** `admin123`

La contraseña se almacena **hasheada con bcrypt** (`password_hash`) en la base
de usuarios; el texto plano `admin123` solo aparece en esta documentacion,
nunca en la base de datos ni en `seed.js`.

### Otros comandos

```bash
make build            # compila el binario en ./api
make unit-test        # tests unitarios (dominio + aplicacion con fakes + HTTP con httptest)
make integration-test # tests de integracion (requiere credenciales reales en .env)
make docker-build     # construye la imagen Docker multi-stage
make docker-run       # levanta Redis y ejecuta el contenedor Docker
```

## Uso

Una vez que el servidor esta corriendo en `http://localhost:8080`:

```bash
curl -X POST http://localhost:8080/ask \
  -H "Authorization: <api_token>" \
  -H "Content-Type: application/json" \
  -d '{"session_id": "demo-1", "question": "Cuantos documentos hay en total?"}'
```

Si no se envia `session_id`, el servidor genera uno automaticamente (UUID) y
lo devuelve en la respuesta.

### Respuesta exitosa

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "session_id": "demo-1",
    "answer": "Hay 42 documentos en la coleccion 'cards'."
  }
}
```

## Endpoints

### POST /ask

Recibe una pregunta en lenguaje natural, la procesa a traves del bucle de
tool-calling del agente y devuelve una respuesta en texto.

- **Request body**:
  ```json
  {
    "session_id": "opcional-solo-alfanumerico",
    "question": "texto de la pregunta (requerido)"
  }
  ```
  - `session_id`: si se omite, se genera un UUID automaticamente. Si se
    incluye, debe cumplir el patron `^[a-zA-Z0-9_-]+$`.
  - `question`: texto de la pregunta. Longitud maxima del body completo:
    64 KB.

- **Response (200)**:
  ```json
  {
    "code": 200,
    "message": "success",
    "data": {
      "session_id": "demo-1",
      "answer": "Respuesta generada por el agente."
    }
  }
  ```

- **Headers requeridos**: `Authorization: <api_token>` (token estatico
  configurado en `API_TOKEN`).

### Codigos de error

| Codigo HTTP | Significado | Causa tipica |
|---|---|---|
| `400` | Bad Request | `question` ausente o vacio; `session_id` con formato invalido |
| `401` | Unauthorized | Header `Authorization` ausente o token incorrecto |
| `502` | Bad Gateway | El agente no produjo respuesta final dentro del limite de iteraciones (`AGENT_MAX_TOOL_ITERATIONS`); el proveedor LLM no esta disponible |
| `503` | Service Unavailable | Redis no responde o no se puede leer/escribir el historial de sesion |
| `504` | Gateway Timeout | La peticion supero el tiempo maximo de procesamiento (`AGENT_REQUEST_TIMEOUT_SECONDS`) |

## Interfaz web de chat

Además de la API REST (`POST /ask`), el proyecto incluye una interfaz de chat
HTML renderizada en servidor, accesible desde el navegador en
`http://localhost:8080/web/login`. Esta interfaz permite mantener varias
conversaciones en paralelo como pestañas (una pestaña = una sesión = un
`session_id`).

- **Tecnología**: HTML renderizado con `html/template` de la librería estándar,
  interactividad vía [htmx](https://htmx.org) (`htmx.min.js` v1.9.12 vendored
  en `pkg/http/web/static/` y servido sin build de frontend ni Node.js).
- **Pestañas (sesiones)**: se muestran como pestañas en el navegador. Se
  listan ordenadas por actividad más reciente. El historial de cada sesión se
  persiste en Redis (mismo TTL que el historial del agente,
  `SESSION_TTL_SECONDS`). Una pestaña nueva no se persiste hasta que se envía
  el primer mensaje.
- **Endpoints**:
  - `GET /web/login` — formulario de login.
  - `POST /web/login` — enviar **usuario y contraseña** (`username`,
    `password`), validados contra la base MongoDB de usuarios (verificacion
    bcrypt); si son correctos se obtiene una cookie de autenticación.
  - `POST /web/logout` — cerrar sesión: revoca la sesión del lado servidor (borra su key en Redis) y expira la cookie.
  - `GET /web` — página principal con la barra de pestañas y el chat activo
    (requiere cookie).
  - `POST /web/tabs` — crear una pestaña nueva (efímera hasta el primer
    mensaje).
  - `GET /web/tabs/:sessionId` — cambiar a una pestaña existente.
  - `POST /web/tabs/:sessionId/messages` — enviar un mensaje.
  - `DELETE /web/tabs/:sessionId` — cerrar una pestaña y eliminar su sesión.
- **Autenticación**: el login web usa **usuario y contraseña** validados
  contra una base MongoDB local de usuarios, con la contraseña verificada por
  **bcrypt** (nunca en texto plano). Tras un login correcto la interfaz `/web`
  crea una **sesión de servidor**: un identificador opaco (256 bits, aleatorio)
  guardado en Redis (namespace `web_session:*`) asociado al usuario autenticado.
  Ese session ID viaja en una cookie HttpOnly, `Path=/web`, `SameSite=Strict`,
  `Secure` configurable (`WEB_COOKIE_SECURE`) y se **valida contra Redis en cada
  petición**. La cookie ya **no transporta `API_TOKEN`** ni la contraseña del
  usuario: conocer `API_TOKEN` ya no permite entrar a `/web`. El logout
  **invalida la sesión del lado servidor** (borra su key en Redis), de modo que
  una cookie robada deja de servir aunque el cliente la conserve. `POST /ask`
  (API REST) no cambia: se sigue autenticando con el header `Authorization` ==
  `API_TOKEN`, independiente del login web. La variable
  `WEB_SESSION_MAX_AGE_SECONDS` (por defecto 7 días) fija a la vez el `MaxAge` de
  la cookie y el TTL de la sesión web en Redis, para que ambos caduquen juntos.
  Credenciales de prueba: ver [Usuario de prueba (login web)](#usuario-de-prueba-login-web).
- **Variables de entorno específicas**: `WEB_AUTH_COOKIE_NAME`,
  `WEB_COOKIE_SECURE`, `WEB_SESSION_MAX_AGE_SECONDS`,
  `WEB_SESSION_TITLE_MAX_LENGTH` — ver `.env.example`.
- **Implementación completa**: la propuesta, diseño detallado y especificación
  de esta interfaz viven en
  `openspec/changes/add-htmx-chat-frontend/`, en particular el
  [`design.md`](./openspec/changes/add-htmx-chat-frontend/design.md), la
  [`spec`](./openspec/changes/add-htmx-chat-frontend/specs/chat-web-frontend/spec.md)
  y el
  [`design-ui.md`](./openspec/changes/add-htmx-chat-frontend/design-ui.md)
  (decisiones de paleta, tipografía y accesibilidad).

## Seguridad

El proyecto implementa las siguientes medidas de seguridad:

- **Autenticacion por token estatico**: toda peticion a `POST /ask` debe
  incluir el header `Authorization` con el token configurado en `API_TOKEN`.
  La comparacion se realiza con `crypto/subtle.ConstantTimeCompare` para
  evitar ataques de timing.
- **Limite de cuerpo**: el body de la peticion HTTP esta limitado a 64 KB
  mediante `http.MaxBytesReader`.
- **Validacion de `session_id`**: se rechazan `session_id` con caracteres no
  alfanumericos (solo `^[a-zA-Z0-9_-]+$`).
- **Mensajes de error genericos**: los errores de dependencias externas
  (MongoDB, Redis, LLM) jamas exponen `err.Error()` crudo al cliente; se
  devuelve un mensaje generico y el detalle se registra solo en logs del
  servidor.
- **Redaccion de credenciales en logs**: la URI de MongoDB (que contiene
  usuario y password) se redacta automaticamente (`<redacted-mongodb-uri>`)
  en los mensajes de error de conexion antes de escribirlos al log.
- **Verificacion de solo lectura al arrancar**: el proceso no arranca si el
  usuario de MongoDB puede escribir (`VerifyReadOnlyGuarantee`).
- **Rechazo de pipelines peligrosos**: `DispatchToolCall` rechaza stages
  `$out` y `$merge` en pipelines de agregacion, incluyendo aquellos anidados
  en sub-pipelines (ej. dentro de `$facet`), antes de enviarlos al driver de
  MongoDB.
- **Contenedor no-root**: la imagen Docker ejecuta el binario con un usuario
  sin privilegios (`appuser`). Ver seccion Docker.
- **.dockerignore**: excluye `.env`, `openspec/`, `test/`, `Makefile` y otros
  archivos que no deben estar en la imagen de produccion.
- **Defensa contra prompt injection**: el system prompt prohibe explicitamente
  revelar su propio contenido y cualquier dato de configuracion interna, y
  ordena tratar todo el contenido devuelto por las tools de MongoDB
  (`list_collections`, `describe_collection`, `query_find`, `query_aggregate`)
  como datos, nunca como instrucciones nuevas -- esto cubre tanto intentos
  directos del usuario como inyeccion indirecta via documentos adversariales
  almacenados en la propia base de datos. Como segunda linea de defensa,
  antes de devolver y persistir la respuesta final se verifica si contiene
  una porcion verbatim sustancial del system prompt; si la detecta, la
  sustituye por un mensaje de rechazo generico.

### Manejo de secretos

El sistema depende de cinco variables de entorno relacionadas con secretos:

| Variable | Si falta o esta vacia |
|---|---|
| `API_TOKEN` | El proceso no arranca (`log.Fatal`, sin abrir el puerto HTTP) |
| `MONGODB_URI` | El proceso no arranca (`log.Fatal`) |
| `MONGODB_DB_NAME` | El proceso no arranca (`log.Fatal`) |
| `OPENCODE_API_KEY` | El proceso no arranca (`log.Fatal`) |
| `REDIS_PASSWORD` | El proceso arranca igualmente, pero registra una advertencia en logs (configuracion valida solo para desarrollo local con Redis sin autenticacion) |

Garantias adicionales:

- Ningun secreto (`API_TOKEN`, `MONGODB_URI`, `OPENCODE_API_KEY`,
  `REDIS_PASSWORD`) se registra en logs, ni completo ni parcialmente: un
  `SecretScrubber` construido una sola vez al arrancar redacta por
  coincidencia exacta cualquier aparicion de estos valores antes de que un
  mensaje de error llegue a un `log.Printf`/`log.Fatalf`, tanto en el
  composition root (`cmd/server/api.go`) como en los adapters HTTP
  (`pkg/http/rest`, `pkg/http/web`).
- Adicionalmente, cualquier cadena con forma `esquema://usuario:contraseña@`
  (no solo el valor exacto de `MONGODB_URI` conocido) se redacta de forma
  generica mediante una expresion regular, cubriendo errores del driver de
  MongoDB que reformateen o trunquen la URI original. Esto se aplica en la
  conexion a MongoDB, la conexion a Redis, y en los errores de tool calls que
  se reenvian al LLM (`DispatchToolCall`).
- Los mensajes de error expuestos al cliente HTTP y al LLM son siempre
  genericos (`utils.HTTPStatusForError`) o han pasado por la redaccion de
  credenciales de MongoDB; nunca se expone `err.Error()` crudo.
- `pkg/agent` no importa el paquete `os` ni recibe ninguna variable de
  entorno como parametro: estructuralmente no tiene acceso a los secretos,
  por lo que ni el system prompt ni el historial de conversacion enviado al
  LLM pueden llegar a incluir `MONGODB_URI` u otra variable de entorno.

## Docker

### Dockerfile

Build multi-stage:

- **Stage 1 (builder)**: `golang:1.25-alpine`. Compila el binario con
  `go build -o /app/api ./cmd/server`.
- **Stage 2 (final)**: `alpine:3.19`. Crea el usuario `appuser` sin
  privilegios, copia el binario, expone el puerto `8080`, y ejecuta como
  `ENTRYPOINT ["./api"]`.

### Construir y ejecutar

```bash
make docker-build
make docker-run
```

`make docker-run` levanta Redis via `docker-compose up -d` y luego ejecuta el
contenedor con `--env-file ./.env` y mapeo de puerto `8080:8080`.

## Tests

```bash
make unit-test         # dominio + aplicacion (fakes) + HTTP (httptest)
make integration-test  # adapters reales: requiere MONGODB_URI/REDIS_ADDR/OPENCODE_API_KEY
```

Los tests unitarios cubren:

- `TestAsk_DirectAnswerWithoutTools`: respuesta directa sin invocar tools.
- `TestAsk_SingleToolCallThenAnswer`: un tool call seguido de respuesta.
- `TestAsk_MultipleIterations`: multiples tool calls en secuencia.
- `TestAsk_ExceedsMaxIterations`: el bucle se detiene al alcanzar el maximo
  de iteraciones (espera `ErrToolLoopExceeded`).
- `TestAsk_LLMError`: error del proveedor LLM propagado correctamente.
- `TestAsk_EmptyQuestion`: pregunta vacia rechazada con `ErrEmptyQuestion`.
- `TestDispatchToolCall_UnknownTool`: tool desconocida reporta error.
- `TestDispatchToolCall_MalformedArguments`: JSON mal formado no causa panic.
- `TestDispatchToolCall_RejectsOutStage`: `$out` rechazado.
- `TestDispatchToolCall_RejectsMergeStage`: `$merge` rechazado.
- `TestDispatchToolCall_RejectsNestedOutStage`: `$out` anidado en `$facet`
  rechazado.
- `TestDispatchToolCall_ForcesMaxResultLimit`: limite forzado a
  `MONGODB_MAX_RESULT_LIMIT`.
- `TestAskQuestion_MissingAuthHeader`: 401 sin token.
- `TestAskQuestion_InvalidToken`: 401 con token incorrecto.
- `TestAskQuestion_MissingQuestionField`: 400 sin question.
- `TestAskQuestion_Success`: 200 con respuesta correcta.
- `TestAskQuestion_GeneratesSessionIDWhenMissing`: genera UUID si no se envia
  `session_id`.
- `TestAskQuestion_LLMErrorReturns502`: error del LLM mapeado a 502.

Los tests de integracion requieren conexiones reales a MongoDB Atlas, Redis
local y OpenCode Zen respectivamente, y se saltan con `t.Skip` si las
variables de entorno correspondientes no estan definidas.

## Creditos y alcance

Estructura de carpetas inspirada en
[`CrudGoExample`](https://github.com/HongXiangZuniga/CrudGoExample) (propio,
publico). Ningun codigo de este repositorio proviene de proyectos internos
o propietarios de terceros -- ver la nota de alcance/legal en `design.md`.

Proceso de desarrollo *spec-first* con
[OpenSpec](https://github.com/Fission-AI/OpenSpec): cada capability nueva o
endurecimiento de una ya existente se documenta como un "change" en
`openspec/changes/` antes de escribir código, con criterios de "listo"
verificables por comando (`grep`, `go test`, `go build`) en vez de
descripciones ambiguas. Los tres changes de este repositorio:

| Change | Que aporta |
|---|---|
| [`add-nl-mongo-agent`](./openspec/changes/add-nl-mongo-agent/) | El agente base: bucle de tool-calling, garantia de solo lectura en 4 capas, API REST autenticada |
| [`add-htmx-chat-frontend`](./openspec/changes/add-htmx-chat-frontend/) | Interfaz web de chat multi-sesion (pestanas), servida con htmx sin build de frontend |
| [`harden-agent-security`](./openspec/changes/harden-agent-security/) | Defensa contra prompt injection, deteccion de fuga del system prompt, redaccion de secretos en logs |
