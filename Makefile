.PHONY: build up-cluster down logs run psql redis-cli reset migrate-up migrate-down

DB_URL ?= postgres://clustercron:clustercron@localhost:5432/clustercron?sslmode=disable

build:
	docker compose build

up-cluster:
	docker compose up -d --build

down:
	docker compose down

logs:
	docker compose logs -f

logs-node-%:
	docker compose logs -f node-$*

reset:
	docker compose down -v && docker compose up -d --build

run:
	set -a; . ./.env; set +a; go run ./cmd/clustercron

psql:
	docker exec -it clustercron-postgres psql -U clustercron -d clustercron

redis-cli:
	docker exec -it clustercron-redis redis-cli

migrate-up:
	goose -dir migrations postgres "$(DB_URL)" up

migrate-down:
	goose -dir migrations postgres "$(DB_URL)" down