# SPEC-0026: Ассистент в Grace и расходы по фото чека

- Статус: draft
- Автор: Codex
- Дата: 2026-10-08
- Acceptance owner: owner
- Acceptance record: отсутствует; владелец поручил проработку архитектуры,
  новую спеку и эксперимент после выпуска существующей версии.
- Implementation branch: не создана
- Implementation commit: отсутствует
- Budget authorization: эксперимент с API до $10; не разрешает превышение
  лимита, автоматическое продление или передачу неуказанных реальных данных.

## Проблема и результат

Владелец хочет встроенного в Grace ассистента, который объясняет расходы,
и ввод расхода по фотографии чека. Jev отвечает типизированными решениями,
а не генеративным финансовым анализом. Нужно добавить LLM, сохранив финансовую
истину в Go/PostgreSQL, явную неопределённость и пользовательские подтверждения.

Результат: вопрос о календарном месяце → объяснение с проверенными числами
и ссылками на факты; фото → проверяемый черновик → подтверждённая покупка
или детализация уже существующей. [Архитектура](../ai-interaction-architecture.md)
определяет роли ядра, Jev и LLM и является частью предложенного scope.

## Наблюдаемое текущее поведение

Grace имеет шесть read-only assistant tools и опциональный выключенный Jev
router. Генеративного диалога/vision, временных фото, истории разговоров и
платного API адаптера нет. Ручной Receipt детализирует существующую purchase
без нового Posting. Сохраняемые импорты и календарная история выпускаются
по SPEC-0017/0023/0027; неотсверенная книга не является подтверждённым ledger.
Незавершённый Gemini proof-of-concept SPEC-0025 не включается и не считается
реализацией этого scope.

## Scope и порядок эксперимента

| Этап | Реализуемый результат | Зависимость |
| --- | --- | --- |
| A: основание | Серверный OpenRouter adapter, capability/cost policy, mock provider, квоты и минимальный audit | Принятие этой спеки; ключ и подтверждённый доступный баланс |
| B: объяснение | Один календарный период, Go report + структурированный ответ LLM, источники и ограничения | A; сначала только synthetic cases |
| C: чтение чека | Фото внутри Grace, typed draft, ручная правка, Jev category suggestions | A; временное storage, synthetic изображения |
| D: подтверждение | Immutable preview, link-existing либо атомарная purchase+Receipt, идемпотентность и удаление фото | C; принятая ниже политика записи |
| E: live experiment | Сравнить 2–3 доступные модели; один разрешённый реальный анализ и чек после согласия | A–D gates; выбранные endpoint/policy и согласие владельца |

Не реализуются в эксперименте: монетизация, BYOK/OAuth, самостоятельные переводы,
платежи/советы о кредитных продуктах, банковские PDF, массовое автоподтверждение,
полная товарная база/сравнение магазинов и произвольный многозадачный агент.
Поддерживаются покупки RUB по одному чеку, до 50 позиций, один счёт оплаты.
Смешанная оплата/возврат/несколько валют остаются ручным путём или отдельной спекой.

## Модель и инварианты

FinancialEvent/Posting остаются источником финансовой истины. Все money values
точные int64 minor units с валютой; vision переносит исходные строки сумм,
которые парсит Go без float. Quantity — quantityMilli. Наименование чека,
бренд, упаковка и широкая группа хранятся раздельно, неизвестные поля null;
AI suggestion отделена от подтверждённого owner value.

ReceiptDraft содержит ownerId (только из сессии), revision, extraction status,
источник/версию модели, extracted fields/evidence text, warnings, выбранное
решение create/link и временный imageObjectId. Original photo не FinancialEvent.
Даты используют precision instant/date/unknown; timezone явно выбран человеком.
Неизвестное время никогда не становится полночью в API/БД. Для Receipt issuedAt
потребуется такая же nullable/date модель, как SPEC-0027 для FinancialEvent.

Conversation хранит отдельные user/assistant messages и ссылки на fact snapshot;
не меняет финансовую память. Чтение tools ограничено текущим owner. LLM не получает
SQL, ключи, ownerId или write tools. Proposed analysis response содержит claims
с fact IDs и value references; числа/проценты для UI подставляет Go. Unsupported
free-text claim не отображается как установленный факт. Jev может оценивать
смысл, но его вероятность не заменяет доказательство/подтверждение.

## Политика AI-originated записи — предмет принятия

1. AI только заполняет изменяемый черновик. Загрузка, extraction и правки
   не создают событий/проводок.
