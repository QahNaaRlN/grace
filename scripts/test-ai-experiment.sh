#!/bin/sh
set -eu
cd "$(dirname "$0")/.."
export COMPOSE_PROJECT_NAME="grace-spec0026-test-$$"
export POSTGRES_PORT=0 API_PORT=0 WEB_PORT=0
cleanup() { docker compose down -v --remove-orphans >/dev/null 2>&1 || true; }
trap cleanup EXIT INT TERM
cleanup
docker compose up -d postgres
until docker compose exec -T postgres pg_isready -U grace -d grace >/dev/null 2>&1; do sleep 1; done
database_url='postgres://grace:grace@postgres:5432/grace?sslmode=disable'
docker compose run --rm migrate -path=/migrations -database="$database_url" up
postgres_port="$(docker compose port postgres 5432 | sed -n '1s/.*://p')"
test -n "$postgres_port"
TEST_DATABASE_URL="postgres://grace:grace@localhost:${postgres_port}/grace?sslmode=disable" \
 go test -tags=integration ./internal/assistant ./internal/receiptdrafts -count=1
if docker compose run --rm migrate -path=/migrations -database="$database_url" goto 10; then
 echo 'AI usage evidence was discarded by downgrade' >&2; exit 1
fi
echo 'SPEC-0026 usage integration checks passed'
