# Makefile (root)

APP_GATEWAY := ./cmd/gateway
APP_CATALOG := ./cmd/catalog
DC := docker compose -f deploy/docker-compose.yml

.PHONY: dev dev-down dev-logs ps \
        run-gateway run-catalog \
        test fmt tidy \
        proto proto-tools

dev:
	$(DC) up -d

dev-down:
	$(DC) down -v

dev-logs:
	$(DC) logs -f --tail=200

ps:
	$(DC) ps

run-gateway:
	go run $(APP_GATEWAY)

run-catalog:
	go run $(APP_CATALOG)

test:
	go test ./...

fmt:
	gofmt -w .

tidy:
	go mod tidy

proto-tools:
	@command -v protoc >/dev/null 2>&1 || (echo "protoc is not installed" && exit 1)
	@command -v protoc-gen-go >/dev/null 2>&1 || (echo "protoc-gen-go is not installed. Run: go install google.golang.org/protobuf/cmd/protoc-gen-go@latest" && exit 1)
	@command -v protoc-gen-go-grpc >/dev/null 2>&1 || (echo "protoc-gen-go-grpc is not installed. Run: go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest" && exit 1)

# Update PROTO_DIR/OUT_DIR to match your layout
PROTO_DIR := api/proto
OUT_DIR := .

proto: proto-tools
	protoc -I $(PROTO_DIR) \
		--go_out=$(OUT_DIR) --go_opt=paths=source_relative \
		--go-grpc_out=$(OUT_DIR) --go-grpc_opt=paths=source_relative \
		$$(find $(PROTO_DIR) -name "*.proto")

run-gateway-dev:
	APP_ENV=dev LOG_LEVEL=debug HTTP_PORT=8080 go run ./cmd/gateway

run-catalog-dev:
	APP_ENV=dev LOG_LEVEL=debug GRPC_PORT=8081 go run ./cmd/catalog
