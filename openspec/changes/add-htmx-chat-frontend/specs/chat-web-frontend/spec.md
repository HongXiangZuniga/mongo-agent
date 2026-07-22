## ADDED Requirements

### Requirement: Página de Chat con Pestañas por Sesión
El sistema SHALL exponer `GET /web`, que devuelve una página HTML completa con una barra de pestañas (una pestaña por sesión de chat existente) y el panel de chat de la pestaña más recientemente activa, renderizados server-side.

#### Scenario: Primera visita sin sesiones previas
- **GIVEN** un usuario autenticado en la interfaz web y ningún `session_id` indexado en Redis
- **WHEN** el usuario solicita `GET /web`
- **THEN** el sistema responde `200 OK` con una página que muestra exactamente una pestaña nueva sin mensajes previos y un formulario de entrada de texto vacío
- **AND** esa pestaña no ha sido persistida todavía en Redis (no aparecerá en una recarga posterior si no se envía ningún mensaje)

#### Scenario: Visita con sesiones existentes
- **GIVEN** existen dos o más sesiones indexadas en Redis con actividad previa
- **WHEN** el usuario solicita `GET /web`
- **THEN** el sistema responde `200 OK` con una barra de pestañas que lista todas las sesiones indexadas, ordenadas de la más reciente a la más antigua
- **AND** el panel de chat activo muestra los mensajes de la sesión con la actividad más reciente
- **AND** cada mensaje mostrado en el panel corresponde a los roles `user` o `assistant` únicamente

### Requirement: Creación de una Pestaña Nueva
El sistema SHALL permitir crear una pestaña de chat nueva sin recargar la página, mediante una petición htmx que intercambia únicamente la barra de pestañas y el panel de chat.

#### Scenario: Crear pestaña nueva
- **GIVEN** un usuario autenticado con la página `/web` ya cargada
- **WHEN** el usuario activa el control de "nueva pestaña" (petición htmx `POST /web/tabs`)
- **THEN** el sistema responde con un fragmento HTML que reemplaza la barra de pestañas (incluyendo la pestaña nueva marcada como activa) y el panel de chat (vacío, sin mensajes previos)
- **AND** el sistema no ejecuta ninguna escritura en Redis como parte de esta petición (la sesión permanece efímera hasta el primer mensaje)

### Requirement: Cambio entre Pestañas
El sistema SHALL permitir cambiar la pestaña activa sin recargar la página, mediante una petición htmx que solo intercambia el panel de chat.

#### Scenario: Cambiar a una pestaña existente
- **GIVEN** un usuario autenticado con al menos dos pestañas visibles en la barra
- **WHEN** el usuario hace clic en una pestaña distinta a la activa (petición htmx `GET /web/tabs/{session_id}`)
- **THEN** el sistema responde con el fragmento del panel de chat correspondiente a esa sesión, incluyendo su historial completo de mensajes `user`/`assistant`
- **AND** el sistema no modifica el orden ni la actividad registrada de ninguna sesión como consecuencia de este cambio

#### Scenario: Cambiar a una pestaña cuya sesión expiró en Redis
- **GIVEN** una pestaña visible en la barra cuya sesión ya expiró por TTL en Redis
- **WHEN** el usuario hace clic en esa pestaña
- **THEN** el sistema responde con un panel de chat vacío para ese `session_id`, sin error visible para el usuario

### Requirement: Envío de Mensajes desde la Pestaña Activa
El sistema SHALL permitir enviar una pregunta al agente desde la pestaña activa mediante una petición htmx, reutilizando exclusivamente el caso de uso `AgentService.Ask` ya existente.

#### Scenario: Enviar un mensaje válido
- **GIVEN** un usuario autenticado con una pestaña activa (nueva o existente)
- **WHEN** el usuario envía un mensaje no vacío (petición htmx `POST /web/tabs/{session_id}/messages`)
- **THEN** el sistema invoca `AgentService.Ask` con ese `session_id` y el texto del mensaje
- **AND** el sistema responde con el fragmento del panel de chat actualizado, mostrando el mensaje del usuario seguido de la respuesta del agente
- **AND** el sistema responde también con una actualización fuera de banda de la barra de pestañas reflejando la actividad más reciente de esa sesión

