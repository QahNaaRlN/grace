# SPEC-0007: периоды относительно подтверждённого дохода

- Статус: integrated
- Автор: Codex
- Дата: 2026-10-05
- Acceptance owner: owner
- Acceptance record: явное согласие owner в Codex-чате, 2026-10-05
- Specification branch: `codex/spec-0007-event-relative-periods`
- Implementation branch: `codex/income-relative-periods`
- Implementation commit: `a351d32`
- Integration merge: `d69c8f1`
- Verification date: 2026-10-05

## Проблема и пользовательский результат

SPEC-0006 показала, что запросы «куда ушли деньги после аванса» и «сравни
поступления и покупки после последнего аванса» нельзя безопасно свести к
`get_period_summary`: инструмент требует точные `from/to`, а Grace не умеет
детерминированно находить финансовое событие, относительно которого задан
период.

Модель не должна угадывать дату аванса по тексту, вычислять финансовый период
или самостоятельно искать его в выгрузке операций. Grace уже хранит
`income_received` с точным `occurred_at` и необязательной income-категорией.
Финансовое ядро должно уметь найти последнее подтверждённое поступление по
явным структурированным критериям и построить каноническую сводку от этого
события до заданного момента.

Пользовательский результат будущего assistant-этапа: Grace сможет достоверно
ответить на запрос «что произошло с деньгами после последнего поступления» и,
если пользователь ведёт отдельную income-категорию «Аванс», — «после
последнего аванса». Этот этап создаёт детерминированную read-only capability,
но ещё не добавляет natural-language router или chat UI.

## Scope

- добавить в `internal/analytics` поиск последнего подтверждённого
  `income_received` не позднее явного `asOf`;
- разрешить необязательный точный фильтр по существующей income-категории;
- использовать стабильный порядок `occurred_at DESC, created_at DESC, id DESC`;
- возвращать найденный anchor и канонический `PeriodSummary` за полуоткрытый
  интервал `[anchor.occurredAt, asOf)`;
- включать само anchor-поступление в period summary;
- выполнять поиск anchor и построение summary в одном согласованном
  PostgreSQL snapshot;
- добавить в assistant registry один read-only tool
  `get_period_since_income` с точными структурированными аргументами;
- сохранить integer minor units, явные валюты и существующую семантику
  `PeriodSummary`;
- покрыть unit tests, PostgreSQL integration tests, dependency-direction check
  и негативные сценарии;
- подготовить capability для будущей production routing specification без
  подключения Jev или другого provider.

## Non-goals

- поиск «следующего поступления»: будущие доходы Grace пока не хранит;
- прогноз зарплаты, аванса или иной суммы/даты;
- recurring-income detection и создание ожидаемых доходов;
- fuzzy matching категории по имени, merchant или description;
- определение, является ли произвольный доход зарплатой или авансом;
- передача списка финансовых событий в Jev/LLM;
- natural-language routing, confidence threshold и provider wiring;
- chat endpoint, conversation storage, streaming и React chat UI;
- generative explanation финансовой сводки;
- write tools и AI-originated writes;
- изменение существующих финансовых событий или категорий;
- сохранённые периоды, пользовательская timezone-настройка или фоновые jobs;
- изменение публичного HTTP API/OpenAPI на этом этапе.

## Текущее поведение и доказательства

- `financial_events.kind = income_received` является подтверждённым фактом и
  имеет точный `occurred_at`;
- income может иметь `category_id`, причём repository требует категорию kind
  `income`;
- `analytics.PeriodSummary` принимает точный `[from,to)` и вычисляется в
  repeatable-read snapshot;
- `assistant.Executor` предоставляет `get_period_summary`, но caller обязан
  заранее знать обе даты;
- SPEC-0006 зафиксировала четыре низкоуверенных RU расхождения; два из них
  требуют периода после последнего аванса;
- модель данных не содержит ожидаемых или запланированных будущих доходов,
  поэтому «до следующего поступления» сейчас неразрешимо без выдумывания
  факта.

## Предметная модель и инварианты

`IncomeAnchorQuery`:

- `asOf` — обязательная исключительная RFC 3339 граница с явным UTC offset;
- `categoryId` — необязательный UUID существующей категории kind `income`.

`IncomeAnchor`:

- `eventId` — id подтверждённого `income_received`;
- `categoryId` и `categoryName` — сохранённая семантическая категория либо
  `null`;
- `occurredAt` — точный timestamp события;
- `postings[]` — credit postings с integer `amountMinor` и currency.

`IncomeRelativePeriod`:

- `anchor` — найденный `IncomeAnchor`;
- `summary` — канонический `analytics.PeriodSummary` с
  `from = anchor.occurredAt` и `to = asOf`.

Инварианты:

1. Anchor выбирается только среди сохранённых `income_received`; purchase,
   refund, transfer и debt payment не подходят.
2. Используется условие `occurred_at < asOf`, совпадающее с исключительной
   верхней границей summary.
3. При `categoryId` выбирается только событие с точным равенством id. Имя,
   регистр, подстрока, merchant и description не участвуют в matching.
4. `categoryId` обязан существовать и иметь kind `income`; неизвестная или
   expense-категория отклоняется до поиска anchor.
5. Без `categoryId` выбирается последнее подтверждённое поступление любой
   income-категории, включая событие без категории.
6. Tie-breaker при одинаковом `occurred_at`:
   `created_at DESC, id DESC`; результат стабилен.
7. Anchor включён в `[from,to)`, поэтому его credit влияет на income и
   opening/closing reconciliation канонического summary.
