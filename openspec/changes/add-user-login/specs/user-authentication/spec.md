## ADDED Requirements

### Requirement: Login Web con Usuario y Contraseña
El sistema SHALL autenticar el acceso a la interfaz web mediante un formulario de usuario y contraseña, validando las credenciales contra una base de datos MongoDB de usuarios y verificando la contraseña con un hash bcrypt, sin ofrecer registro, gestión ni recuperación de usuarios.

#### Scenario: Login exitoso con credenciales válidas
- **GIVEN** existe en la base de usuarios un usuario cuyo hash de contraseña corresponde a la contraseña enviada
- **WHEN** el usuario envía `POST /web/login` con los campos `username` y `password` correctos
- **THEN** el sistema crea una sesión web con un identificador opaco aleatorio y la persiste en el almacén de sesiones del lado servidor con un TTL
- **AND** el sistema fija una cookie HttpOnly con `SameSite=Strict` cuyo valor es ese identificador de sesión opaco (nunca `API_TOKEN` ni la contraseña)
- **AND** el sistema responde con una redirección `303 See Other` hacia `GET /web`

#### Scenario: Login con contraseña incorrecta
- **GIVEN** existe un usuario con ese `username` en la base de usuarios
- **WHEN** el usuario envía `POST /web/login` con el `username` correcto pero una `password` que no coincide con el hash almacenado
- **THEN** el sistema responde `401 Unauthorized` renderizando de nuevo el formulario de login con un mensaje de error genérico
- **AND** el sistema no fija ninguna cookie de sesión

#### Scenario: Login con usuario inexistente
- **GIVEN** no existe ningún usuario con ese `username` en la base de usuarios
- **WHEN** el usuario envía `POST /web/login` con ese `username` y cualquier `password`
- **THEN** el sistema responde `401 Unauthorized` con el mismo mensaje de error genérico que en el caso de contraseña incorrecta
- **AND** el mensaje de error no revela si el fallo fue por usuario inexistente o por contraseña incorrecta
- **AND** el sistema no fija ninguna cookie de sesión

#### Scenario: Formulario de login con dos campos
- **WHEN** el usuario solicita `GET /web/login`
- **THEN** el sistema renderiza un formulario con un campo de texto `username` y un campo de contraseña `password`, ambos obligatorios

### Requirement: Contraseña Almacenada como Hash
El sistema SHALL almacenar la contraseña de cada usuario únicamente como un hash bcrypt en la base de datos, y SHALL verificar las credenciales comparando la contraseña recibida contra ese hash, sin almacenar ni registrar nunca la contraseña en texto plano.

#### Scenario: El documento de usuario no contiene la contraseña en claro
- **GIVEN** un usuario almacenado en la base de usuarios
- **WHEN** se inspecciona su documento en MongoDB
- **THEN** contiene un campo con el hash bcrypt de la contraseña (por ejemplo `password_hash`)
- **AND** no contiene ningún campo con la contraseña en texto plano

#### Scenario: La verificación usa comparación bcrypt
- **GIVEN** una contraseña candidata y el hash bcrypt almacenado del usuario
- **WHEN** el sistema valida las credenciales
- **THEN** el resultado de "válida" o "inválida" se determina mediante una comparación bcrypt del texto contra el hash, no mediante comparación directa de cadenas en claro

### Requirement: Base de Usuarios Separada del MongoDB de Solo Lectura del Agente
El sistema SHALL usar una conexión MongoDB dedicada e independiente para la base de usuarios del login, configurada con variables de entorno propias (`AUTH_MONGODB_URI`, `AUTH_MONGODB_DB_NAME`), distinta de la conexión del agente (`MONGODB_URI`), y el adapter de usuarios NO SHALL formar parte del paquete cuyo invariante de solo lectura se verifica con `mongodb.VerifyReadOnlyGuarantee`.

#### Scenario: Dos conexiones MongoDB independientes
- **GIVEN** el proceso del servidor en arranque
- **WHEN** se inicializan las conexiones a MongoDB
- **THEN** existe una conexión al cluster del agente construida a partir de `MONGODB_URI`
- **AND** existe una conexión separada a la base de usuarios construida a partir de `AUTH_MONGODB_URI`
- **AND** la verificación `mongodb.VerifyReadOnlyGuarantee` se aplica únicamente a la conexión del agente, nunca a la base de usuarios

#### Scenario: Falta de configuración de la base de usuarios impide el arranque
- **GIVEN** la variable de entorno `AUTH_MONGODB_URI` o `AUTH_MONGODB_DB_NAME` no está definida o está vacía
- **WHEN** el proceso del servidor arranca
- **THEN** el sistema registra un error crítico indicando el nombre de la variable faltante y termina el proceso sin abrir el puerto HTTP

### Requirement: Usuario de Prueba Precargado
El sistema SHALL precargar un usuario de prueba en la base de usuarios al inicializar el contenedor MongoDB de autenticación, mediante un script de inicialización, de modo que el login funcione tras `docker-compose up` sin pasos manuales adicionales.

#### Scenario: El usuario de prueba existe tras levantar el compose
- **GIVEN** el contenedor MongoDB de autenticación se inicializa por primera vez con su script de seed montado
- **WHEN** el contenedor termina su arranque
- **THEN** la base de usuarios contiene un usuario de prueba con su hash bcrypt precargado
- **AND** ese usuario permite iniciar sesión en `/web` con las credenciales documentadas en `README.md`

#### Scenario: El seed almacena la contraseña hasheada
- **GIVEN** el script de seed que precarga el usuario de prueba
- **WHEN** se inspecciona el documento insertado
- **THEN** la contraseña del usuario de prueba se guarda como hash bcrypt, no en texto plano
- **AND** la contraseña en texto plano solo aparece en la documentación de prueba, no en la base de datos

