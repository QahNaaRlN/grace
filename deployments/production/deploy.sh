#!/bin/sh
set -eu

if [ "$#" -ne 1 ]; then
  echo "usage: $0 FULL_COMMIT_SHA" >&2
  exit 2
fi

release_sha="$1"
case "$release_sha" in
  *[!0-9a-f]*|'')
    echo "release SHA must be lowercase hexadecimal" >&2
    exit 1
    ;;
esac
if [ "${#release_sha}" -ne 40 ]; then
  echo "release SHA must contain 40 characters" >&2
  exit 1
fi

grace_root="${GRACE_ROOT:-/opt/grace}"
compose_project="${GRACE_COMPOSE_PROJECT_NAME:-grace-production}"
release_dir="$(CDPATH= cd -- "$(dirname "$0")" && pwd)"
release_env="$release_dir/release.env"
backups_dir="$grace_root/backups"
current_link="$grace_root/current"
previous_link="$grace_root/previous"
previous_dir=""
rollback_auth_mode="locked"
rollback_ai_enabled="false"
rollback_analysis_policy=""
rollback_receipt_policy=""
rollback_statement_enabled="false"
rollback_statement_policy=""
ai_configured="false"

if [ ! -f "$release_env" ]; then
  echo "missing release environment: $release_env" >&2
  exit 1
fi
if ! grep -qx "RELEASE_TAG=sha-$release_sha" "$release_env"; then
  echo "release.env does not match requested SHA" >&2
  exit 1
fi

# Operator configuration survives releases; credentials never enter the bundle.
telegram_env="$grace_root/shared/secrets/telegram.env"
if [ -f "$telegram_env" ]; then
  case "$(stat -c '%a' "$telegram_env")" in
    400|600) ;;
    *) echo "Telegram environment must have mode 400 or 600" >&2; exit 1 ;;
  esac
  telegram_owner="$(stat -c '%u' "$telegram_env")"
  if [ "$telegram_owner" != "$(id -u)" ] && [ "$telegram_owner" != "0" ]; then
    echo "Telegram environment must belong to the deployment operator or root" >&2
    exit 1
  fi
  set -a
  . "$telegram_env"
  set +a
fi

# Email configuration overrides the temporary Telegram method only when installed by the operator.
email_env="$grace_root/shared/secrets/email.env"
if [ -f "$email_env" ]; then
  case "$(stat -c '%a' "$email_env")" in 400|600) ;; *) echo "Email environment must have mode 400 or 600" >&2; exit 1 ;; esac
  email_owner="$(stat -c '%u' "$email_env")"
  if [ "$email_owner" != "$(id -u)" ] && [ "$email_owner" != "0" ]; then echo "Email environment must belong to the deployment operator or root" >&2; exit 1; fi
  set -a
  . "$email_env"
  set +a
fi

ai_env="$grace_root/shared/secrets/ai.env"
if [ -f "$ai_env" ]; then
  case "$(stat -c '%a' "$ai_env")" in 400|600) ;; *) echo "AI environment must have mode 400 or 600" >&2; exit 1 ;; esac
  ai_owner="$(stat -c '%u' "$ai_env")"
  if [ "$ai_owner" != "$(id -u)" ] && [ "$ai_owner" != "0" ]; then echo "AI environment must belong to the deployment operator or root" >&2; exit 1; fi
  set -a
  . "$ai_env"
  set +a
  case "${GRACE_AI_ENABLED:-false}" in true|false) ;; *) echo "Invalid AI enable setting" >&2; exit 1 ;; esac
  if [ ! -f "$release_dir/compose.ai.yaml" ]; then echo "Missing AI override" >&2; exit 1; fi
  ai_configured="true"
fi

# Independent statement activation survives releases without changing receipt/analysis.
statement_env="$grace_root/shared/secrets/statement.env"
if [ -f "$statement_env" ]; then
  case "$(stat -c '%a' "$statement_env")" in 400|600) ;; *) echo "Statement environment must have mode 400 or 600" >&2; exit 1 ;; esac
  statement_owner="$(stat -c '%u' "$statement_env")"
  if [ "$statement_owner" != "$(id -u)" ] && [ "$statement_owner" != "0" ]; then echo "Statement environment must belong to operator or root" >&2; exit 1; fi
  set -a
  . "$statement_env"
  set +a