#### Scenario: Enviar un mensaje vacío
- **GIVEN** un usuario autenticado con una pestaña activa
- **WHEN** el usuario envía el formulario de mensaje sin texto
- **THEN** el sistema no invoca a `AgentService.Ask`
- **AND** el sistema responde con el panel de chat sin cambios, mostrando un mensaje de validación

#### Scenario: El agente falla al procesar el mensaje
- **GIVEN** un usuario autenticado envía un mensaje válido
- **WHEN** `AgentService.Ask` devuelve un error (timeout, límite de iteraciones excedido, o dependencia externa no disponible)
- **THEN** el sistema responde con el fragmento del panel de chat mostrando un mensaje de error genérico, sin exponer detalles internos (credenciales, stack traces)
- **AND** el mensaje del usuario permanece visible en el panel aunque la respuesta del agente haya fallado

#### Scenario: Feedback inmediato mientras el agente procesa
- **GIVEN** un usuario autenticado con una pestaña activa
- **WHEN** el usuario envía un mensaje no vacío y la petición htmx está en curso (la respuesta del agente puede tardar varios segundos)
- **THEN** el mensaje del usuario se muestra en el panel de chat inmediatamente, sin esperar a la respuesta del servidor (renderizado optimista en cliente)
- **AND** el campo de entrada queda vacío e inerte junto al botón de envío hasta que llega la respuesta
- **AND** se muestra un indicador de "el agente está escribiendo" (elemento `#typing-indicator` activado vía `hx-indicator`) hasta que el fragmento de respuesta reemplaza el panel
- **AND** el panel de chat desplaza su scroll hasta el mensaje más reciente tras cada intercambio de fragmento

#### Scenario: Mensajes intermedios de tool-calling no se muestran
- **GIVEN** una sesión cuyo historial persistido contiene mensajes de rol `assistant` con contenido vacío (artefactos intermedios que solo portan llamadas a herramientas)
- **WHEN** el sistema obtiene la conversación para presentación
- **THEN** esos mensajes vacíos se omiten del resultado mostrado en el panel de chat

#### Scenario: Una respuesta con tabla Markdown se muestra como tabla HTML
- **GIVEN** un mensaje del asistente cuyo contenido incluye una tabla en notación Markdown con pipes (cabecera, línea separadora `|---|`, filas)
- **WHEN** el sistema renderiza el panel de chat
- **THEN** la tabla se presenta como un elemento `<table>` HTML con cabecera y filas, con estilos coherentes con el tema visual
- **AND** el contenido de cada celda se interpola con escape HTML (ningún marcado incluido en el mensaje se interpreta como HTML)
- **AND** el texto fuera de la tabla se sigue mostrando como texto plano con sus saltos de línea

### Requirement: Cierre de una Pestaña
El sistema SHALL permitir cerrar una pestaña de chat, eliminando su sesión asociada de Redis, mediante una petición htmx.

#### Scenario: Cerrar una pestaña con otras pestañas disponibles
- **GIVEN** un usuario autenticado con al menos dos pestañas visibles
- **WHEN** el usuario cierra una pestaña (petición htmx `DELETE /web/tabs/{session_id}`)
- **THEN** el sistema invoca la eliminación de esa sesión (historial, metadata e índice) en Redis
- **AND** el sistema responde con la barra de pestañas actualizada (sin la pestaña cerrada) y el panel de chat de la pestaña que pasa a estar activa (la más reciente restante)

#### Scenario: Cerrar la última pestaña visible
- **GIVEN** un usuario autenticado con exactamente una pestaña visible
- **WHEN** el usuario cierra esa pestaña
- **THEN** el sistema elimina la sesión asociada de Redis (si estaba persistida)
- **AND** el sistema responde con una pestaña nueva vacía como única pestaña activa, sin persistirla todavía en Redis

### Requirement: Persistencia del Listado de Pestañas en Redis
El sistema SHALL indexar en Redis toda sesión que haya recibido al menos un mensaje, de forma que el listado de pestañas sobreviva a una recarga completa del navegador, y SHALL dejar de listar una sesión automáticamente en cuanto expire su TTL, sin requerir un proceso de limpieza en segundo plano.

#### Scenario: El listado de pestañas persiste tras recargar el navegador
- **GIVEN** un usuario envió al menos un mensaje en dos sesiones distintas
- **WHEN** el usuario recarga completamente la página (`GET /web` de nuevo, nueva petición de navegador)
- **THEN** el sistema muestra ambas pestañas en la barra, en el mismo orden de actividad reciente

