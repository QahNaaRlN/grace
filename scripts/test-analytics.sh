#!/bin/sh
set -eu

cd "$(dirname "$0")/.."

export COMPOSE_PROJECT_NAME="grace-spec0004-test-$$"
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
primary_id='10000000-0000-0000-0000-000000000001'
food_id='20000000-0000-0000-0000-000000000001'
income_id='20000000-0000-0000-0000-000000000004'

cash_id="$(curl --fail --silent -H 'Content-Type: application/json' -d '{"name":"Наличные","type":"cash","currency":"RUB","openingBalanceMinor":100000}' "$api/accounts" | json_value 'data.id')"
curl --fail --silent -H 'Content-Type: application/json' -d '{"name":"Кредитный лимит","type":"credit","currency":"RUB","openingBalanceMinor":999999}' "$api/accounts" >/dev/null
curl --fail --silent -H 'Content-Type: application/json' -d '{"name":"USD","type":"debit","currency":"USD","openingBalanceMinor":100000}' "$api/accounts" >/dev/null

curl --fail --silent -H 'Content-Type: application/json' -H 'Idempotency-Key: analytics-before' -d '{"accountId":"'"$primary_id"'","categoryId":"'"$food_id"'","amountMinor":4000,"merchant":"До периода","description":"Opening boundary","occurredAt":"2026-09-30T23:59:59Z"}' "$api/financial-events/purchases" >/dev/null
purchase="$(curl --fail --silent -H 'Content-Type: application/json' -H 'Idempotency-Key: analytics-purchase' -d '{"accountId":"'"$primary_id"'","categoryId":"'"$food_id"'","amountMinor":10000,"merchant":"Магазин","description":"На границе from","occurredAt":"2026-10-01T00:00:00Z"}' "$api/financial-events/purchases")"
purchase_id="$(printf '%s' "$purchase" | json_value 'data.id')"
curl --fail --silent -H 'Content-Type: application/json' -H 'Idempotency-Key: analytics-income' -d '{"accountId":"'"$primary_id"'","categoryId":"'"$income_id"'","amountMinor":50000,"merchant":"Работодатель","description":"Доход","occurredAt":"2026-10-05T00:00:00Z"}' "$api/financial-events/incomes" >/dev/null
curl --fail --silent -H 'Content-Type: application/json' -H 'Idempotency-Key: analytics-transfer' -d '{"sourceAccountId":"'"$primary_id"'","destinationAccountId":"'"$cash_id"'","amountMinor":20000,"description":"В наличные","occurredAt":"2026-10-06T00:00:00Z"}' "$api/financial-events/transfers" >/dev/null
curl --fail --silent -H 'Content-Type: application/json' -H 'Idempotency-Key: analytics-refund' -d '{"originalEventId":"'"$purchase_id"'","amountMinor":3000,"merchant":"Магазин","description":"Возврат","occurredAt":"2026-10-07T00:00:00Z"}' "$api/financial-events/refunds" >/dev/null
debt="$(curl --fail --silent -H 'Content-Type: application/json' -H 'Idempotency-Key: analytics-debt' -d '{"accountId":"'"$primary_id"'","categoryId":null,"amountMinor":7000,"merchant":"Банк","description":"Платёж","occurredAt":"2026-10-08T00:00:00Z"}' "$api/financial-events/debt-payments")"
debt_id="$(printf '%s' "$debt" | json_value 'data.id')"
curl --fail --silent -H 'Content-Type: application/json' -H 'Idempotency-Key: analytics-at-to' -d '{"accountId":"'"$primary_id"'","categoryId":"'"$food_id"'","amountMinor":999,"merchant":"После периода","description":"Exclusive to boundary","occurredAt":"2026-11-01T00:00:00Z"}' "$api/financial-events/purchases" >/dev/null