fi
case "${GRACE_STATEMENT_AI_ENABLED:-false}" in true|false) ;; *) echo "Invalid statement enable setting" >&2; exit 1 ;; esac
if [ "${GRACE_STATEMENT_AI_ENABLED:-false}" = true ] && { [ "$ai_configured" != true ] || [ "${GRACE_AI_ENABLED:-false}" != true ] || [ -z "${GRACE_STATEMENT_AI_POLICY:-}" ]; }; then
  echo "Statement activation requires configured global AI and pinned statement policy" >&2; exit 1
fi

postgres_password_file="$(sed -n 's/^GRACE_SECRETS_DIR=//p' "$release_env")/postgres_password"
if [ ! -r "$postgres_password_file" ]; then
  echo "PostgreSQL password file is not readable" >&2
  exit 1
fi
POSTGRES_PASSWORD="$(cat "$postgres_password_file")"
export POSTGRES_PASSWORD

compose_release() {
  if [ "$ai_configured" = "true" ]; then
    docker compose --project-name "$compose_project" --env-file "$release_env" \
      -f "$release_dir/compose.yaml" -f "$release_dir/compose.ai.yaml" "$@"
    return
  fi
  docker compose --project-name "$compose_project" --env-file "$release_env" \
    -f "$release_dir/compose.yaml" "$@"
}

if [ -L "$current_link" ]; then
  previous_dir="$(readlink -f "$current_link")"
  previous_api="$(docker compose --project-name "$compose_project" \
    --env-file "$previous_dir/release.env" -f "$previous_dir/compose.yaml" ps -q api)"
  if [ -n "$previous_api" ]; then
    previous_mode="$(docker inspect --format '{{range .Config.Env}}{{println .}}{{end}}' "$previous_api" \
      | sed -n 's/^GRACE_AUTH_MODE=//p')"
    case "$previous_mode" in
      telegram|email|password|locked) rollback_auth_mode="$previous_mode" ;;
      *) echo "cannot determine previous production authentication mode" >&2; exit 1 ;;
    esac
    previous_ai="$(docker inspect --format '{{range .Config.Env}}{{println .}}{{end}}' "$previous_api" | sed -n 's/^GRACE_AI_ENABLED=//p')"
    case "$previous_ai" in true) rollback_ai_enabled="true" ;; false|'') rollback_ai_enabled="false" ;; *) echo "Cannot snapshot previous AI setting" >&2; exit 1 ;; esac
    rollback_analysis_policy="$(docker inspect --format '{{range .Config.Env}}{{println .}}{{end}}' "$previous_api" | sed -n 's/^GRACE_AI_ANALYSIS_POLICY=//p')"
    rollback_receipt_policy="$(docker inspect --format '{{range .Config.Env}}{{println .}}{{end}}' "$previous_api" | sed -n 's/^GRACE_AI_RECEIPT_POLICY=//p')"
    previous_statement="$(docker inspect --format '{{range .Config.Env}}{{println .}}{{end}}' "$previous_api" | sed -n 's/^GRACE_STATEMENT_AI_ENABLED=//p')"
    case "$previous_statement" in true) rollback_statement_enabled=true ;; false|'') rollback_statement_enabled=false ;; *) echo "Cannot snapshot statement setting" >&2; exit 1 ;; esac
    rollback_statement_policy="$(docker inspect --format '{{range .Config.Env}}{{println .}}{{end}}' "$previous_api" | sed -n 's/^GRACE_STATEMENT_AI_POLICY=//p')"
  else
    echo "cannot snapshot previous authentication mode: API container is missing" >&2
    exit 1
  fi
fi

