# Multi-stage build for the Go gateway.
# Build context is the REPO ROOT (so ../proto is available):
#   docker build -f gateway-go/Dockerfile -t trading/gateway .
FROM golang:1.22-bookworm AS build

# protoc for the codegen step; the Go plugins are installed via `make tools`.
RUN apt-get update \
 && apt-get install -y --no-install-recommends protobuf-compiler \
 && rm -rf /var/lib/apt/lists/*

WORKDIR /src
COPY proto/ proto/
COPY gateway-go/ gateway-go/

WORKDIR /src/gateway-go
# golang image already has /go/bin (where make tools installs) on PATH.
RUN make tools \
 && make proto \
 && go mod tidy \
 && CGO_ENABLED=0 go build -ldflags="-s -w" -o /out/gateway ./cmd/gateway

# Tiny, non-root final image. Static binary → distroless static is enough.
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/gateway /gateway
EXPOSE 8090
USER nonroot:nonroot
ENTRYPOINT ["/gateway"]
