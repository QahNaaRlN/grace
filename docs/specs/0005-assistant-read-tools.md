# SPEC-0005: read-only инструменты ассистента

- Статус: integrated
- Автор: Codex
- Дата: 2026-10-04
- Acceptance owner: owner
- Acceptance record: явное согласие owner в Codex-чате, 2026-10-04
- Implementation branch: codex/assistant-read-tools
- Implementation commit: `f7fe016`
- Integration merge: `7cafa63`

## Проблема и пользовательский результат

Финансовое ядро Grace уже хранит счета, финансовые события, чеки,
обязательства и вычисляет period summary/cash position. Однако будущий
ассистент пока не имеет ограниченного машинно-читаемого контракта, через
который он мог бы запросить эти подтверждённые данные.

Нельзя решать это прямым доступом модели к PostgreSQL, передачей всей истории в
prompt или повторением финансовых расчётов в assistant-модуле. Grace должен
предоставить небольшой реестр read-only инструментов с JSON Schema аргументов,
строгой валидацией и типизированными результатами финансового ядра.

Пользовательский результат этого этапа — проверяемый фундамент будущего
диалога: ассистент сможет получать достоверные структурированные данные, но
пока не будет подключён к LLM и не появится в UI.

## Scope

- добавить `internal/assistant` как потребителя существующих финансовых
  модулей;
- ввести реестр инструментов с уникальным стабильным name, description и JSON
  Schema входных аргументов;
- реализовать единый typed dispatch по имени инструмента;
- реализовать инструменты:
  - `get_accounts`;
  - `get_period_summary`;
  - `get_cash_position`;
  - `get_obligations`;
  - `get_receipts`;
- использовать существующие Go repositories/modules как источник данных и
  вычислений;
- возвращать JSON-совместимые результаты без форматирования денежных сумм и без
  natural-language объяснений;
- проверять неизвестные поля, обязательные аргументы, форматы дат/времени,
  порядок и максимальный горизонт теми же правилами, что канонические модули;
- ограничить списочные инструменты серверным maximum;
- добавить unit tests реестра, схем, dispatch и негативных сценариев;
- добавить PostgreSQL integration test, который вызывает tools через Go, без
  LLM, сети провайдера и внешних сервисов;
- доказать направленность зависимостей: `assistant → financial modules`, но не
  наоборот.

## Non-goals

- подключение OpenAI, Anthropic или другого LLM provider;
- chat endpoint, streaming, conversation/thread storage и React chat UI;
- prompts, system messages, model selection и token budgeting;
- autonomous tool loop или orchestration нескольких tool calls;
- любые изменяющие инструменты;
- confirmation policy для AI-originated writes;
- рекомендации, финансовые советы и natural-language объяснения;
- embeddings, vector database, RAG и загрузка всей истории в context;
- прямой SQL из `internal/assistant`;
- новый публичный HTTP endpoint или изменение `api/openapi.yaml`;
- MCP server или внешний plugin protocol;
- авторизация и multi-user isolation.

## Текущее поведение и доказательства

- accounts возвращает текущие вычисленные balances;
- financialevents хранит канонические FinancialEvent/Posting;
- receipts возвращает подтверждённые Receipt/ReceiptItem;
- obligations вычисляет schedule state относительно явного `asOf`;
- analytics вычисляет period summary и cash position в read-only repeatable
  read snapshot;
- `internal/assistant` отсутствует;
- production API dependency graph не содержит AI/LLM/provider модулей;
- публичного assistant/chat API и provider credentials нет.

## Предметная модель и инварианты

`ToolDefinition`:

- `name` — стабильный machine identifier;
- `description` — краткое описание для будущего orchestrator/provider adapter;
- `inputSchema` — JSON Schema object с `additionalProperties = false`;
- `readOnly = true` — обязательный признак этого контракта.

`ToolCall`:

- `name`;
- `arguments` — JSON object, никогда не произвольная строка после dispatch.

`ToolResult`:

- `name`;
- `data` — типизированный результат канонического финансового модуля,
  сериализуемый в JSON;
- результат не содержит model-generated текста.

Инварианты:

1. Реестр содержит ровно пять принятых имён без alias и fuzzy matching.
2. Имена уникальны; порядок списка определений стабилен и тестируется.
3. Все схемы имеют root `type = object`, явно перечисляют required fields и
   запрещают unknown properties.
4. `get_accounts` не принимает аргументов.
5. `get_period_summary` принимает обязательные `from/to` RFC 3339 с явным
   offset и сохраняет полуоткрытую семантику `[from, to)`/лимит 5 лет.
6. `get_cash_position` принимает обязательный `through` и необязательный
   `asOf`; даты и лимит 5 лет совпадают с analytics API.
7. `get_obligations` принимает необязательный `asOf`; default — текущая UTC
   дата. Maximum результата — 100 obligations.
8. `get_receipts` принимает необязательный `limit` от 1 до 100; default 100.
9. Деньги остаются integer minor units с явной валютой. Tool не форматирует и
   не пересчитывает суммы.
10. Assistant не пересчитывает balance, cash flow, schedule state или
    commitments: он вызывает владельца соответствующего правила.
11. Ни один tool не начинает write transaction и не меняет финансовые записи.
12. Один tool call вызывает только один принятый capability и не выполняет
    скрытые дополнительные запросы по другим инструментам.
13. Ошибка валидации не вызывает финансовый dependency.
14. Context cancellation передаётся dependency без замены background context.
15. Финансовые модули, platform/database и HTTP handlers не импортируют
    `internal/assistant`.

## Контракты инструментов

### `get_accounts`

Arguments: `{}`.

