#!/bin/sh
set -eu
repo_root="$(CDPATH= cd -- "$(dirname "$0")/.." && pwd)"
test_root="$(mktemp -d)"
trap 'rm -rf "$test_root"' EXIT INT TERM
mkdir -p "$test_root/previous" "$test_root/next" "$test_root/shared/secrets" "$test_root/mock-bin"
cp "$repo_root/deployments/production/deploy.sh" "$test_root/next/deploy.sh"
cp "$repo_root/deployments/production/compose.yaml" "$test_root/previous/compose.yaml"
cp "$repo_root/deployments/production/compose.yaml" "$test_root/next/compose.yaml"
for release in previous next; do
  cat > "$test_root/$release/release.env" <<EOF
RELEASE_TAG=sha-0000000000000000000000000000000000000000
GRACE_SECRETS_DIR=$test_root/shared/secrets
EOF
done
printf 'synthetic-only' > "$test_root/shared/secrets/postgres_password"
printf 'GRACE_AUTH_MODE=email\n' > "$test_root/shared/secrets/email.env"
chmod 0600 "$test_root/shared/secrets/email.env"
ln -s "$test_root/previous" "$test_root/current"
cat > "$test_root/mock-bin/docker" <<'EOF'
#!/bin/sh
case "$*" in
  inspect*) printf 'GRACE_AUTH_MODE=%s\n' "$PREVIOUS_AUTH_MODE" ;;
  *'ps -q api'*) printf 'previous-api\n' ;;
  *'pull --policy'*) exit 1 ;;
  *'up -d --no-deps api web caddy'*) printf '%s\n' "$GRACE_AUTH_MODE" > "$ROLLBACK_MODE_FILE" ;;
  *) echo "unexpected mock Docker call" >&2; exit 1 ;;
esac
EOF
chmod 0755 "$test_root/mock-bin/docker"
for previous_mode in telegram email password locked; do
if GRACE_ROOT="$test_root" PREVIOUS_AUTH_MODE="$previous_mode" ROLLBACK_MODE_FILE="$test_root/rollback-mode" \
  PATH="$test_root/mock-bin:$PATH" sh "$test_root/next/deploy.sh" \
  0000000000000000000000000000000000000000 > "$test_root/output" 2>&1; then
  echo "failing deployment unexpectedly succeeded" >&2; exit 1
fi
test "$(cat "$test_root/rollback-mode")" = "$previous_mode"
test "$(readlink "$test_root/current")" = "$test_root/previous"
done
echo "SPEC-0022 email rollout rollback preserves previous authentication mode"
