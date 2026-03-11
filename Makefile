.PHONY: dev generate migrate build lint

dev:
	cd backend && go run ./main.go

generate:
	cd backend && go run github.com/99designs/gqlgen generate
	cd backend && sqlc generate

migrate:
	cd backend && migrate -path db/migrations -database "$$DATABASE_URL" up

build:
	cd frontend && npm run build
	cd backend && go build -o dayos ./main.go

lint:
	cd backend && golangci-lint run ./...
