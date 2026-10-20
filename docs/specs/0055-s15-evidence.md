# SPEC-0055 / S1.5 evidence

Дата: 2026-10-20, Asia/Yekaterinburg.
[Контракт](0055-s15-screening-diagnostics.md) accepted прямым «Утверждаю»;
commit `074d1fb` предшествует production-коду `a4d81db`. Ветка
`codex/statement-candidate-smoke`, существующий draft [PR #78](https://github.com/QahNaaRlN/grace/pull/78).

## Изменение

Только отдельный screening path добавляет closed diagnostic и private unverified
correlation. При отказе после decode ID сохраняется даже до validation model/route,
но только с формой gen-[0-9A-Za-z-]+ и длиной <=128. Этот ID не становится Billing.
HTTP status и Content-Type хранятся только как integer/category. Нет raw headers,
body, provider error text, request content или ключа в diagnostic.

Новый report `statement.screening.v2.gemini31-diagnostics` имеет optional failure;
ошибка сохраняет safe code. Failure не добавляется к verified calls и не разрешает
activation. Unknown cost по-прежнему передаёт nil в существующий settlement;
known billing при content failure сохраняется. Ни автоматической сверки, ни retry.
Public screening-summary использует закрытый vocabulary для diagnostic/error и
scorer/validation/coverage codes; неизвестное становится unknown, correlation ID
и billing не публикуются. Product transport сохраняет прежние Error/BillingKnown;
production consent, model/alias checks, endpoint, schema и scorer не расширяются.

## Критерии и исполняемое evidence

| ID | Evidence |
| --- | --- |
| D1 | `go test ./internal/assistant -run 'TestScreeningFailureDiagnosisSeparatesEnvelopeRouteAndContent\|TestScreeningDiagnosticsIOAndProductCompatibility' -count=1`: 24 envelope/route/usage/content cases на обоих путях; 11 I/O/status/pre-request cases на обоих путях. Error/ProtocolReason/BillingKnown и acceptance сохраняются. Screening различает все guards; invalid request до HTTP, transport/timeout, 400/422/429/503/302, broken reader, oversize, SSE, JSON, missing Content-Type |
| D2 | `go test ./internal/statementqualification -run TestScreeningDiagnosticNeverAuthorizesOrSettlesCorrelation -count=1`: private diagnostic извлечён через structured-error interface, 1 reserve/1 call/1 settle, known100 или nil для unknown, невычисленные scored results пусты при первом failure. Correlation не становится verified Call/ActualMicros. Предыдущие успешные case results при более позднем отказе сохраняются прежним runner |
| D3 | `TestScreeningDiagnosticSanitize`, `TestScreeningSummaryOnlyExportsClosedCodes`: malformed/oversized/line-break ID, invalid status/Content-Type и неизвестный reason очищаются; PRIVATE sentinels из error/request/body/header/scorer fields и ID/cost не появляются в summary. Тесты D1 проверяют fixed error text, не upstream strings |
| D4 | `TestScreeningReportNeverActivates`, `TestScreeningDiagnosticNeverAuthorizesOrSettlesCorrelation`: новый report и любой report с failure не активируют product; legacy v1 не проходит новый screening validator. `scripts/test-statement-screening.sh` из make test-production: прежний fixed marker запрещает replay, в том числе с другим SHA, и private report остаётся 0600 |
| D5 | Unit пакеты assistant/statementqualification/statementextract/statement-admin PASS; `scripts/test-bank-statements.sh` exit0: pinned helper + fake provider columns19/19, cards20/20, RU23/23, negatives, SQL lifecycle/restore/unknown-billing; `make test-production` exit0, affected `go vet` и `git diff --check` PASS. `TMPDIR=/var/tmp/grace-quality-next-tests VITEST_MAX_WORKERS=1 make test` exit0: Go + 88 web files / 334 tests + lint/FSD/format/build/Jev PASS |

Последние pre-request/malformed-ID regressions добавлены после начала полного
make test; четыре affected Go packages повторно прошли targeted checks. Production
behavior относительно `a4d81db` после этого не менялся. Current-head remote CI
отдельно, результат старого live screening не подменяется локальными checks.

## Граница результата

External inference=0. Нет deployment, runtime/model/flag/budget/fees изменений,
новых secrets, settlement или обращения к production DB. Old S1.4 report, marker
и authoritative reconciliation на VPS не изменялись. Новый диагностический
report не выпущен в runtime и не является новой авторизацией smoke.

Исходная причина S1.4 provider failure остаётся UNVERIFIED: новый код не может
восстановить утраченный HTTP envelope. Качество Gemini 3.1 Flash Lite не измерено.
Следующий live run требует отдельного решения, нового marker и bounded cap;
повтор S1.4 по прежнему marker не разрешён и не реализован.

Локальные журналы: `/tmp/grace-s15-unit-final.log`, `/tmp/grace-s15-bank.log`,
`/tmp/grace-s15-production.log`, `/tmp/grace-s15-make.log`.
