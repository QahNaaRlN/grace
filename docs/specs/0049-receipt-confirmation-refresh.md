# SPEC-0049: обновление данных после подтверждения расхода по фото чека

- Статус: verified
- Автор: Claude Code по поручению владельца
- Дата: 2026-10-15
- Acceptance owner: владелец Grace
- Acceptance record: владелец 2026-10-16 в чате: «Принимаю SPEC-0049 как есть, без
  инвалидации обязательств».
- Implementation branch: `codex/receipt-confirmation-refresh`
- Implementation commit: `bc84bcc` ([evidence](0049-evidence.md))
- Зависит от: SPEC-0048, фазы 3–5 (PR #46–#48 в `main`, `d486582`)

## Проблема и пользовательский результат

После подтверждения расхода по фото чека остатки счетов, список операций,
денежная позиция, сводка периода, статистика и список чеков остаются
прежними до следующего запроса (`staleTime` 30 с). Пользователь видит
старый остаток сразу после записи расхода.

Результат: после успешного подтверждения эти данные обновляются сразу, как
после ручной записи операции.

## Scope

- Хук `useConfirmReceiptDraft` в `web/src/features/recognize-receipt-photo/api`.
  Он вызывает существующий `confirmReceipt` и после успеха инвалидирует
  ключи `accountQueries.all()`, `financialEventQueries.all()`,
  `receiptQueries.all()`, `cashPositionQueries.all()`,
  `periodSummaryQueries.all()`, `statisticsQueries.all()`.
- `ReceiptPhoto.tsx` подтверждает черновик через этот хук.
- Тест хука на `src/test/query-harness.tsx`.

## Non-goals

- Отмена черновика (`DELETE /api/receipt-drafts/{draftId}`): она меняет
  только статус черновика и не трогает книгу; черновик хранится в состоянии
  компонента, а не в кэше. Хук для неё не вводится.
- Тексты, вёрстка, OpenAPI, Go-код и база данных не меняются.
- Инвалидация в других мутациях (например, отсутствие `statisticsQueries`
  в `useCreateFinancialEvent`) не меняется.

## Текущее поведение и доказательства

- `POST /api/receipt-drafts/{draftId}/confirmations`
  (`internal/receiptdrafts/repository.go`, `Confirm`) в одной транзакции:
  при действии «создать» записывает покупку через
  `financialevents.CreateInTx` (проводки меняют остаток счёта), при
  действии «привязать» использует существующую операцию; в обоих случаях
  создаёт чек через `receipts.CreateInTx`.
- `ReceiptPhoto.tsx` вызывает `confirmReceipt` напрямую; ни одного вызова
  `invalidateQueries` в срезе нет. Зафиксировано в `0048-evidence.md`,
  фаза 5, «Найдено, но не меняется».

## Предметная модель и инварианты

- Арифметика и запись остаются в Go; клиент только обновляет кэш.
- Неуспешное подтверждение (`confirmReceipt` бросает ошибку) кэш не трогает.
- Обязательства и подписки не инвалидируются: покупка из чека связывается с
  ними только отдельным явным действием (`internal/obligations`,
  `internal/commitments`), подтверждение их не меняет.

## API и OpenAPI

Без изменений.

## Данные, миграция и rollback

Без изменений данных. Rollback — revert коммита.

## UI

Без видимых изменений, кроме того, что данные на других экранах свежие.

## Ошибки и негативные сценарии

- Ошибка подтверждения показывается тем же текстом, что сейчас; кэш не
  инвалидируется.
- Повтор с тем же `Idempotency-Key` ведёт себя как сейчас (сервер
  возвращает тот же результат); инвалидация повторяется безвредно.

## Критерии приёмки

- [x] AC1. Успешное подтверждение инвалидирует ровно шесть ключей из Scope.
- [x] AC2. Неуспешное подтверждение не инвалидирует ни одного ключа.
- [x] AC3. Хук передаёт `draftId`, `previewId`, `hash` и `Idempotency-Key`
  без изменений.
- [x] AC4. `ReceiptPhoto.tsx` не вызывает `confirmReceipt` напрямую; тексты
  и существующие тесты `ReceiptPhoto.test.tsx` проходят без правок.
- [x] AC5. `make test` и `git diff --check` проходят.

## Исполняемые проверки

- `web/src/features/recognize-receipt-photo/api/use-confirm-receipt-draft.test.tsx` (AC1–AC3).
- `ReceiptPhoto.test.tsx`, `lint:fsd`, `make test` (AC4–AC5).

## Открытые решения

1. Инвалидировать ли `obligationQueries.all()` для единообразия с
   `useCreateFinancialEvent`. Закрыто 2026-10-16: нет (решение владельца),
   см. инварианты.
