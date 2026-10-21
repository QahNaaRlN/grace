# SPEC-0055 / S1.13: offline implementation evidence

Дата: 2026-10-21, Asia/Yekaterinburg.
[Accepted S1.13](0055-s113-protocol-observability.md): прямое «одобряю, приступай»
приняло O1–O5 **только offline**. Acceptance commit
`9815659ebe2b5df48d0a5dd116abc75cd37747e6` предшествует коду;
[PR №88](https://github.com/QahNaaRlN/grace/pull/88) интегрировал спецификацию
в main `a2a316dbe907a5ed90eb2c8589a086dd1e7d43ba`.
[Accepted-main CI39599794836](https://github.com/QahNaaRlN/grace/actions/runs/39599794836)
и [Delivery39600996469](https://github.com/QahNaaRlN/grace/actions/runs/39600996469) SUCCESS.

Implementation worktree `statement-protocol-observability`, branch
`codex/statement-protocol-observability`, создан от main a2a316d.
Implementation code commit `5f0a55aa2304e73bfcfdfd1d0814e29f3d1563e6`;
следующий commit содержит только это evidence и status/index updates.
Точный итоговый head и его CI run/jobs публикуются в implementation PR и handoff
после окончания; исходный baseline CI не выдаётся за CI реализации.

Статус: implemented / offline local PASS; live_status=NOT_AUTHORIZED.
External provider inference/metadata, SSH/VPS/workflow dispatch/deploy=0.
Исторический C6 FAIL/consumed и exact guard UNVERIFIED сохранены;
S1.13 не реконструирует утраченный response и не открывает новый campaign.

## O1–O5 и исполняемое evidence

| ID | Результат и команда |
| --- | --- |
| O1 | PASS: `go test ./internal/statementextract -run CanonicalDiagnostic -count=1`. Независимый test-only oracle скопирован из f4fc56a;4guards, monetary-first priority, stable label ordering, nil/empty, numeric/cross-field duplicates и unknown direction/role near-misses.500 deterministic generated cases сверяют прежний error/canonical output, а не новый код с самим собой. Входной Proposal не мутируется; публичный diagnostic — только enums/ordinals |
| O2 | PASS: `go test ./internal/statementqualification -run Observed -count=1`. Настоящий RunComparison с causal delivered-tuples fakes достигает отказа после capture; deep copy проверен мутацией исходного alias slice. Unknown/invalid bill, route, duplicate generation, budget, settlement и ineligible corpus/policy boundaries сохраняют прежние calls/reserve/settle/known/stop semantics. Snapshot bounds отбрасывают объект целиком, не усечённый prefix |
| O3 | PASS: те же Go tests + `python3 scripts/research/test-comparison-observed.py`. Реальный writer→decode→Go CLI→Python inspector: оба digests и tuple/stage/capture consistency, bare v1, forged/unknown/duplicate fields, corrupted digest, size/symlink negatives, sentinel privacy. Docker/SQL вызовы inspector заменены offline stubs; Go verification исполняется настоящим binary. Private file bytes/mode/mtime не меняются. Новый artifact не принимается legacy comparison decoder или product qualification decoder |
| O4 | PASS: `TMPDIR=/tmp scripts/test-bank-statements.sh`, TestObservedRealHelperReplay. Публичные columns/cards/RU через pinned Poppler25.04.0/network=none;3alias failures точно replay, invalid_plan causal guard и mismatch остаются честно UNVERIFIED. Полный positive observed campaign даёт прежние19/20/23 и canonical replay6/6; retained/S1.10/S1.12 и bank/SQL regressions PASS. Это fake quality, не качество реальной модели |
| O5 | Local PASS: `TMPDIR=/tmp VITEST_MAX_WORKERS=1 make test` (Go,88 Web files/334 tests, lint/FSD/format/build/Jev); `TMPDIR=/tmp make test-production`; race/vet/diff checks ниже. После последней малой валидационной правки финальные affected Go tests/vet и inspector test повторены PASS; Web-код не менялся. Implementation exact-head CI всех5jobs — отдельный gate перед любым новым live решением, результат фиксируется в PR/handoff |

Дополнительно: `go test -race ./internal/statementextract ./internal/statementqualification
-run 'CanonicalDiagnostic|Observed' -count=1`; `go vet ./cmd/... ./internal/...
./scripts/research/s113-replay`; `go vet -tags=integration ./internal/statementextract
./internal/statementqualification ./scripts/research/s113-replay`; `git diff --check` PASS.

Локальные журналы: `/tmp/grace-s113-make.log`, `/tmp/grace-s113-production.log`,
`/tmp/grace-s113-race.log`, `/tmp/grace-s113-go-final.log`,
`/tmp/grace-s113-inspector-final.log`, `/tmp/grace-s113-bank-final.log`.

## Реализованные границы

Legacy CanonicalProposal/CanonicalComparisonProposal используют один worker с
новым comparison diagnostic entry point. Прежние predicate decisions и
`provider_protocol` сохраняются; shared worker не исправляет входной mapping.
Product/legacy RunComparison не сохраняет snapshot. Только RunObservedComparison
передаёт operator observations после прежних billing/route checks и до
canonicalization. Три fixture names/hashes, exact comparison policies и прежний
profile/plan/request contract сохраняются; CLI по-прежнему извлекает только
публичные PDFs с проверенными hashes.

Private `statement.comparison.observed.v1` содержит прежний v1 report с прежним
digest и отдельные observations/outer digest. Capture<=32768bytes/tuple,
artifact<=524288bytes включая newline. При переполнении удаляются целые snapshots
в обратном tuple order; base report/accounting не усечены. Если даже base envelope
слишком велик, encoder до записи возвращает artifact_unverified; core report и
известный metered расход сохраняются, не превращаются в0/unknown.

Snapshot — typed memory capture, не crash-durable journal. Private writer stream
существующего comparison mode по-прежнему перенаправляется runner в0600 report;
публичные summary/inspection/replay не выводят этот stream или alias/service text.
Inspector проверяет новый envelope через exact-source Go comparison-summary
из уже staged operator bundle: режим возвращается до config/key/SQL/provider
constructors. Bare v1 не требует нового verifier и остаётся observationUnavailableLegacy.
S1.12 fixed historical reader, старые reports/markers и authorization неизменны.

## Воспроизводимый локальный replay

Только локальный offline artifact и прежние публичные fixtures:

```sh
scripts/research/replay-statement-s113.sh /absolute/path/to/offline-observed.json
```

Launcher создаёт локальный pinned helper в network=none/read-only и копию
artifact0600; host UID читает её, GID10001 используется для Unix socket.
Никакого SSH, provider/key/catalog или SQL. Собранный CLI принимает также bare
v1 и выводит unavailable_legacy. При mismatch — UNVERIFIED, без repair/golden.

Фактический launcher smoke PASS: artifact от явно помеченного offline fake
с преднамеренно другим document hash не становится подходящим к public PDF;
result0 UNVERIFIED, остальные unavailable_capture, providerCalls0 и closed output.
Журналы: `/tmp/grace-s113-cli-result.json`, `/tmp/grace-s113-cli-stderr.log`.
Положительные hashes/replay6/6 проверены отдельно real-helper integration test.

Invalid_plan real-helper case преднамеренно мутирует переданный Chunk внутри
offline fake: общий guard фиксирует invalid_plan, snapshot сохраняется, но
transient plan не входит в принятый snapshot. Replay по исходному verified plan
не совпадает и возвращает UNVERIFIED. Придуманного повреждённого plan или
восстановления старого C6 этим evidence не заявляется.

## Scope review и следующий gate

Runtime diff ограничен statementextract canonicalizer/diagnostic,
statementqualification observed runner/envelope, private operator writer/summary,
существующим inspector и локальным replay; остальные изменения — tests/gates/docs.
Тестовый baseline oracle не компилируется в production. Prompt/schema/redaction/
aliases/planner/scorer versions, policy/model/route/price/caps, API/OpenAPI/DTO,
Web/SQL/migrations/financial core и workflow modes не менялись.

Новых provider metadata/inference, SSH/VPS, reconciliation, budget reset или
workflow dispatch нет. Paid campaign и production adoption остаются закрыты.
После exact-head CI5/5 возможен только отдельный review нового live контракта:
call count/caps, новая authorization и exact source, затем прямое решение владельца.
