.DEFAULT_GOAL := build

fmt:
	 go fmt ./...
.PHONY: fmt

vet: fmt
	go vet ./...
.PHONY: vet

cilint:
	golangci-lint run
.PHONY: cilint

swagger:
	swag init
.PHONY: swagger

serve-docs:
	swag init && go run main.go
.PHONY: serve-docs

build:
	go build -o klokku
.PHONY: build

run: go run main.go
.PHONY: run