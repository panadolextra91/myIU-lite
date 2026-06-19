DATABASE_URL ?= postgres://myiu:myiu@localhost:5432/myiu_dev?sslmode=disable

migrate:
	migrate -path db/migrations -database "$(DATABASE_URL)" up

migrate-down:
	migrate -path db/migrations -database "$(DATABASE_URL)" down 1

sqlc:
	sqlc generate

run:
	go run ./cmd/api

test:
	go test ./...
