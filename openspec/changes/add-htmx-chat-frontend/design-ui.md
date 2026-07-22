# Decisiones de diseño UI/UX — `add-htmx-chat-frontend`

## 1. Contexto

Este documento cubre el sistema visual de las dos vistas HTML server-rendered ya
implementadas en `pkg/http/web/templates/`:

- **`layout.html`** — shell principal: barra de pestañas (`#tab-bar-container`) +
  panel de chat activo (`#chat-panel-container`), intercambiados vía htmx sin
  recarga de página.
- **`login.html`** — pantalla de login con un único campo `password` (token de
  acceso).
- Fragmentos parciales que heredan los mismos estilos: `tab_bar_items.html`,
  `chat_panel.html`.

Audiencia: el propio operador (uso personal/portafolio) y cualquier persona
técnica (reclutador, otro developer) que evalúe el proyecto como pieza de
portafolio. No es un producto de consumo masivo — la referencia visual correcta
es una terminal moderna / herramienta de developer (tipo Warp, GitHub Copilot
Chat, un cliente de API), no un chat de mensajería como WhatsApp o Telegram.

El contenido que muestran los mensajes del asistente es, por naturaleza,
técnico: consultas MongoDB y resultados JSON. Esto es una señal de producto
real, no solo estética, para preferir una tipografía monoespaciada en el
cuerpo del chat (ver §3).

## 2. Stack detectado

- Go `html/template` + CSS3 vanilla embebido, **sin paso de build** (confirmado
  leyendo `layout.html` y `login.html`: `<style>` inline, sin clases utility
  generadas, sin Tailwind).
- Único asset de terceros vendorizado hoy: `htmx.min.js` 1.9.12 vía
  `//go:embed`, servido en `/web/static/htmx.min.js`.
- No hay Node/npm/bundler en el repo (`go.mod` es el único manifiesto de
  dependencias; no existe `package.json`).

Toda recomendación de abajo está expresada en CSS3 plano con variables nativas
(`:root { --token: valor; }`), sin utility classes de build tooling.

## 3. Sistema de diseño

### 3.1 Estilo elegido: "AI-Native UI" con inflexión terminal/dev-tool

Búsqueda ejecutada:
```
search.py "developer tool AI conversational agent chat interface terminal-inspired minimal" \
  --design-system --variance 5 --motion 3 --density 6
```
Resultado: estilo **AI-Native UI** (chrome mínimo, streaming text, tono
"agentic/ambient"), soporte completo dark mode, WCAG AA declarado. Densidad
media (6/10, escala de espaciado 16–64px) porque es una sola columna de chat,
no un dashboard denso. Motion baja (3/10, subtle) porque el "avoid" explícito
del propio resultado es **"Heavy chrome + Slow response feedback"** — encaja
con evitar animaciones decorativas.

Se elige **tema oscuro único** (sin toggle claro/oscuro). Justificación: (a)
el código actual no tiene infraestructura de theming (no hay JS de cambio de
tema, no hay `prefers-color-scheme` branching), añadirla sería over-engineering
para una pieza de portafolio de un solo operador; (b) el dark mode es el
lenguaje visual dominante en herramientas de developer (terminales, editores,
CLIs) y refuerza el posicionamiento "dev tool" pedido. Si en el futuro se
quiere modo claro, es una decisión nueva a evaluar aparte, no una omisión de
este documento.

### 3.2 Paleta de color

Basada en la familia Slate (fondo) + acento verde ("code dark + run green",
nota explícita del resultado del skill para este estilo). Ratios de contraste
calculados con la fórmula WCAG de luminancia relativa (no estimados a ojo):