2. Человек выбирает существующий счёт, проверяет сумму/дату/позиции/категорию,
   решает create purchase или link existing. Неизвестные обязательные поля
   блокируют preview. Начальные остатки не создаются по догадке.
3. Go ищет кандидатов по сумме, валюте, календарному дню и известному магазину.
   Совпадения требуют явного решения, включая повторное фото того же чека.
   Hash фото/fiscal key только сигнал совпадения, не разрешение слияния.
4. Сервер создаёт preview с точными эффектами, revision, hash, expiry и owner;
   при link эффекта на баланс нет, Receipt равен покупке. Create даёт одну
   debit-проводку и один Receipt в одной транзакции.
5. Пользователь нажимает отдельную кнопку «Подтвердить расход» или «Добавить
   чек к покупке». Confirmation требует previewId/hash + Idempotency-Key,
   session, CSRF, Origin. Revision/ledger drift/expiry требуют нового preview.
6. Сервер повторно проверяет инварианты и записывает атомарно. Replay возвращает
   прежний результат. AI не может вызвать этот endpoint через tool executor.
7. После успешной записи фото ставится на удаление. Ошибка удаления сохраняет
   cleanup job и состояние «ожидает удаления», не ложный успех удаления.

## Оркестратор и бюджет

Go ограничивает один analysis turn тремя provider calls и четырьмя read-only
tool calls; не более 10 000 input/2 000 output tokens на call, request timeout
30 s и общий timeout 90 s. Receipt extraction — один vision call, повтор
только новым явным действием. Одновременно не более двух платных запросов
на owner. Модель/endpoint allowlist и maximum price задаёт сервер.

Бюджет эксперимента $10 включает обязательные комиссии и новые платные Jev
вызовы. Inference allowance вычисляется из реально приобретённых credits,
не номинального платежа. Upper-bound reservation хранится в БД атомарно;
учитываются текст, изображения, output/reasoning tokens и известные комиссии.
Missing/неясная цена либо невосстановленная usage останавливает дальнейшие
платные вызовы, пока reservation не сверена. Отмена после отправки не
предполагает бесплатный upstream request. Нет автопополнения и скрытого fallback.
Стоимость и usage — decimal strings либо integer USD micro-units, без float.

OpenRouter endpoint `https://openrouter.ai/api/v1/chat/completions` вызывается
сервером. Request policy для реальных данных: data_collection=deny, zdr=true,
allowlist endpoint/provider, require_parameters=true, без server tools/plugins.
Ответ включает фактически выбранную модель/provider и фактический usage/cost.
Возможность соответствующего маршрута проверяется до включения функции.

Перед реальным анализом/фото пользователь видит перечень отправляемых данных,
OpenRouter + выбранного upstream и цель. Согласие сохраняется для конкретного
scope/policy version, отзывается; изменение маршрута/состава требует нового
согласия. Согласие не является финансовым подтверждением. Первые проверки
проводятся только на синтетических fixture данных.

## API/OpenAPI — предложенный контракт

- `GET /api/assistant/capabilities`: enabled modes, выбранные model/provider,
  privacy policy version, доступный лимит без ключа.
- `POST /api/assistant/consents`: согласие на конкретный analysis/receipt scope;
  `DELETE` для отзыва. Не даёт доступ к данным другого owner.
- `POST /api/assistant/conversations`, `GET .../{id}`, `POST .../{id}/messages`:
  bounded read-only turn, структурированный ответ с factRefs, completeness,
  assumptions, cost и friendly provider errors. Old deterministic route сохраняется.
- `POST /api/receipt-drafts`: multipart JPEG/PNG/WebP до 8 MiB, одно изображение;
  content signature, размер, ориентация, удаление EXIF; не URL fetch.
- `POST /api/receipt-drafts/{id}/extract`, `GET/PATCH .../{id}`: explicit inference,
  owner-scoped revision и typed editable fields, unknowns и warnings.
- `POST .../{id}/previews`, `POST .../{id}/confirmations`: отдельные preview/confirm.
- `DELETE /api/receipt-drafts/{id}`: отмена и cleanup; опубликованный Receipt
  этим действием не удаляется.

Контракт меняется только после принятия, TypeScript генерируется из OpenAPI.
Ошибки: ai_disabled, consent_required, budget_exhausted, price_unknown,
provider_unavailable, unsupported_capability, extraction_incomplete,
totals_mismatch, candidate_review_required, stale_preview, image_expired.

## Данные, retention, миграция и rollback

Новые owner-scoped таблицы: conversations/messages/fact_snapshots,
ai_consents, ai_usage/reservations, receipt_drafts/preview/confirmations/cleanup.
RLS под ограниченной ролью, составные owner FKs. Новые таблицы не дают модели
или administrator финансовых привилегий другого пользователя.

