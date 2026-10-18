# SPEC-0055: банковские PDF разных форматов — AI-черновик и подтверждаемый импорт

- Статус: integrated (код в main и post-merge gates пройдены; полная model qualification и owner functional acceptance не завершены)
- Автор: Codex по поручению владельца
- Дата: 2026-10-18
- Acceptance owner: владелец Grace
- Acceptance record: 2026-10-19 владелец явно утвердил предложенный технический вариант: «да, утверждаю»; архитектурное разделение принято ранее в этом чате
- Согласованное направление: текстовый PDF одного дебетового счёта; 2026-10-18 владелец предоставил образец Т-Банка и потребовал учитывать различия банков через LLM или Jev
- Specification branch: `codex/spec-0055-bank-pdf-import`
- Implementation branch: `codex/bank-statement-pdf-import`
- Implementation commit: `8e37465` ([offline evidence](0055-evidence.md)); model qualification NOT_RUN
- Integration: PR #64, merge `b51b74f`, CI main `39167930999` PASS; production deploy `39168570754`, 2026-10-19 02:37:31 +05:00
- Statement AI: disabled / activation BLOCKED (billing review и cards smoke) по прямому поручению владельца 2026-10-19; [activation evidence](0055-activation-evidence.md)
- Baseline: main `a98e304` после PR #61; production `a89a855`
- Этапы roadmap: 2 «Источники/подтверждение», первый срез этапа 5 «Банковские PDF»
- Реализация: integrated/deployed manual path; отдельное Statement AI enable поручено 2026-10-19, activation BLOCKED (unknown billing, cards smoke). Полная qualification и owner functional acceptance ещё открыты.

## Проблема и пользовательский результат

Банки по-разному располагают даты, суммы, остатки и описания. Владелец должен
загрузить выписку, получить сохраняемый проверяемый черновик, увидеть источник
каждого значения, исправить предложения и подтвердить новые события или связи
с уже существующими. Новый банк не должен обязательно требовать отдельного
рукописного parser до первой проверки документа.

Предлагается общий AI-процесс извлечения из локального текстового слоя PDF.
LLM предлагает структуру документа и соответствие исходных фрагментов полям.
Grace проверяет ссылки на текст, нормализует деньги/даты, рассчитывает суммы,
ищет дубли и выполняет явно подтверждённые финансовые действия. Jev рассматривается
как опциональный селектор/верификатор узких решений, а не обязательный второй
провайдер каждого запроса.

Универсальность контракта не означает гарантированную поддержку любой выписки.
Непонятный формат, пропущенные строки и отсутствующие сведения остаются видимыми
проблемами. Первый реальный reference — образец Т-Банка; совместимость остальных
банков подтверждается отдельно. Финансовое ядро и ручной review работают без AI.

## Scope

1. Текстовые PDF-выписки/справки движения по одному дебетовому RUB-счёту за период.
   Разные расположения полей/таблиц преобразуются в одну типизированную модель.
   Банковские профили разрешены как проверенные оптимизации, но не являются
   обязательным allowlist банков или основанием автоматически доверять данным.
2. Приватный оригинал, локальное извлечение текста со страницами/span IDs,
   минимизация передаваемого содержимого и отдельный AI-extraction после consent.
3. LLM mapping/structure proposal с обязательными исходными span IDs; серверные
   grounding/schema/coverage проверки. Никаких model-generated финансовых итогов.
4. Сохраняемые источники и построчные решения, matching и preview/confirmation
   через существующий `internal/imports`. Явный выбор существующего owner-счёта.
5. Создание только существующих `purchase`, `income_received`, `internal_transfer`,
   `debt_payment`, `refund`, с явным подтверждением смысла и guards financial core.
   Неподдерживаемое финансирование сохраняется нерешённым/исключённым с причиной.
6. Раздельные результаты сверки оборотов, проверки имеющихся остатков и покрытия
   строк событиями. Отсутствие opening/closing не подменяется нулём или доступным
   остатком; импорт не изменяет opening balance и не делает balance adjustment.
7. Выбор владельца сохранить/удалить оригинал; source records/provenance остаются.
   Mobile-first UI в «Импортах», OpenAPI/types, budget/consent/privacy policy.

## Non-goals

OCR/сканы и raw-PDF/image upload провайдеру в v1; зашифрованные/повреждённые PDF;
несколько счетов/валют ledger, кредитные условия/грейс, HOLD как Posting,
выдача кредита/новые финансовые виды, автоматические подтверждения/платежи,
изменение существующих проводок/остатков, создание счетов/категорий из догадок.

