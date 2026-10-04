#!/bin/sh
set -eu

cd "$(dirname "$0")/.."

export COMPOSE_PROJECT_NAME="grace-spec0003-test-$$"
export POSTGRES_PORT=0
export API_PORT=0
export WEB_PORT=15173
result_dir=''

cleanup() {
  docker compose down -v --remove-orphans >/dev/null 2>&1 || true
  if [ -n "$result_dir" ]; then
    rm -rf "$result_dir"
  fi
}
trap cleanup EXIT INT TERM
cleanup
result_dir="$(mktemp -d -t grace-spec0003.XXXXXX)"

docker compose up -d postgres
until docker compose exec -T postgres pg_isready -U grace -d grace >/dev/null 2>&1; do
  sleep 1
done

database_url='postgres://grace:grace@postgres:5432/grace?sslmode=disable'
docker compose run --rm migrate -path=/migrations -database="$database_url" up

# Empty obligation schema rolls back safely and can be applied again.
docker compose run --rm migrate -path=/migrations -database="$database_url" down 1
docker compose run --rm migrate -path=/migrations -database="$database_url" up

docker compose up -d --build api
api_port="$(docker compose port api 8080 | sed -n '1s/.*://p')"
test -n "$api_port"
until curl --fail --silent "http://localhost:${api_port}/ready" >/dev/null 2>&1; do
  sleep 1
done

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
account_id='10000000-0000-0000-0000-000000000001'
balance_before="$(curl --fail --silent "$api/accounts" | json_value 'data.items.find(item => item.id === "'"$account_id"'").balanceMinor')"
postings_before="$(docker compose exec -T postgres psql -U grace -d grace -Atc 'SELECT count(*) FROM postings')"

obligation_payload='{"name":"Рассрочка на ноутбук","kind":"installment","currency":"RUB","totalMinor":30000,"payments":[{"dueOn":"2026-09-01","amountMinor":10000},{"dueOn":"2026-11-01","amountMinor":20000}]}'
obligation_response="$(curl --fail --silent --write-out '\n%{http_code}' -H 'Content-Type: application/json' -H 'Idempotency-Key: obligation-1' -d "$obligation_payload" "$api/obligations")"
test "$(printf '%s' "$obligation_response" | tail -n 1)" = "201"
obligation_body="$(printf '%s' "$obligation_response" | sed '$d')"
obligation_id="$(printf '%s' "$obligation_body" | json_value 'data.id')"
first_payment_id="$(printf '%s' "$obligation_body" | json_value 'data.payments[0].id')"
second_payment_id="$(printf '%s' "$obligation_body" | json_value 'data.payments[1].id')"
test "$(printf '%s' "$obligation_body" | json_value 'data.payments.map(item => item.position).join(",")')" = "1,2"

retry_response="$(curl --fail --silent --write-out '\n%{http_code}' -H 'Content-Type: application/json' -H 'Idempotency-Key: obligation-1' -d "$obligation_payload" "$api/obligations")"
test "$(printf '%s' "$retry_response" | tail -n 1)" = "200"
test "$(printf '%s' "$retry_response" | sed '$d' | json_value 'data.id')" = "$obligation_id"

conflict_status="$(curl --silent --output /dev/null --write-out '%{http_code}' -H 'Content-Type: application/json' -H 'Idempotency-Key: obligation-1' -d '{"name":"Другой договор","kind":"loan","currency":"RUB","totalMinor":10000,"payments":[{"dueOn":"2026-12-01","amountMinor":10000}]}' "$api/obligations")"
test "$conflict_status" = "409"
invalid_schedule_status="$(curl --silent --output /dev/null --write-out '%{http_code}' -H 'Content-Type: application/json' -H 'Idempotency-Key: obligation-invalid' -d '{"name":"Ошибка","kind":"loan","currency":"RUB","totalMinor":10000,"payments":[{"dueOn":"2026-12-01","amountMinor":9999}]}' "$api/obligations")"
test "$invalid_schedule_status" = "422"

as_of_body="$(curl --fail --silent "$api/obligations/$obligation_id?asOf=2026-10-04")"
test "$(printf '%s' "$as_of_body" | json_value 'data.payments[0].status')" = "overdue"
test "$(printf '%s' "$as_of_body" | json_value 'data.payments[0].isOverdue')" = "true"
test "$(printf '%s' "$as_of_body" | json_value 'data.payments[1].status')" = "planned"
invalid_as_of_status="$(curl --silent --output /dev/null --write-out '%{http_code}' "$api/obligations?asOf=2026-02-30")"
test "$invalid_as_of_status" = "422"

balance_after_schedule="$(curl --fail --silent "$api/accounts" | json_value 'data.items.find(item => item.id === "'"$account_id"'").balanceMinor')"
postings_after_schedule="$(docker compose exec -T postgres psql -U grace -d grace -Atc 'SELECT count(*) FROM postings')"
test "$balance_before" = "$balance_after_schedule"
test "$postings_before" = "$postings_after_schedule"

