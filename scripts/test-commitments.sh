#!/bin/sh
set -eu
cd "$(dirname "$0")/.."
export COMPOSE_PROJECT_NAME="grace-commitments-test-$$"
export POSTGRES_PORT=0 API_PORT=0 WEB_PORT=0
cleanup(){ docker compose down -v --remove-orphans >/dev/null 2>&1 || true; }
trap cleanup EXIT INT TERM
docker compose up -d postgres
until docker compose exec -T postgres pg_isready -U grace -d grace >/dev/null 2>&1; do sleep 1; done
docker compose run --rm migrate -path=/migrations -database='postgres://grace:grace@postgres:5432/grace?sslmode=disable' up
postgres_port="$(docker compose port postgres 5432 | sed -n '1s/.*://p')"
export TEST_DATABASE_URL="postgres://grace:grace@localhost:${postgres_port}/grace?sslmode=disable"
go test -p 1 -tags=integration ./internal/commitments -count=1
if docker compose exec -T postgres psql -U grace -d grace -v ON_ERROR_STOP=1 < migrations/000018_commitments.down.sql; then
 echo 'Commitment history was discarded by downgrade' >&2; exit 1
fi
echo 'SPEC-0034 commitment integration checks passed'