### Requirement: La API REST Conserva la Autenticación por Token
El sistema SHALL mantener sin cambios la autenticación de `POST /ask` mediante el header `Authorization` comparado contra `API_TOKEN`, sin exigir credenciales de usuario en la API REST.

#### Scenario: POST /ask sigue autenticándose por token
- **GIVEN** el login de usuario/contraseña está habilitado para la interfaz web
- **WHEN** un cliente invoca `POST /ask` con el header `Authorization` igual a `API_TOKEN`
- **THEN** el sistema procesa la petición igual que antes de este cambio
- **AND** el sistema no exige ningún `username`/`password` para la API REST

### Requirement: Sesión Web Respaldada en Redis con TTL
El sistema SHALL respaldar cada sesión web autenticada en un almacén de sesiones del lado servidor (Redis) mediante un identificador opaco aleatorio asociado al menos al `username` autenticado y con un TTL, en un espacio de claves distinto del de la memoria de conversación del agente, y la cookie de sesión web SHALL transportar únicamente ese identificador opaco, nunca `API_TOKEN` ni la contraseña.

#### Scenario: El login crea la sesión en el almacén de servidor
- **GIVEN** un login válido de usuario/contraseña
- **WHEN** el sistema emite la cookie de sesión web
- **THEN** existe en Redis un registro de sesión asociado al identificador de la cookie que contiene el `username` autenticado
- **AND** ese registro tiene un TTL configurado, tras el cual la sesión deja de ser válida

#### Scenario: El identificador de sesión es opaco y no reutiliza el API_TOKEN
- **GIVEN** dos logins válidos distintos
- **WHEN** el sistema emite la cookie de cada uno
- **THEN** el valor de la cookie es un identificador opaco aleatorio distinto en cada login
- **AND** el valor de la cookie no es igual a `API_TOKEN` ni a la contraseña del usuario

### Requirement: Validación de Sesión Web en Cada Petición
El sistema SHALL validar, en cada petición a una ruta web protegida, el identificador de sesión de la cookie contra el almacén de sesiones del lado servidor, permitiendo el acceso solo si la sesión existe y no ha expirado, y SHALL redirigir al login en caso contrario, adaptando la respuesta a peticiones htmx.

#### Scenario: Sesión válida permite el acceso
- **GIVEN** una cookie cuyo identificador corresponde a una sesión existente y no expirada en el almacén
- **WHEN** el usuario solicita una ruta web protegida (por ejemplo `GET /web`)
- **THEN** el sistema procesa la petición normalmente

#### Scenario: Sesión inexistente o expirada redirige al login
- **GIVEN** una cookie cuyo identificador no existe en el almacén o cuya sesión ha expirado
- **WHEN** el usuario solicita una ruta web protegida
- **THEN** el sistema redirige a `/web/login` con `303 See Other` para una petición normal
- **AND** el sistema responde con la cabecera `HX-Redirect: /web/login` cuando la petición trae `HX-Request: true`

#### Scenario: Una cookie con el valor de API_TOKEN es rechazada
- **GIVEN** una cookie de sesión web cuyo valor es `API_TOKEN` (u otro valor no registrado como sesión en el almacén)
- **WHEN** el usuario solicita una ruta web protegida
- **THEN** el sistema no concede acceso y redirige al login
- **AND** conocer `API_TOKEN` no basta para acceder a `/web` sin un login válido

### Requirement: Cierre de Sesión Invalida la Sesión en el Servidor
El sistema SHALL, al hacer logout, eliminar la sesión del almacén del lado servidor además de expirar la cookie del navegador, de modo que el identificador deje de ser válido en peticiones posteriores.

#### Scenario: Logout borra la sesión del almacén
- **GIVEN** una sesión web activa con su cookie e identificador registrados en el almacén
- **WHEN** el usuario envía `POST /web/logout`
- **THEN** el sistema elimina del almacén el registro de sesión asociado a ese identificador
- **AND** el sistema expira la cookie del navegador y redirige a `/web/login`

#### Scenario: Reusar la cookie tras el logout no concede acceso
- **GIVEN** un usuario que hizo logout pero conserva el valor de la cookie anterior
- **WHEN** vuelve a enviar esa cookie a una ruta web protegida
- **THEN** el sistema redirige al login porque la sesión ya no existe en el almacén

### Requirement: Arranque Integral con docker-compose
El sistema SHALL declarar en `docker-compose.yml` los servicios necesarios para que un único `docker-compose up` levante Redis, la base de usuarios (ya seedeada) y la API Go enlazadas, de forma que la API espere a que sus dependencias estén saludables antes de aceptar tráfico.

#### Scenario: docker-compose up levanta toda la topología
- **GIVEN** un archivo `.env` con los secretos externos requeridos (`API_TOKEN`, `MONGODB_URI` de Atlas, `MONGODB_DB_NAME`, `OPENCODE_API_KEY`)
- **WHEN** se ejecuta `docker-compose up`
- **THEN** se levantan los servicios de Redis, base de usuarios MongoDB y API Go
- **AND** el servicio de la API se construye a partir de `build/docker/Dockerfile`
- **AND** el servicio de la API arranca solo después de que Redis y la base de usuarios reporten estado saludable

#### Scenario: La API alcanza sus dependencias por nombre de servicio
- **GIVEN** los servicios levantados por docker-compose en la misma red
- **WHEN** la API se conecta a Redis y a la base de usuarios
- **THEN** la API usa los nombres de servicio de la red de compose como host (no `localhost`) para Redis y para la base de usuarios
