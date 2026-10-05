# SPEC-0017: Сохраняемые черновики и подтверждение импорта

- Статус: accepted
- Автор: Codex
- Дата: 2026-10-06
- Acceptance owner: owner
- Acceptance record: 2026-10-06 владелец поручил «Приступить к спеке 17» после представления полного draft и вопроса о принятии scope; предложенные решения и критерии приняты этим поручением.
- Implementation branch: codex/persistent-import-drafts
- Implementation commit: отсутствует

## Проблема и пользовательский результат

SPEC-0016 сохраняет офлайн-результат сверки, но решения владельца не живут
в Grace. Нужны сохраняемые черновики, просмотр исходных свидетельств,
построчные решения и предварительный просмотр точного результата перед
записью. После перезагрузки страницы проверка продолжается с того же места.
Подтверждение создаёт только явно выбранные допустимые события или связывает
источник с уже существующим событием без повторного движения денег.

## Scope

- Go-модуль `internal/imports`, PostgreSQL, HTTP API и мобильный web-экран.
- Загрузка snapshot `grace.sheet-snapshot.v1` из файла; сервер заново запускает
  `internal/importreview`. Готовый отчёт CLI не принимается как доверенный результат.
- Сохранение всех domains, исходных ячеек, формул, metadata, issues и candidates.
- Исправляемые operation drafts, решения по кандидатам и история проверки.
- Явные действия: оставить нерешённым, исключить с причиной, создать событие,
  связать с существующим событием, признать дублем другой строки этого batch.
- Preview выбранной группы и атомарное подтверждение её действий.
- Создание только существующих `purchase`, `income_received`, `debt_payment`,
  `internal_transfer`, `refund` с полной валидацией финансового ядра.
- Постоянная provenance-связь источника, решения, подтверждения и события.

## Non-goals

Google-синхронизация, PDF/OCR, LLM и изменяющие tools ассистента, новые виды
финансирования, кредитные договоры, товарный каталог, новые счета/категории
из aliases, публикация планов/долговых снимков/документов, изменение существующих
проводок и автоматическая дедупликация. Автоматическая загрузка приватного
снимка, подтверждение реальных операций и production rollout не входят
в реализацию. Владелец загружает и подтверждает данные через новый экран.

## Текущее поведение и доказательства

Основание: main `1f0115f`, [SPEC-0016](0016-spreadsheet-import-review.md),
[evidence](0016-evidence.md), [анализ](../spreadsheet-integration.md).
`internal/importreview/types.go` определяет snapshot/report v1, raw records,
issues и кандидатов; модуль работает без SQL/сети. Приватный dry run обработал
191 строку операций, включая 50 дополнительных; реальные значения не становятся
fixtures. `financialevents.Repository.Create` самостоятельно открывает и
завершает транзакцию, ставит `source=manual`. Для атомарного импорта требуется
общая операция создания внутри переданной tenant-транзакции. Даты событий
сейчас представлены `occurredAt` с временем, а источник может иметь precision=date.
Финансовые таблицы защищены RLS; mutations требуют session, CSRF и Origin.

## Предметная модель и инварианты

1. ImportBatch принадлежит owner: UUID, sourceId, capturedAt, hash исходных
   байтов, version/profile нормализатора, metadata, неизменяемый snapshot,
   revision и статус `reviewing`/`completed`. Completed означает, что все operation
   rows опубликованы, связаны или исключены; остальные domains остаются
   сохранёнными свидетельствами, а не импортированными финансовыми фактами.
2. SourceRecord имеет UUID и уникальный `(owner,batch,snapshotRecordId)`;
   хранит coordinates, originalId, raw, первоначальную нормализацию и issues.
   Правки создают отдельную версию решения, никогда не переписывают источник.
3. ReviewDecision хранит actor, время, revision, действие, исправленные поля,
   обоснование и решения по конфликтам. Редактирование использует optimistic
   concurrency по revision всего batch; гонка возвращает conflict.
4. Решения `create`, `link_existing`, `duplicate_of`, `exclude`, `unresolved`
   не создают проводок при сохранении. Опубликованное решение неизменяемо.
   `duplicate_of` ссылается на каноническую operation row того же batch;
   цепочки и циклы запрещены. Каноническая строка уже опубликована либо
   включена в ту же confirmation с действием create/link_existing.
5. Candidates являются подсказками. Каждое пересечение выбранной строки
   требует решения владельца: «это разные операции» с причиной либо duplicate.
   Сервер дополнительно ищет кандидатов среди существующих owner events;
   совпадение даты/суммы никогда автоматически не создаёт или не исключает факт.
6. Новый snapshot с тем же sourceId не подтверждает новый факт. Повтор тех же
   байтов одному owner возвращает существующий batch. Новый hash создаёт batch,
   но exact matches с опубликованными source records требуют явного link/duplicate
   либо обоснования отдельной операции; совпадение coordinates само по себе
   не доказывает тождество между версиями.
7. Preview — серверный неизменяемый объект: owner, batchRevision, выбранные
   record IDs, полный набор действий/проводок/provenance, warnings, блокирующие
   issues, hash содержания, expiry (15 минут). Максимум 100 строк в confirmation.
   Суммы итогов и изменения баланса вычисляются Go отдельно по каждой валюте.
