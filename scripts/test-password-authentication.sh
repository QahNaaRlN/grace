#!/bin/sh
set -eu
cd "$(dirname "$0")/.."
export COMPOSE_PROJECT_NAME="grace-password-test-$$"
export POSTGRES_PORT=0 API_PORT=0 WEB_PORT=15173
cleanup(){ docker compose down -v --remove-orphans >/dev/null 2>&1 || true; }
trap cleanup EXIT INT TERM
docker compose up -d postgres
until docker compose exec -T postgres pg_isready -U grace -d grace >/dev/null 2>&1; do sleep 1; done
docker compose run --rm migrate -path=/migrations -database='postgres://grace:grace@postgres:5432/grace?sslmode=disable' up
postgres_port="$(docker compose port postgres 5432 | sed -n '1s/.*://p')"
race_flag=''
if [ "$(go env CGO_ENABLED)" = '1' ]; then race_flag='-race'; fi
TEST_DATABASE_URL="postgres://grace:grace@localhost:${postgres_port}/grace?sslmode=disable" go test $race_flag -tags=integration ./internal/auth -run TestPassword -count=1
if docker compose run --rm migrate -path=/migrations -database='postgres://grace:grace@postgres:5432/grace?sslmode=disable' down 1; then
  echo 'Unsafe password credential deletion was accepted' >&2; exit 1
fi
echo 'SPEC-0031 password authentication integration checks passed'
