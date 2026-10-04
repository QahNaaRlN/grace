# SPEC-0006: оценка применимости Jev

- Статус: implementing
- Автор: Codex
- Дата: 2026-10-04
- Acceptance owner: owner
- Acceptance record: явное согласие owner в Codex-чате, 2026-10-04
- Implementation branch: `codex/jev-evaluation-harness`
- Implementation commit: отсутствует

## Проблема и пользовательский результат

Grace рассматривает Jev для маршрутизации естественного языка и других узких
семантических решений. Однако успешные демонстрационные запросы не доказывают
качество на русскоязычных финансовых формулировках Grace, не определяют
безопасные confidence-пороги и не показывают поведение при ошибках provider.

Нельзя подключать Jev к production assistant или финансовым данным до
воспроизводимой оценки на versioned наборе примеров. Нужен development-only
evaluation harness, который сравнивает типизированные ответы Jev с ожидаемыми
решениями и формирует машиночитаемый отчёт без изменения production runtime.

Пользовательский результат этапа — доказательное решение, подходит ли Jev для
первого read-only маршрутизатора Grace и какие вопросы требуют уточнения или
fallback. Этап сам по себе не добавляет пользователю AI-функции.

## Scope

- добавить versioned evaluation dataset из синтетических или необратимо
  обезличенных примеров;
- покрыть русскоязычные формулировки и небольшую английскую контрольную группу;
- оценить следующие judgment families:
  - выбор одного из пяти инструментов SPEC-0005;
  - `clarify` для недостаточно определённого запроса;
  - `unsupported` для запроса вне возможностей текущего assistant;
  - форма периода: calendar period, relative period, explicit range или
    missing period;
  - присутствие необходимых аргументов без генерации отсутствующих значений;
- представить инструмент и форму периода как закрытые `Choice` с явным
  no-match вариантом;
- разрешить несколько независимых вопросов в одном provider request, если они
  используют одно состояние;
- добавить development-only Go evaluator поверх TypeSafe HTTP API;
- использовать API key только из окружения;
- требовать явный versioned model id для benchmark;
- записывать фактический model id, dataset version, question-set version,
  token usage, latency, ответы, probabilities и confidence;
- вычислять exact-match, confusion matrix, coverage, clarification rate,
  high-confidence error rate и confidence buckets;
- отдельно показывать ошибки модели, отсутствие кандидата, ошибки composition
  кода и ошибки provider;
- поддержать offline validation dataset и отчёта без credentials или сети;
- добавить deterministic fake provider для unit tests;
- документировать команду воспроизводимого локального запуска и формат отчёта.

## Non-goals

- production provider wiring в `cmd/api`;
- публичный assistant/chat endpoint, OpenAPI или React UI;
- выполнение выбранного tool против пользовательской базы;
- передача реальных финансовых записей TypeSafe;
- generative ответы и подключение другой LLM;
- автоматический tool loop;
- AI-originated writes и confirmation policy;
- категоризация production-операций;
- OCR, vision и ReceiptDraft;
- merchant alignment, recurring-payment detection и attention ranking;
- выбор окончательных production thresholds до получения baseline;
- доказательство пригодности Jev для всех будущих сценариев Grace;
- отдельный AI-service.

## Текущее поведение и доказательства

- SPEC-0005 интегрировал пять read-only definitions и typed executor;
- имена инструментов стабильны и не поддерживают fuzzy matching;
- tool arguments проходят строгую серверную валидацию до вызова dependency;
- provider SDK/HTTP client отсутствует в production dependency graph;
- публичного assistant API, conversation storage и chat UI нет;
- Grace работает без LLM и внешнего AI provider;
- продуктовые тексты и ожидаемые пользовательские вопросы преимущественно
  русскоязычные;
- актуальная документация TypeSafe указывает английский как основной язык и
  требует проверки других языков на собственных данных.

## Предметная модель и инварианты

`EvaluationCase`:

- `id` — стабильный уникальный identifier;
- `language` — `ru` или `en`;
- `utterance` — синтетический/обезличенный пользовательский текст;
- `referenceDate` — явная дата для относительных выражений, если нужна;
- `expectedTool` — одно из пяти имён, `clarify` или `unsupported`;
- `expectedPeriodShape` — закрытый enum;
- `expectedArgumentPresence` — ожидаемое наличие нужных частей;
- `tags` — классы сложности и негативные сценарии.

`QuestionSet`:

- имеет явную version;
- содержит полное значение решений в instructions/criteria, а не только в id;
- включает `clarify` и `unsupported` как реальные варианты;
- не просит модель вычислять даты, денежные суммы или финансовые агрегаты;
- не включает ответ предыдущего вопроса в независимый вопрос того же request.

`EvaluationObservation`:

- case id и фактический model id;
- типизированные ответы, probabilities и confidence;
- token usage и latency;
- нормализованный outcome evaluator;
- error class без secret или authorization header.

Инварианты:

1. Dataset не содержит реальные имена, номера счетов, банковские реквизиты,
   fiscal identifiers или историю пользователя.
2. Один и тот же dataset/question-set/model tuple воспроизводим и явно указан
   в отчёте.
3. Benchmark запрещает `jev-latest` и другие плавающие aliases.
4. Jev не получает tool results или строки из PostgreSQL.
5. Evaluator не импортируется `cmd/api` или финансовыми модулями.
6. `make test` и обычный запуск Grace не требуют credentials и сети.
7. Отсутствующий API key завершает live run до сетевого вызова понятной
   ошибкой.
8. API key и Authorization header никогда не сериализуются в отчёт и логи.
9. Неизвестный case field, duplicate id, неизвестный enum и пустой utterance
   отклоняются offline validation.
10. Provider timeout/rate-limit/overload/invalid response не учитываются как
    неправильная классификация модели и отражаются отдельным error class.
11. Никакой результат benchmark не разрешает write или production routing.
12. Confidence оценивается как сигнал политики, а не как доказательство
    правильности.

## API и OpenAPI

Публичный `api/openapi.yaml` не меняется.

Evaluator использует серверный HTTP endpoint TypeSafe
`POST /v1/systemone` через узкий development-only adapter. Request содержит
`state`, явный versioned `model` и map типизированных questions. Внешний ответ
проверяется до использования; неизвестные типы или отсутствующие ответы
считаются provider protocol error.

Development-only evaluator находится в `cmd/jev-eval`, а его код — в
`internal/jeveval`. CLI предоставляет команды `validate` и `run`; `cmd/api` не
импортирует evaluator.

## Данные, миграция и rollback

PostgreSQL и migrations не меняются.

В Git хранятся dataset, его schema/version и код evaluator. Сырые live-отчёты
по умолчанию создаются вне репозитория или в явно ignored каталоге. Небольшой
обезличенный baseline report можно добавить отдельным review после проверки на
секреты и персональные данные.

Rollback удаляет development-only evaluator и fixtures; финансовые данные и
production runtime не меняются.

## UI

Изменений React нет. Evaluation report читается разработчиком и owner, а не
показывается конечному пользователю.

## Ошибки и негативные сценарии

- отсутствующий API key: live run не начинается;
- плавающий model alias: benchmark отклоняется;
- malformed dataset или duplicate id: offline validation завершается ошибкой;
- пустой/неполный provider response: `provider_protocol_error`;
- `401`: `provider_authentication_error` без вывода credentials;
- `422`: `provider_request_error` с безопасным диагностическим сообщением;
- `429`/`529`: bounded retry с backoff и отдельный итоговый provider error;
- timeout/cancellation: запрос прекращается и отражается отдельно;
- модель выбирает отсутствующий аргумент: composition отклоняет результат;
- tool выбран верно, но период отсутствует: outcome требует `clarify`, tool не
  считается исполнимым;
- низкий confidence: наблюдение сохраняется, evaluator не подменяет его
  guessed answer;
- русская и английская группы не смешиваются в единственную aggregate metric.

## Критерии приёмки