LLM не получает tools, сетевой поиск, произвольный SQL или доступ к записи.
Jev vision не входит в v1. Автоматическое включение AI в production, пополнение
баланса и отправка предоставленного личного PDF для платного benchmark этим
документом не разрешаются. Реальная загрузка/подтверждение операций — действие
владельца через UI после принятия и реализации пакета.

Миграция legacy `loan_documents`/фото чеков, точная сверка исторического баланса
Grace на границах банковского периода и универсальная поддержка всех банков —
отдельные scope.

## Текущее поведение и доказательства

Основания: [архитектура](../architecture.md), [roadmap](../roadmap.md),
[SPEC-0017](0017-persistent-import-drafts.md), [SPEC-0027](0027-date-only-history.md),
[SPEC-0040](0040-account-reconciliation.md), [SPEC-0044](0044-import-review-at-scale.md),
[SPEC-0051](0051-credit-cards.md), [AI-архитектура](../ai-interaction-architecture.md).

| Путь | Факт baseline | Следствие |
| --- | --- | --- |
| `internal/imports/handler.go` | JSON/CSV, records, previews и confirmations под `/api/imports` | Расширить вход/evidence, сохранить единый процесс записи |
| `internal/importreview/types.go`, `internal/imports/types.go` | SourceRecord ориентирован на sheet/row, PDF source variant отсутствует | Не выдавать страницу PDF за spreadsheet sheet; нужен locator/ordinal |
| `internal/imports/repository.go` | Сохраняемые raw/source/decisions и candidates | Matching PDF должен также находить события другого source/kind по Posting |
| `internal/imports/preview.go` | Revision/hash/expiry, replay, повторный plan и `financialevents.CreateInTx(...,"import")` в tenant transaction | Сохранить атомарность, source=import и owner writer lock |
| `migrations/000009_persistent_import_drafts.up.sql` | Batches/records/audit/confirmations/provenance с RLS и owner FK | Не создавать второй ledger/confirmation mechanism |
| `migrations/000024_credit_cards.up.sql` | Общий `documents` пока только `credit_card_terms`, content обязателен | Расширить kind/retention, сохранив прежние ограничения условий карт |
| `internal/accounts/reconciliation.go` | Отдельная сверка меняет opening/создаёт adjustment после подтверждения | PDF-import этот writer не вызывает |
| `internal/assistant/llm_provider.go` | Server-side OpenRouter policy, цены, ограничения input/output, provider checks | Переиспользовать технический транспорт/бюджет, не новый неограниченный prompt API |
| `internal/receiptdrafts/handler.go` | Отдельный extraction attempt и сохранение результата | Полезный существующий pattern, но consent фото не покрывает банковскую выписку |

### Приватный образец — локальное read-only исследование

Предоставленная справка Т-Банка: 5 страниц, текстовый слой, не зашифрована.
Визуально изучены все страницы. Локальный probe обнаружил 96 начал банковских
строк (14/26/26/26/4 по страницам); суммы debit/credit совпали с напечатанными
оборотами. Две колонки сумм в этом образце совпадают, но это не общий инвариант.
Это исследовательский результат, не реализация parser и не AC12 PASS.

Есть дата/время операции и отдельная дата/время списания, переносы описаний,
повторные заголовки, строки без номера карты, разные форматы десятичной записи,
комиссии, переводы себе и операции «Кубышки». Timezone явно не указан.
Имеется **доступный остаток на дату справки**; начального и конечного
бухгалтерских остатков нет. Нельзя реконструировать их из доступного остатка
и оборотов и затем объявить источниковыми значениями.

Сам PDF, изображения, извлечённый текст, номера/ФИО/адреса, реальные строки и
денежные суммы в Git/CI не добавляются. Документ не отправлялся TypeSafe/OpenRouter
и не импортировался. Содержимое PDF рассматривается только как данные, включая
любые потенциальные инструкции внутри документа.

## Предметная модель и инварианты

### 1. Источник и grounding

- `Document(kind=bank_statement)` хранит owner/UUID, безопасное имя, SHA-256
  исходных байтов, размер/MIME/createdAt и приватный content. Имя не является путём.
- `StatementSource` связывает document, nullable batch, выбранный debit account,
  RUB, банк/замаскированный reference счёта, период, extraction status/revision,
  retention. Совпадение последних цифр не привязывает счёт автоматически.
