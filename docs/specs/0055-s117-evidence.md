# S1.17 — доказательства offline реализации E1–E5

- Принятый контракт: [SPEC-0055 / S1.17](0055-s117-alias-comparison.md),
  E1–E5 приняты 2026-10-23 прямым «да», интегрированы [PR №99](0055-pr99-postmerge.md).
- Base: `94fd4abf4e2a97116d379474e4b692761ad061da`.
- Ветка: `codex/statement-s117-comparison`.
- Implementation commit: `220284f131029812f3ab918625606aa6fc7b9e0b`.
- Статус: integrated offline, PR №100; подтверждено владельцем. E6 execution NOT_STARTED,
  разрешение не использовано, live source/ref не выбран; readiness NOT_READY.

До E6 выявлен [YAML blocker](0055-s117-e6-readiness.md): E2 matrix и CI не
проверяли синтаксис workflow. Прежние offline PASS сохранены как результаты
конкретных проверок; readiness NOT_READY до интеграции исправления и нового CI.

## Результат и границы

Новый fixed campaign `s117-comparison-20261023` использует общий comparison
engine/capture/canonicalizer/scorer/transport без изменения product extraction.
Outer `statement.comparison.observed.s117.v1`, ledger `statement-comparison.s117`,
отдельные CLI modes/inspector/replay/runner и default-false workflow input.
Private marker/report/preflight/exit относятся только к новому identity,0600.

Marker создаётся под общим lock до readiness/preflight. Даже отказ preflight
расходует разрешение. Повтор по тому же или другому SHA, orphan/symlink и
неполный/дублированный manifest блокируются. Проверяются точный SHA, полный
набор bundle members и их hashes. Mixed mode не запускает соседние jobs.

Точный v3 profile проверяется новым inspector. Legacy PROFILE остаётся v2:
старый `_summary` вызывает общий структурный validator со своим frozen PROFILE,
новый inspector — со своим фиксированным v3. Report fields/digests не
переписываются для совместимости; test-only v2 compiler overlay сохранён.
Replay S1.17 принимает только stdin нового envelope и public PDFs/helper;
model/key/SQL/config constructors в нём отсутствуют.

## Критерии и исполняемые проверки

| ID | Проверка фактического пути | Результат |
| --- | --- | --- |
| E1 | `go test ./internal/statementqualification -run TestS117 -count=1`: actual runners→3×3 decoder matrix, auth/outer/inner/profile/schema/digest/duplicate/unknown/trailing negatives; `make test-production` проверяет frozen v2 отдельно | PASS local |
| E2 | `./scripts/test-statement-comparison-s117.sh`: preflight refusal, failure/success after marker, same/different SHA,600 permissions, old artifact hashes, orphan/symlink/lock, damaged/omitted/duplicate/symlink manifest; `python3 scripts/research/test-screening-workflow-modes.py`:65 536 сочетаний actual job conditions | PASS local; ambiguous inputs→0 jobs, значит0SSH/POST/deploy |
| E3 | `go test ./internal/assistant -run TestS117CampaignActualHTTP -count=1`: actual local HTTP, all6 approved plans/messages/schema/enums/request hashes, ABABAB, one aggregate208500 reserve/settle, call_allowed before each POST, known/unknown/identity/protocol/shared-block/settlement and late-plan refusal | PASS local; только fake HTTP |
| E4 | `python3 scripts/research/test-comparison-s117.py`: fake Complete→capture→writer→Go CLI→fixed inspector, exact UUID SQL query, corruption/privacy/read-only. `make test-integration` включает `TestS117RealHelperReplay`, actual replay executable и повтор Python inspector с original real-helper writer output; `TestS117StatementMeterSQL` использует disposable PostgreSQL | PASS local для новых S1.17 путей; scorer19/20/23, captured replay6/6, providerCalls0; fake SQL inspector и реальный SQL meter проверяются раздельно |
| E5 | `make test`, `make test-production`, `make test-integration`, affected race, vet, diff; exact-head CI после публикации | PASS local: все обязательные команды exit0; exact-head CI проверяется в implementation PR |
| E6 | Приёмка/integration E1–E5, immutable source/ref, его CI5/5 и preflight до однократного dispatch | NOT_STARTED; permission unused, readiness NOT_READY |

Overflow/known billing/stop semantics проверяются `TestS117ContextCaptureAndLegacyIsolation`:
сравнение actual старого и нового runners, snapshot32768/report524288,
unavailable capture не превращает known cost в0. Public projections не включают
snapshot/free text/aliases/generation/reservation IDs. Inspector проверяет
bytes/mode/mtime marker/report до/после и возвращает UNVERIFIED при недоступном
или некорректном replay. Positive fake6/6 не доказывает live model compliance.

## Проверки и ограничения evidence

Окончательные `TMPDIR=/tmp VITEST_MAX_WORKERS=1 make test`, `make test-production`,
`make test-integration`, `go test -race ./internal/statementqualification ./internal/assistant ./cmd/statement-admin`,
`go vet ./cmd/... ./internal/...` и `git diff --check` завершены exit0.
Web:88 файлов/334 теста, ESLint/FSD/Prettier/build и offline Jev PASS.
Финальный integration включает весь прежний набор и новые S1.17 пути;
оба полных production gate завершены exit0. Targeted Go/Python/shell/matrix PASS.

При создании нового test harness исправлены три test-only ошибки: host UID
не мог читать helper socket; replay binary был собран с CGO для glibc вместо
static Alpine-compatible binary; отрицательный UUID совпал с UUID нового
positive fixture. Итоговый сквозной real-helper positive и negatives затем
прошли полный integration. Product/helper/socket permissions не менялись.
Ранние неуспешные logs сохраняются и не объявляются PASS.

Окончательные logs: `/tmp/grace-s117-make-final.log`,
`/tmp/grace-s117-production-final.log`, `/tmp/grace-s117-integration-pass.log`,
`/tmp/grace-s117-race-final.log`, `/tmp/grace-s117-vet-final.log`.
Первоначальные harness failures: `/tmp/grace-s117-integration.log`,
`/tmp/grace-s117-integration-rerun.log`, `/tmp/grace-s117-integration-final.log`.
CI head и merge source не отождествляются. Проверка exact-head CI производится
после публикации; текущие head/run и полный5/5 результат фиксируются в описании
implementation PR. До такого результата E5 не считается закрытым целиком;
локальный PASS не подменяет CI, owner acceptance или integration.

Production API/Web/migrations/schema/prompt/planner/redaction/scorer/guards,
модели/flags/allowance и финансовые writers не менялись. New operator path
создаёт только отдельную usage reservation после будущего E6 gate.
Production/SSH/metadata/inference/private artifact reads0 в этой реализации;
локальные synthetic captures не являются D5/private-derived payload.
Primary main94fd4ab clean; старые P6/D5 permissions и artifacts не затрагивались.