Result: `{ "items": Account[] }` с тем же вычисленным balance, что canonical
accounts module.

### `get_period_summary`

Arguments:

```json
{
  "from": "2026-10-01T00:00:00+05:00",
  "to": "2026-11-01T00:00:00+05:00"
}
```

Result: canonical `analytics.PeriodSummary`.

### `get_cash_position`

Arguments:

```json
{
  "asOf": "2026-10-04",
  "through": "2026-11-01"
}
```

`asOf` можно опустить. Result: canonical `analytics.CashPosition`.

### `get_obligations`

Arguments: `{ "asOf": "2026-10-04" }`; `asOf` можно опустить.

Result: `{ "asOf": "YYYY-MM-DD", "items": Obligation[] }`.

### `get_receipts`

Arguments: `{ "limit": 25 }`; `limit` можно опустить.

Result: `{ "items": Receipt[] }`.

## Ошибки

Внутренний typed error содержит стабильный code:

- `unknown_tool` — имя не зарегистрировано;
- `invalid_arguments` — malformed JSON, не-object root, unknown/missing field,
  неверный формат/порядок/горизонт или limit;
- `dependency_failure` — канонический модуль не смог вернуть результат;
- `canceled` — context отменён.

Текст ошибки предназначен для логирования/диагностики, но будущий provider
adapter должен принимать решения по code. Ошибка dependency не превращается в
частичный успешный result.

## API и OpenAPI

Публичный HTTP API не меняется. `api/openapi.yaml` остаётся контрактом клиентов
Grace, а tool registry является внутренним Go-контрактом будущего orchestrator.

Tool definitions используют JSON Schema draft 2020-12 совместимое подмножество:
object, properties, required, additionalProperties, string format/description,
integer minimum/maximum. Схема хранится рядом с executor и покрывается golden
или structural tests; она не генерируется из OpenAPI автоматически на этом
этапе.

## Данные, миграция и rollback

Новые таблицы и миграции не нужны. Инструменты читают текущие канонические
данные через зависимости модулей.

Rollback удаляет `internal/assistant` и его тестовую композицию. Финансовые
данные и публичный API не меняются.

## UI

Изменений React нет. Без provider/orchestrator/chat endpoint инструменты не
выводятся пользователю напрямую.

## Ошибки и негативные сценарии

- неизвестное имя tool: `unknown_tool`, dependency не вызывается;
- пустые/malformed/non-object arguments: `invalid_arguments`;
- неизвестное поле или аргумент неправильного JSON-типа:
  `invalid_arguments`;
- отсутствующий required argument: `invalid_arguments`;
- RFC 3339 без offset, обратный или более пяти лет period:
  `invalid_arguments`;
- невалидные/обратные/слишком далёкие `asOf/through`:
  `invalid_arguments`;
- receipt limit вне 1..100: `invalid_arguments`;
- dependency error: `dependency_failure` без частичного data;
- canceled context: `canceled`;
- пустая база: успешные пустые collections, если canonical module возвращает
  пустой результат;
- сериализация сохраняет int64 minor units и отдельные валюты.

## Критерии приёмки

- [x] Реестр возвращает пять стабильных read-only definitions в принятом
      порядке.
- [x] Каждая definition имеет строгую JSON Schema с
      `additionalProperties=false`.
- [x] Unknown tool и все классы invalid arguments отклоняются до dependency.
- [x] `get_accounts` возвращает canonical computed balances.
- [x] `get_period_summary` сохраняет `[from,to)`, timezone и multi-currency
      семантику analytics.
- [x] `get_cash_position` возвращает canonical current liquid position и
      commitments.
- [x] `get_obligations` использует canonical schedule state для явного/default
      `asOf`.
- [x] `get_receipts` возвращает только подтверждённые receipts и соблюдает
      limit 1..100.
- [x] Tool results сохраняют integer minor units и явную валюту.
- [x] Dependency failure/cancellation имеют стабильные error codes и не
      возвращают partial data.
- [x] Ни один tool call не создаёт и не изменяет финансовые записи.
- [x] `internal/assistant` не содержит SQL и не импортируется финансовыми
      модулями.
- [x] Публичный OpenAPI и React не меняются.
- [x] Production dependency graph не содержит provider SDK/LLM client.
- [x] `make test`, предметный PostgreSQL integration test, `go vet` и
      `git diff --check` проходят в implementation worktree.
- [x] `make test`, `make test-integration` и `git diff --check` проходят после
      merge.

## Исполняемые проверки

```bash
make test
make test-integration
./scripts/test-assistant-tools.sh
go vet ./cmd/... ./internal/...
git diff --check
```

`test-assistant-tools.sh` должен создавать уникальный compose-project,
использовать автоматически назначенный PostgreSQL port, вызывать отдельный
test-only Go integration binary внутри репозитория и очищать окружение через
trap. Production binary и публичные routes от test harness не меняются.

## Принятые решения owner

1. SPEC-0005 создаёт только внутренний Go tool registry/executor; публичный
   HTTP/OpenAPI, provider, chat и UI не добавляются.
2. Первый реестр ограничен пятью инструментами: accounts, period summary, cash
   position, obligations и receipts.
3. Tool schemas хранятся рядом с Go executor как явный JSON Schema контракт, а
   не выводятся автоматически из OpenAPI.
4. Assistant вызывает существующие финансовые модули через узкие consumer-side
   interfaces и не содержит SQL/финансовой арифметики.
5. Все инструменты read-only; confirmation policy и любые writes остаются
   отдельной будущей спецификацией.
6. Integration harness остаётся test-only и не добавляет скрытый production
   endpoint для вызова tools.