#### Scenario: Una pestaña deja de listarse tras expirar su TTL
- **GIVEN** una sesión indexada cuyo TTL en Redis ya expiró
- **WHEN** el usuario recarga la página `GET /web`
- **THEN** el sistema ya no incluye esa sesión en la barra de pestañas
- **AND** el sistema no requiere ningún job ni proceso en segundo plano para lograr este resultado

### Requirement: Autenticación de la Interfaz Web por Cookie
El sistema SHALL exigir una cookie de autenticación válida para acceder a cualquier ruta bajo `/web` salvo `GET /web/login`, `POST /web/login` y los activos estáticos, validando el valor de la cookie contra la misma variable de entorno `API_TOKEN` usada por `POST /ask`.

#### Scenario: Acceso sin cookie a una petición de navegación completa
- **GIVEN** un cliente sin la cookie de autenticación configurada
- **WHEN** el cliente solicita `GET /web` (petición de navegación normal, sin header `HX-Request`)
- **THEN** el sistema responde con una redirección `303 See Other` hacia `GET /web/login`

#### Scenario: Acceso sin cookie a una petición htmx parcial
- **GIVEN** un cliente sin la cookie de autenticación configurada (por ejemplo, la cookie expiró mientras la página seguía abierta)
- **WHEN** el cliente envía una petición htmx (header `HX-Request: true`) hacia cualquier ruta protegida de `/web`
- **THEN** el sistema responde `401 Unauthorized` con el header `HX-Redirect: /web/login` y cuerpo vacío
- **AND** el sistema no invoca a `AgentService` para procesar esa petición

#### Scenario: Login exitoso
- **GIVEN** el servicio tiene configurada la variable de entorno `API_TOKEN`
- **WHEN** el usuario envía `POST /web/login` con el campo `token` igual al valor de `API_TOKEN`
- **THEN** el sistema fija una cookie HttpOnly con `SameSite=Strict` cuyo valor es `API_TOKEN`
- **AND** el sistema responde con una redirección `303 See Other` hacia `GET /web`

#### Scenario: Login con token inválido
- **WHEN** el usuario envía `POST /web/login` con un campo `token` que no coincide con `API_TOKEN`
- **THEN** el sistema responde `401 Unauthorized` renderizando de nuevo el formulario de login con un mensaje de error
- **AND** el sistema no fija ninguna cookie

#### Scenario: Logout
- **GIVEN** un usuario autenticado con la cookie de sesión fijada
- **WHEN** el usuario envía `POST /web/logout`
- **THEN** el sistema revoca la cookie (la invalida fijando un `Max-Age` negativo)
- **AND** el sistema responde con una redirección `303 See Other` hacia `GET /web/login`

### Requirement: Activos Estáticos Servidos sin Build de Frontend
El sistema SHALL servir el script `htmx.min.js`, la hoja de estilos `app.css` y las fuentes tipográficas vendorizadas directamente embebidos en el binario Go, sin depender de Node.js, `npm`, ni ningún paso de compilación de frontend, y sin requerir autenticación para acceder a ellos.

#### Scenario: Descarga del script htmx
- **WHEN** cualquier cliente (autenticado o no) solicita `GET /web/static/htmx.min.js`
- **THEN** el sistema responde `200 OK` con `Content-Type: application/javascript; charset=utf-8` y el contenido del archivo `htmx.min.js` vendored
- **AND** el sistema no requiere autenticación para esta ruta

#### Scenario: Descarga de la hoja de estilos
- **WHEN** cualquier cliente (autenticado o no) solicita `GET /web/static/app.css`
- **THEN** el sistema responde `200 OK` con `Content-Type: text/css; charset=utf-8` y el contenido del archivo `app.css` vendorizado, conteniendo la paleta, tipografía y reglas de componente descritas en `design-ui.md`
- **AND** el sistema no requiere autenticación para esta ruta

#### Scenario: Descarga de una fuente tipográfica vendorizada
- **WHEN** cualquier cliente (autenticado o no) solicita `GET /web/static/fonts/inter-400.woff2` o `GET /web/static/fonts/inter-600.woff2`
- **THEN** el sistema responde `200 OK` con `Content-Type: font/woff2` y el contenido del archivo `.woff2` vendorizado correspondiente
- **AND** el sistema no requiere autenticación para esta ruta
- **AND** ninguna plantilla HTML carga tipografías desde un CDN externo en tiempo de ejecución
