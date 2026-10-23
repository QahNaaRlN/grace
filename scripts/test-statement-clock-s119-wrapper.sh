#!/bin/sh
set -eu
repo="$(CDPATH= cd -- "$(dirname "$0")/.." && pwd)"
root="$(mktemp -d /tmp/grace-clock-wrapper.XXXXXX)"
sha="$(printf '%s' "clock-test-$$" | sha256sum | cut -c1-40)"
bundle="/tmp/grace-statement-clock-s119-$sha"
trap 'rm -rf "$root" "$bundle"' EXIT INT TERM
mkdir "$root/bin"
export GRACE_ROOT="$root/current" CALLS="$root/calls" FAIL_RUN=false
cat > "$root/bin/docker" <<'MOCK'
#!/bin/sh
case "$*" in
 'ps -q '*) echo synthetic-api ;;
 'inspect -f '*) echo synthetic-image ;;
 *'-mode=run '*) echo run >> "$CALLS";echo '{"PRIVATE":"never publish"}';test "$FAIL_RUN" = false ;;
 *'-mode=summary '*) cat >/dev/null;echo '{"shapeVerified":true,"activationAuthorized":false}' ;;
 'exec '*|'cp '*) : ;;
 *) exit 42 ;;
esac
MOCK
chmod 755 "$root/bin/docker"
export PATH="$root/bin:$PATH"
prepare(){
 mkdir -p "$GRACE_ROOT/shared" "$bundle"
 printf '%s\n' "$sha" > "$bundle/SOURCE_SHA"
 for f in grace-clock-pilot cards.pdf cards-document.json MANIFEST.json run.sh; do printf 'public\n' > "$bundle/$f"; done
 (cd "$bundle" && sha256sum grace-clock-pilot cards.pdf cards-document.json SOURCE_SHA MANIFEST.json run.sh > SHA256SUMS)
}
run(){ sh "$repo/deployments/production/statement-clock-s119.sh" "$bundle" "$sha" "$sha" > "$root/out" 2>&1; }
prepare
if sh "$repo/deployments/production/statement-clock-s119.sh" "$bundle" "$sha" wrong > "$root/out" 2>&1; then exit 1; fi
test ! -e "$CALLS"
run
test "$(cat "$CALLS")" = run
test "$(stat -c %a "$GRACE_ROOT/shared/statement-s119-clock-pilot.json")" = 600
if grep -q PRIVATE "$root/out"; then exit 1; fi
prepare
if run;then echo repeated_pilot_accepted;exit 1;fi
test "$(cat "$CALLS")" = run
rm -rf "$bundle"
old_sha="$sha"
sha="$(printf '%s' "second-source-$$" | sha256sum | cut -c1-40)"
bundle="/tmp/grace-statement-clock-s119-$sha"
prepare
if run;then echo cross_source_repeat;exit 1;fi
test "$(cat "$CALLS")" = run
rm -rf "$bundle"
sha="$old_sha"
bundle="/tmp/grace-statement-clock-s119-$sha"
# Invalid manifest consumes the fixed authorization before preflight too.
export GRACE_ROOT="$root/invalid"
prepare
printf 'invalid\n' > "$bundle/SHA256SUMS"
if run;then exit 1;fi
test -e "$GRACE_ROOT/shared/statement-s119-clock-pilot.started"
prepare
if run;then exit 1;fi
test "$(cat "$CALLS")" = run
rm -rf "$bundle"
export GRACE_ROOT="$root/failure" FAIL_RUN=true
prepare
if run;then exit 1;fi
prepare
if run;then exit 1;fi
test "$(wc -l < "$CALLS")" -eq 2
rm -rf "$bundle"
export GRACE_ROOT="$root/symlink" FAIL_RUN=false
prepare
ln -s "$root/absent" "$GRACE_ROOT/shared/statement-s119-clock-pilot.started"
if run;then exit 1;fi
test ! -e "$root/absent"
echo clock_wrapper_one_shot_pass
