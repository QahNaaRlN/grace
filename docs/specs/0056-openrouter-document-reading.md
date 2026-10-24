# SPEC-0056: единый OpenRouter и чтение фото/PDF по возможностям модели

- Статус: **verified**; accepted G1–G4; implementation_status: **implemented_pending_owner_acceptance**.
- Дата: 2026-10-24, Asia/Yekaterinburg.
- Автор: Codex. Acceptance owner: владелец Grace.
- Основание направления: «взаимодействие с LLM в Grace — через OpenRouter…
  Чтение и распознавание фотографий и PDF сделать через тип моделей,
  поддерживающих это чтение. Внедрить эти решения».
- Контракт G1–G4 принят владельцем 2026-10-24 до production-кода.
- База: main `e9d02b2bacad7fdcef919d28b44dd4817b721aae`, PR120.
- Контракт интегрирован PR121, main `1fa1d93bc83e63ff8ddb0143bdf5849e32e6d98d`.
- Реализация: `codex/openrouter-document-reading-implementation`; локальный G1–G4.
- Production activation и качество реальной модели остаются UNVERIFIED.

## Проблема и пользовательский результат

Grace должен отправлять запросы к LLM через один шлюз OpenRouter и выбирать
модель из подходящей группы. Пользователь загружает фото чека, фотографию
выписки либо PDF, получает редактируемый результат рядом с источником,
проверяет его и явно подтверждает импорт. Файлы без текстового слоя также
читаются по изображениям. Одно распознавание не создаёт финансовые записи.

Смена модели внутри группы не требует менять код распознавания. Каталог
OpenRouter сообщает технические возможности; качество чтения конкретной модели
на наших документах проверяется отдельно. Наличие `image` не означает точный OCR.

## Текущее поведение и доказательства

- `internal/assistant/llm_provider.go`: OpenRouter Chat Completions, фиксированная
  ModelPolicy, schema, privacy/identity/cost guards. Объяснения и фото чеков уже
  используют этот путь; `receipt_extractor.go` проверяет `Vision` и отдельное согласие.
- `internal/platform/httpserver/server.go`: опциональный быстрый router вызывает
  `NewTypeSafeProvider`; это действующий обход OpenRouter, сейчас выключенный.
- `internal/statementextract/direct_page_reading_responses.go` и
  `cmd/statement-page-reading`: S1.22 обращается напрямую к OpenAI Responses;
  это отдельный исследовательский путь, не пользовательский импорт.
- `internal/pdfextract` извлекает текст/геометрию в sandbox, но не отдаёт
  изображения страниц. Пустой текст отклоняется. `statementdata.Proposal`
  выбирает реальные spans, а не значения, выдуманные моделью.
- S1.22 принят и интегрирован PR120. Четыре fake controls, включая full32,
  проверяют протокол; реальное качество остаётся UNVERIFIED. Фото без исходного
  текста нельзя считать автоматически связанным с независимым OCR-источником.

## Scope и границы

Один продуктовый пакет: G1 — шлюз и resolver групп; G2 — подготовка документов;
G3 — API/UI черновика, проверка владельцем и существующий confirmation;
G4 — общая регрессия и отчёт. Это последовательные части одной реализации,
не новые эксперименты или отдельные разрешения на каждый файл.

После принятия разрешается реализация исходников, локальные fake HTTP,
публичные синтетические документы, локальный sandbox и integration-тесты
с временной тестовой PostgreSQL. Production SSH/SQL, чтение/настройка secrets,
платные вызовы, deploy и переключение production flags в этот пакет не входят.
Существующий бюджет не увеличивается. Старые one-shot grants остаются CONSUMED;
старый прямой OpenAI V4 не запускается по этому контракту.

Не входят: произвольные типы бизнес-документов, автозапись финансов по ответу
модели, внешний OCR plugin, генерация изображений, агентные tools/search,
неограниченный выбор из всего каталога, новая система финансового учёта.

## G1. Один шлюз и группы возможностей

### Профили и выбор