```css
:root {
  /* Fondo / superficies */
  --color-bg:            #0F172A; /* fondo de página, panel de chat */
  --color-surface:       #1A2436; /* tab-bar, header, .login-box */
  --color-surface-2:     #1E293B; /* burbuja mensaje asistente, hover de botones */
  --color-surface-user:  #16321F; /* burbuja mensaje usuario (verde muy oscuro) */

  /* Bordes */
  --color-border:        #5B6B85; /* bordes de inputs/botones/tabs — cumple 3:1 no-texto */
  --color-border-subtle: #334155; /* separadores decorativos (borde inferior tab-bar, borde de burbuja) */

  /* Texto */
  --color-text:           #F1F5F9; /* texto primario */
  --color-text-secondary: #94A3B8; /* etiquetas de rol, meta info, placeholder */
  --color-text-muted:     #64748B; /* SOLO texto grande (>=18px) o decorativo, no cumple 4.5:1 en texto normal */

  /* Acento */
  --color-accent:        #22C55E; /* indicador de tab activo, foco, enlaces, borde acento */
  --color-accent-strong:  #16A34A; /* hover/active de accent */
  --color-on-accent:      #0B1220; /* texto sobre fondo --color-accent (botón primario) */

  /* Estado */
  --color-danger:         #F87171; /* texto de error sobre fondo oscuro */
  --color-danger-bg:      #3A1518; /* fondo de .error */
  --color-danger-border:  #7F2A2E;
}
```

Pares verificados (fórmula WCAG 2.x, `L = 0.2126R + 0.7152G + 0.0722B` con
corrección gamma; ratio = `(L1+0.05)/(L2+0.05)`):

| Par | Ratio | Uso | Cumple |
|---|---|---|---|
| `--color-text` sobre `--color-bg` | 16.30:1 | texto de página | AAA |
| `--color-text-secondary` sobre `--color-bg` | 6.96:1 | etiquetas de rol, meta | AAA |
| `--color-text-muted` sobre `--color-bg` | 3.75:1 | **solo** texto ≥18px/negrita ≥14px | falla AA normal, ok AA large |
| `--color-accent` sobre `--color-bg` | 7.83:1 | indicador de tab, enlaces | AAA |
| `--color-danger` sobre `--color-bg` | 6.45:1 | texto `.error` | AAA |
| `--color-on-accent` sobre `--color-accent` | 8.22:1 | texto de botón primario (fondo verde) | AAA |
| `--color-text` sobre `--color-surface-2` | 13.35:1 | texto de burbuja asistente | AAA |
| `--color-text-secondary` sobre `--color-surface-2` | 5.71:1 | etiqueta "Agente" | AA |
| `--color-text` sobre `--color-surface-user` | 12.69:1 | texto de burbuja usuario | AAA |
| `--color-accent` sobre `--color-surface-2` | 6.42:1 | acentos sobre burbuja | AAA |
| `--color-border` sobre `--color-bg` | 3.30:1 | borde de inputs/botones (no-texto, mínimo 3:1) | AA (1.4.11) |

`--color-border-subtle` (1.72:1 sobre `--color-bg`) es intencionalmente bajo:
se usa solo en separadores puramente decorativos (línea bajo el tab-bar,
contorno de la burbuja del asistente) donde el límite ya es perceptible por el
cambio de color de fondo, no por el borde — no aplica el requisito de 3:1 de
"non-text contrast" porque no es el único medio de identificar un componente
interactivo.

**Regla de uso de `--color-text-muted`:** no usarlo en texto de mensajes,
labels de formulario ni botones. Reservarlo para placeholder de inputs (que
el navegador ya renderiza con menor énfasis) o metadatos secundarios en
tamaño ≥18px.

### 3.3 Tipografía

**Decisión: JetBrains Mono, autoalojada (self-hosted `.woff2`), no vía CDN de
Google Fonts.**

Justificación explícita del dilema sistema-vs-Google-Font:

