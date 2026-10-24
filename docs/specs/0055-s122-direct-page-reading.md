# SPEC-0055 / S1.22: прямое чтение страниц выписки

- Статус: **accepted V1–V3 offline**; implementation_status: **accepted_implementation / integrated PR120**.
- Направление принято владельцем 2026-10-24: «Значит пора сменить курс, попробовать то, что работает».
- Дата: 2026-10-24, Asia/Yekaterinburg. Автор: Codex.
- База: main `ad1eca3efed097901bd15d8f7207d592c89ec24f`, принятый отчёт PR119.
- Worktree: `statement-direct-page-reading`; branch: `codex/statement-direct-page-reading`.
- Принимаемое решение ниже — конкретный контракт offline-реализации; **V4 NOT_AUTHORIZED**.

## Зачем меняем курс

Последний S1.21 P4/run40109843789 завершён REJECT на grouping/partition_invalid:
base14/20, candidate до извлечения полей не дошёл,2 POST/$0.005509 known/settled.
Отчёт принят и интегрирован PR119/mainad1eca3; main CI40111553326/all5 и
Delivery40112288231/all4 PASS. Старые результаты/разрешения сохраняются, grants CONSUMED.

Прежний протокол давал модели текстовые aliases с координатами и требовал
полного распределения всех фрагментов. Новый candidate читает страницы целиком
и возвращает понятные строки операций. Принадлежность каждого служебного токена
не является задачей модели; финансовая корректность остаётся задачей Grace.
Польза такого входа — **гипотеза**, не установленный факт и не обещание качества.

Один пакет: контракт → offline implementation/fakes → отдельно разрешённая
ограниченная проверка модели → evidence/review. Не создаём отдельные спецификации
для каждого receipt или отчёта. Не продолжаем правки grouping prompt S1.21.

## V1. Вход, результат и четыре фиксированных примера

Модель получает упорядоченные изображения всех страниц PNG (`detail=high`),
локально извлечённый обычный текст при его наличии и один короткий instruction.
Нет span-ID puzzle, golden, эталонных операций, scorer feedback или соседних ответов.
Текст документа и команды внутри изображения всегда считаются недоверенными данными.
Один запрос читает весь документ; нет agent tools, web, repair calls или hidden retries.

Фиксируем corpus **до** model calls:

| Case | Вход и назначение |
| --- | --- |
| cards | Существующий public cards.pdf:2 операции, operation/booking clocks и descriptions |
| columns | Существующий public columns.pdf:2 страницы, разные валютные колонки, available отдельно от book balance |
| cards-image-only | Те же public страницы без текстовой подсказки; контроль visual-входа, **не независимый документ** |
| full32 | Новый публичный synthetic PDF:32 операции/3 страницы, русский текст, повтор заголовков, перенос описания через страницу, разные операции с одинаковыми суммами, собственный перевод, fee и FX |

Full32 и golden создаются из фиксированного детерминированного набора данных,
не из ответов модели. Никаких данных владельца, существующих private payloads,
маскированных копий реальной выписки или изменения goldens после запуска.
Manifest содержит hashes PDF/PNG/plain text/golden, порядок, размеры, число страниц/строк,
renderer image digest/version и prompt/schema. Public fixture path сам по себе
не подтверждает отсутствие персональных данных: corpus проходит content review.

Renderer только локальный, отдельный sandbox: без сети/secrets, read-only input,
non-root, лимиты времени/памяти/процессов; максимум4 страницы на case,10MiB PDF,
PNG long side≤1800px, суммарно≤8MiB PNG на запрос. Полные страницы без обрезки.
Нечитаемость/лимит — явный отказ; image не заменяется незаметно text-only входом.

Выход — строгий research DTO с header и rows: operation/booking date+time
с печатной precision, описание целиком, account amount/currency/direction,
отдельные original amount/currency, печатные available/book balances/turnovers.
Отсутствующие значения — null, неоднозначность — issues, не догадка или ноль.
Для каждого поля обязателен упорядоченный evidence-массив `{page, quote}`;
quote дословная, поле через разрыв страницы имеет несколько evidence segments.
Строка содержит общий source anchor. Денежные строки сохраняют печатное значение; Go разбирает их в
integer minor units с явной валютой, проверяя знак, формат и overflow.
Модель не вычисляет остатки, не назначает финансовые events и не пишет в БД.

