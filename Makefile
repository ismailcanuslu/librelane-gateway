.PHONY: build run dev docker-build docker-run tidy lint

BINARY   := gateway
CMD_PATH := ./cmd/gateway
IMAGE    := ayws-gateway:latest

## Build Go binary
build:
	go build -ldflags="-s -w" -o bin/$(BINARY) $(CMD_PATH)

## Run in development mode (with live config reload)
run:
	go run $(CMD_PATH)/main.go

## Download and tidy dependencies
tidy:
	go mod tidy

## Build Docker image
docker-build:
	docker build -t $(IMAGE) .

## Run Docker container
docker-run:
	docker run --rm -p 8000:8000 --env-file .env $(IMAGE)

## Run tests
test:
	go test ./... -v -race

## Format + vet
lint:
	gofmt -w .
	go vet ./...