- Se descarta la fuente de sistema (`system-ui, ...`) pese a ser la opción de
  menor superficie, porque (a) el estilo "AI-Native UI / terminal" identificado
  por el skill depende de una tipografía monoespaciada consistente para
  transmitir "dev tool", y las fuentes monoespaciadas de sistema varían
  fuertemente entre SO (SF Mono en macOS, Cascadia Code/Consolas en Windows,
  DejaVu Sans Mono en Linux) — el x-height, el tracking y el peso visual
  difieren lo suficiente como para que la pieza de portafolio se vea distinta
  (y menos cuidada) según el visitante; (b) el contenido real que se muestra
  (queries MongoDB, JSON) se beneficia funcionalmente de un monoespaciado
  diseñado para código, no solo estéticamente.
- Se descarta cargarla desde `fonts.googleapis.com` en runtime porque **rompe
  la filosofía "self-contained, sin dependencias externas"** que ya se aplicó
  a `htmx.min.js` (vendorizado vía `//go:embed` en lugar de CDN). Cargar
  Google Fonts en runtime añadiría una dependencia de red no auditada, un
  posible bloqueo de renderizado y una inconsistencia de patrón dentro del
  mismo proyecto.
- Se autoaloja siguiendo el **mismo patrón exacto** que `htmx.min.js`:
  descargar el `.woff2` una vez, vendorizarlo en el repo, servirlo vía
  `//go:embed` en una ruta propia (ver §8), con `@font-face` apuntando a esa
  ruta local. Cero requests a terceros en runtime.
- Solo se vendorizan **2 pesos** (no la familia variable completa) para
  minimizar tamaño: `400` (regular, texto de mensajes/UI) y `600` (semibold,
  título de login, tab activa). Subset `latin` únicamente (suficiente para
  español/inglés técnico). Esto son ~2 archivos `.woff2` de ~20–25 KB cada
  uno — comparable en costo a vendorizar `htmx.min.js`.

```css
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

:root {
  --font-mono: "JetBrains Mono", ui-monospace, "SF Mono", "Cascadia Code",
    "Roboto Mono", monospace;
}

body { font-family: var(--font-mono); }
```

El fallback stack (`ui-monospace, "SF Mono", ...`) se mantiene por si el
`.woff2` tarda en cargar o falla — degradación aceptable, nunca rota
(`font-display: swap` evita FOIT).

**Escala tipográfica:**

```css
:root {
  --font-size-sm:   0.8125rem; /* 13px — labels de rol, texto de tabs, meta info, botones */
  --font-size-base: 1rem;      /* 16px — contenido de mensajes, inputs */
  --font-size-lg:   1.25rem;   /* 20px — título "mongo-agent" en login/h1 */

  --line-height-tight: 1.3; /* tabs, botones, labels */
  --line-height-body:  1.5; /* contenido de mensaje — mejora legibilidad de JSON/texto multilínea */
}
```

Nota: 13px para chrome de UI (tabs, botones) está por encima del piso de
anti-patrón (`<12px`) del checklist de accesibilidad, y es el tamaño idiomático
en herramientas de developer (editores de código, DevTools de navegador). El
cuerpo del mensaje usa 16px completo por legibilidad de párrafos largos.

Con fuente monoespaciada, **no usar peso `700`/bold** para énfasis (el propio
resultado del skill lo señala: "bold ruins mono character" en la familia
JetBrains Mono) — usar el peso `600` vendorizado para jerarquía, o el color
`--color-accent`/`--color-text-secondary` para diferenciar en vez de negrita
pesada.

### 3.4 Densidad / espaciado

```css
:root {
  --space-1: 0.25rem; /* 4px */
  --space-2: 0.5rem;  /* 8px */
  --space-3: 0.75rem; /* 12px */
  --space-4: 1rem;    /* 16px */
  --space-5: 1.5rem;  /* 24px */
  --space-6: 2rem;    /* 32px */
}
```
Escala estándar (densidad 6/10): suficiente aire para que el chat no se sienta
apretado, sin el espaciado expansivo de una landing de marketing (que sería
denso 1-3/10). Se mantiene la métrica ya usada en el CSS actual
(`padding: 0.5rem 1rem` en botones, `padding: 1rem` en `#chat-panel`) — son
compatibles con esta escala, no hace falta reescribirlas desde cero.

