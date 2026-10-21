#!/bin/sh
# Operator-only S1.14. P6 requires a separate owner-approved exact source SHA.
set -eu
bundle="${1:?comparison bundle required}"
source_sha="${2:?source SHA required}"
authorized_sha="${3:-}"
printf '%s' "$source_sha" | grep -Eq '^[0-9a-f]{40}$'
test "$authorized_sha" = "$source_sha" || { echo comparison_source_not_authorized; exit 1; }
case "$bundle" in /tmp/grace-statement-comparison-s114-"$source_sha") ;; *) echo invalid_comparison_bundle; exit 1 ;; esac
root="${GRACE_ROOT:-/opt/grace}"
shared="$root/shared"
authorization=s114-comparison-20261021
report="$shared/statement-comparison-$authorization.json"
marker="$shared/statement-comparison-$authorization.started"
exec 9>"$shared/statement-screening.lock"
flock -n 9 || { echo comparison_already_running; exit 1; }
api="$(docker ps -q --filter label=com.docker.compose.project=grace-production --filter label=com.docker.compose.service=api)"
test -n "$api"
container_dir="/tmp/grace-statement-comparison-s114-$source_sha"
staged=false
cleanup(){
 if [ "$staged" = true ]; then docker exec "$api" rm -rf "$container_dir" >/dev/null 2>&1 || true; fi
 rm -rf "$bundle"
}
trap cleanup EXIT INT TERM
if [ -e "$marker" ] || [ -L "$marker" ] || [ -e "$report" ] || [ -L "$report" ]; then echo comparison_authorization_already_consumed; exit 1; fi
# The preflight/exit paths cannot be dangling symlinks either.
for file in "$report.preflight" "$report.exit"; do
 if [ -e "$file" ] || [ -L "$file" ]; then echo comparison_orphan_artifact; exit 1; fi
done
test "$(cat "$bundle/SOURCE_SHA")" = "$source_sha"
(cd "$bundle" && sha256sum -c SHA256SUMS >/dev/null)
before="$(docker inspect -f '{{.Id}} {{.Image}}' "$api")"
docker exec "$api" wget -qO- http://127.0.0.1:8080/ready >/dev/null
umask 077
set -C
docker exec "$api" mkdir "$container_dir"
staged=true
docker cp "$bundle/." "$api:$container_dir/" >/dev/null
binary="$container_dir/grace-statement-admin"
if ! docker exec "$api" "$binary" -mode=comparison-s114-preflight -comparison-dir="$container_dir" > "$report.preflight"; then
 rm -f "$report.preflight"
 echo comparison_preflight_failed
 exit 1
fi
(set -C; printf '%s\n' "$source_sha" > "$marker")
status=0
docker exec "$api" "$binary" -mode=comparison-s114 -comparison-dir="$container_dir" > "$report" || status=$?
printf '%s\n' "$status" > "$report.exit"
summary_status=0
docker exec -i "$api" "$binary" -mode=comparison-s114-summary < "$report" || summary_status=$?
test "$(docker inspect -f '{{.Id}} {{.Image}}' "$api")" = "$before"
docker exec "$api" wget -qO- http://127.0.0.1:8080/ready >/dev/null
echo comparison_application_image_and_health_preserved
inspection_status=0
python3 "$bundle/inspect.py" "$source_sha" || inspection_status=$?
test "$summary_status" -eq 0 || exit "$summary_status"
test "$inspection_status" -eq 0 || exit "$inspection_status"
exit "$status"
