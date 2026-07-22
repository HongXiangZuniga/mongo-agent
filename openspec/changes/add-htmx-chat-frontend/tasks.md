## 1. Dominio y Ports (pkg/agent) — Backend

- [x] 1.1 En `pkg/agent/session_port.go`, añadir el tipo `type SessionSummary struct { SessionID string; Title string; LastActivity time.Time }` (importar `time` de librería estándar en ese archivo). Criterio de listo: struct exportado con exactamente esos tres campos, `gofmt -l pkg/agent/session_port.go` no reporta errores.
- [x] 1.2 En el mismo archivo, ampliar la interfaz `SessionStore` añadiendo exactamente estos dos métodos (sin eliminar los tres ya existentes):
  ```go
  ListSessions(ctx context.Context) ([]SessionSummary, error)
  TouchSession(ctx context.Context, sessionID string, title string, at time.Time) error
  ```
  Criterio de listo: la interfaz `SessionStore` tiene ahora exactamente 5 métodos: `AppendMessage`, `GetHistory`, `ClearSession`, `ListSessions`, `TouchSession`.
- [x] 1.3 Verificar que `pkg/agent/session_port.go` sigue sin importar `redis`, `gin` ni ningún cliente HTTP concreto. Criterio de listo: `grep -E '"(go.mongodb.org|redis|gin-gonic)"' pkg/agent/session_port.go` no devuelve resultados.

## 2. Application: AgentService (pkg/agent/service.go) — Backend

- [x] 2.1 En `pkg/agent/service.go`, ampliar la interfaz `AgentService` añadiendo exactamente estos tres métodos (sin eliminar `Ask`):
  ```go
  ListSessions(ctx context.Context) ([]SessionSummary, error)
  CreateSession(ctx context.Context) (SessionSummary, error)
  CloseSession(ctx context.Context, sessionID string) error
  GetConversation(ctx context.Context, sessionID string) ([]Message, error)
  ```
  Criterio de listo: la interfaz `AgentService` tiene ahora exactamente 5 métodos.
- [x] 2.2 Añadir el campo `sessionTitleMaxLength int` al struct `service` (en la posición final del struct) y el parámetro correspondiente `sessionTitleMaxLength int` como último parámetro del constructor `NewAgentService`, asignado por posición igual que los demás campos. Criterio de listo: `go build ./pkg/agent/...` falla en `cmd/server/api.go` hasta que la tarea 9.1 actualice la llamada al constructor (esto es esperado y temporal).
- [x] 2.3 En el mismo archivo, implementar la función privada `func deriveTitle(text string, maxLength int) string`: recorta espacios en blanco al inicio/fin de `text` con `strings.TrimSpace`; si el resultado (contado en runes, usar `[]rune`) tiene longitud menor o igual a `maxLength`, devolverlo tal cual; si es más largo, devolver los primeros `maxLength` runes seguidos del sufijo `"…"`. Si `text` recortado queda vacío, devolver el literal `"Nueva conversación"`. Importar `strings` en el archivo. Criterio de listo: cubierto por los tests `TestDeriveTitle_ShortTextUnchanged`, `TestDeriveTitle_LongTextTruncatedWithEllipsis`, `TestDeriveTitle_TrimsWhitespace`, `TestDeriveTitle_EmptyAfterTrimUsesDefault` (sección 10.1).
- [x] 2.4 Dentro de `func (s *service) Ask`, inmediatamente después de la línea que llama a `s.sessions.AppendMessage` para el mensaje del usuario (justo después del paso 4 del bucle descrito en `design.md` de `add-nl-mongo-agent`, antes de entrar al loop de tool-calling), añadir la llamada `if err := s.sessions.TouchSession(askCtx, q.SessionID, deriveTitle(q.Text, s.sessionTitleMaxLength), time.Now()); err != nil { return Answer{}, utils.ErrSessionStoreUnavailable(err) }`. Criterio de listo: cubierto por `TestAsk_TouchesSessionOnFirstMessage` y `TestAsk_TouchesSessionOnEveryMessage` (sección 10.1).
- [x] 2.5 Implementar `func (s *service) ListSessions(ctx context.Context) ([]SessionSummary, error)` como una delegación directa: `return s.sessions.ListSessions(ctx)` (sin lógica adicional, sin envolver el error — el adapter de salida ya devuelve errores utilizables). Criterio de listo: cubierto por `TestListSessions_DelegatesToSessionStore` (sección 10.1).
- [x] 2.6 Implementar `func (s *service) CreateSession(ctx context.Context) (SessionSummary, error)`: genera `sessionID := uuid.NewString()` (importar `github.com/google/uuid`), y devuelve `SessionSummary{SessionID: sessionID, Title: "Nueva conversación", LastActivity: time.Now()}, nil` **sin invocar ningún método de `s.sessions`**. Criterio de listo: cubierto por `TestCreateSession_DoesNotWriteToSessionStore` (sección 10.1), que falla si el fake de `SessionStore` registra alguna llamada durante `CreateSession`.
- [x] 2.7 Implementar `func (s *service) CloseSession(ctx context.Context, sessionID string) error` como delegación directa: `return s.sessions.ClearSession(ctx, sessionID)`. Criterio de listo: cubierto por `TestCloseSession_DelegatesToClearSession` (sección 10.1).
- [x] 2.8 Implementar `func (s *service) GetConversation(ctx context.Context, sessionID string) ([]Message, error)`: obtiene `history, err := s.sessions.GetHistory(ctx, sessionID)`; si `err != nil`, devuelve `nil, utils.ErrSessionStoreUnavailable(err)`; en caso contrario, construye un nuevo slice iterando `history` y añadiendo únicamente los mensajes cuyo `Role == RoleUser || Role == RoleAssistant`, preservando el orden original, y lo devuelve. Criterio de listo: cubierto por `TestGetConversation_FiltersToolAndSystemMessages` y `TestGetConversation_PreservesChronologicalOrder` (sección 10.1).
- [x] 2.9 Verificar que `pkg/agent/service.go` solo importa, además de lo ya existente, `strings` y `github.com/google/uuid` de terceros (más `pkg/utils`, `context`, `errors`, `fmt`, `time`). Criterio de listo: `grep -E '"(go.mongodb.org|redis|gin-gonic|net/http)"' pkg/agent/service.go` no devuelve resultados.

## 3. Utilidades compartidas (pkg/utils) — Backend

- [x] 3.1 Crear `pkg/utils/validation.go` con `package utils`. Definir `var sessionIDPattern = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)` (importar `regexp`) y `func IsValidSessionID(id string) bool { return id != "" && sessionIDPattern.MatchString(id) }`. Criterio de listo: cubierto por `TestIsValidSessionID_AcceptsAlphanumericWithDashesAndUnderscores` y `TestIsValidSessionID_RejectsEmptyOrSpecialCharacters` (sección 10.2).
- [x] 3.2 En `pkg/http/rest/agent.go`, eliminar la variable local `sessionIDRegex` y reemplazar la validación `!sessionIDRegex.MatchString(sessionID)` por `!utils.IsValidSessionID(sessionID)`, usando el paquete `utils` ya importado en ese archivo. Criterio de listo: `grep -n "sessionIDRegex" pkg/http/rest/agent.go` no devuelve resultados; los tests ya existentes en `test/http_rest_test.go` (`TestAskQuestion_*`) siguen pasando sin modificación de sus expectativas.
- [x] 3.3 En `pkg/utils/errors.go`, añadir la función:
  ```go
  func HTTPStatusForError(err error) (int, string) {
      switch {
      case errors.Is(err, ErrToolLoopExceeded()):
          return http.StatusBadGateway, "agent failed to finalize a response"
      case errors.Is(err, ErrRequestTimeout()):
          return http.StatusGatewayTimeout, "request timed out"
      case IsLLMProviderUnavailable(err):
          return http.StatusBadGateway, "llm provider unavailable"
      default:
          return http.StatusServiceUnavailable, "service temporarily unavailable"
      }
  }
  ```
  Importar `net/http` en `pkg/utils/errors.go`. Criterio de listo: cubierto por `TestHTTPStatusForError_MapsKnownErrors` y `TestHTTPStatusForError_DefaultsToServiceUnavailable` (sección 10.2).
- [x] 3.4 En `pkg/http/rest/agent.go`, refactorizar `func (p *agentPort) respondError` para que su cuerpo sea únicamente: obtener `status, message := utils.HTTPStatusForError(err)` y responder `ctx.JSON(status, Response{Code: status, Message: message})`, eliminando el `switch` duplicado. Criterio de listo: el comportamiento observable de `POST /ask` no cambia — todos los tests existentes en `test/http_rest_test.go` (`TestAskQuestion_*`) siguen pasando sin modificar sus aserciones de código de estado ni de mensaje.

## 4. Adapter de salida Redis (pkg/persistence/redis/session_store.go) — Backend

