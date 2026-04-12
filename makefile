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

build-cli:
	go build -o klokku-cli ./cmd/klokku-cli
.PHONY: build-cli

build-all: build build-cli
.PHONY: build-all

run: go run main.go
.PHONY: run