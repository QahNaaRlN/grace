#!/bin/sh
set -eu
# Git Bash on Windows rewrites the migrate -path=/migrations argument into a host
# path (#25); exclude only that argument so native tools still get /tmp and /dev/null.
export MSYS2_ARG_CONV_EXCL="${MSYS2_ARG_CONV_EXCL:+$MSYS2_ARG_CONV_EXCL;}-path="

cd "$(dirname "$0")/.."

export COMPOSE_PROJECT_NAME="grace-spec0002-test-$$"
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
result_dir="$(mktemp -d -t grace-spec0002.XXXXXX)"

docker compose up -d postgres
until docker compose exec -T postgres pg_isready -U grace -d grace >/dev/null 2>&1; do
  sleep 1
done

database_url='postgres://grace:grace@postgres:5432/grace?sslmode=disable'
docker compose run --rm migrate -path=/migrations -database="$database_url" up

# Empty receipt schema rolls back safely and can be applied again. Use the
# explicit pre-receipt version so later migrations do not change this proof.
docker compose run --rm migrate -path=/migrations -database="$database_url" goto 3
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
# JSON Unicode escapes preserve Cyrillic with native curl on Windows as well as Linux.
purchase_payload='{"accountId":"10000000-0000-0000-0000-000000000001","categoryId":"20000000-0000-0000-0000-000000000001","amountMinor":34990,"merchant":"\u0416\u0438\u0437\u043d\u044c\u043c\u0430\u0440\u0442","description":"\u041f\u043e\u043a\u0443\u043f\u043a\u0430 \u0441 \u0447\u0435\u043a\u043e\u043c","occurredAt":"2026-10-04T12:00:00Z"}'
purchase="$(curl --fail --silent -H 'Content-Type: application/json' -H 'Idempotency-Key: receipt-purchase-1' -d "$purchase_payload" "$api/financial-events/purchases")"
purchase_id="$(printf '%s' "$purchase" | json_value 'data.id')"

balance_before="$(curl --fail --silent "$api/accounts" | json_value 'data.items.find(item => item.id === "10000000-0000-0000-0000-000000000001").balanceMinor')"
postings_before="$(docker compose exec -T postgres psql -U grace -d grace -Atc 'SELECT count(*) FROM postings')"

receipt_payload='{"purchaseEventId":"'"$purchase_id"'","seller":"\u0416\u0438\u0437\u043d\u044c\u043c\u0430\u0440\u0442","issuedAt":"2026-10-04T12:00:00Z","fiscalDocumentNumber":"FD-42","items":[{"name":"\u041c\u043e\u043b\u043e\u043a\u043e","quantityMilli":1000,"unitPriceMinor":12990,"lineTotalMinor":12990},{"name":"\u041e\u0432\u043e\u0449\u0438","quantityMilli":650,"unitPriceMinor":33846,"lineTotalMinor":22000}]}'
receipt_response="$(curl --fail --silent --write-out '\n%{http_code}' -H 'Content-Type: application/json' -H 'Idempotency-Key: receipt-1' -d "$receipt_payload" "$api/receipts")"
receipt_status="$(printf '%s' "$receipt_response" | tail -n 1)"
receipt_body="$(printf '%s' "$receipt_response" | sed '$d')"
test "$receipt_status" = "201"
receipt_id="$(printf '%s' "$receipt_body" | json_value 'data.id')"
test "$(printf '%s' "$receipt_body" | json_value 'data.totalMinor')" = "34990"
test "$(printf '%s' "$receipt_body" | json_value 'data.currency')" = "RUB"
test "$(printf '%s' "$receipt_body" | json_value 'data.items.map(item => item.position).join(",")')" = "1,2"

retry_response="$(curl --fail --silent --write-out '\n%{http_code}' -H 'Content-Type: application/json' -H 'Idempotency-Key: receipt-1' -d "$receipt_payload" "$api/receipts")"
test "$(printf '%s' "$retry_response" | tail -n 1)" = "200"
retry_id="$(printf '%s' "$retry_response" | sed '$d' | json_value 'data.id')"
test "$receipt_id" = "$retry_id"

conflict_status="$(curl --silent --output /dev/null --write-out '%{http_code}' -H 'Content-Type: application/json' -H 'Idempotency-Key: receipt-1' -d '{"purchaseEventId":"'"$purchase_id"'","seller":"\u0414\u0440\u0443\u0433\u043e\u0439 \u043f\u0440\u043e\u0434\u0430\u0432\u0435\u0446","issuedAt":"2026-10-04T12:00:00Z","items":[{"name":"\u0418\u043d\u0430\u044f \u043f\u043e\u0437\u0438\u0446\u0438\u044f","quantityMilli":1000,"unitPriceMinor":34990,"lineTotalMinor":34990}]}' "$api/receipts")"
test "$conflict_status" = "409"

