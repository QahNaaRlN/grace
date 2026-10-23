# S1.18 — offline-диагностика качества сохранённых результатов E6

Принятый контракт: [SPEC-0055 / S1.18 Q1–Q5](0055-s118-saved-quality-diagnosis.md).
Владелец принял Q1–Q5 прямым «принимаю»; спецификация интегрирована №103 до
implementation worktree. [Post-merge №103](0055-pr103-postmerge.md).
Implementation commit `f604776e7bb97d42edfd211eda398796771b3e5f`, branch
`codex/statement-s118-quality-diagnosis`; integrated offline №104 / main438256b.
Final head5fdd0f9: [CI39945454718](https://github.com/QahNaaRlN/grace/actions/runs/39945454718)
SUCCESS5/5, canonical exact-SHA/full-job check PASS; [post-merge](0055-pr104-postmerge.md).
На этапе Q1–Q5 Q6 не исполнялся; последующий отдельно разрешённый
[Q6](0055-s118-q6-evidence.md) выполнен один раз: SUCCESS,6/6 VERIFIED, CONSUMED.
Этот документ описывает offline evidence; closed witnesses сюда не переносятся.

## Реализованный контракт и executable evidence

| AC | Реализация и исполняемое подтверждение |
| --- | --- |
| Q1 | `DiagnoseSavedQuality`: fixed S1.17 outer/auth/profile,6 tuples, actual document/plan/request hashes; canonicalize→merge→ValidateProposal→ScoreScreening, full Result/falseAcceptance equality и recorded latency. `TestQualityReplayAndIndependentFailures` прогоняет реальный writer→strict decoder→diagnosis, context/global rejection и snapshot/replay/result mismatch с tuple isolation. |
| Q2 | Actual `NewMappingSnapshot.UnmappedSpanIDs`, source-order positions и lexical class из plan;42 exact reason codes из принятого vocabulary. Список программно сверен со спецификацией. `TestQualityReasonVocabularyAndCounts` проверяет prefix/count/order/duplicates, actual count mismatch, unknown mapping и not_exposed. |
| Q3 | `ScreeningChecks` получает trace непосредственно из существующих scorer closures. `qualityCheck` связывает failed check с actual reasons/positions; clocks date/time/precision/UTC и money amount/currency сравниваются с независимым public golden без вывода literals. `TestQualityClockNearMissAndTrace`, `TestS118RealHelperDiagnosis`: causal lost clock, overlapping evidence→unknown, missing printed turnover; прежние19/20/23 checks сохранены. |
| Q4 | Отдельный `s118-diagnose` читает только fixed E6 report/marker0600 с O_NOFOLLOW; before/after inode/bytes/hash/size/mode/mtime. Strict keys/types/UTF8/duplicate/case/trailing JSON, canonical optional shape,42-code allowlist и bounds. Reader tests проверяют symlink/mode/source/bytes/mtime/inode replacement. `test-statement-s118-diagnosis.py`: Go/Python parity, PRIVATE success/error/timeout negatives, настоящий kernel memfd/all seals/env={}, fixed-host wrapper с mock SSH0, output SHA должен совпасть с compiled verifier SHA. |
| Q5 | Новый default-false `diagnose_statement_s118_only`, отдельный SHA input и job, full manifest binary/context/source/bootstrap/wrapper до SSH, canonical exact-head CI gate. Матрица131072 combinations, YAML parser3 workflows PASS. Payload witnesses не выгружается в Actions artifacts; artifact содержит только manifest/source. `prepare-statement-s118.sh --self-test` использует настоящий pinned Poppler25.04.0 и replay в network=none; real helper/CLI tests PASS. |

`TestQualityActualUnmappedBoundaryAndOutputCap` проверяет настоящие128/129
unmapped spans через ValidateProposal/NewMappingSnapshot и fail-closed64KiB cap.
Ни один UNVERIFIED issue не получает witnesses; verified соседние facts сохраняются.
Context/file integrity failure не публикует частичный context.

## Совместимость и actual diff

Новый код находится в `internal/statementqualification/quality_*`, отдельном
`public_codes.go`, `scripts/research/s118-diagnose`, новых bootstrap/wrapper/
prepare/tests. Дополнительно workflow isolation, Makefile gates и documentation.

Два узких shared changes:

- `Score`/`ScoreScreening` передают optional trace из тех же check closures;
  прежние wrappers вызывают их с nil. Предикаты, порядок, arithmetic/golden,
  scorer/profile versions не менялись; legacy tests и real helper проходят.
- `cmd/statement-admin/screening.go` делегирует public sanitizer в
  `statementqualification.ScreeningPublicCodes`. Тело перенесено byte-identical
  из baseline1b80430, включая прежние row grounding patterns; allowlist не расширен.
  Новый diagnostic использует этот же sanitizer вместо его расходящейся копии.

Product API/OpenAPI/Web/schema/planner/prompt/aliases/redaction/financial guards/
PostgreSQL migrations не менялись. Старые readers/source/ref не перенастраивались.
В Git нет saved E6 bytes, proposals, aliases, source text, UUID/generation/key IDs.
Тестовые fixtures создаются causal delivered-alias fakes; это offline evidence,
не результаты моделей и не фактический диагноз E6.

## Проверки

- `TMPDIR=/tmp VITEST_MAX_WORKERS=1 make test` — PASS:
  Go,88 Web файлов/334 tests, lint/FSD/Prettier/build и offline Jev.
- `TMPDIR=/tmp make test-production` — PASS: новый real-helper/strict-reader/
  memfd gates плюс все legacy operator/production checks.
- `TMPDIR=/tmp make test-integration` — PASS, exit0: реальный PDF→fake provider→
  SQL→confirmation, bank statements, installments/commitments, consolidated/password/
  usage/import/auth, financial events/receipts/obligations/analytics и assistant suites.
  Отдельные optional skipped tests не объявляются выполненными.
- `go test -race ./internal/statementqualification ./scripts/research/s118-diagnose`
  — PASS, включая128/129/64KiB boundaries.
- `go vet ./cmd/... ./internal/... ./scripts/research/s118-diagnose` — PASS.
- После последнего strict/schema/test review: targeted Go reader/qualification/
  CLI tests и Python parity/memfd PASS; YAML3 и полная matrix131072 PASS.
- `git diff --check` — PASS. Exact source CI реализации фиксируется отдельно
  после публикации; CI спецификации не переносится на implementation SHA.

Logs: `/tmp/grace-s118-impl-make.log`, `/tmp/grace-s118-production.log`,
`/tmp/grace-s118-integration.log`, `/tmp/grace-s118-race.log`,
`/tmp/grace-s118-python.log`, `/tmp/grace-s118-helper.log`.
При первой сборке parser обнаружил неверный newline в новом YAML input;
исправлено до публикации, повторный настоящий parser и production suite PASS.

## Граница результата

Q1–Q5 подготовлены offline; их tests не являлись фактической диагностикой E6.
Последующее read-only исполнение и проверка artifact описаны отдельно в
[Q6 evidence](0055-s118-q6-evidence.md). На offline этапе Q1–Q5 ни одного inference/metadata/SSH/SQL
вызова к production, новых billing/settlement/financial writes, изменений
модели/flags/image/health/allowance нет. Локальные integration databases disposable
и не связаны с production. E6 по-прежнему quality FAIL, permission CONSUMED.
Ремонт prompt/guards, новая campaign и SPEC-0056 registry не открыты.