Фото в приватном временном каталоге/volume вне постоянного backup; запрещены
публичные URL, логи с image/prompts и сторонняя telemetry. Draft TTL 24 часа;
очистка каждый час, после подтверждения/отмены — немедленная попытка. Рестарт
возобновляет cleanup, фактическое удаление проверяется по storage.
Нельзя обещать удаление копий у провайдера сверх проверенной endpoint policy.
Extracted text/подтверждённые поля хранятся как provenance. Разговор можно
удалить независимо от неизменяемых подтверждённых финансовых свидетельств.

Функции выключены по умолчанию; отсутствие ключа/лимита не меняет ядро.
Rollback отключает AI routes, но сохраняет подтверждённые события и receipts.
Down запрещён при подтверждённой AI-originated истории; без неё допускается
после остановки запросов и очистки временных фото. Backup/restore проверяется
до rollout. Точные имена/номера миграций определяются после принятия на актуальном main.

## Негативные сценарии

Смазанное/чужое изображение, EXIF/полиглот/слишком большой файл, OCR-подмена цифр,
скидка дважды, неизвестная валюта/дата/время/бренд, existing Receipt,
двойной клик и concurrent confirm, changed purchase after preview,
cross-owner access, instruction injection в чеке/описании, чужой tool argument,
несуществующий factRef, quota race, upstream timeout после billing, missing
cost, provider сменил policy, browser lost response, cleanup failure/restart.

## Критерии приёмки и исполняемая оценка

- [ ] AC1: отключение всех AI providers оставляет manual finance/import/receipts
  работоспособными; существующие mandatory gates проходят.
- [ ] AC2: monthly analysis содержит только Go numbers/fact references, явно
  показывает неполную историю; 20 synthetic questions, включая unknown time,
  пустую историю, несколько валют и ложные предпосылки, без выдуманных чисел.
- [ ] AC3: 20 размеченных synthetic receipts (чёткие/нечёткие/скидки/date-only)
  сравниваются одинаково для 2–3 approved models. Exact total/currency ≥95% на
  читаемой группе; unreadable/unknown обязательные поля всегда блокируют запись.
  Item-level ошибки, latency, input/output/cost и все неудачи публикуются, без
  объявления этих результатов гарантией для новых чеков.
- [ ] AC4: Jev оценивается отдельно на ≥30 synthetic товарных строках с no-match;
  unknown names остаются unknown; threshold выбирается по precision/coverage.
  При недоступности Jev остаётся ручная категория.
- [ ] AC5: upload/extract/edit не меняют ledger; confirm create/link атомарен,
  repeat/concurrent call не дублирует postings; candidate review обязательна.
- [ ] AC6: сумма/скидки/quantity/date precision валидируются Go; original values
  и owner corrections различимы; contracts regenerated.
- [ ] AC7: session/RLS/CSRF/Origin/isolation и injection tests проходят; секреты,
  raw finance/photo не попадают в Git/logs/unauthorized provider context.
- [ ] AC8: concurrent usage не превышает зарезервированный общий лимит;
  missing price/cost, retries и stale response проверены mock provider tests.
- [ ] AC9: фото действительно удалено после confirm/cancel/TTL и рестарта;
  ошибка storage не теряет cleanup job; backup не содержит photo.
- [ ] AC10: mobile camera/file flow, исправление полей, отдельное consent и
  confirmation проверены в браузере; make test, go vet, full integration,
  git diff --check и CI PASS; live эксперимент документирован отдельно.

Планируемые executable checks: Go provider httptest contract/timeout/cost tests,
PostgreSQL integration для quotas/confirm/isolation/cleanup, Web tests для
consent/редактирования/confirmation, offline eval CLI с synthetic fixtures,
`make test`, `go vet ./...`, `make test-integration`, `git diff --check`.
До реализации названия новых tests/CLI — план, а не существующие команды.

## Открытые решения перед реализацией

Нужно принять предложенный scope и политику AI-originated записи, затем
выбрать доступные model/provider IDs и получить серверный ключ с лимитом.
Платёж должен укладываться в $10 с комиссиями. Согласие на конкретный реальный
анализ/чек запрашивается в готовом интерфейсе перед отправкой. Сверка исторической
таблицы по просьбе владельца остаётся отдельной последующей работой.

Источники текущих capabilities и ограничений — в архитектурном документе.
Спека не разрешает production-код до статуса accepted и не объявляет
эксперимент выполненным до фактических запросов и записанных результатов.