## V2. Конкретный candidate и offline source

Предлагается **OpenAI Responses API, gpt-6-astra, reasoning.effort=medium**,
строгая JSON schema, store=false, stream=false, service_tier=default, tools отсутствуют,
max_output_tokens=16000 (включая reasoning), без temperature/seed и fallback.
В качестве входа выбран явный PNG+text, не provider PDF parser: состав запроса
можно проверить по manifest. Польза модели на corpus пока UNVERIFIED.

Это новый изолированный research adapter, не переключение production OpenRouter.
Доступ аккаунта/ключа к модели и privacy/region условия **UNVERIFIED**: ключи,
secrets/config и баланс сейчас не читаем и не создаём. Private/owner документы
для этого пакета запрещены. store=false не объявляется доказательством ZDR;
публичный corpus не снимает будущий отдельный privacy gate для реальной выписки.

Предлагаемое принятие V1–V3 разрешает **создать исходники**:

- `scripts/research/prepare-statement-page-corpus.py` и публичные corpus/goldens;
- `internal/statementextract/direct_page_reading*.go`: DTO, request builder,
  строгий decoder и binding; Responses adapter проверяется только fake HTTP;
- `internal/statementqualification/direct_page_reading*.go`: полный judge/report,
  независимый replay и fake controls;
- `cmd/statement-page-reading`: default check без env/keys/network; gated future
  run mode с exact source, execute flag, one-shot marker и отдельным research allowance.

Это разрешение записи source future capability, **не её исполнения**.
Тесты только fake HTTP/local fixtures. Новых GitHub workflows/VPS wrappers/SSH,
SQL/migrations/API/Web DTO/production wiring в scope нет. Существующие runtime
flags, scorer profiles, financial guards и manual path сохраняются.

## V3. Проверяем результат и сохраняем путь к реальному импорту

Три раздельных результата: **recognitionPassed**, **sourceBindingPassed**,
**importReady**. Последний в исследовании всегда false: owner review/confirmation не выполнены.
Recognition сравнивает каждое поле каждой строки с frozen golden: полнота,
порядок, отсутствие лишних/дублированных операций, точные суммы/валюты/направления,
обе даты/время/precision, полные descriptions, header roles и null для отсутствующих полей.
Нельзя усреднить пропуск строки хорошими суммами или считать правильный JSON успехом.

Binding выполняется **после** распознавания: Go связывает page/quote/row anchor
с существующими source spans. Допустимы только NFC и явно перечисленное свёртывание
пробелов/переносов; нельзя убирать пунктуацию, знак, слова или время.
Совпадение должно быть уникальным в допустимой области строки/страницы.
Неоднозначный match оставляет blocker; запрещено выбирать первое совпадение,
подставлять golden или автоматически «исправлять» модель.
После binding запускаются прежние deterministic grounding/coverage/role checks.
Остаток source остаётся unresolved с видимым coverage blocker. Код не объявляет
непривязанный фрагмент служебным только потому, что модель его не использовала.
Никакого присвоения matched/rolesConfirmed или снятия blocker для прохождения теста.
SourceBindingPassed означает уникальную привязку распознанных полей, не полное
покрытие документа и не разрешение импорта. Owner-review/coverage blockers
публикуются полностью; отсутствие owner review не маскируется под quality PASS.
Новый research target оценивает распознавание+привязку, это явное изменение
исследовательского вопроса; прежний полный scorer запускается и публикуется
отдельно без изменения scores/thresholds. Перенос owner review в будущую UI
интеграцию не доказывает готовность полного импорта.

Для image-only контроля source spans известны checker из исходного public PDF,
но не передаются модели. Это проверка распознавания по картинке, **не готовый OCR**
для произвольной фотографии без текстового слоя. Такая product-поддержка остаётся будущим этапом.

