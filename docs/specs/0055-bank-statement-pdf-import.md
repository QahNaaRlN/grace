# SPEC-0055: PDF-выписка дебетового счёта — источник, черновик и подтверждаемый импорт

- Статус: draft
- Автор: Codex по поручению владельца
- Дата: 2026-10-18
- Acceptance owner: владелец Grace
- Acceptance record: отсутствует; поручение подготовить следующий пакет не является принятием этого документа
- Согласованное направление: текстовая PDF-выписка по одному дебетовому счёту за период; ответ владельца 2026-10-18 «следуем твоей рекомендации»
- Specification branch: `codex/spec-0055-bank-pdf-import`
- Implementation branch: отсутствует
- Implementation commit: отсутствует
- Baseline: `a98e304` (main после PR #61); production `a89a855`
- Этапы roadmap: 2 «Подтверждаемый импорт и происхождение данных», первый срез этапа 5 «Банковские PDF»
- Готовность к реализации: NOT_READY — требуется закрыть решения D1–D3 в конце документа и явно принять итоговый scope

## Проблема и пользовательский результат

Владелец загружает PDF-выписку одного своего дебетового счёта, видит банковские
строки и основания извлечённых значений, продолжает проверку после перезагрузки,
сопоставляет строки с уже записанными операциями и подтверждает только нужные
действия. Повторная выписка, покупка с чеком и перевод не создают вторых расходов.

Результат первого среза — один проверенный профиль конкретного банка/формата,
а не обещание разобрать произвольный PDF. Текстовый слой извлекается локально;
финансовые суммы и проверки вычисляет Go. Приложение работает без LLM.

## Scope

1. Один выбранный банковский профиль с зафиксированными признаками формата и
   версией parser. Текстовый PDF, один дебетовый счёт, одна валюта RUB, конечный
   период и проведённые операции. Профиль должен содержать начальный/конечный
   остаток, обороты за период и даты банковского проведения строк.
2. Загрузка исходника, приватное хранение, локальное извлечение текста с
   координатами страниц/фрагментов, проверка покрытия таблицы и контрольных сумм.
3. Основа источников для банковских выписок: metadata документа, неизменяемые
   source records, значения и их evidence, история ручных решений. Переиспользуется
   общий `documents` и существующий import review/preview/confirmation.
4. Сохраняемый ImportBatch формата `bank_statement_pdf`, выбранный владельцем
   существующий счёт `debit`, построчная проверка и matching. Создание существующих
   пяти видов событий (`purchase`, `income_received`, `internal_transfer`,
   `debt_payment`, `refund`) только после явного решения владельца и финансовой
   валидации. По знаку суммы финансовый смысл автоматически не выбирается.
5. Проверка арифметики документа и покрытия строк финансовыми событиями —
   раздельные серверные результаты. Сверка выписки не устанавливает остаток счёта.
6. Выбор владельца: сохранить оригинал PDF или удалить его после подтверждения;
   удаление байтов не удаляет исходные значения, решения и provenance.
7. Mobile-first путь в существующих «Импортах», OpenAPI и сгенерированные TS-типы.

## Non-goals

OCR/сканы, зашифрованные или повреждённые PDF, другие банки/версии без fixtures,
мультивалютные/кредитные выписки, HOLD как денежный факт, извлечение кредитных
условий, тарифов/грейса, выдача кредита, новые финансовые виды, автоматические
платежи/подтверждения, изменение существующих проводок и остатков, автоматическое
создание счетов/категорий, LLM/Jev и передача документа внешнему провайдеру.

Договоры/условия карт и legacy `loan_documents` не переносятся в новую модель.
Фото чеков сохраняют свой жизненный цикл. Восстановление старых реальных данных,
самостоятельная загрузка выписки владельца и production rollout не входят в
implementation этого draft. Полный исторический баланс Grace на конец периода
не объявляется сверенным в этом срезе.

## Текущее поведение и доказательства

Основание — main `a98e304`, [архитектурный контракт](../architecture.md),
[roadmap](../roadmap.md), [SPEC-0017](0017-persistent-import-drafts.md),
[SPEC-0027](0027-date-only-history.md), [SPEC-0040](0040-account-reconciliation.md),
[SPEC-0044](0044-import-review-at-scale.md), [SPEC-0051](0051-credit-cards.md).

| Путь | Наблюдаемое поведение | Следствие для пакета |
| --- | --- | --- |
| `internal/imports/handler.go` | `/api/imports/batches`, `/csv`, records, previews, confirmations | Расширяем вход и evidence; не строим второй процесс подтверждения |
| `internal/imports/types.go`, `internal/importreview/types.go` | SourceRecord привязан к sheet/row, batch format JSON/CSV; общая модель PDF-страниц отсутствует | Нужен явный PDF-locator; нельзя выдавать страницу за лист таблицы |
| `internal/imports/repository.go` | `uploadReport`, сохранение source, audit, поиск candidates | PDF проходит серверную нормализацию и тот же review; готовый клиентский report не доверенный |
| `internal/imports/preview.go` | Revision/hash/expiry, replay, повторный plan внутри tenant-транзакции, `financialevents.CreateInTx(...,"import")` | Сохраняем writer, атомарность, сериализацию и source=import |
| `migrations/000009_persistent_import_drafts.up.sql` | ImportBatch/records/decisions/previews/confirmations/provenance и owner FK/RLS | Нельзя удалять source после публикации; общая provenance уже существует |
| `migrations/000024_credit_cards.up.sql` | `documents` пока принимает только `credit_card_terms`, content обязателен | Новый kind и управляемое удаление байтов должны сохранить ограничения прежнего kind |
| `migrations/000018_commitments.up.sql` | `loan_documents` отдельно хранит версии PDF | Миграция этих оригиналов не входит в scope |
| `internal/accounts/reconciliation.go` | Отдельный подтверждаемый процесс сверки может менять opening balance/создавать adjustment | PDF-import его не вызывает; остатки выписки остаются свидетельствами |

Поддержка банка/разбора PDF в текущем продукте отсутствует. Образец и банк ещё
не предоставлены; fixture compatibility и parser resource limits — UNVERIFIED.
Синтетический документ не доказывает совместимость с реальным банковским export.

## Предметная модель и инварианты

### Документ и профиль

- `Document(kind=bank_statement)`: owner, UUID, безопасное имя файла, SHA-256
  оригинальных байтов, исходный размер, MIME, время загрузки; content приватен.
  Имя не используется как путь. SHA-256 обозначает файл, не банковскую операцию.
- `StatementSource`: documentId, batchId, профиль/version, extractor/version,
  выбранный accountId, currency, идентификатор банка, замаскированный reference
  счёта, период и смысл его границ; статусы разбора/удаления, revision.
  Совпадение последних цифр не привязывает счёт автоматически.
- Владелец подтверждает соответствие документа выбранному своему счёту. Другой
  owner, другой тип счёта и несовпадение валюты отклоняются. После первой
  confirmation accountId неизменяем; исправление привязки не переносит события.
- Profile manifest фиксирует банк/тип export, проверенные признаки заголовка и
  колонок, порядок дат/сумм, правила переноса строки/повтора заголовка на новой
  странице, дату проведения/операции, границы периода и минимум fixtures.
  Другой формат возвращает `unsupported_statement_profile`, не подбирается LLM.
- Request hash upload — SHA-256 PDF + canonical profile/version/accountId,
  независимо от multipart boundary, временного пути и времени повторного запроса.
  Для bank_statement действует уникальность `(owner, SHA-256 исходных байтов)`.
- Один файл + один owner + профиль/version + accountId образуют ключ upload replay.
  Повтор тех же байтов/параметров возвращает существующий document/batch; другая
  привязка того же файла — явный `document_binding_conflict`, а не новый batch.
  Новый extractor/profile version не создаёт второй ledger-путь для документа.
  Переизвлечение существующего документа в v1 отсутствует: несовпадающая версия
  возвращает conflict. Будущая процедура re-review потребует отдельного scope,
  сохранения старого разбора и инвалидирования preview.

### Строки и происхождение полей

- `StatementRecord` расширяет существующий ImportRecord: стабильный UUID,
  documentId, source ordinal, banking status/direction, исходный reference операции
  (если есть), первоначальные нормализованные поля и issues.
- `FieldEvidence`: recordId или statement-header field, имя поля, исходный текст,
  значение до/после нормализации, page (с 1), позиции фрагментов в неизменяемом
  тексте страницы. Несколько фрагментов/страниц допустимы для одной строки.
  Locator включает extractor/version и hash страницы; его нельзя пересчитать
  из изменённой пользователем строки и представить исходным evidence.
- Хранятся минимальные фрагменты, нужные для проверки полей, а не полная копия
  извлечённого PDF-текста после удаления оригинала. Страницы для координат
  описывают первоначальное извлечение; отсутствие оригинала явно видно в UI.
- Ручная правка не изменяет source/evidence: сохраняются actor, revision,
  предыдущее/новое значение, причина, основание `owner`. Источник `bank` и решение
  `owner` показываются отдельно. Неподтверждённые/неизвлечённые поля — null/issue.
- Неизвестное не становится нулём. Нормализация денежных строк использует точный
  decimal→int64 minor units без float, с проверкой валюты, знака и переполнения;
  диапазон wire — ±9 007 199 254 740 991 minor units. Сумма операции положительна,
  банковское debit/credit хранится отдельно. Итоги считаются checked arithmetic.
- Банковские дата операции и дата проведения сохраняются раздельно. Время,
  которого нет в документе, не придумывается. При отсутствии даты операции
  используется date-only дата проведения с видимым `dateBasis=booking`;
  исходные даты всегда остаются evidence. Полночь/UTC автоматически не подставляются.
  Timezone/смысл границ периода фиксирует профиль по банковскому формату;
  timestamp создаётся только при однозначной зоне/offset. Сравнение дат кандидатов
  использует эту явную зону, сохраняя исходную дату события. Владелец может
  уточнить дату/время отдельным решением с причиной.
- Все распознанные непустые строки таблицы должны быть представлены source
  records либо явными служебными элементами профиля (например, повтор заголовка).
  Неизвестная строка, оборванная таблица, пропуск страницы, неоднозначный перенос
  создают blocker документа; их нельзя молча пропустить или исключить для зелёной суммы.

### Сверка и финансовые действия

1. Загруженный, извлечённый, отредактированный или previewed документ не создаёт
   Posting, Receipt, Settlement, снимок долга, adjustment и не меняет остаток.
2. Банковская арифметика: `closing = opening + credits - debits` по одной RUB
   валюте; суммы строк сверяются с обоими заявленными оборотами. Сравнение точное,
   без скрытой погрешности. Поля и их исходные фрагменты видны владельцу.
   Для выбранного v1 профиля все четыре контрольные суммы и период обязательны;
   missing/mismatch/overflow дают document-level blocker confirmation.
3. Состояние арифметики (`matched`, `mismatch`, `incomplete`) и покрытия строк
   (`not_reviewed`, `partial`, `represented`) независимы. Сверенные обороты PDF
   не означают верный баланс Grace. `represented` требует provenance каждой
   проведённой строки к событию/канонической строке; исключённая денежная строка
   остаётся видимой причиной неполноты, даже если batch больше не требует действий.
4. Ledger comparison балансов на границах периода в v1 возвращает
   `not_evaluated` с причиной: банковское проведение и дата события Grace могут
   различаться, неизвестен opening anchor/полнота истории. Источниковые остатки
   не подставляются в opening balance. Для изменения остатка остаётся отдельная
   принятая процедура SPEC-0040.
5. Автоматически извлекаются сумма, направление, даты, текст и bank reference,
   но не подтверждается вид события. Поступление не считается зарплатой, перевод
   на свой счёт не считается расходом, HOLD/план не становится Posting.
   Заёмное поступление и неподдерживаемая семантика остаются blockers.
6. `create` требует явного выбора одного из пяти существующих видов, валидных
   owner-ссылок и соответствия проводки банковской строке: accountId выбранного
   счёта, валюта, сумма и debit/credit должны совпасть. Refund связан с существующей
   purchase; internal_transfer требует второго своего счёта той же валюты и двух
   атомарных проводок. Платёж по обязательству не создаёт Settlement автоматически.
7. `link_existing` добавляет только provenance к существующему допустимому событию;
   требует matching posting выбранного счёта, направления, валюты и суммы.
   Отличающиеся даты/описания показаны и требуют причины. Kind/финансовый смысл
   подтверждает владелец. Нельзя привязать произвольный owner event только по UUID.
8. Matching ищет кандидатов среди текущего batch, прежних импортов и существующего
   ledger, включая покупки с Receipt и события с отличающимся source/kind.
   Базовый поиск по проводке счёта/направлению/валюте/сумме и датам не должен
   скрывать возможный дубль из-за неизвестного исходного kind. Причины и различия
   доступны в preview. Банковский reference — дополнительный признак, не глобальный ID.
   Соседние даты включаются в предлагаемый диапазон ±3 календарных дня; расширенный
   поиск и ручной выбор остаются доступны, отсутствие кандидата не доказывает уникальность.
9. Повтор другого PDF с теми же операциями создаёт новый источник и требует
   review: hash/имя не заменяют matching. Одна purchase из чека и выписки получает
   две provenance-связи; существующий Receipt и Posting не меняются. Вторая сторона
   перевода связывается с уже существующим internal_transfer, не создаёт ещё два Posting.
10. Для любого кандидата владелец выбирает связь/duplicate либо явно подтверждает
    отдельную операцию с причиной. Две одинаковые реальные покупки допустимы;
    автоматическое слияние/удаление запрещено. Duplicate внутри batch не имеет
    цепочек/циклов; межbatch источник связывается с уже опубликованным событием.
11. `exclude(reason)` сохраняет строку и сумму. Оно не исправляет арифметику
    исходника и не даёт `represented`. Нельзя исключением обойти blocker разбора.
12. Preview и confirmation сохраняют контракт SPEC-0017: до 100 строк, hash,
    15 минут, revision, server-computed changes, replay до expiry, идемпотентность,
    повторная проверка кандидатов и тех же document-level blockers внутри tenant
    transaction. Привязка документа, решения строк, retention choice и их версии
    входят в preview hash. Ledger/provenance/receipt/status пишутся атомарно.
13. Общая блокировка owner writers и уникальность provenance защищают от двух
    confirmations, параллельного импорта и ручной записи между preview/confirm.
    Новый кандидат или изменённый документ/счёт дают 409 и новый review. Повтор
    того же confirmation после потери ответа возвращает сохранённый receipt.

### Хранение оригинала

Предлагаемый v1: default `delete_after_confirmation`; выбор `keep` владелец
делает явно. Пока batch проверяется, PDF доступен 7 суток от upload; этот срок
показывается. Истечение удаляет только исходные байты, не source records/решения.
Завершать проверку без оригинала можно по сохранённым фрагментам с явным статусом
`original_deleted`; профиль не переизвлекается. `keep` снимает auto-delete только
после явного выбора владельца; скачивание/удаление требует его сессии.

`delete_after_confirmation` применяется после завершения всех строк batch,
а не после первой группы из 100. Явное удаление доступно раньше. Confirmation
сохраняет решение о retention в той же транзакции. Если байты в PostgreSQL,
они очищаются транзакционно; ошибка удаления откатывает соответствующее действие,
но не обещает повторную запись уже подтверждённых событий. Cleanup идемпотентен
и работает с теми же owner/RLS границами через ограниченный служебный путь.

Hash, имя, metadata, источниковые суммы/даты, минимальные фрагменты, решения и
provenance сохраняются. На download удалённого оригинала — `410 original_deleted`.
Это логическое удаление из активной БД: прежние database backups могут содержать
PDF до истечения принятой retention backup (см. production runbook). Нельзя
обещать немедленное стирание из всех старых копий; исходные финансовые значения
сохраняются независимо от PDF.

## API и OpenAPI

Ниже проект контракта, а не уже доступные endpoints. Реальный `api/openapi.yaml`
меняется только после принятия спецификации. Существующий prefix — `/api/imports`.

| Метод/путь | Запрос/ответ |
| --- | --- |
| POST `/statements/pdf` | Multipart PDF + accountId + profileId; Idempotency-Key; 201 новый `{document,batch,statement}` / 200 replay |
| GET `/batches/{batchId}/statement` | Metadata, account binding, profile/version, исходные totals с evidence, arithmetic/coverage, retention/revision; без PDF bytes |
| GET `/batches/{batchId}/statement/reconciliation` | Текущая серверная проверка документа и покрытия, blockers; ledger balance comparison `not_evaluated` |
| PATCH `/batches/{batchId}/statement` | expectedRevision, account binding до публикации, confirmation привязки/retention; обязательные Key/CSRF/Origin |
| GET `/batches/{batchId}/statement/original` | Authenticated attachment PDF или 410; no-store, nosniff, безопасный Content-Disposition; без публичного URL |
| DELETE `/batches/{batchId}/statement/original` | expectedRevision + Key; очистка content, tombstone/audit, 200 replay; без удаления batch/provenance |
| Existing records/edits/previews/confirmations | Тот же процесс; PDF source locator/evidence, unknown kind, проверка направления и document blockers дополняют ответы |

Новые типы: StatementSource, StatementFieldEvidence, StatementReconciliation,
StatementRetention и PDF source locator. `ImportSourceRecord` получает явный
variant источника (`spreadsheet`/`bank_statement_pdf`) с сохранением прежних
sheet/row для JSON/CSV; никакой фиктивной страницы `sheet=PDF` как SSOT.
Backward compatibility: старые JSON/CSV uploads/records, фильтры, сортировка и
month-review работают прежним образом; новые страницы читают variant безопасно.
Statement records используют ordinal для пагинации/сортировки, а не SQL cast
несуществующего spreadsheet row. Общие API enums/nullable поля и TS types
генерируются из OpenAPI, DTO вручную в Web не дублируются.

Все финансовые ресурсы owner-scoped: сессия, CSRF/Origin для mutations, RLS и
составные owner FK; admin не получает чужой источник. Audit не пишет PDF,
банковский номер, текст операций или значения секретов в app logs. Ошибки содержат
код, безопасное сообщение и разрешённые locator/field IDs, не весь исходный текст.

## Ошибки и негативные сценарии

### Лимиты и ошибки

Предлагаемые лимиты до принятия D3: PDF до 10 MiB (multipart envelope до 11 MiB),
до 50 страниц, 5 000 operation rows, 8 MiB извлечённого UTF-8 текста и 50 000
кандидатных связей. Existing confirmation — до 100 строк. Превышение не приводит
к молчаливому обрезанию источника.

Разбор синхронный с общим wall timeout 30 s и пределом памяти extractor 512 MiB;
одновременно на инстансе допускается один разбор, следующий получает 429
`extraction_busy` с Retry-After. Очередь/background jobs не добавляются;
дочерний процесс отменяется при timeout/disconnect. External links, JavaScript,
embedded files и actions из PDF не выполняются; extractor не получает сеть,
секреты и доступ к financial writer. Временные файлы private и убираются при
успехе/ошибке; host filesystem paths клиенту не доступны. Extractor не исполняет
команды из имени файла/содержимого. Parser failure до atomic persistence не
оставляет document/batch, которые выглядят успешно загруженными.

| Код/статус | Поведение |
| --- | --- |
| 400 invalid_request | Неверные параметры/ID/profileId; нет частичного batch |
| 413 statement_limit_exceeded | Body/pages/text/rows/candidates limit; no truncation |
| 422 invalid_pdf / encrypted_pdf / no_text_layer | Поломанный, защищённый или скан; OCR fallback отсутствует |
| 422 unsupported_statement_profile | Банк/версия/валюта/тип/несколько счетов не подходят |
| 422 statement_incomplete / totals_mismatch | Отказ confirmation. Успешно сохранённый upload возвращает 201/200 с draft/blockers; ответ 422 не скрывает созданный batch |
| 409 document_binding_conflict / revision_conflict / preview_changed | Новое review; без частичного ledger write |
| 409 idempotency_conflict | Тот же ключ с иным запросом |
| 404 not_found | Чужой/несуществующий документ или ссылка |
| 410 original_deleted | Байты удалены; provenance остаётся |
| 429 extraction_busy | Ограничение конкурентного разбора; retry не создаёт частичных данных |
| 503 extraction_unavailable / extraction_timeout | Безопасная ошибка, writer не вызван; повтор возможен |

`link_existing`/`create` с неверной суммой, направлением или семантикой получает
422. Переполнение/отсутствие суммы нельзя «исправить» приведением к нулю.
Ошибки extraction после распознавания таблицы не повышают draft в complete.

## Данные, миграция и rollback

Предлагается следующая аддитивная миграция (номер выбирается при реализации;
последняя существующая — 000026):

- `documents.kind` допускает `bank_statement`; content nullable только у этого
  kind после очистки оригинала. Для `credit_card_terms` сохраняются обязательные
  байты, размер/хеш и прежняя неизменяемость. Существующие download/API не меняются.
- Таблица `statement_sources` с document/batch/account/owner FK, профиль/version,
  исходные control totals/dates/evidence и retention/tombstone/revision.
- Таблица `statement_field_evidence` с owner/document/record связями и неизменяемыми
  fragments/locators. Документные header fields явно отличаются от record fields.
- Import records получают явный source variant и PDF ordinal/locator. Старые
  snapshots/records не переписываются; existing provenance/decisions/confirmations
  переиспользуются. Report hash учитывает вид источника и evidence revision.
- Все новые связи составные по owner; FORCE RLS, ограниченные grants, immutable
  metadata/source/evidence и published decision guards. Cleanup не получает
  общего права удалять документы условий карт или финансовую историю.
- Upload metadata/content + source + report + records сохраняются в одной tenant
  транзакции после успешной bounded extraction. Decode/failed parser не создаёт
  проводки; confirmation использует существующий writer transaction.

Backend группируется по capability: imports оркестрирует review/confirmation,
новый локальный parser относится к statement-import capability; общие document
хранилище/типы выделяются только для реально общего кода. Financial core не
зависит от extractor/imports/AI. Глобальные controller/service/repository слои
не вводятся.

Down разрешён только при отсутствии новых источников/evidence/confirmations и
`bank_statement` документов, включая tombstones. При наличии данных — явный
отказ без потери истории; предыдущая application version не считается безопасной
после создания PDF batches. Необратимое удаление байтов PDF не маскируется как
восстанавливаемая down-миграция. Перед rollout — verified backup/restore drill
новой схемы; rollback plan должен сохранить новые документы и опубликованные
связи. Автоматическое production-развёртывание этим draft не разрешается.

## UI

«Импорты» → «Выписка PDF» → выбор существующего дебетового счёта и поддерживаемого
профиля → upload → документ/период/контрольные суммы → список строк → решение →
preview новых движений и связей → отдельное подтверждение владельцем → receipt.
После reload сохраняются batch, решения, версия, исходные evidence и retention.

В списке видны проведённая дата, исходная дата операции (если есть), debit/credit,
сумма, описание, вид решения, статус кандидатов и номер страницы. Выбор строки
открывает исходные фрагменты и разницу с решением владельца. Банк/счёт/валюта
подтверждаются до первого preview. Неизвестная категория не придумывается;
явное сохранение без категории использует действующее разрешение reviewed imports.

На карточке три независимых результата: «Арифметика выписки», «Строки связаны»,
«Баланс Grace на границах периода не проверялся». Нельзя показывать единый зелёный
статус «Счёт сверён». Отдельно видны новые события, связи без новых Posting,
исключённые суммы и неполные/конфликтующие строки. Выбор retention и дата удаления
оригинала показаны до confirmation. На удалённом PDF — tombstone вместо битой ссылки.

375 px, controls не меньше 44 px, keyboard focus, читаемые суммы/исходные фрагменты,
нет горизонтальной прокрутки всего экрана. Существующие экраны таблиц/CSV и иерархия
FSD сохраняются. Финансовые eligibility/totals/blockers вычисляет сервер.

## Критерии приёмки

Все сценарии реализации используют синтетические данные; банковский образец
хранится вне Git, без публикации личных операций в CI/logs. Evidence явно отделяет
совместимость с предоставленным export от прохождения синтетических fixtures.

| AC | Исполняемое доказательство реализации |
| --- | --- |
| AC1 Профиль и происхождение | Тесты положительного fixture выбранного формата, многостраничной таблицы и переноса; каждый обязательный field/row имеет неизменяемый locator, raw fragment и parser version. Неизвестная строка не теряется |
| AC2 Нет записи до подтверждения | После upload/edit/preview/reload совпадают counts/hashes финансовых events/postings/receipts/settlements/остатки; сохраняются draft и решения |
| AC3 Арифметика и полнота | Fixture: opening 100 000 minor, credits 50 000, debits 30 000, closing 120 000; mismatch на 1 minor, пропуск строки/страницы, missing total и overflow блокируют confirmation; exclude не снимает blocker |
| AC4 Один факт, несколько источников | Existing purchase с Receipt + повторная PDF/перегенерированный файл → после owner link один event/Posting и неизменный Receipt, несколько provenance; identical real purchases допускаются только после отдельного решения |
| AC5 Финансовая семантика | Банковский credit не становится доходом без решения; собственный перевод имеет ровно две проводки, повтор второй стороны не добавляет проводок; неверная сумма/account/direction link отклоняются; refund/debt_payment используют прежние доменные guards |
| AC6 Конкуренция и replay | Два confirm/два imports/manual writer после preview → один результат или 409 с новым review; replay после потери ответа/expiry возвращает receipt; ошибка одной строки откатывает всю группу, provenance и statuses |
| AC7 Owner isolation | Два owner и admin: HTTP/SQL RLS, download/delete, account binding, evidence и compound FK не раскрывают/не связывают чужие ресурсы; CSRF/Origin проверены |
| AC8 Retention | keep/delete_after_confirmation, частичная confirmation, expiry/cleanup/retry и rollback SQL transaction; удалённый оригинал даёт 410, source/evidence/provenance/суммы остаются; credit_card_terms и loan_documents не меняются |
| AC9 Неподдерживаемые входы | Scan/encrypted/corrupt/другой профиль/мультивалюта, timeout, memory/body/pages/text/rows/candidates limits: безопасный отказ, нет truncation/частичного ledger/утечек и temp files; без внешних API |
| AC10 UX и API | OpenAPI/types, FSD hooks/cache invalidation только после успеха; 409 сохраняет ввод, reload продолжает review; browser 375 px, keyboard, console, отсутствие overflow; backend отдаёт totals/blockers |
| AC11 Регрессия | Все прежние JSON/CSV/source/date-only/month-review и финансовые suites проходят; свежая миграция, backup→restore новой схемы и отказ destructive down с данными проверены; make test, integration, production, vet, diff-check |
| AC12 Реальный формат | В приватной read-only проверке предоставленного export сходятся количество строк, страницы, исходные поля и control totals. Приватные значения не попадают в Git/evidence; результат отдельно от synthetic PASS и owner acceptance |

Ни один AC реализации сейчас не объявляется выполненным; этот пакет — draft.

## Исполняемые проверки

Для текущего draft:

- обязательные разделы template, локальные Markdown links, согласованность scope;
- `git diff --check`, отсутствие изменений вне docs и записанного acceptance;
- `make test` на baseline main — регрессионная проверка существующего продукта,
  не доказательство реализации PDF.

После принятия implementation добавляет bounded `scripts/test-bank-statements.sh`
в `make test-integration`: отдельный Compose project/порты/синтетическая БД,
настоящий PDF→extract→parse→persist→review→confirm путь и SQL guards.
Нельзя подменить parser готовым успешным JSON во всех интеграционных проверках.
Pure unit покрывают decimal/date/profile/locators и document arithmetic;
API/integration — AC2–AC9/AC11; web tests/browser — AC10; приватный fixture — AC12.

Общие команды: `make test`, `make test-integration`, `make test-production`,
`go vet ./cmd/... ./internal/...`, `git diff --check`; отдельно OpenAPI generation
без diff и backup/restore. Mandatory post-merge gates повторяются. Тесты не
обращаются к платным провайдерам, не импортируют реальные операции в production
и не объявляют owner acceptance на основании browser smoke агента.

## Открытые решения

| Решение | Предложение | Что требуется для закрытия |
| --- | --- | --- |
| D1 Банк и формат | Один текстовый export дебетового RUB-счёта с остатками/оборотами/проведёнными строками; рекомендованный класс согласован владельцем | Название банка и образец конкретного export, приватный разбор его структуры, профиль и синтетические fixtures. Обезличивание не должно ломать вёрстку/текстовый слой. До этого совместимость UNVERIFIED |
| D2 Локальный extractor | Без LLM/OCR/сети; page text+locators, pinned dependency/runtime, controlled process bounds | Выбрать инструмент по образцу D1, проверить лицензию, упаковку в API image, реальные limits/cancel и повторяемость locators. Это техническое решение принимается до application code |
| D3 Численные лимиты и retention | 10 MiB/50 страниц/5 000 строк/8 MiB текста/50 000 candidates; 30 s/512 MiB; review PDF 7 дней, default delete_after_confirmation, keep явно | Проверить на D1 и доступных ресурсах VPS, принять/скорректировать значения и default. Active DB deletion не обещает стирание из старых backups |
| D4 Граница сверки | В v1 — арифметика документа + покрытие событиями; точный historical ledger balance not_evaluated | Явное принятие scope; автоматический opening_set/adjustment и сравнение банковской booking date с event date исключены |

Следующий шаг после review — закрыть D1–D3 и принять SPEC-0055. Общее согласие с
направлением не повышает этот draft в accepted. До этого не меняются production
код, OpenAPI, схема БД, окружение, реальные источники и financial ledger.


## Проверка draft — 2026-10-18

- `make test` на main `a98e304` — exit 0: Go, 327 Web-тестов, ESLint,
  Steiger, Prettier, OpenAPI generation, TypeScript/Vite и Jev datasets v1/v2.
- Проверены обязательные разделы, локальные Markdown links и `git diff --check`.
- Diff пакета содержит только этот draft, реестр спецификаций и roadmap.
- AC1–AC12, совместимость банковского формата, extractor и performance/limits
  не выполнялись и не объявляются PASS. Production остаётся `a89a855`.
