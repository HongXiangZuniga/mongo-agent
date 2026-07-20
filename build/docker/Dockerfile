# Scaffold inicial. Ajustar la version de Go para que coincida con go.mod
# (ver tarea 9.4 de openspec/changes/add-nl-mongo-agent/tasks.md).
FROM golang:1.22 AS builder

WORKDIR /app

COPY . .

RUN go mod download && go build -o /app/api ./cmd/server

FROM alpine:3.19

WORKDIR /app

COPY --from=builder /app/api ./api

EXPOSE 8080

ENTRYPOINT ["./api"]