Новый research profile `statement.direct-page-reading.v1` не переписывает
исторические scores S1.19/S1.21. Старые fixed-corpus checks сохраняются; новый full32
проверяется построчно по всем тем же смысловым категориям без сокращения покрытия.
Положительные fake controls обязаны пройти все4 cases; meaningful negatives —
потерянная/повторная строка или страница, неверная валюта/1 minor/time precision,
потерянное слово, available как closing, ложная quote, ambiguous match, malformed/
truncated/oversize output, injection, unknown usage/cost, timeout и повторный запуск.

Будущая campaign — каждый case два раза, фиксированный порядок
cards, columns, cards-image-only, full32, затем тот же порядок; ≤8 POST.
Это3 разных синтетических документа и visual-control, не8 независимых samples и не p95.
Все запросы фиксированы до первого ответа; второй проход не получает первый.

- **KEEP_FOR_FURTHER_REVIEW**: все8 результатов полные и точные; recognition и
  source binding PASS во всех случаях; нет source-value/role contradictions;
  полные прежние checks и unresolved/owner blockers опубликованы; cost/protocol evidence известны.
  importReady=false сохраняется, KEEP не объявляется full quality/coverage PASS.
- **REJECT**: доказанная смысловая ошибка/пропуск/неуспешный source binding в любом
  полном валидном результате. Остаток campaign not_sent, без повторной попытки.
- **INCONCLUSIVE**: infrastructure/transport/usage/identity/output-limit refusal,
  нет валидного результата для решения. Не подменять этим известную смысловую ошибку.

Запрошенная и возвращённая model identity сверяются с явным manifest allowlist;
неизвестная identity — INCONCLUSIVE, без переключения на другую модель.
Judge/replay сохраняют исходный тип результата, каждый check, witness, not_sent
suffix, identity, hashes, usage, длительности и ограничения. Конкретный дефект
не должен снова скрываться за одним общим кодом; public synthetic evidence может
содержать соответствующую quote/ожидание/наблюдение. Ключи/headers туда не попадают.
KEEP открывает review следующей интеграции, не готовность продукта. Для исходной
цели ещё нужны согласованные real/representative samples, owner UI review→confirm→
reload/dedup и отдельный production release. Финансовые проверки/подтверждение сохраняются.

## V4. Будущий exact-source запуск: NOT_AUTHORIZED

До отдельно разрешённого запуска: source/CI5/5, corpus/prompt/schema hashes,
account/model доступ и funding, endpoint/region/privacy, точные flags/paths,
token/price bounds и предельная сумма опубликованы для владельца.
Никаких calls для проверки ключа/модели до соответствующего допуска.

Предлагаемые bounds:8 POST только `/v1/responses`, provider GET/Files API/tools0,
≤30000 billable input tokens/≤16000 output tokens на POST, включая изображения и reasoning.
Preflight считает консервативную верхнюю границу text/schema/image tokens по
зафиксированным размерам и официальным правилам. Если bound не доказан — отказ.
Резерв по price ceiling $14 input/$55 output за1M tokens: **≤$1.30/request,
≤$10.40 provider за campaign**. Input ceiling включает запас под опубликованные
cache-write ставки и regional uplift; fast/long-context/длинное cache retention
не включаются молча. Их наличие или неподтверждённые доплаты — отказ до calls.
Это предложение ceiling, **не разрешение расходов,
пополнения или увеличения прежнего Grace budget**. Актуальные цены, cache-write/retention условия, доплаты и
средства должны быть явно согласованы в exact-source решении; неизвестные fees — отказ.

Изолированный local research allowance/atomic receipt привязан к одному source
и одному будущему допуску; это execution accounting, не финансовая БД Grace.
Весь ceiling резервируется до первого POST. После неизвестного результата
расход остаётся uncertain, новая попытка/освобождение для повторного run запрещены.
Usage×rates записывается как **calculated provider cost**, не выдается за фактический
invoice или существующий production settlement. Key только из заранее согласованной
env-переменной после execute gate, не argv/log/report; никакого чтения ключей чата.
Любой отказ расходует one-shot, SDK retries=0, redirect/fallback0; timeout≤180s на POST/≤1500s campaign;
при observation timeout читается
тот же handle/result, не запускается второй run. SSH/VPS/production SQL/deploy0.

## Acceptance criteria и решение владельца