## 4. Decisiones por componente

### 4.1 Barra de pestañas (`#tab-bar`, `#tab-bar button`, `.tab-active`)

- Contenedor: `background: var(--color-surface); border-bottom: 1px solid var(--color-border-subtle);` (se mantiene `overflow-x: auto` ya existente para muchas pestañas).
- Pestaña **inactiva**: `background: transparent; color: var(--color-text-secondary); border: 1px solid transparent; font-weight: 400;`.
- Pestaña inactiva **hover**: `background: var(--color-surface-2); color: var(--color-text); transition: background-color 150ms ease, color 150ms ease; cursor: pointer;`.
- Pestaña **activa** (`.tab-active`): `background: var(--color-bg)` (funde visualmente con el panel de chat de abajo, patrón clásico de tab UI), `color: var(--color-text)`, `font-weight: 600`, `font-family` hereda el peso 600 vendorizado, y un indicador de 2px en el borde superior: `border-top: 2px solid var(--color-accent);` — refuerzo visual adicional a color/peso para no depender solo del color (evita el anti-patrón "no visual feedback on current location" del check de navegación).
- Se conserva `border-radius: 0.375rem 0.375rem 0 0` ya existente.
- Botón `+` (nueva pestaña): mismo estilo base que pestaña inactiva pero `color: var(--color-accent)` para diferenciarlo como acción, no como pestaña de sesión.

### 4.2 Panel de chat y mensajes (`#chat-panel`, `.msg`, `.msg-user`, `.msg-assistant`)

- `#chat-panel`: `background: var(--color-bg);` (ya hereda de `body`, no requiere override), mantiene `overflow-y: auto` y `padding: var(--space-4)`.
- `.msg`: mantiene `max-width: 80%`, `border-radius: 0.75rem`, `white-space: pre-wrap` (ya presente en el CSS actual — correcto, preservar: es necesario para que el output JSON/queries del agente no colapse el formato). Añadir `word-break: break-word;` por si el JSON incluye strings largos sin espacios.
- `.msg-user`: `align-self: flex-end; background: var(--color-surface-user); border: 1px solid rgba(34,197,94,0.25); color: var(--color-text);`.
- `.msg-assistant`: `align-self: flex-start; background: var(--color-surface-2); border: 1px solid var(--color-border-subtle); color: var(--color-text);`.
- **Etiqueta de rol ("Tú" / "Agente")** — implementable 100% en CSS, sin tocar el HTML (`{{.Role}}` ya produce las clases `msg-user`/`msg-assistant`):

```css
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
```

### 4.3 Formulario de envío y botones

- `input[type="text"]`, `input[type="password"]`: `background: var(--color-surface); border: 1px solid var(--color-border); color: var(--color-text); border-radius: 0.375rem;`. Placeholder (si se agrega) hereda `--color-text-muted`.
- Botón de envío ("Enviar"): recomendado como acción primaria — `background: var(--color-accent); color: var(--color-on-accent); border: 1px solid var(--color-accent); font-weight: 600;`, hover `background: var(--color-accent-strong)`.
- Botón "Cerrar pestaña" y botón `+`: acción secundaria — `background: transparent; color: var(--color-text-secondary); border: 1px solid var(--color-border);`, hover `color: var(--color-text); border-color: var(--color-accent);`.

  **Nota para tasks.md (no es mío editarlo):** hoy el selector `button[type="submit"], button` en el CSS actual aplica el mismo estilo a "Enviar", "Cerrar pestaña" y "+" — para diferenciar primaria vs. secundaria como se describe arriba hace falta un cambio mínimo de marcado: agregar `class="btn-primary"` al botón "Enviar" en `chat_panel.html` y `class="btn-secondary"` a "Cerrar pestaña" y al botón `+` en `tab_bar_items.html`. Es un cambio de una línea por template, no un rediseño — lo señalo para que se incluya como tarea de implementación.

