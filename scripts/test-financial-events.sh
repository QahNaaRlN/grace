#!/bin/sh
set -eu

cd "$(dirname "$0")/.."

export COMPOSE_PROJECT_NAME="grace-spec0001-test-$$"
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
docker compose run --rm migrate -path=/migrations -database="$database_url" goto 2

docker compose exec -T postgres psql -U grace -d grace -v ON_ERROR_STOP=1 -c "
  INSERT INTO transactions (
    id, account_id, category_id, amount_minor, currency, direction, kind,
    source, merchant, description, occurred_at
  ) VALUES (
    '30000000-0000-0000-0000-000000000001',
    '10000000-0000-0000-0000-000000000001',
    '20000000-0000-0000-0000-000000000001',
    12345, 'RUB', 'expense', 'purchase', 'manual',
    'Legacy merchant', 'Backfill proof', '2026-10-03T12:00:00Z'
  );
  UPDATE accounts
  SET balance_minor = balance_minor - 12345
  WHERE id = '10000000-0000-0000-0000-000000000001';
"

docker compose run --rm migrate -path=/migrations -database="$database_url" up

# Legacy-only state can roll back without data loss, then migrate forward again.
docker compose run --rm migrate -path=/migrations -database="$database_url" down 1
legacy_rows="$(docker compose exec -T postgres psql -U grace -d grace -Atc "SELECT count(*) FROM transactions WHERE id = '30000000-0000-0000-0000-000000000001'")"
test "$legacy_rows" = "1"
docker compose run --rm migrate -path=/migrations -database="$database_url" up

docker compose up -d --build api

api_port="$(docker compose port api 8080 | sed -n '1s/.*://p')"
test -n "$api_port"

until curl --fail --silent "http://localhost:${api_port}/ready" >/dev/null 2>&1; do
  sleep 1
done

cors_headers="$(curl --fail --silent --dump-header - --output /dev/null -X OPTIONS \
  "http://localhost:${api_port}/api/v1/financial-events/purchases" \
  -H "Origin: http://localhost:${WEB_PORT}" \
  -H 'Access-Control-Request-Method: POST' \
  -H 'Access-Control-Request-Headers: content-type,idempotency-key')"
printf '%s' "$cors_headers" | grep -qi '^Access-Control-Allow-Headers:.*Idempotency-Key'

json_value() {
  node -e '
    let input = "";
    process.stdin.on("data", chunk => input += chunk);
    process.stdin.on("end", () => {
      const value = Function("data", "return " + process.argv[1])(JSON.parse(input));
      process.stdout.write(String(value));
    });
  ' "$1"
}

api="http://localhost:${api_port}/api/v1"
accounts="$(curl --fail --silent "$api/accounts")"
primary_id="$(printf '%s' "$accounts" | json_value 'data.items.find(item => item.name === "Основная карта").id')"
legacy_balance="$(printf '%s' "$accounts" | json_value 'data.items.find(item => item.id === "'"$primary_id"'").balanceMinor')"
test "$legacy_balance" = "12487655"

savings="$(curl --fail --silent -H 'Content-Type: application/json' -d '{
  "name":"Накопительный","type":"debit","currency":"RUB","openingBalanceMinor":500000
}' "$api/accounts")"
savings_id="$(printf '%s' "$savings" | json_value 'data.id')"

purchase_payload='{"accountId":"'"$primary_id"'","categoryId":"20000000-0000-0000-0000-000000000001","amountMinor":10000,"merchant":"Магазин","description":"Идемпотентная покупка","occurredAt":"2026-10-04T12:00:00Z"}'
purchase="$(curl --fail --silent -H 'Content-Type: application/json' -H 'Idempotency-Key: purchase-1' -d "$purchase_payload" "$api/financial-events/purchases")"
purchase_id="$(printf '%s' "$purchase" | json_value 'data.id')"
retry_id="$(curl --fail --silent -H 'Content-Type: application/json' -H 'Idempotency-Key: purchase-1' -d "$purchase_payload" "$api/financial-events/purchases" | json_value 'data.id')"
test "$purchase_id" = "$retry_id"

