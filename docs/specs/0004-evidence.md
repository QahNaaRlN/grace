# SPEC-0004 — verification evidence

- Статус проверки: integrated
- Implementation commit: `841edbd`
- Integration merge: `1c97604`
- Ветка: `codex/financial-period-summary`
- Дата: 2026-10-04

## Автоматические проверки

| Проверка | Результат |
| --- | --- |
| `make test` | Go tests, ESLint, OpenAPI TypeScript generation, TypeScript и Vite production build прошли |
| `make test-integration` | SPEC-0001, SPEC-0002, SPEC-0003 и SPEC-0004 прошли в отдельных compose-projects |
| `./scripts/test-analytics.sh` | Завершился `SPEC-0004 analytics integration checks passed` |
| `go vet ./cmd/... ./internal/...` | Прошёл без замечаний |
| `git diff --check` | Прошёл без замечаний |

## Исполняемое предметное доказательство

`scripts/test-analytics.sh` создаёт уникальный compose-project, автоматически
назначает host ports и удаляет PostgreSQL/API/volumes через `trap`. Сценарий
доказал:

- включение события ровно на `from` и исключение события ровно на `to`;
- реконструкцию opening/closing balance и равенство изменения баланса внешнему
  денежному потоку;
- независимые RUB и USD результаты без конвертации;
- отдельные income, purchase, refund, debt payment и однократный transfer
  volume;
- netting возврата против исходной категории покупки;
- `422` для отсутствующих, обратных, невалидных и превышающих пять лет границ;
- liquid balance только по debit/cash: созданный credit account не попал в
  доступные деньги;
- overdue и upcoming outstanding amounts относительно явных `asOf/through`;
- уменьшение commitment после частичного settlement без второго изменения
  баланса;
- серверный `availableAfterObligations` отдельно по валютам;
- отсутствие AI/LLM/provider/OCR модулей в dependency graph API.

## Согласованность чтения

Оба repository-метода открывают read-only `REPEATABLE READ` транзакцию и
вычисляют все части одного ответа внутри неё. Поэтому конкурентная запись не
может попасть только в balance, flow, category или obligation часть ответа.

## UI и контракт

- OpenAPI описывает оба read-only endpoint и все response schemas;
- generated `schema.d.ts` пересоздан из `api/openapi.yaml`, ручных DTO нет;
- React по умолчанию строит текущий календарный месяц в локальной временной
  зоне браузера и отправляет точные RFC 3339 границы с offset;
- period summary и cash position отображаются отдельными секциями и никогда не
  объединяют валюты в общий total;
- UI нейтрально поясняет, что cash position является арифметическим фактом, а
  не бюджетом, прогнозом или рекомендацией.

## Post-merge verification

На `main` после merge `1c97604` повторно прошли:

- `make test`;
- `make test-integration`, включая SPEC-0001, SPEC-0002, SPEC-0003 и SPEC-0004;
- `git diff --check`.

Основная копия после проверок оставалась чистой до фиксации этого evidence
update. SPEC-0004 интегрирована.
