#!/bin/sh
set -eu

if [ "$#" -ne 1 ] || [ ! -f "$1" ]; then
  echo "usage: $0 BACKUP_FILE" >&2
  exit 2
fi

backup="$1"
grace_root="${GRACE_ROOT:-/opt/grace}"
compose_project="${GRACE_COMPOSE_PROJECT_NAME:-grace-production}"
current_dir="$(readlink -f "$grace_root/current")"
release_env="$current_dir/release.env"
drill_db="grace_restore_drill"

postgres_password_file="$(sed -n 's/^GRACE_SECRETS_DIR=//p' "$release_env")/postgres_password"
POSTGRES_PASSWORD="$(cat "$postgres_password_file")"
export POSTGRES_PASSWORD

compose() {
  docker compose --project-name "$compose_project" --env-file "$release_env" \
    -f "$current_dir/compose.yaml" "$@"
}

compose exec -T postgres dropdb -U grace --if-exists "$drill_db"
compose exec -T postgres createdb -U grace "$drill_db"
if ! compose exec -T postgres pg_restore -U grace --exit-on-error \
  --no-owner --dbname "$drill_db" < "$backup"; then
  compose exec -T postgres dropdb -U grace --if-exists "$drill_db"
  exit 1
fi
compose exec -T postgres psql -U grace -d "$drill_db" -v ON_ERROR_STOP=1 \
  -c 'SELECT 1;' >/dev/null
compose exec -T postgres dropdb -U grace "$drill_db"
echo "restore drill passed: $backup"