second_receipt_status="$(curl --silent --output /dev/null --write-out '%{http_code}' -H 'Content-Type: application/json' -H 'Idempotency-Key: receipt-2' -d "$receipt_payload" "$api/receipts")"
test "$second_receipt_status" = "409"

test "$(curl --fail --silent "$api/receipts/$receipt_id" | json_value 'data.items[0].name')" = "Молоко"
test "$(curl --fail --silent "$api/receipts" | json_value 'data.items.length')" = "1"

balance_after="$(curl --fail --silent "$api/accounts" | json_value 'data.items.find(item => item.id === "10000000-0000-0000-0000-000000000001").balanceMinor')"
postings_after="$(docker compose exec -T postgres psql -U grace -d grace -Atc 'SELECT count(*) FROM postings')"
test "$balance_before" = "$balance_after"
test "$postings_before" = "$postings_after"

mismatch_purchase="$(curl --fail --silent -H 'Content-Type: application/json' -H 'Idempotency-Key: receipt-purchase-2' -d '{"accountId":"10000000-0000-0000-0000-000000000001","categoryId":"20000000-0000-0000-0000-000000000001","amountMinor":10000,"merchant":"\u041c\u0430\u0433\u0430\u0437\u0438\u043d","description":"\u041d\u0435\u0441\u043e\u0432\u043f\u0430\u0434\u0435\u043d\u0438\u0435","occurredAt":"2026-10-04T13:00:00Z"}' "$api/financial-events/purchases")"
mismatch_purchase_id="$(printf '%s' "$mismatch_purchase" | json_value 'data.id')"
mismatch_status="$(curl --silent --output /dev/null --write-out '%{http_code}' -H 'Content-Type: application/json' -H 'Idempotency-Key: receipt-mismatch' -d '{"purchaseEventId":"'"$mismatch_purchase_id"'","seller":"\u041c\u0430\u0433\u0430\u0437\u0438\u043d","issuedAt":"2026-10-04T13:00:00Z","items":[{"name":"\u0422\u043e\u0432\u0430\u0440","quantityMilli":1000,"unitPriceMinor":9999,"lineTotalMinor":9999}]}' "$api/receipts")"
test "$mismatch_status" = "422"

income="$(curl --fail --silent -H 'Content-Type: application/json' -H 'Idempotency-Key: receipt-income-1' -d '{"accountId":"10000000-0000-0000-0000-000000000001","categoryId":"20000000-0000-0000-0000-000000000004","amountMinor":10000,"merchant":"\u0420\u0430\u0431\u043e\u0442\u043e\u0434\u0430\u0442\u0435\u043b\u044c","description":"\u0414\u043e\u0445\u043e\u0434","occurredAt":"2026-10-04T14:00:00Z"}' "$api/financial-events/incomes")"
income_id="$(printf '%s' "$income" | json_value 'data.id')"
income_receipt_status="$(curl --silent --output /dev/null --write-out '%{http_code}' -H 'Content-Type: application/json' -H 'Idempotency-Key: receipt-income' -d '{"purchaseEventId":"'"$income_id"'","seller":"\u041d\u0435 \u0447\u0435\u043a","issuedAt":"2026-10-04T14:00:00Z","items":[{"name":"\u0421\u0442\u0440\u043e\u043a\u0430","quantityMilli":1000,"unitPriceMinor":10000,"lineTotalMinor":10000}]}' "$api/receipts")"
test "$income_receipt_status" = "422"

savings="$(curl --fail --silent -H 'Content-Type: application/json' -d '{"name":"\u0412\u0442\u043e\u0440\u043e\u0439 \u0441\u0447\u0451\u0442","type":"debit","currency":"RUB","openingBalanceMinor":0}' "$api/accounts")"
savings_id="$(printf '%s' "$savings" | json_value 'data.id')"
transfer="$(curl --fail --silent -H 'Content-Type: application/json' -H 'Idempotency-Key: receipt-transfer-1' -d '{"sourceAccountId":"10000000-0000-0000-0000-000000000001","destinationAccountId":"'"$savings_id"'","amountMinor":10000,"description":"\u041f\u0435\u0440\u0435\u0432\u043e\u0434","occurredAt":"2026-10-04T14:10:00Z"}' "$api/financial-events/transfers")"
transfer_id="$(printf '%s' "$transfer" | json_value 'data.id')"
transfer_receipt_status="$(curl --silent --output /dev/null --write-out '%{http_code}' -H 'Content-Type: application/json' -H 'Idempotency-Key: receipt-transfer' -d '{"purchaseEventId":"'"$transfer_id"'","seller":"\u041d\u0435 \u0447\u0435\u043a","issuedAt":"2026-10-04T14:10:00Z","items":[{"name":"\u0421\u0442\u0440\u043e\u043a\u0430","quantityMilli":1000,"unitPriceMinor":10000,"lineTotalMinor":10000}]}' "$api/receipts")"
test "$transfer_receipt_status" = "422"