В capability-модуле ассистента вводятся профили `text_reasoning` и
`document_reading`. Первый требует text input/output и strict JSON schema;
второй дополнительно image input, поддержку числа/размеров изображений и
проверяемую верхнюю границу их стоимости. Назначение документа (`receipt` или
`bank_statement`) задаёт пользовательский сценарий и schema, а не новая LLM.

Операторская конфигурация содержит версионированные группы конкретных
model/provider endpoint: ID, canonical identity, capabilities, приоритет,
price/token/image bounds, privacy/region, qualification по назначению и срок.
Resolver пересекает группу с актуальными model/endpoint metadata OpenRouter,
ценовыми и privacy ограничениями. Возможности проверяются по фактическим
`architecture.input_modalities`, `output_modalities`, `supported_parameters`
и endpoint; один старый bool `Vision` больше не служит достаточным доказательством.
Metadata proof действует не более 24 часов; устаревшая запись требует обновления.
Группа содержит не более 8 участников; metadata refresh ограничен одним
catalog GET, одним ZDR-list GET и одним endpoint GET на участника, без retry.
Данные документа при metadata refresh не передаются.

Из подходящих участников Grace автоматически выбирает первый по явному
приоритету, затем model/provider ID для устойчивого порядка. Выбор происходит
до consent/reservation. Пользователь видит выбранную модель, поставщика и
потолок стоимости. План фиксирует выбор, group revision, proof digest,
назначение, prompt/schema, source revision и hashes страниц. Изменение любого
из них требует нового preview/consent. В рамках запроса переключений нет.

Это выбор по типу в Grace; в OpenRouter отправляется конкретный model ID.
`openrouter/auto`, wildcard/latest, fallback и response-healing в этом выпуске
не используются: они добавляли бы скрытый выбор после согласия. Тестовый
каталог содержит как минимум две подходящие модели и неподходящие варианты;
одна hard-coded Astra не считается реализацией группы.

Каталог не добавляет новых участников в разрешённую группу автоматически.
Новая модель без qualification может использоваться только в отдельно
согласованной проверке; production-доступность по ней false. Существующие
проверенные analysis/receipt policies мигрируют явно, без изменения provider
или расширения отправляемых данных по одному факту наличия модели в каталоге.
Если proof/privacy/стоимость/qualification не подтверждены, профиль недоступен.

### Transport, privacy и деньги

Единственный live transport к генеративным моделям — OpenRouter Chat Completions.
Общий adapter принимает профиль/schema/страницы, не названия поставщиков API.
Все production composition roots и live research entrypoints используют его.
Прямые OpenAI/TypeSafe live entrypoints удаляются либо отказывают до чтения
ключа; старые raw fixtures и offline replay остаются историческими данными.
Неиспользуемые credentials не читаются и не подхватываются как fallback.

Быстрый read-only router переводится на metered OpenRouter structured response
с теми же шестью разрешёнными инструментами. Go валидирует route/аргументы;
неясный запрос требует уточнения. Самооценка вероятности обычной LLM не
подменяет прежние Jev calibrated probabilities. Для отправки текста действует
явное согласие, связанное с новой policy; без LLM ручной выбор tools доступен.
Не подключённый сейчас `TypeSafeGroupSuggester` не активируется: его live
adapter также закрывается, сохранённые offline оценки не превращаются в
разрешение нового товарного классификатора.

Wire сохраняет `provider.only`, `allow_fallbacks=false`, `require_parameters=true`,
`data_collection=deny`, `zdr=true`, ценовые потолки; регион проверяется отдельно
и не выводится из имени модели. Существующие более строгие требования региона
сохраняются. Запрещены redirects, tools/plugins, BYOK, скрытая pipeline и retry.
Canonical alias допускается только по свежему proof; returned model/provider,
selected endpoint, direct strategy/attempt1 и отсутствие pipeline проверяются.

До POST резервируется сумма по выбранной модели, полному тексту/schema,
всем изображениям, maximum output и существующим fees. Арифметика integer
USD micros с округлением вверх; ни image cost, ни reasoning tokens не теряются.
Если верхнюю границу нельзя доказать, POST нет. Тарифы из старого OpenAI
эксперимента не используются для OpenRouter. Usage/cost сверяются с reserve;
неизвестное списание удерживает reservation и блокирует повтор до сверки.

