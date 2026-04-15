.PHONY: up down logs reset psql redis-cli

up:
	docker compose up -d

down:
	docker compose down

logs:
	docker compose logs -f

reset:
	docker compose down -v && docker compose up -d

psql:
	docker exec -it clustercron-postgres psql -U clustercron -d clustercron

redis-cli:
	docker exec -it clustercron-redis redis-cli
run:
	set -a; . ./.env; set +a; go run ./cmd/clustercron

DB_URL ?= postgres://clustercron:clustercron@localhost:5433/clustercron?sslmode=disable

.PHONY: migrate-up migrate-down migrate-status migrate-new

migrate-up:
	goose -dir migrations postgres "$(DB_URL)" up

migrate-down:
	goose -dir migrations postgres "$(DB_URL)" down

migrate-status:
	goose -dir migrations postgres "$(DB_URL)" status

# usage: make migrate-new name=add_retries
migrate-new:
	goose -dir migrations create $(name) sql	