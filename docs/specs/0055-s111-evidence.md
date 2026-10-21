# SPEC-0055 / S1.11: offline implementation evidence

Дата: 2026-10-21, Asia/Yekaterinburg.
[Accepted S1.11](0055-s111-model-comparison.md): прямое «одобряю» принято как
разрешение **offline C1–C5**, без C6. Acceptance commit
`3282f42e8d8af736276f1217cd61409b01f6c816` предшествует коду и интегрирован
[PR №83](https://github.com/QahNaaRlN/grace/pull/83), main
`11f26f8cf40a74f2dd7502ff25e1d0b9be836706`.
Implementation source `133ec1293cd93c94b1a52bad1aaad96257fec6db`;
worktree `statement-comparison-implementation`, branch
`codex/statement-comparison-implementation`. Status: local_verified.
**C6 NOT_AUTHORIZED / NOT_RUN; live quality/latency UNVERIFIED.**

## Критерии и исполняемое evidence

| ID | Результат и воспроизведение |
| --- | --- |
| C1 | PASS: `go test ./internal/assistant -run Comparison -count=1`: обе exact policies, alias/canonical/catalog, endpoint/ZDR/price/cap/status/parameter/refusal/refresh/redirect matrix; реальные fake HTTP requests сохраняют only/ZDR/deny/no-fallback/reasoning/caps. Wrong selected/provider/display/request/attempt/pipeline/usage/content отвергаются. Product transport/Policy.Valid не принимают comparison, старый screening не ослаблен. Strict JSON error сохраняет known bill |
| C2 | PASS: `TMPDIR=/tmp scripts/test-bank-statements.sh`: настоящий pinned helper + TestComparisonRealHelperSixTuples. Для **обеих** пар columns19/19, cards20/20, RU23/23; description-gap и identifier-role negatives FAIL при полном6tuple проходе. Fake видит только delivered aliases/text/geometry, не оригинальный layout, golden или скрытые значения. Старые activation/screening, retained expiry/legacy read/preview/confirm, owner/revision/manual-edit, restore и destructive-down regressions PASS |
| C3 | PASS: `go test ./internal/statementqualification -run Comparison -count=1`: порядок ABABAB, максимум6Complete, один reserve208500/settle, known quality failures продолжают, unknown/protocol/invalid cost/duplicate IDs останавливают, unknown не становится0; reserve failure POST0, cancellation и settlement failure не дают PASS. Shared block проверяется перед каждым вызовом. `TestStatementComparisonSharedLedger` на PostgreSQL: exact campaign anchor/reservation, чужой uncertain запрещает call, собственный settle600 не снимает чужой block, zero financial writes; transaction rollback сохраняет исходный budget |
| C4 | PASS: `sh scripts/test-statement-comparison-s111.sh`; `python3 scripts/research/test-screening-workflow-modes.py`; `python3 scripts/research/test-statement-comparison-s111.py`; `go test ./cmd/statement-admin -run Comparison -count=1`. Новый source authorization, replay/cross-SHA/orphan/symlink failures POST0; success и failure расходуют marker. Исторические synthetic S1.4/S1.6/S1.8/S1.9 artifacts неизменны. **4096 реальных сочетаний workflow conditions**, default=false/exclusive jobs/exact-source CI5jobs/bundle hashes. Inspector читает точный reservation UUID через BEGIN READ ONLY, проверяет policy/profile/proofs/known-uncertain/cost; sentinel и файл-byte/mtime/mode immutability PASS. Public Go summary не выводит private text/cost/IDs |
| C5 | PASS: `TMPDIR=/tmp VITEST_MAX_WORKERS=1 make test` (Go, 88 Web файлов/334 теста, lint/FSD/Prettier/build/Jev); финальный `make test-go`; `TMPDIR=/tmp make test-production`; `scripts/test-bank-statements.sh`; `go test -race ./internal/assistant ./internal/statementqualification -run Comparison -count=1`; `go vet ./cmd/... ./internal/... ./scripts/research/s19-replay`; `go vet -tags=integration ./internal/statementqualification ./internal/imports`; staged `git diff --check`. External inference=0, production workflow dispatch=0, deployment/model switch=0 |
| C6 | NOT_AUTHORIZED / NOT_RUN. Ни одной реальной model request; metadata snapshot остаётся ранее прочитанным public GET evidence. Offline synthetic PASS не является качеством A/B, production qualification или разрешением C6 |

## Реализация и границы

`ComparisonReasoner` создаётся только отдельным operator constructor для A/B;
fixed comparison policies не проходят product `Policy.Valid`. Common wire
получил scoped comparison proof/identity branch, старые screening pins и generic
product behavior сохранены. Свежие metadata читаются без ключа: alias/canonical,
reasoning=none, capability/price и exact ZDR; каждая proof содержит sources/hashes.
Начальные proofs обеих пар сохраняются даже при первом unknown POST. Перед
каждым POST proof свежая, а `grace_statement_call_allowed` проверяет shared ledger.

`RunComparison` заранее строит шесть plans, резервирует всю сумму один раз,
сохраняет каждый tuple/trace/closed failure и делает точный settlement. Shared
ledger row имеет model=`statement-comparison.s111`, provider=`fixed-A-B`,
purpose=`statement`; это метка всей кампании, не новая модель/маршрут. Private
report хранит exact reservation UUID, две настоящие policies и каждый bill.
Идентификатор settlement — существующий `statement-qualification:<reservation>`;
старые inspectors с другими pins/ceilings не принимают этот campaign за свой.

Report version `statement.comparison.v1.grounding`; старые activation/screening
validators его отвергают. Canonical request-input hash покрывает policy,
messages и schema; это не hash HTTP headers/credentials. Fixed transport flags
и limits проверены wire tests и profile. Scorer/golden/counts19/20/23 прежние.
Время inference/campaign context ограничено900s; final settlement выполняется
в отдельном отменоустойчивом контексте <=5s, report validator оставляет <=906s
для этой финализации, без дополнительных POST после deadline.

Private metrics сохраняют стоимость всех известных attempts и точную дробь
numerator/denominator на successful document. При0success — undefined; при
incomplete/unknown/settlement failure — UNVERIFIED. False acceptance считается
как deterministic review без blockers при проваленном независимом scorer;
известные quality failures не теряются из стоимости. p95 и full qualification
не заявляются, ranking/adoption автоматом отсутствуют.

CLI: `-mode=comparison-preflight`, `-mode=comparison`, `-mode=comparison-summary`,
`-comparison-dir=<absolute-directory>`. Paid command здесь **не выполнялся**.
Новый workflow input `statement_comparison_s111_only=false` по умолчанию;
`comparison_authorized_sha` пуст. Operator job требует separately approved
exact SHA, CI со всеми5jobs и полный bundle hash. C6 не разрешён этим evidence;
runtime settings и пользовательские документы не трогались.

Новых DTO/OpenAPI/SQL columns/migrations/UI нет; financial core, prompt/schema,
redaction/aliases/planner/grounding и historical S1.8/S1.9 artifacts не менялись.
Legacy S1.8 остаётся FAIL17/19, current trace16/19 не пересчитан в новый PASS.
Unknown billing не восстанавливается автоматически; отдельный recovery для
нового campaign этим пакетом не реализуется и не разрешён.

## Журналы и CI

Локальные журналы: `/tmp/grace-s111-make-final.log`,
`/tmp/grace-s111-go-final.log`, `/tmp/grace-s111-production-final.log`,
`/tmp/grace-s111-bank-final.log`, `/tmp/grace-s111-focused-last.log`,
`/tmp/grace-s111-final-report-test.log`, `/tmp/grace-s111-race.log`,
`/tmp/grace-s111-vet-final.log`, `/tmp/grace-s111-vet-integration-final.log`.
После полного make test финально повторены затронутые Go/inspector проверки;
Web-код не менялся. При staged check удалён один trailing space в новом mock script.

Принятая спецификация: [final-head CI39565320427](https://github.com/QahNaaRlN/grace/actions/runs/39565320427)5/5 PASS;
[merge CI39566592410](https://github.com/QahNaaRlN/grace/actions/runs/39566592410) и
[Delivery39567640692](https://github.com/QahNaaRlN/grace/actions/runs/39567640692) SUCCESS.
Это CI документационного PR №83, **не CI реализации**. Implementation final-head
checks публикуются в отдельном PR; до их завершения full CI PASS не заявляется.
Runtime/production health в этой сессии не проверялась: сохранность image/health
проверена только локальными runner mocks, не выдана за production evidence.
