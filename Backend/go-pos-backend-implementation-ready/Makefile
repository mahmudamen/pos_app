APP=pos-api
GOOSE=go run github.com/pressly/goose/v3/cmd/goose@v3.21.1

.PHONY: db-up db-down migrate migrate-down run build test race vet fmt lint security check docker-dev docker-prod docker-ubuntu

db-up:
	docker compose -f deployments/docker/docker-compose.yml up -d postgres redis

db-down:
	docker compose -f deployments/docker/docker-compose.yml down

docker-dev: ## Full containerized dev stack (postgres + redis + migrate + api)
	./scripts/docker-dev.sh

docker-prod: ## Production stack (postgres + redis + migrate + api + caddy)
	./scripts/docker-prod.sh

docker-ubuntu: ## Install Docker + Compose v2 on an Ubuntu/Debian host (requires sudo)
	sudo ./scripts/setup_ubuntu.sh

migrate:
	$(GOOSE) -dir internal/infrastructure/database/migrations postgres "$$DATABASE_URL" up

migrate-down:
	$(GOOSE) -dir internal/infrastructure/database/migrations postgres "$$DATABASE_URL" down

run:
	go run ./cmd/api

build:
	mkdir -p bin
	go build -trimpath -ldflags="-s -w" -o bin/$(APP) ./cmd/api

test:
	go test ./...

race:
	go test -race ./...

vet:
	go vet ./...

fmt:
	gofmt -w $$(find . -type f -name '*.go' -not -path './vendor/*')

lint:
	golangci-lint run ./...

security:
	gosec ./...

check: fmt vet test
