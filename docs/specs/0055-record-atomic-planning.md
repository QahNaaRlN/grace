# SPEC-0055 / S1: целый документ и короткие source aliases

- Дата: 2026-10-19 (Asia/Yekaterinburg).
- Статус: accepted, 2026-10-19; implementation `not_started`.
- Основание: принятая [проработка качества](../research/statement-quality/README.md), revision `c63e7cd3f7ca355ee7470036919b948d0078b73f` / PR #68; поручение продолжить движение к включению Statement AI.
- Acceptance record: владелец выбрал «Утвердить S1 с новыми лимитами (рекомендую)» в ответ на конкретный контракт / PR #69, revision `fa4617f2239bb5c4ca51b82a4a8ff463f16f2191`. Приняты 90000 bytes / 2000 spans / 100000 input / 16000 output и максимум 70000 provider micros/call; общий бюджет сохраняется, платные прогоны требуют отдельного решения.
- Product baseline: `7be572ceaac5819a2031059ef60388dcd0711614`, PR #66; research base `f3476a3`.
- Ветка: `codex/statement-record-planning`; product implementation отсутствует.

## Принятое решение

Передавать redacted текст всего допустимого документа одним запросом с короткими
локальными aliases. Целый документ — единственный atom; все source spans
доступны одновременно, поэтому операции не режутся ни между строками, ни
между страницами. Финансовый смысл и правильность ответа это не гарантирует.
Крупные документы получают явный preflight отказ и ручной review.

Это замена прежнего предложения геометрического multi-chunk planner, а не
реализация всех его обещаний. [Offline comparison](../research/statement-quality/atoms.md)
отклонил оба геометрических кандидата: разрывы core/full records сохраняются
на wide-gap и cross-page mutations. Document+aliases при старых 9000 bytes
тоже не проходит прежний S1-AC2 по доступности: дополнительные отказы нельзя
записать как успех. Только явно предложенный ниже увеличенный cap вместил все
56 public cases этой стратегии без разрывов. Весь comparison — 280 cases на
двух PDF, а не 280 банков или проверка модели.

## Область и инварианты

S1 меняет план, wire locators, request limits, проверку metadata, metering
bounds и их отображение в consent. Сохраняются существующие model/provider,
ZDR/deny, no fallback, text-only, disable reasoning, Go grounding/normalization,
полная source coverage, общая budget ledger и financial confirmation.

Ответ по-прежнему имеет существующую структуру `Proposal`; ссылки в нём
переводятся обратно в canonical source IDs до merge/validation. Новый coverage
vector S2, автоматический выбор модели SPEC-0056, новые провайдеры, OCR,
изменение финансовых правил и автоматический повтор не входят в S1.
Увеличение общего бюджета и платные campaigns этой редакцией не разрешаются.

## Точный алгоритм document-atom.v1

1. Применить существующий `Document.Validate` и validate visible IDs. Сохранить
   текущие PDF/page/extractor limits. Дополнительно `len(Spans)>2000` возвращает
   `statement_document_span_limit` (422) до consent/reservation/call.
2. Canonical IDs уникальны и уже связаны с исходным документом. Отсортировать
   копию их списка лексикографически по UTF-8 bytes. ID с индексом `i` получает
   alias `s` + lowercase base36(i), без ведущих нулей (`s0`, `s1`, … `sz`, `s10`).
   Map alias→canonical обязана быть биекцией на все source IDs; лишние,
   повторные и отсутствующие значения запрещены. Aliases не переносятся между
   документами. Alias не отражает банковский смысл.
3. Создать один ordered atom из всех canonical IDs в исходном порядке
   `Document.Spans`. Геометрия и текст не создают внутренних границ.
   Исходные bbox/page сохраняются; данные не сортируются моделью.
4. Применить действующую allowlist/visibility redaction к исходным spans.
   Затем заменить только структурный ID на alias. Созданный redactor-ом opaque
   placeholder `[text_<canonical>]` заменить на `[text_<alias>]`; это не
   string replacement во входном банковском тексте. Denied identifiers/email/
   account fragments остаются скрыты даже при explicit visibility.
5. Создать единственный `Chunk{Index:1, Context:[], Spans:allRedactedAliased}`.
   Всё evidence уже находится в Spans; повторный Context не нужен. RequestSchema
   формирует существующий monetary enum по wire aliases. Schema не получает
   незамаскированный оригинал, canonical map не передаётся провайдеру.
6. Измерить actual JSON messages + request schema прежним `InputBytes`.
   Если размер >90000 bytes, вернуть `statement_atomic_block_limit` (422).
   Не резать, не усекать, не выбирать другой маршрут. Ровно 90000 допустимы;
   более ранний отказ по span/PDF limits остаётся допустимым.
7. Завершённый plan имеет один chunk и один call. `MaximumMicros` определяется
   pinned policy и новыми bounds ниже. Не возвращать частичный пригодный plan
   при отказе. Тот же document/visible set/policy/versions даёт тот же plan hash.

