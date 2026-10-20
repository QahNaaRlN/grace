#!/bin/sh
# Единственное разрешение S1.8; приложение и его настройки не меняются.
set -eu
bundle="${1:?screening bundle required}"
source_sha="${2:?source SHA required}"
printf '%s' "$source_sha" | grep -Eq '^[0-9a-f]{40}$'
case "$bundle" in /tmp/grace-statement-screening-"$source_sha") ;; *) echo invalid_screening_bundle; exit 1 ;; esac
root="${GRACE_ROOT:-/opt/grace}"
shared="$root/shared"
authorization=s18-gemini31-identity-20261021
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
# Повтор workflow или другой SHA не расходуют разрешение повторно.
if [ -e "$marker" ] || [ -L "$marker" ] || [ -e "$report" ] || [ -L "$report" ]; then echo screening_authorization_already_consumed; exit 1; fi
test "$(cat "$bundle/SOURCE_SHA")" = "$source_sha"
(cd "$bundle" && sha256sum -c SHA256SUMS >/dev/null)
before="$(docker inspect -f '{{.Id}} {{.Image}}' "$api")"
docker exec "$api" wget -qO- http://127.0.0.1:8080/ready >/dev/null
umask 077
docker exec "$api" mkdir "$container_dir"
staged=true
docker cp "$bundle/." "$api:$container_dir/" >/dev/null
binary="$container_dir/grace-statement-admin"
# Чтение readiness, точные публичные примеры и свежие metadata провайдера.
if ! docker exec "$api" "$binary" -mode=screening-preflight -screening-dir="$container_dir" > "$report.preflight"; then
 echo screening_preflight_failed
 exit 1
fi
# Фиксированный marker сохраняется при прерывании и неизвестном расходе.
(set -C; printf '%s\n' "$source_sha" > "$marker")
status=0
docker exec "$api" "$binary" -mode=screening -screening-dir="$container_dir" > "$report" || status=$?
printf '%s\n' "$status" > "$report.exit"
# В публичный журнал попадают только безопасные коды и счётчики; отчёт приватный.
summary_status=0
docker exec -i "$api" "$binary" -mode=screening-summary < "$report" || summary_status=$?
test "$(docker inspect -f '{{.Id}} {{.Image}}' "$api")" = "$before"
docker exec "$api" wget -qO- http://127.0.0.1:8080/ready >/dev/null
echo screening_application_image_and_health_preserved
test "$summary_status" -eq 0 || exit "$summary_status"
exit "$status"
