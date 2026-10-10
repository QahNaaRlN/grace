#!/bin/sh
set -eu

if [ "$#" -ne 1 ]; then
  echo "usage: $0 OUTPUT_DIRECTORY" >&2
  exit 2
fi

repo_root="$(CDPATH= cd -- "$(dirname "$0")/.." && pwd)"
output="$1"

mkdir -p "$output"
if [ -n "$(find "$output" -mindepth 1 -print -quit)" ]; then
  echo "output directory must be empty: $output" >&2
  exit 1
fi

mkdir -p "$output/migrations"
cp "$repo_root/deployments/production/compose.yaml" "$output/compose.yaml"
cp "$repo_root/deployments/production/compose.ai.yaml" "$output/compose.ai.yaml"
cp "$repo_root/deployments/production/Caddyfile" "$output/Caddyfile"
cp "$repo_root/deployments/production/deploy.sh" "$output/deploy.sh"
cp "$repo_root/scripts/check-public-entrypoint.sh" "$output/check-public-entrypoint.sh"
cp "$repo_root/deployments/production/backup.sh" "$output/backup.sh"
cp "$repo_root/deployments/production/restore-drill.sh" "$output/restore-drill.sh"
cp "$repo_root/deployments/production/grace-backup.service" "$output/grace-backup.service"
cp "$repo_root/deployments/production/grace-backup.timer" "$output/grace-backup.timer"
cp "$repo_root/deployments/production/grace-firewall.nft" "$output/grace-firewall.nft"
cp "$repo_root/deployments/production/install-firewall.sh" "$output/install-firewall.sh"
cp "$repo_root/deployments/production/grace-firewall.service" "$output/grace-firewall.service"

for migration in "$repo_root"/migrations/*.sql; do
  case "$(basename "$migration")" in
    000002_development_seed.*)
      ;;
    *)
      cp "$migration" "$output/migrations/"
      ;;
  esac
done

cp "$repo_root"/deployments/production/migrations/000002_reference_categories.* \
  "$output/migrations/"

chmod 0755 "$output/deploy.sh" "$output/check-public-entrypoint.sh" "$output/backup.sh" "$output/restore-drill.sh" \
  "$output/install-firewall.sh"
