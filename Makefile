proto:
	@protoc \
		--go_out=. \
		--go_opt=paths=source_relative \
		--go-grpc_out=. \
		--go-grpc_opt=paths=source_relative \
		./frontend/grpc/keyvalue.proto
	@echo "Generated protobuff"

lint:
	@go mod tidy
	@golangci-lint run
	@echo "Linted"

gen: proto

dev:
	@go run .

build:
	@CGO_ENABLED=0 GOOS=linux go build -o main -ldflags "-s -w" .

image:
	@docker build -t kvs -f build/kvs.Dockerfile .
	@docker run --rm -p 42069:42069 -p 8008:8008 kvs

docker:
	@docker compose down
	@docker compose up --build --remove-orphans

.PHONY: proto lint gen dev build image docker
