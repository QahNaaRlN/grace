#!/bin/sh
# Локальные заглушки S1.17: без Docker, SSH, ключей и вызовов модели.
set -eu
repo="$(CDPATH= cd -- "$(dirname "$0")/.." && pwd)"
root="$(mktemp -d)"
sha="$(printf '%s' "screening-s18-test-$$" | sha256sum | cut -d' ' -f1 | cut -c1-40)"
bundle="/tmp/grace-statement-comparison-s117-$sha"
trap 'rm -rf "$root" "$bundle"' EXIT INT TERM
mkdir -p "$root/shared" "$root/bin"
export GRACE_ROOT="$root" CALLS="$root/calls" FAIL_PREFLIGHT=false FAIL_RUN=false
cat > "$root/bin/docker" <<'MOCK'
#!/bin/sh
case "$*" in
 'ps -q '*) echo synthetic-api ;;
 'inspect -f '*) echo synthetic-container-image ;;
 *'-mode=comparison-s117-preflight '*) echo preflight >> "$CALLS";echo '{"comparisonPreflight":true}';test "$FAIL_PREFLIGHT" = false ;;
 *'-mode=comparison-s117-summary'*) cat >/dev/null;echo '{"comparisonPassed":true,"activationAuthorized":false}' ;;
 *'-mode=comparison-s117 '*) echo run >> "$CALLS";echo '{"version":"synthetic-private-report","costMicros":100}';test "$FAIL_RUN" = false ;;
 'exec '*|'cp '*) : ;;
 *) exit 42 ;;
esac
MOCK
chmod 755 "$root/bin/docker"
export PATH="$root/bin:$PATH"
prepare(){
 mkdir "$bundle"
 printf '%s\n' "$sha" > "$bundle/SOURCE_SHA"
 for file in grace-statement-admin grace-s117-replay columns.pdf cards.pdf turnover-roles-ru.pdf run.sh inspect-statement-comparison-s111.py; do
  printf 'synthetic\n' > "$bundle/$file"
 done
 printf 'print("comparison_inspection_mock_passed")\n' > "$bundle/inspect.py"
 (cd "$bundle" && sha256sum SOURCE_SHA grace-statement-admin grace-s117-replay columns.pdf cards.pdf turnover-roles-ru.pdf run.sh inspect.py inspect-statement-comparison-s111.py > SHA256SUMS)
}
run(){ sh "$repo/deployments/production/statement-comparison-s117.sh" "$bundle" "$sha" "$sha" > "$root/output" 2>&1; }
prepare
if sh "$repo/deployments/production/statement-comparison-s117.sh" "$bundle" "$sha" wrong > "$root/output" 2>&1;then echo unapproved_source_accepted;exit 1;fi
test ! -e "$CALLS"
rm -rf "$bundle"
# Старые артефакты обязаны остаться неизменными во всех сценариях.
printf 'старый marker\n' > "$root/shared/statement-screening-s14-gemini31-20261020.started"
printf 'старый отчёт\n' > "$root/shared/statement-screening-s14-gemini31-20261020.json"
printf 'старый marker S1.6\n' > "$root/shared/statement-screening-s16-gemini31-diag-20261021.started"
printf 'старый отчёт S1.6\n' > "$root/shared/statement-screening-s16-gemini31-diag-20261021.json"
printf 'старый marker S1.8\n' > "$root/shared/statement-screening-s18-gemini31-identity-20261021.started"
printf 'старый report S1.8\n' > "$root/shared/statement-screening-s18-gemini31-identity-20261021.json"
printf 'старый replay S1.9\n' > "$root/shared/statement-diagnosis-s19-replay.json"
printf 'consumed C6 marker\n' > "$root/shared/statement-comparison-s111-comparison-20261021.started"
printf 'historical C6 report\n' > "$root/shared/statement-comparison-s111-comparison-20261021.json"
printf 'consumed P6 marker\n' > "$root/shared/statement-comparison-s114-comparison-20261021.started"
printf 'historical P6 report\n' > "$root/shared/statement-comparison-s114-comparison-20261021.json"
(cd "$root/shared" && sha256sum statement-comparison-s114-* statement-comparison-s111-* statement-screening-s14-* statement-screening-s16-* statement-screening-s18-* statement-diagnosis-s19-* > "$root/old.sha256")
prepare
export FAIL_PREFLIGHT=true
if run; then echo preflight_failure_accepted;exit 1;fi
test "$(cat "$CALLS")" = preflight
test -e "$root/shared/statement-comparison-s117-comparison-20261023.started"
test "$(stat -c '%a' "$root/shared/statement-comparison-s117-comparison-20261023.started")" = 600
# Refusal is consumed; a repeat cannot reach preflight or inference.
prepare
if run; then echo refused_preflight_replayed;exit 1;fi
test "$(cat "$CALLS")" = preflight
rm -rf "$bundle"
# Independent fixed authorization storage for the failure-after-marker case.
export GRACE_ROOT="$root/after-marker";mkdir -p "$GRACE_ROOT/shared"
prepare
export FAIL_PREFLIGHT=false FAIL_RUN=true
if run; then echo failed_run_accepted;exit 1;fi
test "$(grep -c '^run$' "$CALLS")" -eq 1
test -r "$GRACE_ROOT/shared/statement-comparison-s117-comparison-20261023.json"
test "$(stat -c '%a' "$GRACE_ROOT/shared/statement-comparison-s117-comparison-20261023.json")" = 600
if grep -q costMicros "$root/output";then echo private_billing_leak;exit 1;fi
prepare
export FAIL_RUN=false
if run;then echo replay_accepted;exit 1;fi
test "$(grep -c '^run$' "$CALLS")" -eq 1
grep -q comparison_authorization_already_consumed "$root/output"
# Другой SHA также ограничен тем же фиксированным разрешением.
sha="$(printf '%s' "other-$$" | sha256sum | cut -d' ' -f1 | cut -c1-40)"
bundle="/tmp/grace-statement-comparison-s117-$sha"
prepare
if run;then echo cross_sha_replay_accepted;exit 1;fi
test "$(grep -c '^run$' "$CALLS")" -eq 1
(cd "$root/shared" && sha256sum -c "$root/old.sha256" >/dev/null)
# Успех тоже расходует новое разрешение, без удаления прежних файлов.
root_success="$root/success"
mkdir -p "$root_success/shared"
export GRACE_ROOT="$root_success"
prepare
run
test "$(grep -c '^run$' "$CALLS")" -eq 2
prepare
if run;then echo success_replay_accepted;exit 1;fi
test "$(grep -c '^run$' "$CALLS")" -eq 2
# Отчёт без marker также не перезаписывается и не разрешает inference.
rm "$root_success/shared/statement-comparison-s117-comparison-20261023.started"
prepare
if run;then echo orphan_report_replayed;exit 1;fi
test "$(grep -c '^run$' "$CALLS")" -eq 2
# Dangling symlink также блокирует новое разрешение, без перезаписи цели.
root_symlink="$root/symlink"
mkdir -p "$root_symlink/shared"
export GRACE_ROOT="$root_symlink"
ln -s "$root/absent-target" "$root_symlink/shared/statement-comparison-s117-comparison-20261023.json"
prepare
if run;then echo symlink_report_replayed;exit 1;fi
test "$(grep -c '^run$' "$CALLS")" -eq 2
test ! -e "$root/absent-target"
rm "$root_symlink/shared/statement-comparison-s117-comparison-20261023.json"
ln -s "$root/absent-target" "$root_symlink/shared/statement-comparison-s117-comparison-20261023.started"
prepare
if run;then echo symlink_marker_replayed;exit 1;fi
test "$(grep -c '^run$' "$CALLS")" -eq 2
(cd "$root/shared" && sha256sum -c "$root/old.sha256" >/dev/null)
# Orphan preflight/exit and concurrent old/new modes share the same lock.
for suffix in preflight exit; do
 export GRACE_ROOT="$root/orphan-$suffix";mkdir -p "$GRACE_ROOT/shared"
 printf private > "$GRACE_ROOT/shared/statement-comparison-s117-comparison-20261023.json.$suffix"
 prepare
 if run;then echo orphan_metadata_replayed;exit 1;fi
 test "$(grep -c '^run$' "$CALLS")" -eq 2
