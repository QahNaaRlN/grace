#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root"

go test ./internal/assistant ./internal/routereval ./cmd/jev-eval
go run ./cmd/jev-eval validate --dataset testdata/jev-evaluation/v2/cases.json

if rg -n 'internal/assistant' internal/accounts internal/analytics internal/financialevents internal/obligations internal/receipts; then
  echo "financial core imports internal/assistant" >&2
  exit 1
fi

if go list -deps ./cmd/api | rg -q '^github\.com/QahNaaRlN/grace/internal/(jeveval|routereval)$'; then
  echo "production API depends on an evaluation package" >&2
  exit 1
fi

if rg -n --glob '*.go' '(SELECT|INSERT|UPDATE|DELETE)[[:space:]]' internal/assistant; then
  echo "SQL found in internal/assistant" >&2
  exit 1
fi

export COMPOSE_PROJECT_NAME="grace-spec0008-test-$$"
export POSTGRES_PORT=0
export API_PORT=0
export WEB_PORT=15173
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
  go test -tags=integration ./internal/assistanttest -run '^TestReadOnlyRouterAgainstPostgreSQL$' -count=1

echo "SPEC-0008 production read-only router checks passed"
