#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root"

go test ./cmd/jev-eval ./internal/jeveval ./internal/routereval
go run ./cmd/jev-eval validate --dataset testdata/jev-evaluation/v1/cases.json
go run ./cmd/jev-eval validate --dataset testdata/jev-evaluation/v2/cases.json

if go list -deps ./cmd/api | rg -q '^github\.com/QahNaaRlN/grace/internal/jeveval$'; then
  echo "cmd/api must not depend on internal/jeveval" >&2
  exit 1
fi

echo "SPEC-0006 Jev evaluation offline checks passed"
