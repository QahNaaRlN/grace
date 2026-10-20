#!/bin/sh
# Operator-only single authorization. No application config/image changes.
set -eu
bundle="${1:?screening bundle required}"
source_sha="${2:?source SHA required}"
printf '%s' "$source_sha" | grep -Eq '^[0-9a-f]{40}$'
case "$bundle" in /tmp/grace-statement-screening-"$source_sha") ;; *) echo invalid_screening_bundle; exit 1 ;; esac
root="${GRACE_ROOT:-/opt/grace}"
shared="$root/shared"
authorization=s14-gemini31-20261020
report="$shared/statement-screening-$authorization.json"
marker="$shared/statement-screening-$authorization.started"
exec 9>"$shared/statement-screening.lock"
flock -n 9 || { echo screening_already_running; exit 1; }
api="$(docker ps -q --filter label=com.docker.compose.project=grace-production --filter label=com.docker.compose.service=api)"
test -n "$api"
container_dir="/tmp/grace-statement-screening-$source_sha"
staged=false
cleanup(){
 if [ "$staged" = true ]; then docker exec "$api" rm -rf "$container_dir" >/dev/null 2>&1 || true; fi
 rm -rf "$bundle"
}
trap cleanup EXIT INT TERM
# A workflow rerun or another SHA cannot consume the same owner allowance twice.
if [ -e "$marker" ]; then echo screening_authorization_already_consumed; exit 1; fi
test "$(cat "$bundle/SOURCE_SHA")" = "$source_sha"
(cd "$bundle" && sha256sum -c SHA256SUMS >/dev/null)
before="$(docker inspect -f '{{.Id}} {{.Image}}' "$api")"
umask 077
docker exec "$api" mkdir "$container_dir"
staged=true
docker cp "$bundle/." "$api:$container_dir/" >/dev/null
binary="$container_dir/grace-statement-admin"
# Read-only readiness, exact public fixtures and authoritative current metadata.
if ! docker exec "$api" "$binary" -mode=screening-preflight -screening-dir="$container_dir" > "$report.preflight"; then
 cat "$report.preflight"
 exit 1
fi
# Exclusive fixed authorization marker remains even on interruption/unknown bill.
(set -C; printf '%s\n' "$source_sha" > "$marker")
status=0
docker exec "$api" "$binary" -mode=screening -screening-dir="$container_dir" > "$report" || status=$?
printf '%s\n' "$status" > "$report.exit"
# Only safe codes/check counts reach public logs. Full billing/report stays here.
docker exec -i "$api" "$binary" -mode=screening-summary < "$report"
test "$(docker inspect -f '{{.Id}} {{.Image}}' "$api")" = "$before"
docker exec "$api" wget -qO- http://127.0.0.1:8080/ready >/dev/null
echo screening_application_image_and_health_preserved
exit "$status"
