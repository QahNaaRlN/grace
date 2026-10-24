#!/bin/sh
# SPEC-0056: disposable SQL, pinned networkless PDF helper, fake providers only.
set -eu
export MSYS2_ARG_CONV_EXCL="${MSYS2_ARG_CONV_EXCL:+$MSYS2_ARG_CONV_EXCL;}-path="
cd "$(dirname "$0")/.."
export COMPOSE_PROJECT_NAME="grace-spec0056-$$"
export POSTGRES_PORT=0 API_PORT=0 WEB_PORT=0
# No local live-provider credential is inherited by compose or test binaries.
unset OPENROUTER_API_KEY TYPESAFE_API_KEY GRACE_STATEMENT_PAGE_OPENAI_KEY
compose(){ docker compose -f compose.yaml -f compose.pdf-extractor.yaml "$@"; }
work=$(mktemp -d /tmp/grace0056-test.XXXXXX)
cleanup(){ rm -rf "$work";compose down -v --remove-orphans >/dev/null 2>&1 || true; }
trap cleanup EXIT INT TERM
compose up -d --build postgres pdf-extractor
tries=0
until compose exec -T postgres pg_isready -U grace -d grace >/dev/null 2>&1;do tries=$((tries+1));test "$tries" -lt 60;sleep 1;done
compose run --rm migrate -path=/migrations -database=postgres://grace:grace@postgres:5432/grace?sslmode=disable up
compose run --rm migrate -path=/migrations -database=postgres://grace:grace@postgres:5432/grace?sslmode=disable down 1
compose run --rm migrate -path=/migrations -database=postgres://grace:grace@postgres:5432/grace?sslmode=disable up
helper=$(compose ps -q pdf-extractor)
test "$(docker inspect -f '{{.HostConfig.NetworkMode}} {{.HostConfig.ReadonlyRootfs}} {{.HostConfig.Memory}} {{.HostConfig.PidsLimit}}' "$helper")" = 'none true 536870912 16'
volume=$(docker inspect -f '{{range .Mounts}}{{if eq .Destination "/run/grace-pdf"}}{{.Name}}{{end}}{{end}}' "$helper")
tries=0
until docker exec "$helper" test -S /run/grace-pdf/extractor.sock;do tries=$((tries+1));test "$tries" -lt 30;sleep 1;done
CGO_ENABLED=0 go test -c -tags=integration -o "$work/pdf.test" ./internal/pdfextract
CGO_ENABLED=0 go test -c -tags=integration -o "$work/statement.test" ./internal/statementextract
chmod 755 "$work" "$work/pdf.test" "$work/statement.test"
docker run --rm --network=none --read-only --user 10001:10001 --cap-drop=ALL --security-opt=no-new-privileges:true --memory=512m --pids-limit=32 \
 -v "$work/pdf.test:/run/pdf.test:ro" -v "$PWD:/src:ro" -v "$volume:/run/grace-pdf:ro" \
 -e TEST_PDF_EXTRACTOR_SOCKET=/run/grace-pdf/extractor.sock -w /src/internal/pdfextract alpine:3.22 /run/pdf.test -test.run '^TestS56' -test.v
docker run --rm --network="${COMPOSE_PROJECT_NAME}_default" --read-only --user 10001:10001 --cap-drop=ALL --security-opt=no-new-privileges:true --memory=512m --pids-limit=32 \
 -v "$work/statement.test:/run/statement.test:ro" -v "$PWD:/src:ro" -v "$volume:/run/grace-pdf:ro" \
 -e TEST_S56_DOCUMENTS=true -e TEST_PDF_EXTRACTOR_SOCKET=/run/grace-pdf/extractor.sock \
 -e TEST_DATABASE_URL=postgres://grace:grace@postgres:5432/grace?sslmode=disable \
 -w /src/internal/statementextract alpine:3.22 /run/statement.test -test.run '^TestS56' -test.v
# Separate fresh database: unknown-billing controls cannot contaminate another suite.
compose exec -T postgres createdb -U grace receipt_documents
compose run --rm migrate -path=/migrations -database=postgres://grace:grace@postgres:5432/receipt_documents?sslmode=disable up
test -f internal/receiptdrafts/document_integration_test.go
CGO_ENABLED=0 go test -c -tags=integration -o "$work/receipt.test" ./internal/receiptdrafts
chmod 755 "$work/receipt.test"
docker run --rm --network="${COMPOSE_PROJECT_NAME}_default" --read-only --user 10001:10001 --cap-drop=ALL --security-opt=no-new-privileges:true --memory=512m --pids-limit=32 --tmpfs /tmp:rw,nosuid,nodev,size=64m,mode=1777 \
 -v "$work/receipt.test:/run/receipt.test:ro" -v "$PWD:/src:ro" -v "$volume:/run/grace-pdf:ro" \
 -e TEST_S56_DOCUMENTS=true -e TEST_PDF_EXTRACTOR_SOCKET=/run/grace-pdf/extractor.sock \
 -e TEST_DATABASE_URL=postgres://grace:grace@postgres:5432/receipt_documents?sslmode=disable \
 -w /src/internal/receiptdrafts alpine:3.22 /run/receipt.test -test.run '^TestS56' -test.v
CGO_ENABLED=0 go test -c -tags=integration -o "$work/assistant.test" ./internal/assistant
chmod 755 "$work/assistant.test"
docker run --rm --network="${COMPOSE_PROJECT_NAME}_default" --read-only --user 10001:10001 --cap-drop=ALL --security-opt=no-new-privileges:true --memory=512m --pids-limit=32 \
 -v "$work/assistant.test:/run/assistant.test:ro" -e TEST_S56_DOCUMENTS=true \
 -e TEST_DATABASE_URL=postgres://grace:grace@postgres:5432/receipt_documents?sslmode=disable \
 alpine:3.22 /run/assistant.test -test.run '^TestS56QuickRouter' -test.v
# SPEC-0056 Q1–Q3: actual renderer parity and the same shared SQL functions.
compose exec -T postgres createdb -U grace document_qualification
compose run --rm migrate -path=/migrations -database=postgres://grace:grace@postgres:5432/document_qualification?sslmode=disable up
CGO_ENABLED=0 go test -c -tags=integration -o "$work/qualification.test" ./internal/documentqualification
chmod 755 "$work/qualification.test"
docker run --rm --network="${COMPOSE_PROJECT_NAME}_default" --read-only --user 10001:10001 --cap-drop=ALL --security-opt=no-new-privileges:true --memory=512m --pids-limit=32 \
 -v "$work/qualification.test:/run/qualification.test:ro" -v "$PWD:/src:ro" -v "$volume:/run/grace-pdf:ro" \
 -e TEST_Q_DOCUMENTS=true -e TEST_PDF_EXTRACTOR_SOCKET=/run/grace-pdf/extractor.sock \
 -e TEST_DATABASE_URL=postgres://grace:grace@postgres:5432/document_qualification?sslmode=disable \
 -w /src/internal/documentqualification alpine:3.22 /run/qualification.test -test.run '^(TestQActualRendererCorpus|TestQSQLBudget)$' -test.v
printf '%s\n' 'SPEC-0056 document reading integration passed; live provider calls=0'
