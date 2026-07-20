# La version de Go del builder coincide con la directiva `go` de go.mod
# (ver tarea 9.4 de openspec/changes/add-nl-mongo-agent/tasks.md).
FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY . .

RUN go mod download && go build -o /app/api ./cmd/server

FROM alpine:3.19
RUN addgroup -S appgroup && adduser -S appuser -G appgroup
WORKDIR /app
COPY --from=builder /app/api ./api
USER appuser
EXPOSE 8080
ENTRYPOINT ["./api"]