8. Поиск anchor и все агрегаты summary видят один PostgreSQL snapshot.
9. Результаты разделяются по валютам; anchor не приводит к FX conversion.
10. Grace не выводит из категории или текста дату будущего дохода.
11. Capability read-only и не создаёт/не меняет financial events, postings или
    categories.
12. Финансовые модули не импортируют `internal/assistant` или provider code.

## Assistant tool contract

В registry добавляется шестой инструмент после пяти инструментов SPEC-0005:

`get_period_since_income`

Arguments без category filter:

```json
{
  "asOf": "2026-10-05T12:00:00+05:00"
}
```

Arguments для категории «Аванс»:

```json
{
  "asOf": "2026-10-05T12:00:00+05:00",
  "categoryId": "00000000-0000-0000-0000-000000000000"
}
```

JSON Schema имеет root `type=object`, `additionalProperties=false`, required
`asOf`, RFC 3339 date-time и optional UUID `categoryId`.

Tool возвращает `IncomeRelativePeriod` без natural-language текста. Выбор
`categoryId` из пользовательской фразы относится к будущему router и не
реализуется в SPEC-0007.

## API и OpenAPI

Публичный `api/openapi.yaml` не меняется. Capability доступна как внутренний Go
контракт `analytics` и через read-only assistant tool. Это не скрытый HTTP
endpoint.

Будущая спецификация может переиспользовать контракт в публичном API, но должна
отдельно определить авторизацию, multi-user isolation и наблюдаемое поведение.

## Данные, миграция и rollback

Новые таблицы и миграции не требуются. Реализация читает существующие:

- `financial_events`;
- `postings`;
- `accounts`;
- `categories`.

Семантика «Аванс» задаётся существующей пользовательской income-категорией, а
не новой глобальной enum. Существующие доходы без категории остаются валидными
и участвуют только в запросе без `categoryId`.

Rollback удаляет read-only query/method/tool и тесты. Финансовые данные не
изменяются.

## UI

React не меняется. Создание и выбор income-категории уже поддерживаются
существующим financial-event UI. Отдельный интерфейс периодов относительно
дохода остаётся будущим work item.

## Ошибки и негативные сценарии

- отсутствующий, malformed или не содержащий offset `asOf`:
  `invalid_arguments` на tool boundary;
- `asOf` позже допустимого operational horizon — отдельное открытое решение;
- malformed `categoryId`: `invalid_arguments` до dependency;
- неизвестная или expense-категория: typed `invalid_anchor_category`;
- подходящее поступление отсутствует: typed `anchor_not_found`, не пустая
  успешная сводка;
- dependency/database error: `dependency_failure` без partial anchor/summary;
- cancellation: `canceled`;
- concurrent write не может попасть только в anchor либо только в summary;
- несколько доходов с одинаковым `occurred_at` разрешаются стабильным
  tie-breaker;
- запрос о следующем доходе не маппится на этот tool и должен приводить к
  `clarify`/unsupported в будущем router.

## Критерии приёмки

- [x] Последний `income_received` до `asOf` находится детерминированно.
- [x] Точная income-категория ограничивает anchor; expense/unknown category
      отклоняется.
- [x] Без категории учитываются категоризированные и некатегоризированные
      доходы.
- [x] Одинаковые timestamps разрешаются через принятый стабильный tie-breaker.
- [x] Summary использует `[anchor.occurredAt, asOf)` и включает anchor income.
- [x] Anchor и summary вычисляются в одном согласованном snapshot.
- [x] Multi-currency результаты не смешиваются и сохраняют integer minor
      units.
- [x] Отсутствие anchor возвращает typed error без guessed date или partial
      result.
- [x] Assistant registry содержит ровно шесть tools в принятом порядке, а
      новый tool имеет строгую JSON Schema.
- [x] Invalid tool arguments отклоняются до вызова analytics dependency.
- [x] Capability не создаёт и не изменяет финансовые данные.
- [x] Provider/Jev не появляется в production dependency graph.
- [x] `api/openapi.yaml`, migrations и React не изменены.
- [x] Unit, PostgreSQL integration и dependency-direction tests проходят.
- [x] `make test`, `make test-integration`, `go vet ./cmd/... ./internal/...`
      и `git diff --check` проходят в implementation worktree и после merge.

## Исполняемые проверки

```bash
make test
make test-integration
./scripts/test-income-relative-periods.sh
./scripts/test-assistant-tools.sh
go vet ./cmd/... ./internal/...
git diff --check
```

Новый integration script должен использовать уникальный compose-project,
автоматически назначенный PostgreSQL port и cleanup через trap.

## Принятые решения owner

1. Существующая income-категория является единственным детерминированным
   способом различить «Аванс», «Зарплату» и другие виды дохода.
2. Запрос без `categoryId` включает income без категории.
3. Anchor входит в summary, чтобы reconciliation показывал полученную сумму.
4. `asOf` обязателен и задаётся явным timestamp вместо серверного `now` для
   воспроизводимости и явной timezone.
5. Максимальная давность поиска anchor ограничена пятью годами относительно
   `asOf`.
6. Analytics boundary возвращает отдельный typed code
   `invalid_anchor_category`; отсутствие события имеет code
   `anchor_not_found`.
7. Стабильный assistant registry расширяется с пяти до шести tools.
8. Будущие/ожидаемые доходы требуют отдельной сущности и спецификации и не
   выводятся из подтверждённой истории.
9. Способ передачи доступных income-категорий будущему router остаётся за
   пределами SPEC-0007.
