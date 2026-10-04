# SPEC-0005 — verification evidence

- Статус проверки: verified
- Implementation commit: `f7fe016`
- Ветка: `codex/assistant-read-tools`
- Дата: 2026-10-04

## Автоматические проверки

| Проверка | Результат |
| --- | --- |
| `make test` | Go tests, ESLint, неизменная OpenAPI TypeScript generation, TypeScript и Vite production build прошли |
| `make test-integration` | SPEC-0001…SPEC-0005 прошли в отдельных compose-projects |
| `./scripts/test-assistant-tools.sh` | Завершился `SPEC-0005 assistant read tools integration checks passed` |
| `go vet ./cmd/... ./internal/...` | Прошёл без замечаний |
| `git diff --check` | Прошёл без замечаний |

## Реестр и dispatch

Unit tests доказали:

- ровно пять definitions в порядке `get_accounts`, `get_period_summary`,
  `get_cash_position`, `get_obligations`, `get_receipts`;
- `readOnly=true`, root object schema, draft 2020-12 и
  `additionalProperties=false` для каждого инструмента;
- принятые required fields и receipt limit 1..100/default 100;
- строгий отказ malformed/non-object/unknown/missing/wrong arguments до вызова
  dependency;
- canonical period, cash horizon и obligation `asOf` validation;
- передачу исходного context dependency;
- стабильные `unknown_tool`, `invalid_arguments`, `dependency_failure` и
  `canceled` без partial result.

## PostgreSQL integration evidence

`scripts/test-assistant-tools.sh` создаёт уникальный compose-project,
автоматически назначает PostgreSQL host port, применяет все миграции и запускает
test-only Go binary `internal/assistanttest`.

Реальная композиция использует repositories accounts, analytics, obligations и
receipts через consumer-side interfaces. Проверены все пять calls, canonical
seed balance `12500000 RUB`, period opening/closing, liquid cash position,
пустые confirmed receipts/obligations и JSON serialization результатов.

Количество строк в accounts, financial_events, postings, receipts,
receipt_items, obligations, scheduled_payments и settlements до и после пяти
calls совпало. Tools не изменили финансовые записи.

## Границы архитектуры

- `internal/assistant` не содержит SQL;
- SQL test harness вынесен в отдельный `internal/assistanttest`;
- финансовые и platform модули не импортируют `internal/assistant`;
- production API dependency graph не содержит provider SDK/LLM client;
- `git diff --exit-code b15dd89..f7fe016 -- api/openapi.yaml web` прошёл:
  публичный API, generated TypeScript и React не изменились;
- новых миграций, public routes, LLM provider, prompt или write tools нет.

## Post-merge verification

Не выполнена: implementation commit ещё не интегрирован в `main`.
