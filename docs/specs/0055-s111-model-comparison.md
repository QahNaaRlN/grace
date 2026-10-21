# SPEC-0055 / S1.11: ограниченное сравнение моделей после grounding guards

- Статус: accepted; implementation_status: integrated (PR №84, merge012f604; C1–C5/post-merge PASS); live_status: not_started.
- Автор: Codex; дата: 2026-10-21, Asia/Yekaterinburg.
- Acceptance owner: владелец Grace; acceptance record: 2026-10-21, прямое «одобряю»
  в ответ на предложение принять offline C1–C5. Разрешены C1–C5; C6 не разрешён.
- Основание подготовки: «Подтверждаю, приступить к работе по выбранной программе»
  после программы post-merge №82 → контракт сравнения → запуск после утверждения.
  Это разрешает подготовку, но не является принятием ранее не предъявленных
  маршрутов, новых правил кампании и суммы расходов.
- Worktree: `statement-model-comparison`, ветка `codex/statement-model-comparison`.
- База: main `2fe0d79041e3177c36cb7b2e6d5a5d2347cb9509`, PR №82 / accepted S1.10.
- Implementation branch: `codex/statement-comparison-implementation`, отдельный worktree
  `statement-comparison-implementation` от main `11f26f8` (PR №83).
- Implementation commit: `133ec1293cd93c94b1a52bad1aaad96257fec6db`.
- [C1–C5 evidence](0055-s111-evidence.md): offline PASS; C6 NOT_AUTHORIZED / NOT_RUN.

## Проблема и пользовательский результат

S1.10 обнаруживает внутренние пропуски description и неподтверждённые роли
книжных остатков до финансового подтверждения. Он не улучшил ответы модели.
S1.8 остаётся historical FAIL17/19; тот же trace с новым validator даёт16/19.
Нельзя объявлять новую модель эффективной по цене токена или сравнивать разные
scorer/prompt. Нужна одинаковая наблюдаемая база качества, стоимости и задержки.

Результат: воспроизводимая таблица двух exact model/endpoint пар на трёх
публичных документах, включая все неудачи и стоимость. Полный PASS даёт лишь
кандидата для расширенной qualification, не production admission. При отказе
получается честный FAIL/UNVERIFIED с сохранёнными evidence и billing state.

## Scope и non-goals

Предлагаются offline реализация C1–C5 и затем отдельно разрешаемый единственный
live C6. До принятия этого документа production-код не меняется. Даже принятие
только C1–C5 не открывает C6; операторский запуск требует явного решения о
ровно этой кампании на прошедшем CI source SHA.

Scope: изолированные comparison policies/reasoners для двух пар, report/CLI,
preflight и one-shot runner/workflow mode, offline wire/fake/SQL регрессии,
одна кампания до6POST после разрешения C6, read-only aggregate inspector.

Non-goals: product registry SPEC-0056, production allowlist/flags/model switch,
deployment, личные PDF/drafts/financial writes, OCR, расширение корпуса,
planner/prompt/schema/aliases/redaction/grounding/scorer repair, старые run retries,
автоматические fallback/repair, увеличение общего бюджета, qualification всех банков.

## Текущее поведение и доказательства

- [S1.10](0055-s110-grounding-guards.md), [G1–G5](0055-s110-evidence.md),
  [post-merge №82](0055-pr82-postmerge.md).
- [Свежий metadata shortlist](../research/statement-quality/model-shortlist-2026-10-21.md),
  timestamp 07:37:42 UTC, public GET only. A/B unqualified.
- `NewStatementScreeningReasoner`, `ScreeningPolicyValid`, `RunScreening` и
  `ValidateScreeningReport` сейчас закреплены за Gemini 3.1 и прекращают run
  после первой ошибки. Они не реализуют этот двухмодельный контракт. Старые
  runner/marker/report/reconciliation пути не переиспользуются как разрешение.
- Gemini 2.5 Flash / Google EU в свежем публичном snapshot отсутствует. Он
  остаётся исторической baseline, без новой попытки или смены route. Если он
  вернётся, добавление третьей пары потребует amendment, hashes и нового cap.

## Предметная модель и инварианты

### Exact policies и preflight

| ID | Request alias → разрешённый canonical | Endpoint / display provider | Ceilings input/output micros за 1M | Max provider micros/document |
| --- | --- | --- | --- | --- |
| A | google/gemini-3.1-flash-lite → google/gemini-3.1-flash-lite-20260507 | google-vertex/eu / Google | 275000 / 1650000 | 53900 |
| B | inception/mercury-3 → inception/mercury-3-20261019 | inception / Inception | 100000 / 350000 | 15600 |

До первого POST построить все6plans, проверить shipped fixture hashes,
helper/profile, caps, весь общий бюджет и свежие endpoint/catalog/ZDR metadata
для обеих пар. Proof alias→canonical однозначен, <=15min, не из будущего,
со snapshot digest; перед каждым POST proof/metadata должны оставаться свежими.
Публичные ссылки и hashes сохраняются в report. Устаревшее/отсутствующее/
двойное соответствие, изменение canonical/route/цены сверх потолков/параметров
останавливает кампанию до соответствующего POST, без замены пары.

