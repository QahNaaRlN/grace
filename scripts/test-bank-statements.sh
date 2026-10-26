#!/bin/sh
set -eu
export MSYS2_ARG_CONV_EXCL="${MSYS2_ARG_CONV_EXCL:+$MSYS2_ARG_CONV_EXCL;}-path="
cd "$(dirname "$0")/.."
export COMPOSE_PROJECT_NAME="grace-spec0055-db-$$"
export POSTGRES_PORT=0 API_PORT=0 WEB_PORT=0
statement_compose(){ docker compose -f compose.yaml -f compose.pdf-extractor.yaml "$@"; }
cleanup(){ statement_compose down -v --remove-orphans >/dev/null 2>&1 || true; }
trap cleanup EXIT INT TERM
statement_compose up -d --build postgres pdf-extractor
tries=0
until statement_compose exec -T -e PGPASSWORD=grace postgres psql -h 127.0.0.1 -U grace -d grace -Atqc 'SELECT 1' >/dev/null 2>&1; do
 tries=$((tries+1)); test "$tries" -lt 60; sleep 1
done
database_url='postgres://grace:grace@postgres:5432/grace?sslmode=disable'
statement_compose run --rm migrate -path=/migrations -database="$database_url" up
statement_compose run --rm migrate -path=/migrations -database="$database_url" down 1
statement_compose run --rm migrate -path=/migrations -database="$database_url" up
postgres_port="$(statement_compose port postgres 5432 | sed -n '1s/.*://p')"
# One full real-PDF -> fake-provider -> SQL -> confirmation path; the helper
# stays network=none while the disposable test runner can reach PostgreSQL.
helper="$(statement_compose ps -q pdf-extractor)"
socket_volume="$(docker inspect -f '{{range .Mounts}}{{if eq .Destination "/run/grace-pdf"}}{{.Name}}{{end}}{{end}}' "$helper")"
tries=0
until docker exec "$helper" test -S /run/grace-pdf/extractor.sock; do
 tries=$((tries+1)); test "$tries" -lt 30; sleep 1
done
statement_test_binary="$(mktemp /tmp/grace0055-statement-test.XXXXXX)"
trap 'rm -f "$statement_test_binary" "${statement_test_binary}.activation" "${statement_test_binary}.s117-replay" "${statement_test_binary}.s114-replay" "${statement_test_binary}.s115" /tmp/grace-statement-restore-$$.sql; if [ -n "${s117_export:-}" ]; then rm -rf "$s117_export"; fi; cleanup' EXIT INT TERM
CGO_ENABLED=0 go test -c -tags=integration -o "$statement_test_binary" ./internal/statementextract
chmod 755 "$statement_test_binary"
docker run --rm --read-only --user 10001:10001 --cap-drop=ALL --security-opt=no-new-privileges:true \
 --network="${COMPOSE_PROJECT_NAME}_default" --memory=512m --pids-limit=16 \
 -v "$statement_test_binary:/run/statement.test:ro" -v "$PWD:/src:ro" -v "$socket_volume:/run/grace-pdf:ro" \
 -e TEST_BANK_STATEMENTS=true -e TEST_PDF_EXTRACTOR_SOCKET=/run/grace-pdf/extractor.sock \
 -e TEST_DATABASE_URL=postgres://grace:grace@postgres:5432/grace?sslmode=disable \
 -w /src/internal/statementextract alpine:3.22 /run/statement.test -test.run '^(TestStatementRealPDFFakeProviderLifecycle|TestStatementDocumentLayouts)$' -test.v
# Activation runner exercises both exact shipped PDFs through the same real
# helper with delivered-tuples-only fakes, without network or SQL privileges.
CGO_ENABLED=0 go test -c -tags=integration -o "${statement_test_binary}.activation" ./internal/statementqualification
chmod 755 "${statement_test_binary}.activation"
CGO_ENABLED=0 go build -trimpath -o "${statement_test_binary}.s114-replay" ./scripts/research/s113-replay
chmod 755 "${statement_test_binary}.s114-replay"
CGO_ENABLED=0 go build -trimpath -o "${statement_test_binary}.s117-replay" ./scripts/research/s117-replay
chmod 755 "${statement_test_binary}.s117-replay"
CGO_ENABLED=0 go build -trimpath -o "${statement_test_binary}.s115" ./scripts/research/s115-diagnose
chmod 755 "${statement_test_binary}.s115"
docker run --rm --read-only --user 10001:10001 --cap-drop=ALL --security-opt=no-new-privileges:true \
 --network=none --memory=512m --pids-limit=32 \
 -v "${statement_test_binary}.s117-replay:/run/s117-replay:ro" -e TEST_S117_REPLAY_BINARY=/run/s117-replay -v "${statement_test_binary}.s115:/run/s115:ro" -e TEST_S115_BINARY=/run/s115 -v "${statement_test_binary}.s114-replay:/run/s114-replay:ro" -e TEST_S114_REPLAY_BINARY=/run/s114-replay -v "${statement_test_binary}.activation:/run/activation.test:ro" -v "$PWD:/src:ro" -v "$socket_volume:/run/grace-pdf:ro" \
 -e TEST_PDF_EXTRACTOR_SOCKET=/run/grace-pdf/extractor.sock \
 -w /src/internal/statementqualification alpine:3.22 /run/activation.test -test.run '^(TestActivationRealHelperSmoke|TestTurnoverRolesRealHelper|TestScreeningRealHelperThreeCases|TestComparisonRealHelperSixTuples|TestObservedRealHelperReplay|TestS114RealHelperReplay|TestS117RealHelperReplay|TestS115RealHelperDiagnosis)$' -test.v
