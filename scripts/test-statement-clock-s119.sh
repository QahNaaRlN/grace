#!/bin/sh
set -eu
cd "$(dirname "$0")/.."
./scripts/test-statement-clock-s119-wrapper.sh
work="$(mktemp -d /tmp/grace-clock-helper.XXXXXX)"
name="grace-clock-helper-$$"
volume="$name-socket"
cleanup(){ docker rm -f "$name" >/dev/null 2>&1 || true;docker volume rm "$volume" >/dev/null 2>&1 || true;rm -f "$work/tests";rmdir "$work"; }
trap cleanup EXIT INT TERM
chmod 755 "$work"
CGO_ENABLED=0 go test -c -tags=integration -o "$work/tests" ./internal/statementqualification
chmod 755 "$work/tests"
docker build -q -f Dockerfile.pdf-extractor -t grace-clock-pdf:local . >/dev/null
docker volume create "$volume" >/dev/null
docker run -d --name "$name" --network=none --read-only --user 10001:10001 --cap-drop=ALL --security-opt=no-new-privileges:true --memory=512m --pids-limit=16 --tmpfs /tmp:size=16777216,mode=700,uid=10001,gid=10001,noexec,nosuid,nodev -v "$volume:/run/grace-pdf" grace-clock-pdf:local >/dev/null
tries=0
until docker exec "$name" test -S /run/grace-pdf/extractor.sock; do tries=$((tries+1));test "$tries" -lt 60;sleep 1;done
docker run --rm --network=none --read-only --user 10001:10001 --cap-drop=ALL --security-opt=no-new-privileges:true --memory=512m --pids-limit=32 -v "$work/tests:/run/tests:ro" -v "$PWD:/src:ro" -v "$volume:/run/grace-pdf:ro" -w /src/internal/statementqualification -e TEST_PDF_EXTRACTOR_SOCKET=/run/grace-pdf/extractor.sock alpine:3.22 /run/tests -test.run '^TestClockRealHelper$' -test.v