Requests: exact model, provider.only ровно одна пара, allow_fallbacks=false,
require_parameters=true, zdr=true, data_collection=deny, reasoning disabled,
strict response_format schema; tools/images/raw PDF/network plugins отсутствуют.
Ответ сверяется с exact request alias либо единственным доказанным canonical,
selected route/display, одной фактической попыткой, прежними usage/cost/finish
и envelope checks. Metadata совместимость B не заменяет fake wire proof.
Новый Inception profile допускается исключительно для публичного synthetic
comparison, не объявляется EU и не расширяет политику личных данных.

Production `Policy.Valid`, HTTP/consent, product reasoner и activation validators
обязаны отвергать comparison policies/reports, включая A, совпадающий по модели
со старым screening. Новый mode не подключается к финансовому API.

### Единый corpus и профиль

| Case | SHA256 PDF | Checks |
| --- | --- | --- |
| columns | 097c3d35d4dd9787457b13f61bd524ff83c18d969ce50b93f7c8cecfeb569c8c | 19/19 |
| cards | d4dd02153724c1599d4d89eb073e8cfaaf3fbce1696a4d691b69d150d0ccdce6 | 20/20 |
| turnover-roles-ru | 648ac4477b9a78a7a1bb43eb9ae918fd5b6a2eb338e2ee07038146141781686f | 23/23 |

Все PDF — существующие `internal/pdfextract/testdata/*.pdf`, hashes сверены
локально 2026-10-21. Независимые golden и scorer assertions остаются прежними.
На обеих парах неизменны actual helper, layout, whole-document planner, prompt,
ProposalSchema hash, redaction, aliases и новые S1.10 guards. Различаться могут
только exact policy/identity/цена. Report фиксирует profile и document/plan/
request hashes каждой пары; request-hash различается из-за identity.

Scorer=`statement-screening.v2.grounding19-20-23`, core scorer=
`statement-fixed-corpus.v3.grounding`; старые reports не принимаются. Новый report
`statement.comparison.v1.grounding` содержит независимый список обеих пар,
6 запланированных case-policy tuples, attempted/completed/not_sent и причины.
У каждой пары PASS требует ровно3 полных успешных результата19/20/23, zero
blockers/issues и проверенные identity/known billing. Пропуск кейса не PASS.

### Кампания, остановки и расходы

Предлагаемый единственный порядок: A-columns → B-columns → A-cards → B-cards →
A-turnover-roles-ru → B-turnover-roles-ru. По одному наблюдению на tuple,
максимум6POST, 120s/call и 900s/campaign; прежние bounds100000input/16000output,
90000bytes/2000spans на документ. Нет retries даже при transient error.

Отличие от старого fail-fast screening явно входит в предлагаемое решение:
quality/grounding FAIL при проверенном route, валидном protocol и known billing
сохраняется, но не останавливает оставшиеся заранее разрешённые tuples.
Так обе пары получают одинаковые документы; неудача не исключается из стоимости.
Transport/protocol/identity/privacy/cost failure, timeout/неясность отправки,
settlement failure, unknown billing или общий budget block немедленно
останавливают ВСЮ кампанию. Остальные tuples=not_sent, сравнение неполно;
другая модель не используется как fallback. Для content/parse отказа с
неустановленной категорией применять строгую остановку, не quality-only bypass.

Whole-campaign reservation до POST: A3×53900 + B3×15600 = **208500 USD micros
($0.2085 provider cap)**; A subcap161700, B subcap46800. Никаких отдельных
бюджетов кандидатов в обход shared ledger. Прежний $10 ceiling с allowance/fees
и историческим расходом сохраняется; фактический доступный остаток здесь
UNVERIFIED, authoritative reserve решает достаточность. Fees не списываются
догадкой и не увеличиваются. Если billing known, учитываются все attempts,
включая content FAIL; exact amount через существующий settlement. До POST
определённый zero-call отказ может settle0; uncertain никогда не равен0.
Unknown блокирует будущие paid calls до отдельной read-only reconciliation
и разрешённого settlement, без автоматического generation GET или сброса ledger.

Новый authorization ID `s111-comparison-20261021`, exclusive production lock,
private report/marker с защитой symlink/orphan/duplicate, marker до первой
возможной отправки. Перезапуск workflow/смена SHA не возвращают разрешение;
неиспользованные tuples не переносятся. Source CI проверяет точный implementation
SHA и весь состав обязательных jobs до оплаты; bundle hashes проверяются на VPS.
Mode default=false, взаимоисключён с deploy/inspect/reconcile/старым screening;
сохранены serialized workflow concurrency и application image/health до/после.
До C6 существующие workflow inputs не dispatch: они расходуют старые разрешения.