done
export GRACE_ROOT="$root/locked";mkdir -p "$GRACE_ROOT/shared"
prepare
if (flock -x 8;run) 8>"$GRACE_ROOT/shared/statement-screening.lock";then echo concurrent_run_allowed;exit 1;fi
test "$(grep -c '^run$' "$CALLS")" -eq 2
# Lock refusal deliberately retains an uncertain staged bundle; this test owns it.
rm -rf "$bundle"
# Full manifest is checked before Complete.
export GRACE_ROOT="$root/manifest";mkdir -p "$GRACE_ROOT/shared"
prepare;printf forged >> "$bundle/grace-statement-admin"
if run;then echo forged_manifest_allowed;exit 1;fi
test "$(grep -c '^run$' "$CALLS")" -eq 2
# An omitted/duplicate manifest member or symlink cannot pass the full bundle gate.
for kind in missing duplicate symlink; do
 export GRACE_ROOT="$root/manifest-$kind";mkdir -p "$GRACE_ROOT/shared"
 prepare
 case "$kind" in
  missing) sed -i '/grace-s117-replay/d' "$bundle/SHA256SUMS" ;;
  duplicate) head -n 1 "$bundle/SHA256SUMS" >> "$bundle/SHA256SUMS" ;;
  symlink) rm "$bundle/grace-s117-replay";ln -s grace-statement-admin "$bundle/grace-s117-replay" ;;
 esac
 if run;then echo invalid_manifest_accepted;exit 1;fi
 test "$(grep -c '^run$' "$CALLS")" -eq 2
 done
echo 'Проверки S1.17 пройдены: отказ и успех не разрешают повтор, старые артефакты сохранены.'
