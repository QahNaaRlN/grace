# SPEC-0049: реализация и проверка

Дата: 2026-10-16. Спецификация: [accepted SPEC-0049](0049-receipt-confirmation-refresh.md).
Ветка: `codex/receipt-confirmation-refresh` от `main` `5b70e38`, implementation commit `bc84bcc`.
Статус: `verified` локально (Windows).

| Критерий | Реализация | Исполняемое свидетельство |
|---|---|---|
| AC1 | `features/recognize-receipt-photo/api/use-confirm-receipt-draft.ts`: после успеха инвалидирует `accountQueries`, `financialEventQueries`, `receiptQueries`, `cashPositionQueries`, `periodSummaryQueries`, `statisticsQueries` (`all()`); `obligationQueries` — нет, по решению владельца | `use-confirm-receipt-draft.test.tsx` «confirms the reviewed preview…»: `invalidated()` равен ровно этим шести ключам |
| AC2 | `confirmReceipt` бросает ошибку без `data`; `onSuccess` не вызывается | Тот же файл, «rejects a failed confirmation and keeps the cache»: `mutateAsync` отклоняется с прежним текстом, `invalidated()` пуст |
| AC3 | Хук передаёт `draftId`, `preview` и `key` в существующий `confirmReceipt` | Первый тест: `POST /api/receipt-drafts/{draftId}/confirmations` с `path.draftId`, `Idempotency-Key`, `previewId`, `hash` |
| AC4 | `ReceiptPhoto.tsx` подтверждает через `useConfirmReceiptDraft().mutateAsync`; прямого вызова `confirmReceipt` нет, тексты не менялись | `ReceiptPhoto.test.tsx` — 3 теста проходят, утверждения не менялись (см. отклонение) |
| AC5 | — | `make test`: exit 0 — Go `ok`, web 75 файлов / 294 теста, ESLint, `steiger --fail-on-warnings`, Prettier, сборка, Jev offline checks; `git diff --check` — чисто |

## Отклонение от AC4

AC4 требовал, чтобы существующие тесты проходили «без правок». Хук использует
`useQueryClient`, поэтому компоненту нужен `QueryClientProvider`; в приложении
он уже есть выше по дереву. Два теста рендерили компонент без провайдера:
`ReceiptPhoto.test.tsx` и `AssistantPage.test.tsx` (страница содержит
`ReceiptPhoto`). В обоих добавлена только обёртка `queryHarness().wrapper`;
утверждения и данные тестов не менялись.

Не входит: отмена черновика (книгу не меняет, кэш не затрагивает).