debt_one="$(curl --fail --silent -H 'Content-Type: application/json' -H 'Idempotency-Key: obligation-debt-1' -d '{"accountId":"'"$account_id"'","categoryId":null,"amountMinor":4000,"merchant":"Банк","description":"Частичная оплата","occurredAt":"2026-10-04T12:00:00Z"}' "$api/financial-events/debt-payments")"
debt_one_id="$(printf '%s' "$debt_one" | json_value 'data.id')"
settlement_payload='{"financialEventId":"'"$debt_one_id"'"}'
balance_before_settlement="$(curl --fail --silent "$api/accounts" | json_value 'data.items.find(item => item.id === "'"$account_id"'").balanceMinor')"
postings_before_settlement="$(docker compose exec -T postgres psql -U grace -d grace -Atc 'SELECT count(*) FROM postings')"
settlement_response="$(curl --fail --silent --write-out '\n%{http_code}' -H 'Content-Type: application/json' -H 'Idempotency-Key: settlement-1' -d "$settlement_payload" "$api/obligations/$obligation_id/scheduled-payments/$first_payment_id/settlements")"
test "$(printf '%s' "$settlement_response" | tail -n 1)" = "201"
test "$(printf '%s' "$settlement_response" | sed '$d' | json_value 'data.amountMinor')" = "4000"

settlement_retry="$(curl --fail --silent --write-out '\n%{http_code}' -H 'Content-Type: application/json' -H 'Idempotency-Key: settlement-1' -d "$settlement_payload" "$api/obligations/$obligation_id/scheduled-payments/$first_payment_id/settlements")"
test "$(printf '%s' "$settlement_retry" | tail -n 1)" = "200"
reuse_status="$(curl --silent --output /dev/null --write-out '%{http_code}' -H 'Content-Type: application/json' -H 'Idempotency-Key: settlement-reuse' -d "$settlement_payload" "$api/obligations/$obligation_id/scheduled-payments/$second_payment_id/settlements")"
test "$reuse_status" = "409"

balance_after_settlement="$(curl --fail --silent "$api/accounts" | json_value 'data.items.find(item => item.id === "'"$account_id"'").balanceMinor')"
postings_after_settlement="$(docker compose exec -T postgres psql -U grace -d grace -Atc 'SELECT count(*) FROM postings')"
test "$balance_before_settlement" = "$balance_after_settlement"
test "$postings_before_settlement" = "$postings_after_settlement"

partial_body="$(curl --fail --silent "$api/obligations/$obligation_id?asOf=2026-10-04")"
test "$(printf '%s' "$partial_body" | json_value 'data.payments[0].status')" = "partially_paid"
test "$(printf '%s' "$partial_body" | json_value 'data.payments[0].isOverdue')" = "true"
test "$(printf '%s' "$partial_body" | json_value 'data.payments[0].remainingMinor')" = "6000"

overpay="$(curl --fail --silent -H 'Content-Type: application/json' -H 'Idempotency-Key: obligation-debt-overpay' -d '{"accountId":"'"$account_id"'","categoryId":null,"amountMinor":7000,"merchant":"Банк","description":"Сверх остатка","occurredAt":"2026-10-04T13:00:00Z"}' "$api/financial-events/debt-payments")"
overpay_id="$(printf '%s' "$overpay" | json_value 'data.id')"
overpay_status="$(curl --silent --output /dev/null --write-out '%{http_code}' -H 'Content-Type: application/json' -H 'Idempotency-Key: settlement-overpay' -d '{"financialEventId":"'"$overpay_id"'"}' "$api/obligations/$obligation_id/scheduled-payments/$first_payment_id/settlements")"
test "$overpay_status" = "422"

purchase="$(curl --fail --silent -H 'Content-Type: application/json' -H 'Idempotency-Key: obligation-purchase' -d '{"accountId":"'"$account_id"'","categoryId":"20000000-0000-0000-0000-000000000001","amountMinor":6000,"merchant":"Магазин","description":"Не settlement","occurredAt":"2026-10-04T14:00:00Z"}' "$api/financial-events/purchases")"
purchase_id="$(printf '%s' "$purchase" | json_value 'data.id')"
purchase_status="$(curl --silent --output /dev/null --write-out '%{http_code}' -H 'Content-Type: application/json' -H 'Idempotency-Key: settlement-purchase' -d '{"financialEventId":"'"$purchase_id"'"}' "$api/obligations/$obligation_id/scheduled-payments/$first_payment_id/settlements")"
test "$purchase_status" = "422"

debt_two="$(curl --fail --silent -H 'Content-Type: application/json' -H 'Idempotency-Key: obligation-debt-2' -d '{"accountId":"'"$account_id"'","categoryId":null,"amountMinor":6000,"merchant":"Банк","description":"Закрытие платежа","occurredAt":"2026-10-04T15:00:00Z"}' "$api/financial-events/debt-payments")"
debt_two_id="$(printf '%s' "$debt_two" | json_value 'data.id')"
curl --fail --silent -H 'Content-Type: application/json' -H 'Idempotency-Key: settlement-2' -d '{"financialEventId":"'"$debt_two_id"'"}' "$api/obligations/$obligation_id/scheduled-payments/$first_payment_id/settlements" >/dev/null
test "$(curl --fail --silent "$api/obligations/$obligation_id?asOf=2026-10-04" | json_value 'data.payments[0].status')" = "paid"