| AC V1–V3 | Исполняемое evidence после принятия |
| --- | --- |
| Реальный visual wire | Fake HTTP проверяет PNG bytes/hashes/page order/detail/text/schema/model; golden/aliases/feedback отсутствуют; image-only case без text |
| Полный corpus | Детерминированный renderer/generator,4 positive controls, full32 с32 уникальными строками/3 страницами, frozen manifest до calls |
| Семантика и binding | Full field/row/header/precision checks; meaningful negatives; unique quote binding→старые guards; unresolved/blockers сохранены, importReady=false, без weaken/repair/golden substitution |
| Safe defaults/caps | Check с invalid HOME/key/PATH spies0 внешних действий; source/execute/marker refusal; fake requests/cost/uncertainty/cleanup/retry matrices |
| Replay/privacy | Независимый replay/все checks и конкретный witness, strict source-bound report; secret/raw header не появляется; старые reports byte-equal |
| Regression | make test; предметные fake/renderer tests; go vet; git diff --check; exact-head CI5/5 до review |

Владелец принимает конкретные **V1–V3 offline**, включая перечисленное создание
исходников future CLI/adapter и local fake tests. Реализация начинается после
фиксации этого принятия до кода. Merge, V4, key setup, paid calls, SSH/SQL/deploy
и product integration этим решением **не разрешаются**.

## Документированные основания, проверены 2026-10-24

- [OpenAI Docs: gpt-6-astra](https://developers.openai.com/api/docs/models/gpt-6-astra): image input, structured outputs, Responses, reasoning; стандартные ставки $10/$50 за1M input/output tokens. Это API candidate, не утверждение идентичности чат-сеансу владельца.
- [OpenAI Docs: file inputs](https://developers.openai.com/api/docs/guides/file-inputs): PDF-вход может включать текст и изображения; здесь выбран явный PNG+text для наблюдаемого wire.
- [OpenAI Docs: images and vision](https://developers.openai.com/api/docs/guides/images-vision): detail и image-token accounting входят в preflight; visual input сам по себе не гарантирует правильность.
- [Pricing](https://developers.openai.com/api/docs/pricing): regional processing может добавлять10%; price ceiling выше учитывает это только как консервативную границу, не обещание доступности региона/аккаунта.

Проверка подготовки предложения: изменены только5 Markdown файлов; `make test`
PASS (Go,89 Web suites/336 tests, lint/FSD/format/build/Jev), `go vet` PASS,
`git diff --check` PASS. Код V1–V3 не написан, model calls/keys/config/SSH/SQL/deploy0.

## Прямое принятие до реализации

2026-10-24 владелец прямо ответил **«Принять V1–V3 и реализовать offline»**
на PR120/head `fa18c519bf579776696b0bdf30c2ba0a62275416`. Принят перечисленный
source corpus/renderer/DTO/adapter/judge/CLI и только local/fake проверки.
V4/paid calls/чтение и настройка ключей/SSH/SQL/deploy/activation/merge не разрешены.
Этот receipt фиксируется отдельным commit до первого файла кода.

## Offline-реализация V1–V3

Реализована в том же PR120 после receipt `e54b154`. [Отчёт и исполняемые AC](0055-s122-offline-evidence.md). Четыре public fake controls проходят recognition/binding; полный full32 проверяет32 строки. Это evidence кода, качество реальной модели и account/key readiness остаются UNVERIFIED; V4/activation не разрешены. Merge отдельно принят ниже.

## Принятие реализации и интеграция

Владелец ответил «Принять реализацию и слить №120» для source
`52842439df5a3eb58df52b38690098a792f12602`. PR120 интегрирован main
`e9d02b2bacad7fdcef919d28b44dd4817b721aae`; source CI40117756468/all5,
main CI40118781180/all5 и Delivery40119558599/all4 PASS. Tree main совпал с
принятым source. Это приёмка offline-реализации, не качества модели.

Последующее поручение перевести всё LLM-взаимодействие на OpenRouter оформлено
в [SPEC-0056](0056-openrouter-document-reading.md), пока draft. Исторический
прямой OpenAI wire остаётся описанием реализованного S1.22; отдельный V4 не
запускался и не разрешён. Новое направление не расходует старый allowance.
