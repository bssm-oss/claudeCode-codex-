APP_NAME := ccagent

.PHONY: fmt test build run doctor lint

fmt:
	go fmt ./...

test:
	go test ./...

lint:
	golangci-lint run ./...

build:
	go build ./...

run:
	go run ./cmd/ccagent chat

doctor:
	go run ./cmd/ccagent doctor