rollback_application() {
  if [ -n "$previous_dir" ] && [ -f "$previous_dir/release.env" ]; then
    if [ ! -f "$previous_dir/migrations/000018_commitments.up.sql" ]; then
      has_active_column="$(compose_release exec -T postgres psql -U grace -d grace -At -c "SELECT EXISTS(SELECT 1 FROM information_schema.columns WHERE table_name='scheduled_payments' AND column_name='plan_active');")" || return 1
      if [ "$has_active_column" = "t" ]; then
        retained_history="$(compose_release exec -T postgres psql -U grace -d grace -At -c 'SELECT EXISTS(SELECT 1 FROM scheduled_payments WHERE NOT plan_active);')" || return 1
        if [ "$retained_history" = "t" ]; then
          echo "Rollback refused: previous image cannot read retained schedule history. Keep schema and use a compatible release." >&2
          return 1
        fi
      fi
    fi
    if [ -f "$release_dir/migrations/000027_bank_statement_sources.up.sql" ] && [ ! -f "$previous_dir/migrations/000027_bank_statement_sources.up.sql" ]; then
      has_statement_sources="$(compose_release exec -T postgres psql -U grace -d grace -Atc "SELECT to_regclass('public.statement_sources') IS NOT NULL;")" || return 1
      if [ "$has_statement_sources" = "t" ]; then
        has_statement_data="$(compose_release exec -T postgres psql -U grace -d grace -Atc "SELECT EXISTS(SELECT 1 FROM statement_sources) OR EXISTS(SELECT 1 FROM documents WHERE kind='bank_statement');")" || return 1
        if [ "$has_statement_data" = "t" ]; then
          echo "Rollback refused: previous image cannot review retained PDF sources. Keep schema and use a compatible release." >&2
          return 1
        fi
      fi
    fi
    echo "rolling application containers back to $previous_dir" >&2
    # Restore the mode actually used by the previous image, not a new operator override.
    GRACE_AUTH_MODE="$rollback_auth_mode"
    export GRACE_AUTH_MODE
    GRACE_AI_ENABLED="$rollback_ai_enabled"
    GRACE_AI_ANALYSIS_POLICY="$rollback_analysis_policy"
    GRACE_AI_RECEIPT_POLICY="$rollback_receipt_policy"
    export GRACE_AI_ENABLED GRACE_AI_ANALYSIS_POLICY GRACE_AI_RECEIPT_POLICY
    GRACE_STATEMENT_AI_ENABLED="$rollback_statement_enabled"
    GRACE_STATEMENT_AI_POLICY="$rollback_statement_policy"
    export GRACE_STATEMENT_AI_ENABLED GRACE_STATEMENT_AI_POLICY
    if [ "$ai_configured" = "true" ] && [ -f "$previous_dir/compose.ai.yaml" ]; then
      docker compose --project-name "$compose_project" --env-file "$previous_dir/release.env" \
        -f "$previous_dir/compose.yaml" -f "$previous_dir/compose.ai.yaml" up -d --no-deps api web caddy || return 1
    else
      docker compose --project-name "$compose_project" \
        --env-file "$previous_dir/release.env" -f "$previous_dir/compose.yaml" \
        up -d --no-deps api web caddy || return 1
    fi
    ln -sfn "$previous_dir" "$current_link"
  else
    compose_release stop api web caddy >/dev/null 2>&1 || true
  fi
}

on_failure() {
  status="$?"
  trap - EXIT HUP INT TERM
  if ! rollback_application; then echo "Automatic application rollback requires operator review" >&2; fi
  exit "$status"
}
trap on_failure EXIT HUP INT TERM

mkdir -p "$backups_dir"
chmod 0700 "$backups_dir"

compose_release pull --policy missing postgres pdf-extractor api web caddy
compose_release up -d postgres

attempt=0
until compose_release exec -T postgres pg_isready -U grace -d grace >/dev/null 2>&1; do
  attempt=$((attempt + 1))
  if [ "$attempt" -ge 30 ]; then
    echo "PostgreSQL did not become ready" >&2
    exit 1
  fi
  sleep 2
done

