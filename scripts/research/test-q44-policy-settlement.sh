#!/bin/sh
# SPEC-0056 V: disposable PostgreSQL only, no ports/network/provider credentials.
set -eu
cd "$(dirname "$0")/../.."
grace_v_name="grace-v-test-$$"
grace_v_container=''
cleanup(){
 if [ -n "$grace_v_container" ];then
  test "$(docker inspect -f '{{index .Config.Labels "grace.test.spec"}} {{.HostConfig.NetworkMode}}' "$grace_v_container")" = '0056-v none'
  docker stop --time 5 "$grace_v_container" >/dev/null
 fi
}
trap cleanup EXIT INT TERM
grace_v_container=$(docker run --detach --rm --name "$grace_v_name" --label grace.test.spec=0056-v --network none --read-only --memory 512m --pids-limit 128 --tmpfs /var/lib/postgresql/data:rw,size=268435456 --tmpfs /var/run/postgresql --tmpfs /tmp -e POSTGRES_USER=grace -e POSTGRES_DB=grace -e POSTGRES_HOST_AUTH_METHOD=trust postgres:16-alpine)
grace_v_tries=0
until docker exec "$grace_v_container" pg_isready -h 127.0.0.1 -U grace -d grace >/dev/null 2>&1;do
 grace_v_tries=$((grace_v_tries+1));test "$grace_v_tries" -lt 60;sleep 1
done
GRACE_V_TEST_CONTAINER="$grace_v_name" python3 -B docs/research/statement-quality/q44-policy-settle-test.py
printf '%s\n' 'SPEC-0056 V: isolated SQL commit/rollback/owner/conflict/replay PASS; live SSH/HTTP/key reads=0'
