.PHONY: up down logs migrate migrate-test test test-unit test-integration run cover build

# ?= にしてあるので、環境変数で上書きできる
DSN_DEV  ?= postgres://dev:dev@localhost:5432/url_shortener?sslmode=disable
DSN_TEST ?= postgres://test:test@localhost:5433/url_shortener_test?sslmode=disable

up:
	docker compose up -d

down:
	docker compose down

logs:
	docker compose logs -f

migrate:
	goose -dir migrations postgres "$(DSN_DEV)" up

migrate-test:
	goose -dir migrations postgres "$(DSN_TEST)" up

run:
	set -a; . ./.env; set +a; go run .

test:
	TEST_DATABASE_URL="$(DSN_TEST)" go test ./... -v

test-unit:
	go test ./service/... ./handler/... -v

test-integration:
	TEST_DATABASE_URL="$(DSN_TEST)" go test ./repository/... -v

cover:
	TEST_DATABASE_URL="$(DSN_TEST)" go test ./... -coverprofile=cover.out
	go tool cover -html=cover.out

build:
	go build -o bin/server .