- `ExtractionAttempt` хранит version локального extractor, hash текстового
  состояния, schema/prompt/model/provider/policy revisions, consent и budget
  reservation/usage, status/error и provenance предложения. Полный provider
  request/response с личными данными не попадает в logs/telemetry.
- После первой confirmation accountId неизменяем. До неё смена привязки повышает
  revision и инвалидирует preview. Нельзя перенести уже опубликованные события.
- Upload request hash — canonical params + SHA-256 PDF, не multipart boundary,
  временный путь или время запроса. Уникальность `(owner, bank_statement SHA-256)`
  возвращает существующий источник; иная привязка — `document_binding_conflict`.
  SHA файла не является ID операции; перегенерированный PDF проходит matching.
- Локальное извлечение создаёт неизменяемые page/span IDs и layout locators.
  `FieldEvidence` хранит field role, source span IDs, исходные фрагменты,
  первоначальную нормализацию и provenance proposal. LLM может выбирать/
  группировать эти spans; значение без существующего span не становится полем.
- Go копирует literal из указанного исходника и нормализует его; модель не
  передаёт доверенный amountMinor, calculated balance или разрешение на запись.
  Несуществующий/несовпадающий span, неверная роль валютной колонки и противоречивые
  связи дают issues/blockers. Роль поля подтверждается отдельно от равенства текста.
- Ручные правки сохраняются как решения owner с причиной/историей; source/evidence
  не переписываются. Origin `bank text`, интерпретация `model`, исправление `owner`
  различаются. Confidence не заменяет доказательство, полноту или подтверждение.

### 2. Даты, суммы и остатки

- Раздельно сохраняются operation local date/time и booking local date/time,
  их исходная точность и timezone/offset, если он известен. Отсутствие зоны не
  разрешает выбрать её по адресу владельца или банковскому бренду.
- Без подтверждённой timezone source local clock остаётся evidence; ledger
  проекция date-only имеет явную причину `timezone_unknown`. После подтверждения
  банковской зоны владелец может получить точный timestamp через revisioned review.
  Не подставлять полночь и не терять дату списания при выборе даты операции.
- `transactionAmount` и `accountAmount` с явными валютами — разные поля. Для
  Posting используется сумма в валюте выбранного RUB-счёта; исходная валютная
  сумма остаётся evidence. Отсутствующую accountAmount нельзя рассчитать по
  придуманному курсу. Отдельный FX writer/второй валютный счёт в v1 не создаётся.
- Деньги нормализует Go: точная decimal→integer minor units, без float,
  проверка знака/валюты/переполнения и wire bounds ±9 007 199 254 740 991.
  Сумма операции положительна, debit/credit отдельно; aggregates — checked arithmetic.
- Заголовочные значения типизированы: `openingBookBalance`, `closingBookBalance`,
  `availableBalanceObserved`, `debitTurnover`, `creditTurnover`, их даты/основания.
  Неизвлечённое/отсутствующее — null + reason, не ноль.
- `availableBalanceObserved` никогда не используется как closingBookBalance или
  подтверждение бухгалтерского остатка. Банк может прислать только обороты;
  отсутствие book balances само по себе не запрещает подтверждать проверенные строки.

### 3. AI-extraction, Jev и границы доверия

Предлагаемая последовательность:

```text
PDF → локальные текст/страницы/spans → минимизированное состояние
    → LLM: структура/группировка/роли полей + span IDs
    → Go: schema/grounding/amount/date/coverage/totals checks
    → сохраняемый review → candidates → preview → owner confirmation
    → financial writer + provenance в одной PostgreSQL transaction
```

- Основной предлагаемый adapter — существующий server-side OpenRouter transport
  с явно выбранными model/endpoint, schema/prompt revisions и минимальным входом.
  Конкретные model/provider приняты в [техническом контракте](0055-technical-contract.md):
  Gemini 2.5 Flash, google-vertex/eu. Live calls не выполнялись.
- Jev — кандидат для выбора между найденными span IDs (`Choice` с `none`) и
  точечной проверки утверждений о роли/группировке (`Noul`), либо routing в manual
  review. Он не обязан вызываться после каждого LLM ответа. Включение зависит от
  измеренного дополнительного качества/стоимости на наших fixtures и privacy policy.
- Опциональный Jev-only путь может выбирать значения/роли из полного набора
  кандидатов; пропущенный локальным finder кандидат он выбрать не сможет.
  Сначала сравниваются LLM-only, bounded Jev-selection и cascade на синтетике.
  Русскоязычность и банковские сокращения проверяются отдельно.
