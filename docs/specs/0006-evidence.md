# SPEC-0006 — implementation and verification evidence

- Состояние: `VERIFIED`
- Ветка: `codex/jev-evaluation-harness`
- Implementation commit: `3f12d8c`
- Dataset: `testdata/jev-evaluation/v1/cases.json`, version `v1`
- Question set: version `v1`
- Requested model: `jev-1.13.0`
- Actual model: `jev-1.13.0`
- Verification date: 2026-10-05

SPEC-0006 прошла offline-проверки и live benchmark. Результат подтверждает
перспективность Jev для отдельной confidence-gated read-only маршрутизации, но
не разрешает production wiring и не устанавливает окончательный threshold.

## Реализованный контур

- development-only CLI `cmd/jev-eval` с командами `validate`, `run` и
  `analyze`;
- строгая загрузка и проверка versioned JSON dataset;
- 42 синтетических русскоязычных и 7 английских контрольных cases;
- три независимых `Choice`: tool route, period shape, argument presence;
- TypeSafe HTTP adapter для `POST /v1/systemone`;
- запрет aliases и обязательный versioned Jev model id;
- bounded retry для `429`/`529`, timeout/cancellation и типизированные errors;
- deterministic fake provider и end-to-end fake HTTP test;
- JSON report с model ids, usage, latency, answers, probabilities, confidence,
  RU/EN metrics, confusion matrix, confidence buckets и threshold analysis;
- offline reanalysis сохранённых raw answers после исправления composition;
- report создаётся вне репозитория с mode `0600` без перезаписи;
- production `cmd/api` не импортирует `internal/jeveval`.

## Offline dataset evidence

Команда:

```bash
go run ./cmd/jev-eval validate \
  --dataset testdata/jev-evaluation/v1/cases.json
```

Результат:

```text
validated dataset=v1 cases=49
```

Validator и unit tests проверяют unknown fields, duplicate ids, enums,
expected-outcome combinations, reference dates, минимальное покрытие классов и
запрещённые secret/identifier patterns.

## Live benchmark

Owner выполнил benchmark с ключом только в окружении:

```bash
TYPESAFE_API_KEY=<redacted> \
go run ./cmd/jev-eval run \
  --dataset testdata/jev-evaluation/v1/cases.json \
  --model jev-1.13.0 \
  --report /tmp/grace-jev-evaluation-2026-10-04.json
```

Raw report:

- путь: `/tmp/grace-jev-evaluation-2026-10-04.json`;
- SHA-256:
  `94152d187ff8d7d3ee7f629600f6e26899b41a414e0a39d4952ab973ef20f83c`;
- mode: `0600`;
- provider errors: 0;
- requested/actual model: `jev-1.13.0`;
- secret markers и authorization header: не обнаружены.

Первичный анализ выявил три `composition_error`. Причиной были не ответы Jev,
а evaluator: он применял speculative argument answer к ветвям, где период не
используется. Composition исправлена так, чтобы неиспользуемые независимые
ветви не влияли на outcome. Raw answers повторно обработаны без нового
provider request:

```bash
go run ./cmd/jev-eval analyze \
  --dataset testdata/jev-evaluation/v1/cases.json \
  --input-report /tmp/grace-jev-evaluation-2026-10-04.json \
  --report /tmp/grace-jev-evaluation-2026-10-05-analyzed.json
```

Reanalyzed report:

- путь: `/tmp/grace-jev-evaluation-2026-10-05-analyzed.json`;
- SHA-256:
  `47c45d1cc118b5da51a3dd3d64c419878ade451cc7b4f9e491442fcf706b1d51`;
- mode: `0600`;
- provider errors: 0;
- composition errors: 0;
- secret markers и authorization header: не обнаружены.

## Итоговые метрики

| Язык | Completed | Tool exact / accuracy | Period exact / accuracy | Arguments exact / accuracy | Clarification rate |
| --- | ---: | ---: | ---: | ---: | ---: |
| RU | 42/42 | 38 / `0.9048` | 29 / `0.6905` | 36 / `0.8571` | `0.2143` |
| EN | 7/7 | 7 / `1.0000` | 4 / `0.5714` | 7 / `1.0000` | — |

RU confidence policy candidates для итогового исполнимого route:

