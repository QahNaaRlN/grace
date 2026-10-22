# SPEC-0055 / S1.16: допустимые alias во всех selections запроса

- Статус: accepted; implementation_status: integrated (offline; PR98); post_merge_verification: PASS.
- Автор: Codex; дата: 2026-10-23, Asia/Yekaterinburg.
- Acceptance owner: владелец Grace; acceptance record: 2026-10-23,
  прямое «принимаю» на контракт A1–A6 в PR №97 (review head `50f57dc`).
- Основание подготовки: следующий ограниченный контракт после интеграции PR №96;
  принятие A1–A6 зафиксировано отдельно выше.
- База: main `5b6ed20e94f184bac0b8d8c930f4db76d9cb9a19`, PR №96.
- Spec branch: `codex/statement-alias-schema-spec`.
- Implementation branch: `codex/statement-alias-schema`; commit: `e44963915d1e1c976af0b5dbbe30e84415846086`.
- Implementation base: main `a1cc07218da1aa9302f600d24c135aa67e09a1d6` (PR №97).
- [Implementation evidence](0055-s116-evidence.md); A1–A6 acceptance и выполнение различаются.

## Проблема и пользовательский результат

Текущий Go canonicalizer отвергает неизвестные alias во всех typed selections.
Однако исходящий JSON Schema перечисляет допустимые alias только для денежных
полей. Остальные selection items описаны как произвольные строки. Предлагается
согласовать множество допустимых ссылок в запросе с уже существующей проверкой
принимающего Go-кода.

Результат пакета — детерминированный контракт запроса и offline evidence его
передачи через настоящий transport на локальный fake endpoint. Это проверяет,
что провайдеру передаётся ограничение, но не доказывает соблюдение схемы моделью,
качество извлечения, полноту descriptions или семантическую пригодность alias.
Эффект на live failure rate остаётся UNVERIFIED до отдельного эксперимента.

## Scope

После принятия только A1–A6 ниже: request schema в `internal/statementextract`,
версия planner/plan contract, проверка wire в `internal/assistant`, регрессии
current qualification profile и причинные offline/SQL/HTTP tests. Изменения
`internal/statementqualification` допустимы только для проверки новой identity
запроса и совместимости; новый runner/campaign/report format не вводится.

Контракт A1–A6 принят и интегрирован PR №97; offline implementation проверена
в отдельном worktree от maina1cc072, интегрирована PR №98; post-merge gates
зафиксированы [отдельно](0055-pr98-postmerge.md).
Acceptance не является выполнением A1–A6 или разрешением live experiment.

## Non-goals

Prompt/redaction/alias allocation/lexical classification/финансовые guards/scorer,
полнота description, назначение неразмеченных spans, repair/retry/fallback,
изменение моделей/endpoint/цен/бюджета, новый corpus, production policy/flag/deploy,
любой внешний provider metadata/inference, workflow dispatch, SSH/VPS, чтение
private artifacts или публикация private-derived payload. Старые разрешения
на исполнение остаются consumed; S1.16 не создаёт новое разрешение или marker.
SPEC-0056 и financial acceptance не входят.

## Текущее поведение и доказательства

Проверено по public repository source на базе выше:

- `internal/statementextract/schema.go`: `ProposalSchema` задаёт string items
  для обычных selections и currency selections. `RequestSchema` добавляет enum
  только в `$defs.moneySelection` по server-owned `ClassNumericFragment`.
- `internal/statementextract/plan.go`: один document chunk, context пуст,
  `InputBytes` считает JSON messages и actual `RequestSchema`; schema template,
  planner version, chunks и aliases входят в plan/hash.
- `internal/statementextract/aliases.go` и `canonical_diagnostic.go`: все typed
  selections проходят membership/duplicate/monetary eligibility проверки.
- `internal/assistant/statement_reasoner.go`: `RequestSchema(chunk)` передаётся
  как `LLMRequest.Schema`; проверять нужно также body фактического HTTP запроса.
- `internal/statementqualification/contract.go`: `CurrentProfile` включает
  planner version и schema hash; comparison request hash включает actual schema.

[S1.15 receipt](0055-s115-d5-execution-receipt.md) фиксирует завершение workflow
и consumed permission. Закрытый payload D5 здесь не воспроизводится и не является
публичным fixture. Основание предлагаемого ограничения — приведённый код;
содержимое приватного отчёта для разработки и тестов не требуется.
[PR №96 post-merge](0055-pr96-postmerge.md).

## Предметная модель и инварианты

### A1. Одно множество ссылок и все selection paths

Для допустимого текущего plan множество D — distinct exact IDs из фактически
переданных `Chunk.Spans` в source order. Текущий plan требует пустой Context;
S1.16 не разрешает context-only ссылки или несколько chunks. Чистая функция
`RequestSchema` может сохранить поддержку union Context+Spans для существующих
низкоуровневых callers: distinct IDs в стабильном порядке Context, затем Spans;
это не ослабляет validation допустимого plan/transport.