- [x] 4.1 En `pkg/persistence/redis/session_store.go`, añadir dos funciones privadas de construcción de claves: `func (s *store) metaKey(sessionID string) string { return fmt.Sprintf("session:%s:meta", sessionID) }` y una constante `const sessionsIndexKey = "sessions:index"` a nivel de paquete.
- [x] 4.2 Implementar `func (s *store) TouchSession(ctx context.Context, sessionID string, title string, at time.Time) error`: ejecutar en una única llamada `s.client.Pipelined(ctx, func(pipe goredis.Pipeliner) error { ... })` que dentro de la función encole, en este orden: `pipe.HSetNX(ctx, s.metaKey(sessionID), "title", title)`, `pipe.HSet(ctx, s.metaKey(sessionID), "last_activity", at.Unix())`, `pipe.Expire(ctx, s.metaKey(sessionID), s.ttl)`, `pipe.ZAdd(ctx, sessionsIndexKey, goredis.Z{Score: float64(at.Unix()), Member: sessionID})`; si el `Pipelined` devuelve error, envolverlo con `fmt.Errorf("failed to touch session: %w", err)`. Criterio de listo: cubierto por los tests de integración `TestSessionStore_TouchSession_SetsTitleOnFirstCall` y `TestSessionStore_TouchSession_PreservesTitleOnSubsequentCalls` (sección 10.3).
- [x] 4.3 Implementar `func (s *store) ListSessions(ctx context.Context) ([]agent.SessionSummary, error)`:
  1. `ids, err := s.client.ZRevRange(ctx, sessionsIndexKey, 0, -1).Result()`; si falla, devolver `fmt.Errorf("failed to list session ids: %w", err)`.
  2. Si `len(ids) == 0`, devolver `[]agent.SessionSummary{}, nil` inmediatamente.
  3. Ejecutar `cmds := make(map[string]*goredis.MapStringStringCmd, len(ids))` y una única llamada `s.client.Pipelined(ctx, func(pipe goredis.Pipeliner) error { for _, id := range ids { cmds[id] = pipe.HGetAll(ctx, s.metaKey(id)) }; return nil })`; si el `Pipelined` devuelve un error que no sea `goredis.Nil`, devolverlo envuelto.
  4. Recorrer `ids` en el mismo orden ya obtenido de `ZRevRange`; para cada `id`, leer `meta, _ := cmds[id].Result()`; si `len(meta) == 0` (clave expirada), acumular `id` en un slice `expired` y continuar sin añadirlo al resultado; si `len(meta) > 0`, parsear `last_activity` con `strconv.ParseInt(meta["last_activity"], 10, 64)` (si falla el parseo, usar `time.Time{}` sin fallar toda la operación) y construir `agent.SessionSummary{SessionID: id, Title: meta["title"], LastActivity: time.Unix(parsed, 0)}`, añadiéndolo al slice de resultado.
  5. Si `len(expired) > 0`, ejecutar una única llamada adicional `s.client.ZRem(ctx, sessionsIndexKey, toAnySlice(expired)...)` (definir el helper privado `func toAnySlice(ids []string) []any` en el mismo archivo) para limpiar el índice; ignorar (solo loguear con un comentario, no fallar la función) si esta llamada de limpieza falla, ya que no debe impedir devolver el resultado ya calculado.
  6. Devolver el slice de resultado construido en el paso 4.
  Criterio de listo: cubierto por `TestSessionStore_ListSessions_OrderedByMostRecentFirst` y `TestSessionStore_ListSessions_LazilyRemovesExpiredEntries` (sección 10.3).
- [x] 4.4 Modificar `func (s *store) ClearSession(ctx context.Context, sessionID string) error` para que, en lugar de un único `s.client.Del`, ejecute una única llamada `s.client.Pipelined(ctx, func(pipe goredis.Pipeliner) error { pipe.Del(ctx, s.sessionKey(sessionID)); pipe.Del(ctx, s.metaKey(sessionID)); pipe.ZRem(ctx, sessionsIndexKey, sessionID); return nil })`, envolviendo cualquier error con `fmt.Errorf("failed to clear session: %w", err)`. Criterio de listo: cubierto por `TestSessionStore_ClearSession_RemovesIndexEntry` (sección 10.3).
- [x] 4.5 Verificar que `pkg/persistence/redis/session_store.go` implementa la interfaz `agent.SessionStore` completa (5 métodos) — `go build ./pkg/persistence/redis/...` debe compilar sin error de interfaz incompleta.

## 5. Adapter de entrada REST — Wiring compartido (pkg/http/rest) — Backend

- [x] 5.1 En `pkg/http/rest/router.go`, extraer de `NewHandler` la lógica de registro de rutas a una nueva función exportada `func RegisterRoutes(r *gin.Engine, agentHandler AgentHandlers, apiToken string) { r.POST("/ask", TokenAuthMiddleware(apiToken), agentHandler.AskQuestion) }`. Modificar `NewHandler` para que quede: `func NewHandler(agentHandler AgentHandlers, apiToken string) *gin.Engine { r := gin.Default(); RegisterRoutes(r, agentHandler, apiToken); return r }`. Criterio de listo: `NewHandler` sigue teniendo la misma firma y el mismo comportamiento observable; todos los tests existentes que usan `rest.NewHandler` (`test/http_rest_test.go`) siguen pasando sin modificación.

## 6. Adapter de entrada Web — Assets y plantillas (pkg/http/web) — Frontend

- [x] 6.1 Crear el directorio `pkg/http/web/static/` y descargar el archivo `htmx.min.js` versión `1.9.12` desde `https://unpkg.com/htmx.org@1.9.12/dist/htmx.min.js`, guardándolo verbatim (sin modificar) como `pkg/http/web/static/htmx.min.js`. Si no hay acceso a red durante la implementación, usar como alternativa `https://raw.githubusercontent.com/bigskysoftware/htmx/v1.9.12/dist/htmx.min.js` (mismo contenido, misma versión pineada). Criterio de listo: el archivo existe, no está vacío, y contiene la cadena literal `htmx.org` en algún punto del contenido (`grep -c "htmx" pkg/http/web/static/htmx.min.js` devuelve un número mayor que 0).
- [x] 6.2 Crear el directorio `pkg/http/web/templates/` y el archivo `pkg/http/web/templates/layout.html` con `{{define "layout"}}` que renderiza un documento HTML5 completo: `<head>` con `<meta charset="utf-8">`, un `<title>` fijo (`"mongo-agent · chat"`), un `<script src="/web/static/htmx.min.js"></script>`, y opcionalmente una hoja de estilos mínima inline (`<style>` embebido en el mismo archivo, sin archivo CSS externo adicional — no se requiere un segundo asset estático). El `<body>` debe contener `<div id="tab-bar-container">{{template "tab_bar" .Tabs}}</div>` y `<div id="chat-panel-container">{{template "chat_panel" .ActiveTab}}</div>`. Criterio de listo: el archivo es HTML5 válido a simple inspección (doctype, head, body cerrados) y referencia exactamente los dos `{{template}}` indicados.
- [x] 6.3 Crear `pkg/http/web/templates/login.html` con `{{define "login"}}`: documento HTML5 completo (no depende de `layout`, es una página independiente) con un `<form method="POST" action="/web/login">` que contiene un `<input type="password" name="token" required>` y un botón de envío; si el dato pasado a la plantilla tiene un campo `Error` no vacío, mostrarlo en un `<p class="error">{{.Error}}</p>` justo encima del formulario. Criterio de listo: la plantilla compila con `template.Must(template.ParseFS(...))` (cubierto por la tarea 7.1) y referencia el campo `.Error` condicionalmente con `{{if .Error}}`.
- [x] 6.4 Crear `pkg/http/web/templates/tab_bar.html` con `{{define "tab_bar"}}`: el dato recibido es un slice de un tipo `tabViewModel` (a definir en la tarea 8.1) con campos `SessionID`, `Title`, `Active`. Renderizar un `<div id="tab-bar">` que itera con `{{range .}}` un botón por pestaña: `<button hx-get="/web/tabs/{{.SessionID}}" hx-target="#chat-panel-container" hx-swap="innerHTML" class="{{if .Active}}tab-active{{end}}">{{.Title}}</button>`, y al final del `<div>`, fuera del `{{range}}`, un botón fijo `<button hx-post="/web/tabs" hx-target="#chat-panel-container" hx-swap="innerHTML">+</button>`. El elemento raíz debe tener siempre `id="tab-bar"` (sin importar cuántas pestañas existan) para permitir que otras plantillas lo reemplacen vía `hx-swap-oob="true"` cuando se renderice como fragmento independiente (ver tarea 6.6). Criterio de listo: la plantilla itera correctamente sobre un slice vacío sin error (produce solo el botón "+").
- [x] 6.5 Crear `pkg/http/web/templates/chat_panel.html` con `{{define "chat_panel"}}`: el dato recibido es un tipo `chatPanelViewModel` (a definir en la tarea 8.1) con campos `SessionID`, `Messages []messageViewModel`, `Error string`. Renderizar un `<div id="chat-panel">` que: (a) si `.Error` no está vacío, muestra `<p class="error">{{.Error}}</p>`; (b) itera `{{range .Messages}}` mostrando cada mensaje en un `<div class="msg msg-{{.Role}}">{{.Content}}</div>` (usar `{{.Content}}`, NUNCA `{{.Content | ...}}` con funciones que desactiven el auto-escape de `html/template` — el contenido del mensaje es texto no confiable); (c) un formulario `<form hx-post="/web/tabs/{{.SessionID}}/messages" hx-target="#chat-panel-container" hx-swap="innerHTML">` con un `<input type="text" name="message" required>` y un botón de envío; (d) un botón de cierre `<button hx-delete="/web/tabs/{{.SessionID}}" hx-target="#chat-panel-container" hx-swap="innerHTML">Cerrar pestaña</button>`. Criterio de listo: el elemento raíz tiene siempre `id="chat-panel"`; con `Messages` vacío la plantilla renderiza el formulario sin errores.
- [x] 6.6 Crear `pkg/http/web/templates/tab_bar_oob.html` con `{{define "tab_bar_oob"}}` que envuelve exactamente el mismo contenido interno que `tab_bar.html` pero en un `<div id="tab-bar" hx-swap-oob="true">` en vez de `<div id="tab-bar">` (para poder incluirse como actualización fuera de banda dentro de la respuesta de `chat_panel` sin reemplazar el `hx-target` principal de la petición). Para evitar duplicar el marcado de las pestañas, esta plantilla debe reutilizar el bloque interno definiendo un sub-template compartido `{{define "tab_bar_items"}}` (que contiene el `{{range}}` de botones + el botón "+") en un archivo separado `pkg/http/web/templates/tab_bar_items.html`, e invocado tanto desde `tab_bar.html` (`<div id="tab-bar">{{template "tab_bar_items" .}}</div>`) como desde `tab_bar_oob.html` (`<div id="tab-bar" hx-swap-oob="true">{{template "tab_bar_items" .}}</div>`). Criterio de listo: `tab_bar.html` y `tab_bar_oob.html` no duplican el marcado de los botones de pestaña (ambos delegan en `tab_bar_items`).
- [x] 6.7 Crear `pkg/http/web/templates.go` con `package web`. Definir `//go:embed templates/*.html` sobre una variable `var templateFS embed.FS`, y `var tmpl = template.Must(template.ParseFS(templateFS, "templates/*.html"))` a nivel de paquete. Definir la función `func render(w io.Writer, name string, data any) error { return tmpl.ExecuteTemplate(w, name, data) }`. Importar `embed`, `html/template`, `io`. Criterio de listo: `go build ./pkg/http/web/...` compila (una vez existan también los tipos de la tarea 8.1) y `template.Must` no hace panic al parsear (es decir, todas las plantillas de 6.2–6.6 son sintácticamente válidas y sin nombres de `{{define}}` duplicados).
- [x] 6.8 Crear `pkg/http/web/static.go` con `package web`. Definir `//go:embed static/htmx.min.js` sobre `var staticFS embed.FS`, y `func serveHTMXAsset(c *gin.Context) { c.Header("Cache-Control", "public, max-age=86400"); c.FileFromFS("static/htmx.min.js", http.FS(staticFS)); }` (usar `http.FS` para adaptar `embed.FS` a `http.FileSystem`; `gin` infiere `Content-Type: application/javascript` a partir de la extensión `.js`). Criterio de listo: cubierto por `TestHTMXAsset_ServesJavascriptWithContentType` (sección 10.4).

