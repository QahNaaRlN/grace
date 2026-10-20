# SPEC-0055 / S1.4 evidence

Дата: 2026-10-20, Asia/Yekaterinburg. [Контракт](0055-s14-candidate-screening.md)
accepted commit `a94cbb4` предшествует коду. Baseline `7c4ef83` включает локально
проверенную S1.3. Ветка `codex/statement-candidate-smoke`.

## Локальная проверка

| ID | Исполняемое evidence |
| --- | --- |
| C1 | `TestScreeningCannotAuthorizeProductConsent`: product Policy.Valid/BuildPlan/CanonicalProposal и persisted-plan guard отвергают candidate. Только exact bounded pair работает в BuildScreeningPlan; stale/tampered plan не проходит |
| C2 | `TestScreeningTransportIsExplicitAndPinned`: fake HTTP проверяет model/only EU/ZDR/deny/no-fallback/reasoning disabled/16000 output. Malformed answer сохраняет known billing; отсутствие privacy membership останавливает POST. Обычный reasoner и provider по-прежнему отвергают candidate |
| C3 | `TestScreeningRealHelperThreeCases`: pinned Poppler → columns19/19 + cards20/20 + RU23/23. Whole-run reservation161700, один settle300 synthetic micros, 3 calls. RU omitted description FAIL, без retry. Это fake-provider evidence, не измерение Gemini |
| C4 | `TestScreeningStopSettleAndPreflight`, `TestScreeningReportNeverActivates`: wrong order/hash/policy до reservation, known error billed, unknown freezes; report с другим version/profile никогда не разрешает activation. `scripts/test-statement-screening.sh`: failed preflight без marker; failed run сохраняет report600/marker и не повторяется, включая другой SHA; billing не попадает в stdout |
| C5 | PASS: `TMPDIR=/var/tmp/grace-quality-next-tests VITEST_MAX_WORKERS=1 make test` (Go + 88 web files / 334 tests + lint/FSD/format/build/Jev); `scripts/test-bank-statements.sh`; `make test-production`; `go vet ./cmd/... ./internal/...`; `git diff --check`; YAML/exclusive-job guards |
| C6 | **FAIL**, [run 39459614043](https://github.com/QahNaaRlN/grace/actions/runs/39459614043), source `e36dbcbb5c57ed8b2060cc2bcbc1ebc8482fd728`, 2026-10-20 16:41:56 UTC. `qualification_provider_failed`, results empty, до первого score. Application image/health preserved. Автоповторов нет; качество кандидата не измерено, billing read-only audit: unknown / uncertain, 0 recorded calls / 0 scores |

Remote harness добавлен как explicit `statement_screening_only` в существующий
workflow. Deploy/inspect jobs взаимоисключены с screening. Exact-source CI обязателен
до запуска; binary/fixtures manifest проверяется на VPS; прежний API container/image
и health проверяются после, config/flags/drafts не меняются. Authorization marker
`s14-gemini31-20261020` общий для разных workflow attempts/SHAs; предварительный
отказ до его создания не считается платным выполнением. После marker нет paid
replay даже при interruption. Приватный report остаётся в `/opt/grace/shared/`,
временный bundle и container directory очищаются. Public logs только fixed codes,
check counts и activationAuthorized=false; usage/generation data остаются приватны.

Прямой SSH из локальной сессии недоступен (`Permission denied`); используется
existing GitHub production environment/key. Новые credentials не запрашивались.

Локальные журналы: `/tmp/grace-s14-unit-final.log`, `/tmp/grace-s14-go-final.log`,
`/tmp/grace-s14-bank.log`, `/tmp/grace-s14-make.log`,
`/tmp/grace-s14-production-tests.log`, `/tmp/grace-s14-remote-test.log`,
`/tmp/grace-s14-vet.log`. Они не заменяют exact-source CI или live evidence.

Полная qualification, model registry и product adoption остаются вне этого
пакета; даже полный 19/20/23 PASS не гарантирует все банки. Никакого automatic
production switch или финансового подтверждения по результату screening.

## CI и единичный live run

[PR #78](https://github.com/QahNaaRlN/grace/pull/78), exact-source
[CI 39458315766](https://github.com/QahNaaRlN/grace/actions/runs/39458315766):
все 5 jobs PASS, включая обычный parallel web run, полный make test-integration
и 3 image builds. Screening dispatched ровно один раз, отдельной job;
inspect/deploy skipped. Неуспешный provider run не является quality FAIL модели
или отрицательным результатом scorer: структурированный результат первого
документа до scorer не дошёл. В публичном summary нет authoritative HTTP
reason/status или billing values, точная причина пока не установлена.
Для аудита добавлен read-only fixed-report inspector, который выводит только
booleans/counts и не делает inference, reservation, settlement или config writes.

## Read-only audit и историческая блокировка

[Audit 39460065144](https://github.com/QahNaaRlN/grace/actions/runs/39460065144),
2026-10-20 16:44:29 UTC, source `652ba6b`: fixed report private, expected version/
model/endpoint matched, authorizationConsumed=true, billingKnown=false,
recordedCalls=0, scoredCases=0, passed=false. Это **не доказательство нулевого
расхода или отсутствия upstream request**: отчёт не содержит подтверждённого
billing и generation ID. Existing readiness отдельно вывел
statement_qualification_usage_requires_review; job overall FAIL ожидаем из-за
uncertain ledger, read-only report inspection step прошёл.

На момент audit runtime flag Statement AI оставался enabled, image/health
сохранялись, но shared uncertain ledger блокировал новые платные запросы. Ручной импорт доступен.
Никакого budget reset, вымышленного zero settlement, нового inference или смены
модели не сделано. Для reconciliation был запрошен generation ID из Activity для
Gemini 3.1 Flash Lite / Google EU около 20.10.2026 16:41:56 UTC / 21:41:56 +05.
Запрошен только ID, не API key. [Официальный generation endpoint](https://openrouter.ai/docs/api/api-reference/generations/get-generation)
требует ID; [Analytics API](https://openrouter.ai/blog/announcements/activity-dashboard/)
требует management key, regular inference key его не заменяет. Management key
не запрашивался и не добавлялся. Без authoritative evidence блокировку не снимаем.

## Authoritative reconciliation завершён — 2026-10-20 17:10:51 UTC

Владелец предоставил ID. [Принятый ops-контракт и критерии](0055-s14-reconciliation.md)
исполнены на source `865ccbb`: [read-only inspect 39463492420](https://github.com/QahNaaRlN/grace/actions/runs/39463492420)
и [apply 39463769303](https://github.com/QahNaaRlN/grace/actions/runs/39463769303) PASS.
ID/provider/time/non-BYOK и authoritative cost совпали с единственной reservation.
Generation API возвращает canonical slug, report содержит alias; точная пара
подтверждена свежим официальным GET /models. Это объясняет первоначальный отказ
ops-сопоставления metadata, но не доказывает причину provider-stage failure.

Ровно одна reservation settled через существующую owner-scoped grace_ai_settle;
`billingReconciled=true`, `sharedUncertainBlockCleared=true`. Значения расхода, ID
и before/after остаются в private audit0600 на VPS. Transient GitHub secret удалён,
exact-name lookup вернул 0. Новых inference не было, screening/deploy jobs skipped.
Исходный report остаётся FAIL / billingKnown=false; отдельный reconciliation audit
подтверждает учёт расхода, не переписывая историческое evidence. Качество Gemini
3.1 Flash Lite остаётся UNVERIFIED, authorization consumed, paid retry не разрешён.
Runtime flag enabled и production-модель сохранены. Снятие metering-блокировки
не является новым end-to-end подтверждением работоспособности импорта.

Обязательный `make test` повторён для ops-изменений: exit0, 334 web tests и
Go/lint/FSD/format/build/Jev PASS (`/tmp/grace-s14-reconcile-make.log`). Offline
reconciliation match/mismatch/ceil/replay/canonical checks и diff check PASS.

## Диагностика без paid replay — 2026-10-20 17:18:36 UTC

[Read-only run 39464907701](https://github.com/QahNaaRlN/grace/actions/runs/39464907701),
source `0f8a062`: source-bound saved metadata и committed settlement audit совпали,
runtime/shared-ledger readiness PASS; screening/deploy skipped. 14 synthetic
transport counterexamples PASS, inspector private-output/digest/permissions/symlink
checks PASS. [Полный разбор](../research/statement-quality/s14-provider-failure-diagnosis.md)
отделяет generation metadata от chat response и локальные counterexamples от
утраченного live envelope. Точная первопричина UNVERIFIED, подтверждён пробел
observability. [S1.5](0055-s15-screening-diagnostics.md) подготовлен как draft,
production-код и новые paid calls не выполнялись.

Обязательный make test для diagnostic edits завершён exit0: Go, 334 web tests,
lint/FSD/format/build/Jev PASS (`/tmp/grace-s14-diagnosis-make.log`); diff check PASS.
Это локальные checks; current-head remote CI и live quality остаются отдельными.
