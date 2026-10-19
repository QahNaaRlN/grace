#!/bin/sh
# Reproduce historical prompt sensitivity without editing production source.
set -eu
cd "$(dirname "$0")/../.."
study_baseline='7be572ceaac5819a2031059ef60388dcd0711614'
study_overlay_ref='8a83cdcad87cb62e5a8e6b7e5213c3201bd64d37'
# Refuse to label a different product implementation as this historical study.
git cat-file -e "$study_overlay_ref:internal/statementextract/plan.go"
git diff --quiet "$study_baseline" -- cmd internal api web migrations go.mod go.sum Dockerfile.pdf-extractor compose.yaml compose.pdf-extractor.yaml
study_output="${1:?usage: probe-statement-chunks.sh OUTPUT_DIRECTORY}"
mkdir -p "$study_output"
study_output="$(CDPATH= cd -- "$study_output" && pwd)"
study_tmp="$(mktemp -d /tmp/grace-statement-quality.XXXXXX)"
export COMPOSE_PROJECT_NAME="grace-statement-quality-$$"
export POSTGRES_PORT=0 API_PORT=0 WEB_PORT=0
study_compose(){ docker compose -f compose.yaml -f compose.pdf-extractor.yaml "$@"; }
cleanup(){ study_compose down -v --remove-orphans >/dev/null 2>&1 || true; rm -rf "$study_tmp"; }
trap cleanup EXIT INT TERM
study_compose up -d --build pdf-extractor >&2
study_helper="$(study_compose ps -q pdf-extractor)"
test "$(docker inspect -f '{{.HostConfig.NetworkMode}} {{.HostConfig.ReadonlyRootfs}}' "$study_helper")" = 'none true'
study_volume="$(docker inspect -f '{{range .Mounts}}{{if eq .Destination "/run/grace-pdf"}}{{.Name}}{{end}}{{end}}' "$study_helper")"
study_tries=0
until docker exec "$study_helper" test -S /run/grace-pdf/extractor.sock; do
 study_tries=$((study_tries+1)); test "$study_tries" -lt 30; sleep 1
done
CGO_ENABLED=0 go build -trimpath -o "$study_tmp/v7" ./scripts/research/statement-quality-probe.go
# The overlay changes only the historical prompt/version in the same planner.
git show "$study_overlay_ref:internal/statementextract/plan.go" > "$study_tmp/v8-plan.go"
python3 - "$PWD/internal/statementextract/plan.go" "$study_tmp/v8-plan.go" "$study_tmp/overlay.json" <<'PY'
import json,sys
with open(sys.argv[3],'w') as f:json.dump({'Replace':{sys.argv[1]:sys.argv[2]}},f)
PY
CGO_ENABLED=0 go build -trimpath -overlay "$study_tmp/overlay.json" -o "$study_tmp/v8" ./scripts/research/statement-quality-probe.go
chmod 755 "$study_tmp" "$study_tmp/v7" "$study_tmp/v8"
for revision in v7 v8; do
 docker run --rm --network=none --read-only --user 10001:10001 --cap-drop=ALL --security-opt=no-new-privileges:true --memory=512m --pids-limit=16 \
  -v "$study_tmp/$revision:/run/probe:ro" -v "$PWD/internal/pdfextract/testdata:/fixtures:ro" -v "$study_volume:/run/grace-pdf:ro" alpine:3.22 /run/probe > "$study_output/$revision.json"
done
python3 - "$study_output" "$study_baseline" "$study_overlay_ref" <<'PY'
from pathlib import Path
import hashlib,json,subprocess,sys
p=Path(sys.argv[1]);reports=[json.loads((p/(r+'.json')).read_text()) for r in ('v7','v8')]
summary=[]
for report in reports:
 for case in report['cases']:
  summary.append({'prompt':report['promptVersion'],'name':case['name'],'variant':case['variant'],'chunks':case['chunks'],'splitCoreRows':[r['row'] for r in case['rows'] if not r['coreAtomic']],'error':case.get('error')})
(p/'summary.json').write_text(json.dumps({'providerCalls':0,'databaseWrites':0,'results':summary},indent=2)+'\n')
provenance={'baselineCommit':sys.argv[2],'historicalPromptOverlayCommit':sys.argv[3],'goVersion':subprocess.check_output(['go','version'],text=True).strip(),'probeSHA256':hashlib.sha256(Path('scripts/research/statement-quality-probe.go').read_bytes()).hexdigest(),'shellSHA256':hashlib.sha256(Path('scripts/research/probe-statement-chunks.sh').read_bytes()).hexdigest(),'extractorDockerfileSHA256':hashlib.sha256(Path('Dockerfile.pdf-extractor').read_bytes()).hexdigest(),'providerCalls':0,'databaseWrites':0}
(p/'provenance.json').write_text(json.dumps(provenance,indent=2)+'\n')
print(json.dumps({'reportsWritten':True,'providerCalls':0,'databaseWrites':0,'cases':len(summary)}))
PY
