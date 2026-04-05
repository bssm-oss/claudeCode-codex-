APP_NAME := ccagent

.PHONY: fmt test build run doctor

fmt:
	go fmt ./...

test:
	go test ./...

build:
	go build ./...

run:
	go run ./cmd/ccagent chat

doctor:
	go run ./cmd/ccagent doctor
