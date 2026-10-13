#!/bin/sh
set -eu
# Git Bash on Windows rewrites the migrate -path=/migrations argument into a host
# path (#25); exclude only that argument so native tools still get /tmp and /dev/null.
export MSYS2_ARG_CONV_EXCL="${MSYS2_ARG_CONV_EXCL:+$MSYS2_ARG_CONV_EXCL;}-path="

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

if go list -deps ./cmd/api | grep -Eiq '/(jeveval|routereval)(/|$)'; then
  echo "evaluation dependency found in production API dependency graph" >&2
  exit 1
fi

before_statistics="$(curl --fail --silent "$api/financial-events")"
statistics="$(curl --fail --silent "$api/analytics/statistics?from=2026-10-01&to=2026-11-01&granularity=month")"
test "$(printf '%s' "$statistics" | json_value 'data.coverage')" = "recorded_only"
test "$(printf '%s' "$statistics" | json_value 'data.currencies.map(c=>c.currency).join(",")')" = "RUB,USD"
test "$(printf '%s' "$statistics" | json_value 'data.currencies[0].series[0].purchasesMinor')" = "14000"
test "$(printf '%s' "$statistics" | json_value 'data.currencies[0].series[0].refundsMinor')" = "3000"
test "$(printf '%s' "$statistics" | json_value 'data.currencies[0].series[0].incomeMinor')" = "50000"
test "$(printf '%s' "$statistics" | json_value 'data.currencies[0].series[0].outflowsMinor')" = "21000"
test "$(printf '%s' "$statistics" | json_value 'data.currencies[0].categories[0].shareBasisPoints')" = "10000"
test "$(printf '%s' "$statistics" | json_value 'data.currencies[1].series[0].outflowsMinor')" = "0"
weekly="$(curl --fail --silent "$api/analytics/statistics?from=2026-10-01&to=2026-11-01&granularity=week")"
test "$(printf '%s' "$weekly" | json_value 'data.currencies[0].series.length')" = "5"
test "$(printf '%s' "$weekly" | json_value 'data.currencies[0].series.reduce((sum,b)=>sum+b.outflowsMinor,0)')" = "21000"
test "$before_statistics" = "$(curl --fail --silent "$api/financial-events")"
for query in 'from=2026-02-30&to=2026-03-01' 'from=2025-01-01&to=2026-01-02' 'from=2026-10-01&to=2026-11-01&granularity=day'; do
  test "$(curl --silent --output /dev/null --write-out '%{http_code}' "$api/analytics/statistics?$query")" = "422"
done
# Date-only history is grouped by its actual calendar date, with no fabricated time.
docker compose exec -T postgres psql -U grace -d grace -v ON_ERROR_STOP=1 -c "UPDATE financial_events SET occurred_at=NULL, occurred_date='2026-10-01' WHERE id='$purchase_id'" >/dev/null
date_only="$(curl --fail --silent "$api/analytics/statistics?from=2026-10-01&to=2026-11-01&granularity=month")"
test "$(printf '%s' "$date_only" | json_value 'data.currencies[0].series[0].purchasesMinor')" = "14000"
empty_stats="$(curl --fail --silent "$api/analytics/statistics?from=2026-08-01&to=2026-09-01")"
test "$(printf '%s' "$empty_stats" | json_value 'data.currencies[0].series[0].purchasesMinor')" = "0"
test "$(printf '%s' "$empty_stats" | json_value 'data.currencies[0].categories.length')" = "0"
docker compose exec -T postgres psql -U grace -d grace -v ON_ERROR_STOP=1 <<'SQL' >/dev/null
INSERT INTO users(id,display_name,role) VALUES ('33333333-3333-4333-8333-333333333333','Synthetic isolation','member');
INSERT INTO accounts(id,name,type,currency,opening_balance_minor,owner_id) VALUES ('44444444-4444-4444-8444-444444444444','Other owner','cash','GBP',0,'33333333-3333-4333-8333-333333333333');
INSERT INTO financial_events(id,kind,source,occurred_at,idempotency_key,request_hash,owner_id) VALUES ('55555555-5555-4555-8555-555555555555','purchase','manual','2026-10-03T00:00:00Z','statistics-isolation',repeat('0',64),'33333333-3333-4333-8333-333333333333');
INSERT INTO postings(financial_event_id,account_id,direction,amount_minor,currency,owner_id) VALUES ('55555555-5555-4555-8555-555555555555','44444444-4444-4444-8444-444444444444','debit',999999,'GBP','33333333-3333-4333-8333-333333333333');
SQL
isolated="$(curl --fail --silent "$api/analytics/statistics?from=2026-10-01&to=2026-11-01")"
test "$(printf '%s' "$isolated" | json_value 'data.currencies.map(c=>c.currency).join(",")')" = "RUB,USD"
test "$(printf '%s' "$isolated" | json_value 'data.currencies[0].series[0].purchasesMinor')" = "14000"
# Historical imports can contain an unknown category; manual purchase API requires one.
docker compose exec -T postgres psql -U grace -d grace -v ON_ERROR_STOP=1 <<SQL >/dev/null
INSERT INTO financial_events(id,kind,source,occurred_date,idempotency_key,request_hash,owner_id) VALUES ('66666666-6666-4666-8666-666666666666','purchase','bank_import','2026-10-15','statistics-uncategorized',repeat('0',64),'00000000-0000-4000-8000-000000000001');
INSERT INTO postings(financial_event_id,account_id,direction,amount_minor,currency,owner_id) VALUES ('66666666-6666-4666-8666-666666666666','$primary_id','debit',20,'RUB','00000000-0000-4000-8000-000000000001');
SQL
docker compose exec -T postgres psql -U grace -d grace -v ON_ERROR_STOP=1 -c "UPDATE accounts SET balance_reconciled=false WHERE id='$primary_id'" >/dev/null
unknown="$(curl --fail --silent "$api/analytics/statistics?from=2026-10-01&to=2026-11-01")"
test "$(printf '%s' "$unknown" | json_value 'data.currencies[0].summary.openingBalanceMinor')" = "null"
test "$(printf '%s' "$unknown" | json_value 'data.currencies[0].summary.closingBalanceMinor')" = "null"
test "$(printf '%s' "$unknown" | json_value 'data.currencies[0].summary.purchasesMinor')" = "14020"
test "$(printf '%s' "$unknown" | json_value 'data.currencies[0].categories.find(c=>c.categoryId===null).categoryName')" = "Без категории"
test "$(printf '%s' "$unknown" | json_value 'data.currencies[0].categories.find(c=>c.categoryId===null).purchasesMinor')" = "20"
echo "SPEC-0004 and SPEC-0030 analytics integration checks passed"
