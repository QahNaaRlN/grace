.PHONY: dev down logs test test-go test-web test-jev-eval test-integration build generate-api

dev:
	docker compose up --build

down:
	docker compose down

logs:
	docker compose logs -f

generate-api:
	cd web && npm run generate:api

test: test-go test-web test-jev-eval

test-go:
	go test ./cmd/... ./internal/...

test-web:
	cd web && npm run lint && npm run build

test-jev-eval:
	./scripts/test-jev-evaluation.sh

test-integration:
	./scripts/test-financial-events.sh
	./scripts/test-receipts.sh
	./scripts/test-obligations.sh
	./scripts/test-analytics.sh
	./scripts/test-assistant-tools.sh
	./scripts/test-income-relative-periods.sh
	./scripts/test-assistant-router.sh

build:
	go build -o bin/grace-api ./cmd/api
	cd web && npm run build
