# S1.15 — offline implementation evidence D1–D4

- Дата: 2026-10-21, Asia/Yekaterinburg.
- Accepted contract: [S1.15](0055-s115-saved-proposal-diagnosis.md), D1–D4;
  прямое «Приступай» после PR №93 и явной границы D5. Acceptance commit `d43247f`.
- Implementation: `de770793bba711dada87446ddcd3a442b74f5d5f`,
  branch `codex/statement-saved-proposal-diagnosis`, base main `868c0a4`.
- Статус: **LOCAL_VERIFIED**, готово к review; integration/merge отдельно.
  Exact-head CI5/5 подтверждается отдельно в PR, локальные PASS не заменяют CI.
- **D5 NOT_AUTHORIZED / NOT_EXECUTED**. SSH/VPS, provider/metadata, live SQL,
  workflow dispatch, deploy, image/flag/model switch и ledger writes: **0**.
  Локальный bank suite использовал disposable PostgreSQL; это не live SQL.

## Поведение

Новый pure `DiagnoseSavedProposals` проверяет S1.14 context, обе digest цепочки,
все6 document/plan/request bindings и независимо replay каждой captured issue.
Он не угадывает actual P6 по публичным summaries. Expected P6 shape фиксирован:
две completed quality failures и третий canonicalization failure, затем3 not_sent.

D1 классифицирует exact unknown operationDate selection как original_span_id,
canonical_alias_syntax_absent либо outside_alias_syntax. Admission не меняется;
ни raw item, ни hash/prefix/length/replacement не публикуются.

D2 использует общий `firstDescriptionGap` с прежним production predicate,
выдаёт source ordinals/printed positions первой пары row2 и число промежуточных
невыбранных source positions. D3 берёт membership из actual ValidateProposal через
NewMappingSnapshot; подтверждает saved scorer result и coverage blocker, выводит
только позиции/lexical class. Лимит128 не превращает частичный список в VERIFIED.
Никакой семантической классификации token как service/financial/safe-to-ignore нет.

Закрытый `statement.saved-proposal-diagnosis.v1` содержит6tuple rows; независимый
UNVERIFIED не скрывает другой VERIFIED. Fixed reader разрешает только P6 report
и marker, O_NOFOLLOW/regular0600/size bounds, historical marker source301afd6,
content/size/mode/mtime comparison до и после. Atime не является критерием.

Новый memfd bootstrap исполняет verifier с env={}, без persistent files;
wrapper и Go boundary проверяют закрытую схему, exact keys и numeric/enum types.
Ошибка, private stdout/stderr, timeout или integrity failure не экспортируют raw
exception. Read-only persistentWritesRequested0 не означает отсутствие OS/SSH logs.
Workflow default-off; mixed inputs выбирают0 execution jobs. До будущего D5 нужны
owner-approved exact verifier SHA, CI всех5jobs и полный manifest binary/context/
source/bootstrap/wrapper. Наличие workflow не является разрешением его запуска.

## Критерии и исполняемые свидетельства

