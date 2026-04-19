.PHONY: build up-cluster down logs-scheduler logs-worker-1 logs-worker-2

build:
	docker compose build

up-cluster:
	docker compose up -d --build

down:
	docker compose down

logs:
	docker compose logs -f

logs-scheduler:
	docker compose logs -f scheduler

logs-worker-1:
	docker compose logs -f worker-1

logs-worker-2:
	docker compose logs -f worker-2	

reset:
	docker compose down -v && docker compose up -d --build

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