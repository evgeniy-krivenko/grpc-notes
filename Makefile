LOCAL_BIN:=$(CURDIR)/bin
PATH  := $(PATH):$(PWD)/bin

.PHONY: install-deps
install-deps:
	$(info Installing binary dependencies...)
	mkdir -p $(LOCAL_BIN)
	GOBIN=$(LOCAL_BIN) go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.31.0 && \
    GOBIN=$(LOCAL_BIN) go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.3.0 && \
    GOBIN=$(LOCAL_BIN) go install github.com/easyp-tech/easyp/cmd/easyp@v0.7.15

.PHONY: generate
generate:
	@$(LOCAL_BIN)/easyp generate

.PHONY: lint
lint:
	@$(LOCAL_BIN)/easyp lint --path api

.PHONY: breaking
breaking:
	@$(LOCAL_BIN)/easyp breaking --against main --path api

.PHONY: mod-download
mod-download:
	@$(LOCAL_BIN)/easyp mod download