8. Confirmation передаёт previewId и его hash, требует отдельного нажатия
   владельца и Idempotency-Key. Сервер проверяет revision, срок, владельца,
   актуальные ссылки и кандидатов; новое совпадение после preview требует нового
   review. Ledger, provenance, receipt подтверждения и статусы записываются
   в одной tenant-транзакции. При любой ошибке вся группа откатывается.
9. Повтор подтверждения возвращает тот же результат, включая после потери
   ответа или истечения preview; другой payload с тем же ключом — conflict.
   Новые ключи/два параллельных запроса не публикуют одну строку дважды:
   уникальная provenance-связь и блокировка batch защищают от повтора.
   Проверка новых совпадений и запись сериализуются с другими импортами
   и manual financial writers того же owner (общий transaction lock).
10. Link/duplicate создают только provenance. Link_existing требует явного
    выбора доступного owner event, показанных различий и причины при отличиях;
    неподходящий kind/currency запрещён. Они не меняют event, Receipt, Settlement
    или баланс. Несколько свидетельств могут относиться к одному событию.
11. Деньги — положительный int64 minor units с явной валютой; она должна
    соответствовать счетам. Internal transfer требует двух разных счетов одной
    валюты и двух атомарных проводок. Refund требует уже существующую purchase
    владельца; создание purchase и её refund одной группой в этом scope запрещено.
12. Date-only сохраняет исходную точность. Для create владелец явно вводит
    время в Asia/Yekaterinburg; решение хранит precision и основание уточнения.
    Автоматическая подстановка полуночи запрещена. Link_existing сохраняет
    date-only свидетельство без изменения времени существующего события.
13. План, HOLD, unknown kind, заёмное поступление и неподдерживаемая долговая
    семантика не превращаются в income/debt_payment. Неустранённые смысловые
    конфликты блокируют create. Необязательные funding/subcategory/priority
    сохраняются как свидетельства с видимым предупреждением о непереносе
    в financial core; исчезновение warning не должно означать решение issue.
14. Source финансового события, созданного импортом, равен `import`.
    Financial core не зависит от imports; imports вызывает общий финансовый
    writer в tenant-транзакции. Manual API сохраняет текущие контракты.
15. Загрузка, review и preview не меняют финансовые остатки. Формулы и заметки
    являются данными; они не выполняются и не отображаются как HTML.

## API и OpenAPI

Предлагаемые routes под `/api/imports` (точный prefix сверяется с текущим router):

| Метод/путь | Контракт |
| --- | --- |
| POST `/batches` | JSON snapshot v1, Idempotency-Key; 201 новый batch, 200 существующий |
| GET `/batches` | Owner-only список, cursor pagination, limit 1–100 |
| GET `/batches/{id}` | Metadata, revision, counts по статусам/domains, без всего snapshot |
| GET `/batches/{id}/records` | Cursor, domain/status filters, limit 1–100 |
| GET `/batches/{id}/records/{recordId}` | Raw/provenance, issues, draft, candidates, audit history |
| PATCH `/batches/{id}/records/{recordId}` | expectedRevision, действие/поля/причина, Idempotency-Key |
| POST `/batches/{id}/previews` | expectedRevision, 1–100 уникальных recordIds; серверный preview |
| POST `/batches/{id}/confirmations` | previewId, previewHash, Idempotency-Key; атомарный receipt |
| GET `/batches/{id}/confirmations/{confirmationId}` | Сохранённые действия и event/provenance IDs |

Все mutations требуют session, CSRF и exact Origin. OpenAPI определяет
закрытые enums, integer bounds, UUID/даты, discriminated actions, responses
и error codes; TypeScript-типы генерируются. UI получает eligibility,
блокировки, candidate comparisons и денежные итоги от сервера. Ограничения
snapshot SPEC-0016 сохраняются, включая 16 MiB и лимиты rows/cells/candidates.
Перед разбором проверяется body limit; preview и создание batch синхронны.
Preview с blockers возвращает видимый результат проверки, но не может
подтверждаться; ошибки payload не создают частичный batch.

## Данные, миграция и rollback

Новая миграция добавляет batches, source records, decisions/audit, candidates,
previews, confirmations и provenance. Все таблицы имеют owner_id, FORCE RLS,
ограниченные grants; составные FK изолируют batch/record/event/actor связи.
Общие категории используются только для чтения. Snapshot хранится приватно
в PostgreSQL; документы по внешним ссылкам не скачиваются. Размеры и числа
записей проверяются до вставки; настоящие данные не попадают в Git или логи.
Новая миграция допускает source=import; финансовые значения не backfill-ятся.

Rollback до первой публикации удаляет только новые таблицы и расширение enum.
После публикации down migration должна отказать при наличии import events,
чтобы не стирать provenance или ledger. Отключение UI/routes сохраняет
подтверждённые события; финансовый rollback требует отдельной процедуры,
а не удаления source rows. Backup включает snapshot, audit и provenance.

