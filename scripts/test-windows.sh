#!/usr/bin/env bash
# Git for Windows: keep Docker container paths intact and JSON request bytes UTF-8.
set -euo pipefail
export PATH="/mingw64/bin:/usr/bin:/bin:$PATH"
cd "$(dirname "$0")/.."
case "${OSTYPE:-}" in msys*|cygwin*) ;; *) echo 'Use Git for Windows bash.' >&2; exit 1 ;; esac
export LANG=en_US.UTF-8 LC_ALL=en_US.UTF-8 MSYS_NO_PATHCONV=1
# SPEC-0016 deliberately refuses CLI output inside any Git checkout.
task_temp="$(cd .. && pwd)/.grace-test-temp"
mkdir -p "$task_temp" bin/go-cache
export TEMP="$(cygpath -w "$task_temp")"
export TMP="$TEMP"
export GOCACHE="$(cygpath -w "$PWD/bin/go-cache")"
curl() {
  (
    unset MSYS_NO_PATHCONV
    local body='' has_body=0
    local -a args=()
    while (( $# )); do
      case "$1" in
        -d|--data|--data-binary)
          body="$2"; has_body=1; args+=(--data-binary @-); shift 2 ;;
        *) args+=("$1"); shift ;;
      esac
    done
    if (( has_body )); then
      printf '%s' "$body" | command curl "${args[@]}"
    else
      command curl "${args[@]}"
    fi
  )
}
export -f curl
case "${1:-unit}" in
  unit) make test; go vet ./... ;;
  integration) make test-integration ;;
  *) echo 'Usage: bash scripts/test-windows.sh [unit|integration]' >&2; exit 1 ;;
esac
