#!/bin/sh
set -eu
# Git Bash on Windows rewrites the migrate -path=/migrations argument into a host
# path (#25); exclude only that argument so native tools still get /tmp and /dev/null.
export MSYS2_ARG_CONV_EXCL="${MSYS2_ARG_CONV_EXCL:+$MSYS2_ARG_CONV_EXCL;}-path="
cd "$(dirname "$0")/.."
export COMPOSE_PROJECT_NAME="grace-spec0017-test-$$"
export POSTGRES_PORT=0
export API_PORT=0
export WEB_PORT=15173
cleanup() { docker compose down -v --remove-orphans >/dev/null 2>&1 || true; }
trap cleanup EXIT INT TERM
docker compose up -d postgres
until docker compose exec -T postgres pg_isready -U grace -d grace >/dev/null 2>&1; do sleep 1; done
database_url='postgres://grace:grace@postgres:5432/grace?sslmode=disable'
docker compose run --rm migrate -path=/migrations -database="$database_url" up
# Empty import schema can be rolled back and restored.
docker compose run --rm migrate -path=/migrations -database="$database_url" goto 8
docker compose run --rm migrate -path=/migrations -database="$database_url" up
postgres_port="$(docker compose port postgres 5432 | sed -n '1s/.*://p')"
TEST_DATABASE_URL="postgres://grace:grace@localhost:${postgres_port}/grace?sslmode=disable" \
  go test -tags=integration ./internal/imports -count=1
# Production 2026-10-14: migration 22 must apply over published import rows,
# which a trigger keeps immutable.
published="$(docker compose exec -T postgres psql -U grace -d grace -Atc 'SELECT count(*) FROM import_records WHERE published')"
test "$published" -gt 0
questions_sql='SELECT md5(string_agg(id::text || ":" || questions, "," ORDER BY id)) FROM import_records WHERE NOT published'
questions_sql="$(printf '%s' "$questions_sql" | tr '"' "'")"
counted_on_upload="$(docker compose exec -T postgres psql -U grace -d grace -Atc "$questions_sql")"
test -n "$counted_on_upload"
docker compose run --rm migrate -path=/migrations -database="$database_url" goto 21
docker compose run --rm migrate -path=/migrations -database="$database_url" up
# The backfill counts exactly what Go counted on upload.
test "$(docker compose exec -T postgres psql -U grace -d grace -Atc "$questions_sql")" = "$counted_on_upload"
if docker compose run --rm migrate -path=/migrations -database="$database_url" goto 8; then
  echo "rollback destroyed published import provenance" >&2; exit 1
fi
echo "SPEC-0017 import integration gates passed"