- Текущая TypeSafe документация описывает images/Box как экспериментальные;
  поэтому старое абсолютное утверждение «Jev не принимает изображения» не
  используется как текущий технический факт. Для v1 выбран только текстовый input.
- PDF text/имена/заметки не являются system instructions. Модель не получает
  tools, SQL, URL fetch или capability финансового writer. Prompt injection
  проверяется adversarial fixture; его обнаружение не заменяет guards Go/owner.
- Отключённый/недоступный AI не блокирует existing ledger/JSON/CSV/review.
  Загруженный PDF остаётся источником; владелец может вручную подготовить строки
  с привязкой к spans через explicit manual-draft endpoint. Нет скрытого
  переключения на другого провайдера.

### 4. Полнота и сверка

Три независимых результата: `turnoverCheck`, `bookBalanceCheck`, `rowCoverage`.
Для каждого — matched/mismatch/incomplete/not_provided с машинными причинами.

- Суммы debit/credit строк сравниваются с соответствующими явно присутствующими
  банковскими оборотами. Если обе book balances известны и имеют совместимые
  даты/определения, Go дополнительно проверяет `closing = opening + credits - debits`.
  Если book balances отсутствуют, bookBalanceCheck=not_provided.
- Совпавшие обороты не доказывают правильность группировки/смысла и не дают
  общего «счёт сверён». Подмена колонок, доступного остатка и двойной учёт двух
  колонок одной операции должны быть отдельными негативными fixtures.
- Coverage проверяется по всем страницам/табличным областям и денежным/date spans.
  Каждый потенциальный финансовый фрагмент либо связан со строкой/полем, либо
  явно помечен служебным с основанием. LLM не может молча выбросить страницу,
  строку, перенесённое описание или неопределённый фрагмент. Неполный coverage — blocker.
- Наличие candidate spans не гарантирует, что локальный finder обнаружил все
  строки. Для нового layout требуется явная проверка покрытия владельцем по
  страницам; до неё coverage=not_reviewed и confirmation запрещена. Даже после
  model confidence и арифметического совпадения это отдельное действие review.
- Missing totals дают incomplete/not_provided, не выдуманные значения. При
  достаточном grounding/coverage и owner review отдельные строки разрешено
  подтвердить с явным предупреждением «оборот/остаток не проверен». Присутствующий
  conflicting total или ошибка разбора дают blocker, а не автоматическое override.
- `rowCoverage` по источнику и финансовое покрытие через provenance различаются.
  Финансовое `represented` требует связи каждой проведённой строки с событием/
  канонической строкой. `exclude(reason)` не удаляет денежную строку, не исправляет
  source totals и оставляет financial coverage неполным.
- Исторический ledger balance comparison — `not_evaluated`: банковское проведение
  и дата события различаются, полнота истории/anchor не доказаны. Импорт не
  вызывает SPEC-0040 opening_set/adjustment и не меняет остаток автоматически.

### 5. Решения и подтверждение

1. Upload/extraction/edit/review/preview не создают Posting/Receipt/Settlement,
   debt snapshot или adjustment. Только отдельная owner confirmation пишет ledger.
2. Debit не означает purchase, credit не означает доход. «Кубышка», заёмное
   поступление, перевод себе/на брокерский счёт и погашение требуют проверки
   смысла и owner-ссылок. Заёмное поступление нельзя подтвердить как доход
   ради завершения batch; для неподдерживаемой семантики — defer/exclude с причиной.
3. `create` требует явного вида и matching Posting выбранного счёта: account,
   currency, amount и direction должны совпасть с проверенной строкой. Перевод
   между своими счетами одной валюты создаёт две атомарные проводки, refund
   требует существующую purchase; debt_payment не создаёт Settlement автоматически.
4. Matching ищет по Posting account/direction/currency/amount и обеим датам,
   включая events другого source/kind и покупки с Receipt. Предлагаемый диапазон
   ±3 дня не заменяет расширенный поиск/ручной выбор. Отсутствие кандидата не
   доказывает уникальность; банковский reference — дополнительное свидетельство.
5. `link_existing` добавляет только provenance: matching posting обязателен,
   отличающиеся даты/описания видны и требуют причины, существующие Posting/Receipt
   не меняются. Вторая сторона своего перевода связывается с тем же event.
