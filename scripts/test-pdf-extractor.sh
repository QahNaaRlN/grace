#!/bin/sh
set -eu
cd "$(dirname "$0")/.."
export COMPOSE_PROJECT_NAME="grace-spec0055-pdf-$$"
export POSTGRES_PORT=0 API_PORT=0 WEB_PORT=0
compose() { docker compose -f compose.yaml -f compose.pdf-extractor.yaml "$@"; }
cleanup() { compose down -v --remove-orphans >/dev/null 2>&1 || true; }
trap cleanup EXIT INT TERM
compose up -d --build pdf-extractor
container="$(compose ps -q pdf-extractor)"
test "$(docker inspect -f '{{.HostConfig.NetworkMode}} {{.HostConfig.ReadonlyRootfs}} {{.HostConfig.Memory}} {{.HostConfig.PidsLimit}}' "$container")" = 'none true 536870912 16'
# Mount the private socket into a Go test container; it has no provider keys.
volume="$(docker inspect -f '{{range .Mounts}}{{if eq .Destination "/run/grace-pdf"}}{{.Name}}{{end}}{{end}}' "$container")"
tries=0
until docker exec "$container" test -S /run/grace-pdf/extractor.sock; do
 tries=$((tries+1)); test "$tries" -lt 30; sleep 1
done
docker run --rm --network=none --user 10001:10001 \
 -v "$PWD:/src:ro" -v "$volume:/run/grace-pdf:ro" \
 -v "$(go env GOPATH)/pkg/mod:/go/pkg/mod:ro" \
 -e GOCACHE=/tmp/go-cache -e GOPATH=/go -e TEST_PDF_EXTRACTOR_SOCKET=/run/grace-pdf/extractor.sock \
 -w /src golang:1.24-alpine go test -tags=integration ./internal/pdfextract -run TestRealPDFLayoutPipeline -count=1 -v
echo 'SPEC-0055 isolated synthetic PDF extraction passed'
