#!/bin/sh
set -eu
# Git Bash on Windows rewrites the migrate -path=/migrations argument into a host
# path (#25); exclude only that argument so native tools still get /tmp and /dev/null.
export MSYS2_ARG_CONV_EXCL="${MSYS2_ARG_CONV_EXCL:+$MSYS2_ARG_CONV_EXCL;}-path="
cd "$(dirname "$0")/.."
export COMPOSE_PROJECT_NAME="grace-ux-test-$$"
export POSTGRES_PORT=0 API_PORT=0 WEB_PORT=15173
cleanup(){ docker compose down -v --remove-orphans >/dev/null 2>&1 || true; }
trap cleanup EXIT INT TERM
docker compose up -d postgres
until docker compose exec -T postgres pg_isready -U grace -d grace >/dev/null 2>&1; do sleep 1; done
docker compose run --rm migrate -path=/migrations -database='postgres://grace:grace@postgres:5432/grace?sslmode=disable' up
postgres_port="$(docker compose port postgres 5432 | sed -n '1s/.*://p')"
export TEST_DATABASE_URL="postgres://grace:grace@localhost:${postgres_port}/grace?sslmode=disable"
go test -p 1 -tags=integration ./internal/accounts ./internal/categories ./internal/financialevents ./internal/auth ./internal/assistant ./internal/receiptdrafts -count=1
# Test the credential rollback guard itself despite later consolidated migrations.
if docker compose exec -T postgres psql -U grace -d grace -v ON_ERROR_STOP=1 < migrations/000012_password_credentials.down.sql; then
 echo 'Unsafe password credential deletion was accepted' >&2; exit 1
fi
if docker compose exec -T postgres psql -U grace -d grace -v ON_ERROR_STOP=1 < migrations/000017_personal_categories.down.sql; then
 echo 'Personal category ownership was discarded' >&2; exit 1
fi
echo 'SPEC-0033 consolidated integration checks passed'
