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

build:
	go build -o klokku
.PHONY: build

run: go run main.go
.PHONY: run
