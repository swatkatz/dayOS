include .env
export

.PHONY: dev dev-backend dev-frontend generate generate-validation migrate build lint

dev:
	$(MAKE) dev-backend & $(MAKE) dev-frontend & wait

dev-backend:
	cd backend && go run .

dev-frontend:
	cd frontend && npm run dev

generate: generate-validation
	cd backend && go run github.com/99designs/gqlgen generate
	cd backend && sqlc generate

generate-validation:
	cd backend && go run ./cmd/genvalidation

migrate:
	cd backend && migrate -path db/migrations -database "$$DATABASE_URL" up

build:
	cd frontend && npm run build
	cd backend && go build -o dayos .

lint:
	cd backend && golangci-lint run ./...
