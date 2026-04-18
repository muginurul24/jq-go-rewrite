SHELL := /bin/zsh

.PHONY: install up down compose-migrate up-full down-full dev-web build-web lint-web dev-api dev-worker dev-scheduler build-api migrate-up migrate-baseline migrate-down migrate-status migrate-reset migrate-seed migrate-seed-demo smoke-runtime reconcile-finance preflight-staging test-api test-api-auth-integration test-api-webhooks-integration test-web-e2e-list

install:
	pnpm install --recursive

up:
	docker compose up -d

down:
	docker compose down

compose-migrate:
	docker compose -f docker-compose.yml -f docker-compose.full.yml --profile ops run --rm migrate

up-full:
	docker compose -f docker-compose.yml -f docker-compose.full.yml up -d --build

down-full:
	docker compose -f docker-compose.yml -f docker-compose.full.yml down

dev-web:
	pnpm --filter web dev

build-web:
	pnpm --filter web build

lint-web:
	pnpm --filter web lint

dev-api:
	cd apps/api && go run ./cmd/server

dev-worker:
	cd apps/api && go run ./cmd/worker

dev-scheduler:
	cd apps/api && go run ./cmd/scheduler

build-api:
	cd apps/api && go build ./...

migrate-up:
	cd apps/api && go run ./cmd/migrate up

migrate-baseline:
	cd apps/api && go run ./cmd/migrate baseline

migrate-down:
	cd apps/api && go run ./cmd/migrate down

migrate-status:
	cd apps/api && go run ./cmd/migrate status

migrate-reset:
	cd apps/api && go run ./cmd/migrate reset

migrate-seed:
	cd apps/api && go run ./cmd/migrate --seed

migrate-seed-demo:
	cd apps/api && go run ./cmd/migrate --seed --seed-profile demo

smoke-runtime:
	./scripts/smoke-runtime.sh

reconcile-finance:
	./scripts/reconcile-finance.sh

preflight-staging:
	./scripts/preflight-staging.sh

test-api:
	cd apps/api && go test ./...

test-api-auth-integration:
	cd apps/api && go test ./internal/auth -run Integration -v

test-api-webhooks-integration:
	cd apps/api && go test ./internal/modules/webhooks -run Integration -v

test-web-e2e-list:
	cd apps/web && pnpm exec playwright test --list