- [ ] Dataset schema и offline validation реализованы и покрыты unit tests.
- [ ] Dataset содержит согласованный owner минимум русскоязычных примеров для
      каждого tool, `clarify`, `unsupported` и каждого period shape.
- [ ] Fixtures синтетические/обезличенные и проходят автоматическую проверку
      запрещённых полей/паттернов.
- [ ] Question set использует только типизированные Choice/Noul/Score и имеет
      явную version.
- [ ] Live evaluator требует versioned model id и сохраняет фактический id из
      provider response.
- [ ] Fake provider обеспечивает полностью offline unit tests success/error
      paths.
- [ ] Report содержит раздельные метрики для `ru` и `en`, confusion matrix,
      coverage и high-confidence errors.
- [ ] Ошибки provider отделены от ошибок model judgment и composition.
- [ ] API key, authorization header и реальные финансовые данные отсутствуют в
      fixtures, отчёте и логах.
- [ ] Production API dependency graph не содержит TypeSafe adapter.
- [ ] `api/openapi.yaml`, migrations и React не изменены.
- [ ] В evidence записаны dataset/question/model versions, точная команда,
      итоговые метрики и разобранные high-confidence ошибки.
- [ ] `make test`, offline evaluator checks, `go vet` и `git diff --check`
      проходят без credentials и сети.

## Исполняемые проверки

Минимальные offline gates:

```bash
make test
go test ./...
go vet ./cmd/... ./internal/...
go run ./cmd/jev-eval validate --dataset testdata/jev-evaluation/v1/cases.json
git diff --check
```

Live benchmark после предоставления owner-controlled credentials:

```bash
TYPESAFE_API_KEY=<redacted> \
go run ./cmd/jev-eval run \
  --dataset testdata/jev-evaluation/v1/cases.json \
  --model <versioned-model-id> \
  --report <path-outside-repository>
```

## Открытые решения на момент принятия

1. Минимальное количество cases на tool/period/negative class.
2. Точный package path и CLI интерфейс development-only evaluator.
3. Формат dataset и report: JSON, JSONL либо сочетание JSONL + Markdown summary.
4. Нужен ли отдельный Noul для достаточности аргументов или достаточно Choice
   с `clarify`.
5. Какие threshold candidates сравнивать после baseline; cookbook thresholds
   не принимаются автоматически.
6. Какая доля high-confidence ошибок допустима для перехода к отдельной
   production routing specification.
7. Как owner предоставляет локальный API key и кто имеет доступ к live report.
8. Нужно ли проверять конкретный versioned Jev model либо несколько доступных
   моделей из `GET /v1/models`.
9. Разрешено ли сохранять обезличенный baseline report в репозитории.
10. Требуется ли отдельный privacy/legal review до любого эксперимента с
    реальными пользовательскими формулировками.

## Решения для implementation

Owner разрешил приступить к реализации в Codex-чате 2026-10-04. В рамках
принятого scope открытые технические пункты разрешены так:

1. Dataset содержит не менее пяти русскоязычных cases на каждый tool outcome,
   `clarify` и `unsupported`, не менее пяти на каждый period shape и не менее
   одного английского case на каждый tool outcome.
2. CLI/package paths: `cmd/jev-eval` и `internal/jeveval`.
3. Dataset и машиночитаемый report используют JSON; краткая сводка выводится в
   stdout.
4. Достаточность аргументов — закрытый `Choice` с вариантами `sufficient`,
   `missing` и `not_applicable`, чтобы иметь распределение и confidence.
5. Report сравнивает threshold candidates `0.50`, `0.60`, `0.70`, `0.80`,
   `0.90`, `0.95`; ни один не становится production policy автоматически.
6. Go/no-go error budget остаётся решением будущей production routing spec.
7. Live run читает `TYPESAFE_API_KEY` только из локального окружения;
   сохранение ключа или live report в Git запрещено.
8. Один запуск проверяет один явный versioned model id; сравнение моделей
   выполняется отдельными отчётами.
9. Baseline report не добавляется в Git этим work item.
10. Реальные пользовательские формулировки запрещены до отдельного
    privacy/legal review.
