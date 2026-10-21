#!/bin/sh
# Local synthetic helper/replay only. No production containers or credentials.
set -eu
cd "$(dirname "$0")/../.."
input="${1:---prepare}"
case "$input" in --self-test|--prepare) ;; *) echo s112_prepare_only; exit 1 ;; esac
work="$(mktemp -d /tmp/grace-s112-diagnose.XXXXXX)"
name="grace-s112-diagnose-$$"
volume="$name-socket"
cleanup(){ docker rm -f "$name" >/dev/null 2>&1 || true; docker volume rm "$volume" >/dev/null 2>&1 || true; rm -f "$work/replay" "$work/replay.test" "$work/projection.json"; rmdir "$work"; }
trap cleanup EXIT INT TERM
chmod 755 "$work"
printf '%s\n' '{}' > "$work/projection.json"
chmod 644 "$work/projection.json"
CGO_ENABLED=0 go build -trimpath -o "$work/replay" ./scripts/research/s112-diagnose
CGO_ENABLED=0 go test -c -tags=integration -o "$work/replay.test" ./scripts/research/s112-diagnose
chmod 755 "$work/replay" "$work/replay.test"
docker build -q -f Dockerfile.pdf-extractor -t grace-s112-pdf:local . >/dev/null
docker volume create "$volume" >/dev/null
docker run -d --name "$name" --network=none --read-only --user 10001:10001 --cap-drop=ALL --security-opt=no-new-privileges:true --memory=512m --pids-limit=16 --tmpfs /tmp:size=16777216,mode=700,uid=10001,gid=10001,noexec,nosuid,nodev -v "$volume:/run/grace-pdf" grace-s112-pdf:local >/dev/null
tries=0
until docker exec "$name" test -S /run/grace-pdf/extractor.sock; do
 tries=$((tries+1)); test "$tries" -lt 60; sleep 1
done
# Both the helper and replay have network=none. Only a Unix socket is shared.
docker run --rm --network=none --read-only --user 10001:10001 --cap-drop=ALL --security-opt=no-new-privileges:true --memory=512m --pids-limit=32 -v "$work:/run/replay:ro" -v "$PWD:/src:ro" -v "$volume:/run/grace-pdf:ro" -w /src/scripts/research/s112-diagnose -e TEST_PDF_EXTRACTOR_SOCKET=/run/grace-pdf/extractor.sock alpine:3.22 /run/replay/replay.test -test.run '^TestS112RealHelperStages$' -test.v >&2
if [ "$input" = --self-test ]; then echo s112_real_helper_counterexamples_pass; exit 0; fi
docker run --rm --network=none --read-only --user 10001:10001 --cap-drop=ALL --security-opt=no-new-privileges:true --memory=512m --pids-limit=32 -v "$work:/run/replay:ro" -v "$PWD/internal/pdfextract/testdata/columns.pdf:/run/columns.pdf:ro" -v "$volume:/run/grace-pdf:ro" alpine:3.22 /run/replay/replay prepare /run/columns.pdf /run/grace-pdf/extractor.sock
