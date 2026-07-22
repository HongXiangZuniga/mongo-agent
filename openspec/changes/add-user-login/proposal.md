## Why

Hoy `mongo-agent` no tiene un "sistema de usuarios". Toda la autenticación se
reduce a un único secreto compartido, `API_TOKEN`:

- `POST /ask` (`pkg/http/rest/router.go`, `TokenAuthMiddleware`) compara el header
  `Authorization` contra `API_TOKEN` con `subtle.ConstantTimeCompare`.
- La interfaz web (`pkg/http/web`) protege `/web` con una cookie cuyo valor es el
  propio `API_TOKEN` (`CookieAuthMiddleware`), y el formulario de login
  (`login.html` + `WebHandlers.Login`) pide un único campo `token` que se compara
  contra `API_TOKEN`.

No existe ningún concepto de usuario/contraseña. Para una POC de "inicio de
sesión" se necesita: (1) un login web con usuario y contraseña validado contra
una base de datos de usuarios, y (2) que ese login sea trivial de probar, con un
usuario precargado y las credenciales documentadas.

Adicionalmente, el propio binario Go **no está en `docker-compose.yml`** (el
compose actual solo levanta `redis`). Ya existe `build/docker/Dockerfile`
(multi-stage, usuario no-root), pero no hay un servicio que lo use. El objetivo
de negocio es "llegar y montar": que un solo `docker-compose up` levante Redis,
la nueva base de usuarios (ya seedeada) y la API Go, todo enlazado.

Restricción arquitectónica clave: la base de usuarios/login **debe ser una
instancia MongoDB local separada** del cluster MongoDB Atlas de solo lectura que
usa el agente (`MONGODB_URI`, protegido por `mongodb.VerifyReadOnlyGuarantee`).
Son dos bases con propósito y ciclo de vida distintos y no deben mezclarse ni
compartir conexión.

## What Changes

**[Login web con usuario y contraseña]**
- From: `POST /web/login` recibe un único campo `token` y lo compara contra `API_TOKEN`; `login.html` muestra un solo input "Token de acceso".
- To: `POST /web/login` recibe los campos `username` y `password`, y los valida contra un caso de uso de autenticación (`pkg/auth`) que consulta una base MongoDB de usuarios y verifica la contraseña con bcrypt. `login.html` muestra dos inputs (usuario y contraseña).
- Reason: Es el "sistema de usuarios" pedido para el inicio de sesión de la POC.
- Impact: Breaking para el flujo de login web (quien antes ingresaba el `API_TOKEN` ahora ingresa usuario/contraseña). No afecta a `POST /ask`.

**[La API REST `POST /ask` conserva la autenticación por `API_TOKEN`]**
- From: `POST /ask` se autentica con el header `Authorization` == `API_TOKEN`.
- To: Sin cambios. El login de usuario aplica solo a la superficie web (humana); la API REST sigue siendo máquina-a-máquina con token.
- Reason: Introducir usuario/contraseña en una API programática (curl/scripts) exigiría sesiones/JWT, fuera del alcance de la POC. Ver `design.md` (decisión D1) para la justificación de la convivencia.
- Impact: Non-breaking. Los clientes de `POST /ask` no cambian.

**[La cookie de sesión web transporta un session ID opaco respaldado en Redis]**
- From: La cookie web se fija con el valor `API_TOKEN` (un secreto de servidor único y estático), y `CookieAuthMiddleware` valida la cookie comparándola contra `API_TOKEN`. Cualquiera que ya tuviera esa cookie o conociera/adivinara `API_TOKEN` entraba a `/web` sin pasar por el login real de usuario/contraseña.
- To: Tras validar usuario/contraseña, `Login` genera un session ID aleatorio y opaco (distinto del session ID de conversación del agente) y guarda en Redis un registro con TTL que lo asocia al `username` autenticado. La cookie transporta ese session ID (nunca `API_TOKEN` ni la contraseña). `CookieAuthMiddleware` valida en cada request la cookie contra Redis (existe y no expiró) en lugar de compararla contra un valor estático, y `Logout` borra la key de Redis además de expirar la cookie.
- Reason: Corregir un bug de diseño real en producción: la autenticación web quedaba puenteada porque la cookie era un secreto compartido único, no una sesión por usuario. "Guardar la sesión en Redis y validarla" hace que el login de usuario/contraseña sea la única puerta de entrada real. Ver `design.md` (decisiones D2, D10, D11, D12).
- Impact: Breaking para el modelo de cookie interno (`CookieConfig` pierde `APIToken`; `CookieAuthMiddleware` y `RegisterRoutes` cambian de firma para recibir el gestor de sesiones). No afecta a `POST /ask` (sigue por `API_TOKEN`). La sesión web depende ahora de Redis, ya requerido por el agente.

