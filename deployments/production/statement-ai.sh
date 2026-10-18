#!/bin/sh
# Owner-requested enable/disable. No key or budget values are printed or changed.
set -eu
case "${1:-}" in enable|disable) action="$1" ;; *) echo 'usage: statement-ai.sh enable|disable' >&2; exit 2 ;; esac
grace_root="${GRACE_ROOT:-/opt/grace}"
release_dir="$(CDPATH= cd -- "$(dirname "$0")" && pwd)"
release_sha="$(sed -n 's/^RELEASE_TAG=sha-//p' "$release_dir/release.env")"
statement_env="$grace_root/shared/secrets/statement.env"
api="$(docker ps -q --filter label=com.docker.compose.project=grace-production --filter label=com.docker.compose.service=api)"
test -n "$api"
# Serialize manual callers too; a workflow concurrency group is not a host lock.
lock="$grace_root/shared/statement-activation.lock"
if ! mkdir "$lock"; then echo 'Statement activation is already running' >&2; exit 1; fi
umask 077
work=""
had_previous=false
changed=false
committed=false
cleanup(){
  if [ "$changed" = true ] && [ "$committed" != true ]; then
    if [ "$had_previous" = true ]; then mv "$work/previous.env" "$statement_env"; else rm -f "$statement_env"; fi
  fi
  if [ -n "$work" ]; then rm -rf "$work"; fi
  rmdir "$lock"
}
trap cleanup EXIT
trap 'exit 130' HUP INT TERM
work="$(mktemp -d "$grace_root/shared/statement-activation.XXXXXX")"
if [ -f "$statement_env" ]; then
 case "$(stat -c '%a' "$statement_env")" in 400|600) ;; *) echo 'Statement environment must have mode 400 or 600' >&2; exit 1 ;; esac
 statement_owner="$(stat -c '%u' "$statement_env")"
 if [ "$statement_owner" != "$(id -u)" ] && [ "$statement_owner" != 0 ]; then echo 'Statement environment must belong to operator or root' >&2; exit 1; fi
 cp "$statement_env" "$work/previous.env"
 had_previous=true
fi
if [ "$action" = enable ] && [ "$had_previous" = true ] && grep -qx GRACE_STATEMENT_AI_ENABLED=true "$statement_env" && docker exec "$api" grace-statement-admin -mode=check >/dev/null 2>&1; then
  echo 'Statement AI already enabled with a valid pinned policy; no qualification calls repeated.'
  exit 0
fi
if [ "$action" = enable ]; then
 # Both PDFs are fixed synthetic fixtures inside the image; no document input.
 # Output contains only evaluation metrics and public policy, never key/budget values.
 report="$grace_root/shared/statement-qualification-$(date -u +%Y%m%dT%H%M%SZ)-$$.json"
 if ! docker exec "$api" grace-statement-admin -mode=qualify > "$report"; then
   echo 'Synthetic statement qualification failed; activation not applied. Protected report retained.' >&2
   exit 1
 fi
 docker exec -i "$api" grace-statement-admin -mode=summary < "$report"
 policy="$(docker exec "$api" grace-statement-admin -mode=policy)"
 printf "GRACE_STATEMENT_AI_ENABLED=true\nGRACE_STATEMENT_AI_POLICY='%s'\n" "$policy" > "$work/next.env"
else
 printf 'GRACE_STATEMENT_AI_ENABLED=false\n' > "$work/next.env"
fi
chmod 0600 "$work/next.env"
changed=true
mv "$work/next.env" "$statement_env"
if ! "$release_dir/deploy.sh" "$release_sha"; then
 # deploy.sh rolls runtime back to snapshotted flags/policy; restore persistence.
 echo 'Statement activation failed; previous operator configuration restored.' >&2
 exit 1
fi
committed=true
printf 'Statement AI action %s completed; document consent and owner financial confirmation remain required.\n' "$action"
