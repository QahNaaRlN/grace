# SPEC-0008 — implementation and live-gate evidence

- Состояние: `VERIFIED`
- Ветка: `codex/read-only-router`
- Implementation commit: `a2c0dae`
- Composition correction: `b9205f7`
- Question-set v3 implementation: `48f7e64`
- Dataset: `testdata/jev-evaluation/v2/cases.json`, version `v2`, 64 RU + 8 EN
- Question set: version `v3`
- Requested/actual model: `jev-1.13.0`
- Дата проверки: 2026-10-05

Production router реализован и прошёл все локальные, PostgreSQL и locked live
gates. Question set v3 поднял RU safe-decision coverage с `0.4844` до
`0.8594`, сохранив нулевое число wrong auto-executions и provider/composition
errors. SPEC-0008 имеет статус `verified`; endpoint по-прежнему default-off до
явного включения operator.

## Реализованный контракт

- `POST /api/v1/assistant/query` со strict request validation и typed
  `completed`, `clarify`, `unsupported`, `unavailable` responses;
- exact model `jev-1.13.0`, fixed TypeSafe endpoint, 5-second total deadline и
  не более двух attempts для `429`/`529`;
- provider state содержит только utterance, language и reference time;
- два независимых opaque-id `Choice`: route и argument mode;
- confidence threshold `0.70` и закрытая compatibility matrix применяются до
  `assistant.Executor`;
- исполняются только accounts/none, obligations/reference_time,
  receipts/defaults и period-since-income/reference_time;
- financial core остаётся детерминированным и не импортирует assistant;
- endpoint default-off, key server-only, raw provider bodies и utterance не
  логируются и не сохраняются;
- OpenAPI и generated TypeScript описывают конкретные read-only result shapes;
- React behavior и PostgreSQL schema не изменены.

## Локальные и PostgreSQL проверки

| Проверка | Результат |
| --- | --- |
| `make test` | PASS |
| `make test-integration` | PASS: SPEC-0001..0005, 0007 и новый 0008 gate |
| `./scripts/test-assistant-router.sh` | PASS |
| `go vet ./cmd/... ./internal/...` | PASS |
| `git diff --check` | PASS |

PostgreSQL test провёл все четыре auto-executable routes через настоящий
`assistant.Executor` и подтвердил неизменность количества строк accounts,
events, postings, receipts, obligations и schedules.

## Первый locked live benchmark — question set v2

Команда:

```bash
TYPESAFE_API_KEY=<redacted> \
go run ./cmd/jev-eval run \
  --dataset testdata/jev-evaluation/v2/cases.json \
  --model jev-1.13.0 \
  --report /tmp/grace-spec0008-jev-report.json
```

Raw report:

- SHA-256:
  `4c65a72edc4b4efaaa1a810fe253c212a9fba1f29c6f2da0e1203fdecccf4162`;
- mode `0600`;
- provider errors: 0;
- actual model: `jev-1.13.0`;
- secret markers: не обнаружены;
- initial evaluator result: RU exact `28/64`, coverage `0.2969`, wrong
  auto-executions 0, gate failed.

Первичный анализ обнаружил composition defect: evaluator и production router
применяли speculative argument answer к terminal routes `clarify` и
`unsupported`. Это противоречило принятому инварианту, по которому
неиспользуемая независимая ветвь не влияет на outcome. Исправление не меняет
question set, dataset, threshold или compatibility matrix.

Сохранённые ответы пересчитаны без нового provider request:

```bash
go run ./cmd/jev-eval analyze \
  --dataset testdata/jev-evaluation/v2/cases.json \
  --input-report /tmp/grace-spec0008-jev-report.json \
  --report /tmp/grace-spec0008-jev-report-reanalyzed.json
```

Reanalyzed report:

- SHA-256:
  `5890028715a813d7847fae62f929d47e91db22d647754280bbd092352d339fc8`;
- mode `0600`;
- provider/composition errors: 0;
- wrong auto-executions: 0;
- unsafe modifier executions: 0;
- unsafe write executions: 0;
- RU exact: `40/64`;
- RU eligible exact at `0.70`: `31/64`;
- RU safe-decision coverage: `0.4844`, ниже принятого floor `0.65`;
- gate: failed.

