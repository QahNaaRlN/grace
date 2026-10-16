#!/bin/sh
set -eu

repo_root="$(CDPATH= cd -- "$(dirname "$0")/.." && pwd)"
tmp_dir="$(mktemp -d)"

cleanup() {
  if [ -n "${proxy_container:-}" ]; then
    docker rm -f "$proxy_container" >/dev/null 2>&1 || true
  fi
  case "$tmp_dir" in
    /tmp/*) rm -rf "$tmp_dir" ;;
  esac
}
trap cleanup EXIT INT TERM

bundle="$tmp_dir/bundle"
secrets="$tmp_dir/secrets"
mkdir -p "$bundle" "$secrets"
printf '%s\n' 'test-postgres-password' > "$secrets/postgres_password"

"$repo_root/scripts/build-production-bundle.sh" "$bundle"
cp "$repo_root/deployments/production/release.env.example" "$bundle/release.env"

POSTGRES_PASSWORD=test-postgres-password GRACE_SECRETS_DIR="$secrets" docker compose \
  --env-file "$bundle/release.env" -f "$bundle/compose.yaml" config --quiet

sh -n "$bundle/deploy.sh"
sh -n "$bundle/backup.sh"
sh -n "$bundle/restore-drill.sh"
sh -n "$bundle/offsite-upload.sh"
sh -n "$bundle/backup-failure-mail.sh"
sh -n "$bundle/install-firewall.sh"
test -f "$bundle/grace-backup.service"
test -f "$bundle/grace-backup.timer"
test -f "$bundle/grace-backup-failure.service"
grep -q 'OnFailure=grace-backup-failure.service' "$bundle/grace-backup.service"
test -f "$bundle/grace-firewall.service"
test -f "$bundle/grace-firewall.nft"

grep -q 'pre-release-' "$bundle/deploy.sh"
grep -q "name 'daily-\\*.dump'" "$bundle/backup.sh"
grep -q '8,\$p' "$bundle/backup.sh"
grep -q 'udp dport { 443, 51820 } accept' "$bundle/grace-firewall.nft"
grep -q 'iifname "wg0" accept' "$bundle/grace-firewall.nft"

rotation_root="$tmp_dir/rotation-root"
rotation_bin="$tmp_dir/rotation-bin"
mkdir -p "$rotation_root/releases/test" "$rotation_root/backups" \
  "$rotation_bin"
ln -s "$rotation_root/releases/test" "$rotation_root/current"
cat > "$rotation_root/releases/test/release.env" <<EOF
GRACE_SECRETS_DIR=$secrets
RELEASE_TAG=sha-0000000000000000000000000000000000000000
EOF
cat > "$rotation_bin/docker" <<'EOF'
#!/bin/sh
case "$*" in
  *'pg_dump'*) printf 'test dump' ;;
  *'pg_restore --list'*) cat >/dev/null ;;
  *) exit 1 ;;
esac
EOF
chmod 0755 "$rotation_bin/docker"
for day in 01 02 03 04 05 06 07 08; do
  old_backup="$rotation_root/backups/daily-202609${day}T000000Z-sha-old.dump"
  printf 'old dump' > "$old_backup"
  touch -d "2026-09-$day 00:00:00 UTC" "$old_backup"
done
printf 'manual dump' > "$rotation_root/backups/manual-keep.dump"
GRACE_ROOT="$rotation_root" PATH="$rotation_bin:$PATH" "$bundle/backup.sh" daily \
  >/dev/null
test "$(find "$rotation_root/backups" -type f -name 'daily-*.dump' | wc -l)" -eq 7
test -f "$rotation_root/backups/manual-keep.dump"

test ! -e "$bundle/migrations/000002_development_seed.up.sql"
test -e "$bundle/migrations/000002_reference_categories.up.sql"
if grep -REn 'Основная карта|12500000' "$bundle/migrations"; then
  echo "production migrations contain development account data" >&2
  exit 1
fi

if grep -REn 'localhost:8080' "$repo_root/web/Dockerfile" \
  "$repo_root/web/src/shared/config"; then
  echo "production Web still embeds localhost API origin" >&2
  exit 1
fi

published_ports="$(POSTGRES_PASSWORD=test-postgres-password GRACE_SECRETS_DIR="$secrets" docker compose \
  --env-file "$bundle/release.env" -f "$bundle/compose.yaml" config \
  --format json | node -e '
    let input = "";
    process.stdin.on("data", chunk => input += chunk);
    process.stdin.on("end", () => {
      const config = JSON.parse(input);
      const apiNetworks = Object.keys(config.services.api.networks || {});
      if (!apiNetworks.some(name => config.networks[name]?.internal !== true)) {
        throw new Error("Production API cannot reach Telegram token/JWKS endpoints");
      }
      for (const [name, service] of Object.entries(config.services)) {
        if (service.ports) process.stdout.write(`${name}\n`);
      }
    });
  ')"
test "$published_ports" = "caddy"

# Exercise the production proxy routes with real Caddy and deterministic upstreams.
# Only replace upstream addresses and use HTTP on a private ephemeral test port.
sed -e 's/api:8080/127.0.0.1:18080/' -e 's/web:80/127.0.0.1:18081/' \
  "$bundle/Caddyfile" > "$tmp_dir/Caddyfile"
cat >> "$tmp_dir/Caddyfile" <<'EOF'

:18080 {
  header Content-Type application/json
  @meta path /api/meta
  respond @meta `{"service":"grace"}` 200
  respond `{"code":"unauthenticated","message":"Login required"}` 401
}
:18081 {
  header Content-Type text/html
  respond `<html><div id="root"></div></html>` 200
}
EOF
proxy_container="$(docker create -e GRACE_DOMAIN=:80 -p 127.0.0.1::80 caddy:2-alpine)"
docker cp "$tmp_dir/Caddyfile" "$proxy_container:/etc/caddy/Caddyfile"
docker start "$proxy_container" >/dev/null
proxy_port="$(docker inspect --format '{{(index (index .NetworkSettings.Ports "80/tcp") 0).HostPort}}' "$proxy_container")"
attempt=0
until curl --silent --fail "http://127.0.0.1:$proxy_port/api/meta" >/dev/null; do
  attempt=$((attempt + 1))
  if [ "$attempt" -ge 20 ]; then
    echo "test Caddy did not become ready" >&2; exit 1
  fi
  sleep 1
done
"$bundle/check-public-entrypoint.sh" "http://127.0.0.1:$proxy_port"
# Exercise release promotion against real public Caddy. Docker lifecycle and
# backup are mocked; HTTP assertions run through the bundled checker.
promotion_root="$tmp_dir/promotion-root"
promotion_bin="$tmp_dir/promotion-bin"
mkdir -p "$promotion_root" "$promotion_bin"
cat > "$bundle/release.env" <<EOF
GRACE_DOMAIN=entrypoint.example
GRACE_HTTPS_PORT=443
GRACE_SECRETS_DIR=$secrets
RELEASE_TAG=sha-0000000000000000000000000000000000000000
EOF
cat > "$promotion_bin/docker" <<'EOF'
#!/bin/sh
case "$*" in *'pg_dump'*) printf 'synthetic backup' ;; esac
exit 0
EOF
cat > "$promotion_bin/curl" <<'EOF'
#!/bin/bash
args=()
while (( $# )); do
  case "$1" in
    --resolve) shift 2 ;;
    https://entrypoint.example:443/*)
      args+=("$PUBLIC_CHECK_BASE${1#https://entrypoint.example:443}"); shift ;;
    *) args+=("$1"); shift ;;
  esac
done
exec "$REAL_CURL" "${args[@]}"
EOF
chmod 0755 "$promotion_bin/docker" "$promotion_bin/curl"
real_curl="$(command -v curl)"
GRACE_ROOT="$promotion_root" REAL_CURL="$real_curl" \
  PUBLIC_CHECK_BASE="http://127.0.0.1:$proxy_port" PATH="$promotion_bin:$PATH" \
  "$bundle/deploy.sh" 0000000000000000000000000000000000000000 \
  >"$tmp_dir/promotion.log" 2>&1
test "$(readlink "$promotion_root/current")" = "$bundle"
grep -q 'release sha-.* is healthy' "$tmp_dir/promotion.log"
# Release 090ebad left migration 22 dirty after a rolled-back failure: the next
# deployment clears the mark, but never when any part of 22 is present.
cat > "$promotion_bin/docker" <<'EOF'
#!/bin/sh
case "$*" in
  *'pg_dump'*) printf 'synthetic backup' ;;
  *'FROM schema_migrations'*) printf '22:true\n' ;;
  *'information_schema.columns'*) printf '%s\n' "$APPLIED_COLUMNS" ;;
  *'force 21'*) touch "$FORCE_MARKER" ;;
esac
exit 0
EOF
# Fresh roots: with a current release the mock Docker has no previous API.
mkdir -p "$tmp_dir/dirty-root" "$tmp_dir/partial-root"
GRACE_ROOT="$tmp_dir/dirty-root" REAL_CURL="$real_curl" APPLIED_COLUMNS=0 FORCE_MARKER="$tmp_dir/forced" \
  PUBLIC_CHECK_BASE="http://127.0.0.1:$proxy_port" PATH="$promotion_bin:$PATH" \
  "$bundle/deploy.sh" 0000000000000000000000000000000000000000 \
  >"$tmp_dir/dirty.log" 2>&1 || { cat "$tmp_dir/dirty.log" >&2; exit 1; }
test -f "$tmp_dir/forced"
grep -q 'clearing its dirty mark' "$tmp_dir/dirty.log"
rm -f "$tmp_dir/forced"
if GRACE_ROOT="$tmp_dir/partial-root" REAL_CURL="$real_curl" APPLIED_COLUMNS=1 FORCE_MARKER="$tmp_dir/forced" \
  PUBLIC_CHECK_BASE="http://127.0.0.1:$proxy_port" PATH="$promotion_bin:$PATH" \
  "$bundle/deploy.sh" 0000000000000000000000000000000000000000 \
  >"$tmp_dir/partial.log" 2>&1; then
  echo "deployment forced a partly applied migration" >&2; exit 1
fi
test ! -e "$tmp_dir/forced"
grep -q 'operator review required' "$tmp_dir/partial.log" || { cat "$tmp_dir/partial.log" >&2; exit 1; }
docker rm -f "$proxy_container" >/dev/null
proxy_container=""

# Exercise credential loading before a mock Docker stops the deployment.
credential_root="$tmp_dir/credential-root"
credential_bin="$tmp_dir/credential-bin"
mkdir -p "$credential_root/shared/secrets" "$credential_bin"
cp "$bundle/deploy.sh" "$credential_root/deploy.sh"
cat > "$credential_root/release.env" <<EOF
GRACE_SECRETS_DIR=$secrets
RELEASE_TAG=sha-0000000000000000000000000000000000000000
EOF
cat > "$credential_root/shared/secrets/telegram.env" <<'EOF'
GRACE_AUTH_MODE=telegram
TELEGRAM_CLIENT_ID=test-client
TELEGRAM_CLIENT_SECRET=test-only-credential
EOF
cat > "$credential_bin/docker" <<'EOF'
#!/bin/sh
case "$*" in
  *'pull --policy'*)
    test "${GRACE_AUTH_MODE:-}" = telegram || exit 43
    test "${TELEGRAM_CLIENT_ID:-}" = test-client || exit 43
    test "${TELEGRAM_CLIENT_SECRET:-}" = test-only-credential || exit 43
    touch "$CREDENTIAL_CHECK_MARKER"
    exit 42 ;;
  *) exit 0 ;;
esac
EOF
chmod 0755 "$credential_bin/docker"
chmod 0644 "$credential_root/shared/secrets/telegram.env"
if GRACE_ROOT="$credential_root" PATH="$credential_bin:$PATH" "$credential_root/deploy.sh" \
  0000000000000000000000000000000000000000 >"$tmp_dir/credential-error" 2>&1; then
  echo "deployment accepted a public credential file" >&2; exit 1
fi
grep -q 'Telegram environment must have mode 400 or 600' "$tmp_dir/credential-error"
test ! -e "$tmp_dir/credential-loaded"
chmod 0600 "$credential_root/shared/secrets/telegram.env"
if GRACE_ROOT="$credential_root" CREDENTIAL_CHECK_MARKER="$tmp_dir/credential-loaded" \
  PATH="$credential_bin:$PATH" "$credential_root/deploy.sh" \
  0000000000000000000000000000000000000000 >"$tmp_dir/credential-output" 2>&1; then
  echo "mock deployment unexpectedly succeeded" >&2; exit 1
fi
test -f "$tmp_dir/credential-loaded"
if grep -q 'test-only-credential' "$tmp_dir/credential-output"; then
  echo "deployment leaked credentials" >&2; exit 1
fi

sh "$repo_root/scripts/test-email-rollout.sh"
echo "SPEC-0012 production deployment checks passed"
