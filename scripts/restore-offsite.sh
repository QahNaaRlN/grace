#!/bin/sh
# SPEC-0050 off-site restore drill, run on the owner's machine (Docker and a POSIX
# shell; Git Bash works). Downloads an encrypted copy with the owner's read-only
# key, checks its SHA-256, decrypts it with the owner's age identity and restores
# it into a throwaway PostgreSQL. Nothing is written to production.
set -eu

if [ "$#" -lt 2 ] || [ "$#" -gt 3 ]; then
  echo "usage: $0 OWNER_OFFSITE_ENV AGE_IDENTITY_FILE [OBJECT_NAME]" >&2
  echo "  without OBJECT_NAME the latest daily-*.dump.age is restored" >&2
  exit 2
fi
env_file="$1"
identity="$2"
object="${3:-}"
for file in "$env_file" "$identity"; do
  [ -f "$file" ] || { echo "no such file: $file" >&2; exit 2; }
done

# A file saved on Windows ends lines with CRLF; a stray \r would join every value.
env_clean="$(mktemp)"
trap 'rm -f "$env_clean"' EXIT
tr -d '\r' < "$env_file" > "$env_clean"
set -a
# shellcheck source=/dev/null
. "$env_clean"
set +a
rm -f "$env_clean"
prefix="${GRACE_OFFSITE_PREFIX:-grace}"
remote="offsite:$GRACE_OFFSITE_BUCKET/$prefix"
RCLONE_CONFIG_OFFSITE_TYPE=s3
RCLONE_CONFIG_OFFSITE_PROVIDER=Other
RCLONE_CONFIG_OFFSITE_ENDPOINT="$GRACE_OFFSITE_ENDPOINT"
RCLONE_CONFIG_OFFSITE_REGION="$GRACE_OFFSITE_REGION"
RCLONE_CONFIG_OFFSITE_ACCESS_KEY_ID="$GRACE_OFFSITE_ACCESS_KEY_ID"
RCLONE_CONFIG_OFFSITE_SECRET_ACCESS_KEY="$GRACE_OFFSITE_SECRET_ACCESS_KEY"
export RCLONE_CONFIG_OFFSITE_TYPE RCLONE_CONFIG_OFFSITE_PROVIDER RCLONE_CONFIG_OFFSITE_ENDPOINT \
  RCLONE_CONFIG_OFFSITE_REGION RCLONE_CONFIG_OFFSITE_ACCESS_KEY_ID RCLONE_CONFIG_OFFSITE_SECRET_ACCESS_KEY
# Container paths must reach Docker unchanged under Git Bash.
MSYS_NO_PATHCONV=1
export MSYS_NO_PATHCONV

tools_image="${GRACE_RESTORE_TOOLS_IMAGE:-alpine:3.20}"
postgres_image="${GRACE_RESTORE_POSTGRES_IMAGE:-postgres:16-alpine}"
network_args="${GRACE_RESTORE_NETWORK:+--network $GRACE_RESTORE_NETWORK}"
tools=""
postgres=""
cleanup() {
  [ -z "$tools" ] || docker rm -f "$tools" >/dev/null 2>&1 || true
  [ -z "$postgres" ] || docker rm -f "$postgres" >/dev/null 2>&1 || true
}
trap cleanup EXIT

# shellcheck disable=SC2086 # network_args is intentionally split into two words.
tools="$(docker run -d $network_args \
  -e RCLONE_CONFIG_OFFSITE_TYPE -e RCLONE_CONFIG_OFFSITE_PROVIDER -e RCLONE_CONFIG_OFFSITE_ENDPOINT \
  -e RCLONE_CONFIG_OFFSITE_REGION -e RCLONE_CONFIG_OFFSITE_ACCESS_KEY_ID \
  -e RCLONE_CONFIG_OFFSITE_SECRET_ACCESS_KEY "$tools_image" sleep 3600)"
# A stalled mirror must not hang the drill: bounded attempts.
docker exec "$tools" sh -c 'for attempt in 1 2 3; do
  timeout 120 apk add --no-cache --quiet age rclone && exit 0; done; exit 1'
# Keep only key lines: a copy of age-keygen's screen output starts with "Public key: …".
if ! tr -d '\r' < "$identity" | grep -q '^AGE-SECRET-KEY-'; then
  echo "no AGE-SECRET-KEY line in $identity" >&2
  exit 2
fi
tr -d '\r' < "$identity" | grep '^AGE-SECRET-KEY-' |
  docker exec -i "$tools" sh -c 'umask 077; mkdir /work; cat > /work/identity'

if [ -z "$object" ]; then
  object="$(docker exec "$tools" rclone lsf --include 'daily-*.dump.age' "$remote" | sort | tail -n 1)"
  [ -n "$object" ] || { echo "no daily copy found in the bucket" >&2; exit 1; }
fi
case "$object" in
  *.dump.age) ;;
  *) echo "object must be a *.dump.age copy: $object" >&2; exit 2 ;;
esac
echo "restoring $object"
docker exec "$tools" rclone copyto "$remote/$object" "/work/$object"
docker exec "$tools" rclone copyto "$remote/$object.sha256" "/work/$object.sha256"
docker exec -w /work "$tools" sha256sum -c "$object.sha256"
docker exec "$tools" age -d -i /work/identity -o /work/backup.dump "/work/$object"

postgres="$(docker run -d -e POSTGRES_USER=grace -e POSTGRES_PASSWORD=restore-drill -e POSTGRES_DB=grace "$postgres_image")"
tries=0
until docker exec "$postgres" pg_isready -U grace -d grace >/dev/null 2>&1; do
  tries=$((tries + 1))
  [ "$tries" -lt 60 ] || { echo "PostgreSQL did not start" >&2; exit 1; }
  sleep 1
done
# The init entrypoint restarts the server once; wait for the final instance.
sleep 2
until docker exec "$postgres" pg_isready -U grace -d grace >/dev/null 2>&1; do sleep 1; done
docker exec "$postgres" psql -U grace -d grace -v ON_ERROR_STOP=1 -q \
  -c 'CREATE ROLE grace_financial NOLOGIN;'
docker exec "$tools" cat /work/backup.dump |
  docker exec -i "$postgres" pg_restore -U grace -d grace --exit-on-error
docker exec "$postgres" psql -U grace -d grace -v ON_ERROR_STOP=1 -At -F ' ' -c "
  SELECT 'financial_events', count(*) FROM financial_events
  UNION ALL SELECT 'postings', count(*) FROM postings
  UNION ALL SELECT 'accounts', count(*) FROM accounts
  UNION ALL SELECT 'loan_documents', count(*) FROM loan_documents;"
echo "off-site restore drill passed: $object"