## Mismatch analysis

| Expected route | Cases | Exact | Eligible exact at 0.70 |
| --- | ---: | ---: | ---: |
| `clarify` | 8 | 8 | 8 |
| `get_accounts` | 8 | 1 | 1 |
| `get_cash_position` | 8 | 5 | 2 |
| `get_obligations` | 8 | 6 | 6 |
| `get_period_since_income` | 8 | 3 | 1 |
| `get_period_summary` | 8 | 7 | 4 |
| `get_receipts` | 8 | 2 | 1 |
| `unsupported` | 8 | 8 | 8 |

Route judgment в большинстве проблемных cases был правильным. Coverage
снижает второй независимый judgment:

- accounts часто выбирает `none` правильно, но с confidence ниже `0.70`, либо
  путает его с `reference_time`;
- generic recent receipts путаются с `none`/`needs_resolution`, хотя explicit
  limit/filter отсутствует;
- generic «после последнего дохода» ошибочно считается
  `needs_resolution`, хотя только named income category требует resolution;
- отдельные current obligations имеют правильный `reference_time`, но низкий
  confidence.

Ни одного unsafe tool call live composition не разрешила. Недостаток — низкая
полезная coverage, а не нарушение read-only safety boundary.

## Recovery gate

Dataset v2 и expected outcomes после live run остаются неизменными. Threshold,
model, compatibility matrix и privacy contract также не меняются.

Owner явно принял 2026-10-05 продолжение как question set `v3`, потому что
семантическая правка criteria требует новой версии:

1. оставить route question без изменений;
2. переписать только argument-mode instructions/criteria как приоритетный
   classifier с явными positive/negative правилами для accounts, current
   obligations, default recent receipts, latest income, named income category
   и explicit modifiers;
3. повторить locked benchmark на неизменном dataset v2;
4. сохранить threshold `0.70` и floor `0.65`;
5. не интегрировать ветку, если повторный gate не пройдёт.

## Повторный locked live benchmark — question set v3

Команда выполнена с тем же dataset v2, моделью и threshold; изменён только
принятый argument-mode question set:

```bash
TYPESAFE_API_KEY=<redacted> \
go run ./cmd/jev-eval run \
  --dataset testdata/jev-evaluation/v2/cases.json \
  --model jev-1.13.0 \
  --report /tmp/grace-spec0008-jev-v3-report.json
```

Результат:

- dataset `v2`, question set `v3`;
- requested/actual model: `jev-1.13.0`;
- EN exact `8/8`, safe-decision coverage `1.0000`;
- RU exact `60/64`;
- RU eligible exact at threshold `0.70`: `55/64`;
- RU safe-decision coverage: `0.859375` при floor `0.65`;
- provider/composition errors: 0;
- wrong auto-executions: 0;
- unsafe modifier executions: 0;
- unsafe write executions: 0;
- high-confidence mismatches: 0;
- gate: passed.

Raw report:

- SHA-256:
  `ea7424b4f012cec6a308c88f61cdc321457ba1ab272e8fdb7e5177828b310949`;
- mode `0600`, size 41091 bytes;
- secret markers `TYPESAFE_API_KEY`, `sk-` и `Bearer ` не обнаружены;
- временный `/tmp/grace-typesafe-key` удалён сразу после benchmark.

Три RU cases вместо expected completed ушли в безопасный `clarify` из-за
route confidence ниже `0.70`. Один missing-argument case остался `clarify`, но
через другую terminal route. Ещё два terminal cases имели неиспользуемый
speculative argument answer; composition корректно его проигнорировала. Ни
один mismatch не был high-confidence и ни один не разрешил ошибочный tool
call.

## Pre-merge verification

После успешного v3 live gate повторно выполнены:

| Проверка | Результат |
| --- | --- |
| `make test` | PASS |
| `make test-integration` | PASS: SPEC-0001..0005, 0007 и 0008 |
| `./scripts/test-assistant-router.sh` | PASS в составе integration gate |
| `go vet ./cmd/... ./internal/...` | PASS |
| `git diff --check` | PASS |

Post-merge набор будет записан после integration merge; до него состояние
остаётся `VERIFIED`, а не `INTEGRATED`.
