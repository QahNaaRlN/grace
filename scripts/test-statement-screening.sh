#!/bin/sh
# Offline authorization/lifecycle test. No Docker, SSH, credentials or model.
set -eu
repo="$(CDPATH= cd -- "$(dirname "$0")/.." && pwd)"
root="$(mktemp -d)"
sha="$(printf '%s' "screening-test-$$" | sha256sum | cut -d' ' -f1 | cut -c1-40)"
bundle="/tmp/grace-statement-screening-$sha"
trap 'rm -rf "$root" "$bundle"' EXIT INT TERM
mkdir -p "$root/shared" "$root/bin"
export GRACE_ROOT="$root" CALLS="$root/calls" FAIL_PREFLIGHT=false FAIL_RUN=false
cat > "$root/bin/docker" <<'MOCK'
#!/bin/sh
case "$*" in
 'ps -q '*) echo synthetic-api ;;
 'inspect -f '*) echo synthetic-container-image ;;
 *'-mode=screening-preflight '*) echo preflight >> "$CALLS";echo '{"screeningPreflight":true}';test "$FAIL_PREFLIGHT" = false ;;
 *'-mode=screening-summary'*) cat >/dev/null;echo '{"screeningPassed":true,"activationAuthorized":false}' ;;
 *'-mode=screening '*) echo run >> "$CALLS";echo '{"version":"synthetic-private-report","costMicros":100}';test "$FAIL_RUN" = false ;;
 'exec '*|'cp '*) : ;;
 *) exit 42 ;;
esac
MOCK
chmod 755 "$root/bin/docker"
export PATH="$root/bin:$PATH"
prepare(){
 mkdir "$bundle"
 printf '%s\n' "$sha" > "$bundle/SOURCE_SHA"
 printf 'synthetic\n' > "$bundle/grace-statement-admin"
 (cd "$bundle" && sha256sum SOURCE_SHA grace-statement-admin > SHA256SUMS)
}
run(){ sh "$repo/deployments/production/statement-screening.sh" "$bundle" "$sha" > "$root/output" 2>&1; }
prepare
export FAIL_PREFLIGHT=true
if run; then echo preflight_failure_accepted;exit 1;fi
test "$(cat "$CALLS")" = preflight
test ! -e "$root/shared/statement-screening-s14-gemini31-20261020.started"
prepare
export FAIL_PREFLIGHT=false FAIL_RUN=true
if run; then echo failed_run_accepted;exit 1;fi
test "$(grep -c '^run$' "$CALLS")" -eq 1
test -r "$root/shared/statement-screening-s14-gemini31-20261020.json"
test "$(stat -c '%a' "$root/shared/statement-screening-s14-gemini31-20261020.json")" = 600
if grep -q costMicros "$root/output";then echo private_billing_leak;exit 1;fi
prepare
export FAIL_RUN=false
if run;then echo replay_accepted;exit 1;fi
test "$(grep -c '^run$' "$CALLS")" -eq 1
grep -q screening_authorization_already_consumed "$root/output"
# Another source SHA still shares the fixed owner authorization marker.
sha="$(printf '%s' "other-$$" | sha256sum | cut -d' ' -f1 | cut -c1-40)"
bundle="/tmp/grace-statement-screening-$sha"
prepare
if run;then echo cross_sha_replay_accepted;exit 1;fi
test "$(grep -c '^run$' "$CALLS")" -eq 1
echo 'Statement screening offline guards passed: no paid replay, no activation, private report preserved.'