| Threshold | Eligible | Coverage | Errors | Error rate |
| ---: | ---: | ---: | ---: | ---: |
| `0.50` | 36/42 | `0.8571` | 0 | `0.0000` |
| `0.60` | 34/42 | `0.8095` | 0 | `0.0000` |
| `0.70` | 32/42 | `0.7619` | 0 | `0.0000` |
| `0.80` | 30/42 | `0.7143` | 0 | `0.0000` |
| `0.90` | 20/42 | `0.4762` | 0 | `0.0000` |
| `0.95` | 12/42 | `0.2857` | 0 | `0.0000` |

EN сохранил 7/7 eligible и 0 errors на thresholds `0.50`–`0.80`, 6/7 на
`0.90` и 5/7 на `0.95`. Размер английской контрольной группы недостаточен для
отдельного production-вывода.

High-confidence route errors отсутствуют. Это результат малого синтетического
baseline, а не доказательство нулевого риска на production-трафике.

## Низкоуверенные расхождения и capability gap

Четыре RU tool-routing расхождения имеют итоговый route confidence ниже
`0.50`:

| Case | Запрос | Expected | Outcome | Route confidence |
| --- | --- | --- | --- | ---: |
| `ru-accounts-05` | На каком счёте сколько осталось? | accounts | clarify | `0.25` |
| `ru-summary-03` | Куда ушли деньги после аванса? | period summary | clarify | `0.45` |
| `ru-summary-05` | Сравни поступления и покупки после последнего аванса | period summary | clarify | `0.33` |
| `ru-cash-03` | Хватит ли текущего остатка на обязательства до следующего поступления? | cash availability | clarify | `0.36` |

Три запроса задают границы периода через финансовое событие: последний аванс
или следующее поступление. Инструменты SPEC-0005 принимают конкретные даты и не
умеют сами находить такие события. Поэтому осторожный `clarify` выявляет
реальный capability gap; dataset v1 после просмотра результата не
перенастраивался.

Period-shape accuracy недостаточна для прямого исполнения аргументов. В
следующей спецификации нужно разделить календарные, относительные и
event-relative интервалы и детерминированно преобразовывать подтверждённую
форму периода в даты до вызова финансового core.

## Вывод о применимости

Jev применим как кандидат для узкого read-only routing слоя Grace при
обязательном confidence gate и безопасном `clarify`/fallback. Baseline показал
нулевую ошибку среди RU routes с confidence не ниже `0.50` при coverage
`85.71%`, но dataset мал, синтетичен и был использован для первичной оценки.

До production-подключения необходимы отдельная accepted specification,
расширенный held-out dataset, политика для event-relative периодов,
детерминированная date composition и независимо принятый threshold. SPEC-0006
не добавляет provider в production dependency graph, не разрешает передачу
финансовых данных и не разрешает AI-originated writes.

## Автоматические проверки

После исправления composition выполнены:

| Проверка | Результат |
| --- | --- |
| `make test` | PASS: Go tests, ESLint, TypeScript/Vite build и offline evaluator |
| `./scripts/test-jev-evaluation.sh` | PASS: `SPEC-0006 Jev evaluation offline checks passed` |
| `go test ./...` | PASS |
| `go vet ./cmd/... ./internal/...` | PASS |
| `git diff --check` | PASS |

Vite выводит существующее предупреждение о chunk больше 500 kB; build успешен,
а SPEC-0006 не меняет React bundle.

## Архитектурная граница

`scripts/test-jev-evaluation.sh` выполняет `go list -deps ./cmd/api` и завершает
проверку ошибкой, если production API зависит от `internal/jeveval`.

Изменения отсутствуют в:

- `api/openapi.yaml`;
- `migrations/`;
- `web/src/`;
- production composition `cmd/api`.

Evaluator не читает PostgreSQL, не вызывает `assistant.Executor`, не меняет
финансовые записи и не принимает пользовательские данные из Grace.

## Acceptance matrix

Все критерии SPEC-0006 закрыты исполняемыми проверками, versioned live report и
метриками выше. Live artifacts намеренно сохранены вне Git. Состояние
`VERIFIED` означает завершение принятого evaluation scope, но не интеграцию
production routing и не принятие следующей спецификации.
