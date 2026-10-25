#!/bin/sh
# SPEC-0056 U: disposable PostgreSQL only, no ports/network/provider credentials.
set -eu
cd "$(dirname "$0")/../.."
grace_u_name="grace-u-test-$$"
grace_u_container=''
cleanup(){
 if [ -n "$grace_u_container" ];then
  test "$(docker inspect -f '{{index .Config.Labels "grace.test.spec"}} {{.HostConfig.NetworkMode}}' "$grace_u_container")" = '0056-u none'
  docker stop --time 5 "$grace_u_container" >/dev/null
 fi
}
trap cleanup EXIT INT TERM
grace_u_container=$(docker run --detach --rm --name "$grace_u_name" --label grace.test.spec=0056-u --network none --read-only --memory 512m --pids-limit 128 --tmpfs /var/lib/postgresql/data:rw,size=268435456 --tmpfs /var/run/postgresql --tmpfs /tmp -e POSTGRES_USER=grace -e POSTGRES_DB=grace -e POSTGRES_HOST_AUTH_METHOD=trust postgres:16-alpine)
grace_u_tries=0
until docker exec "$grace_u_container" pg_isready -h 127.0.0.1 -U grace -d grace >/dev/null 2>&1;do
 grace_u_tries=$((grace_u_tries+1));test "$grace_u_tries" -lt 60;sleep 1
done
GRACE_U_TEST_CONTAINER="$grace_u_name" python3 -B docs/research/statement-quality/q43-owner-activity-settle-test.py
printf '%s\n' 'SPEC-0056 U: isolated SQL commit/rollback/owner/conflict/replay PASS; live SSH/HTTP/key reads=0'
