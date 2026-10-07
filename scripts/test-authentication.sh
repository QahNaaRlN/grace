#!/bin/sh
set -eu
cd "$(dirname "$0")/.."

export COMPOSE_PROJECT_NAME="grace-spec0013-test-$$"
export POSTGRES_PORT=0
export API_PORT=0
export WEB_PORT=15173
cleanup() { docker compose down -v --remove-orphans >/dev/null 2>&1 || true; }
trap cleanup EXIT INT TERM

docker compose up -d postgres
until docker compose exec -T postgres pg_isready -U grace -d grace >/dev/null 2>&1; do sleep 1; done
database_url='postgres://grace:grace@postgres:5432/grace?sslmode=disable'
docker compose run --rm migrate -path=/migrations -database="$database_url" goto 5
before="$(docker compose exec -T postgres psql -U grace -d grace -Atc 'SELECT sum(opening_balance_minor) FROM accounts')"
docker compose run --rm migrate -path=/migrations -database="$database_url" goto 6
after="$(docker compose exec -T postgres psql -U grace -d grace -Atc 'SELECT sum(opening_balance_minor) FROM accounts')"
test "$before" = "$after"
# Owner-only rollback is supported and can migrate forward again.
docker compose run --rm migrate -path=/migrations -database="$database_url" goto 5
docker compose run --rm migrate -path=/migrations -database="$database_url" up
postgres_port="$(docker compose port postgres 5432 | sed -n '1s/.*://p')"
TEST_DATABASE_URL="postgres://grace:grace@localhost:${postgres_port}/grace?sslmode=disable" \
  go test -tags=integration ./internal/authtest -count=1
# The suite creates another owner's financial data; flattening must fail.
if docker compose run --rm migrate -path=/migrations -database="$database_url" goto 5; then
  echo "unsafe multi-owner downgrade was accepted" >&2; exit 1
fi
echo "SPEC-0013 authentication and ownership integration checks passed"
