# SPEC-0059: реальный чек — доступный маршрут и один product-прогон

- Status: **accepted**
- Acceptance record: 2026-10-26 владелец ответил «Принять SPEC-0059 и выполнить прогон» на явный вопрос о реализации/CI/merge/deploy, передаче этого фото через OpenRouter/Google EU, ≤1POST/≤$0.04 и нулевых финансовых записях.
- Date: 2026-10-26
- Implementation status: **integrated PR142 / deployed dab06a9; one product call completed, raw contract79/80; grant CONSUMED1/1**
- Evidence: [0059-evidence.md](0059-evidence.md)
- Worktree: `receipt-photo-reading-spec`, branch `codex/receipt-photo-reading-spec`
- Основание: владелец предоставил реальное фото, подтвердил расшифровку и поручил продолжить проверку через Grace.
- Родители: принятые SPEC-0026/SPEC-0056; исключение SPEC-0058 P остаётся только statement/plain.

## Проблема и результат

Загрузка реального фото в обычном production UI работает, но кнопка чтения отключена.
Для receipt сохранён старый Gemini 2.5 Flash / Google Vertex Global. Read-only сверка
2026-10-26: каталог содержит модель, но настроенный endpoint отсутствует и в
endpoint-списке этой модели, и среди ZDR routes. Это NOT_SENT, не ошибка чтения фото.

Пользовательский результат: загрузить один чек, увидеть модель/провайдера/потолок,
разрешить передачу, получить редактируемые позиции и сверить их с исходным фото.
Финансовое сохранение требует отдельного подтверждения и в проверку не входит.

## Scope и non-goals

Исправить только readiness receipt и провести один новый product-вызов на уже
предоставленном фото. Модель — `google/gemini-3.1-flash-lite`, canonical
`google/gemini-3.1-flash-lite-20260507`, provider `google-vertex/eu` / Google.
Этот маршрут ранее проверен на PDF, а качество реального чека до данного вызова
остаётся UNVERIFIED. Нет fallback, автоматического retry и подмены эталоном.
Analysis, statement, финансовые правила, категории, счета, покупки и закрытые
экспериментальные серии не расширяются. Промпт, receipt envelope/JSON schema и
существующий decoder сохраняются: сначала проверяется фактический product-путь,
без добавления quotes, координат, evidence bounds или дополнительных LLM-вызовов.

## Контракт тарифа и инварианты

Новый явный opt-in `requestFeeEvidence=receipt_router_ceiling_only` допустим только
для purpose `receipt` / profile `document_reading`. Только отсутствующий ключ
`pricing.request` допускается при положительном request ceiling ≤1000 USD micros
($0.001), полностью зарезервированном и переданном в `provider.max_price.request`.
Отсутствие цены не трактуется как нулевая цена. Присутствующие null, пустая,
некорректная, отрицательная и превышающая ceiling цены отвергаются.

Остальные catalog/canonical/provider/ZDR/region/image/token/qualification/proof
проверки обязательны. Пределы: input ≤275000 и output ≤1650000 USD micros на
миллион токенов, image ≤1 USD micro; imageTokenUpperBound 4200; maxImages=1,
maxImageSide=2048; request input bound ≤30000, output ≤16000.
Свежий metadata proof ≤24 часов. Временная операторская квалификация для первого
owner-approved product-теста ≤24 часов не объявляет качество уже проверенным.
После результата режим и необходимость дальнейшей квалификации фиксируются явно;
автоматическое продление и новый платный тест не разрешены.

Код Go проверяет денежные строки и арифметику, деньги хранятся в minor units/RUB.
Сохраняются отдельный consent, source/draft revision, preview hash, owner isolation,
ручные исправления, неизвестные поля и подтверждение покупки. Без часового пояса
issuedAt остаётся null, issuedDate содержит календарную дату; напечатанное время
сохраняется в sourceText, а не превращается в выдуманный UTC timestamp.

## API/OpenAPI и данные

