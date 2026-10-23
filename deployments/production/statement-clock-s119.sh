#!/bin/sh
# Fixed S1.19 research entrypoint; a separate exact-source approval is required.
set -eu
bundle="${1:?bundle required}"
source_sha="${2:?source required}"
authorized_sha="${3:-}"
printf '%s' "$source_sha" | grep -Eq '^[0-9a-f]{40}$'
test "$authorized_sha" = "$source_sha" || { echo clock_source_not_authorized; exit 1; }
case "$bundle" in /tmp/grace-statement-clock-s119-"$source_sha") ;; *) echo clock_bundle_invalid; exit 1 ;; esac
shared="${GRACE_ROOT:-/opt/grace}/shared"
marker="$shared/statement-s119-clock-pilot.started"
report="$shared/statement-s119-clock-pilot.json"
test ! -L "$shared/statement-screening.lock"
exec 9>"$shared/statement-screening.lock"
flock -n 9 || { echo clock_already_running; exit 1; }
for file in "$marker" "$report"; do
 if [ -e "$file" ] || [ -L "$file" ]; then echo clock_authorization_consumed; exit 1; fi
done
umask 077
set -C
printf '%s\n' "$source_sha" > "$marker"
api=
staged=false
cleanup(){
 if [ "$staged" = true ]; then docker exec "$api" rm -rf "$bundle" >/dev/null 2>&1 || true; fi
 rm -rf "$bundle"
}
trap cleanup EXIT INT TERM
test -f "$bundle/SHA256SUMS" && test ! -L "$bundle/SHA256SUMS"
awk '
 BEGIN { n=split("grace-clock-pilot cards.pdf cards-document.json SOURCE_SHA MANIFEST.json run.sh",names," ");for(i=1;i<=n;i++)expected[names[i]]=1 }
 NF!=2 || length($1)!=64 || $1~/[^0-9a-f]/ || !($2 in expected) || seen[$2]++ {bad=1}
 END {if(NR!=n || bad)exit 1}
' "$bundle/SHA256SUMS"
for file in grace-clock-pilot cards.pdf cards-document.json SOURCE_SHA MANIFEST.json run.sh; do test -f "$bundle/$file" && test ! -L "$bundle/$file"; done
test "$(cat "$bundle/SOURCE_SHA")" = "$source_sha"
(cd "$bundle" && sha256sum -c SHA256SUMS >/dev/null)
api="$(docker ps -q --filter label=com.docker.compose.project=grace-production --filter label=com.docker.compose.service=api)"
test -n "$api"
before="$(docker inspect -f '{{.Id}} {{.Image}}' "$api")"
docker exec "$api" mkdir "$bundle"
staged=true
docker cp "$bundle/." "$api:$bundle/" >/dev/null
status=0
docker exec "$api" "$bundle/grace-clock-pilot" -mode=run -authorized-source="$source_sha" > "$report" || status=$?
summary_status=0
docker exec -i "$api" "$bundle/grace-clock-pilot" -mode=summary -authorized-source="$source_sha" < "$report" || summary_status=$?
test "$(docker inspect -f '{{.Id}} {{.Image}}' "$api")" = "$before"
echo clock_application_image_preserved
test "$summary_status" -eq 0 || exit "$summary_status"
exit "$status"
