GO ?= go
CGO_ENABLED ?= 1
PORT ?= 8080

.PHONY: help run test build fmt vendor

help:
	@echo "Targets disponíveis:"
	@echo "  make run    - roda a API localmente"
	@echo "  make test   - executa os testes do projeto"
	@echo "  make build  - compila a aplicação"
	@echo "  make fmt    - formata o código Go"
	@echo "  make vendor - atualiza a pasta vendor"

run:
	CGO_ENABLED=$(CGO_ENABLED) PORT=$(PORT) $(GO) run ./cmd

test:
	CGO_ENABLED=$(CGO_ENABLED) $(GO) test ./...

build:
	CGO_ENABLED=$(CGO_ENABLED) $(GO) build ./cmd

fmt:
	$(GO) fmt ./...

vendor:
	$(GO) mod vendor