usd_account="$(curl --fail --silent -H 'Content-Type: application/json' -d '{"name":"USD счёт","type":"debit","currency":"USD","openingBalanceMinor":100000}' "$api/accounts")"
usd_account_id="$(printf '%s' "$usd_account" | json_value 'data.id')"
usd_debt="$(curl --fail --silent -H 'Content-Type: application/json' -H 'Idempotency-Key: obligation-debt-usd' -d '{"accountId":"'"$usd_account_id"'","categoryId":null,"amountMinor":1000,"merchant":"Банк","description":"Другая валюта","occurredAt":"2026-10-04T15:30:00Z"}' "$api/financial-events/debt-payments")"
usd_debt_id="$(printf '%s' "$usd_debt" | json_value 'data.id')"
currency_status="$(curl --silent --output /dev/null --write-out '%{http_code}' -H 'Content-Type: application/json' -H 'Idempotency-Key: settlement-usd' -d '{"financialEventId":"'"$usd_debt_id"'"}' "$api/obligations/$obligation_id/scheduled-payments/$second_payment_id/settlements")"
test "$currency_status" = "422"

final_debt="$(curl --fail --silent -H 'Content-Type: application/json' -H 'Idempotency-Key: obligation-debt-final' -d '{"accountId":"'"$account_id"'","categoryId":null,"amountMinor":20000,"merchant":"Банк","description":"Закрытие обязательства","occurredAt":"2026-10-04T15:45:00Z"}' "$api/financial-events/debt-payments")"
final_debt_id="$(printf '%s' "$final_debt" | json_value 'data.id')"
curl --fail --silent -H 'Content-Type: application/json' -H 'Idempotency-Key: settlement-final' -d '{"financialEventId":"'"$final_debt_id"'"}' "$api/obligations/$obligation_id/scheduled-payments/$second_payment_id/settlements" >/dev/null
completed_body="$(curl --fail --silent "$api/obligations/$obligation_id?asOf=2026-10-04")"
test "$(printf '%s' "$completed_body" | json_value 'data.status')" = "completed"
test "$(printf '%s' "$completed_body" | json_value 'data.remainingMinor')" = "0"

concurrent_obligation="$(curl --fail --silent -H 'Content-Type: application/json' -H 'Idempotency-Key: obligation-concurrent' -d '{"name":"Конкурентный график","kind":"other_fixed","currency":"RUB","totalMinor":10000,"payments":[{"dueOn":"2026-12-01","amountMinor":10000}]}' "$api/obligations")"
concurrent_obligation_id="$(printf '%s' "$concurrent_obligation" | json_value 'data.id')"
concurrent_payment_id="$(printf '%s' "$concurrent_obligation" | json_value 'data.payments[0].id')"
concurrent_a="$(curl --fail --silent -H 'Content-Type: application/json' -H 'Idempotency-Key: obligation-concurrent-debt-a' -d '{"accountId":"'"$account_id"'","categoryId":null,"amountMinor":6000,"merchant":"Банк","description":"A","occurredAt":"2026-10-04T16:00:00Z"}' "$api/financial-events/debt-payments" | json_value 'data.id')"
concurrent_b="$(curl --fail --silent -H 'Content-Type: application/json' -H 'Idempotency-Key: obligation-concurrent-debt-b' -d '{"accountId":"'"$account_id"'","categoryId":null,"amountMinor":6000,"merchant":"Банк","description":"B","occurredAt":"2026-10-04T16:01:00Z"}' "$api/financial-events/debt-payments" | json_value 'data.id')"
curl --silent --output /dev/null --write-out '%{http_code}' -H 'Content-Type: application/json' -H 'Idempotency-Key: settlement-concurrent-a' -d '{"financialEventId":"'"$concurrent_a"'"}' "$api/obligations/$concurrent_obligation_id/scheduled-payments/$concurrent_payment_id/settlements" >"$result_dir/a.status" &
first_pid=$!
curl --silent --output /dev/null --write-out '%{http_code}' -H 'Content-Type: application/json' -H 'Idempotency-Key: settlement-concurrent-b' -d '{"financialEventId":"'"$concurrent_b"'"}' "$api/obligations/$concurrent_obligation_id/scheduled-payments/$concurrent_payment_id/settlements" >"$result_dir/b.status" &
second_pid=$!
wait "$first_pid"
wait "$second_pid"
test "$(sort "$result_dir/a.status" "$result_dir/b.status" | tr '\n' ' ')" = "201 422 "

if docker compose run --rm migrate -path=/migrations -database="$database_url" down 1 >/dev/null 2>&1; then
  echo "obligation migration rollback unexpectedly removed user data" >&2
  exit 1
fi

if go list -deps ./cmd/api | grep -Eiq '/(assistant|ai|llm|ocr)(/|$)'; then
  echo "AI dependency found in production API dependency graph" >&2
  exit 1
fi

echo "SPEC-0003 obligation integration checks passed"
