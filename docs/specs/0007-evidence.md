# SPEC-0007 — verification evidence

- Состояние: `VERIFIED`
- Ветка: `codex/income-relative-periods`
- Implementation commit: `a351d32`
- Дата проверки: 2026-10-05

## Реализованный контракт

- `analytics.Repository.IncomeRelativePeriod` находит последний подтверждённый
  `income_received` в пяти-летнем окне до явного `asOf`;
- optional `categoryId` проверяется как существующая income-категория и
  сопоставляется только по точному UUID;
- порядок anchor стабилен:
  `occurred_at DESC, created_at DESC, id DESC`;
- anchor и канонический `PeriodSummary` за `[anchor.occurredAt, asOf)`
  вычисляются в одной read-only repeatable-read transaction;
- anchor income включён в summary, а все суммы остаются integer minor units с
  явной валютой;
- assistant registry расширен шестым read-only tool
  `get_period_since_income`;
- malformed/null/non-string category, отсутствующий `asOf`, timestamp без
  offset и unknown fields отклоняются до dependency;
- `invalid_anchor_category` и `anchor_not_found` сохраняются как стабильные
  typed error codes без partial result;
- публичный OpenAPI, migrations, React и production provider graph не
  изменены.

## PostgreSQL evidence

Команда:

```bash
./scripts/test-income-relative-periods.sh
```

Результат:

```text
ok github.com/QahNaaRlN/grace/internal/assistanttest
SPEC-0007 income-relative periods integration checks passed
```

Integration fixture доказывает:

- точный выбор income-категории «Аванс»;
- выбор более поздно созданного события при одинаковом `occurred_at`;
- исключительность верхней границы `asOf`;
- участие некатегоризированного дохода в запросе без category filter;
- включение anchor и последующих фактов в canonical summary;
- исключение будущего события за `asOf`;
- исключение anchor старше пяти лет;
- typed errors для expense/unknown category и отсутствующего anchor;
- неизменность количества финансовых строк после всех read calls.

Существующий analytics integration test дополнительно доказывает сохранение
multi-currency семантики `PeriodSummary` без скрытой конвертации.

## Автоматические проверки

| Проверка | Результат |
| --- | --- |
| `make test` | PASS: Go tests, ESLint, TypeScript/Vite build, offline Jev evaluator |
| `make test-integration` | PASS: SPEC-0001, 0002, 0003, 0004, 0005 и 0007 PostgreSQL gates |
| `./scripts/test-income-relative-periods.sh` | PASS |
| `./scripts/test-assistant-tools.sh` | PASS |
| `go vet ./cmd/... ./internal/...` | PASS |
| `git diff --check` | PASS |

Vite сохранил существующее предупреждение о chunk больше 500 kB; production
build успешен, а SPEC-0007 не меняет React bundle.

## Границы результата

SPEC-0007 не подключает Jev, не выбирает category по пользовательскому тексту,
не добавляет HTTP/chat UI и не прогнозирует следующее поступление. Запрос «до
следующего поступления» остаётся безопасным `clarify` до отдельной модели
ожидаемых доходов. Production routing требует следующей принятой спецификации.