## G2. Документы и локальная подготовка

Фото: JPEG/PNG/WebP, до 8 MiB и 10 миллионов decoded pixels, не больше
10000 по стороне; нормализация orientation/metadata по существующему photo
контракту. В модель передаётся приватное изображение data URL, не публичная
ссылка. Фото выписки — один документ/одна страница; пакет фотографий позже.

PDF: до 10 MiB, сохранённые PDF/manual сценарии до 50 страниц продолжают
работать. Первый visual reader принимает целиком не более 4 страниц и
200 операций; превышение явно отклоняется без POST, без чтения только префикса.
Sandbox рендерит все страницы в PNG с длинной стороной до 1800, суммарно
до 8 MiB; padding/rotation/page order и source hashes фиксируются. Извлечение
нативного текста/геометрии сохраняется, если текст есть. Отсутствие текста
допустимо для visual path, но не меняет контракт старого text-only extractor.

В API-процессе PDF parser не запускается: расширяются Unix-socket helper,
лимиты процессов/времени/памяти и проверка версии. Helper без сети/credentials,
без исполнения вложений/JS, без внешних URL. Рендер не использует провайдеров. Неподдерживаемый encrypted PDF или неверная структура дают отказ без отправки.
OpenRouter получает изображения всех страниц плюс доступный plain text;
raw PDF/file-parser plugins не отправляются. Поэтому требуется image-capable
модель; отдельный `file` capability не обязателен для локально отрисованного PDF.

Один запуск чтения — один POST, не более 30000 input и 16000 output tokens,
до 180 секунд. Это верхние границы профиля, а не обещание тарифа для любой
модели. Малый context/output limit участника исключает его до POST. Обрезка,
неполный ответ или >200 строк дают явный отказ; частичный ответ не готов к импорту.

## G3. Черновик, происхождение и подтверждение

Результат модели: строки в порядке документа, исходные строковые значения,
page/quote и bounded page regions, null/нечитаемое и issues. Даты/время и валюта
не дополняются догадкой; денежные суммы разбирает и сверяет Go. Schema и prompt
разделяют операции, opening/closing BOOK balances, available и обороты.

Два честных вида evidence:

- `native_text`: цитата однозначно связана с настоящими spans; прежние
  grounding/roles/description/coverage проверки продолжают действовать.
- `image_review`: модельная транскрипция и область изображения — кандидат,
  не независимый OCR. Нельзя создавать фиктивные Poppler spans и объявлять
  прежний source binding PASS. Проверка владельцем хранится отдельно.

Для image_review владелец открывает страницу и подтверждает каждую строку
целиком, включая обязательные поля/описание; отдельно отмечает полноту страниц
и роли итогов. Receipt применяется существующий review/confirmation черновика.
Подтверждение привязано к source hash/revision, row ID и digest всех её полей;
редактирование строки сбрасывает её проверку, изменение источника — все проверки.
Исправление хранит actor/reason и исходный кандидат. Model output никогда не
устанавливает owner-reviewed flags. Пустая сумма/дата/валюта, неоднозначность,
duplicate/transfer/conflict или несверенные обязательные показатели оставляют
blocker; строки с blocker нельзя провести. Отсутствующий optional показатель
остаётся unknown, его не заменяют нулём и не показывают как успешную сверку.

Recognition, native source binding, owner visual review, reconciliation и
import readiness возвращаются отдельно. Проверка модели не становится PASS
по одному факту owner confirmation. `imports` остаётся владельцем существующего
preview/confirmation, idempotency и matching по Posting; financial core от LLM
не зависит. Confirmation перечитывает источник, revisions, review и blockers
в транзакции. Удалённый/истёкший источник, чужой owner и stale preview отвергаются.
LLM adapter не пишет FinancialEvent/Posting/Receipt самостоятельно.

### API/OpenAPI и UI