## Принятые request bounds и стоимость

| Параметр | Текущий product | S1 candidate |
| --- | --- | --- |
| Messages + schema bytes / request | 9000 | 90000 |
| Input token upper bound | 10000 | 100000 |
| Output token hard cap | 2000 | 16000 |
| Calls / новый attempt | число chunks, максимум 32 | ровно 1 |
| Source spans / новый plan | общий extractor limit | максимум 2000 |
| Provider ceiling / call при прежних ценовых ceilings | 8000 micros | 70000 micros (0.07 USD) |

Расчёт candidate ceiling: ceil((300000×100000 + 2500000×16000)/1000000).
Это публичный теоретический предел provider cost, не фактический расход,
остаток владельца или гарантия latency. Существующие fees/reservation guards
учитываются общим budget ledger; общий allowance не меняется. Нет доступного
остатка/цены/metadata — отказ до вызова. Повтор требует нового явного действия,
нового актуального preview/consent; unknown billing блокирует новые calls.
Существующий lifetime guard 32 calls/document не повышается. Нельзя переиспользовать
старый smoke/campaign cap так, будто один новый call всё ещё стоит максимум 8000.

[Публичные endpoint metadata OpenRouter](https://openrouter.ai/api/v1/models/google/gemini-2.5-flash/endpoints),
проверенные 2026-10-19, для exact `google-vertex/eu` показывали context 1048576,
max completion 65535, structured_outputs/response_format/max_tokens и тарифы в
пределах прежней policy. Это подтверждает заявленную capability, а не реальную
совместимость/quality нового request. Перед consent и отправкой повторно
проверять exact route, цену, privacy и capability; контекст должен вмещать
100000 input + 16000 output. Нельзя использовать metadata другого Google tag.
Request envelope должен помещаться в оставленный input margin; иначе
preflight отказ, а не недооценённая reservation. Timeout ограничен отдельным server-owned statement profile ниже,
без скрытого retry; обрезанный/пустой JSON не считается успехом.

## Профиль транспорта и реальный HTTP путь

Одного изменения констант planner недостаточно. Проверены три действующих
ограничения: `ModelPolicy.ReservationMicros` принимает максимум 10000/2000;
provider и statement call имеют 30 s; общий API `WriteTimeout` — 15 s.
CLI smoke не проверяет последний предел, поэтому не заменяет HTTP integration.

Ввести отдельный server-owned transport profile `statement-document.v1`:
100000 input / 16000 output / 120 s provider request, text-only с прежней pinned
route. Создавать отдельный provider instance в composition root для
StatementReasoner. Default constructor/profile, `ReservationMicros` и 30 s
для объяснений/чеков сохраняются. Новый profile проверяет собственные bounds и
reservation; нельзя просто глобально разрешить всем callers 100000/16000.
Profile не выбирается из HTTP body, документа или ответа модели. Statement
reasoner требует именно этот profile и совпадение approved policy/plan limits.
HTTP client timeout и внутренний context обязаны оба допускать 120 s; меньший
скрытый timeout не оставляется. Input envelope/response-size guard проверяются
под profile, остальные transport privacy/route checks не ослабляются.

Один statement attempt ограничен 150 s с прежним revision/tombstone watcher и
отдельным bounded persist context. Только extraction HTTP route получает write
deadline 180 s через ResponseController/поддерживаемый writer; глобальные
Read/WriteTimeout API не увеличиваются. Если route не умеет установить deadline,
отказать до reservation/call, не запускать платный запрос с обречённым ответом.
Проверить middleware/proxy/client цепочку и disconnect/cancel. Это synchronous
существующий API, новые background jobs и 202/poll protocol не вводятся.

Acceptance: real HTTP server с коротким global WriteTimeout + delayed fake
provider отвечает полным result на extraction route; прочие routes сохраняют
исходный timeout. Проверка длинного ответа не должна ждать 120 s: тайминги
сокращаются в test transport/HTTP fixture. Unknown billing после timeout по-прежнему
замораживает новые calls. Production private PDF не нужен для этого теста.

## Canonicalization ответа и финансовые проверки

Перед `MergeProposals` перечислить и преобразовать каждую typed Selection:
row operation/booking dates, amounts/currencies/description/directionEvidence;
header balances/dates/currency и каждую header.labels selection; service spans.
Нельзя заменять строки в сыром JSON, direction или service reason. Каждая
ссылка обязана быть alias из approved map; canonical ID, чужой alias или
unknown alias в wire response → `provider_protocol`, без финансового draft.
Повторное использование одного известного alias в разных допустимых ролях
(например currency) сохраняет существующие правила, не считается новым ID.

После canonicalization работают существующие merge/grounding/coverage guards,
включая bound/unbound header labels fix PR #66. Не делать unused IDs service
автоматически. Недостающий span, неверные роли/значения, неполная строка,
несогласованные totals остаются видимым FAIL/manual-review outcome. Whole-document
контекст не делает ответ модели достоверным. Financial writer не расширяется.

## Persisted plan, consent, API и rollback

Обязательные поля `StatementPlan`: `plannerVersion` =
`statement-document-atom.v1`, `aliasVersion` = `statement-alias.lex-base36.v1`,
`atoms` (один список всех canonical IDs), `aliases` (alias→canonical),
`limits` (`maxInputBytes:90000`, `inputTokens:100000`, `outputTokens:16000`,
`maxSpans:2000`, `callTimeoutSeconds:120`, `attemptTimeoutSeconds:150`). Существующие documentHash/policy/chunks/MaximumMicros входят в
тот же hash; actual RequestSchema детерминированно восстанавливается по chunks.

PromptVersion меняется на `statement-span-mapping.v9.document-alias`;
RedactionVersion — `statement-allowlist.v2.alias-placeholders`. Текст prompt
явно описывает wire aliases, целый документ и отсутствие дополнительного context.
`currentPlanContract` проверяет новые версии, limits, biекция map/atom/chunk,
ровно один chunk и фактический размер; чужая/пустая версия → `preview_changed`
(409) до reservation. Открытие preview не перестраивает план молча.

Данные остаются в существующем JSON plan `statement_extraction_previews`,
отдельная SQL колонка не нужна. Актуальность source revision, RLS/owner,
idempotency и expiry сохраняются. `loadAttempt` читает исторические завершённые
attempts без нового inference; это проверяется integration test.
Новая prompt version делает plan несовместимым со старым binary при rollback,
даже если тот игнорирует незнакомые JSON keys. Проверить отказ в обе стороны;
история и ранее подтверждённый импорт при rollback сохраняются.

Публичный контракт изменяется через `api/openapi.yaml` и generated TypeScript.
Visibility input остаётся canonical; source grid не должен превращаться в alias
API. Owner preview показывает фактический redacted payload, один call и новую
предельную стоимость. Ошибки размера/количества spans объясняют доступность
ручного review. Web не выполняет финансовые расчёты или выбор маршрута.

## Acceptance matrix

Prototype evidence ниже проверяет размер/целостность, не реализацию API/SQL/LLM.
Все integration/product критерии сейчас `NOT_RUN`.

| ID | Критерий | Исполняемое evidence для реализации |
| --- | --- | --- |
| S1-AC1 | Оба baseline PDF принимаются целиком | Real pinned helper → plan; exactly one chunk; независимые полные record IDs внутри chunk |
| S1-AC2 | Все 32 исторических случая и 24 дополнительных stress случая candidate проходят без source loss/split | v7/v8 research comparison, exact-once ID membership и full-record oracle; все 56 positive structural cases accepted |
| S1-AC3 | Преобразование обратимо и безопасно | Биекция, duplicate/unknown/foreign/canonical wire IDs, alias reuse currency, header labels, service; значения/IDs исходного Document неизменны |
| S1-AC4 | Пределы не обходятся | 89999/90000/90001 actual bytes, 2000/2001 spans, cap overflow → ноль reasoner calls/reservations; input margin/schema growth/capability mismatch |
| S1-AC5 | Privacy не ухудшилась | Denied integer fragments/email/phone даже при visibility; wire не содержит canonical map или оригинальный private text; approved preview совпадает с request |
| S1-AC6 | Новый contract требует нового consent | Changed version/map/limits/source revision, stale preview, чужой owner, idempotency conflict; ноль calls на отказе; roundtrip SQL plan/hash |
| S1-AC7 | Причинный real-helper → fake-provider → SQL путь | Fake получает только реальный approved payload, возвращает aliases; Go canonicalizes, verifies rows/clocks/FX/totals/coverage; unknown refs и пропуски отдельно FAIL |
| S1-AC8 | Финансовая запись подтверждается владельцем | До confirmation ноль financial writes; после него ожидаемые postings ровно один раз; manual path без Reasoner |
| S1-AC9 | Upgrade/rollback и budget честны | Исторические attempts читаются; старый/new consent несовместимы; reserve/settle по новым bounds, exact route/capability; стандартный transport сохраняет старые limits; real HTTP deadline/cancel test; generated types, API/web build |

Fake harness проверяет один целый request вместо выдуманного single-chunk assert
для старого splitter; negative oversize case обязан отказать до fake. Старые
multi-chunk tests legacy merge сохраняются как regression checks, но новый planner
не обещает multi-chunk support. Это явное изменение ранее предложенного S1 scope.

Обязательные команды: `make test`, `make test-integration`,
`go vet ./cmd/... ./internal/...`, `git diff --check`; bank/PDF suite использует
isolated PostgreSQL и fake provider. После принятия technical revision —
implementation и независимое evidence. S2/S3/corpus qualification, отдельно
утверждённый bounded paid campaign и release/activation gates сохраняются.
Этот кандидат не включает Statement AI и не объявляет model quality PASS.