6. Candidates требуют решения: link/duplicate или отдельная операция с причиной.
   Две одинаковые реальные покупки допустимы; автоматического слияния нет.
   Duplicate внутри batch без цепочек/циклов; между batches — link к event.
7. Сохраняются existing preview contract: до 100 строк, 15 минут, revision/hash,
   owner/account/document/coverage/retention revisions, server-computed changes.
   В confirmation повторно проверяются plan, candidates, blockers и owner-ссылки
   под общей writer lock. Нельзя подменить их model confidence.
8. Ledger/provenance/receipt/status меняются атомарно; ошибка одной строки
   откатывает всю группу. Replay проверяется до expiry/revision и возвращает
   прежний receipt; иной payload с ключом — 409. Между preview/confirm новая
   ручная запись или импорт может дать новый кандидат и потребовать новый review.

### 6. Consent, бюджет и приватность

- Перед отдельной extraction UI показывает provider/model, цель, состав
  передаваемых fragments и стоимость верхней оценки. Согласие на фото чека/
  аналитику или загрузка PDF в Grace не равны согласию на отправку выписки.
- По умолчанию наружу передаётся минимизированный текст: без ФИО/адреса, полного
  номера счёта/договора, телефонов и лишних полей. Стабильные placeholders внутри
  попытки сохраняют связь строк. Описания merchant всё равно могут быть личными;
  UI называет эти остаточные данные, не обещает анонимность.
- Raw PDF/страницы-изображения в v1 не отправляются. Нет server tools/web search,
  plugin telemetry или передачи whole ledger. Document hash/state hash и scope
  согласия связываются с attempt; изменение данных/provider/purpose требует
  нового согласия. Jev как второй провайдер требует отдельного разрешённого scope.
- Сохраняются действующие OpenRouter policy: pinned provider/model, ZDR endpoint,
  data_collection=deny, без silent fallback. Для TypeSafe отдельно проверяются
  фактические условия аккаунта; нельзя переносить OpenRouter guarantees на Jev.
- Используется существующий общий бюджет/usage ledger, без нового лимита поверх
  прежнего разрешения и без автопополнений. Цена/доступный остаток должны быть
  проверены; резервируется весь верхний bound extraction до сетевого вызова.
- Страница/чанки не получают каждый полный бюджет документа. Сумма входных/
  выходных tokens, число calls и цена ограничены на attempt/document/owner.
  Короткий ответ, malformed JSON и partial extraction считаются расходом, но не
  успешным импортом. Неизвестная стоимость/потерянный ответ → usage_requires_review,
  без бесплатного по предположению автоматического retry.
- Сеть не выполняется внутри длительной financial SQL transaction. Reservation,
  attempt claim и сохранение proposal защищены от повторов/конкуренции; устаревший
  ответ не переписывает owner edits. Recovery после disconnect/crash не удваивает
  вызов с неопределённым исходом и не создаёт финансовое событие.

### 7. Retention оригинала

Предлагается default `delete_after_confirmation`, `keep` выбирается явно.
PDF в review доступен 7 дней с upload; дата видна. Удаление default выполняется
после завершения всех строк, не после первой группы из 100. Явное удаление
доступно раньше; `keep` снимает TTL только по действию владельца.

При удалении оригинала остаются hash, metadata, source values, минимальные
field fragments, решения, provenance и tombstone. Download → 410. Полные page text,
сырой model response и временное extraction состояние удаляются вместе с
оригиналом; их нельзя незаметно сохранить как вторую полную копию PDF.
Пока нужен consent preview/manual review, текст хранится приватно и ограниченно.

Extraction in-flight привязана к source revision: delete/expiry отменяет
дальнейшие вызовы, а ответ устаревшей попытки не материализует новый draft.
Один очищающий writer с attempt locks исключает воскрешение удалённых байтов.
Ранее опубликованные события/история неизменяемы. Логическое удаление из активной
БД не удаляет PDF из прежних backup до истечения их принятой retention.

## API и OpenAPI

API под `/api/imports` реализован в implementation branch;
[offline evidence и оставшиеся qualification gates](0055-evidence.md).

