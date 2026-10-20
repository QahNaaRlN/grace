# SPEC-0055 — implementation evidence

- Дата: 2026-10-19
- Основание: [принятая SPEC-0055](0055-bank-statement-pdf-import.md) и
  [принятый технический контракт](0055-technical-contract.md), owner approval
  «да, утверждаю». Принятая спецификация интегрирована PR #62, main `4a6f1e7`.
- Implementation branch: `codex/bank-statement-pdf-import`.
- Implementation commit: `8e37465ae4bd88b8b12cc3b81b1913c4cf243b9d`.
- Статус кода: `integrated`, PR #64, `b51b74f`, post-merge CI `39167930999` PASS.
  Это executor evidence, не owner functional acceptance; AC12 остаётся открытым.
- Production: первый manual release `b51b74f`, 2026-10-19 02:37:31 +05:00.
  Текущий `7b2deea` (S1.2 PR #74), main CI `39395041781`, Delivery
  `39396018990`, deploy `39396162864` **PASS**. Оба synthetic cases 19/20 PASS;
  **AI enabled**, actual-image/runtime/policy/public checks подтверждены
  2026-10-20 13:51:07 +05:00 / 08:51:07 UTC.
  Private inference/импорт личных операций агентом не выполнялись.
  [Текущее evidence D1–D5](0055-s12-evidence.md), [история FAIL](0055-activation-evidence.md).


## Реализованный путь

Локальный bounded Poppler helper → immutable page/span layout → ручное
сопоставление либо document-scoped consent/LLM mapping → Go grounding,
normalization/coverage/reconciliation → persisted `imports` draft → существующие
matching/preview/confirmation. Модель выбирает spans, не передаёт доверенные
числа и не получает financial tools. `statementextract` зависит от `imports`,
ядро не зависит от AI. Отдельная statement policy выключена по умолчанию.

Новые миграции 27/28 расширяют documents и существующий budget ledger, добавляют
source/audit/consent/attempt/call evidence с FORCE RLS и owner compound FK.
Финансовый writer не дублируется. Confirm проверяет source revision и owner
review страниц/ролей; после первой публикации source account/grouping неизменны.

## Критерии и воспроизводимые доказательства

PASS ниже означает указанные offline assertions. Он не доказывает точность
live модели, универсальную поддержку банков или все мыслимые crash schedules.
Воспроизведение PostgreSQL/helper проверок требует Docker.

| AC | Фактическое evidence | Состояние |
| --- | --- | --- |
| AC1 layout/grounding | `scripts/test-pdf-extractor.sh`: настоящий Poppler для `columns.pdf` (таблица/две страницы) и `cards.pdf` (вертикальные блоки); `TestRealPDFLayoutPipeline`, `TestLayoutGrounding`, `TestLayoutLimitsAndUntrustedXML`, `TestProposalCoverageRolesAndAmounts`. Полный real PDF→fake provider→SQL в `TestStatementRealPDFFakeProviderLifecycle`. Частный reference проверен только локально, без model mapping | Offline PASS; real-reference model accuracy UNVERIFIED |
| AC2 no pre-confirm write | `TestStatementRealPDFFakeProviderLifecycle` проверяет неизменность events/postings/receipts/scheduled_payment_settlements/opening до confirmation; `TestStatementOfflineLifecycle`, `TestStatementExtractionConsentMeteringReplay`: persisted source/draft/attempt без AI-ledger write | PASS |
| AC3 typed/missing balances | `TestMoneyExactAndBounded`, `TestLocalClocksAndProjection`, `TestReconciliationDoesNotInventBalance`, `TestDateAndClockAcrossAdjacentLinesRetainBothClocks`, оба PDF fixture: отсутствие book balance, отдельный available, mismatch, FX, overflow, operation/booking и неизвестная zone | PASS |
| AC4 one fact/multiple sources | `TestStatementOfflineLifecycle`: другой PDF связывается с Posting существующей purchase/Receipt, новая provenance без второго расхода; `TestStatementPostingEvidenceMatchesEachTransferSide`: обе стороны transfer и неправильные amount/account/currency/date | PASS для указанных cases |
| AC5 meaning | `TestStatementCreditRequiresOwnerMeaningAndBorrowingIsRetained`, statement lifecycle guards, explicit owner meaning в web review; fake provider PDF test исключает own-transfer строку с причиной, financial coverage остаётся incomplete | PASS для guards; model meaning quality UNVERIFIED |
| AC6 atomicity/replay | `TestStatementOfflineLifecycle`: late SQL failure откатывает ledger и не удаляет original; confirm replay. `TestStatementIsolationRegroupAndConcurrentManual`, `TestStatementPartialConfirmationAndDirectSQLCleanup`: конкуренция/regroup/revisions. Extraction integration: stale owner edit и replay без повторного provider call | PASS для перечисленных interleavings |
| AC7 isolation | `TestStatementIsolationRegroupAndConcurrentManual`, extraction integration: чужой owner/admin в HTTP/SQL, RLS и compound FK. Routes под существующими auth/session/CSRF/Origin middleware; baseline auth suites в integration | PASS |
| AC8 retention | lifecycle/partial-confirm/direct SQL cleanup, keep/expiry; down guards и неизменность card terms. Extraction integration проверяет cancel/stale result и удаление plan с сохранением policy/provenance. Полная очистка только после всех строк; metadata-only delete не скрывает retained PDF bytes | PASS для указанных offline cases |
| AC9 provider/cost | `TestStatementExtractionConsentMeteringReplay`, `TestPlanRedactionBoundsAndFullCoverage`, `TestMergeRejectsFabricatedSpansAndConflictingTotals`, `TestStatementReasonerUsesTextOnlyBoundedTransportAndRetainsBillingOnBadJSON`: scoped consent, no call before consent, malformed/unknown cost, shared freeze, stale edit, 32 lifetime calls, redaction, model/endpoint/caps. Web AI-off/manual и explicit consent tests | Offline PASS; actual provider usage/privacy/latency UNVERIFIED |
| AC10 coverage/UX | Pure coverage/duplicate/role/adversarial checks; schema generated из OpenAPI; 330 web tests, lint/FSD/format/build. Browser 375×812 на synthetic PDF: upload/manual save/roles/pages/reload/AI-off, Tab focus, без horizontal overflow; 409 test сохраняет ввод и обновляет revision. Source literal рендерится как текст | Offline PASS; live prompt-injection resistance UNVERIFIED |
| AC11 regression/storage | `make test`, `make test-integration`, `make test-production`, `go vet ./cmd/... ./internal/...`, race новых pure/orchestration packages, `git diff --check`. `test-bank-statements.sh`: empty down/up, real helper→fake provider→SQL confirmation, pg_dump/restore новой схемы/readback, down с evidence обязан отказать | PASS локально; CI точного PR head отдельно |
| AC12 bounded model evaluation | LLM-only vs Jev-selection vs cascade не запускались; нет field/row/column accuracy, omissions, abstention, latency и actual cost измерений реального endpoint | NOT_RUN; блокирует полную qualification/acceptance |

`TestStatementRestoredEvidence` запускается скриптом после реального pg_dump/restore.
Integration tests требуют своих environment flags; обычный `go test` не заменяет
эти скрипты и не доказывает БД/assertions, если suite был skipped.

## Локальные результаты первого implementation snapshot и границы

- `make test`: 330 Web tests плюс Go, ESLint, Steiger, Prettier, generated API,
  production web build и offline Jev evaluation checks.
- `make test-integration`: полный набор пройден дважды; после последнего изменения
  pure date/clock grounding и добавления общего PDF test повторно пройден
  `scripts/test-bank-statements.sh` с настоящим PDF→helper→fake provider→SQL.
- `make test-production`: synthetic restore/readiness/deployment/offsite tests PASS;
  это не production deployment.
- Helper cancellation проверяет process group, busy/timeout/file cleanup и
  output cap. Default compose не запускает helper; opt-in sandbox проверяется
  `scripts/test-pdf-extractor.sh`. CI matrix собирает helper image.
- Browser обнаружил duplicate React keys у source forms; исправлены отдельными
  owner/manual/AI keys и повторно проверены после reload. Baseline HydrateFallback
  warning остаётся; новых duplicate-key ошибок после исправления не было.
- В окружении появился посторонний read-only `/tmp/.git`: существующий privacy
  guard CLI корректно запрещал temp-output внутри распознанного repository.
  Guard не ослаблялся, чужой путь не удалялся. Финальные CLI checks запускались
  с `TMPDIR` в собственном каталоге вне `/tmp`; это условие воспроизведения
  для окружения с таким посторонним `.git`, не изменение поведения продукта.

Verbose локальные logs остаются вне Git: `/tmp/grace0055-test-complete.log`,
`/tmp/grace0055-integration-final.log`, `/tmp/grace0055-production-final.log`,
`/tmp/grace0055-bank-full-pdf-final2.log`, `/tmp/grace0055-vet-complete.log`,
`/tmp/grace0055-race-complete.log`. Они не являются постоянным CI artifact.
Screenshot synthetic mobile review: `/tmp/grace0055-mobile-summary.jpg`.

## Частный reference: только локальный preflight

Предоставленный владельцем PDF прочитан read-only в локальном контейнере без
сети; отдельный helper также network=none. Preflight не использует БД, credentials
или провайдера; stdout содержит только counts/bounds. Оригинал не изменён.
Историческая проверка prompt/redaction исходного implementation head: 5 страниц, 1661 span,
16 planned calls, max messages+schema **8963 bytes** (лимит 9000),
maximum **128000 micros ($0.128)** по утверждённым ceilings, **providerCalls=0**.
Это reservation plan, не actual token usage, не обещание completion ≤120 s
и не доказательство полноты/точности model extraction.

PDF/text/images/личные строки/суммы/идентификаторы не добавлены в Git/CI и
не отправлены OpenRouter/TypeSafe. В Git только явно синтетические fixtures.

## Оставшиеся gates — актуализация 2026-10-19

Review/integration/main CI/первый production release кода завершены (PR #64);
механизм активации также интегрирован (PR #65). Это не полный model benchmark.

1. Unknown billing последнего synthetic cards probe точно сверён
   2026-10-19 12:38:04 +05:00 (run `39217226634`). Флаг остаётся false;
   сверка не заменяет успешную проверку качества.
2. Завершить cards smoke и выпустить проверенные исправления; включить AI только
   при успешных обоих synthetic layouts и проверенной runtime policy.
3. AC12: отдельный opt-in benchmark на synthetic/de-identified fixtures с cap
   в прежнем бюджете: field/row/column accuracy, omissions, grounding,
   abstentions, usage/cost/latency, LLM-only/Jev/cascade. Columns PASS v7
   не доказывает качество других форматов или личной выписки.
4. Owner functional acceptance. Приватный reference передаётся только после
   document-specific consent; финансовая запись требует отдельного подтверждения.
   При измеренной пользе Jev нужны отдельные privacy/consent policy; он выключен.

[Гайд владельца и локального оператора](../bank-statements.md).


## Owner release instruction — 2026-10-19

Владелец явно поручил «Выполнить деплой, я проверю на месте». Это разрешение
включает integration/release для ручной проверки; AC12 и statement AI activation
не объявляются выполненными. Production wiring добавлено в этот PR: публикация
exact-SHA helper image, production sandbox/socket/healthcheck и non-root smoke.
Deployment regression проверяет отказ rollback на старый image с сохранёнными
PDF sources и разрешённый rollback на совместимый image. На этом историческом этапе это был release candidate; фактические результаты
выпуска приведены ниже, текущая активация — в отдельном evidence.


## Подтверждённый выпуск — 2026-10-19

PR #64: main `b51b74f`, CI `39167930999` PASS, Delivery `39168461596` PASS,
Deploy production `39168570754` PASS. Backup проверен и зашифрованная off-site
копия загружена; migrations 27/28 применены. Helper non-root/socket smoke,
API readiness и публичный auth smoke прошли. HTTPS `/`, `/login`, `/api/meta` =200;
`/api/auth/me`, `/api/v1/accounts`, unauthenticated POST PDF =401. Опубликованный
imports chunk содержит PDF UI и upload route. Statement AI на этом выпуске был
false; более ранние разделы описывают offline этап и не отменяют этот release.
Owner функциональную проверку обещал выполнить на месте; результат ещё не получен.