available_kb="$(df -Pk "$backups_dir" | awk 'NR == 2 { print $4 }')"
if [ "$available_kb" -lt 2097152 ]; then
  echo "less than 2 GiB is available for PostgreSQL and backups" >&2
  exit 1
fi

timestamp="$(date -u +%Y%m%dT%H%M%SZ)"
backup="$backups_dir/pre-release-${timestamp}-sha-${release_sha}.dump"
umask 077
compose_release exec -T postgres pg_dump -U grace -d grace -Fc > "$backup"
compose_release exec -T postgres pg_restore --list < "$backup" >/dev/null
chmod 0600 "$backup"
echo "verified pre-deploy backup: $backup"
# An off-site failure must not block a release; the next daily run or
# `backup.sh upload-pending` uploads the copy (SPEC-0050).
"$release_dir/offsite-upload.sh" "$backup" || echo "warning: pre-release backup is not off-site yet" >&2

# Release 090ebad (2026-10-14): migration 22 failed on published import rows.
# Its single transaction rolled back, but migrate left version 22 marked dirty.
# Clear that mark only in exactly this state and only if nothing of 22 exists.
migration_state="$(compose_release exec -T postgres psql -U grace -d grace -Atc \
  "SELECT version || ':' || dirty::text FROM schema_migrations")"
if [ "$migration_state" = "22:true" ]; then
  applied_columns="$(compose_release exec -T postgres psql -U grace -d grace -Atc \
    "SELECT count(*) FROM information_schema.columns WHERE table_schema = 'public' AND ((table_name = 'import_records' AND column_name = 'questions') OR (table_name = 'import_batches' AND column_name IN ('file_name', 'format')))")"
  if [ "$applied_columns" != "0" ]; then
    echo "migration 22 is dirty and partly applied; operator review required" >&2
    exit 1
  fi
  echo "migration 22 rolled back in an earlier release; clearing its dirty mark"
  compose_release run --rm migrate \
    'exec migrate -path=/migrations -database="postgres://grace:$(cat /run/secrets/postgres_password)@postgres:5432/grace?sslmode=disable" force 21'
fi

compose_release run --rm migrate
compose_release up -d api web caddy
# The PDF helper has no credentials/network; API receives only its private socket.
compose_release exec -T pdf-extractor /bin/sh -ec 'test "$(id -u)" = 10001 && test -S /run/grace-pdf/extractor.sock'
compose_release exec -T api /bin/sh -ec 'test -S /run/grace-pdf/extractor.sock; case "$GRACE_STATEMENT_AI_ENABLED" in false) ;; true) exec grace-statement-admin -mode=check ;; *) exit 1 ;; esac'
if [ "$ai_configured" = "true" ] && [ "${GRACE_AI_ENABLED:-false}" = "true" ]; then
  compose_release exec -T api /bin/sh -ec 'test "$(id -u)" = 10001 && test -r /run/secrets/openrouter_api_key && test -w /var/lib/grace/temporary-receipt-photos'
fi

attempt=0
until compose_release exec -T api wget -qO- http://127.0.0.1:8080/ready >/dev/null 2>&1; do
  attempt=$((attempt + 1))
  if [ "$attempt" -ge 30 ]; then
    echo "Grace API did not become ready" >&2
    exit 1
  fi
  sleep 2
done

domain="$(sed -n 's/^GRACE_DOMAIN=//p' "$release_env")"
https_port="$(sed -n 's/^GRACE_HTTPS_PORT=//p' "$release_env")"
attempt=0
until "$release_dir/check-public-entrypoint.sh" "https://$domain:$https_port" \
  "$domain:$https_port:127.0.0.1" >/dev/null 2>&1; do
  attempt=$((attempt + 1))
  if [ "$attempt" -ge 45 ]; then
    echo "Caddy did not expose the public login entrypoint and protected API" >&2
    exit 1
  fi
  sleep 2
done

if [ -n "$previous_dir" ]; then
  ln -sfn "$previous_dir" "$previous_link"
fi
ln -sfn "$release_dir" "$current_link"

trap - EXIT HUP INT TERM
echo "release sha-$release_sha is healthy"
