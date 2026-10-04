#!/bin/sh
set -eu

repo_root="$(CDPATH= cd -- "$(dirname "$0")/.." && pwd)"
tmp_dir="$(mktemp -d)"

cleanup() {
  case "$tmp_dir" in
    /tmp/*) rm -rf "$tmp_dir" ;;
  esac
}
trap cleanup EXIT INT TERM

bundle="$tmp_dir/bundle"
secrets="$tmp_dir/secrets"
mkdir -p "$bundle" "$secrets"
printf '%s\n' 'test-postgres-password' > "$secrets/postgres_password"
printf '%s\n' '$2a$14$test-only-placeholder-hash' > "$secrets/basic_auth_hash"

"$repo_root/scripts/build-production-bundle.sh" "$bundle"
cp "$repo_root/deployments/production/release.env.example" "$bundle/release.env"

POSTGRES_PASSWORD=test-postgres-password GRACE_SECRETS_DIR="$secrets" docker compose \
  --env-file "$bundle/release.env" -f "$bundle/compose.yaml" config --quiet

sh -n "$bundle/deploy.sh"
sh -n "$bundle/backup.sh"
sh -n "$bundle/restore-drill.sh"
sh -n "$bundle/install-firewall.sh"
test -f "$bundle/grace-backup.service"
test -f "$bundle/grace-backup.timer"
test -f "$bundle/grace-firewall.service"
test -f "$bundle/grace-firewall.nft"

rg -q 'pre-release-' "$bundle/deploy.sh"
rg -q "name 'daily-\\*.dump'" "$bundle/backup.sh"
rg -q '8,\$p' "$bundle/backup.sh"
rg -q 'udp dport \{ 443, 51820 \} accept' "$bundle/grace-firewall.nft"
rg -q 'iifname "wg0" accept' "$bundle/grace-firewall.nft"

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
if rg -n 'Основная карта|12500000' "$bundle/migrations"; then
  echo "production migrations contain development account data" >&2
  exit 1
fi

if rg -n 'localhost:8080' "$repo_root/web/Dockerfile" \
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
      for (const [name, service] of Object.entries(config.services)) {
        if (service.ports) process.stdout.write(`${name}\n`);
      }
    });
  ')"
test "$published_ports" = "caddy"

echo "SPEC-0012 production deployment checks passed"
