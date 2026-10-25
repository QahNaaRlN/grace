#!/bin/sh
# SPEC-0057: one private five-page experiment, no deployment or financial import.
set -eu
umask 077
test "$#" -eq 5
bundle=$1
source_sha=$2
bundle_sha=$3
binary_sha=$4
approval_sha=$5
case "$source_sha$bundle_sha$binary_sha$approval_sha" in *[!0-9a-f]*) exit 2;;esac
test "${#source_sha}" -eq 40 && test "${#approval_sha}" -eq 40
test "${#bundle_sha}" -eq 64 && test "${#binary_sha}" -eq 64
test "$bundle" = "/tmp/grace-real-document-check-$source_sha"
shared=/opt/grace/shared
test ! -L "$shared/statement-screening.lock"
exec 9>"$shared/statement-screening.lock"
flock -n 9
marker="$shared/real-document-check-spec0057.started"
report="$shared/real-document-check-spec0057.private.json"
for file in "$marker" "$report";do test ! -e "$file" && test ! -L "$file";done
for file in real-document-check input.json run.sh SOURCE_SHA SHA256SUMS;do test -f "$bundle/$file" && test ! -L "$bundle/$file";done
awk '
 BEGIN {n=split("real-document-check input.json run.sh SOURCE_SHA",names," ");for(i=1;i<=n;i++)expected[names[i]]=1}
 NF!=2 || length($1)!=64 || $1~/[^0-9a-f]/ || !($2 in expected) || seen[$2]++ {bad=1}
 END {if(NR!=n || bad)exit 1}
' "$bundle/SHA256SUMS"
(cd "$bundle" && sha256sum -c SHA256SUMS >/dev/null)
test "$(sha256sum "$bundle/real-document-check" | cut -d ' ' -f 1)" = "$binary_sha"
test "$(sha256sum "$bundle/input.json" | cut -d ' ' -f 1)" = "$bundle_sha"
test "$(cat "$bundle/SOURCE_SHA")" = "$source_sha"
"$bundle/real-document-check" -mode=check -bundle="$bundle/input.json" >/dev/null
# Reuse only the previously verified operator identity, never financial rows.
owner=$(/usr/bin/python3 -I -B - <<'PY'
import os,stat,json,hashlib,uuid
p='/opt/grace/shared/document-reading-spec0056-q4.policy-settlement.json'
f=os.open(p,os.O_RDONLY|os.O_NOFOLLOW|os.O_NOATIME|os.O_CLOEXEC)
s=os.fstat(f)
assert stat.S_ISREG(s.st_mode) and stat.S_IMODE(s.st_mode)==0o600 and s.st_size<2<<20
raw=os.read(f,(2<<20)+1);os.close(f)
assert hashlib.sha256(raw).hexdigest()=='2531586b9f8fdb988c06f2bfd6d120f4848b573276953890473b14de4af11236'
a=json.loads(raw);assert a['committed'] is True and a['usageBefore']['owner_id']==a['usageAfter']['owner_id']
owner=a['usageBefore']['owner_id'];assert str(uuid.UUID(owner))==owner
print(owner)
PY
)
api="$(docker ps -q --filter label=com.docker.compose.project=grace-production --filter label=com.docker.compose.service=api)"
case "$api" in ''|*[!0-9a-f]*) exit 2;;esac
test "${#api}" -le 64
before="$(docker inspect -f '{{.Id}} {{.Image}}' "$api")"
set -C
printf '%s %s %s\n' "$source_sha" "$bundle_sha" "$approval_sha" > "$marker"
staged=false
cleanup(){
 if [ "$staged" = true ];then docker exec "$api" rm -rf "$bundle" >/dev/null 2>&1 || true;fi
 rm -rf "$bundle"
}
trap cleanup EXIT INT TERM
docker exec "$api" mkdir "$bundle"
staged=true
docker cp "$bundle/." "$api:$bundle/" >/dev/null
# Preserve private0600 inputs while making only this transient directory readable
# to the existing unprivileged API user. No environment/service configuration changes.
test "$(docker exec "$api" id -u)" = 10001
docker exec -u 0 "$api" chown -R 10001:10001 "$bundle"
status=0
docker exec -e "GRACE_STATEMENT_OPERATOR_OWNER_ID=$owner" "$api" timeout -s TERM -k 10 240 "$bundle/real-document-check" -mode=run -bundle="$bundle/input.json" -approved-sha256="$bundle_sha" -approval="$approval_sha" -execute-owner-approved=true || status=$?
docker exec "$api" test -f "$bundle/result.json"
docker exec "$api" cat "$bundle/result.json" > "$report"
chmod 600 "$report"
test "$(docker inspect -f '{{.Id}} {{.Image}}' "$api")" = "$before"
exit "$status"