| Метод/путь | Контракт |
| --- | --- |
| POST `/statements/pdf` | Multipart PDF + accountId; Key; 201 `{statement,document,batchId:null}` / 200 upload replay. Только локальная bounded обработка, без provider call |
| GET `/statements/{id}` | Extraction/retention/revision, metadata, bank fields/evidence, nullable batchId, проверки и проблемы |
| POST `/statements/{id}/extraction-previews` | expectedRevision; redacted input/fields/providers, state hash, token/cost bound; без сетевого AI-вызова |
| POST `/statements/{id}/extractions` | expectedRevision, previewId/hash, consentId, Key; local/AI mode из policy; persisted attempt/result и batchId только после validation |
| POST `/statements/{id}/manual-drafts` | expectedRevision, owner grouping/field-role mapping по source span IDs, Key; validated batch без provider call; те же grounding/coverage/confirmation guards |
| PATCH `/statements/{id}` | expectedRevision, подтверждение account/field roles/page coverage/retention, причина; версии инвалидируют preview |
| GET `/statements/{id}/original` | Owner-only attachment PDF, no-store/nosniff, безопасное имя; 410 если удалён |
| DELETE `/statements/{id}/original` | expectedRevision + Key; отмена новых extractions, очистка/tombstone/audit, идемпотентный replay |
| GET `/batches/{id}/statement/reconciliation` | Turnover/book balance/source coverage/financial representation раздельно; ledger balance not_evaluated |
| Existing records/edits/previews/confirmations | PDF source variant/evidence и document blockers; единый financial confirmation |

Новые types: StatementSource, StatementExtractionPreview/Attempt/Proposal,
StatementFieldEvidence, StatementReconciliation и retention. Source variant
`spreadsheet`/`bank_statement_pdf` дополняет прежние DTO без фиктивного `sheet=PDF`;
старые JSON/CSV fields не теряются. Пагинация PDF по ordinal, не SQL cast отсутствующего
sheet row; month-review/старые filters имеют явную совместимость.

Предлагаемый upload: 10 MiB PDF / 11 MiB multipart, 50 страниц, 5 000 строк,
8 MiB extracted text, 50 000 candidate links; confirmation до 100 строк.
Локальный extractor: 30 s, 512 MiB, один process на инстанс, private temp и cleanup,
без сети/секретов/исполнения JS/actions/attachments/команд из PDF. AI input/output
bound и время полного attempt выбираются по модели/образцу в D3; старый provider
имеет собственные более узкие token bounds, их нельзя обойти большим PDF.
Превышение не обрезает документ молча. `429 extraction_busy` с Retry-After.

Все routes owner scoped, session, CSRF/exact Origin на mutations, RLS/compound FK,
Idempotency-Key для изменяющих операций. Admin не получает чужой документ.
Клиентские DTO генерируются из OpenAPI; финансовые rules/totals не в React.

## Ошибки и негативные сценарии

- 400 invalid_request; 413 statement_limit_exceeded; 422 invalid_pdf/encrypted_pdf/
  no_text_layer/unsupported_document_type; 429 extraction_busy — без частичного
  ledger. Неизвестный банк сам по себе не unsupported, если общий контур разобрал
  документ и прошёл evidence/coverage/owner review.
- 422 ungrounded_field/ambiguous_amount_role/incomplete_coverage/totals_mismatch:
  proposal/draft виден с blockers; нельзя принять неполный результат как успех.
  Missing book balances → not_provided, не totals_mismatch.
- 409 document_binding_conflict/revision_conflict/preview_changed/idempotency_conflict;
  404 чужой источник; 410 original_deleted. Результат upload/extraction не скрывается
  за ошибкой: успешный persisted draft возвращает resource с issues, confirmation
  с blockers — 422.
- ai_disabled/consent_required/provider_unavailable/provider_protocol/provider_timeout,
  price_unknown/budget_exhausted/usage_requires_review: сохраняемый понятный status,
  manual fallback; без silent provider/model change или повторного платного вызова.
- Equal turnover с неверной датой, переставленные amount columns, доступный остаток
  как closing, duplicate row из перекрытия чанков и потеря страницы — отдельные
  негативные cases. Schema-valid response и высокая confidence не дают acceptance.

## Данные, миграция и rollback

Аддитивная миграция после существующей 000026 (номер фиксируется при реализации):

- `documents.kind=bank_statement`, nullable content только для удалённого оригинала
  этого kind. Существующий `credit_card_terms` сохраняет обязательные байты/размер
  и неизменяемость; legacy loan documents не меняются.
- `statement_sources`, extraction attempts/previews/consent/audit и field evidence;
  отдельные source values и versioned owner decisions. Nullable batchId до
  validated extraction/materialization. Успешное создание batch/report/records и
  привязка source выполняются атомарно, повтор не создаёт вторую группу.
  Manual-draft материализует тот же batch через span mappings, не принимает
  доверенный model/client amountMinor и не создаёт параллельный batch. После
  публикации строк extraction/manual regroup существующего source запрещены;
  следующие owner edits используют прежний revisioned record review.
