#!/bin/sh
# Exercise the real S1 planner under current and historical prompt sizes.
set -eu
cd "$(dirname "$0")/../.."
s1_output="${1:?usage: verify-statement-document-plan.sh OUTPUT_DIRECTORY}"
mkdir -p "$s1_output"
s1_output="$(CDPATH= cd -- "$s1_output" && pwd)"
s1_tmp="$(mktemp -d /tmp/grace-s1-layouts.XXXXXX)"
export COMPOSE_PROJECT_NAME="grace-s1-layouts-$$"
export POSTGRES_PORT=0 API_PORT=0 WEB_PORT=0
s1_compose(){ docker compose -f compose.yaml -f compose.pdf-extractor.yaml "$@"; }
cleanup(){ s1_compose down -v --remove-orphans >/dev/null 2>&1 || true; rm -rf "$s1_tmp"; }
trap cleanup EXIT INT TERM
s1_compose up -d --build pdf-extractor >&2
s1_helper="$(s1_compose ps -q pdf-extractor)"
test "$(docker inspect -f '{{.HostConfig.NetworkMode}} {{.HostConfig.ReadonlyRootfs}}' "$s1_helper")" = 'none true'
s1_volume="$(docker inspect -f '{{range .Mounts}}{{if eq .Destination "/run/grace-pdf"}}{{.Name}}{{end}}{{end}}' "$s1_helper")"
docker exec "$s1_helper" pdftotext -v > "$s1_tmp/poppler.txt" 2>&1
s1_tries=0
until docker exec "$s1_helper" test -S /run/grace-pdf/extractor.sock; do s1_tries=$((s1_tries+1));test "$s1_tries" -lt 30;sleep 1;done
for s1_variant in v7 v8 current; do
 python3 - "$PWD" "$s1_tmp" "$s1_variant" <<'PY'
from pathlib import Path
import json,re,subprocess,sys
root,tmp=map(Path,sys.argv[1:3]);variant=sys.argv[3];source=(root/'internal/statementextract/plan.go').read_text()
if variant!='current':
 ref={'v7':'7be572ceaac5819a2031059ef60388dcd0711614','v8':'8a83cdcad87cb62e5a8e6b7e5213c3201bd64d37'}[variant]
 old=subprocess.check_output(['git','show',ref+':internal/statementextract/plan.go'],text=True)
 version=re.search(r'PromptVersion\s*=\s*"([^"]+)"',old).group(1);prompt=re.search(r'const SystemPrompt = `([^`]+)`',old).group(1)
 source=re.sub(r'PromptVersion\s*=\s*"[^"]+"',lambda _: 'PromptVersion = '+json.dumps(version),source,count=1)
 source=re.sub(r'const SystemPrompt = `[^`]+`',lambda _: 'const SystemPrompt = `'+prompt+'`',source,count=1)
target=tmp/(variant+'.go');target.write_text(source)
(tmp/'overlay.json').write_text(json.dumps({'Replace':{str(root/'internal/statementextract/plan.go'):str(target)}}))
PY
 CGO_ENABLED=0 go test -c -tags=integration -trimpath -overlay "$s1_tmp/overlay.json" -o "$s1_tmp/$s1_variant.test" ./internal/statementextract
 chmod 755 "$s1_tmp" "$s1_tmp/$s1_variant.test"
 docker run --rm --network=none --read-only --user 10001:10001 --cap-drop=ALL --security-opt=no-new-privileges:true --memory=512m --pids-limit=16 \
  -v "$s1_tmp/$s1_variant.test:/run/layout.test:ro" -v "$PWD:/src:ro" -v "$s1_volume:/run/grace-pdf:ro" \
  -e TEST_PDF_EXTRACTOR_SOCKET=/run/grace-pdf/extractor.sock -w /src/internal/statementextract alpine:3.22 /run/layout.test -test.run '^TestStatementDocumentLayouts$' -test.v > "$s1_output/$s1_variant.log"
done
python3 - "$s1_output" "$s1_tmp/poppler.txt" <<'PY'
from pathlib import Path
import hashlib,json,re,subprocess,sys
out=Path(sys.argv[1]);variants=[]
for name in ('v7','v8','current'):
 log=(out/(name+'.log')).read_text();count=len(re.findall(r'--- PASS: TestStatementDocumentLayouts/(?:columns|cards)/',log));assert count==28 and '\nFAIL' not in log
 variants.append({'prompt':name,'structuralCases':count,'passed':True})
paths=['internal/statementextract/plan.go','internal/statementextract/aliases.go','internal/statementextract/schema.go','internal/statementextract/lexical.go','internal/statementextract/document_layout_integration_test.go','Dockerfile.pdf-extractor','scripts/research/verify-statement-document-plan.sh','internal/pdfextract/testdata/columns.pdf','internal/pdfextract/testdata/cards.pdf']
source=Path('internal/statementextract/plan.go').read_text()
profile={key:re.search(key+r'\s*=\s*"([^"]+)"',source).group(1) for key in ('PlannerVersion','AliasVersion','PromptVersion','RedactionVersion')}
report={'currentProductionProfile':profile,'variants':variants,'providerCalls':0,'databaseWrites':0,'goVersion':subprocess.check_output(['go','version'],text=True).strip(),'popplerVersion':Path(sys.argv[2]).read_text().splitlines()[0],'sourceHashes':{p:hashlib.sha256(Path(p).read_bytes()).hexdigest() for p in paths}}
(out/'summary.json').write_text(json.dumps(report,indent=2)+'\n');print(json.dumps({'passed':True,'structuralCases':sum(v['structuralCases'] for v in variants),'providerCalls':0,'databaseWrites':0}))
PY
