#!/bin/sh
# Two shipped synthetic PDFs; no credentials, SQL or model endpoint.
set -eu
cd "$(dirname "$0")/../.."
output="${1:?usage: probe-statement-description.sh OUTPUT_FILE}"
probe_dir="$(mktemp -d /tmp/grace-description-replay.XXXXXX)"
export COMPOSE_PROJECT_NAME="grace-description-replay-$$"
export POSTGRES_PORT=0 API_PORT=0 WEB_PORT=0
compose(){ docker compose -f compose.yaml -f compose.pdf-extractor.yaml "$@"; }
cleanup(){ compose down -v --remove-orphans >/dev/null 2>&1 || true;rm -rf "$probe_dir"; }
trap cleanup EXIT INT TERM
compose up -d --build pdf-extractor >&2
helper="$(compose ps -q pdf-extractor)"
test "$(docker inspect -f '{{.HostConfig.NetworkMode}} {{.HostConfig.ReadonlyRootfs}}' "$helper")" = 'none true'
volume="$(docker inspect -f '{{range .Mounts}}{{if eq .Destination "/run/grace-pdf"}}{{.Name}}{{end}}{{end}}' "$helper")"
tries=0
until docker exec "$helper" test -S /run/grace-pdf/extractor.sock;do tries=$((tries+1));test "$tries" -lt 30;sleep 1;done
CGO_ENABLED=0 go build -trimpath -o "$probe_dir/probe" scripts/research/statement-description-replay.go
chmod 755 "$probe_dir" "$probe_dir/probe"
docker run --rm --network=none --read-only --user 10001:10001 --cap-drop=ALL --security-opt=no-new-privileges:true --memory=512m --pids-limit=16 \
 -v "$probe_dir/probe:/run/probe:ro" -v "$PWD:/src:ro" -v "$volume:/run/grace-pdf:ro" -w /src alpine:3.22 /run/probe > "$output"
