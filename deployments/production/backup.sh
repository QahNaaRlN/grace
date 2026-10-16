#!/bin/sh
set -eu

label="${1:-manual}"
script_dir="$(CDPATH='' cd -- "$(dirname "$0")" && pwd)"
case "$label" in
  daily|manual) ;;
  upload-pending) exec "$script_dir/offsite-upload.sh" --pending ;;
  *)
    echo "usage: $0 [daily|manual|upload-pending]" >&2
    exit 2
    ;;
esac

grace_root="${GRACE_ROOT:-/opt/grace}"
compose_project="${GRACE_COMPOSE_PROJECT_NAME:-grace-production}"
current_dir="$(readlink -f "$grace_root/current")"
release_env="$current_dir/release.env"
backups_dir="$grace_root/backups"

if [ ! -f "$release_env" ]; then
  echo "no current Grace release" >&2
  exit 1
fi

postgres_password_file="$(sed -n 's/^GRACE_SECRETS_DIR=//p' "$release_env")/postgres_password"
POSTGRES_PASSWORD="$(cat "$postgres_password_file")"
export POSTGRES_PASSWORD

compose() {
  docker compose --project-name "$compose_project" --env-file "$release_env" \
    -f "$current_dir/compose.yaml" "$@"
}

mkdir -p "$backups_dir"
chmod 0700 "$backups_dir"
timestamp="$(date -u +%Y%m%dT%H%M%SZ)"
release_tag="$(sed -n 's/^RELEASE_TAG=//p' "$release_env")"
backup="$backups_dir/${label}-${timestamp}-${release_tag}.dump"

umask 077
compose exec -T postgres pg_dump -U grace -d grace -Fc > "$backup"
compose exec -T postgres pg_restore --list < "$backup" >/dev/null
chmod 0600 "$backup"

if [ "$label" = "daily" ]; then
  find "$backups_dir" -maxdepth 1 -type f -name 'daily-*.dump' -printf '%T@ %p\n' |
    sort -rn |
    sed -n '8,$p' |
    cut -d' ' -f2- |
    while IFS= read -r expired_backup; do
      case "$expired_backup" in
        "$backups_dir"/daily-*.dump) rm -f -- "$expired_backup" "$expired_backup.uploaded" ;;
      esac
    done
fi

echo "$backup"
# The local dump is complete either way; a failed upload fails the unit (SPEC-0050).
"$script_dir/offsite-upload.sh" "$backup"