## UI

Раздел «Импорт»: выбор файла, явное сообщение «Сохранён черновик», список
batch и продолжение проверки. На телефоне строки представлены карточками;
фильтры выделяют нерешённые, конфликты и неподдерживаемые данные. Детали
показывают исходные значения/координаты, нормализацию, сравнение кандидатов,
форму исправления и историю. Пользователь выбирает существующие счета и
категории; aliases не создают сущности самостоятельно.

Кнопка «Предварительный просмотр» открывает серверный результат: выбранные
строки, новые события, связи, исключения, предупреждения и изменения по
счетам/валютам. Отдельная кнопка «Подтвердить импорт N операций» доступна
только без blockers; N означает количество новых событий, рядом показано
число связей/исключений. Для группы без новых событий — «Подтвердить решения».
Сохранение черновика и Enter в форме не запускают confirmation. Истёкший или
устаревший preview требует обновления. Результат содержит ссылки на созданные
и связанные события; непринятые строки остаются черновиками. Основные touch
targets ≥44 px, видимый focus, keyboard navigation и состояния loading/error.

## Ошибки и негативные сценарии

400 malformed/version/action; 401 без session; 403 CSRF/Origin; 404 для чужих
или отсутствующих объектов без раскрытия владельца; 413 body too large;
422 invalid row/refs/unsupported semantics; 409 revision_conflict,
idempotency_conflict, preview_expired, preview_changed, already_published.
Частичный результат запрещён. Request errors не содержат raw snapshot,
банковских реквизитов или dump SQL. На session expiry draft сохраняется,
confirmation автоматически не повторяется после нового входа.

## Критерии приёмки

- [ ] AC1: Snapshot сохраняется целиком; все records имеют raw/provenance;
  перезапуск API/браузера сохраняет решения; ledger не меняется до confirmation.
- [ ] AC2: Тот же snapshot не создаёт второй batch; новый hash не обходит
  проверку опубликованных свидетельств; нет автоматической дедупликации.
- [ ] AC3: Правки имеют audit и revision; циклы/цепочки duplicate запрещены;
  новые исходные данные не перезаписывают прошлые решения.
- [ ] AC4: RLS/FK, session, CSRF/Origin защищают все маршруты и новые таблицы;
  второй owner/admin не читает и не связывает чужие records/events.
- [ ] AC5: Preview точно показывает действия и изменения по счетам/валютам;
  устаревший/истёкший preview и новые совпадения блокируют первую confirmation.
- [ ] AC6: Confirmation атомарна; concurrent/retry/lost-response не создают
  повторных postings/provenance; invalid row откатывает всю выбранную группу.
- [ ] AC7: Пять существующих kinds сохраняют инварианты; link/duplicate/exclude
  не меняют баланс; date-only и unsupported debt/plan/HOLD не публикуются молча.
- [ ] AC8: UI продолжает черновик после reload, показывает исходники и конфликты,
  требует отдельное подтверждение и работает на узком экране/с клавиатуры.
- [ ] AC9: Contract/types, миграции и rollback проверены; make test, go vet,
  integration и git diff --check проходят; evidence связывает каждый AC с тестом.

## Исполняемые проверки

План реализации (файлы этих проверок ещё не существуют):

- `go test ./internal/imports/... ./internal/financialevents/...`:
  AC2/3/5/7, точные деньги/даты, blockers, cycles, expiry, revisions.
- `./scripts/test-imports.sh`: AC1–7/9 на PostgreSQL с ограниченной ролью:
  synthetic snapshot с основным/дополнительным блоком, сохранение/reload,
  два owners/admin, FK/RLS, invalid requests, пять kinds, новый snapshot,
  повтор/concurrent confirmation, lost response, ошибка в последней строке,
  изменения ledger между preview/confirm, up/down без/с публикацией.
- `cd web && npm test -- --run`: AC8, загрузка/редактирование/preview,
  отдельное подтверждение, retry/error/session expiry и server blockers.
- `make generate-api` и проверка отсутствия drift: AC9.
- `make test`, `go vet ./...`, `make test-integration`,
  `git diff --check`: общие gates и регрессии.
- Визуальная проверка импорта на 390 px и desktop, keyboard/focus: AC8.

Implementation evidence должен содержать точные команды, результаты,
implementation commit и post-merge gates. Проверка 191 реальной строки
может быть отдельным авторизованным dry run без публикации и без raw в evidence;
для CI используются только синтетические fixtures.

## Открытые решения

Принятые владельцем решения:

1. Включить публикацию пяти существующих kinds после отдельного preview;
   новые кредитные смыслы и неподдерживаемые domains оставить черновиками.
2. Date-only create требует явного уточнения времени; не расширять модель
   дат финансового ядра в этом work item.
3. Группа до 100 строк атомарна; незавершённый batch допускает несколько
   независимых подтверждений, каждый с новым preview.
4. Исходники/история/provenance сохраняются постоянно в приватной БД;
   удаление опубликованного импорта и retention API — отдельный scope.

Scope принят 2026-10-06; публикация реальных финансовых данных и production rollout не разрешены этой спецификацией.
