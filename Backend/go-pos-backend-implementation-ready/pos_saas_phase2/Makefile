run:
	go run ./cmd/api

test:
	go test ./...

race:
	go test -race ./...

vet:
	go vet ./...

fmt:
	gofmt -w .

up:
	docker compose up -d

down:
	docker compose down

migrate:
	@echo "Install goose and run migrations against DATABASE_URL"
