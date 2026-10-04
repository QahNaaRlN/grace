# SPEC-0008: production read-only router с Jev

- Статус: accepted
- Автор: Codex
- Дата: 2026-10-05
- Acceptance owner: owner
- Acceptance record: owner явно поручил приступить к реализации 2026-10-05
- Specification branch: `codex/spec-0008-read-only-router`
- Implementation branch: отсутствует
- Implementation commit: отсутствует

## Проблема и пользовательский результат

Grace имеет шесть детерминированных read-only assistant tools, а SPEC-0006
подтвердила применимость Jev к русскоязычной маршрутизации. Однако production
API пока не принимает естественно-языковой запрос, provider не подключён к
runtime, а confidence и fallback не являются исполняемой политикой.

Нельзя сразу строить свободный chat/tool loop. Текущий tool contract не умеет
безопасно извлекать произвольные даты, лимиты и income category id из текста.
Маршрутизатор обязан различать понимание намерения и готовность аргументов:
верно выбранный tool нельзя вызывать с guessed/default arguments, если это
изменит смысл запроса.

Пользовательский результат первого production-среза — single-turn read-only
endpoint. Для безопасных запросов он возвращает подтверждённые данные Grace;
для временных/неполных/низкоуверенных запросов — структурированное уточнение;
для неподдерживаемых запросов — явный отказ. Grace остаётся полностью
работоспособным при выключенном Jev или ошибке provider.

## Scope

- добавить server-side Jev router внутри business capability `assistant`;
- добавить production TypeSafe HTTP adapter для `POST /v1/systemone`;
- использовать model id `jev-1.13.0`, который прошёл SPEC-0006 benchmark;
- отправлять provider только utterance, language и явный reference time;
- задать два независимых typed `Choice` в одном request:
  - один из шести tools, `clarify` или `unsupported`;
  - способ получения аргументов без генерации значений;
- применять confidence gate и compatibility matrix в Go до любого tool call;
- автоматически исполнять только принятые безопасные комбинации;
- возвращать structured `completed`, `clarify`, `unsupported` или
  `unavailable`, без model-generated текста;
- добавить opt-in configuration, timeout, bounded retry и safe provider errors;
- добавить `POST /api/v1/assistant/query`, OpenAPI contract и generated
  TypeScript types;
- не менять React UI в этом work item;
- создать question-set/dataset version `v2` с шестым tool и argument modes;
- выполнить locked live benchmark до статуса `verified`;
- добавить unit, fake-provider, handler и PostgreSQL integration tests;
- доказать направленность зависимостей и отсутствие writes.

## Non-goals

- conversation/thread storage, multi-turn memory и streaming;
- natural-language/generative explanation результата;
- React chat UI;
- произвольное извлечение дат, сумм, UUID или receipt limits;
- fuzzy matching income category по имени;
- выполнение `get_period_summary` или `get_cash_position` в этом срезе;
- выполнение `get_period_since_income` для названной категории;
- следующий/ожидаемый доход и forecast;
- autonomous tool loop или несколько tool calls на один запрос;
- write tools, confirmation policy и AI-originated writes;
- категоризация операций, OCR или ReceiptDraft;
- передача financial records, tool results или PostgreSQL rows provider;
- прямой доступ provider к `assistant.Executor` или базе;
- отдельный AI-service, queue, vector database или prompt storage;
- авторизация и multi-user isolation, которых пока нет в Grace;
- model aliases и автоматическое обновление модели;
- использование SPEC-0006 evaluator package из production runtime.

## Текущее поведение и доказательства

- SPEC-0005 интегрировала пять read-only tools с typed executor;
- SPEC-0007 добавила шестой tool `get_period_since_income`;
- все финансовые вычисления принадлежат детерминированным Go/PostgreSQL
  модулям;
- SPEC-0006 live benchmark на `jev-1.13.0` дал RU tool accuracy `0.9048`;
- при route confidence `>= 0.70` RU coverage составила `0.7619`, ошибок — 0;
- baseline мал и синтетичен, поэтому не является достаточным production gate;
- period-shape accuracy `0.6905` недостаточна для direct argument execution;
- существующий evaluation adapter development-only и не входит в dependency
  graph `cmd/api`;
- API key сейчас используется только локальным evaluator;
- публичного assistant endpoint и React assistant UI нет.

## Предметная модель и инварианты

`AssistantQuery`:

- `utterance` — обязательная строка после trim, 1..2000 UTF-8 bytes;
- `language` — `ru` или `en`; первый production corpus оптимизирован для `ru`;
- `referenceTime` — обязательный RFC 3339 timestamp с явным UTC offset.

`RouteChoice` — закрытый enum:

- `get_accounts`;
- `get_period_summary`;
- `get_cash_position`;
- `get_obligations`;
- `get_receipts`;
- `get_period_since_income`;
- `clarify`;
- `unsupported`.