refund="$(curl --fail --silent -H 'Content-Type: application/json' -H 'Idempotency-Key: receipt-refund-1' -d '{"originalEventId":"'"$mismatch_purchase_id"'","amountMinor":5000,"merchant":"\u041c\u0430\u0433\u0430\u0437\u0438\u043d","description":"\u0412\u043e\u0437\u0432\u0440\u0430\u0442","occurredAt":"2026-10-04T14:20:00Z"}' "$api/financial-events/refunds")"
refund_id="$(printf '%s' "$refund" | json_value 'data.id')"
refund_receipt_status="$(curl --silent --output /dev/null --write-out '%{http_code}' -H 'Content-Type: application/json' -H 'Idempotency-Key: receipt-refund' -d '{"purchaseEventId":"'"$refund_id"'","seller":"\u041d\u0435 \u0447\u0435\u043a","issuedAt":"2026-10-04T14:20:00Z","items":[{"name":"\u0421\u0442\u0440\u043e\u043a\u0430","quantityMilli":1000,"unitPriceMinor":5000,"lineTotalMinor":5000}]}' "$api/receipts")"
test "$refund_receipt_status" = "422"

empty_status="$(curl --silent --output /dev/null --write-out '%{http_code}' -H 'Content-Type: application/json' -H 'Idempotency-Key: receipt-empty' -d '{"purchaseEventId":"'"$mismatch_purchase_id"'","seller":"\u041c\u0430\u0433\u0430\u0437\u0438\u043d","issuedAt":"2026-10-04T13:00:00Z","items":[]}' "$api/receipts")"
test "$empty_status" = "422"

missing_status="$(curl --silent --output /dev/null --write-out '%{http_code}' -H 'Content-Type: application/json' -H 'Idempotency-Key: receipt-missing' -d '{"purchaseEventId":"00000000-0000-0000-0000-000000000000","seller":"\u041c\u0430\u0433\u0430\u0437\u0438\u043d","issuedAt":"2026-10-04T13:00:00Z","items":[{"name":"\u0422\u043e\u0432\u0430\u0440","quantityMilli":1000,"unitPriceMinor":10000,"lineTotalMinor":10000}]}' "$api/receipts")"
test "$missing_status" = "422"

concurrent_purchase="$(curl --fail --silent -H 'Content-Type: application/json' -H 'Idempotency-Key: receipt-purchase-concurrent' -d '{"accountId":"10000000-0000-0000-0000-000000000001","categoryId":"20000000-0000-0000-0000-000000000001","amountMinor":5000,"merchant":"\u041f\u0430\u0440\u0430\u043b\u043b\u0435\u043b\u044c\u043d\u044b\u0439 \u0447\u0435\u043a","description":"\u041a\u043e\u043d\u043a\u0443\u0440\u0435\u043d\u0442\u043d\u0430\u044f \u0437\u0430\u043f\u0438\u0441\u044c","occurredAt":"2026-10-04T15:00:00Z"}' "$api/financial-events/purchases")"
concurrent_purchase_id="$(printf '%s' "$concurrent_purchase" | json_value 'data.id')"
concurrent_payload='{"purchaseEventId":"'"$concurrent_purchase_id"'","seller":"\u041f\u0430\u0440\u0430\u043b\u043b\u0435\u043b\u044c\u043d\u044b\u0439 \u0447\u0435\u043a","issuedAt":"2026-10-04T15:00:00Z","items":[{"name":"\u0422\u043e\u0432\u0430\u0440","quantityMilli":1000,"unitPriceMinor":5000,"lineTotalMinor":5000}]}'
curl --silent --output /dev/null --write-out '%{http_code}' -H 'Content-Type: application/json' -H 'Idempotency-Key: receipt-concurrent-a' -d "$concurrent_payload" "$api/receipts" >"$result_dir/a.status" &
first_pid=$!
curl --silent --output /dev/null --write-out '%{http_code}' -H 'Content-Type: application/json' -H 'Idempotency-Key: receipt-concurrent-b' -d "$concurrent_payload" "$api/receipts" >"$result_dir/b.status" &
second_pid=$!
wait "$first_pid"
wait "$second_pid"
concurrent_statuses="$(sort "$result_dir/a.status" "$result_dir/b.status" | tr '\n' ' ')"
test "$concurrent_statuses" = "201 409 "

# Confirmed receipt data makes schema rollback refuse instead of losing data.
if docker compose run --rm migrate -path=/migrations -database="$database_url" goto 3 >/dev/null 2>&1; then
  echo "receipt migration rollback unexpectedly removed confirmed data" >&2
  exit 1
fi

if go list -deps ./cmd/api | grep -Eiq '/(jeveval|routereval)(/|$)'; then
  echo "evaluation dependency found in production API dependency graph" >&2
  exit 1
fi

echo "SPEC-0002 receipt integration checks passed"
