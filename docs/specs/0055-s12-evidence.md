# SPEC-0055 / S1.2 implementation evidence

- Дата актуализации: 2026-10-20 (Asia/Yekaterinburg).
- Accepted contract: [description separators](0055-s12-description-separators.md), прямое решение владельца «Утвердить S1.2, один smoke и включение при PASS». Принятие записано в `bf4b3c8` до production-кода.
- Implementation integrated: [PR #74](https://github.com/QahNaaRlN/grace/pull/74), merge `7b2deeaf65fe05d2da0f2cffedb7f14be89e67c1`, 2026-10-20 08:40:00 UTC / 13:40:00 +05:00. Branch `codex/statement-description-separators`, base `ef782ec`; PR CI `39393961404` — все 5 jobs PASS.
- Production baseline: S1.1 `ef782ec`, deploy `39255669019` 2026-10-19, internal health PASS; smoke columns `record_description` FAIL, cards NOT_RUN, flag false. Это использованный S1.1 run, не новое разрешение S1.2.
- После восстановления quota автоматическая проверка разрешила запись в worktree. Ранее подготовленный `/tmp/grace-s12-review/S1.2.patch` применён через неё, без обхода. Один S1.2 paid smoke использован, оба cases PASS; enable подтверждён ниже.

## Изменение и проверка D1–D5

| ID | Evidence |
| --- | --- |
| D1 | [Public-source replay](../research/statement-quality/s11-description-offline.json), 0 provider calls / 0 SQL. Печатный `p2-b8-l1-w3` (`-`) пропущен из второй description в service; первая exact. Кандидат messages+schema 9060/9401 bytes, оба <90000. Исторические source hashes относятся к S1.1, не к текущему v11 |
| D2 | В `internal/statementextract/plan.go` добавлена ровно принятая инструкция; PromptVersion `statement-span-mapping.v11.description-separators`. Alias/planner/redaction/schema/scorer/report versions, tuples, лимиты, privacy и financial Go не меняются. Existing hashed-plan/profile validation требует fresh preview/consent; старый prompt несовместим |
| D3 | `TestActivationRealHelperSmoke/missing_description_separator`: actual pinned helper → delivered-tuples-only fake → Go/scorer. Negative переносит separator в service, сохраняя financial fields и coverage. Обязательный исход: 2 rows / 19 checks / 18 correct, единственный code `record_description`, без второго call, известный fake bill settled. Complete baseline PASS; остальные negative checks сохранены |
| D4 | [Actual planner proof](../research/statement-quality/s12-evidence/summary.json): **84 PASS**, pinned helper network-none/read-only UID10001, source hashes соответствуют текущему коду. `make test`, `make test-production`, `go vet ./cmd/... ./internal/...`, `git diff --check` **PASS** (exit 0). Полный `make test-integration` также **PASS**, включая bank/negative/SQL/restore и baseline suites. Все обязательные локальные команды завершены exit 0; PR CI `39393961404` 5/5 PASS. Main CI `39395041781` 5/5, Delivery `39396018990` 4/4, post-merge `make test`/`go vet`/`git diff --check` **PASS**; primary clean перед release |
| D5 | **PASS**: единственный run `39396162864`, columns **19/19**, cards **20/20**; strict current-profile summary `activationSmokePassed:true`, `syntheticCases:2`. Enable/health 08:51:05 UTC, exact API image/runtime `GRACE_STATEMENT_AI_ENABLED=true` + current policy 08:51:07 UTC, public/auth 200/401 08:51:10 UTC. [Safe actual payloads](../research/statement-quality/s12-activation-runtime.json). Никакого повторного paid run |

Local logs: `/tmp/grace-s12-test.log`, `/tmp/grace-s12-integration.log`,
`/tmp/grace-s12-production.log`, `/tmp/grace-s12-vet.log`,
`/tmp/grace-s12-structural.log`. PASS определяется завершённым exit status.
При наличии постороннего read-only `/tmp/.git` CLI privacy guards сохраняются;
для финальных CLI gates использовать собственный TMPDIR вне этого repository.

Никакая автоматическая починка separator/нормализация description не добавлена.
Scorer остаётся строгим, 19/20. Live v11 прошёл оба fixed synthetic cases один раз; это не доказательство устойчивости на других выписках. Ни successful fake, ни byte/structural PASS не заменяют live evidence. Full AC12/Jev/cascade/corpus и owner private acceptance остаются самостоятельными этапами.

Read-only baseline 2026-10-20: run `39393586879` PASS, fixed payload `statement_ai_runtime_disabled`, activation prerequisites passed; без inference/configuration writes.

## Verified owner-test activation — 2026-10-20

Run [39396162864](https://github.com/QahNaaRlN/grace/actions/runs/39396162864)
завершился SUCCESS. Actual API `Config.Image` совпал с
`ghcr.io/qahnaarln/grace-api:sha-7b2deeaf65fe05d2da0f2cffedb7f14be89e67c1`;
workflow проверил **runtime** флаг и выполнил fresh `grace-statement-admin
-mode=check` (`statementAI:true, policyVerified:true`). После этого `/`,
`/login`, `/api/meta` вернули 200, `/api/auth/me` и `/api/v1/accounts` — 401,
без Basic challenge. Это наблюдение фактического процесса после enable,
не вывод из env-файла или намерения.

Strict summary текущего binary вызвал ValidateReport: profile/scorer, обе
fixture hashes/plan evidence, 19/20 checks, ровно 2 calls, known verified billing,
no errors/validation/coverage, digest/freshness — обязательны. Usage, cost,
remaining и generation IDs остались в mode0600 audit на VPS, не в Git/CI.
Ни личная выписка, ни imports/events в smoke не использовались.

Owner UI/private functional acceptance **pending**: Импорты → один debit RUB
счёт → текстовый PDF → новый AI preview/consent → review → financial preview и
явное подтверждение. Старый preview требует обновления. Shared budget/fees,
explicit consent, global kill switch и ручной путь сохраняются. S2–S5, полный
AC12 и SPEC-0056 не объявляются принятыми/выполненными этим smoke.

## Независимая post-enable readiness

Read-only [39396521229](https://github.com/QahNaaRlN/grace/actions/runs/39396521229)
PASS 2026-10-20 **08:52:24 UTC / 13:52:24 +05:00**: direct
`statement_ai_runtime_enabled`; activation prerequisites PASS, shared allowance
готов и unknown usage не блокирует. [Safe payloads](../research/statement-quality/s12-postenable-readiness.json)
не содержат величину бюджета/расхода. Никакого configuration write/inference.