summary="$(curl --fail --silent --get --data-urlencode 'from=2026-10-01T00:00:00Z' --data-urlencode 'to=2026-11-01T00:00:00Z' "$api/analytics/period-summary")"
test "$(printf '%s' "$summary" | json_value 'data.currencies.map(item => item.currency).join(",")')" = "RUB,USD"
test "$(printf '%s' "$summary" | json_value 'data.currencies.find(item => item.currency === "RUB").openingBalanceMinor')" = "13595999"
test "$(printf '%s' "$summary" | json_value 'data.currencies.find(item => item.currency === "RUB").closingBalanceMinor')" = "13631999"
test "$(printf '%s' "$summary" | json_value 'data.currencies.find(item => item.currency === "RUB").incomeMinor')" = "50000"
test "$(printf '%s' "$summary" | json_value 'data.currencies.find(item => item.currency === "RUB").purchasesMinor')" = "10000"
test "$(printf '%s' "$summary" | json_value 'data.currencies.find(item => item.currency === "RUB").refundsMinor')" = "3000"
test "$(printf '%s' "$summary" | json_value 'data.currencies.find(item => item.currency === "RUB").debtPaymentsMinor')" = "7000"
test "$(printf '%s' "$summary" | json_value 'data.currencies.find(item => item.currency === "RUB").netExternalCashFlowMinor')" = "36000"
test "$(printf '%s' "$summary" | json_value 'data.currencies.find(item => item.currency === "RUB").internalTransferVolumeMinor')" = "20000"
test "$(printf '%s' "$summary" | json_value 'data.currencies.find(item => item.currency === "RUB").expenseCategories.find(item => item.categoryName === "Продукты").netExpenseMinor')" = "7000"
test "$(printf '%s' "$summary" | json_value 'data.currencies.find(item => item.currency === "USD").openingBalanceMinor')" = "100000"
test "$(printf '%s' "$summary" | json_value 'data.currencies.find(item => item.currency === "USD").closingBalanceMinor')" = "100000"

for url in \
  "$api/analytics/period-summary?to=2026-11-01T00%3A00%3A00Z" \
  "$api/analytics/period-summary?from=2026-11-01T00%3A00%3A00Z&to=2026-10-01T00%3A00%3A00Z" \
  "$api/analytics/period-summary?from=2020-01-01T00%3A00%3A00Z&to=2026-01-02T00%3A00%3A00Z"; do
  test "$(curl --silent --output /dev/null --write-out '%{http_code}' "$url")" = "422"
done

obligation="$(curl --fail --silent -H 'Content-Type: application/json' -H 'Idempotency-Key: analytics-obligation' -d '{"name":"План","kind":"loan","currency":"RUB","totalMinor":20000,"payments":[{"dueOn":"2026-09-30","amountMinor":8000},{"dueOn":"2026-10-20","amountMinor":12000}]}' "$api/obligations")"
obligation_id="$(printf '%s' "$obligation" | json_value 'data.id')"
first_payment_id="$(printf '%s' "$obligation" | json_value 'data.payments[0].id')"
curl --fail --silent -H 'Content-Type: application/json' -H 'Idempotency-Key: analytics-settlement' -d '{"financialEventId":"'"$debt_id"'"}' "$api/obligations/$obligation_id/scheduled-payments/$first_payment_id/settlements" >/dev/null

cash="$(curl --fail --silent "$api/analytics/cash-position?asOf=2026-10-04&through=2026-10-31")"
test "$(printf '%s' "$cash" | json_value 'data.currencies.map(item => item.currency).join(",")')" = "RUB,USD"
test "$(printf '%s' "$cash" | json_value 'data.currencies.find(item => item.currency === "RUB").liquidBalanceMinor')" = "12631001"
test "$(printf '%s' "$cash" | json_value 'data.currencies.find(item => item.currency === "RUB").overdueObligationsMinor')" = "1000"
test "$(printf '%s' "$cash" | json_value 'data.currencies.find(item => item.currency === "RUB").upcomingObligationsMinor')" = "12000"
test "$(printf '%s' "$cash" | json_value 'data.currencies.find(item => item.currency === "RUB").obligationsThroughMinor')" = "13000"
test "$(printf '%s' "$cash" | json_value 'data.currencies.find(item => item.currency === "RUB").availableAfterObligationsMinor')" = "12618001"
test "$(printf '%s' "$cash" | json_value 'data.currencies.find(item => item.currency === "RUB").scheduledPayments.map(item => item.timing).join(",")')" = "overdue,upcoming"
test "$(printf '%s' "$cash" | json_value 'data.currencies.find(item => item.currency === "USD").liquidBalanceMinor')" = "100000"

for url in \
  "$api/analytics/cash-position?asOf=2026-10-04" \
  "$api/analytics/cash-position?asOf=2026-10-04&through=2026-10-03" \
  "$api/analytics/cash-position?asOf=2020-01-01&through=2026-01-02"; do
  test "$(curl --silent --output /dev/null --write-out '%{http_code}' "$url")" = "422"
done

if go list -deps ./cmd/api | grep -Eiq '/(assistant|ai|llm|ocr)(/|$)'; then
  echo "AI dependency found in production API dependency graph" >&2
  exit 1
fi

echo "SPEC-0004 analytics integration checks passed"