Множество N — только IDs с server-owned `ClassNumericFragment` среди тех же
переданных spans. В valid plan N является подмножеством D. Тип/текст placeholder
не определяет membership или числовую пригодность. Все enum детерминированы,
без дублей, без значений или canonical source IDs, отсутствующих в wire tuples.

| Selection paths | Допустимые items | Сохраняемое ограничение |
| --- | --- | --- |
| `rows[].operationDate`, `bookingDate`, `description`, `directionEvidence` | D | Порядок/роли проверяет Go |
| `rows[].accountCurrency`, `transactionCurrency` | D | `minItems:0`, `maxItems:1` |
| `header.openingBookDate`, `closingBookDate` | D | Даты проверяет Go |
| `header.currency` | D | `minItems:0`, `maxItems:1` |
| `header.labels.{openingBookBalance,closingBookBalance,availableBalanceObserved,debitTurnover,creditTurnover}` | D | Связь label с суммой проверяет Go |
| `service[].spans` | D | Service не поглощает пропуски автоматически |
| `rows[].accountAmount`, `transactionAmount`; `header.{openingBookBalance,closingBookBalance,availableBalanceObserved,debitTurnover,creditTurnover}` | N | Прежняя monetary eligibility |

`direction` остаётся enum `"" / debit / credit`, `service[].reason` — обычной
строкой; они не ссылки и не получают alias enum. Required fields и
`additionalProperties:false` сохраняются. Пустые selections допустимы.
Если D или N пусто, соответствующие arrays допускают только `[]` через
`maxItems:0`; не использовать пустой enum или фиктивный alias. Пустой document
по-прежнему не создаёт допустимый extraction plan.

### A2. Компактная схема и защита принимающей стороны

Добавить общее definition string item для D и `$ref` на него из обычных и
currency arrays; числовой enum N сохраняется отдельно. Не размножать полный D
в каждой field schema. Использовать уже применяемые `$defs/$ref`, type, enum,
array bounds; не вводить pattern-based или provider-specific расширения.
`ProposalSchema` содержит новую структуру references, `RequestSchema` заполняет
её exact membership для chunk, не изменяя глобальный template и входной chunk.
Повторные/параллельные вызовы для разных документов не смешивают enum.

Go canonicalization/merge/grounding/coverage/role guards остаются обязательными:
fake provider, игнорирующий schema, должен по-прежнему получить отказ на unknown,
duplicate и non-numeric money alias до materialization. Нельзя автоматически
исправлять ссылку, отбрасывать неверный item, дополнять selection или service.
Membership не доказывает финансовую роль или корректность значения.

### A3. Identity, consent и история

Принятая новая `PlannerVersion`: `statement-document-atom.v3.alias-enum`.
Template `ProposalSchema` меняется вместе с references, значит `SchemaHash`
меняется содержательно. PromptVersion/текст SystemPrompt, AliasVersion,
RedactionVersion и scorer version сохраняются. Actual request schema однозначно
восстанавливается из согласованных chunks и новой версии planner.

Новый plan hash и actual comparison request hash включают новый контракт.
Старые v2 previews/consents не дают разрешение нового запроса: штатный
`preview_changed` 409 до reservation/provider Complete, нужен новый preview и
явный consent. Historical completed attempts/reviews остаются читаемыми;
сохранённые результаты, snapshots, digests и billing не переписываются.

Current qualification/report gates не принимают прежний planner/schema profile
для новых calls/enable. Сохранённые отчёты не переводятся в PASS и не получают
новые hashes. Старый D5 воспроизводим только на frozen source его receipt;
S1.16 не обещает, что новый binary пересчитает прежний wire identity.
Не ослаблять validators ради прохождения legacy fixtures.

### A4. Actual bytes и transport

Лимиты сохраняются: 90000 UTF-8 messages+schema bytes, 2000 spans,
100000 input / 16000 output bound, 120 s/call, 150 s/product attempt;
budget/fees/reservation ceilings и один document request не расширяются.
`InputBytes` равен actual JSON messages bytes + actual request schema bytes;
это прежний расчёт, не размер всего HTTP envelope и не измеренные model tokens.

Границы bytes проверить точно ниже/на/выше cap, spans на/выше 2000, используя
реальный новый schema и planner. При превышении — прежний
`statement_atomic_block_limit` / `statement_document_span_limit` до reservation
и provider call; без truncation/split/retry. Fake HTTP endpoint проверяет schema
в сериализованном `response_format`, равенство с согласованным plan и membership
на каждом path, сохранение required/array bounds. Transport pinning, no-fallback,
ZDR, reasoning off и billing settlement не меняются.

## API, OpenAPI и UI

Новых endpoints/DTO/UI нет; statement plan JSON сохраняет прежние поля.
PlannerVersion и SchemaHash меняют значения без изменения типов. Если при
реализации обнаружится необходимость изменить публичную структуру, вернуть
контракт на review, обновить OpenAPI и сгенерированные TS по отдельному решению;
вручную DTO не копировать. Ручной импорт и финансовое ядро работают без LLM.

