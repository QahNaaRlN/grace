# SPEC-0003 — verification evidence

- Статус проверки: verified
- Implementation commit: `dce34e5`
- Integration merge: отсутствует
- Ветка: `codex/obligations`
- Дата: 2026-10-04

## Автоматические проверки

| Проверка | Результат |
| --- | --- |
| `make test` | Go tests, ESLint, OpenAPI TypeScript generation, TypeScript и Vite production build прошли |
| `make test-integration` | SPEC-0001, SPEC-0002 и SPEC-0003 прошли в отдельных compose-projects |
| `./scripts/test-obligations.sh` | Завершился `SPEC-0003 obligation integration checks passed` |
| `go vet ./cmd/... ./internal/...` | Прошёл без замечаний |
| `git diff --check` | Прошёл без замечаний |

Предметный integration test доказал:

- миграции с нуля, rollback пустой схемы и отказ rollback при наличии данных;
- атомарное создание obligation и упорядоченного schedule;
- точное равенство total сумме платежей и серверную проверку дат;
- отсутствие новых проводок и изменения баланса от самого графика;
- вычисление planned, overdue, partially_paid, paid и completed для явного
  `asOf`;
- `201` для создания и `200` для идемпотентного повтора;
- `409` для конфликтующего payload и повторного использования debt payment;
- settlement только для однопроводочного debit `debt_payment` той же валюты;
- частичные оплаты, отказ переплаты и последовательное закрытие графика;
- неизменность баланса и количества проводок при settlement;
- сериализацию конкурентных settlements одного scheduled payment;
- отсутствие AI/LLM/provider/OCR модулей в dependency graph API.

Регрессионный тест receipts переведён с относительного `down 1` на явную
границу `goto 3`, поэтому последующие миграции не меняют смысл его rollback
проверки.

## Браузерный smoke test

Полный compose-стек был запущен из implementation worktree на изолированных
портах `55437/18083/15176`. Через React подтверждены:

- создание обязательства `Smoke рассрочка` на `100,00 ₽`;
- показ прошедшей даты как `Просрочен` без изменения остатка;
- создание `debt_payment` на `40,00 ₽` через форму финансового события;
- уменьшение счёта ровно один раз: с `125 000,00 ₽` до `124 960,00 ₽`;
- выбор этого события для settlement;
- состояние `Частично оплачен · просрочен`, paid `40,00 ₽` и remaining
  `60,00 ₽`;
- неизменный остаток `124 960,00 ₽` после settlement.

Временный браузерный таб и compose-стек после проверки удалены.

## Post-merge verification

Ожидает интеграции implementation commit в `main`.