### Метрики и решение

Private report хранит точные per-tuple scores/closed causes, row/role/money/date/
description/coverage результаты, latencies, known usage/cost и provenance.
Публичный CLI выводит только закрытые codes/bools/counts; API keys, raw provider
text, generation IDs, owner/ledger IDs и billing amounts не попадают в CI logs.
Сводную таблицу стоимости для владельца готовить отдельно из private evidence.

На пару: completed/3, successful/3, false acceptance count, полный breakdown
critical errors; total known provider cost всех её attempts / число полностью
успешных документов. При denominator0 значение undefined, не0. Если campaign
incomplete или есть unknown cost, итоговая эффективность UNVERIFIED; можно
показать только наблюдаемые known partial amounts с соответствующей пометкой.
Общие fees отражать отдельно; per-model allocation не изобретать. Давать также
per-case latency всех отправленных запросов и время всей кампании. При n=3
не публиковать p95 или вывод о статистической устойчивости; без новых повторов.

Одна пара с3/3 и нулевым false acceptance может быть рекомендована только для
расширенного corpus. Если обе3/3, сначала сравнить стоимость успешного документа,
затем наблюдаемые per-case latency; иначе не ранжировать неполные модели как
победителя. При обеих FAIL сохранить результат и предложить отдельное решение.
Ни один результат не включает automatic activation или квалификацию всех банков.

## API и OpenAPI, UI

Endpoints, DTO, generated TypeScript, React и финансовые правила не меняются.
Только operator CLI/закрытый report. Если понадобится публичный контракт,
остановить расширение и вернуть draft на review.

## Данные, миграция и rollback

SQL schema/migrations и пользовательские данные не меняются; shared usage ledger
используется без обхода reserve/settle. Reports приватны на VPS с прежними
permissions; исторические artifacts сохраняются. Read-only inspector выбирает
ровно новый report и точную campaign reservation и не пишет DB/файлы при чтении.
Rollback до live — revert operator changes; после live сохраняются marker,
report и фактический расход. Rollback не отменяет uncertain и не возвращает C6.

## Ошибки и негативные сценарии

Проверить alias/canonical ambiguity, другой endpoint/display, stale/future proof,
нет ZDR/structured outputs/reasoning=none, дорогую цену, неверный PDF/hash/order,
legacy report/current version mismatch, candidate product leakage, duplicate/
symlink/orphan marker, insufficient shared reserve, known contentFAIL continuation,
unknown/protocolFAIL global stop, timeout после POST, settlement fail, sanitized
summary sentinel, workflow mutual exclusion, no financial writes/image changes.
Неполный результат не скрывать и не исправлять дополнительным вызовом.

## Критерии приёмки и исполняемые проверки

| ID | Проверяемое evidence |
| --- | --- |
| C1 | Policy/identity/wire unit tests обеих пар; неверные routes/canonical/params/usage отвергаются; product/activation validators не принимают comparison |
| C2 | Pinned helper + delivered-tuples fake: обе пары19/20/23 с одинаковым profile; wrong record/role negatives FAIL без ослабления scorer; старые grounding/retained SQL regressions PASS |
| C3 | Fake6tuple campaign: фиксированный порядок, <=6POST, одна whole reserve/settle, cost subcaps; quality-only known FAIL продолжает, protocol/unknown останавливает весь run; n0 и incomplete metrics честные |
| C4 | One-shot/workflow/inspector mocks: replay0POST, exact-source CI и hashes, default false/exclusive jobs, read-only reservation selection, privacy sentinel, image/health preserved; старые S1.4–S1.9 unchanged |
| C5 | make test; make test-production; scripts/test-bank-statements.sh; go vet ./cmd/... ./internal/... ./scripts/research/s19-replay; git diff --check PASS, external inference=0 до C6 |
| C6 | После отдельного разрешения один live campaign <=6POST/208500micros; обе пары3/3 либо явный FAIL/incomplete; known/uncertain accounting и image/health evidence, без retries/adoption |

Точные команды новых CLI/tests и source SHA фиксируются в implementation evidence
до live. Нынешние `screening`/S1.8 commands не являются реализацией C6.

## Решение владельца и отдельный live gate

Владелец принял **offline C1–C5**, включая synthetic-only Inception
profile и новый comparison report. Принятие фиксируется до реализации в main. После реализации, CI и предъявления exact SHA
отдельно разрешить или отклонить **C6: одна кампания, до6 вызовов/$0.2085 provider**.
Это небольшое предварительное сравнение, а не расширенная qualification.
Offline C1–C5 реализованы и проверены; C6 остаётся NOT_AUTHORIZED / NOT_RUN.
Runtime production не менялся. [PR №84 post-merge](0055-pr84-postmerge.md): exact-head/main CI5/5, local и
Delivery4/4 PASS. [Конкретный C6 decision](0055-s111-c6-run-decision.md) остаётся
draft: одна кампания требует отдельного owner approval на exact source.