## Данные, миграция и rollback

Новых SQL миграций/колонок/финансовых writes нет. Новый preview сохраняет новую
версию в существующем JSON plan. Version/hash проверяются при load/consent/extract.
При rollback старый binary отвергает v3 preview, а новый отвергает v2; никакого
прозрачного переноса согласия между версиями. Исторические attempts доступны
без inference. Rollback deployment этим пакетом не разрешён.

## Ошибки и негативные сценарии

- Ссылка синтаксически похожа на alias, но не входит в D: не входит в schema
  enum; если fake provider всё равно её возвращает, прежний Go отказ сохраняется.
- Canonical source ID / placeholder / отсутствующий в D alias другого документа / строка с пробелами
  или иным регистром: только точное membership, без trim/case/auto-map repair.
  Alias локален документу: одинаковое написание в другом plan не имеет отдельной
  identity; связь с документом обеспечивает весь согласованный plan/consent.
- Numeric alias разрешён в D и N, но сам по себе не доказывает сумму; текстовый
  alias допустим в D и недопустим в N. Source punctuation roles не меняются.
- Currency selection длиной 2, duplicate финансовых selections, неверный порядок,
  пропуск description/coverage и неверные header roles по-прежнему FAIL.
- Старый consent/profile, изменённые chunk/schema/hash, превышение bound:
  отказ до отправки, без новых финансовых данных.

## Критерии приёмки и исполняемое evidence реализации

Контракт A1–A6 принят. Текущий результат выполнения и команды находятся в
[implementation evidence](0055-s116-evidence.md); принятие реализации владельцем
и live qualification этим документом не объявляются.

| ID | Требование | Исполняемое evidence |
| --- | --- | --- |
| A1 | Все 22 selection paths ограничены D или N; empty sets, currency max1 и non-reference strings сохранены | Table-driven тест actual `RequestSchema`, resolve `$ref` и независимая проверка positive/negative instance на каждом path; пути из явного oracle, не из обхода implementation |
| A2 | Детерминизм, template/chunk не мутируют, разные документы изолированы; Go guards не ослаблены | Повторные/параллельные schema tests и fabricated/unknown/duplicate/non-numeric proposals через настоящий canonicalizer; `go test -race ./internal/statementextract ./internal/assistant` |
| A3 | Version/template/plan/request identity меняются; old/new consent и profile несовместимы | Real plan/SQL/HTTP lifecycle: старый persisted consent →409 без reserve/Complete, новый consent→actual request; history read и rollback version cases; mutation tests request/profile hashes |
| A4 | Actual bytes/spans limits и real HTTP schema совпадают с approved plan | Exact byte/span boundary tests, local httptest wire capture через штатный provider; overrun до fake transport/reservation; public synthetic corpus size report |
| A5 | Полный public corpus проходит детерминированный путь с fake aliases, negatives остаются FAIL | Real PDF helper → planner → fake provider → canonical/merge/Go → неизменённый scorer columns/cards/RU 19/20/23; independent fixture oracle, без private payload или paid calls; financial writes0 до confirmation |
| A6 | Регрессии и изоляция существующих operational modes | `make test`, `make test-integration`, `make test-production`, `go vet ./cmd/... ./internal/...`, `git diff --check`; source/head/log references в implementation report |

Тесты не должны объявлять модель schema-compliant на основании fake endpoint.
Новый offline size report содержит только public shipped fixtures, actual
schema/messages bytes и versions/hashes, без D5-derived данных. Результаты
старых live campaigns и consumed markers не меняются.

## Открытые решения и следующие gates

1. Контракт A1–A6 принят владельцем 2026-10-23 прямым «принимаю»;
   повторное acceptance этого scope не требуется.
2. Интеграция принятой спеки выполнена PR №97. Offline implementation
   проверена в `statement-alias-schema` от maina1cc072 и интегрирована PR №98.
   [Post-merge evidence](0055-pr98-postmerge.md); новый [draft S1.17](0055-s117-alias-comparison.md)
   не является разрешением следующей реализации или запуска.
3. Нужен ли новый live эксперимент, его модель/endpoint/corpus/ceiling/exact SHA
   и разрешение на передачу данных — отдельная спецификация/решение после
   offline evidence. Никакой live permission в S1.16 не подразумевается.
4. Полнота description и семантика неразмеченных spans остаются отдельным
   кандидатом; schema membership не закрывает эти задачи.

## Историческая проверка документационного PR после принятия

`TMPDIR=/tmp VITEST_MAX_WORKERS=1 make test` — PASS в spec worktree;
log `/tmp/grace-s116-acceptance-make.log`: Go, 88 Web files/334 tests,
ESLint/FSD/Prettier/build и offline Jev. `git diff --check`, проверка
локальных ссылок и пустого runtime diff относительно main5b6ed20 — PASS.
Это историческая проверка docs-only изменения; на той точке implementation
был not_started. Текущий статус — в начале документа и implementation evidence.
Повторное принятие этого контракта не требуется.