- Immutable source/evidence, explicit source variant, page/span/ordinal и nullable
  header balances; старые snapshots не переписываются. Constraints/FORCE RLS,
  compound owner FK и точечные grants; cleanup не получает право стирать card terms.
- AI/extraction orchestration зависит от financial/import capabilities, не наоборот.
  `internal/imports` preview/confirm не вызывает модель. Технический provider
  переиспользуется через interface/composition, не через зависимость финансового
  ядра от `assistant`. Глобальные controller/service/repository слои не вводятся.
- Down отказывается при любых новых sources/evidence/attempts/confirmations и
  tombstones. Понижение до приложения, не понимающего PDF batches, не объявляется
  безопасным. Проверяются fresh migration и backup→restore с новой историей;
  удаление оригинала нельзя представить обратимым down.

Rollout требует отдельного решения о provider/privacy/budget и production release.
Удаление/изменение реальных финансовых данных ради migration тестов запрещено.

## UI

«Импорты» → «PDF-выписка» → свой debit account → upload → локальный preview
источника → «Распознать с AI» с видимыми redacted данными/provider/cost и отдельным
consent → сохраняемый draft → проверка страниц/полей/строк → matching → financial
preview → отдельное подтверждение владельцем → receipt.

Рядом с предложением видны literal source fragments/страница, model provenance,
issues, ручная правка и её причина. Source text не исполняется и не рендерится
как HTML. После reload всё продолжается; AI unavailable не удаляет draft.

UI отдельно показывает проверенные обороты, непроверенные/отсутствующие book
balances, доступный остаток как отдельное наблюдение, полноту source coverage и
связи строк. Не показывает единый зелёный «Счёт сверён». Loan/own-transfer hints
не записываются автоматически; пропуск неподдерживаемой строки остаётся заметным.

375 px, controls ≥44 px, keyboard focus, отсутствие overflow, FSD, hooks/cache
invalidation только после успешной mutation. Сервер возвращает eligibility/
blockers/totals; исходные суммы и совпавшие две колонки не суммируются дважды.

## Критерии приёмки

| AC | Требуемое исполняемое доказательство |
| --- | --- |
| AC1 Heterogeneous layout/grounding | Реальный reference + минимум два структурно отличающихся синтетических layout. Page/span IDs для каждого поля, переносы/повтор заголовка, unsupported text; fabricated span/число отклонены |
| AC2 No pre-confirm write | Upload/extraction/edit/reload/preview не меняют events/postings/receipts/settlements/остатки; attempts/draft/decisions сохраняются |
| AC3 Missing/typed balances | Справка без opening/closing, но с available balance и оборотами: available не становится closing; суммы minor units точны. Отдельно полный balance fixture, mismatch 1 minor, разные валютные колонки, overflow |
| AC4 One fact/multiple sources | Existing purchase с Receipt, другой/перегенерированный PDF и повторная сторона transfer → прежние events/Posting/Receipt, новые provenance, без двойного расхода |
| AC5 Financial meaning | Credit/«Кубышка» не становятся доходом автоматически; loans/own transfers/fees видны; неверные posting amount/direction/account/currency link отклоняются; новый type не придумывается |
| AC6 Atomicity/replay | Два confirms/import/manual writer, response loss/expiry, one-row SQL failure, stale extraction response после owner edit → один результат/409, atomic rollback и неизменный source |
| AC7 Isolation | Два owner/admin, HTTP и прямой SQL/RLS, original download/delete, evidence, attempts/consent/budget и составные FK; CSRF/Origin |
| AC8 Retention | Keep/default delete, partial confirmation, expiry/inflight cancel/cleanup retry: 410, fragments/provenance остаются, full text/model payload убраны, другие documents не меняются |
| AC9 Provider and cost | Fake provider через настоящий orchestration: malformed/partial/timeout/unknown cost, reservation/retry limits, consent scope, redaction, stale policy. Нулевой provider call до consent; manual path при отключённом AI |
| AC10 Coverage/adversarial UX | Потеря страницы/строки, duplicate chunk overlap, одинаковые totals при неверной группировке, prompt injection, owner page review. OpenAPI/types, FSD, 375 px/keyboard/console, сохранение ввода при 409 |
| AC11 Regression/storage | JSON/CSV/date-only/month-review/все финансовые suites; fresh migration, restore новой схемы и отказ down с данными; make test/integration/production/vet/diff-check |
| AC12 Bounded model evaluation | Синтетические/обезличенные fixtures: LLM-only/Jev-selection/cascade, field/row/column correctness, omissions, grounding, cost/latency и abstentions. Приватный reference read-only отдельно; без разрешения нет real-data API calls |

