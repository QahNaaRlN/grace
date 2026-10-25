#!/bin/sh
# SPEC-0056 Q4 only: one separately approved source/manifest, no deployment.
set -eu
umask 077
test "$#" -eq 5
bundle=$1
source_sha=$2
authorized_sha=$3
manifest_sha=$4
approval_sha=$5
case "$source_sha$approval_sha$manifest_sha" in *[!0-9a-f]*) echo source_invalid;exit 1;;esac
test "${#source_sha}" -eq 40 && test "${#approval_sha}" -eq 40 && test "${#manifest_sha}" -eq 64
test "$authorized_sha" = "$source_sha" || { echo source_not_authorized;exit 1; }
test "$bundle" = "/tmp/grace-document-reading-q-$source_sha" || { echo bundle_invalid;exit 1; }
shared="${GRACE_ROOT:-/opt/grace}/shared"
test ! -L "$shared/statement-screening.lock"
exec 9>"$shared/statement-screening.lock"
flock -n 9 || { echo already_running;exit 1; }
marker="$shared/document-reading-spec0056-q2.started"
report="$shared/document-reading-spec0056-q2.private.json"
for file in "$marker" "$report";do if [ -e "$file" ] || [ -L "$file" ];then echo authorization_consumed;exit 1;fi;done
set -C
printf '%s %s %s\n' "$source_sha" "$manifest_sha" "$approval_sha" > "$marker"
api=
staged=false
cleanup(){
 if [ "$staged" = true ];then docker exec "$api" rm -rf "$bundle" >/dev/null 2>&1 || true;fi
 rm -rf "$bundle"
}
trap cleanup EXIT INT TERM
owner="${GRACE_STATEMENT_OPERATOR_OWNER_ID:-}"
printf '%s' "$owner" | grep -Eq '^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$'
test -f "$bundle/SHA256SUMS" && test ! -L "$bundle/SHA256SUMS"
awk '
 BEGIN {n=split("grace-document-reading-eval corpus.json MANIFEST.json SOURCE_SHA run.sh",names," ");for(i=1;i<=n;i++)expected[names[i]]=1}
 NF!=2 || length($1)!=64 || $1~/[^0-9a-f]/ || !($2 in expected) || seen[$2]++ {bad=1}
 END {if(NR!=n || bad)exit 1}
' "$bundle/SHA256SUMS"
for file in grace-document-reading-eval corpus.json MANIFEST.json SOURCE_SHA run.sh;do test -f "$bundle/$file" && test ! -L "$bundle/$file";done
test -x "$bundle/grace-document-reading-eval"
test "$(cat "$bundle/SOURCE_SHA")" = "$source_sha"
(cd "$bundle" && sha256sum -c SHA256SUMS >/dev/null)
test "$(sha256sum "$bundle/MANIFEST.json" | cut -d ' ' -f 1)" = "$manifest_sha" || { echo manifest_not_authorized;exit 1; }
api="$(docker ps -q --filter label=com.docker.compose.project=grace-production --filter label=com.docker.compose.service=api)"
case "$api" in ''|*[!0-9a-f]*) echo api_unavailable;exit 1;;esac
test "${#api}" -le 64
before="$(docker inspect -f '{{.Id}} {{.Image}}' "$api")"
docker exec "$api" mkdir "$bundle"
staged=true
docker cp "$bundle/." "$api:$bundle/" >/dev/null
status=0
docker exec -e "GRACE_STATEMENT_OPERATOR_OWNER_ID=$owner" "$api" timeout -s TERM -k 10 1810 "$bundle/grace-document-reading-eval" -mode=run -authorized-source="$source_sha" -approved-manifest="$manifest_sha" -approval-id="$approval_sha" -execute-approved-reading=true > "$report" 2>/dev/null || status=$?
summary_status=0
summary="$(docker exec -i "$api" timeout -s TERM -k 2 60 "$bundle/grace-document-reading-eval" -mode=summary -authorized-source="$source_sha" < "$report" 2>/dev/null)" || summary_status=$?
test "$(docker inspect -f '{{.Id}} {{.Image}}' "$api")" = "$before"
test "$summary_status" -eq 0 || exit "$summary_status"
printf '%s\n' "$summary"
exit "$status"
