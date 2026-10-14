# SPEC-0046: реализация и проверка

Дата: 2026-10-14. Спецификация: [accepted SPEC-0046](0046-payment-due-soon.md).
Ветка: `codex/payment-due-soon`, implementation commit `20a9d31`.
Статус: `verified` локально (Windows + Docker PostgreSQL).

| Критерий | Реализация | Исполняемое свидетельство |
|---|---|---|
| AC1 | `obligations.DueSoonDays = 3`, `DueIn`, `DueSoon`; `CashScheduledPayment.timing = due_soon` + `daysUntilDue`; `ScheduledPayment.dueSoon` + `daysUntilDue` | Unit `TestDueSoonWindows` (0/1/3 дня — скоро; 4 дня, вчера, оплачен — нет; переход через месяц и год). Интеграция `TestDueSoonInCashPositionAndSchedules`: даты −1/0/1/3/4 дня → overdue, due_soon ×3, upcoming; в графике `dueSoon` и `daysUntilDue` совпадают |
| AC2 | `due_soon` входит в «Предстоит» | Интеграция: просрочено 10 ₽, предстоит 40 ₽, всего 50 ₽ |
| AC3 | Обзор: «Завтра», «Через 3 дня» в «Ближайших платежах»; «Платежи»: жёлтый бейдж «Сегодня» рядом со статусом; «Частично оплачен» — `info`; позиция по обязательствам — «· завтра» | `DueSoon.test.tsx`: тон `warning` у «Завтра» и «Сегодня», `info` у «Частично оплачен», дальний платёж без пометки |

Проверки: `make test-integration` (все пакеты, exit 0); `make test` — Go `ok`, web 197 тестов,
lint, build; `git diff --check` — чисто.

Email-напоминания за 3 и 1 день (SPEC-0034) не менялись: они работают для подписок и
кредитов с включённым переключателем; окна совпадают с порогом «скоро».