Ни один implementation AC пока не выполнен. Локальное исследование образца не
является реализацией, quality benchmark моделей или owner acceptance.

## Исполняемые проверки

Draft: обязательные sections, ссылки, согласованность scope, `git diff --check`,
изменения только docs. Baseline `make test` на `a98e304` прошёл ранее в этом чате
(327 Web-тестов); код продукта с того времени не менялся. Он не доказывает PDF/AI.

После принятия — isolated `scripts/test-bank-statements.sh` в integration target,
настоящий PDF→local spans→fake provider→validated draft→review→confirm→SQL guards;
нельзя заменить все проверки готовым JSON и объявить extraction проверенным.
Pure unit — decimal/date/role/grounding/coverage; API/PostgreSQL — AC2–AC9/AC11;
web/browser — AC10; отдельный opt-in bounded model run — AC12.

Общие gates: `make test`, `make test-integration`, `make test-production`,
`go vet ./cmd/... ./internal/...`, `git diff --check`, OpenAPI generation без diff,
backup/restore и post-merge repeat. Платные API и реальные финансовые события
не нужны обычным offline CI suites. Raw sample/PII не попадают в Git.

## Принятые решения и остающаяся проверка

Владелец 2026-10-19 утвердил технический вариант D1–D4. Локальная qualification
и решения D2/D3 вынесены в [технический контракт](0055-technical-contract.md):
Poppler Alpine проверен на reference, публичные metadata модели сверены,
старый generic provider tag не проходит verifier. Это не model quality PASS.


| Решение | Предложение | Осталось |
| --- | --- | --- |
| D1 Поддерживаемый результат | Разные layout текстовых PDF одного debit RUB-счёта; общий AI/schema, Т-Банк как первый реальный reference; отсутствие book balances допустимо | Образец изучен локально. Подготовить синтетические layout/ground truth, критерии coverage и явный approved scope; универсальная поддержка не обещается |
| D2 AI/локальный extractor | Local text+spans, LLM structure proposal; Jev выборочно после сравнения, без обязательного cascade; reuse server OpenRouter policy/transport | Выбрать pinned extractor/model/provider по проверяемым privacy/price/quality. Политики TypeSafe проверяются отдельно. Датированное уточнение ADR-0002 сохраняет историческое решение и не включает vision автоматически |
| D3 Limits/budget | 10 MiB/50 pages/5 000 rows/8 MiB text/50 000 candidates; local 30 s/512 MiB/1 process; AI caps на весь attempt в рамках оставшегося общего бюджета | Chunk/token/call/cost/timeout policy и VPS resource bounds проверить на fixtures; не копировать cookbook thresholds. Проверить реальный остаток/endpoint без скрытых платных probes |
| D4 Retention/сверка | Default delete_after_confirmation, review TTL 7 дней, keep явно; turnover/book balance/coverage раздельно; ledger balance not_evaluated | Явное принятие этих defaults и критериев manual review; признать retention старых backups |

Следующий шаг — интеграция accepted specification, затем offline implementation
и синтетические fixtures в отдельном worktree. Model quality/пределы/cancellation
подтверждаются реализационными AC. Принятие не включает AI в production и не
разрешает передачу личного reference без document-specific consent.

## Внешние первичные источники — проверено 2026-10-18

- [TypeSafe: выбор исходных значений из кандидатов](https://docs.typesafe.ai/cookbooks/pre_parsed_value_extraction_cookbook): значения копирует/нормализует код; полнота кандидатов — отдельное условие.
- [TypeSafe: extraction cascade](https://docs.typesafe.ai/cookbooks/sde_cascade): generative extraction и узкая verification — разные этапы; демонстрационные thresholds не являются нашей acceptance policy.
- [TypeSafe: images](https://docs.typesafe.ai/images): текущие image/Box capabilities экспериментальные, не основание вводить их в production v1.
- [TypeSafe: models](https://docs.typesafe.ai/models): versioned IDs, ограничения и данные требуют проверки при выборе provider; alias не фиксирует поведение.