| AC | Результат | Проверка |
| --- | --- | --- |
| D1 | PASS offline | `go test ./internal/statementqualification -run TestSavedIndependentAliasAndContext -count=1`: три relations, s00/sA near misses, actual writer/decoder, first operationDate guard до последующего private defect, context mismatch и per-issue replay independence |
| D2 | PASS offline | `go test ./internal/statementdata -run Description -count=1`; `TestS115RealHelperDiagnosis`: production predicate, row2 source positions, synthetic restoration removes guard; consecutive/cross-line/invalid-evidence/service/other-field cases; legacy grounding tests сохраняются |
| D3 | PASS offline | `go test ./internal/statementqualification -run TestSavedUnmappedLimitsAndMembership -count=1`: actual mapping membership, source order, service inclusion/exclusion,128/129 boundary без golden selectors; real-helper B/columns17/19 и один actual unmapped witness |
| D4 | PASS local | `go test ./scripts/research/s115-diagnose`; `python3 scripts/research/test-statement-s115-diagnosis.py`: writer→fixed reader→closed projection; actual sealed memfd/all seals/env poison, dual Go/Python schemas, private sentinel/success/error/timeout, wrong mode/source/symlink/orphan/partial/oversize/duplicate/case/null/UTF8 и mutation negatives |
| D4 workflow | PASS local | `python3 scripts/research/test-screening-workflow-modes.py`:32,768 combinations; exact source, five-job CI check twice, complete manifest checked before sole SSH wrapper |
| D4 real helper | PASS local | `scripts/research/prepare-statement-s115.sh --self-test`: exact3 public fixture hashes, pinned Poppler25.04.0/network=none; delivered-alias fake → actual S1.14 writer/decoder → replay/scorer equivalence16/19,17/19 → structural witnesses → actual verifier validate |
| D5 | NOT_AUTHORIZED | Actual saved P6 relations/gap positions/unmapped witnesses остаются UNVERIFIED до отдельного exact-source approval и одного fixed-host read-only run |

## Обязательные gates

Все команды выполнены в отдельном worktree, основной checkout чистый:

```sh
TMPDIR=/tmp VITEST_MAX_WORKERS=1 make test
TMPDIR=/tmp make test-production
TMPDIR=/tmp scripts/test-bank-statements.sh
go test ./cmd/... ./internal/... ./scripts/research/s115-diagnose
go test -race ./internal/statementdata ./internal/statementextract ./internal/statementqualification ./scripts/research/s115-diagnose
go vet ./cmd/... ./internal/... ./scripts/research/...
scripts/research/prepare-statement-s115.sh --self-test
git diff --check
```

PASS:88 Web files/334 tests, lint/FSD/format/build/Jev; production shell/Python
regressions; local disposable SQL/HTTP/bank lifecycle/restore и all real-helper suites;
final Go/race/vet и diff check. После boundary hardening отдельно повторены final
Go/verifier/dual-schema/helper/race/vet checks. CI5/5 — отдельный PR gate.
Локальные stdout logs: `/tmp/grace-s115-{make,production-final,bank,go-final,race-final,vet-final,transport-final,helper-final}.log`.

## Actual diff и сохранённые границы

- Pure diagnosis/schema + tests:4 новых файла `internal/statementqualification/saved_diagnosis*`.
- Единственный production-path diff: прежний description gap loop вынесен в общий
  predicate; добавлен read-only ordinal helper и near-miss tests в statementdata.
  Предикат/legacy error/output/scorer/prompt/planner/aliases неизменны.
- Новый verifier/strict decoder/reader tests/integration context test:
  `scripts/research/s115-diagnose/`; отдельные prepare/wrapper/bootstrap/privacy tests.
- Default-off workflow input/job и mutual exclusion; новые tests в Makefile
  production gate и bank integration gate. Старые fixed readers не изменялись.
- Docs: accepted spec, этот отчёт, README/roadmap, ранее подготовленная PR92 closure.

`git diff --exit-code 868c0a4 -- api web cmd internal/statementextract
scripts/research/s112-diagnose scripts/research/run-s112-diagnosis.py
scripts/research/s112-bootstrap.py scripts/research/inspect-statement-comparison-s114.py`
— PASS. Financial arithmetic/storage/migrations/OpenAPI/Web не затронуты;
TypeScript generation не нужна. S1.14 report/marker/source301/ref и permission consumed
сохранены, actual P6 quality остаётся FAIL. Синтетические counterexamples не live quality.

## Следующий gate

Review actual diff, exact-head CI5/5, затем integration. После этого отдельно
предъявить owner конкретный verifier SHA/ref для D5. D1–D4 acceptance не открывает
SSH/read-only run, provider calls, исправление proposal/prompt/scorer или новый P6.
