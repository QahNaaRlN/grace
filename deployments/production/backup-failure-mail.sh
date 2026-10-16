#!/bin/sh
# SPEC-0050: tells the owner that the daily backup or its off-site copy failed.
# The message names the stage, file and time only: no secrets, no financial data.
set -eu

grace_root="${GRACE_ROOT:-/opt/grace}"
secrets_dir="$grace_root/shared/secrets"
email_env="$secrets_dir/email.env"
offsite_env="${GRACE_OFFSITE_ENV:-$secrets_dir/offsite.env}"
failure_file="$grace_root/backups/.offsite-failure"

for file in "$email_env" "$offsite_env"; do
  if [ ! -f "$file" ]; then
    echo "cannot send backup alert: $file is missing" >&2
    exit 1
  fi
  case "$(stat -c '%a' "$file")" in
    400|600) ;;
    *) echo "cannot send backup alert: $file must have mode 400 or 600" >&2; exit 1 ;;
  esac
done
set -a
# shellcheck source=/dev/null
. "$email_env"
# shellcheck source=/dev/null
. "$offsite_env"
set +a
if [ -z "${GRACE_OFFSITE_ALERT_TO:-}" ] || [ -z "${GRACE_SMTP_HOST:-}" ]; then
  echo "cannot send backup alert: recipient or SMTP host is not configured" >&2
  exit 1
fi

stage="backup"
file="unknown"
if [ -f "$failure_file" ]; then
  stage="$(sed -n 's/^stage=//p' "$failure_file")"
  file="$(sed -n 's/^file=//p' "$failure_file")"
fi
from_address="$(printf '%s' "$GRACE_SMTP_FROM" | sed 's/.*<\(.*\)>.*/\1/')"
# shellcheck disable=SC2153 # GRACE_SMTP_* come from email.env.
case "${GRACE_SMTP_SECURITY:-starttls}" in
  tls) url="smtps://$GRACE_SMTP_HOST:$GRACE_SMTP_PORT" ;;
  *) url="smtp://$GRACE_SMTP_HOST:$GRACE_SMTP_PORT" ;;
esac

tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT
umask 077
# Credentials go through a private curl config, never the process arguments.
printf 'user = "%s:%s"\n' "$GRACE_SMTP_USERNAME" "$GRACE_SMTP_PASSWORD" > "$tmp/curl.conf"
cat > "$tmp/message" <<EOF
From: $GRACE_SMTP_FROM
To: $GRACE_OFFSITE_ALERT_TO
Subject: Grace: backup failed ($stage)
Content-Type: text/plain; charset=utf-8

The Grace backup failed on $(hostname) at $(date -u +%Y-%m-%dT%H:%M:%SZ).
Stage: $stage
File: $file

The local dump is kept when only the off-site stage failed.
Check: journalctl -u grace-backup.service
Retry the upload: /opt/grace/current/backup.sh upload-pending
EOF
curl --silent --show-error --ssl-reqd --url "$url" -K "$tmp/curl.conf" \
  --mail-from "$from_address" --mail-rcpt "$GRACE_OFFSITE_ALERT_TO" \
  --upload-file "$tmp/message"
echo "backup alert sent to the owner"
