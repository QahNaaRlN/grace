#!/bin/sh
set -eu

cd "$(dirname "$0")/.."

export COMPOSE_PROJECT_NAME="grace-spec0005-test-$$"
export POSTGRES_PORT=0
export API_PORT=0
export WEB_PORT=15173

cleanup() {
  docker compose down -v --remove-orphans >/dev/null 2>&1 || true
}
trap cleanup EXIT INT TERM
cleanup

docker compose up -d postgres
until docker compose exec -T postgres pg_isready -U grace -d grace >/dev/null 2>&1; do
  sleep 1
done

database_url='postgres://grace:grace@postgres:5432/grace?sslmode=disable'
docker compose run --rm migrate -path=/migrations -database="$database_url" up

postgres_port="$(docker compose port postgres 5432 | sed -n '1s/.*://p')"
test -n "$postgres_port"
TEST_DATABASE_URL="postgres://grace:grace@localhost:${postgres_port}/grace?sslmode=disable" \
  go test -tags=integration ./internal/assistanttest -run '^TestReadToolsAgainstPostgreSQL$' -count=1

if rg -n --glob '*.go' '(SELECT|INSERT|UPDATE|DELETE)[[:space:]]' internal/assistant; then
  echo "SQL found in internal/assistant" >&2
  exit 1
fi

if rg -n 'internal/assistant' internal/accounts internal/analytics internal/financialevents internal/obligations internal/receipts internal/platform; then
  echo "financial or platform module imports internal/assistant" >&2
  exit 1
fi

if go list -deps ./cmd/api | grep -Eiq 'openai|anthropic|/llm(/|$)|provider'; then
  echo "provider dependency found in production API graph" >&2
  exit 1
fi

echo "SPEC-0005 assistant read tools integration checks passed"