Новые API DTO, маршруты, миграции и изменение схемы БД не нужны. Используются
существующие upload → reading preview → consent → reading → draft/reload.
Неизменённый ответ и распознанные поля не подменяются локальным эталоном.
Фото и эталон содержат частные данные и остаются вне Git/GitHub; публичный отчёт
содержит только агрегаты. Срок хранения фото по прежнему контракту — до суток,
подтверждения или отмены. При истечении срока повторная загрузка того же оригинала
допустима до единственного inference, без увеличения числа платных попыток.

## Разрешение на выполнение после acceptance

Принятие открывает реализацию, обязательные проверки, PR/CI/merge, exact-SHA
Delivery/deploy с backup/readiness, обновление только receipt membership и один
production Web/API inference на предоставленном фото через OpenRouter/Google EU.
До inference показываются исходное фото, модель и максимальная стоимость и
используется обычный browser consent. Эталон модели не передаётся.

Всего ≤1 provider POST, ≤$0.04 резерва и суммарного расхода с комиссиями, без
увеличения общего allowance. До первого вызова ≤3 metadata GET для fresh proof;
при истёкшем proof работа останавливается, новое превышение GET не подразумевается.
При unknown bill резерв удерживается, новые POST запрещены; settlement только по
подтверждённой стоимости, административный ноль не разрешён. При HTTP/protocol или
quality REJECT — отчёт и остановка, без второго вызова. Финансовых записей — 0.

## Приёмка и исполняемые проверки

- A1: unit tests разрешают opt-in только receipt/document_reading с ограниченным
  положительным ceiling. Другие purpose/profile и прежние experiment/plain режимы
  не расширяются. Негативные цены и отсутствие opt-in отвергаются.
- A2: wire test подтверждает reservation и `provider.max_price.request`, конкретный
  endpoint/ZDR, отсутствие fallback/retry. Подготовка и загрузка не делают inference.
- A3: существующие receipt SQL/owner/consent/revision/preview/confirmation guards
  проходят; не меняются финансовая семантика и API.
- A4: `make test`, `git diff --check`, `go vet ./cmd/... ./internal/...`, профильные
  SQL integration tests; CI/Delivery/deploy и post-merge проверки точного SHA.
- A5: обычный браузер получает новый ответ, draft/reload сохраняет результат.
  Сверка до ручных правок: магазин, календарная дата, валюта, итог, все 9 строк
  включая бесплатную служебную позицию, названия/количества/цены/суммы, видимое время
  в sourceText. Нормализуются только кавычки/пробелы продавца и регистр/пробелы
  названий. Денежные значения, дата, количество и состав строк — точные.
  Остальные напечатанные поля, поддержанные decoder, проверяются отдельно по фото.
- A6: raw mismatch и результат после ручной правки сообщаются раздельно. Один чек
  не доказывает качество всех фото. Подтверждены известный bill/settlement,
  ≤1POST/≤$0.04 и нулевые изменения финансовых событий/проводок.

## Ошибки и rollback

Недоступный маршрут, отсутствующие тарифы/qualification/proof, неподходящее фото,
старый consent, timeout, malformed/incomplete response дают явную остановку.
Сохранённый upload без вызова не считается quality failure. Для rollback отключить
receipt route/вернуть прежнюю недоступную receipt membership; не менять statement,
analysis, финансовые записи и retained drafts, не откатывать additive migrations.

## Открытые решения

Открытых решений по этому scope нет. Принятие записано до кода и изменения
операторской конфигурации. На момент acceptance paid POST=0; слот доступен 1/1.

## Итог однократного прогона

Все9 строк распознаны, raw79/80: одно верно прочитанное локальное время помещено
в issuedAt без RFC3339/timezone. Actual$0.003055 settled, reserve$0.029642,
financial deltas0. Trial закрыт, повтор не открыт. [Полный отчёт](0059-evidence.md).
[Уточнение R](0059-receipt-normalization-candidate.md) — отдельный draft; код по
нему не реализован, новых платных вызовов он не подразумевает.
