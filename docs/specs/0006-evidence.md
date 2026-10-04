# SPEC-0006 — implementation evidence

- Состояние: `READY_FOR_LIVE_EVALUATION`
- Ветка: `codex/jev-evaluation-harness`
- Implementation commit: `cc84b79`
- Dataset: `testdata/jev-evaluation/v1/cases.json`, version `v1`
- Question set: version `v1`
- Live model: не проверен; `TYPESAFE_API_KEY` отсутствует в окружении

SPEC-0006 остаётся в статусе `implementing`. Offline-реализация завершена, но
спецификация не может перейти в `verified` без реального benchmark и разбора
high-confidence ошибок.

## Реализованный контур

- development-only CLI `cmd/jev-eval` с командами `validate` и `run`;
- строгая загрузка и проверка versioned JSON dataset;
- 42 синтетических русскоязычных и 7 английских контрольных cases;
- три независимых `Choice`: tool route, period shape, argument presence;
- TypeSafe HTTP adapter для `POST /v1/systemone`;
- запрет aliases и обязательный versioned Jev model id;
- bounded retry для `429`/`529`, timeout/cancellation и типизированные errors;
- deterministic fake provider и end-to-end fake HTTP test;
- JSON report с model ids, usage, latency, answers, probabilities, confidence,
  RU/EN metrics, confusion matrix, confidence buckets и threshold analysis;
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

## Автоматические проверки

| Проверка | Результат |
| --- | --- |
| `make test` | PASS: Go tests, ESLint, TypeScript/Vite build и offline evaluator |
| `./scripts/test-jev-evaluation.sh` | PASS: `SPEC-0006 Jev evaluation offline checks passed` |
| `go test ./...` | PASS |
| `go vet ./cmd/... ./internal/...` | PASS |
| `git diff --check` | PASS |

Vite выводит существующее предупреждение о chunk больше 500 kB; build успешен,
а SPEC-0006 не меняет React bundle.

## Offline report evidence

End-to-end test поднимает локальный fake HTTP provider, проверяет Bearer header,
обрабатывает все 49 cases через реальный CLI/HTTP/validation/report pipeline и
получает:

- RU: 42 completed, tool accuracy `1.0`;
- EN: 7 completed, tool accuracy `1.0`;
- 42 RU observations в confidence bucket `[0.90,1.00]`;
- high-confidence model errors: 0.

Это доказательство корректности evaluator, а не качества Jev. Fake-provider
результаты запрещено использовать как product baseline.

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

## Неисполненный live gate

Проверка окружения 2026-10-04 вернула:

```text
TYPESAFE_API_KEY_MISSING
```

После предоставления owner-controlled key требуется выполнить, сохраняя report
вне репозитория:

```bash
TYPESAFE_API_KEY=<redacted> \
go run ./cmd/jev-eval run \
  --dataset testdata/jev-evaluation/v1/cases.json \
  --model jev-1.13.0 \
  --report /tmp/grace-jev-evaluation.json
```

Затем evidence должен быть дополнен фактическим model id, RU/EN metrics,
threshold coverage/error rate и разбором каждого high-confidence error. Только
после этого возможны независимая проверка и статус `verified`.
