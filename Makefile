# Makefile (root)

APP_GATEWAY := ./cmd/gateway
APP_CATALOG := ./cmd/catalog
DC := docker compose -f deploy/docker-compose.yml

.PHONY: dev dev-down dev-logs ps \
        run-gateway run-catalog \
        test fmt tidy \
        proto proto-tools \
        sqlc migrate-catalog

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

PROTO_DIR := api/proto
MODULE := github.com/dwikikusuma/shoping-llm

proto-tools:
	@command -v protoc >/dev/null 2>&1 || (echo "protoc is not installed" && exit 1)
	@command -v protoc-gen-go >/dev/null 2>&1 || (echo "missing protoc-gen-go: go install google.golang.org/protobuf/cmd/protoc-gen-go@latest" && exit 1)
	@command -v protoc-gen-go-grpc >/dev/null 2>&1 || (echo "missing protoc-gen-go-grpc: go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest" && exit 1)


proto: proto-tools
	protoc -I $(PROTO_DIR) \
		--go_out=. --go_opt=module=$(MODULE) --go_opt=paths=import \
		--go-grpc_out=. --go-grpc_opt=module=$(MODULE) --go-grpc_opt=paths=import \
		$$(find $(PROTO_DIR) -name "*.proto")

run-gateway-dev:
	APP_ENV=dev LOG_LEVEL=debug HTTP_PORT=8080 go run ./cmd/gateway

run-catalog-dev:
	APP_ENV=dev LOG_LEVEL=debug GRPC_PORT=8081 go run ./cmd/catalog



sqlc:
	@echo "🤖 Generating SQLC code..."
	@sqlc generate
	@echo "✅ SQLC Generation Complete!"

migrate-catalog:
	$(DC) exec -T postgres psql -U shopping -d shopping_db < internal/catalog/infra/postgres/migrations/001_init.sql
	$(DC) exec -T postgres psql -U shopping -d shopping_db < internal/cart/infra/postgres/migrations/001_create_cart.up.sql

migrate-order:
	$(DC) exec -T postgres psql -U shopping -d shopping_db < internal/order/infra/postgres/migrations/001_create_order_table.up.sql
	$(DC) exec -T postgres psql -U shopping -d shopping_db < internal/order/infra/postgres/migrations/002_create_order_item_table.up.sql