## 7. Adapter de entrada Web — Router y middleware de autenticación (pkg/http/web) — Frontend

- [x] 7.1 Crear `pkg/http/web/config.go` con `package web`. Definir `type CookieConfig struct { CookieName string; MaxAge time.Duration; Secure bool; APIToken string }`. Importar `time`. Criterio de listo: struct exportado con exactamente esos 4 campos.
- [x] 7.2 Crear `pkg/http/web/middleware.go` con `package web`. Implementar `func CookieAuthMiddleware(cfg CookieConfig) gin.HandlerFunc` que: lee la cookie con `c.Cookie(cfg.CookieName)`; si hay error (cookie ausente) o el valor no coincide en tiempo constante (`crypto/subtle.ConstantTimeCompare`) con `cfg.APIToken`, ejecuta la lógica de fallo descrita en la tarea 7.3; si coincide, llama a `c.Next()`. Criterio de listo: cubierto por `TestCookieAuthMiddleware_RejectsMissingCookie` y `TestCookieAuthMiddleware_RejectsWrongToken` (sección 10.4).
- [x] 7.3 En el mismo archivo, implementar la lógica de fallo de autenticación como una función privada `func denyAuth(c *gin.Context)`: si `c.GetHeader("HX-Request") == "true"`, ejecutar `c.Header("HX-Redirect", "/web/login"); c.AbortWithStatus(http.StatusUnauthorized)`; en caso contrario, ejecutar `c.Redirect(http.StatusSeeOther, "/web/login"); c.Abort()`. Criterio de listo: cubierto por `TestCookieAuthMiddleware_HXRequestGetsHXRedirectHeader` y `TestCookieAuthMiddleware_NormalRequestGetsSeeOtherRedirect` (sección 10.4).
- [x] 7.4 Crear `pkg/http/web/router.go` con `package web`. Implementar:
  ```go
  func RegisterRoutes(r *gin.Engine, h WebHandlers, cfg CookieConfig) {
      r.GET("/web/static/htmx.min.js", serveHTMXAsset)
      r.GET("/web/login", h.LoginForm)
      r.POST("/web/login", h.Login)
      r.POST("/web/logout", h.Logout)

      authed := r.Group("/web")
      authed.Use(CookieAuthMiddleware(cfg))
      authed.GET("", h.Index)
      authed.POST("/tabs", h.NewTab)
      authed.GET("/tabs/:sessionId", h.SwitchTab)
      authed.POST("/tabs/:sessionId/messages", h.SendMessage)
      authed.DELETE("/tabs/:sessionId", h.CloseTab)
  }
  ```
  Criterio de listo: la función compila una vez exista la interfaz `WebHandlers` (tarea 8.2); las rutas registradas coinciden exactamente con las 8 listadas.
- [x] 7.5 Verificar que `pkg/http/web/router.go`, `pkg/http/web/middleware.go` y `pkg/http/web/config.go` no importan `pkg/persistence/mongodb`, `pkg/persistence/redis`, `pkg/llm/opencodezen` ni `pkg/http/rest`. Criterio de listo: `grep -E '"(pkg/persistence|pkg/llm|pkg/http/rest)"' pkg/http/web/router.go pkg/http/web/middleware.go pkg/http/web/config.go` no devuelve resultados.

## 8. Adapter de entrada Web — Handlers (pkg/http/web) — Frontend

- [x] 8.1 Crear `pkg/http/web/viewmodels.go` con `package web`. Definir los tipos privados de presentación:
  ```go
  type tabViewModel struct {
      SessionID string
      Title     string
      Active    bool
  }
  type messageViewModel struct {
      Role    string
      Content string
  }
  type chatPanelViewModel struct {
      SessionID string
      Messages  []messageViewModel
      Error     string
  }
  type pageViewModel struct {
      Tabs      []tabViewModel
      ActiveTab chatPanelViewModel
  }
  ```
  Definir también las funciones de mapeo `func toTabViewModels(summaries []agent.SessionSummary, activeSessionID string) []tabViewModel` (mapea cada `SessionSummary` marcando `Active: s.SessionID == activeSessionID`) y `func toMessageViewModels(messages []agent.Message) []messageViewModel` (mapea `Role: string(m.Role)`, `Content: m.Content`). Importar `agent "github.com/HongXiangZuniga/mongo-agent/pkg/agent"`. Criterio de listo: `go build ./pkg/http/web/...` compila.
- [x] 8.2 Crear `pkg/http/web/handlers.go` con `package web`. Definir la interfaz:
  ```go
  type WebHandlers interface {
      Index(*gin.Context)
      LoginForm(*gin.Context)
      Login(*gin.Context)
      Logout(*gin.Context)
      NewTab(*gin.Context)
      SwitchTab(*gin.Context)
      SendMessage(*gin.Context)
      CloseTab(*gin.Context)
  }
  ```
  y el struct `type webPort struct { agentService agent.AgentService; cookieCfg CookieConfig }` con constructor `func NewWebHandler(agentService agent.AgentService, cookieCfg CookieConfig) WebHandlers { return &webPort{agentService, cookieCfg} }`. Criterio de listo: struct implementa la interfaz (verificado por el compilador una vez existan los métodos de las tareas siguientes).
