#!/bin/sh
set -eu
repo_root="$(CDPATH= cd -- "$(dirname "$0")/.." && pwd)"
test_root="$(mktemp -d)"
trap 'rm -rf "$test_root"' EXIT INT TERM
mkdir -p "$test_root/previous" "$test_root/next" "$test_root/shared/secrets" "$test_root/mock-bin"
cp "$repo_root/deployments/production/deploy.sh" "$test_root/next/deploy.sh"
for release in previous next; do
 cp "$repo_root/deployments/production/compose.yaml" "$test_root/$release/compose.yaml"
 cp "$repo_root/deployments/production/compose.ai.yaml" "$test_root/$release/compose.ai.yaml"
 cat > "$test_root/$release/release.env" <<EOF
RELEASE_TAG=sha-0000000000000000000000000000000000000000
GRACE_SECRETS_DIR=$test_root/shared/secrets
GRACE_DOMAIN=grace.invalid
GRACE_HTTPS_PORT=443
EOF
done
printf 'synthetic-only' > "$test_root/shared/secrets/postgres_password"
printf "GRACE_AI_ENABLED=true\nGRACE_AI_ANALYSIS_POLICY='new-analysis'\nGRACE_AI_RECEIPT_POLICY='new-receipt'\n" > "$test_root/shared/secrets/ai.env"
chmod 0600 "$test_root/shared/secrets/ai.env"
printf '#!/bin/sh\nexit 0\n' > "$test_root/next/check-public-entrypoint.sh"
chmod 0755 "$test_root/next/check-public-entrypoint.sh"
ln -s "$test_root/previous" "$test_root/current"
cat > "$test_root/mock-bin/docker" <<'EOF'
#!/bin/sh
case "$*" in
 inspect*) printf 'GRACE_AUTH_MODE=password\nGRACE_AI_ENABLED=%s\nGRACE_AI_ANALYSIS_POLICY=old-analysis\nGRACE_AI_RECEIPT_POLICY=old-receipt\n' "$PREVIOUS_AI" ;;
 *'ps -q api'*) printf 'previous-api\n' ;;
 *'pull --policy'*)
   test "$GRACE_AI_ENABLED" = true
   case "$*" in *'compose.ai.yaml'*) ;; *) exit 44 ;; esac
   test "$FAIL_RELEASE" = false ;;
 *'information_schema.columns'*) printf 't\n' ;;
 *'NOT plan_active'*) printf '%s\n' "$HAS_HISTORY" ;;
 *'to_regclass'*) printf '%s\n' "$HAS_STATEMENT_SCHEMA" ;;
 *'FROM statement_sources'*) printf '%s\n' "$HAS_STATEMENT_DATA" ;;
 *'up -d --no-deps api web caddy'*)
   printf '%s\n%s\n%s\n' "$GRACE_AI_ENABLED" "$GRACE_AI_ANALYSIS_POLICY" "$GRACE_AI_RECEIPT_POLICY" > "$ROLLBACK_STATE_FILE" ;;
 *'pg_dump'*) printf 'synthetic dump' ;;
 *'pg_restore --list'*) cat >/dev/null ;;
 *'exec -T api /bin/sh'*) touch "$UID_CHECK_FILE" ;;
 *) exit 0 ;;
esac
EOF
chmod 0755 "$test_root/mock-bin/docker"
export GRACE_ROOT="$test_root" ROLLBACK_STATE_FILE="$test_root/rollback-state" UID_CHECK_FILE="$test_root/uid-check"
export PATH="$test_root/mock-bin:$PATH"
export FAIL_RELEASE=true HAS_HISTORY=f
for PREVIOUS_AI in false true; do
 export PREVIOUS_AI
 if sh "$test_root/next/deploy.sh" 0000000000000000000000000000000000000000 > "$test_root/output" 2>&1; then exit 1; fi
 test "$(sed -n '1p' "$ROLLBACK_STATE_FILE")" = "$PREVIOUS_AI"
 test "$(sed -n '2p' "$ROLLBACK_STATE_FILE")" = old-analysis
 test "$(sed -n '3p' "$ROLLBACK_STATE_FILE")" = old-receipt
 test "$(readlink "$test_root/current")" = "$test_root/previous"
done
rm "$ROLLBACK_STATE_FILE"
export HAS_HISTORY=t
if sh "$test_root/next/deploy.sh" 0000000000000000000000000000000000000000 > "$test_root/output" 2>&1; then exit 1; fi
grep -q 'Rollback refused' "$test_root/output"
test ! -e "$ROLLBACK_STATE_FILE"
# A previous image without PDF support must not hide retained source evidence.
mkdir -p "$test_root/next/migrations" "$test_root/previous/migrations"
touch "$test_root/next/migrations/000027_bank_statement_sources.up.sql" "$test_root/previous/migrations/000018_commitments.up.sql"
export HAS_STATEMENT_SCHEMA=t HAS_STATEMENT_DATA=t
if sh "$test_root/next/deploy.sh" 0000000000000000000000000000000000000000 > "$test_root/output" 2>&1; then exit 1; fi
grep -q 'previous image cannot review retained PDF sources' "$test_root/output"
test ! -e "$ROLLBACK_STATE_FILE"
export HAS_STATEMENT_DATA=f
if sh "$test_root/next/deploy.sh" 0000000000000000000000000000000000000000 > "$test_root/output" 2>&1; then exit 1; fi
test -f "$ROLLBACK_STATE_FILE"
rm "$ROLLBACK_STATE_FILE"
# Compatible previous PDF image is allowed to roll back with retained evidence.
touch "$test_root/previous/migrations/000027_bank_statement_sources.up.sql"
export HAS_STATEMENT_DATA=t
if sh "$test_root/next/deploy.sh" 0000000000000000000000000000000000000000 > "$test_root/output" 2>&1; then exit 1; fi
test -f "$ROLLBACK_STATE_FILE"
export FAIL_RELEASE=false
sh "$test_root/next/deploy.sh" 0000000000000000000000000000000000000000 > "$test_root/output" 2>&1
test -f "$UID_CHECK_FILE"
test "$(readlink "$test_root/current")" = "$test_root/next"
chmod 0644 "$test_root/shared/secrets/ai.env"
if sh "$test_root/next/deploy.sh" 0000000000000000000000000000000000000000 > "$test_root/output" 2>&1; then exit 1; fi
grep -q 'AI environment must have mode' "$test_root/output"
echo 'SPEC-0034 AI activation, non-root gate and compatible rollback checks passed'
