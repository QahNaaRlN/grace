#!/bin/sh
set -eu

if [ "$#" -lt 1 ] || [ "$#" -gt 2 ]; then
  echo "usage: $0 BASE_URL [CURL_RESOLVE]" >&2
  exit 2
fi
base_url="${1%/}"
resolve="${2:-}"
tmp_dir="$(mktemp -d)"
trap 'rm -f "$tmp_dir/headers" "$tmp_dir/body"; rmdir "$tmp_dir"' EXIT INT TERM

request() {
  if [ -n "$resolve" ]; then
    curl --resolve "$resolve" "$@"
  else
    curl "$@"
  fi
}

check_route() {
  route="$1"
  expected_status="$2"
  expected_type="$3"
  expected_body="$4"
  status="$(request --silent --show-error --max-time 20 \
    --dump-header "$tmp_dir/headers" --output "$tmp_dir/body" \
    --write-out '%{http_code}' "$base_url$route")"
  if [ "$status" != "$expected_status" ]; then
    echo "$route: expected $expected_status, received $status" >&2
    exit 1
  fi
  if grep -Eiq '^WWW-Authenticate:.*Basic' "$tmp_dir/headers"; then
    echo "$route: unexpected browser password challenge" >&2
    exit 1
  fi
  grep -Eiq "^Content-Type: $expected_type" "$tmp_dir/headers"
  grep -Eq "$expected_body" "$tmp_dir/body"
  echo "$route: $status, no Basic challenge"
}

# Fresh anonymous requests: no cookies, Basic credentials or invitation token.
check_route / 200 text/html 'id="root"'
check_route /login 200 text/html 'id="root"'
check_route /api/meta 200 application/json '"service"[[:space:]]*:[[:space:]]*"grace"'
check_route /api/auth/me 401 application/json '"code"[[:space:]]*:[[:space:]]*"unauthenticated"'
check_route /api/v1/accounts 401 application/json '"code"[[:space:]]*:[[:space:]]*"unauthenticated"'
