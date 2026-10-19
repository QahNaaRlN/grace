#!/bin/sh
# Geometry comparison only: product source is replaced in build overlays, never edited.
set -eu
cd "$(dirname "$0")/../.."
study_baseline='7be572ceaac5819a2031059ef60388dcd0711614'
study_historical='8a83cdcad87cb62e5a8e6b7e5213c3201bd64d37'
git diff --quiet "$study_baseline" -- cmd internal api web migrations go.mod go.sum Dockerfile.pdf-extractor compose.yaml compose.pdf-extractor.yaml
study_output="${1:?usage: probe-statement-atoms.sh OUTPUT_DIRECTORY}"
mkdir -p "$study_output"
study_output="$(CDPATH= cd -- "$study_output" && pwd)"
study_tmp="$(mktemp -d /tmp/grace-statement-atoms.XXXXXX)"
export COMPOSE_PROJECT_NAME="grace-statement-atoms-$$"
export POSTGRES_PORT=0 API_PORT=0 WEB_PORT=0
study_compose(){ docker compose -f compose.yaml -f compose.pdf-extractor.yaml "$@"; }
cleanup(){ study_compose down -v --remove-orphans >/dev/null 2>&1 || true; rm -rf "$study_tmp"; }
trap cleanup EXIT INT TERM
study_compose up -d --build pdf-extractor >&2
study_helper="$(study_compose ps -q pdf-extractor)"
test "$(docker inspect -f '{{.HostConfig.NetworkMode}} {{.HostConfig.ReadonlyRootfs}}' "$study_helper")" = 'none true'
docker exec "$study_helper" pdftotext -v > "$study_tmp/poppler-version.txt" 2>&1
study_volume="$(docker inspect -f '{{range .Mounts}}{{if eq .Destination "/run/grace-pdf"}}{{.Name}}{{end}}{{end}}' "$study_helper")"
study_tries=0
until docker exec "$study_helper" test -S /run/grace-pdf/extractor.sock; do
 study_tries=$((study_tries+1)); test "$study_tries" -lt 30; sleep 1
done
git show "$study_historical:internal/statementextract/plan.go" > "$study_tmp/v8.go"
for prompt in v7 v8; do
 for algorithm in gap blocks document document_alias document_alias_90k; do
  study_name="$prompt-$algorithm"
  python3 - "$PWD" "$study_tmp" "$prompt" "$algorithm" <<'PY'
from pathlib import Path
import json,sys
root,tmp=map(Path,sys.argv[1:3]);prompt,algorithm=sys.argv[3:5]
source=(root/'internal/statementextract/plan.go').read_text() if prompt=='v7' else (tmp/'v8.go').read_text()
start=source.index('\t// Horizontal bands')
end=source.index('\tcontextSpans :=',start)
source=source[:start]+'\tgroups, keys := researchGroups(d, visible)\n'+source[end:]
if algorithm.startswith('document_alias'):
 marker='\tcurrent := Chunk{'
 wire='''
 aliases:=ResearchAliasMap(d);canonicalToAlias:=map[string]string{}
 for alias,id:=range aliases{canonicalToAlias[id]=alias}
 wireSpan:=func(s RedactedSpan)RedactedSpan{old:=s.ID;s.ID=canonicalToAlias[old];if s.Text=="[text_"+old+"]"{s.Text="[text_"+s.ID+"]"};return s}
 for key,spans:=range groups{for i:=range spans{spans[i]=wireSpan(spans[i])};groups[key]=spans}
 for i:=range contextSpans{contextSpans[i]=wireSpan(contextSpans[i])}
 '''
 if algorithm=='document_alias_90k':
  source=source.replace('> 9000','> 90000').replace('InputBound       = 10000','InputBound       = 100000').replace('OutputBound      = 2000','OutputBound      = 16000')
  source=source.replace('\tvisible := map[string]bool{}','\tif len(d.Spans)>2000 {return p,errors.New("statement_document_span_limit")}\n\tvisible := map[string]bool{}',1)
  wire='contextSpans=[]RedactedSpan{}\n'+wire
 source=source.replace(marker,wire+marker,1)
source+='\n'+(root/'scripts/research/atoms/groups.go.txt').read_text().replace('"ALGORITHM"',json.dumps(algorithm))
target=tmp/(prompt+'-'+algorithm+'.go');target.write_text(source)
(tmp/'overlay.json').write_text(json.dumps({'Replace':{str(root/'internal/statementextract/plan.go'):str(target),str(root/'scripts/research/statement-quality-probe.go'):str(root/'scripts/research/atoms/probe.go.txt')}}))
PY
  CGO_ENABLED=0 go build -trimpath -overlay "$study_tmp/overlay.json" -o "$study_tmp/$study_name" ./scripts/research/statement-quality-probe.go
  chmod 755 "$study_tmp" "$study_tmp/$study_name"
  docker run --rm --network=none --read-only --user 10001:10001 --cap-drop=ALL --security-opt=no-new-privileges:true --memory=512m --pids-limit=16 \
   -v "$study_tmp/$study_name:/run/probe:ro" -v "$PWD/internal/pdfextract/testdata:/fixtures:ro" -v "$study_volume:/run/grace-pdf:ro" alpine:3.22 /run/probe > "$study_output/$study_name.json"
 done
done
python3 - "$study_output" "$study_baseline" "$study_historical" "$study_tmp/poppler-version.txt" <<'PY'
from pathlib import Path
import hashlib,json,subprocess,sys
out=Path(sys.argv[1]);summary=[]
for prompt in ('v7','v8'):
 for algorithm in ('gap','blocks','document','document_alias','document_alias_90k'):
  reportpath=out/(prompt+'-'+algorithm+'.json');report=json.loads(reportpath.read_text());rows=report['results'];accepted=[r for r in rows if not r.get('error')]
  reportpath.write_text(json.dumps(report,indent=2)+'\n')
  assert all(check['passed'] for check in report['boundaryChecks'])
  if algorithm=='document_alias_90k':
   assert len(accepted)==28 and all(not r['splitFullRecordRows'] and not r['splitCoreRows'] and r['sourceExactlyOnce'] and r['chunks']==1 for r in accepted)
  summary.append({'prompt':prompt,'algorithm':algorithm,'cases':len(rows),'accepted':len(accepted),'refused':len(rows)-len(accepted),'acceptedCoreSplit':sum(bool(r['splitCoreRows']) for r in accepted),'acceptedFullRecordSplit':sum(bool(r['splitFullRecordRows']) for r in accepted),'geometricRecordSplit':sum(bool(r['recordSplitAcrossAtoms']) for r in rows),'baselineAccepted':all(not r.get('error') for r in rows if r['mutation']=='prefix_0')})
assert sum(r['cases'] for r in summary)==280
(out/'summary.json').write_text(json.dumps({'providerCalls':0,'databaseWrites':0,'results':summary},indent=2)+'\n')
files=['scripts/research/probe-statement-atoms.sh','scripts/research/atoms/probe.go.txt','scripts/research/atoms/groups.go.txt','Dockerfile.pdf-extractor']
provenance={'baselineCommit':sys.argv[2],'historicalPromptCommit':sys.argv[3],'popplerVersion':Path(sys.argv[4]).read_text().splitlines()[0],'goVersion':subprocess.check_output(['go','version'],text=True).strip(),'fileHashes':{p:hashlib.sha256(Path(p).read_bytes()).hexdigest() for p in files},'providerCalls':0,'databaseWrites':0}
(out/'provenance.json').write_text(json.dumps(provenance,indent=2)+'\n')
print(json.dumps(summary,indent=2))
PY
