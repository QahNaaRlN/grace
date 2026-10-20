# SPEC-0055 — Statement AI activation

Актуализация: 2026-10-19 (Asia/Yekaterinburg). **Statement AI выключен; цель включения ещё не достигнута.**
Production `ef782ec` (S1.1 PR #73): main CI `39254418192` и Delivery `39255451276` PASS.
Deploy `39255669019`: internal health PASS **12:55:14 UTC / 17:55:14 +05:00**;
единственный разрешённый S1.1 smoke FAIL на columns: только `record_description`
(дефис из второго описания в service), cards NOT_RUN. Полные критерии сохранены.
Read-only `39255943568` PASS 12:56:16 UTC: actual runtime disabled, unknown
usage не блокирует. Standalone public/auth smoke PASS (200/401). Повтора нет.

[S1.1 evidence L1–L7](0055-s11-evidence.md) сохраняет executable проверки и FAIL.
[S1.2](0055-s12-description-separators.md) отдельно принят владельцем: конкретное
prompt уточнение и negative regression, один новый smoke ≤2 calls / $0.14,
включение только при обоих полных PASS. Реализация ещё не начата.

## Исторические попытки и исходы (не текущий release)

| Срез | Дата / evidence | Статус |
| --- | --- | --- |
| Functional/technical contract | Owner «да, утверждаю», 2026-10-19; PR #62, main `4a6f1e7` | accepted |
| Код PDF import | PR #64, merge `b51b74f`, main CI `39167930999` | integrated; gates PASS |
| Первый production manual path | Deploy `39168570754`, healthy 2026-10-19 02:37:31 +05:00 / 2026-10-18 21:37:31 UTC | deployed |
| Statement activation approval | Прямое поручение владельца, 2026-10-19 | authorized |
| Первичная readiness | Read-only [39169569800](https://github.com/QahNaaRlN/grace/actions/runs/39169569800), 2026-10-19 | PASS на момент проверки |
| Механизм активации | [PR #65](https://github.com/QahNaaRlN/grace/pull/65), merge `1ada384`; PR CI `39170768987`, main CI `39171530255`, Delivery `39172115986` | integrated; gates PASS |
| Исторический production release | [39172255492](https://github.com/QahNaaRlN/grace/actions/runs/39172255492): deploy/internal health PASS, 2026-10-19 03:28:08 +05:00 / 2026-10-18 22:28:08 UTC | `1ada384` deployed; последующий activation step FAIL |
| Отдельный public smoke | `./scripts/check-public-entrypoint.sh https://graceapp.ru`, 2026-10-19: `/`, `/login`, `/api/meta` 200; protected routes 401 | PASS; не означает AI enabled |
| Первое включение | Тот же run, 03:28:14 +05:00: synthetic quality gate FAIL; public post-enable smoke skipped | disabled; smoke не разрешил изменение флага |
| Уточнение synthetic mapping | Ветка `codex/statement-ai-activation`, prompt v7: columns PASS; cards merge FAIL ([39173759321](https://github.com/QahNaaRlN/grace/actions/runs/39173759321)); конфигурация приложения не менялась | частичный результат, общий smoke FAIL |
| Исторический provider FAIL | [39174108946](https://github.com/QahNaaRlN/grace/actions/runs/39174108946), 2026-10-19 03:53 +05:00 / 2026-10-18 22:53 UTC: columns PASS, cards provider FAIL | historical billing uncertain; сверено ниже |
| Историческая readiness | [39174447002](https://github.com/QahNaaRlN/grace/actions/runs/39174447002): `statement_qualification_usage_requires_review` | FAIL на тот момент; сверено ниже |
| Приватная сверка расхода | Owner предоставил ID; read-only metadata match [39217122586](https://github.com/QahNaaRlN/grace/actions/runs/39217122586) PASS; settlement [39217226634](https://github.com/QahNaaRlN/grace/actions/runs/39217226634), 2026-10-19 12:38:04 +05:00 / 07:38:04 UTC | точный расход подтверждён, ledger reconciled, дальнейшая qualification разрешена |
| После сверки: v7 + merge fix | [39217497119](https://github.com/QahNaaRlN/grace/actions/runs/39217497119), 2026-10-19 12:41:18 +05:00 / 07:41:18 UTC: columns PASS, cards финансовые поля PASS, coverage FAIL (`unmapped_spans:4`) | overall FAIL; расход известен и settled |
| Read-only source diagnosis | [39218054190](https://github.com/QahNaaRlN/grace/actions/runs/39218054190), 12:46:13 +05:00: повторная currency и три labels второй строки | PASS; без inference |
| v8 coverage-audit candidate | [39218319463](https://github.com/QahNaaRlN/grace/actions/runs/39218319463), 12:48:57 +05:00 / 07:48:57 UTC: columns PASS, cards FAIL (лишняя неполная строка и coverage) | REJECTED; v8 prompt откачен, не публикуется как улучшение |
| Исправления mapping/диагностики/billing | Рабочая ветка; исходы неуспешных probes сохранены | candidate; ещё не deployed |
| AC12 representative corpus / Jev comparison / cascade | Малый activation smoke не заменяет полный benchmark | NOT_RUN |
| Private bank inference / owner functional acceptance | Агентом не выполнялись / результата владельца ещё нет | NOT_RUN / pending |

## Политика и границы

Только `google/gemini-2.5-flash`, `google-vertex/eu`, Google, text-only,
reasoning explicitly disabled, no fallback/ZDR/deny. Принятые ceilings:
input 300000/output 2500000 micros/1M. Перед активацией endpoint metadata
повторно проверяет серверный verifier.
[Первичный каталог endpoint](https://openrouter.ai/api/v1/models/google/gemini-2.5-flash/endpoints)
проверен 2026-10-19. Текущий production prompt: `statement-span-mapping.v10.lexical-records` (S1.1);
исторические v2/v7/v8 не являются текущим runtime profile.

Исторический pre-S1 activation smoke использовал два synthetic PDF
(columns/cards): ≤4 calls, reserve ≤32000 micros ($0.032). Текущий separately
accepted S1/S1.1/S1.2 smoke: ≤2 calls / $0.14 aggregate, 300 s/run, прежний
ledger. Это ceiling одного запуска, не сумма фактических расходов серии probes.
Лимит/fees не увеличивались. Raw PDF/images и личные документы провайдеру не
передавались; smoke не создаёт import draft или financial events. Go проверяет
row count, operation/booking clocks, account/FX amounts/currencies, direction,
printed turnovers, coverage и различие available/book balances. Все критерии
остаются обязательными; частичный PASS не включает настройку.

## Сверка расхода и следующий шаг — 2026-10-19

После неуспешного cards probe общий ledger остановил calls из-за неизвестного
расхода. Владелец предоставил два generation ID. Серверный GET generation metadata
однозначно сопоставил их с известным columns call, оставшимся cards call,
моделью/провайдером, временем и единственной соответствующей reservation.
Из exact authoritative cost с округлением вверх до micros подтверждён общий
расход запуска через существующий owner-scoped `grace_ai_settle`. Неизвестный
расход не обнулён по предположению. Allowance/fees не изменены.

Settlement run `39217226634` PASS, 2026-10-19 07:38:04 UTC. Приватный audit
`/opt/grace/shared/statement-reconciliation-39174108946.json`, mode 0600,
сохраняет original report digest, upstream metadata и ledger before/after.
Ключ не покидал VPS, ID/суммы не публиковались в Git/CI. Временный GitHub secret
с ID удалён; временные workflow controls не входят в итоговый diff.
Metadata не установила точную причину исходного provider failure; конкретный
finish reason не предполагается. Сверка расходов не является quality PASS.

Следующий gate — успешные columns/cards smoke исправленного кандидата,
проверки точного SHA, release и runtime policy/flag. До этого Statement AI false.
Повторный synthetic probe `39217497119` после reconciliation завершился
quality FAIL: четыре непокрытых locators. Read-only diagnosis установила их
роли, после чего один bounded v8 probe `39218319463` также дал FAIL с
регрессией числа/полноты строк. V8 отклонён, сохранён v7 + проверенный merge fix.
Никакая настройка приложения не менялась, личные документы не передавались.
Дальнейшие paid probes этой редакции остановлены; результат не повышен до PASS.
Следующий модельный путь требует отдельно проверенного кандидата/контракта;
публичная доступность модели не является evidence пригодности.

## Исправления и проверяемые критерии

Это исправления принятой SPEC-0055, не расширение финансового scope:

| Критерий | Исправление / executable evidence |
| --- | --- |
| AC3/AC4: только grounded значения | Dynamic money span enum; ровно один currency locator на поле; явная валюта каждой строки. `go test ./internal/statementextract` проверяет состав enum и запрет пропуска context/source spans |
| AC4/AC7: объединение header и полнота | Labels участвуют в merge/coverage только вместе с непустым значением из того же чанка; orphan hints не удаляют явный service. Неизвестные IDs и конфликтующие реальные labels/values по-прежнему FAIL. `TestMergeUnboundContextLabelDoesNotConflictWithBoundHeader`, `TestUnboundLabelsNeverClaimCoverageOrRemoveExplicitService` (RED воспроизведён, GREEN PASS) |
| AC4/AC7: полное покрытие без изменения денег | Только явно классифицированный service set упорядочивается по документу; неизвестные/повторные/пересекающиеся IDs и неверный порядок финансовых selections отклоняются. Те же unit tests |
| AC9/AC10: cap, audit, отсутствие paid replay | Проверенные route/ID/cost сохраняются отдельно от непригодного JSON; неизвестный billing остаётся unknown. `TestRejectedOutputRetainsVerifiedBillingWithoutReturningContent`, `TestStatementReasonerRetainsOnlyVerifiedBillOnProviderError` |
| AC11: безопасная диагностика | `TestSyntheticDiagnosticDoesNotExportValuesReasonsOrBilling`; private synthetic traces, наружу только статические codes и source locator IDs |
| AC12: качество реальной модели | columns PASS v7; cards FAIL; полный корпус/Russian/Jev/cascade NOT_RUN. Fake tests не считаются model qualification |

Usage/trace report хранится в `/opt/grace/shared/statement-qualification-*.json`
с mode 0600. Бюджет, остаток, tokens, generation IDs и actual cost не
публикуются в Git/CI; диагностируются только фиксированные synthetic имена,
коды ошибок и source locator IDs. Приватный reference требует document-specific
consent; запись финансовых данных требует отдельного owner confirmation.
Jev выключен.

## Локальные проверки исправлений — 2026-10-19

- `make test`: PASS с `VITEST_MAX_WORKERS=2`, 87 web suites / 330 tests,
  Go tests, lint/FSD/format/build и offline Jev checks. Первый запуск при
  параллельной нагрузке завершился двумя UI ожиданиями FAIL; полный повтор
  прошёл без изменения web-кода/тестов.
- `make test-production`: PASS (activation/rollback/replay/disable и прежние
  deployment/restore/off-site backup regression checks).
- `./scripts/test-bank-statements.sh`: PASS — real PDF/helper → fake provider →
  PostgreSQL/confirmation, schema/restore и downgrade guard. Это не live model quality.
- `go vet ./cmd/... ./internal/...`, targeted Go tests после последней правки
  billing и `git diff --check`: PASS.
- Отдельный public entrypoint smoke: PASS; owner session/functional acceptance
  не использовались и не объявляются пройденными.

Локальные логи вне Git: `/tmp/grace-statement-followup-test.log` (первый FAIL),
`/tmp/grace-statement-followup-test-2workers.log` (PASS),
`/tmp/grace-statement-followup-production.log`,
`/tmp/grace-statement-followup-bank.log`. CI/release результаты PR #65 выше
относятся к `1ada384`; исправления остаются кандидатом до отдельных gates.

## Проверки после reconciliation — 2026-10-19

- `make test` после merge fix: PASS (включая 330 web tests); v8 regression suite
  также проходит, но live quality FAIL, поэтому эта prompt-редакция отклонена.
- `./scripts/test-bank-statements.sh` после merge fix: PASS. Повтор для v8
  FAIL: `fixture plan split unexpectedly 2` — увеличенный prompt изменил
  chunk boundary. Это дополнительное основание отклонения v8. Финальный v7
  product code побайтно совпадает с прошедшим после merge fix (`e484964`);
  targeted Go tests после отката PASS. Fake-provider regression не доказывает
  качество настоящей модели.
- Source coverage и реальные conflicts не ослаблены. Model quality блокирует
  production activation даже при зелёных unit/integration gates.
- Временный reconciliation/candidate/source-diagnostic код удалён из итогового
  diff. Только приватные защищённые reports/audit остаются на VPS. Общий бюджет
  и fees не менялись; metering updates не являются финансовыми Posting.

Финальный v7 regression evidence: `/tmp/grace-statement-reconciled-test.log`,
`/tmp/grace-statement-reconciled-bank.log` (PASS); production gates
`/tmp/grace-statement-v8-production.log` (PASS, итоговый workflow тот же).
Отклонённые v8 результаты сохранены отдельно в `/tmp/grace-statement-v8-test.log`
(unit/web PASS), `/tmp/grace-statement-v8-bank.log` (plan-boundary FAIL) и live
run выше. Они не переименованы в PASS и не относятся к финальной prompt policy.

## S1.1 release — 2026-10-19

[PR #73](https://github.com/QahNaaRlN/grace/pull/73) → `ef782ec`; PR CI
`39253076844`, main CI `39254418192`, Delivery `39255451276` PASS.
Run `39255669019` internal health PASS, activation FAIL; текущий baseline
выключен. [Public-source replay](../research/statement-quality/s11-description-offline.json)
сопоставил missing separator с service без нового inference. Cards NOT_RUN;
полная AC12 qualification, Jev/cascade и owner private functional test открыты.