# Actual S1.17 writer -> Go summary -> Python inspector -> pinned-helper replay6/6.
# This directory holds only public synthetic fixtures; report bytes stay0600.
s117_export="$(mktemp -d /tmp/grace-s117-export.XXXXXX)"
chmod 733 "$s117_export"
docker run --rm --read-only --user 10001:10001 --cap-drop=ALL --security-opt=no-new-privileges:true \
 --network=none --memory=512m --pids-limit=32 \
 -v "${statement_test_binary}.activation:/run/activation.test:ro" -v "${statement_test_binary}.s117-replay:/run/s117-replay:ro" \
 -v "$PWD:/src:ro" -v "$socket_volume:/run/grace-pdf:ro" -v "$s117_export:/run/export:rw" \
 -e TEST_PDF_EXTRACTOR_SOCKET=/run/grace-pdf/extractor.sock -e TEST_S117_REPLAY_BINARY=/run/s117-replay -e S117_REAL_EXPORT_DIR=/run/export \
 -w /src/internal/statementqualification alpine:3.22 /run/activation.test -test.run '^TestS117RealHelperReplay$' -test.v
# Transfer only the test-created synthetic artifact to a host-owned private file.
(umask 077; docker run --rm --read-only --user 10001:10001 --cap-drop=ALL --security-opt=no-new-privileges:true --network=none \
 -v "$s117_export:/run/export:ro" alpine:3.22 cat /run/export/observed.json > "$s117_export/inspection.json")
S117_REAL_FIXTURE="$s117_export/inspection.json" S117_HELPER_VOLUME="$socket_volume" python3 scripts/research/test-comparison-s117.py
rm -rf "$s117_export"
# S1.9 operator replay uses the same pinned helper, no TCP/SQL privileges.
CGO_ENABLED=0 go test -c -tags=integration -o "${statement_test_binary}.s19" ./scripts/research/s19-replay
chmod 755 "${statement_test_binary}.s19"
docker run --rm --read-only --user 10001:10001 --cap-drop=ALL --security-opt=no-new-privileges:true \
 --network=none --memory=512m --pids-limit=16 \
 -v "${statement_test_binary}.s19:/run/s19.test:ro" -v "$PWD:/src:ro" -v "$socket_volume:/run/grace-pdf:ro" \
 -e TEST_PDF_EXTRACTOR_SOCKET=/run/grace-pdf/extractor.sock \
 -w /src/scripts/research/s19-replay alpine:3.22 /run/s19.test -test.run '^TestS19(RealHelperReplay|RetainedReplay)$' -test.v
rm -f "${statement_test_binary}.s19"
CGO_ENABLED=0 go test -c -tags=integration -o "${statement_test_binary}.s112" ./scripts/research/s112-diagnose
chmod 755 "${statement_test_binary}.s112"
docker run --rm --read-only --user 10001:10001 --cap-drop=ALL --security-opt=no-new-privileges:true \
 --network=none --memory=512m --pids-limit=16 \
 -v "${statement_test_binary}.s112:/run/s112.test:ro" -v "$PWD:/src:ro" -v "$socket_volume:/run/grace-pdf:ro" \
 -e TEST_PDF_EXTRACTOR_SOCKET=/run/grace-pdf/extractor.sock \
 -w /src/scripts/research/s112-diagnose alpine:3.22 /run/s112.test -test.run '^TestS112RealHelperStages$' -test.v
rm -f "${statement_test_binary}.s112"

TEST_BANK_STATEMENTS=true TEST_DATABASE_URL="postgres://grace:grace@localhost:${postgres_port}/grace?sslmode=disable" \
 go test -tags=integration ./cmd/statement-admin -run '^TestS11(4|7)StatementMeterSQL$' -count=1 -v
TEST_BANK_STATEMENTS=true TEST_DATABASE_URL="postgres://grace:grace@localhost:${postgres_port}/grace?sslmode=disable" \
 go test -tags=integration ./cmd/statement-clock-pilot -run '^TestClockMeterSQL$' -count=1 -v
TEST_BANK_STATEMENTS=true TEST_DATABASE_URL="postgres://grace:grace@localhost:${postgres_port}/grace?sslmode=disable" \
 go test -p 1 -tags=integration ./internal/imports ./internal/statementextract -run Statement -count=1 -v
statement_compose exec -T postgres pg_dump -U grace -d grace > /tmp/grace-statement-restore-$$.sql
trap 'rm -f "$statement_test_binary" "${statement_test_binary}.activation" "${statement_test_binary}.s117-replay" "${statement_test_binary}.s114-replay" "${statement_test_binary}.s115" /tmp/grace-statement-restore-$$.sql; if [ -n "${s117_export:-}" ]; then rm -rf "$s117_export"; fi; cleanup' EXIT INT TERM
statement_compose exec -T postgres createdb -U grace statement_restore
statement_compose exec -T postgres psql -U grace -d statement_restore -v ON_ERROR_STOP=1 < /tmp/grace-statement-restore-$$.sql >/dev/null
TEST_DATABASE_URL="postgres://grace:grace@localhost:${postgres_port}/statement_restore?sslmode=disable" \
 TEST_STATEMENT_RESTORE=true go test -tags=integration ./internal/imports -run TestStatementRestoredEvidence -count=1 -v
# Pin the statement retention gate; later unrelated migrations may roll back safely.
if statement_compose run --rm migrate -path=/migrations -database="$database_url" goto 30; then
 echo 'Statement evidence/tombstones were discarded by downgrade' >&2; exit 1
fi
echo 'SPEC-0055 bank statement database checks passed'
