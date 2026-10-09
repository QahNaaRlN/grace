# SPEC-0026 — implementation evidence

Дата: 2026-10-09. Принятая спецификация:
[0026-ai-assistant-receipts.md](0026-ai-assistant-receipts.md).
Ветка: `codex/ai-assistant-receipts`, foundation `d0031ca`, совмещена с main `f257f8e`.
Это локальная реализация и подготовка эксперимента, не завершённый live experiment.

## Реализовано

OpenRouter adapter с явным model/provider, ZDR/data-collection policy,
запретом fallback/tools/plugins, max_price, ограничением контекста/выхода,
точным разбором usage/cost и глобальными атомарными reservations.
Unknown billing блокирует новые запросы; нет автоматических retry.
Ключ загружается только сервером, в том числе из защищённого файла.
Отсутствие/неоднозначность файла отключает ИИ и не блокирует финансовое ядро.

Анализ календарного месяца использует Go/analytics facts, explicit coverage,
структурированные предположения и ссылки на fact IDs. Модель не публикует
числа вместо ядра. Согласие связано с purpose/model/provider/policy version;
разговоры owner-scoped и удаляются независимо от финансовой истории.

Чеки: normalize JPEG/PNG/WebP, EXIF removal/orientation, private storage,
typed nullable extraction, редактирование с revision, проверка discounts/quantity/
totals без float, поиск совпадений, immutable preview и ledger fingerprint.
Подтверждение create/link идемпотентно и атомарно. Manual account/receipt writes
используют тот же owner ledger lock. Cleanup jobs сохраняются в БД;
confirm/cancel запускают немедленную попытку, hourly worker обрабатывает TTL
и возобновляет cleanup после рестарта. Старые orphan файлы удаляются после 25 часов.
Нет публичных URL фото, permanent backup фото или provider write tools.

Миграции 000012–000015 следуют текущей 000011 unreconciled balances. Down
отказывается удалять usage, подтверждённую provenance и неудалённые фото.
OpenAPI обновлён, TypeScript сгенерирован. Mobile UI разделяет consent,
recognition, editing, preview и confirmation. Опциональный compose монтирует
key-file и отдельный volume, готовит ownership под non-root API UID 10001.

## Критерии приёмки

| AC | Статус | Исполняемое свидетельство / оставшаяся зависимость |
| --- | --- | --- |
| AC1 | Local PASS | `make test`; `TestOpenRouterSecretFileIsServerOnlyAndMissingKeyKeepsCore`; ручные finance/import/receipt/auth regression suites при выключенном ИИ |
| AC2 | Частично | `TestAnalysisUsesExactCoreFactsAndCalendarPeriod`, `TestAnalysisRejectsInventedNumbersReferencesAndAdditionalFields`; подготовлены 20 questions. Live ответы на весь corpus ещё не измерены |
| AC3 | Не завершён | `go run ./cmd/ai-eval`, `-render`, `-score`; 20 размеченных PNG (18 readable, 2 unknown). Сравнение 2–3 моделей, exact ≥95%, latency/cost/item errors требуют ключа и выбранных маршрутов |
| AC4 | Частично | `TestProductGroupNoMatchThresholdAndEvidence`, `TestProductGroupsBatchOnlyNamesAndPreserveUnknownBilling`, typed question, metered adapter и optional port, 30 labelled rows. Live precision/coverage threshold, billing reconciliation и disclosure TypeSafe ещё не подключены; manual group остаётся |
| AC5 | Local PASS | `TestReceiptDraftAtomicConfirmationIsolationAndCleanup`: edit не создаёт posting, 8 concurrent confirms дают одну posting, конфликт key/duplicate/stale ledger отвергается, link-existing не добавляет posting |
| AC6 | Local PASS | `TestReceiptAmountsDiscountsAndUnknowns`, точные ParseScaled, discount проверяется один раз, date-only сохраняет неизвестное время; `npm run generate:api` + build; оригинальные extractions отдельно от owner fields |
| AC7 | Частично | RLS cross-owner draft/usage tests; отсутствие principal отвергается; existing session/CSRF/Origin suites PASS; `TestOpenRouterNeverRetriesOrReflectsPrivateError`, unsupported fact references/extra fields. Live endpoint-policy gate и полный adversarial corpus остаются |
| AC8 | Local PASS | `TestGlobalAIUsageReservationIsolationAndUncertainBilling`, provider missing cost/provider change/truncation tests и exact decimal cost tests; budget включает fees и не расширяется financial role |
| AC9 | Local PASS, rollout pending | Receipt integration проверяет confirm/cancel/TTL/restart и искусственный storage failure: pending_delete не теряется, повторная очистка удаляет файл. Новый temporary volume отсутствует в pg_dump backup; фактический backup/restore на сервере перед rollout ещё не выполнялся |
| AC10 | Частично | `make test` PASS: 45 Web tests, lint/build, Go + offline Jev gates; `go vet ./...`, `git diff --check` PASS. Mobile browser на synthetic mock: consent → preview → confirmation и deletion status. Linux CI и live experiment проверяются отдельно |

## Наблюдаемые проверки

- `make test` — PASS на Windows через Git Bash, с разрешёнными локальными
  loopback test servers; 11 Web test files / 45 tests.
- `go vet ./...` — PASS.
- Docker API image build и проверка временного volume от non-root UID 10001 — PASS.
- `git diff --check` — PASS.
- `scripts/test-ai-experiment.sh` — PASS: usage concurrency/uncertainty и
  receipt create/link/isolation/cleanup; downgrade корректно отказался
  удалять подтверждённую provenance.
- Finance, receipts, obligations, analytics, imports, authentication,
  assistant tools, income-relative periods и router integration suites — PASS
  по отдельным изолированным PostgreSQL compose projects. На Windows нужен
  UTF-8 stdin adapter для JSON body в curl: прямые Unicode argv искажались.
- Полный `make test-integration` первоначально остановился на Unicode transport,
  затем на устаревшем SQL-free guard двух ранних спецификаций. Guards обновлены:
  assistant storage разрешён SPEC-0026, прямой financial SQL и обратная
  зависимость ядра от assistant остаются запрещены. Оставшиеся suites прошли.
- `make test-production` локально — ограничение Windows symlink/readlink:
  mock backup fixture сообщает `no current Grace release`. Это не production
  проверка и не заявлено PASS; Linux CI должен подтвердить bundle gate.
- [UI evidence](0026-ui-synthetic.jpg) — искусственные поля, mock API;
  эта картинка не доказывает live vision accuracy, server persistence или
  удаление upstream копий. Server persistence/cleanup проверены отдельно в БД.

## Осталось до завершения SPEC-0026

Аккаунт/пополнение готовы по ответу владельца; ключ не установлен.
Нужны protected key-file, подтверждённый allowance после fees, выбор свежих
model/provider policies, live synthetic comparison, Jev evaluation и metered
adapter, явное согласие на один личный анализ/чек, CI PASS и отдельный rollout
с backup/restore. Платные внешние inference calls в этой работе не выполнялись.
Production и реальные финансовые данные не менялись.

Автоматический reviewer сначала отклонил push как передачу исходного кода без
явного разрешения на GitHub. Владелец 2026-10-09 прямо разрешил отправку ветки
в `QahNaaRlN/grace` и создание draft PR для CI. Результат Linux CI фиксируется
после наблюдаемого завершения; local PASS не означает remote PASS.

Порядок установки и эксперимента: [ai-experiment.md](../ai-experiment.md).
