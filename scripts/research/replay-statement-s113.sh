#!/bin/sh
# Offline only: private local artifact, pinned public PDFs, no TCP/provider/SQL.
set -eu
cd "$(dirname "$0")/../.."
input="${1:?absolute offline artifact path required}"
case "$input" in /*) ;; *) echo s113_absolute_artifact_required; exit 1 ;; esac
umask 077
work="$(mktemp -d /tmp/grace-s113-replay.XXXXXX)"
name="grace-s113-replay-$$"
volume="$name-socket"
cleanup(){ docker rm -f "$name" >/dev/null 2>&1 || true; docker volume rm "$volume" >/dev/null 2>&1 || true; rm -f "$work/replay" "$work/artifact.json"; rmdir "$work"; }
trap cleanup EXIT INT TERM
chmod 711 "$work"
cp "$input" "$work/artifact.json"
chmod 600 "$work/artifact.json"
CGO_ENABLED=0 go build -trimpath -o "$work/replay" ./scripts/research/s113-replay
chmod 755 "$work/replay"
docker build -q -f Dockerfile.pdf-extractor -t grace-s113-pdf:local . >/dev/null
docker volume create "$volume" >/dev/null
docker run -d --name "$name" --network=none --read-only --user 10001:10001 --cap-drop=ALL --security-opt=no-new-privileges:true --memory=512m --pids-limit=16 --tmpfs /tmp:size=16777216,mode=700,uid=10001,gid=10001,noexec,nosuid,nodev -v "$volume:/run/grace-pdf" grace-s113-pdf:local >/dev/null
tries=0
until docker exec "$name" test -S /run/grace-pdf/extractor.sock; do
 tries=$((tries+1)); test "$tries" -lt 60; sleep 1
done
# Keep the copied artifact0600: host UID reads it; GID10001 reaches only the helper socket.
docker run --rm --network=none --read-only --user "$(id -u):10001" --cap-drop=ALL --security-opt=no-new-privileges:true --memory=512m --pids-limit=32 -v "$work:/run/replay:ro" -v "$PWD/internal/pdfextract/testdata:/run/fixtures:ro" -v "$volume:/run/grace-pdf:ro" alpine:3.22 /run/replay/replay /run/replay/artifact.json /run/fixtures /run/grace-pdf/extractor.sock
