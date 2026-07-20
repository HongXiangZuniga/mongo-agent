run:
	docker-compose up -d
	go mod download
	go run cmd/server/api.go

install:
	go mod download

build:
	go build -o api ./cmd/server

unit-test:
	go test ./test/...

integration-test:
	go test -tags=integration ./test/integration/...

docker-build:
	docker build \
	-f build/docker/Dockerfile \
	-t agente-inaricards:local .

docker-run:
	docker-compose up -d
	docker run --rm -it -p 8080:8080 \
	--env-file ./.env \
	agente-inaricards:local
