# SPEC-0037: реализация и проверка

Дата: 2026-10-13. Спецификация: [accepted SPEC-0037](0037-long-schedule-entry.md).
Ветка: `codex/long-schedule-entry`, база `765e36a` (SPEC-0036, потому что
`MoneyInput` используется в графике), implementation commit `b846159`.
Статус: `verified` локально (Windows); интеграция в `main` и production не выполнялись.

| Критерий | Реализация | Исполняемое свидетельство |
|---|---|---|
| AC1 | `internal/obligations/schedulepreview.go`, режим `equal`: даты через `internal/platform/calendar.Occurrence` (anchor day, конец месяца) | `TestPreviewEqualMortgage`: 120 платежей с 2027-01-15, последний 2036-12-15, первая/последняя суммы, итог; результат проходит `normalizeAndValidateCreate`. `TestPreviewEqualKeepsAnchorDayAcrossShortMonths`: 31.12 → 31.01 → 29.02.2028 → 31.03. Web: `CreateObligationForm.test.tsx` — ипотека 120 × 20 000 000 ₽: название, дата, количество, сумма, «Построить», «Сохранить» (6 действий) |
| AC2 | Режим `paste`: дата `ДД.ММ.ГГГГ`/ISO, ячейки по табу/`;`, иначе по группировке разрядов; ₽/«руб.»/NBSP; сумма — первая денежная колонка после даты или выбранная | `TestPreviewPasteBankTable`: 240 строк из таблицы банка (№, дата, платёж, долг, проценты, остаток) + заголовок + «Итого» → 240 платежей, 2 отклонённые строки с причинами. `TestPreviewPasteFormats`, `TestPreviewPasteChosenColumnAndLimits` (31.02, неубывающие даты, нет суммы, 0, лимит 600, 64 КиБ). Web: `ScheduleBuilder.test.tsx` — список пропущенных строк |
| AC3 | «Общая сумма» удалена из формы; перед созданием форма вызывает preview `rows`, `totalMinor` и платежи берутся из ответа Go | `CreateObligationForm.test.tsx`: поля нет, `totalMinor` = 120 × 2 000 000 000; строка с ошибкой не сохраняется |
| AC4 | Таблица свёрнута при > 6 строках (3 + 3 + «Показать все N»); сводка «N платежей · даты · итог» из preview `rows` с задержкой 300 мс | `ScheduleBuilder.test.tsx`: 6 видимых строк из 120, раскрытие до 120, итог `3 000 000,00 ₽`; Enter в последней сумме добавляет следующий месяц (дата от Go), «Заполнить сумму вниз» |
| AC5 | `LoanPanel` использует тот же `ScheduleBuilder`; оплаченные строки `locked`, генераторы заменяют только неоплаченные | `LoanPanel.test.tsx`: блокировка оплаченной строки, сохранение черновика с `sourceId/paidMinor`, отдельное подтверждение применения; генерация 24 платежей сохраняет оплаченную первую строку |
| AC6 | `api/openapi.yaml`: `previewObligationSchedule`, `SchedulePreviewRequest` (oneOf equal/paste/rows), `SchedulePreview`, `RejectedScheduleLine`; TS-типы перегенерированы, ручных DTO нет | `npm run generate:api` (+106 строк в `schema.d.ts`), `tsc -b`; `TestPreviewScheduleHandler`: 200, 400 (валидация и неизвестные поля), 422 `no_payments` |

Общие gates в worktree: `make test` — Go (все пакеты `ok`), web 22 файла / 85 тестов,
lint, build, офлайн Jev; `git diff --check` — чисто.

Отклонения от текста спецификации (внесены в спецификацию):

- В API добавлен режим `rows` — нужен для AC4, чтобы итог отредактированного
  графика считал Go, а не React.
- Колонка суммы по умолчанию — первая денежная после даты, а не последняя:
  в графиках банков последняя колонка — остаток долга.

`calendar.Occurrence` перенесён из `internal/commitments` в
`internal/platform/calendar` (commitments вызывает его через прежнюю функцию):
иначе `obligations` → `commitments` давал цикл импорта в
`commitments/integration_test.go`. Юнит-тесты календаря commitments проходят;
`make test-integration` (PostgreSQL) в этой среде не запускался — нужен Linux CI
или Docker перед интеграцией.