conflict_status="$(curl --silent --output /dev/null --write-out '%{http_code}' -H 'Content-Type: application/json' -H 'Idempotency-Key: purchase-1' -d '{
  "accountId":"'"$primary_id"'","categoryId":"20000000-0000-0000-0000-000000000001","amountMinor":10001,"merchant":"Магазин","description":"Другой payload","occurredAt":"2026-10-04T12:00:00Z"
}' "$api/financial-events/purchases")"
test "$conflict_status" = "409"

curl --fail --silent -H 'Content-Type: application/json' -H 'Idempotency-Key: income-1' -d '{
  "accountId":"'"$primary_id"'","categoryId":"20000000-0000-0000-0000-000000000004","amountMinor":8000000,"merchant":"Работодатель","description":"Зарплата","occurredAt":"2026-10-04T13:00:00Z"
}' "$api/financial-events/incomes" >/dev/null

curl --fail --silent -H 'Content-Type: application/json' -H 'Idempotency-Key: debt-1' -d '{
  "accountId":"'"$primary_id"'","categoryId":"20000000-0000-0000-0000-000000000003","amountMinor":25000,"merchant":"Банк","description":"Платёж по обязательству","occurredAt":"2026-10-04T13:30:00Z"
}' "$api/financial-events/debt-payments" >/dev/null

before_transfer="$(curl --fail --silent "$api/accounts")"
total_before="$(printf '%s' "$before_transfer" | json_value 'data.items.reduce((sum, item) => sum + item.balanceMinor, 0)')"
curl --fail --silent -H 'Content-Type: application/json' -H 'Idempotency-Key: transfer-1' -d '{
  "sourceAccountId":"'"$primary_id"'","destinationAccountId":"'"$savings_id"'","amountMinor":200000,"description":"В накопления","occurredAt":"2026-10-04T14:00:00Z"
}' "$api/financial-events/transfers" >/dev/null
after_transfer="$(curl --fail --silent "$api/accounts")"
total_after="$(printf '%s' "$after_transfer" | json_value 'data.items.reduce((sum, item) => sum + item.balanceMinor, 0)')"
test "$total_before" = "$total_after"

curl --fail --silent -H 'Content-Type: application/json' -H 'Idempotency-Key: refund-1' -d '{
  "originalEventId":"'"$purchase_id"'","amountMinor":5000,"merchant":"Магазин","description":"Частичный возврат","occurredAt":"2026-10-04T15:00:00Z"
}' "$api/financial-events/refunds" >/dev/null
refund_status="$(curl --silent --output /dev/null --write-out '%{http_code}' -H 'Content-Type: application/json' -H 'Idempotency-Key: refund-2' -d '{
  "originalEventId":"'"$purchase_id"'","amountMinor":6000,"merchant":"Магазин","description":"Сверх покупки","occurredAt":"2026-10-04T16:00:00Z"
}' "$api/financial-events/refunds")"
test "$refund_status" = "422"

curl --fail --silent -H 'Content-Type: application/json' -H 'Idempotency-Key: negative-1' -d '{
  "accountId":"'"$savings_id"'","categoryId":"20000000-0000-0000-0000-000000000001","amountMinor":1000000,"merchant":"Крупная покупка","description":"Проверка отрицательного остатка","occurredAt":"2026-10-04T17:00:00Z"
}' "$api/financial-events/purchases" >/dev/null
negative_balance="$(curl --fail --silent "$api/accounts" | json_value 'data.items.find(item => item.id === "'"$savings_id"'").balanceMinor')"
test "$negative_balance" -lt 0

old_endpoint_status="$(curl --silent --output /dev/null --write-out '%{http_code}' "$api/transactions")"
test "$old_endpoint_status" = "404"

event_count="$(curl --fail --silent "$api/financial-events" | json_value 'data.items.length')"
test "$event_count" -ge 7

echo "SPEC-0001 integration checks passed"