- [x] 8.3 Implementar `func (p *webPort) Index(c *gin.Context)`: llama `summaries, err := p.agentService.ListSessions(c.Request.Context())`; si `err != nil`, responder `503` con un cuerpo HTML mínimo de error (usar `render` con un template de error simple, o simplemente `c.String(http.StatusServiceUnavailable, "...")` — decisión: usar `c.String` para este caso extremo de fallo de infraestructura, ya que no hay página que renderizar) y retornar; si `summaries` está vacío, llamar a `p.agentService.CreateSession(c.Request.Context())` para obtener una `SessionSummary` efímera y usarla como única pestaña activa (sin más llamadas al `AgentService`); si `summaries` NO está vacío, usar `summaries[0]` (la más reciente, dado que `ListSessions` ya devuelve orden descendente) como sesión activa y llamar `p.agentService.GetConversation(c.Request.Context(), summaries[0].SessionID)` para obtener sus mensajes. Construir `pageViewModel{Tabs: toTabViewModels(summaries_o_lista_con_la_efimera, activeSessionID), ActiveTab: chatPanelViewModel{SessionID: activeSessionID, Messages: toMessageViewModels(messages)}}` (si la pestaña es efímera y nueva, `Messages` es un slice vacío, sin llamar a `GetConversation`) y renderizar con `c.Status(http.StatusOK); c.Header("Content-Type", "text/html; charset=utf-8"); render(c.Writer, "layout", vm)`. Criterio de listo: cubierto por `TestIndex_ShowsEphemeralTabWhenNoSessions` y `TestIndex_ShowsMostRecentSessionAsActive` (sección 10.4).
- [x] 8.4 Implementar `func (p *webPort) LoginForm(c *gin.Context)`: responde `200 OK`, `Content-Type: text/html; charset=utf-8`, renderizando el template `login` con un dato `struct{ Error string }{}` (sin error). Criterio de listo: cubierto por `TestLoginForm_RendersWithoutError` (sección 10.4).
- [x] 8.5 Implementar `func (p *webPort) Login(c *gin.Context)`: lee el campo de formulario `token := c.PostForm("token")`; compara en tiempo constante con `p.cookieCfg.APIToken` usando `crypto/subtle.ConstantTimeCompare`; si NO coincide, responder `401 Unauthorized` renderizando de nuevo `login` con `struct{ Error string }{Error: "Token inválido"}`; si coincide, fijar la cookie con `c.SetCookie(p.cookieCfg.CookieName, token, int(p.cookieCfg.MaxAge.Seconds()), "/web", "", p.cookieCfg.Secure, true)` (el último parámetro `true` es `httpOnly`) y responder `c.Redirect(http.StatusSeeOther, "/web")`. Criterio de listo: cubierto por `TestLogin_ValidTokenSetsCookieAndRedirects` y `TestLogin_InvalidTokenRendersErrorWithoutCookie` (sección 10.4).
- [x] 8.6 Implementar `func (p *webPort) Logout(c *gin.Context)`: `c.SetCookie(p.cookieCfg.CookieName, "", -1, "/web", "", p.cookieCfg.Secure, true)` seguido de `c.Redirect(http.StatusSeeOther, "/web/login")`. Criterio de listo: cubierto por `TestLogout_ClearsCookieAndRedirects` (sección 10.4).
- [x] 8.7 Implementar `func (p *webPort) NewTab(c *gin.Context)`: llama `newSession, err := p.agentService.CreateSession(c.Request.Context())`; si `err != nil`, renderizar `chat_panel` con `Error: "no se pudo crear la pestaña"` y responder `200` (ver D-W9/D-W7 en `design.md`: los fragmentos htmx siempre responden `200` con el error embebido en el HTML, nunca un código 5xx, para que htmx sí reemplace el contenido del target); llama también `existing, _ := p.agentService.ListSessions(c.Request.Context())` (ignorar error de esta segunda llamada — si falla, tratar como lista vacía) para construir la barra de pestañas completa: la pestaña nueva (`newSession`, marcada `Active: true`) antepuesta a las `existing` (marcadas `Active: false`); renderizar en una sola respuesta, en este orden exacto: primero el fragmento `tab_bar` (plantilla `tab_bar`, el que reemplaza directamente `#chat-panel-container`... **corrección de destino**: dado que el `hx-target` del botón "+" es `#chat-panel-container` (tarea 6.4), la respuesta de este handler debe escribir, concatenados en el mismo cuerpo HTTP: (a) el fragmento `chat_panel` para la pestaña nueva vacía, que es lo que reemplaza el contenido de `#chat-panel-container`; (b) inmediatamente después, el fragmento `tab_bar_oob` (con `hx-swap-oob="true"`) para actualizar `#tab-bar` fuera de banda con la lista completa de pestañas incluyendo la nueva activa. Escribir ambos fragmentos con dos llamadas sucesivas a `render(c.Writer, ...)` sobre el mismo `c.Writer`, tras fijar `c.Status(http.StatusOK)` y el header `Content-Type` una sola vez al inicio. Criterio de listo: cubierto por `TestNewTab_ReturnsChatPanelAndOOBTabBar` y `TestNewTab_DoesNotCallSessionStoreWrites` (sección 10.4).
- [x] 8.8 Implementar `func (p *webPort) SwitchTab(c *gin.Context)`: lee `sessionID := c.Param("sessionId")`; si `!utils.IsValidSessionID(sessionID)`, responder `400` con `c.String(http.StatusBadRequest, "invalid session id")`; en caso contrario, llama `messages, err := p.agentService.GetConversation(c.Request.Context(), sessionID)` (si `err != nil`, tratar `messages` como slice vacío y seguir — una sesión sin historial es un caso válido, no un error de UI); renderiza únicamente el fragmento `chat_panel` (sin `tab_bar_oob`, porque cambiar de pestaña no modifica el índice ni el orden — ver D-W5/decisión en `design.md`) con `chatPanelViewModel{SessionID: sessionID, Messages: toMessageViewModels(messages)}`, respondiendo `200`. Criterio de listo: cubierto por `TestSwitchTab_ReturnsChatPanelForRequestedSession` y `TestSwitchTab_InvalidSessionIDReturns400` (sección 10.4).
- [x] 8.9 Implementar `func (p *webPort) SendMessage(c *gin.Context)`: lee `sessionID := c.Param("sessionId")`; si `!utils.IsValidSessionID(sessionID)`, responder `400`; lee `message := strings.TrimSpace(c.PostForm("message"))`; si `message == ""`, renderizar únicamente el fragmento `chat_panel` actual (recuperado con `GetConversation`, sin invocar `Ask`) con `Error: "el mensaje no puede estar vacío"` y responder `200`; en caso contrario, invocar `_, err := p.agentService.Ask(c.Request.Context(), agent.Question{SessionID: sessionID, Text: message})`; si `err != nil`, loguear con `log.Printf` el error real (nunca exponerlo al cliente) y renderizar `chat_panel` recuperando el historial actual vía `GetConversation` (que ya incluirá el mensaje del usuario, persistido antes de que `Ask` fallara) más `Error: "no se pudo procesar el mensaje"`, respondiendo `200`; si `err == nil`, recuperar `messages, _ := p.agentService.GetConversation(c.Request.Context(), sessionID)` y `sessions, _ := p.agentService.ListSessions(c.Request.Context())`, y responder con dos fragmentos concatenados igual que en la tarea 8.7: (a) `chat_panel` con los mensajes actualizados; (b) `tab_bar_oob` con la barra de pestañas actualizada (marcando `sessionID` como activa). Criterio de listo: cubierto por `TestSendMessage_Success_ReturnsUpdatedPanelAndOOBTabBar`, `TestSendMessage_EmptyMessage_DoesNotCallAsk`, y `TestSendMessage_AgentError_RendersGenericErrorInline` (sección 10.4).
- [x] 8.10 Implementar `func (p *webPort) CloseTab(c *gin.Context)`: lee `sessionID := c.Param("sessionId")`; si `!utils.IsValidSessionID(sessionID)`, responder `400`; invoca `_ = p.agentService.CloseSession(c.Request.Context(), sessionID)` (ignorar el error específico de "la sesión no existía" — cerrar una sesión ya inexistente no es un fallo desde la perspectiva de la UI); llama `remaining, _ := p.agentService.ListSessions(c.Request.Context())`; si `len(remaining) > 0`, la nueva sesión activa es `remaining[0]` y se recuperan sus mensajes con `GetConversation`; si `len(remaining) == 0`, llama `p.agentService.CreateSession(c.Request.Context())` para obtener una pestaña efímera nueva con `Messages` vacío; renderiza los mismos dos fragmentos concatenados que en 8.7/8.9 (`chat_panel` de la nueva pestaña activa + `tab_bar_oob` con la lista resultante). Criterio de listo: cubierto por `TestCloseTab_ActivatesNextMostRecentSession` y `TestCloseTab_CreatesEphemeralTabWhenNoneRemain` (sección 10.4).
- [x] 8.11 Verificar que ningún handler de `pkg/http/web/handlers.go` incluye `err.Error()` crudo en un `render`/`c.String` dirigido al cliente cuando el error proviene de `AgentService` — todos los mensajes de error mostrados en HTML deben ser literales fijos en español, y el error real solo se registra vía `log.Printf`. Criterio de listo: `grep -n "err.Error()" pkg/http/web/handlers.go` solo aparece dentro de llamadas a `log.Printf`, nunca dentro de `render(...)` ni `c.String(...)`.

## 9. Infraestructura / Wiring (cmd/server/api.go) — Backend + Frontend

- [x] 9.1 En `cmd/server/api.go`, añadir las nuevas variables package-level: `webAuthCookieName string`, `webCookieSecure bool`, `webSessionMaxAge time.Duration`, `webSessionTitleMaxLength int`. En `func init()`, leerlas con: `webAuthCookieName = getenv("WEB_AUTH_COOKIE_NAME", "web_auth")`, `webCookieSecure = getenv("WEB_COOKIE_SECURE", "false") == "true"`, `webSessionMaxAge = parseDurationSeconds(getenv("WEB_SESSION_MAX_AGE_SECONDS", "604800"))`, `webSessionTitleMaxLength = parseInt(getenv("WEB_SESSION_TITLE_MAX_LENGTH", "40"))`. No añadir estas variables a `requireEnv` (todas tienen default válido, ninguna es secreta). Criterio de listo: `go build ./cmd/server/...` compila tras completar las tareas siguientes.
- [x] 9.2 En `func main()`, actualizar la llamada a `agent.NewAgentService(...)` para incluir el nuevo último parámetro `webSessionTitleMaxLength` (ver tarea 2.2). Criterio de listo: la llamada tiene ahora 9 argumentos posicionales, en el mismo orden que la firma actualizada del constructor.
- [x] 9.3 En `func main()`, sustituir la construcción actual (`agentHandler := rest.NewAgentHandler(agentService); r := rest.NewHandler(agentHandler, apiToken)`) por:
  ```go
  agentHandler := rest.NewAgentHandler(agentService)
  webHandler := web.NewWebHandler(agentService, web.CookieConfig{
      CookieName: webAuthCookieName,
      MaxAge:     webSessionMaxAge,
      Secure:     webCookieSecure,
      APIToken:   apiToken,
  })

  r := gin.Default()
  rest.RegisterRoutes(r, agentHandler, apiToken)
  web.RegisterRoutes(r, webHandler, web.CookieConfig{
      CookieName: webAuthCookieName,
      MaxAge:     webSessionMaxAge,
      Secure:     webCookieSecure,
      APIToken:   apiToken,
  })
  ```
  Añadir el import `"github.com/HongXiangZuniga/mongo-agent/pkg/http/web"` y `"github.com/gin-gonic/gin"` a `cmd/server/api.go`. Criterio de listo: `go build ./cmd/server/...` compila; `r.Run(":" + port)` (línea ya existente, sin cambios) sigue siendo la única llamada que arranca el servidor HTTP.
- [x] 9.4 Verificar que `cmd/server/api.go` sigue siendo el único archivo del proyecto que importa simultáneamente `pkg/agent`, `pkg/persistence/mongodb`, `pkg/persistence/redis`, `pkg/llm/opencodezen`, `pkg/http/rest` y (ahora también) `pkg/http/web`. Criterio de listo: `grep -rl "pkg/http/web\"" --include=*.go . | grep -v _test.go` devuelve únicamente `cmd/server/api.go` y los archivos dentro de `pkg/http/web/*`.

## 10. Tests

- [x] 10.1 Ampliar `test/agent_service_test.go`: extender el `fakeSessionStore` ya existente con implementaciones en memoria de `ListSessions` (devuelve las entradas conocidas ordenadas por `LastActivity` descendente) y `TouchSession` (registra/actualiza título y actividad en el mapa interno, preservando el título si ya existía) y con un contador de invocaciones por método (para poder aserir "cero llamadas" en `TestCreateSession_DoesNotWriteToSessionStore`). Añadir los tests: `TestDeriveTitle_ShortTextUnchanged`, `TestDeriveTitle_LongTextTruncatedWithEllipsis`, `TestDeriveTitle_TrimsWhitespace`, `TestDeriveTitle_EmptyAfterTrimUsesDefault`, `TestAsk_TouchesSessionOnFirstMessage`, `TestAsk_TouchesSessionOnEveryMessage`, `TestListSessions_DelegatesToSessionStore`, `TestCreateSession_DoesNotWriteToSessionStore`, `TestCloseSession_DelegatesToClearSession`, `TestGetConversation_FiltersToolAndSystemMessages`, `TestGetConversation_PreservesChronologicalOrder`. Criterio de listo: `go test ./test/... -run 'TestDeriveTitle|TestAsk_Touches|TestListSessions|TestCreateSession|TestCloseSession|TestGetConversation'` pasa en verde.
- [x] 10.2 Crear `test/utils_test.go`, `package test`. Escribir `TestIsValidSessionID_AcceptsAlphanumericWithDashesAndUnderscores`, `TestIsValidSessionID_RejectsEmptyOrSpecialCharacters`, `TestHTTPStatusForError_MapsKnownErrors` (verifica los 3 mapeos explícitos de la tarea 3.3), `TestHTTPStatusForError_DefaultsToServiceUnavailable`. Criterio de listo: `go test ./test/... -run 'TestIsValidSessionID|TestHTTPStatusForError'` pasa en verde.
- [x] 10.3 Ampliar `test/integration/redis_session_test.go` (mismo build tag `integration`, mismo patrón `t.Skip` si `REDIS_ADDR` no está definida). Añadir: `TestSessionStore_TouchSession_SetsTitleOnFirstCall`, `TestSessionStore_TouchSession_PreservesTitleOnSubsequentCalls` (llama `TouchSession` dos veces con títulos distintos y verifica que `ListSessions` devuelve el primero), `TestSessionStore_ListSessions_OrderedByMostRecentFirst` (crea 3 sesiones con `TouchSession` en momentos distintos y verifica el orden devuelto), `TestSessionStore_ListSessions_LazilyRemovesExpiredEntries` (usa un `store` construido con un `ttl` muy corto, ej. 1 segundo, espera a que expire con `time.Sleep`, y verifica que una llamada posterior a `ListSessions` ya no incluye esa sesión), `TestSessionStore_ClearSession_RemovesIndexEntry` (verifica que tras `ClearSession`, `ListSessions` ya no incluye esa sesión). Criterio de listo: `go test -tags=integration ./test/integration/... -run TestSessionStore` pasa en verde con `REDIS_ADDR` configurada, y se salta sin ella.
- [x] 10.4 Crear `test/http_web_test.go`, `package test`. Definir un `fakeAgentService` que implemente el `agent.AgentService` ampliado (5 métodos), con contadores de invocaciones por método y datos configurables. Usando `net/http/httptest` y `web.NewWebHandler`/`web.RegisterRoutes` sobre un `gin.Engine` de test, escribir: `TestCookieAuthMiddleware_RejectsMissingCookie`, `TestCookieAuthMiddleware_RejectsWrongToken`, `TestCookieAuthMiddleware_HXRequestGetsHXRedirectHeader`, `TestCookieAuthMiddleware_NormalRequestGetsSeeOtherRedirect`, `TestIndex_ShowsEphemeralTabWhenNoSessions`, `TestIndex_ShowsMostRecentSessionAsActive`, `TestLoginForm_RendersWithoutError`, `TestLogin_ValidTokenSetsCookieAndRedirects`, `TestLogin_InvalidTokenRendersErrorWithoutCookie`, `TestLogout_ClearsCookieAndRedirects`, `TestNewTab_ReturnsChatPanelAndOOBTabBar`, `TestNewTab_DoesNotCallSessionStoreWrites`, `TestSwitchTab_ReturnsChatPanelForRequestedSession`, `TestSwitchTab_InvalidSessionIDReturns400`, `TestSendMessage_Success_ReturnsUpdatedPanelAndOOBTabBar`, `TestSendMessage_EmptyMessage_DoesNotCallAsk`, `TestSendMessage_AgentError_RendersGenericErrorInline`, `TestCloseTab_ActivatesNextMostRecentSession`, `TestCloseTab_CreatesEphemeralTabWhenNoneRemain`, `TestHTMXAsset_ServesJavascriptWithContentType`. Criterio de listo: `go test ./test/... -run 'TestCookieAuthMiddleware|TestIndex|TestLoginForm|TestLogin_|TestLogout|TestNewTab|TestSwitchTab|TestSendMessage|TestCloseTab|TestHTMXAsset'` pasa en verde.
- [x] 10.5 Ejecutar `go test ./test/...` completo (sin el build tag `integration`) y confirmar que TODOS los tests existentes de `add-nl-mongo-agent` (`TestAsk_*`, `TestDispatchToolCall_*`, `TestAskQuestion_*`) siguen pasando sin ninguna modificación de sus aserciones — esto es el criterio de no-regresión explícito de `design.md`.

## 11. Documentación

- [x] 11.1 Añadir a `.env.example`, en una nueva sección `# --- Interfaz web de chat (htmx) ---`, las variables: `WEB_AUTH_COOKIE_NAME=web_auth`, `WEB_COOKIE_SECURE=false`, `WEB_SESSION_MAX_AGE_SECONDS=604800`, `WEB_SESSION_TITLE_MAX_LENGTH=40`, cada una con un comentario de una línea explicando su propósito (ninguna es una credencial real). Criterio de listo: las 4 variables aparecen en `.env.example` con esos nombres exactos.
- [x] 11.2 Actualizar `README.md`: añadir `pkg/http/web/` a la lista de la sección "Arquitectura (resumen)" (adapter de entrada: interfaz de chat HTML+htmx), añadir una sección nueva "Interfaz web de chat" describiendo brevemente `GET /web`, el login por cookie, y que la implementación completa vive en `openspec/changes/add-htmx-chat-frontend/`. Referenciar `design.md` y `specs/chat-web-frontend/spec.md` de ese change. Criterio de listo: el README menciona explícitamente la ruta `/web/login` y el hecho de que la cookie reutiliza `API_TOKEN`.
- [x] 11.3 Verificar que `.gitignore` no excluye `pkg/http/web/static/htmx.min.js` (el asset vendored SÍ debe commitearse, a diferencia de `.env`). Criterio de listo: `git check-ignore pkg/http/web/static/htmx.min.js` no devuelve la ruta (es decir, no está ignorada).

## 12. Validación final

- [x] 12.1 Ejecutar `gofmt -l .` desde la raíz del proyecto y corregir cualquier archivo listado (debe devolver una lista vacía).
- [x] 12.2 Ejecutar `go vet ./...` y corregir cualquier advertencia.
- [x] 12.3 Ejecutar `go build ./...` y confirmar que compila sin errores.
- [x] 12.4 Ejecutar `make unit-test` (equivalente a `go test ./test/...` excluyendo el paquete `integration`) y confirmar que todos los tests pasan, incluidos los de no-regresión de la tarea 10.5.
- [x] 12.5 Si hay acceso a un Redis local (`docker-compose up -d`), ejecutar `make integration-test` con `REDIS_ADDR` configurada y confirmar que los tests de la tarea 10.3 pasan.
- [x] 12.6 Ejecutar `npx -y @fission-ai/openspec@latest validate add-htmx-chat-frontend --strict` desde la raíz del repo y corregir cualquier error de formato reportado antes de considerar el cambio listo para `openspec archive add-htmx-chat-frontend`.

## 13. UI/UX — Sistema visual (pkg/http/web) — Frontend

Estas tareas aplican las decisiones visuales formalizadas en `openspec/changes/add-htmx-chat-frontend/design-ui.md` (paleta, tipografía, espaciado, estados de mensaje/pestaña, accesibilidad y micro-animación de swap) sobre el código ya implementado en las secciones 6-9. No renumeran ni modifican ninguna tarea ya marcada `[x]`; son tareas nuevas que **modifican archivos ya creados** por esas tareas.

- [x] 13.1 Crear el directorio `pkg/http/web/static/fonts/` y vendorizar los dos pesos de la tipografía `JetBrains Mono` (subset `latin`, paquete `@fontsource/jetbrains-mono` versión pineada `5.0.20`, ver `design-ui.md` §3.3), descargando verbatim (sin modificar):
  - `https://cdn.jsdelivr.net/npm/@fontsource/jetbrains-mono@5.0.20/files/jetbrains-mono-latin-400-normal.woff2` → guardar como `pkg/http/web/static/fonts/jetbrains-mono-400.woff2`
  - `https://cdn.jsdelivr.net/npm/@fontsource/jetbrains-mono@5.0.20/files/jetbrains-mono-latin-600-normal.woff2` → guardar como `pkg/http/web/static/fonts/jetbrains-mono-600.woff2`
  Si no hay acceso a red durante la implementación, cualquier build estático equivalente del mismo paquete/versión sirve (debe ser el mismo peso y subset). Criterio de listo: ambos archivos existen, no están vacíos, y `file pkg/http/web/static/fonts/jetbrains-mono-400.woff2` (y el equivalente para `600`) reporta `Web Open Font Format (Version 2)`; el tamaño de cada archivo está entre 15 KB y 30 KB.
- [x] 13.2 Crear `pkg/http/web/static/app.css` con el siguiente contenido exacto (fuente de verdad: `design-ui.md` §3-§7; este bloque ya integra esas decisiones sobre la estructura de selectores ya usada por las plantillas existentes — copiarlo verbatim, sin modificar valores):
  ```css
  :root {
    /* Colores: fondo / superficies */
    --color-bg:            #0F172A;
    --color-surface:       #1A2436;
    --color-surface-2:     #1E293B;
    --color-surface-user:  #16321F;

    /* Colores: bordes */
    --color-border:        #5B6B85;
    --color-border-subtle: #334155;

    /* Colores: texto */
    --color-text:           #F1F5F9;
    --color-text-secondary: #94A3B8;
    --color-text-muted:     #64748B;

    /* Colores: acento */
    --color-accent:         #22C55E;
    --color-accent-strong:  #16A34A;
    --color-on-accent:      #0B1220;

    /* Colores: estado de error */
    --color-danger:         #F87171;
    --color-danger-bg:      #3A1518;
    --color-danger-border:  #7F2A2E;

    /* Tipografia */
    --font-mono: "JetBrains Mono", ui-monospace, "SF Mono", "Cascadia Code", "Roboto Mono", monospace;
    --font-size-sm:   0.8125rem;
    --font-size-base: 1rem;
    --font-size-lg:   1.25rem;
    --line-height-tight: 1.3;
    --line-height-body:  1.5;

    /* Espaciado */
    --space-1: 0.25rem;
    --space-2: 0.5rem;
    --space-3: 0.75rem;
    --space-4: 1rem;
    --space-5: 1.5rem;
    --space-6: 2rem;
  }

  @font-face {
    font-family: "JetBrains Mono";
    src: url("/web/static/fonts/jetbrains-mono-400.woff2") format("woff2");
    font-weight: 400;
    font-style: normal;
    font-display: swap;
  }
  @font-face {
    font-family: "JetBrains Mono";
    src: url("/web/static/fonts/jetbrains-mono-600.woff2") format("woff2");
    font-weight: 600;
    font-style: normal;
    font-display: swap;
  }

  * { box-sizing: border-box; }

  body {
    font-family: var(--font-mono);
    font-size: var(--font-size-base);
    line-height: var(--line-height-body);
    margin: 0;
    height: 100vh;
    display: flex;
    flex-direction: column;
    background: var(--color-bg);
    color: var(--color-text);
  }

  body.login-page {
    align-items: center;
    justify-content: center;
  }

  :focus-visible {
    outline: 2px solid var(--color-accent);
    outline-offset: 2px;
    border-radius: 0.25rem;
  }

  #tab-bar {
    display: flex;
    gap: var(--space-1);
    padding: var(--space-2) var(--space-3) 0;
    background: var(--color-surface);
    border-bottom: 1px solid var(--color-border-subtle);
    overflow-x: auto;
  }
  #tab-bar button {
    border: 1px solid transparent;
    border-bottom: none;
    background: transparent;
    color: var(--color-text-secondary);
    font-family: var(--font-mono);
    font-size: var(--font-size-sm);
    line-height: var(--line-height-tight);
    font-weight: 400;
    padding: 0.625rem var(--space-4);
    cursor: pointer;
    white-space: nowrap;
    border-radius: 0.375rem 0.375rem 0 0;
    transition: background-color 150ms ease, color 150ms ease;
  }
  #tab-bar button:hover {
    background: var(--color-surface-2);
    color: var(--color-text);
  }
  #tab-bar button.tab-active {
    background: var(--color-bg);
    color: var(--color-text);
    font-weight: 600;
    border-top: 2px solid var(--color-accent);
  }
  #tab-bar button.btn-secondary {
    color: var(--color-accent);
  }

  #chat-panel {
    flex: 1;
    display: flex;
    flex-direction: column;
    padding: var(--space-4);
    overflow-y: auto;
    background: var(--color-bg);
  }
  .msg {
    max-width: 80%;
    padding: var(--space-3) var(--space-4);
    margin-bottom: var(--space-3);
    border-radius: 0.75rem;
    line-height: var(--line-height-body);
    white-space: pre-wrap;
    word-break: break-word;
  }
  .msg-user {
    align-self: flex-end;
    background: var(--color-surface-user);
    border: 1px solid rgba(34, 197, 94, 0.25);
    color: var(--color-text);
  }
  .msg-assistant {
    align-self: flex-start;
    background: var(--color-surface-2);
    border: 1px solid var(--color-border-subtle);
    color: var(--color-text);
  }
  .msg-user::before,
  .msg-assistant::before {
    display: block;
    font-size: var(--font-size-sm);
    color: var(--color-text-secondary);
    margin-bottom: var(--space-1);
    letter-spacing: 0.02em;
    text-transform: uppercase;
  }
  .msg-user::before { content: "Tú"; text-align: right; }
  .msg-assistant::before { content: "Agente"; }

  form {
    display: flex;
    gap: var(--space-2);
    margin-top: auto;
    padding-top: var(--space-4);
  }
  input[type="text"],
  input[type="password"] {
    flex: 1;
    padding: 0.625rem var(--space-3);
    background: var(--color-surface);
    border: 1px solid var(--color-border);
    border-radius: 0.375rem;
    color: var(--color-text);
    font-family: var(--font-mono);
    font-size: var(--font-size-base);
  }
  input[type="text"]::placeholder,
  input[type="password"]::placeholder {
    color: var(--color-text-muted);
  }

  button {
    padding: 0.625rem var(--space-4);
    border: 1px solid transparent;
    border-radius: 0.375rem;
    cursor: pointer;
    font-family: var(--font-mono);
    font-size: var(--font-size-sm);
    line-height: var(--line-height-tight);
    background: transparent;
    color: var(--color-text);
  }
  .btn-primary {
    background: var(--color-accent);
    color: var(--color-on-accent);
    border-color: var(--color-accent);
    font-weight: 600;
  }
  .btn-primary:hover {
    background: var(--color-accent-strong);
    border-color: var(--color-accent-strong);
  }
  .btn-secondary {
    background: transparent;
    color: var(--color-text-secondary);
    border-color: var(--color-border);
  }
  .btn-secondary:hover {
    color: var(--color-text);
    border-color: var(--color-accent);
  }

  .login-box {
    background: var(--color-surface);
    border: 1px solid var(--color-border-subtle);
    border-radius: 0.5rem;
    padding: var(--space-6);
    width: 100%;
    max-width: 24rem;
    box-shadow: 0 8px 24px rgba(0, 0, 0, 0.35);
  }
  .login-box h1 {
    margin: 0 0 var(--space-4);
    font-size: var(--font-size-lg);
    font-weight: 600;
    color: var(--color-text);
    font-family: var(--font-mono);
  }
  .login-box label {
    display: block;
    margin-bottom: var(--space-2);
    font-weight: 600;
    color: var(--color-text-secondary);
    font-size: var(--font-size-sm);
  }
  .login-box .btn-primary {
    width: 100%;
  }

  .error {
    color: var(--color-danger);
    background: var(--color-danger-bg);
    border: 1px solid var(--color-danger-border);
    border-radius: 0.375rem;
    padding: var(--space-3) var(--space-4);
    margin-bottom: var(--space-4);
  }

  .htmx-swapping {
    opacity: 0;
    transition: opacity 120ms ease-out;
  }
  @media (prefers-reduced-motion: reduce) {
    .htmx-swapping { transition: none; }
  }
  ```
  Criterio de listo: el archivo existe, contiene las cadenas literales `--color-accent:         #22C55E;`, `.btn-primary`, `.msg-user::before`, `:focus-visible` y `.htmx-swapping`; no contiene la cadena `#f5f6f7` ni `#1a1a1a` (paleta gris anterior ya reemplazada).
- [x] 13.3 Modificar `pkg/http/web/static.go`: ampliar la directiva de embed existente a `//go:embed static/htmx.min.js static/app.css static/fonts/*.woff2` sobre la misma variable `staticFS`. Añadir dos funciones nuevas en el mismo archivo:
  ```go
  func serveAppCSS(c *gin.Context) {
      c.Header("Cache-Control", "public, max-age=31536000, immutable")
      c.Header("Content-Type", "text/css; charset=utf-8")
      c.FileFromFS("static/app.css", http.FS(staticFS))
  }

  func serveFont(c *gin.Context) {
      filename := c.Param("filename")
      if filename != "jetbrains-mono-400.woff2" && filename != "jetbrains-mono-600.woff2" {
          c.AbortWithStatus(http.StatusNotFound)
          return
      }
      c.Header("Cache-Control", "public, max-age=31536000, immutable")
      c.Header("Content-Type", "font/woff2")
      c.FileFromFS("static/fonts/"+filename, http.FS(staticFS))
  }
  ```
  La función `serveFont` valida `filename` contra una lista cerrada (evita servir rutas arbitrarias fuera de los dos archivos vendorizados). Criterio de listo: `go build ./pkg/http/web/...` compila; cubierto por `TestAppCSSAsset_ServesCSSWithContentType`, `TestFontAsset_ServesWoff2WithContentType`, `TestFontAsset_RejectsUnknownFilename` (sección 13.10).
- [x] 13.4 Modificar `pkg/http/web/router.go`: en `RegisterRoutes`, junto a la línea ya existente `r.GET("/web/static/htmx.min.js", serveHTMXAsset)`, añadir (mismo nivel, fuera del grupo `authed`, sin `CookieAuthMiddleware`):
  ```go
  r.GET("/web/static/app.css", serveAppCSS)
  r.GET("/web/static/fonts/:filename", serveFont)
  ```
  Criterio de listo: las 3 rutas de assets estáticos (`htmx.min.js`, `app.css`, `fonts/:filename`) están registradas antes del grupo `authed` en el mismo archivo; `go build ./pkg/http/web/...` compila.
- [x] 13.5 Modificar `pkg/http/web/templates/layout.html`: eliminar por completo el bloque `<style>...</style>` inline (líneas actuales dentro de `<head>`), y en su lugar añadir `<link rel="stylesheet" href="/web/static/app.css">` inmediatamente antes de `<script src="/web/static/htmx.min.js"></script>`. No modificar el resto del archivo (`{{template "tab_bar" .Tabs}}`, `{{template "chat_panel" .ActiveTab}}` permanecen igual). Criterio de listo: `grep -c "<style>" pkg/http/web/templates/layout.html` devuelve `0`; `grep -c 'href="/web/static/app.css"' pkg/http/web/templates/layout.html` devuelve `1`.
- [x] 13.6 Modificar `pkg/http/web/templates/login.html`: eliminar por completo el bloque `<style>...</style>` inline, añadir `<link rel="stylesheet" href="/web/static/app.css">` inmediatamente antes de `<script src="/web/static/htmx.min.js"></script>`, cambiar la etiqueta `<body>` a `<body class="login-page">` (necesario para que la regla `body.login-page` de `app.css` centre `.login-box`, ver tarea 13.2), y añadir `class="btn-primary"` al `<button type="submit">Ingresar</button>` existente. No modificar el resto del formulario. Criterio de listo: `grep -c "<style>" pkg/http/web/templates/login.html` devuelve `0`; `grep -c 'class="login-page"' pkg/http/web/templates/login.html` devuelve `1`; `grep -c 'class="btn-primary"' pkg/http/web/templates/login.html` devuelve `1`.
- [x] 13.7 Modificar `pkg/http/web/templates/chat_panel.html`: añadir `class="btn-primary"` al `<button type="submit">Enviar</button>` existente; añadir `class="btn-secondary"` al `<button hx-delete=...>Cerrar pestaña</button>` existente; añadir `aria-label="Mensaje"` al `<input type="text" name="message" required>` existente (el input no tiene `<label>` visible porque compartiría la línea con el botón de envío; `aria-label` cubre el requisito de accesibilidad sin romper el layout de una sola línea, ver `design-ui.md` §5). No modificar los atributos `hx-post`/`hx-target`/`hx-swap` en esta tarea (ver tarea 13.9 para el modificador de `hx-swap`). Criterio de listo: `grep -c 'class="btn-primary"' pkg/http/web/templates/chat_panel.html` devuelve `1`; `grep -c 'class="btn-secondary"' pkg/http/web/templates/chat_panel.html` devuelve `1`; `grep -c 'aria-label="Mensaje"' pkg/http/web/templates/chat_panel.html` devuelve `1`.
- [x] 13.8 Modificar `pkg/http/web/templates/tab_bar_items.html`: añadir `class="btn-secondary"` al botón fijo `<button hx-post="/web/tabs" ...>+</button>` existente (los botones de pestaña ya tienen su propio manejo de clase condicional `{{if .Active}}tab-active{{end}}` y no se tocan en esta tarea). Criterio de listo: `grep -c 'class="btn-secondary"' pkg/http/web/templates/tab_bar_items.html` devuelve `1`.
- [x] 13.9 En `pkg/http/web/templates/tab_bar_items.html` y `pkg/http/web/templates/chat_panel.html`, cambiar cada atributo `hx-swap="innerHTML"` existente a `hx-swap="innerHTML swap:120ms"` (4 ocurrencias en total: cambiar de pestaña y crear pestaña en `tab_bar_items.html`; enviar mensaje y cerrar pestaña en `chat_panel.html`). Este modificador es requerido por htmx para que añada la clase `htmx-swapping` al contenido saliente durante el intervalo indicado, activando la transición CSS `.htmx-swapping` definida en `app.css` (tarea 13.2) — sin este modificador la clase se añade y se retira en 0ms y la transición nunca es visible. Criterio de listo: `grep -c 'swap:120ms' pkg/http/web/templates/tab_bar_items.html pkg/http/web/templates/chat_panel.html` reporta 2 coincidencias por archivo (4 en total); ningún atributo `hx-swap` quedó como `hx-swap="innerHTML"` sin el modificador en esos dos archivos.
- [x] 13.10 Ampliar `test/http_web_test.go` con: `TestAppCSSAsset_ServesCSSWithContentType` (verifica `GET /web/static/app.css` responde `200` con `Content-Type: text/css; charset=utf-8`), `TestFontAsset_ServesWoff2WithContentType` (verifica `GET /web/static/fonts/jetbrains-mono-400.woff2` responde `200` con `Content-Type: font/woff2`), `TestFontAsset_RejectsUnknownFilename` (verifica `GET /web/static/fonts/otra-cosa.woff2` responde `404`). Ninguna de estas rutas requiere cookie de autenticación (verificar explícitamente en los tests que no se envía cookie en la petición). Criterio de listo: `go test ./test/... -run 'TestAppCSSAsset|TestFontAsset'` pasa en verde.
- [x] 13.11 Actualizar `pkg/http/web/templates.go` si `template.Must(template.ParseFS(...))` fallara tras los cambios de 13.5/13.6/13.7/13.8 (no se espera fallo, ya que estos cambios son de marcado HTML/atributos, no de sintaxis `{{}}` de `html/template`); si falla, corregir la sintaxis introducida hasta que `go build ./pkg/http/web/...` compile sin panic en tiempo de inicialización. Criterio de listo: `go build ./pkg/http/web/...` compila y un test que invoque `web.NewWebHandler` (cualquiera de los ya existentes en `test/http_web_test.go`) no hace panic al arrancar.
- [x] 13.12 En `README.md`, en la sección "Interfaz web de chat", extender la línea de "Implementación completa" (que ya referencia `design.md` y `spec.md` del change `add-htmx-chat-frontend`) para incluir también `design-ui.md`: `[`design-ui.md`](./openspec/changes/add-htmx-chat-frontend/design-ui.md)` (decisiones de paleta, tipografía y accesibilidad). Criterio de listo: `grep -c "design-ui.md" README.md` devuelve al menos `1`.

## 14. Validación final (UI/UX)

- [x] 14.1 Ejecutar `gofmt -l .` desde la raíz del proyecto y confirmar que no reporta ningún archivo modificado en la sección 13.
- [x] 14.2 Ejecutar `go vet ./...` y `go build ./...` y confirmar que ambos terminan sin errores tras los cambios de la sección 13.
- [x] 14.3 Ejecutar `make unit-test` y confirmar que todos los tests pasan, incluidos los nuevos de la tarea 13.10 y todos los de la sección 10 (no deben requerir ninguna modificación de sus aserciones — los cambios de la sección 13 son de presentación, no de comportamiento observable a nivel de JSON/HTTP salvo las dos rutas nuevas de assets estáticos).
- [x] 14.4 Ejecutar `npx -y @fission-ai/openspec@latest validate add-htmx-chat-frontend --strict` desde la raíz del repo y corregir cualquier error de formato reportado (incluyendo los ajustes de `design.md` y `specs/chat-web-frontend/spec.md` hechos junto con esta sección) antes de considerar el cambio listo para `openspec archive add-htmx-chat-frontend`.

## 15. UI/UX — Feedback de espera y fluidez del envío — Frontend + Backend

Estas tareas corrigen los problemas observados en la prueba de uso real (respuestas del agente >10 s sin feedback visible, mensaje del usuario que no aparece hasta la respuesta, y burbujas `assistant` vacías de tool-calling en el historial). Decisiones D-U1 a D-U5 de `design-ui.md` §9.

- [x] 15.1 En `pkg/agent/service.go`, modificar `GetConversation` (D-U5): además de filtrar por rol, omitir los mensajes `assistant` con `Content == ""` (artefactos intermedios de tool-calling). Los mensajes `user` se conservan siempre; los `assistant` solo si tienen contenido no vacío. Criterio de listo: cubierto por `TestGetConversation_OmitsEmptyAssistantToolCallMessages`; los tests preexistentes `TestGetConversation_FiltersToolAndSystemMessages` y `TestGetConversation_PreservesChronologicalOrder` siguen pasando sin modificación.
- [x] 15.2 En `test/agent_service_test.go`, añadir `TestGetConversation_OmitsEmptyAssistantToolCallMessages`: historial con user + assistant vacío con ToolCalls + tool + assistant con contenido devuelve exactamente 2 mensajes (user y el assistant con contenido). Criterio de listo: `go test ./test/... -run TestGetConversation` pasa en verde.
- [x] 15.3 En `pkg/http/web/templates/chat_panel.html` (D-U2): añadir `<div id="typing-indicator" class="msg msg-assistant" aria-live="polite"><span class="typing-dots"><span></span><span></span><span></span></span></div>` inmediatamente antes del formulario; añadir `id="message-form"` y `hx-indicator="#typing-indicator"` al `<form>` de envío; añadir `autocomplete="off"` al input de mensaje. Criterio de listo: `grep -c 'id="typing-indicator"' pkg/http/web/templates/chat_panel.html` devuelve `1`; `grep -c 'hx-indicator="#typing-indicator"' pkg/http/web/templates/chat_panel.html` devuelve `1`.
- [x] 15.4 En `pkg/http/web/static/app.css` (D-U2/D-U3): añadir `#typing-indicator { display: none; }` y `#typing-indicator.htmx-request { display: block; }`; la animación `.typing-dots span` con `@keyframes typing-bounce` (tres puntos, retardo escalonado 0.15s) y su desactivación bajo `@media (prefers-reduced-motion: reduce)`; y la regla `#message-form.htmx-request input, #message-form.htmx-request .btn-primary { opacity: 0.55; pointer-events: none; }`. Criterio de listo: `grep -c 'typing-dots' pkg/http/web/static/app.css` mayor que 0 y `grep -c 'message-form.htmx-request' pkg/http/web/static/app.css` mayor que 0.
- [x] 15.5 En `pkg/http/web/templates/layout.html` (D-U1/D-U4): añadir un `<script>` inline (sin dependencias externas) que: (a) define `scrollChatToBottom()` (`panel.scrollTop = panel.scrollHeight` sobre `#chat-panel`) y lo invoca en `DOMContentLoaded` y en `htmx:afterSwap` cuando `evt.detail.target.id === "chat-panel-container"`; (b) escucha `htmx:beforeRequest` y, solo si `evt.detail.elt.id === "message-form"`: cancela con `evt.preventDefault()` si el form ya tiene `htmx-request` (evita doble envío por teclado); en caso contrario crea una burbuja `div.msg.msg-user` con `textContent` (nunca `innerHTML`), elimina un `.error` previo si existe, inserta la burbuja con `panel.insertBefore(bubble, indicator)` antes de `#typing-indicator`, limpia el input y hace scroll al fondo. Criterio de listo: el HTML de `GET /web` contiene `htmx:beforeRequest`, `htmx:afterSwap` e `insertBefore(bubble, indicator)`; `go build ./pkg/http/web/...` compila y las plantillas parsean sin panic.
- [x] 15.6 En `test/http_web_test.go`, añadir `TestIndex_ChatPanelIncludesTypingIndicator`: `GET /web` autenticado responde 200 y el cuerpo contiene `id="typing-indicator"` y `hx-indicator="#typing-indicator"`. Criterio de listo: `go test ./test/... -run TestIndex` pasa en verde.
- [x] 15.7 Actualizar `design-ui.md` (§9 con las decisiones D-U1 a D-U5), `specs/chat-web-frontend/spec.md` (escenarios "Feedback inmediato mientras el agente procesa" y "Mensajes intermedios de tool-calling no se muestran") y `specs/nl-mongo-agent/spec.md` (escenario de obtención de historial para presentación incluye la omisión de `assistant` vacíos). Criterio de listo: `npx -y @fission-ai/openspec@latest validate add-htmx-chat-frontend --strict` pasa sin errores.
- [x] 15.8 Validación de la sección: `gofmt -l .` vacío, `go vet ./...` y `go build ./...` sin errores, `go test ./test/...` completo en verde (sin modificar aserciones preexistentes), y verificación de humo con el servidor real (`GET /web` autenticado contiene los marcadores del script y del indicador; `app.css` contiene `typing-dots` y `message-form.htmx-request`).

## 16. UI/UX — Chat acotado en pantalla completa + tipografía Inter — Frontend

Estas tareas aplican la segunda revisión de alcance visual (decisiones D-U6 a D-U9 de `design-ui.md` §10): el chat deja de ocupar todo el ancho en pantallas grandes, la tipografía pasa de JetBrains Mono a Inter (Google Fonts, self-hosted), y se documenta la configuración del bucle agéntico.

- [x] 16.1 Vendorizar la tipografía `Inter` (subset `latin`, `@fontsource/inter` versión pineada `5.2.5`): descargar verbatim `files/inter-latin-400-normal.woff2` → `pkg/http/web/static/fonts/inter-400.woff2` y `files/inter-latin-600-normal.woff2` → `pkg/http/web/static/fonts/inter-600.woff2`. Eliminar los dos archivos `jetbrains-mono-*.woff2` anteriores. Criterio de listo: ambos archivos existen y `file` reporta `Web Open Font Format (Version 2)`; los archivos jetbrains ya no existen.
- [x] 16.2 En `pkg/http/web/static/app.css` (D-U6): reemplazar los dos `@font-face` por los de `Inter` (pesos 400 y 600, `font-display: swap`), renombrar la variable `--font-mono` a `--font-sans` con stack `"Inter", system-ui, -apple-system, "Segoe UI", Roboto, sans-serif` y actualizar todos sus usos. Criterio de listo: `grep -c 'JetBrains\|--font-mono' pkg/http/web/static/app.css` devuelve `0`; `grep -c 'inter-400.woff2' pkg/http/web/static/app.css` devuelve `1`.
- [x] 16.3 En `pkg/http/web/static.go` (D-U6): actualizar la lista cerrada de `serveFont` a `inter-400.woff2` e `inter-600.woff2` (cualquier otro nombre sigue devolviendo 404). Criterio de listo: `TestFontAsset_ServesWoff2WithContentType` y `TestFontAsset_RejectsUnknownFilename` pasan con los nombres nuevos.
- [x] 16.4 En `pkg/http/web/templates/layout.html` y `pkg/http/web/static/app.css` (D-U7): envolver `#tab-bar-container` y `#chat-panel-container` en `<main id="chat-shell">`; añadir las reglas `#chat-shell { display: flex; flex-direction: column; width: 100%; max-width: 52rem; height: 100vh; margin: 0 auto; border-left/right: 1px solid var(--color-border-subtle); }` y `#chat-panel-container { flex: 1; display: flex; flex-direction: column; min-height: 0; }`; `body` deja de ser flex (`body.login-page` conserva su propio centrado flex). Criterio de listo: `GET /web` contiene `id="chat-shell"` y `app.css` contiene `max-width: 52rem` y `#chat-shell`.
- [x] 16.5 En `pkg/http/web/static/app.css` (D-U8): añadir `#chat-panel > .btn-secondary { align-self: flex-end; margin-top: var(--space-2); }` para que el botón "Cerrar pestaña" no ocupe todo el ancho. Criterio de listo: `grep -c '#chat-panel > .btn-secondary' pkg/http/web/static/app.css` devuelve `1`.
- [x] 16.6 En `test/http_web_test.go`: actualizar `TestFontAsset_ServesWoff2WithContentType` al nombre `inter-400.woff2`, y ampliar `TestIndex_ChatPanelIncludesTypingIndicator` para asertar también `id="chat-shell"`. Criterio de listo: `go test ./test/... -run 'TestFontAsset|TestIndex'` pasa en verde.
- [x] 16.7 En `.env.example` (D-U9): documentar en la sección "Bucle agéntico" que `AGENT_MAX_TOOL_ITERATIONS` puede subirse (p. ej. 10) ante errores "tool loop exceeded" y que `AGENT_REQUEST_TIMEOUT_SECONDS` debe dar margen acorde. Criterio de listo: los comentarios aparecen en `.env.example`.
- [x] 16.8 Actualizar `design-ui.md` (§10, decisiones D-U6 a D-U9) y `specs/chat-web-frontend/spec.md` (nombres de fuentes vendorizadas). Criterio de listo: `npx -y @fission-ai/openspec@latest validate add-htmx-chat-frontend --strict` pasa sin errores.
- [x] 16.9 Validación de la sección: `gofmt -l .` vacío, `go vet ./...` y `go build ./...` sin errores, `go test ./test/...` completo en verde, y verificación de humo con el servidor real (fuentes `inter-*.woff2` responden 200 `font/woff2`, la antigua `jetbrains-mono-400.woff2` responde 404, `GET /web` contiene `id="chat-shell"`).

## 17. Respuestas tabulares — CSV en tools + tablas HTML en el chat — Backend + Frontend

Estas tareas implementan las decisiones D-W12 (formato CSV opcional en tools de consulta) y D-W13 (renderizado seguro de tablas Markdown) de `design.md`, y los estilos de `design-ui.md` §11. Motivación: las respuestas tabulares se mostraban como texto plano con adornos markdown literales (`**`, guiones), y los resultados JSON grandes inflaban el contexto del bucle agéntico.

- [x] 17.1 Crear `pkg/agent/csv_format.go` (D-W12) con `func jsonArrayToCSV(jsonResult string) (string, error)`: parsea el array JSON de objetos; columnas = unión estable de claves de primer nivel (claves ordenadas alfabéticamente por documento antes de la unión); valores primitivos tal cual (`float64` con `strconv.FormatFloat(v, 'f', -1, 64)`), `null` como celda vacía, objetos/arrays anidados como JSON compacto; usa `encoding/csv` de la librería estándar. Criterio de listo: cubierto por los tests internos `TestJSONArrayToCSV_*` (objetos planos, unión de columnas, anidados como JSON, array vacío, JSON inválido) en `pkg/agent/csv_format_test.go`.
- [x] 17.2 En `pkg/agent/tools.go` (D-W12): añadir la propiedad opcional `format` (`enum: ["json","csv"]`) a los schemas de `query_find` y `query_aggregate` en `BuildToolDefinitions`; añadir el helper privado `requestedFormat(args)` (devuelve `"csv"` solo si `format == "csv"`, si no `"json"`); en `DispatchToolCall`, tras un `Find`/`Aggregate` exitoso, si el formato es `"csv"` convertir con `jsonArrayToCSV` y devolver error de tool si la conversión falla. El comportamiento por defecto (JSON) no cambia. Criterio de listo: cubierto por `TestDispatchToolCall_FindWithCSVFormat`, `TestDispatchToolCall_AggregateWithCSVFormat` y `TestDispatchToolCall_DefaultFormatIsJSON`; los tests preexistentes `TestDispatchToolCall_*` siguen pasando sin modificación.
- [x] 17.3 Extender `fakeMongoRepo` de `test/agent_service_test.go` con los campos opcionales `findResult`/`aggregateResult string` (si vacíos, comportamiento anterior `"[]"`). Criterio de listo: todos los tests preexistentes pasan sin modificar sus aserciones.
- [x] 17.4 Actualizar el `systemPrompt` de `cmd/server/api.go`: instruir al agente a presentar datos tabulares SIEMPRE como tabla Markdown con pipes (cabecera + línea `|---|`), a no usar `**`/cursivas/encabezados (se muestran literales), y a usar `format="csv"` cuando el usuario pida exportar o el resultado tabular sea grande. Criterio de listo: `go build ./cmd/server/...` compila.
- [x] 17.5 Crear `pkg/http/web/markdown_table.go` (D-W13) con `type messageSegment struct { Kind string; Text string; Header []string; Rows [][]string }` (constantes `segmentKindText`/`segmentKindTable`) y `func parseMessageContent(content string) []messageSegment`: segmenta texto y tablas GFM (línea con pipes + línea separadora solo de `|`, `-`, `:` y espacios con al menos un guion); pipes sueltos en texto normal NO disparan el parser; nunca genera HTML. Criterio de listo: cubierto por los tests internos `TestParseMessageContent_*` en `pkg/http/web/markdown_table_test.go`.
- [x] 17.6 En `pkg/http/web/viewmodels.go`: cambiar `messageViewModel` de `Content string` a `Segments []messageSegment`; `toMessageViewModels` aplica `parseMessageContent` a cada mensaje. En `pkg/http/web/templates/chat_panel.html`: renderizar cada mensaje iterando segmentos — texto con `{{.Text}}` y tabla como `<table class="md-table">` con `{{.}}` en cabeceras y celdas (auto-escape intacto, sin `template.HTML`). Criterio de listo: `go build ./pkg/http/web/...` compila; cubierto por `TestIndex_RendersMarkdownTableAsHTMLTable` y `TestIndex_MessageContentIsHTMLEscapedInsideTable` (el payload `<script>` en una celda aparece escapado).
- [x] 17.7 En `pkg/http/web/static/app.css`: añadir los estilos `.md-table` de `design-ui.md` §11 (borde `--color-border-subtle`, cabecera con fondo `--color-surface` y texto `--color-text-secondary`, zebra striping sutil, `word-break: break-word`). Criterio de listo: `grep -c '.md-table' pkg/http/web/static/app.css` mayor que 0.
- [x] 17.8 Actualizar `design.md` (D-W12 y D-W13), `design-ui.md` (§11), `specs/nl-mongo-agent/spec.md` (Requirement "Formato de Resultado CSV en Consultas de Lectura" con sus 2 escenarios) y `specs/chat-web-frontend/spec.md` (escenario "Una respuesta con tabla Markdown se muestra como tabla HTML"). Criterio de listo: `npx -y @fission-ai/openspec@latest validate add-htmx-chat-frontend --strict` pasa sin errores.
- [x] 17.9 Validación de la sección: `gofmt -l .` vacío, `go vet ./...` y `go build ./...` sin errores, `go test ./test/... ./pkg/...` completo en verde (16 tests nuevos), y verificación de humo con el servidor real: una pregunta que pide tabla produce en la respuesta de `POST /web/tabs/:id/messages` un fragmento con `<table class="md-table">`, `<th>` y `<td>`.

## 18. Robustez de los tests de integración sobre Redis compartido — Tests

Corrección detectada en la validación final integral: los tests de integración de la sección 10.3 asumían un Redis vacío (aserciones `Len(sessions, N)` / `Empty(sessions)` sobre el listado global), por lo que fallaban en cuanto el Redis de desarrollo contenía sesiones reales de la app o residuos de corridas previas. El código de producción era correcto; el defecto era de las aserciones de los tests.

- [x] 18.1 En `test/integration/redis_session_test.go`, añadir el helper `findSession(sessions, sessionID) (agent.SessionSummary, bool)` y reescribir las aserciones de los 5 tests `TestSessionStore_*` (excepto `TestSessionStore_AppendAndGetHistory_RealRedis`, que no usa el índice) para que sean relativas a los IDs creados por el propio test: presencia por ID y título/actividad esperados (`TouchSession_*`), orden relativo de los 3 IDs dentro del listado global (`OrderedByMostRecentFirst`, con `assert.Less` sobre posiciones), y ausencia del ID tras expirar o limpiar (`LazilyRemovesExpiredEntries`, `ClearSession_RemovesIndexEntry`, con `assert.False(found)` en vez de `Empty`). Criterio de listo: los tests pasan con un Redis que contiene sesiones ajenas.
- [x] 18.2 En los mismos tests, registrar `t.Cleanup` con `ClearSession` de todos los IDs usados, de forma que la limpieza ocurra aunque el test falle a mitad (evita residuos en el índice `sessions:index`). Criterio de listo: tras dos corridas consecutivas de `go test -tags=integration ./test/integration/... -run TestSessionStore`, no quedan entradas de los tests en el listado.
- [x] 18.3 Validación de la sección: `gofmt -l .` vacío, `go vet -tags=integration ./test/integration/...` sin errores, y `REDIS_ADDR=localhost:6379 go test -tags=integration ./test/integration/...` en verde sobre el Redis de desarrollo con datos reales.
