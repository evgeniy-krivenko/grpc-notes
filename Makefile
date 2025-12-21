include .env

LOCAL_BIN:=$(CURDIR)/bin
PATH  := $(PATH):$(PWD)/bin

GOOSE_DBSTRING := "host=$(DB_HOST) user=$(DB_USER) dbname=$(DB_NAME) password=$(DB_PASSWORD) sslmode=disable" 

.PHONY: install-deps
install-deps:
	$(info Installing binary dependencies...)
	mkdir -p $(LOCAL_BIN)
	GOBIN=$(LOCAL_BIN) go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.31.0 && \
	GOBIN=$(LOCAL_BIN) go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.3.0 && \
	GOBIN=$(LOCAL_BIN) go install github.com/easyp-tech/easyp/cmd/easyp@v0.7.15 && \
	GOBIN=$(LOCAL_BIN) go install github.com/pressly/goose/v3/cmd/goose@latest && \
	GOBIN=$(LOCAL_BIN) go install github.com/kazhuravlev/options-gen/cmd/options-gen@v0.55.3

.PHONY: generate
generate:
	@$(LOCAL_BIN)/easyp generate

.PHONY: generate-options
generate-options:
	@go generate ./...

.PHONY: lint
lint:
	@$(LOCAL_BIN)/easyp lint --path api

.PHONY: breaking
breaking:
	@$(LOCAL_BIN)/easyp breaking --against main --path api

.PHONY: mod-download
mod-download:
	@$(LOCAL_BIN)/easyp mod download

.PHONY: run
run:
	go run ./cmd/server/

.PHONY: migrate-up
migrate-up:
	@$(LOCAL_BIN)/goose -dir migrate/migrations postgres $(GOOSE_DBSTRING) up

.PHONY: migrate-status
migrate-status:
	@$(LOCAL_BIN)/goose -dir migrate/migrations postgres $(GOOSE_DBSTRING) status

.PHONY: migrate-reset
migrate-reset:
	@$(LOCAL_BIN)/goose -dir migrate/migrations postgres $(GOOSE_DBSTRING) reset

.PHONY: migrate
migrate: migrate-up