Сохраняются существующие пути PDF/manual, receipt, extraction previews/consents/
attempts. Новый upload `POST /api/imports/statements/images` принимает multipart
`file` и `accountId` по правилам PDF upload. Формат определяется bytes+MIME;
переименование `.jpg` в `.pdf` не обходит проверку.

Statement получает `sourceKind` (`pdf`/`image`), `readingMode`
(`native_text`/`image_review`), page metadata без публичных файловых путей.
Owner-scoped `GET /api/imports/statements/{id}/pages/{page}` отдаёт только
нормализованное изображение с приватным cache policy; удалённый оригинал —410.
Extraction preview дополняется `profile`, `groupRevision`, выбранной model/provider,
metadata proof digest, maximumCost и полным списком отправляемых page hashes.
`visibleSpanIds` сохраняется для старого text path; visual consent охватывает
целые явно показанные страницы, а не изображает их span-redaction.

Новый `POST /api/imports/statements/{id}/visual-reviews` принимает
`expectedRevision`, source hash, проверенные row IDs с field digests, reviewed
pages и roles confirmation; CSRF/session/RLS/idempotency обязательны. Ни raw
модельный ответ, ни один клиентский `verified=true` не могут обойти серверную
проверку revision/digests/blockers. Возвращается актуальный review/revision.

Для PDF чека добавляется `POST /api/receipt-drafts/documents` с multipart `file`;
старый upload `photo` сохраняется. `GET /api/receipt-drafts/{draftId}/pages/{page}`
отдаёт приватный preview. Один PDF содержит один чек, до 4 страниц и прежних
50 позиций; несколько чеков в одном файле дают incomplete_reading. Назначение
выбирает пользователь до отправки. Receipt extractor получает страницы через
общий document_reading, сохраняя matching, edit, previews и confirmations.
Источник/страницы/consent hashes и retention расширяются и для ReceiptDraft.

В Web: выбор фото/PDF и назначения, preview изображений/таблицы рядом, поля
unknown/issues, исправление и отметки проверенных строк/страниц, явное итоговое
подтверждение. AI недоступен — объяснимая причина и ручной путь. Типы генерируются
из OpenAPI; бизнес-правила остаются Go, FSD и лимиты модулей сохраняются.

## Данные, миграция и rollback

Аддитивная миграция: source kind/reading mode для statement и receipt drafts, версии render/profile/proof,
page image metadata/storage refs, candidate rows/regions и owner review receipts.
Изображения хранятся приватно с owner isolation вне Web root; оригинал и render
удаляются одним retention/cleanup, минимальное evidence сохраняется по прежнему
контракту. Значения чека/выписки и изображения не попадают в logs, CI или public PR.

Исторические native_text reviews, Posting/Receipt/events не переписываются;
старые согласия не превращаются в новые visual consent. Откат: выключить новый
профиль, сохранить/обработать черновики вручную, оставить аддитивные данные.
Возврат на старую binary с неизвестным image source запрещён до проверки
совместимости; rollback не включает удаление source/evidence или возврат
прямых обращений к другому API. Точный миграционный diff входит в implementation review.

## Ошибки и негативные сценарии

Явные коды: no_eligible_model, capability_unverified, policy_changed,
price_unknown, budget_exceeded, consent_required, document_limit,
invalid_image, extractor_unavailable, provider_protocol, incomplete_reading,
owner_visual_review_required, revision_conflict, statement_original_gone.
Существующие публичные коды сохраняются; новые описываются в OpenAPI.
Отказ до POST не тратит provider budget; неопределённый результат после отправки
не вызывает автоматический повтор. Отказ распознавания не создаёт финансовых записей.

## Критерии приёмки и исполняемые проверки

Локальные G1–G4 проверки выполнены; actual commands/results и ограничения —
[0056-evidence.md](0056-evidence.md). Приёмка implementation PR и merge pending.

