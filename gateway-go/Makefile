# Generates Go stubs from the shared contract at ../proto/trading.proto.
#
# The proto file is the single source of truth and stays free of any
# consumer-specific options: the Go import path is injected here via the
# module-mapping flag (M...) instead of a go_package option in the .proto,
# so C++ and Java codegen are unaffected.

MODULE      := github.com/shubham/trading-platform/gateway-go
PROTO_DIR   := ../proto
PROTO_FILE  := trading.proto
OUT_DIR     := internal/pb
PKG_PATH    := $(MODULE)/$(OUT_DIR)

.PHONY: proto tools test vet build run

proto:
	protoc \
		--proto_path=$(PROTO_DIR) \
		--go_out=$(OUT_DIR) --go_opt=paths=source_relative \
		--go_opt=M$(PROTO_FILE)=$(PKG_PATH) \
		--go-grpc_out=$(OUT_DIR) --go-grpc_opt=paths=source_relative \
		--go-grpc_opt=M$(PROTO_FILE)=$(PKG_PATH) \
		$(PROTO_FILE)

# Installs the protoc Go plugins (protoc itself comes from the OS package).
tools:
	go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.34.2
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.5.1

vet:
	go vet ./...

test:
	go test -race ./...

build:
	go build -o bin/gateway ./cmd/gateway

run:
	go run ./cmd/gateway