### 4.4 Login (`.login-box`)

- `.login-box`: `background: var(--color-surface); border: 1px solid var(--color-border-subtle); box-shadow: 0 8px 24px rgba(0,0,0,0.35);` (sombra más marcada que en el CSS actual porque sobre fondo oscuro una sombra sutil no se percibe; se compensa con opacidad mayor).
- `h1`: `font-size: var(--font-size-lg); font-weight: 600; color: var(--color-text); font-family: var(--font-mono);` — mantiene el texto literal `mongo-agent`, coherente con la identidad "dev tool".
- Botón de submit: mismo tratamiento que botón primario del §4.3 (`background: var(--color-accent); color: var(--color-on-accent);`) en vez del `background: #1a1a1a` actual — unifica el lenguaje de "acción primaria" entre login y chat.

### 4.5 Mensaje de error (`.error`)

```css
.error {
  color: var(--color-danger);
  background: var(--color-danger-bg);
  border: 1px solid var(--color-danger-border);
  border-radius: 0.375rem;
  padding: var(--space-3) var(--space-4);
  margin-bottom: var(--space-4);
}
```

## 5. Accesibilidad — checklist aplicado

- **Contraste de texto (WCAG AA/AAA):** todos los pares texto/fondo usados en
  contenido real (mensajes, labels, botones, errores) verificados ≥5.7:1 —
  ver tabla en §3.2. Único token restringido: `--color-text-muted` (3.75:1,
  no usar en texto normal <18px).
- **Foco de teclado (`:focus-visible`)** — gap real detectado: el CSS actual
  no define ningún estilo de foco. Se agrega globalmente:

```css
:focus-visible {
  outline: 2px solid var(--color-accent);
  outline-offset: 2px;
  border-radius: 0.25rem;
}
/* Nunca outline:none sin reemplazo — si algún selector legado lo tuviera, eliminarlo */
```

  Aplica automáticamente (por ser regla global) a: botones de pestaña,
  botón `+`, input de mensaje, input de password, botón "Enviar", botón
  "Cerrar pestaña". El contraste de `--color-accent` (#22C55E) contra
  `--color-bg`/`--color-surface` es 7.83:1 / ≥6.4:1 — cumple holgadamente el
  3:1 mínimo de indicadores de foco no-textuales.
- **Touch targets:** botones existentes usan `padding: 0.5rem 1rem` sobre
  fuente 13–16px, lo que en la práctica da una altura ≈36–40px. Se recomienda
  subir el `padding` vertical de botones táctiles críticos (Enviar, tabs) a
  `0.625rem` para acercarse a los 44px recomendados en touch, sin necesidad de
  rediseñar el layout (relevante porque htmx no distingue mouse/touch y el
  proyecto es responsive por el `viewport meta` ya presente).
- **Etiquetas de formulario:** ya existen (`<label for="token">`) en
  `login.html` — correcto, mantener. El input de mensaje en `chat_panel.html`
  no tiene `<label>` visible; recomendado (para el arquitecto) usar
  `aria-label="Mensaje"` en el `<input type="text">` ya que un label visible
  rompería el layout de una sola línea input+botón — es un cambio de atributo,
  no de diseño visual, lo señalo pero no lo implemento.
- **`prefers-reduced-motion`:** respetado explícitamente en la animación de
  §6.
- **Roles de mensaje perceptibles sin depender solo del color:** la etiqueta
  textual "Tú"/"Agente" (§4.2) y la alineación izquierda/derecha ya cumplen
  esto — el color no es el único indicador de rol.

## 6. Animación (opcional, justificada)

Se recomienda una transición de fade **muy sutil** (120ms) en los swaps de
htmx, aprovechando las clases que htmx ya añade automáticamente durante el
ciclo de swap (`htmx-swapping`) — **sin JavaScript adicional**, solo CSS:

```css
.htmx-swapping {
  opacity: 0;
  transition: opacity 120ms ease-out;
}

@media (prefers-reduced-motion: reduce) {
  .htmx-swapping { transition: none; }
}
```

Justificación: el propio resultado del skill marca **"Heavy chrome + Slow
response feedback"** como anti-patrón a evitar — es decir, la recomendación no
es añadir movimiento por estética, sino evitar que el intercambio de fragmento
se sienta "roto" (contenido reemplazado de golpe). 120ms está por debajo del
rango sugerido de 150–300ms para mantenerlo casi imperceptible y no introducir
latencia percibida en una herramienta que se vende por rapidez de respuesta.
Si se prefiere cero animación por simplicidad, esta sección es completamente
omisible sin afectar ninguna otra decisión del documento — no es requisito
funcional del proposal.

## 7. Anti-patrones evitados (del output del skill, no genéricos)

- **"Heavy chrome + Slow response feedback"** (anti-patrón explícito del
  estilo AI-Native UI) → chrome mínimo en tab-bar (sin sombras duras, sin
  bordes gruesos), animación de swap ≤120ms.
- **Bold en fuente monoespaciada** ("bold ruins mono character", nota del
  resultado de typography) → se usa el peso `600` vendorizado en vez de
  negrita sintética del navegador, y color/mayúsculas para jerarquía en vez
  de peso extra.
- **Depender solo del color para indicar estado** (tab activa, rol de
  mensaje) → se refuerza con `font-weight`, borde superior de acento,
  alineación y etiqueta textual, no solo con un cambio de tono.
- **Quitar el outline de foco sin reemplazo** (anti-patrón "Removing focus
  rings" del checklist de accesibilidad, prioridad 1/CRITICAL) → gap real
  detectado en el código actual, corregido con `:focus-visible` global en
  §5.
- **Texto secundario ilegible sobre fondo oscuro** (anti-patrón "Gray-on-gray")
  → `--color-text-muted` (3.75:1) queda explícitamente restringido a texto
  grande/decorativo, no se usa en contenido de mensajes ni labels.

## 8. Archivos a tocar/crear (resumen para el arquitecto)
**Recomendación: extraer a `pkg/http/web/static/app.css`, servido vía
`GET /web/static/app.css` con el mismo patrón `//go:embed` que
`htmx.min.js`.**

No mantener el `<style>` inline duplicado en `layout.html` y `login.html`.

Trade-off evaluado:
- *A favor de extraer:* el `<style>` actual ya está duplicado byte-a-byte
  entre `layout.html` y `login.html` (confirmado leyendo ambos archivos) —
  cualquier cambio de paleta/tipografía a futuro requeriría editar dos
  lugares y mantenerlos sincronizados a mano, lo cual es frágil. Un archivo
  único es la fuente de verdad, cacheable por el navegador entre la página de
  login y el chat (el usuario ya paga el costo de descarga de `htmx.min.js`
  por separado; un `app.css` embebido sigue el mismo patrón arquitectónico ya
  aceptado en el proyecto).
- *En contra (costo asumido):* una request HTTP adicional en el primer load
  (mitigable con `Cache-Control: public, max-age=31536000, immutable` ya que
  el contenido es estático y versionado por deploy) y un archivo más a
  mantener en el repo — aceptable dado que ya existe el patrón de
  `//go:embed` + ruta pública para `htmx.min.js`, no es una técnica nueva.

Archivos que resultarían de aplicar este documento (a cargo del arquitecto,
no de mí):
- **Nuevo:** `pkg/http/web/static/app.css` (contenido: todo el CSS de §3–§7).
- **Nuevo:** `pkg/http/web/static/fonts/jetbrains-mono-400.woff2` y
  `jetbrains-mono-600.woff2` (vendorizados, subset `latin`).
- **Modificar:** `layout.html` y `login.html` — quitar el bloque `<style>`
  inline, agregar `<link rel="stylesheet" href="/web/static/app.css">` y
  `<link rel="preload" as="font" ...>` opcional para los `.woff2`.
- **Modificar (marcado mínimo, no visual):** `chat_panel.html` (clase
  `btn-primary` en "Enviar", `aria-label="Mensaje"` en el input) y
  `tab_bar_items.html` (clase `btn-secondary` en botones no-primarios) — ver
  notas en §4.3 y §5.
- Requiere una nueva ruta/handler Go (`GET /web/static/app.css` y
  `GET /web/static/fonts/*`) con el mismo mecanismo `//go:embed` ya usado
  para `htmx.min.js` — esto es responsabilidad del arquitecto en
  `tasks.md`, no algo que yo implemente.

## 9. Feedback de espera y fluidez del envío (añadido tras prueba de uso real)

Problema observado en uso real: la respuesta del agente puede tardar >10 s
(llamada LLM + tool-calling). Con el re-render completo del panel (D-W7 de
`design.md`), durante toda esa espera (a) el mensaje recién enviado no
aparece en pantalla, (b) el texto queda "atascado" en el input, y (c) no hay
ningún indicador de actividad. Además, el historial mostraba burbujas
`assistant` vacías (mensajes intermedios de tool-calling sin contenido).

Decisiones (todas implementadas en la sección 15 de `tasks.md`):

- **D-U1. Renderizado optimista del mensaje del usuario.** Un `<script>`
  inline mínimo en `layout.html` (sin build, sin framework, coherente con el
  enfoque de htmx ya vendorizado) escucha `htmx:beforeRequest` sobre el form
  `#message-form`: crea la burbuja `.msg .msg-user` con `textContent`
  (nunca `innerHTML`, para no reintroducir un vector de XSS), la inserta
  antes de `#typing-indicator`, limpia el input y elimina cualquier `.error`
  previo. Cuando llega la respuesta, el swap reemplaza el panel completo con
  el estado real del servidor, por lo que no hay divergencia posible.
- **D-U2. Indicador "el agente está escribiendo" (`#typing-indicator`).**
  Burbuja `.msg .msg-assistant` con tres puntos animados en CSS puro
  (`@keyframes typing-bounce`, 1.2 s, respetando
  `prefers-reduced-motion`), oculta por defecto (`display: none`) y activada
  por htmx vía `hx-indicator="#typing-indicator"` en el form (htmx le añade
  la clase `htmx-request` durante la petición). El indicador vive dentro del
  propio `#chat-panel`, así que desaparece solo con el swap — no hay estado
  que limpiar.
- **D-U3. Formulario inerte durante el envío.** htmx añade `htmx-request`
  al propio `<form>`; con CSS (`#message-form.htmx-request input,
  #message-form.htmx-request .btn-primary { opacity: 0.55; pointer-events:
  none; }`) se comunica el estado de espera y se evitan dobles clics. El
  listener de D-U1 además cancela (`evt.preventDefault()`) un segundo submit
  por teclado mientras el form ya tiene `htmx-request`.
- **D-U4. Auto-scroll al último mensaje.** El mismo `<script>` inline hace
  `panel.scrollTop = panel.scrollHeight` en `DOMContentLoaded` (carga
  inicial con historial), tras cada `htmx:afterSwap` cuyo target es
  `#chat-panel-container`, y tras el renderizado optimista de D-U1.
- **D-U5. Los mensajes `assistant` vacíos no se presentan.** Corrección de
  fondo (no cosmética): `AgentService.GetConversation` omite además los
  mensajes `assistant` con `Content == ""` (artefactos de tool-calling),
  coherente con D-W8 de `design.md` (el detalle de tool-calling no es
  contenido presentable). El historial completo para el LLM no cambia.

## 10. Revisión de alcance visual (tras segunda prueba de uso real)

Feedback del operador: el chat ocupando todo el ancho de una pantalla grande
resultaba excesivo, y se pidió una tipografía distinta (otra de Google Fonts)
y un conjunto "más pequeño y mucho más sencillo".

- **D-U6. Tipografía Inter (reemplaza a JetBrains Mono).** Búsqueda en la
  base del skill (`--domain typography`, query "simple minimal chat
  interface clean readable sans"): primer resultado "Minimal Swiss"
  (Inter para headings y body; notas: "ultimate simplicity"). Se vendorizan
  los pesos `400`/`600` (`inter-latin-*-normal.woff2`,
  `@fontsource/inter@5.2.5`) siguiendo exactamente el mismo patrón
  self-hosted de §3.3 (sin CDN en runtime). La variable CSS pasa a llamarse
  `--font-sans`. Los archivos de JetBrains Mono se eliminan del repo; la
  lista cerrada de `serveFont` se actualiza a los dos nuevos nombres.
  Consecuencia aceptada: el contenido técnico (JSON/queries) se muestra en
  sans-serif; se prioriza la sencillez pedida sobre el aire "terminal".
- **D-U7. Contenedor centrado `#chat-shell` (chat acotado en pantalla
  completa).** `layout.html` envuelve tab-bar y panel en
  `<main id="chat-shell">`: `max-width: 52rem; margin: 0 auto; height:
  100vh;` con bordes laterales `--color-border-subtle`. En pantallas
  pequeñas ocupa el 100% del ancho (no requiere media queries). Se corrige
  además la cadena de alturas flex (`#chat-panel-container { flex: 1;
  min-height: 0; display: flex; flex-direction: column; }`) que hacía que,
  en una pestaña vacía, el formulario no se anclara al fondo.
- **D-U8. "Cerrar pestaña" como acción secundaria discreta.**
  `#chat-panel > .btn-secondary { align-self: flex-end; margin-top:
  var(--space-2); }` — deja de ocupar todo el ancho del panel (defecto
  visible en la captura de la pestaña vacía).
- **D-U9. Configuración del bucle agéntico documentada en `.env.example`.**
  El fallo "tool loop exceeded" observado en uso real se mitiga por
  configuración (sin cambio de código): `.env.example` documenta que
  `AGENT_MAX_TOOL_ITERATIONS` puede subirse (p. ej. 10) cuando las consultas
  requieren muchas iteraciones de tool-calling, y que
  `AGENT_REQUEST_TIMEOUT_SECONDS` debe dar margen acorde.

## 11. Tablas de datos dentro de mensajes (`.md-table`)

El agente presenta resultados tabulares como tablas Markdown (ver D-W13 de
`design.md`). Las tablas renderizadas dentro de las burbujas usan:

```css
.md-table { border-collapse: collapse; margin: var(--space-2) 0;
  font-size: var(--font-size-sm); line-height: var(--line-height-tight);
  max-width: 100%; }
.md-table th, .md-table td { border: 1px solid var(--color-border-subtle);
  padding: var(--space-2) var(--space-3); text-align: left;
  white-space: normal; word-break: break-word; }
.md-table th { background: var(--color-surface);
  color: var(--color-text-secondary); font-weight: 600; }
.md-table tbody tr:nth-child(even) { background: rgba(255,255,255,0.03); }
```

Decisiones: tamaño `--font-size-sm` (tabla = contenido denso, patrón de
dev-tool); cabecera diferenciada por fondo `--color-surface` y color
secundario (no por negrita pesada); zebra striping muy sutil para
legibilidad de filas; `word-break: break-word` porque las celdas pueden
contener JSON compacto largo (ver D-W12 de `design.md`). Todo el contenido
de celdas se interpola con el auto-escape de `html/template` — el parser de
tablas nunca genera HTML por sí mismo.