| AC | Наблюдаемый результат и обязательная проверка |
| --- | --- |
| G1.1 | Production и live research composition не отправляют inference в OpenAI/TypeSafe напрямую; endpoint inventory + fake transport проверяют URL и отсутствие чтения старых ключей |
| G1.2 | Fixture-каталог с ≥2 vision-моделями: выбирается подходящая по группе; text-only, нет schema, stale proof, неверный endpoint/privacy/тариф →0 POST; один конкретный ID не зашит в reader |
| G1.3 | Неверная returned identity/alias, pipeline/fallback/BYOK/attempt2, неизвестный bill и превышение cap не дают успешного результата; reservation/settlement не дублируются |
| G1.4 | Анализ, receipt и быстрый router работают через fake OpenRouter и общий meter; без key/consent/qualified группы нет отправки, deterministic tools доступны |
| G2.1 | Sandbox реально рендерит public cards/columns/full32 и PDF без текста; порядок/число/хеши и лимиты проверяются; API не парсит PDF in-process |
| G2.2 | Настоящие PNG/JPEG/WebP fixtures доходят до image input; EXIF удалён, bytes/MIME проверены; oversize/malformed/timeout/5-я страница →0 POST |
| G3.1 | Один offline сценарий upload→preview→consent→fake reading→edit/review→confirm→reload для фото выписки, PDF выписки, фото чека и PDF чека, включая scan-only, без фиктивного native binding |
| G3.2 | Все32 операции/3 страницы full32 сравниваются с golden; описание через страницу, FX, own transfer, fee и точность времени проверяются; recognition отдельно от readiness |
| G3.3 | До owner review/при null/duplicate/подмене digest/чужом owner/expired source/stale revision подтверждение отказывает и финансовых записей0; успешный import атомарен, повтор не создаёт дублей |
| G3.4 | Native-text guards и старый ручной PDF/JSON/CSV/receipt путь не ухудшены; после import/повторной загрузки totals и postings точны; retention удаляет original+render |
| G4 | OpenAPI/TS синхронны; make test, make test-integration, make test-production, go vet ./cmd/... ./internal/..., git diff --check PASS; UI evidence с локальными fake-вызовами |

Конкретные test names/commands и actual output добавляются в один
`0056-evidence.md` после реализации. Команды общего уровня не заменяют
перечисленные причинные негативные сценарии. Тесты не используют живые ключи.

## Внедрение и открытые решения

1. Принять этот контракт, записать receipt и интегрировать спецификацию.
2. Реализовать G1–G4 от принятого main в отдельном worktree; представить один
   implementation PR с AC/evidence. Merge реализации оценивается по готовому diff.
3. Отдельно согласовать проверку конкретных моделей и пользовательских
   документов: точный source, разрешённые данные, число вызовов/цена. Без неё
   real model quality и новые production group members остаются UNVERIFIED.
4. После evidence и приёмки — отдельное решение о production rollout.

Для принятия G1–G4 не требуются новые ключи или выбор единственной модели.
Начальный production-состав группы и качество на реальных документах пока
не подтверждены; fake catalog проверяет механизм, не квалифицирует модель.
Решение о rollout/платной проверке здесь не подменяется статусом кода.

## Официальные источники для контракта

Проверены 2026-10-24, без provider API/inference calls:

- [Models API: modalities, parameters, pricing](https://openrouter.ai/docs/api/api-reference/models/list-all-models-and-their-properties).
- [Image input через Chat Completions](https://openrouter.ai/docs/guides/overview/multimodal/image-understanding).
- [PDF: native input и внешний parser при fallback](https://openrouter.ai/docs/guides/overview/multimodal/pdfs).
- [Provider constraints и max_price](https://openrouter.ai/docs/guides/routing/provider-selection).

## Принятие владельцем до реализации

2026-10-24 владелец ответил **«Принять контракт, интегрировать и реализовать»**
на конкретный G1–G4 в PR121/head
`63209fc672fa677ec41a7fadb6afd44a6f4c498a`. Разрешены фиксация принятия,
интеграция контракта после exact-head CI5/5 и последующая полная локальная
реализация G1–G4 в отдельном worktree, включая fake/sandbox/integration проверки.
Платные вызовы, чтение/настройка ключей, production SSH/SQL/deploy и activation
не разрешены. Слияние будущего implementation PR не входит в это решение.
