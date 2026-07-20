module github.com/HongXiangZuniga/agente-inaricards

go 1.22

// NOTE: este go.mod es un scaffold inicial. Las dependencias reales se
// resuelven ejecutando `go mod tidy` como parte de la tarea 9.1 de
// openspec/changes/add-nl-mongo-agent/tasks.md, una vez que el código de
// las siguientes tareas importe cada paquete.
//
// Dependencias esperadas (ver design.md de la change add-nl-mongo-agent):
//   github.com/gin-gonic/gin           -> pkg/http/rest
//   go.mongodb.org/mongo-driver        -> pkg/persistence/mongodb, cmd/server
//   github.com/redis/go-redis/v9       -> pkg/persistence/redis, cmd/server
//   github.com/joho/godotenv           -> cmd/server
//   github.com/google/uuid             -> pkg/http/rest
