#!/bin/sh
set -eu
root="$(mktemp -d)"
trap 'rm -rf "$root"' EXIT INT TERM
mkdir -p "$root/release" "$root/shared/secrets" "$root/bin"
repo="$(CDPATH= cd -- "$(dirname "$0")/.." && pwd)"
cp "$repo/deployments/production/statement-ai.sh" "$root/release/statement-ai.sh"
printf 'RELEASE_TAG=sha-0000000000000000000000000000000000000000\n' > "$root/release/release.env"
cat > "$root/release/deploy.sh" <<'MOCK'
#!/bin/sh
printf 'deploy\n' >> "$CALLS"
test "$FAIL_DEPLOY" = false
MOCK
cat > "$root/bin/docker" <<'MOCK'
#!/bin/sh
case "$*" in
 'ps -q '*) echo synthetic-api ;;
 *'-mode=check'*) test "$ACTIVE" = true ;;
 *'-mode=qualify'*) printf 'qualify\n' >> "$CALLS"; echo '{"passed":true}'; test "$FAIL_QUALIFY" = false ;;
 *'-mode=summary'*) cat >/dev/null; echo synthetic-summary; test "$FAIL_SUMMARY" = false ;;
 *'-mode=policy'*) echo '{"model":"google/gemini-2.5-flash","provider":"google-vertex/eu"}' ;;
 *) exit 33 ;;
esac
MOCK
chmod 755 "$root/release/deploy.sh" "$root/bin/docker"
export PATH="$root/bin:$PATH" GRACE_ROOT="$root" CALLS="$root/calls"
export ACTIVE=false FAIL_DEPLOY=false FAIL_QUALIFY=true FAIL_SUMMARY=false
statement_env="$root/shared/secrets/statement.env"
if "$root/release/statement-ai.sh" enable > "$root/output" 2>&1; then exit 1; fi
test ! -e "$statement_env"
test "$(cat "$CALLS")" = qualify
printf 'GRACE_STATEMENT_AI_ENABLED=false\n' > "$statement_env"
chmod 600 "$statement_env"
cp "$statement_env" "$root/original"
export FAIL_QUALIFY=false FAIL_DEPLOY=true
if "$root/release/statement-ai.sh" enable > "$root/output" 2>&1; then exit 1; fi
cmp "$statement_env" "$root/original"
export FAIL_DEPLOY=false FAIL_SUMMARY=true
if "$root/release/statement-ai.sh" enable > "$root/output" 2>&1; then exit 1; fi
cmp "$statement_env" "$root/original"
export FAIL_SUMMARY=false
"$root/release/statement-ai.sh" enable > "$root/output" 2>&1
grep -qx 'GRACE_STATEMENT_AI_ENABLED=true' "$statement_env"
test "$(stat -c '%a' "$statement_env")" = 600
test "$(grep -c '^qualify$' "$CALLS")" -eq 4
export ACTIVE=true
before="$(wc -l < "$CALLS")"
"$root/release/statement-ai.sh" enable > "$root/output" 2>&1
test "$(wc -l < "$CALLS")" -eq "$before"
"$root/release/statement-ai.sh" disable > "$root/output" 2>&1
grep -qx 'GRACE_STATEMENT_AI_ENABLED=false' "$statement_env"
test "$(grep -c '^qualify$' "$CALLS")" -eq 4
echo 'Statement activation: failed qualification stays disabled, failed rollout restores config, replay makes no paid call, disable preserved.'