**[Base MongoDB local dedicada para usuarios, separada del Atlas de solo lectura]**
- From: El sistema solo conoce una conexión MongoDB (`MONGODB_URI`, Atlas remoto de solo lectura del agente).
- To: Se añade una segunda conexión MongoDB independiente (`AUTH_MONGODB_URI`, instancia local) usada exclusivamente por el repositorio de usuarios del login. El adapter de usuarios vive en un paquete separado (`pkg/persistence/authmongo`) para no contaminar el invariante de solo lectura de `pkg/persistence/mongodb`.
- Reason: Requisito explícito de no mezclar la base de usuarios con el cluster de solo lectura del agente.
- Impact: Non-breaking para el agente; añade nuevas variables de entorno (`AUTH_MONGODB_URI`, `AUTH_MONGODB_DB_NAME`, `AUTH_MONGODB_USERS_COLLECTION`).

**[Contraseña hasheada en la base, nunca en texto plano]**
- From: No existe modelo de usuario persistido.
- To: El documento de usuario en MongoDB guarda `password_hash` (bcrypt), nunca la contraseña en claro. La contraseña en texto plano solo aparece en el seed documentado y en la documentación de prueba.
- Reason: Buena práctica mínima de seguridad sin complejidad relevante.
- Impact: Non-breaking. Define el esquema del documento de usuario.

**[Usuario de prueba precargado (seed) al levantar el compose]**
- From: No hay ningún usuario ni base de usuarios.
- To: Al levantar la base de usuarios vía `docker-compose up`, un script de inicialización (`/docker-entrypoint-initdb.d`) precarga un usuario de prueba con su `password_hash` bcrypt. Las credenciales en claro (`admin` / `admin123`) quedan documentadas en `README.md` y en `design.md` para probar el login sin pasos manuales.
- Reason: "Llegar y montar": poder loguearse inmediatamente tras `docker-compose up`, sin registro ni pasos extra.
- Impact: Non-breaking. Añade `build/mongo-auth/seed.js` y assets de compose.

**[La API Go y la base de usuarios se integran en docker-compose]**
- From: `docker-compose.yml` solo declara el servicio `redis`; el binario Go no está en el compose y no existe servicio de base de usuarios.
- To: `docker-compose.yml` añade (a) un servicio `api` que construye `build/docker/Dockerfile`, depende de `redis` y `mongo-auth`, y recibe sus env vars; (b) un servicio `mongo-auth` (imagen oficial `mongo`) con volumen de persistencia, healthcheck y seed montado. `docker-compose up` levanta los tres servicios enlazados.
- Reason: Objetivo de negocio "llegar y montar" con un solo comando.
- Impact: Non-breaking para el código; cambia la topología de despliegue local. La API sigue necesitando un `.env` con los secretos externos (Atlas `MONGODB_URI`, `OPENCODE_API_KEY`, `API_TOKEN`).

## Capabilities

### New Capabilities
- `user-authentication`: login web basado en usuario/contraseña validado contra una base MongoDB local dedicada, con contraseñas hasheadas (bcrypt) y un usuario de prueba precargado. Incluye la garantía de separación respecto al MongoDB Atlas de solo lectura del agente.

### Modified Capabilities
(ninguna modelada como MODIFIED. El login web ya descrito por `chat-web-frontend` en `add-htmx-chat-frontend` no está aún materializado en `openspec/specs/`, por lo que —siguiendo el mismo criterio que `harden-agent-security`— este cambio se modela como capability nueva y aditiva. Ver `design.md`, sección "Decisión sobre modelado de capability".)

## Impact

- Código nuevo:
  - `pkg/auth/` (dominio + aplicación): `auth.go` (entidad `User`, `ErrInvalidCredentials`), `authenticator.go` (puerto de entrada `Authenticator`), `user_port.go` (puerto de salida `UserRepository`, `ErrUserNotFound`), `password_port.go` (puerto de salida `PasswordHasher`), `service.go` (caso de uso).
  - `pkg/persistence/authmongo/user_repository.go` (adapter de salida: implementa `auth.UserRepository` sobre la base de usuarios).
  - `pkg/security/bcrypt.go` (adapter de salida: implementa `auth.PasswordHasher` con `golang.org/x/crypto/bcrypt`).
  - `build/mongo-auth/seed.js` (seed del usuario de prueba).
- Código modificado:
  - `pkg/http/web/handlers.go` (`Login` valida usuario/contraseña; `webPort` recibe `auth.Authenticator`; `NewWebHandler` amplía su firma), `pkg/http/web/templates/login.html` (dos campos).
  - `cmd/server/api.go` (lee las nuevas env vars, abre la conexión a la base de usuarios, construye el adapter de usuarios + hasher + servicio de auth, y los inyecta en el handler web).
  - `docker-compose.yml` (servicios `api` y `mongo-auth`, volumen y healthchecks), `.env.example` (nuevas variables), `README.md` y `Makefile` (documentación y comando de arranque).
- Dependencias: `golang.org/x/crypto/bcrypt` ya está disponible vía `golang.org/x/crypto v0.48.0` en `go.mod` (pasa de indirecta a directa con `go mod tidy`). No se añaden módulos nuevos.
- Sin impacto en el contrato de `POST /ask` ni en el dominio del agente (`pkg/agent`) ni en el adapter de solo lectura (`pkg/persistence/mongodb`).