`ArgumentMode` — закрытый enum:

- `none` — выбранный tool требует `{}`;
- `reference_time` — достаточно детерминированно использовать
  `referenceTime`;
- `defaults` — optional arguments можно безопасно опустить;
- `needs_resolution` — пользователь указал или подразумевает значения,
  которые этот router не умеет преобразовать в typed arguments;
- `missing` — обязательная информация отсутствует;
- `not_applicable` — route не исполняется как tool.

`RoutingJudgment` хранит внутри request scope:

- requested/actual model id;
- route и полный probability distribution;
- route confidence;
- argument mode и полный probability distribution;
- argument confidence;
- effective confidence;
- token usage и latency;
- provider/composition error class без credential или raw utterance.

`AssistantQueryResult`:

- `status`: `completed`, `clarify`, `unsupported` или `unavailable`;
- `route`: optional выбранный tool;
- `confidence`: optional effective confidence;
- `clarificationCode`: optional стабильный code;
- `requiredArguments[]`: имена полей, которые нельзя угадывать;
- `data`: только typed результат `assistant.Executor` при `completed`;
- `errorCode`: только безопасный публичный code при `unavailable`.

Инварианты:

1. Provider не получает accounts, events, categories, receipts, obligations,
   balances, money, tool results или другие строки PostgreSQL.
2. API key существует только на сервере и никогда не сериализуется/логируется.
3. В provider request нет chat history: только текущий utterance, language и
   reference time.
4. Question ids не несут смысл; полные правила находятся в instructions и
   criteria.
5. Вопросы route и argument mode независимы и не читают ответы друг друга.
6. Composition использует argument answer только для выбранной tool-ветви.
7. Effective confidence — минимум confidence всех реально используемых
   judgments, а не произведение и не среднее.
8. Никакой confidence не разрешает write или неизвестный tool.
9. До executor допускаются только tool name и arguments из закрытой
   compatibility matrix.
10. Router никогда не создаёт дату, UUID, limit или финансовое значение,
    отсутствующее в deterministic policy.
11. Один query вызывает не более одного read-only tool.
12. Provider error не вызывает executor и не возвращает partial financial
    data.
13. Executor error не превращается в model-generated ответ.
14. Raw utterance, request/response body и probabilities по умолчанию не
    пишутся в application logs.
15. Финансовые модули не импортируют router/provider; направление остаётся
    `assistant → financial core`.
16. Отключение router, отсутствие key или outage provider не нарушает другие
    Grace API.

## Jev question set v2

State:

```json
{
  "utterance": "Покажи мои счета",
  "language": "ru",
  "reference_time": "2026-10-05T12:00:00+05:00"
}
```

Route — `Choice` по восьми outcomes. Criteria содержат полные определения
шести tools, отдельные `clarify` и `unsupported`, запрет writes и запрет
выдумывать аргументы.

Argument mode — независимый `Choice`. Он отвечает не «достаточно ли запрос
понятен вообще», а «каким принятым способом код может получить arguments для
tool, соответствующего смыслу utterance». Speculative ответ не используется,
если route равен `clarify`/`unsupported`.

Question set имеет version `v2`. Любая смысловая правка instructions/criteria
требует новой версии и повторного benchmark.

Документация TypeSafe, использованная для design:

- [function calling](https://docs.typesafe.ai/cookbooks/function_calling);
- [confidence-gated routing](https://docs.typesafe.ai/patterns/confidence-routing);
- [intent routing](https://docs.typesafe.ai/patterns/intent-routing);
- [state](https://docs.typesafe.ai/concepts/state);
- [HTTP API](https://docs.typesafe.ai/api).

## Confidence и compatibility policy

Первый candidate threshold: `0.70`. Он выбран консервативнее минимального
безошибочного threshold `0.50` из малого SPEC-0006 baseline. Значение становится
production policy только после принятого v2 live gate.

Compatibility matrix:

| Route | Допустимый argument mode | Deterministic arguments | Действие |
| --- | --- | --- | --- |
| `get_accounts` | `none` | `{}` | execute |
| `get_obligations` | `reference_time` | `asOf = local date(referenceTime)` | execute |
| `get_receipts` | `defaults` | `{}` | execute |
| `get_period_since_income` | `reference_time` | `asOf = referenceTime`, category omitted | execute |
| `get_period_summary` | любой | отсутствуют | clarify |
| `get_cash_position` | любой | отсутствуют | clarify |
| любой tool | `needs_resolution` или `missing` | отсутствуют | clarify |
| `clarify` | `not_applicable` | отсутствуют | clarify |
| `unsupported` | `not_applicable` | отсутствуют | unsupported |
| несовместимая комбинация | любой | отсутствуют | clarify + composition code |

Дополнительные правила:

- если effective confidence `< 0.70`, результат всегда `clarify` с code
  `low_confidence`;
- named income category («после аванса») имеет `needs_resolution` и не
  исполняется с omitted category;
- explicit obligation date или receipt limit имеет `needs_resolution`, чтобы
  default не подменил пользовательский смысл;
- `anchor_not_found` из SPEC-0007 возвращает `clarify/no_income_anchor`;
- `invalid_anchor_category` недостижим для auto-execution, потому что category
  не передаётся;
- `unsupported` с confidence ниже threshold нормализуется в `clarify`, а не в
  уверенный отказ.

## API и OpenAPI

Добавляется:

```text
POST /api/v1/assistant/query
```

Request:

```json
{
  "utterance": "Покажи мои счета",
  "language": "ru",
  "referenceTime": "2026-10-05T12:00:00+05:00"
}
```

Пример completed response:

```json
{
  "status": "completed",
  "route": "get_accounts",
  "confidence": 0.93,
  "data": {
    "name": "get_accounts",
    "data": { "items": [] }
  }
}
```

Пример clarify response:

```json
{
  "status": "clarify",
  "route": "get_period_summary",
  "confidence": 0.82,
  "clarificationCode": "arguments_required",
  "requiredArguments": ["from", "to"]
}
```

HTTP policy:

- `200` — `completed`, `clarify` или `unsupported`;
- `422` — malformed/empty/oversized query, invalid language/referenceTime;
- `503` — disabled router, missing credentials, provider failure или financial
  dependency failure; body содержит безопасный `errorCode`;
- client cancellation прекращает provider/dependency context и не создаёт
  второй ответ.

OpenAPI описывает discriminated response shapes и конкретные read-only tool
result schemas. Web TypeScript types регенерируются; ручные DTO запрещены.

## Provider, configuration и отказоустойчивость

Production endpoint TypeSafe фиксирован:
`https://api.typesafe.ai/v1/systemone`. Test constructor может принимать fake
endpoint/client; production env не может подменить host.

Configuration:

- `GRACE_ASSISTANT_ROUTING_ENABLED=true` — явный opt-in; default `false`;
- `TYPESAFE_API_KEY` — server-only secret;
- model — compile-time/config-owned exact `jev-1.13.0`, aliases запрещены.

Provider policy:

- общий deadline candidate: 5 секунд;
- максимум 2 attempts только для `429`/`529`;
- `Retry-After` учитывается, но не выходит за общий deadline;
- `401`, `422`, invalid/trailing/oversized JSON и model mismatch не retry;
- фактический response model обязан быть `jev-1.13.0`;
- response обязан содержать ровно два ожидаемых Choice answers, полный
  probability distribution и значения confidence `0..1`;
- transport/timeout/auth/rate-limit/overload/protocol имеют разные внутренние
  codes, но не раскрывают provider body пользователю;
- handler не логирует Authorization, key, utterance или raw provider body.

Актуальная TypeSafe API документация требует Bearer authorization, возвращает
structured answers/usage и рекомендует exponential backoff для `429`/`529`.

## Privacy gate

При включённом endpoint полный текущий utterance и reference metadata уходят в
TypeSafe. Финансовые записи не отправляются, но сам текст пользователя может
содержать чувствительную финансовую информацию.

Принятие SPEC-0008 означает явное согласие owner на этот минимальный
production data flow при условиях:

- feature default-off;
- key и включение контролирует owner/operator;
- нет raw request/response logging;
- нет сохранения utterance в Grace этим work item;
- нет утверждения о zero-data-retention без отдельного договорного основания;
- перед распространением Grace другим пользователям требуется отдельный
  privacy/legal review и пользовательское disclosure/consent.

## Dataset v2 и live gate

До live run фиксируются question set `v2`, evaluator composition и новый
synthetic dataset. После просмотра live результата cases и expected outcomes
не изменяются в рамках того же baseline.

Dataset minimum:

- не менее 64 RU cases: минимум 8 на каждый из шести tools, `clarify` и
  `unsupported`;
- не менее 8 EN control cases: по одному на каждый route outcome;
- отдельные tags для default-safe, explicit modifier, missing argument,
  named income category, unsupported write и ambiguous route;
- минимум 20 RU cases, которые могли бы привести к auto-execution по matrix;
- реальные пользовательские данные запрещены.

Go/no-go для threshold `0.70`:

1. Provider и composition errors: 0.
2. Wrong auto-executions при effective confidence `>= 0.70`: 0.
3. RU safe-decision coverage при `>= 0.70`: не ниже `0.65`.
4. Все explicit modifier/named category cases либо корректно получают
   `needs_resolution`, либо уходят в `clarify`; silent default запрещён.
5. Все write cases получают `unsupported` или low-confidence `clarify` и
   никогда не tool execution.
6. Каждый high-confidence mismatch разобран в evidence.

Если любой gate не выполнен, status остаётся `implementing`, endpoint остаётся
default-off и изменение не интегрируется в `main` как production routing.

## Данные, миграция и rollback

PostgreSQL schema и migrations не меняются. Query state не сохраняется.

Rollback удаляет endpoint/router/provider wiring и возвращает OpenAPI/client
types. Финансовые данные, tools и evaluator artifacts не меняются.

## UI

React behavior не меняется. Generated TypeScript contract обновляется только
из OpenAPI. Chat UI и тексты уточнений будут отдельной спецификацией.

## Ошибки и негативные сценарии

- router disabled/key missing: `503 assistant_unavailable`, Grace core жив;
- blank/oversized utterance: `422 validation_error` без provider call;
- unknown language/referenceTime без offset: `422`;
- provider timeout/cancel: executor не вызывается;
- `401`: typed internal authentication error без provider body/key;
- `429`/`529`: bounded retry, затем `503`;
- invalid JSON/unknown option/model mismatch: protocol error и `503`;
- low confidence: `200 clarify/low_confidence`;
- route/argument incompatibility: `200 clarify/composition_mismatch`;
- period/cash route: `200 clarify/arguments_required`;
- named income category: `200 clarify/income_category_required`;
- missing income anchor: `200 clarify/no_income_anchor`;
- unsupported write/advice/OCR: `200 unsupported` при достаточном confidence;
- financial dependency error: `503 dependency_unavailable`, без partial data;
- API response не содержит probabilities, token usage, provider error text или
  raw utterance.

## Критерии приёмки

- [ ] Question set v2 имеет шесть tools, `clarify`, `unsupported` и шесть
      argument modes с полными criteria.
- [ ] Provider state содержит только utterance/language/reference time.
- [ ] Exact model id и строгая response validation исключают model drift и
      неизвестные ответы.
- [ ] Confidence `0.70` и compatibility matrix применяются в Go до executor.
- [ ] Auto-execution возможна только для четырёх принятых комбинаций.
- [ ] Period/cash/named-category/explicit-modifier запросы не получают guessed
      arguments.
- [ ] Low confidence, incompatible answers и missing arguments дают typed
      `clarify`, а не tool call.
- [ ] Provider failures/cancellation не вызывают executor и не возвращают
      partial financial data.
- [ ] Endpoint default-off; отсутствие key не ломает startup и остальные API.
- [ ] API key, raw utterance и provider bodies отсутствуют в logs/responses.
- [ ] Public OpenAPI и generated TypeScript точно описывают request и все
      response variants.
- [ ] React behavior, migrations и финансовые модули не изменены.
- [ ] Dataset v2 закрывает согласованный минимум и проходит offline validation.
- [ ] Locked live benchmark `jev-1.13.0` выполняет все go/no-go gates.
- [ ] Fake provider покрывает success, low-confidence, protocol, auth, rate
      limit, overload, timeout и cancellation.
- [ ] PostgreSQL integration вызывает auto-executable routes через настоящий
      Executor и доказывает отсутствие writes.
- [ ] Dependency checks запрещают provider imports финансовыми модулями и
      production import `internal/jeveval`.
- [ ] `make test`, `make test-integration`, `go vet ./cmd/... ./internal/...`
      и `git diff --check` проходят до и после merge.

## Исполняемые проверки

```bash
make test
make test-integration
./scripts/test-assistant-router.sh
go test ./...
go vet ./cmd/... ./internal/...
git diff --check
```

Live gate:

```bash
TYPESAFE_API_KEY=<redacted> \
go run ./cmd/jev-eval run \
  --dataset testdata/jev-evaluation/v2/cases.json \
  --model jev-1.13.0 \
  --report <path-outside-repository>
```

## Принятые решения owner

Owner поручил приступить к реализации 2026-10-05 и тем самым явно принял весь
контракт этой спецификации, включая:

1. threshold `0.70` и RU coverage floor `0.65`;
2. четыре auto-executable комбинации и обязательный clarify для
   period/cash/named-category/explicit-modifier cases;
3. публичный single-turn endpoint сейчас и отдельный будущий этап для React UI;
4. 5-second total deadline и максимум два attempts для `429`/`529`;
5. exact production model `jev-1.13.0` без aliases и env override;
6. minimum 64 RU + 8 EN synthetic cases для первого low-risk read-only gate;
7. minimal production data flow в TypeSafe при default-off configuration,
   отсутствии raw logging и отдельном privacy/legal gate до расширения rollout
   за пределы owner;
8. раскрытие выбранных `route` и `confidence` в public response для
   наблюдаемости первого production-среза.
