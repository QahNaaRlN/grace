#!/bin/sh
# SPEC-0050: encrypts verified dumps with the owner's age public key and uploads
# them to Object Storage. The VPS holds no private key and the bucket credentials
# may only put objects, so the upload is checked by the PUT ETag (MD5), not by
# reading the object back.
set -eu

if [ "$#" -lt 1 ]; then
  echo "usage: $0 BACKUP_FILE... | --pending" >&2
  exit 2
fi

grace_root="${GRACE_ROOT:-/opt/grace}"
backups_dir="$grace_root/backups"
offsite_env="${GRACE_OFFSITE_ENV:-$grace_root/shared/secrets/offsite.env}"
failure_file="$backups_dir/.offsite-failure"

if [ ! -f "$offsite_env" ]; then
  echo "warning: off-site backup is not configured ($offsite_env is missing); copies stay on this VPS only" >&2
  exit 0
fi
case "$(stat -c '%a' "$offsite_env")" in
  400|600) ;;
  *) echo "off-site environment must have mode 400 or 600" >&2; exit 1 ;;
esac
set -a
# shellcheck source=/dev/null
. "$offsite_env"
set +a
for name in GRACE_OFFSITE_ENDPOINT GRACE_OFFSITE_REGION GRACE_OFFSITE_BUCKET \
  GRACE_OFFSITE_ACCESS_KEY_ID GRACE_OFFSITE_SECRET_ACCESS_KEY GRACE_OFFSITE_AGE_RECIPIENT; do
  eval "value=\${$name:-}"
  if [ -z "$value" ]; then
    echo "off-site environment is missing $name" >&2
    exit 1
  fi
done
prefix="${GRACE_OFFSITE_PREFIX:-grace}"

# rclone reads the remote from the environment; nothing is written to a config file.
RCLONE_CONFIG_OFFSITE_TYPE=s3
RCLONE_CONFIG_OFFSITE_PROVIDER=Other
RCLONE_CONFIG_OFFSITE_ENDPOINT="$GRACE_OFFSITE_ENDPOINT"
RCLONE_CONFIG_OFFSITE_REGION="$GRACE_OFFSITE_REGION"
RCLONE_CONFIG_OFFSITE_ACCESS_KEY_ID="$GRACE_OFFSITE_ACCESS_KEY_ID"
RCLONE_CONFIG_OFFSITE_SECRET_ACCESS_KEY="$GRACE_OFFSITE_SECRET_ACCESS_KEY"
export RCLONE_CONFIG_OFFSITE_TYPE RCLONE_CONFIG_OFFSITE_PROVIDER RCLONE_CONFIG_OFFSITE_ENDPOINT \
  RCLONE_CONFIG_OFFSITE_REGION RCLONE_CONFIG_OFFSITE_ACCESS_KEY_ID RCLONE_CONFIG_OFFSITE_SECRET_ACCESS_KEY

put() {
  # A single PUT (cutoff 5 GiB) returns the MD5 as its ETag, which rclone checks.
  rclone copyto --no-check-dest --s3-no-check-bucket --s3-no-head \
    --s3-upload-cutoff 5G --retries 3 --low-level-retries 3 --stats 0 \
    "$1" "offsite:$GRACE_OFFSITE_BUCKET/$prefix/$2"
}

fail() {
  printf 'stage=%s\nfile=%s\ntime=%s\n' "$1" "$2" "$(date -u +%Y-%m-%dT%H:%M:%SZ)" > "$failure_file"
  echo "off-site upload failed at $1: $2" >&2
  exit 1
}

upload() {
  dump="$1"
  name="$(basename "$dump")"
  encrypted="$dump.age"
  umask 077
  age -r "$GRACE_OFFSITE_AGE_RECIPIENT" -o "$encrypted" "$dump" || fail encrypt "$name"
  sha256sum "$encrypted" | sed "s|  .*|  $name.age|" > "$encrypted.sha256"
  put "$encrypted" "$name.age" >/dev/null 2>&1 || { rm -f "$encrypted" "$encrypted.sha256"; fail upload "$name"; }
  put "$encrypted.sha256" "$name.age.sha256" >/dev/null 2>&1 || { rm -f "$encrypted" "$encrypted.sha256"; fail upload "$name"; }
  echo "off-site copy uploaded: $name.age sha256=$(cut -d' ' -f1 "$encrypted.sha256")"
  rm -f "$encrypted" "$encrypted.sha256"
  touch "$dump.uploaded"
}

if [ "$1" = "--pending" ]; then
  set --
  for dump in "$backups_dir"/*.dump; do
    [ -f "$dump" ] && [ ! -f "$dump.uploaded" ] && set -- "$@" "$dump"
  done
fi

for dump in "$@"; do
  case "$dump" in
    "$backups_dir"/*.dump) [ -f "$dump" ] || { echo "no such backup: $dump" >&2; exit 2; } ;;
    *) echo "not a Grace backup: $dump" >&2; exit 2 ;;
  esac
  upload "$dump"
done
rm -f "$failure_file"
