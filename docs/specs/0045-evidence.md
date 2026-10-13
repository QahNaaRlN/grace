# SPEC-0045: реализация и проверка

Дата: 2026-10-13. Спецификация: [accepted SPEC-0045](0045-schedule-by-period.md).
Ветка: `codex/schedule-by-period`, база `01eb67e`, implementation commit `8a96d60`.
Статус: `verified` локально (Windows).

| Критерий | Реализация | Исполняемое свидетельство |
|---|---|---|
| AC1 | `internal/obligations/schedulepreview.go`: режим `period` (число месяца, месяц начала и окончания включительно) | `TestPreviewPeriodMortgage`: 15-е, янв 2027 – дек 2036 → 120 платежей, последний 2036-12-15; UI: `ScheduleBuilder.test.tsx` «builds payments from a day of month…» |
| AC2 | Каждый месяц ограничивает число сам по себе; «Последний день месяца» = 31 | `TestPreviewPeriodClampsEachMonthIndependently`: 31 → 2028-01-31, 02-29, 03-31, 04-30; 30 в феврале → 28 |
| AC3 | Результат попадает в ту же таблицу `ScheduleBuilder` и сохраняется как в SPEC-0037 | UI-тест: свёрнутая таблица «Показать все 120», даты строк |

Проверки: `make test` — Go `ok`, web 26 файлов / 169 тестов, lint, build, Jev;
`git diff --check` — чисто. Ошибки ввода (конец раньше начала, 0/32, 601 месяц,
нет суммы) — `TestPreviewPeriodRejectsInvalidRequests`.

Отклонение от текста: `dayOfMonth` — число 1–31 без значения `"last"`:
31 с ограничением месяцем совпадает с «последним днём» во всех месяцах.
