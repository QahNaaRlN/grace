# SPEC-0056 Q: проверка чтения документов через OpenRouter

- Статус: **accepted**; Q1–Q3 **ACCEPTED**, implementation_status: **implemented_pending_owner_acceptance**.
- Дата: 2026-10-25, Asia/Yekaterinburg. Исполнитель: Codex.
- Основание: поручение владельца «Идем дальше» после принятия реализации G1–G4.
- База: PR122, main `10a0059bb90265ea1b39a51f6f84037ebc4dd9c8`.
- Родитель: [принятый SPEC-0056 G1–G4](0056-openrouter-document-reading.md).
- Q4 **NOT_AUTHORIZED**. Контракт принят до нового кода; offline-реализация в этом же PR, owner acceptance pending.

## Цель и граница решения

Ответить на практический вопрос: читает ли один конкретный image-capable
endpoint OpenRouter наши документы через уже реализованный адаптер Grace,
сохраняя поля, строки и известную стоимость? Fake-контроли G1–G4 проверили
протокол; они не доказывают качество настоящей модели.

Принятый пакет Q1–Q3: подготовить воспроизводимый набор, runner,
полный scorer, replay и операционные исходники, проверить их только локально.
Принятие Q1–Q3 разрешило создание перечисленных ниже файлов и необходимых
общих компонентов в том же PR. Дополнительная спецификация на каждый файл
не нужна. До принятия production-код по этому дополнению не изменяется.

Платный запуск Q4 — отдельное решение по готовому exact source, manifest,
конкретным model/provider и тарифным границам. Даже успешный Q4 не включает
production-профиль, не добавляет участника в рабочую группу и не разрешает
импорт финансов. Старые S1.19/P4/B4/V4 grants не переиспользуются.

## Q1. Восемь публичных документов и полный scorer

Порядок фиксирован. Один документ — не более одного POST, без повторных
выборок и изменения промпта между случаями. Все данные синтетические;
частные документы, старые private payloads и финансовые записи исключены.

| № | Документ | Представление и проверяемый результат |
| --- | --- | --- |
| 1 | cards | `internal/pdfextract/testdata/cards.pdf`: 1 страница, 2 операции, native text/spans + PNG |
| 2 | columns | `internal/pdfextract/testdata/columns.pdf`: 2 страницы, 2 операции, native text/spans + все PNG |
| 3 | cards-photo | PNG cards/page-1 из `internal/statementextract/testdata/direct-page-reading`, нормализованный как фото; 2 операции, text/spans отсутствуют |
| 4 | full32-native | `internal/statementextract/testdata/direct-page-reading/full32.pdf`: 3 страницы, все32 операции, native text/spans + PNG |
| 5 | full32-scan | `internal/pdfextract/testdata/full32-scan.pdf`: 3 страницы, все32 операции, только PNG; отсутствие text проверяется локально |
| 6 | receipt-discount | `testdata/ai-evaluation/images/receipt-01.png`: seller/date/currency, одна позиция, скидка10.00, total90.00 RUB |
| 7 | receipt-pdf | Локальный image-only PDF из неизменного `receipt-02.png`: одна страница, одна позиция, total100.00 RUB |
| 8 | receipt-unreadable | `receipt-19.png`: неизвестные обязательные поля; ожидается явный отказ от полного извлечения, без выдуманной суммы |

Manifest связывает original SHA, способ нормализации, все PNG/text/spans,
порядок страниц, размеры, golden SHA и версии renderer. Подготовка corpus
фиксирует Go1.24.0 из go.mod для воспроизводимых JPEG нормализации;
исходники и фактический request используют сохранённые approved bytes. Для receipt-pdf
локально создаётся воспроизводимый PDF, визуально проверяется и рендерится
тем же sandbox. Это новое публичное test fixture, не новый источник данных.
Пиксели/text модели отделены от golden: ожидаемые ответы и scorer не входят
в prompt или provider body. Старый S1.22 manifest с Astra/order8 не является
новым manifest и не диктует модель либо повтор случаев.

Выписки сравниваются по полному `MeasureDirect` и обязательным VisualReading
проверкам; формат ответа адаптируется без удаления полей или фильтрации ошибок.
Проверяются header/все строки, суммы/знак/валюта, описание, дата/время/precision,
FX, fee, own-transfer evidence, coverage и blockers. Full32-native сохраняет
**297 checks** прежнего scorer; full32-scan имеет то же сравнение значений,
но не вымышленное независимое native binding. Никаких first-N/prefix passes.

Для чеков читается полный expected из `synthetic.v1.json`: seller, issuedDate,
issuedAt, currency, total и каждая позиция name/quantity/unitPrice/lineTotal/
discount/brand/package/broadGroup. Decimal → minor/quantityMilli проверяет Go.
Null отличается от нуля; скидка не вычитается дважды. Старый `ai-eval` scorer,
сравнивающий только total/currency, недостаточен. Сетевой отказ/ошибка billing
на нечитаемом чеке не засчитывается как правильное распознавание unknown:
нужен сохранённый валидный ответ с явными неизвестными/неполнотой.

Recognition, protocol, native binding, image review и financial readiness
отражаются отдельно. Для фото/скана quotes/box проверяются структурно;
геометрия модели не превращается в независимый OCR. `importReady=false`;
owner review и confirmation в paid runner отсутствуют. Многопозиционные чеки,
200 строк, 4 страницы, плохие реальные фотографии и межбанковская обобщаемость
остаются за пределами этого небольшого измерения.

## Q2. Тот же gateway и экспериментальный допуск endpoint

Runner `cmd/document-reading-eval` использует `assistant.ModelGroups`,
`DocumentReasoner`, `ReceiptExtractor`, `DocumentRequest` и существующий
OpenRouter transport. Statement/receipt prompt/schema и лимиты G1–G4 не
переписываются для лучшего результата. Прямой Responses/TypeSafe/новый HTTP
клиент и обход identity/privacy/cost guards запрещены. `ai-eval` и закрытый
`statement-page-reading run` не превращаются в альтернативный live путь.

Catalog capabilities сами по себе не означают qualification. Для исследования
нужен ограниченный `experiment_only` допуск, связанный с approval/source/
manifest, одним model/provider, назначениями statement/receipt и сроком запуска.
Он создаёт только изолированные группы внутри исследовательского процесса.
Проверки modality/schema/context/ZDR/region/prices/returned identity остаются
теми же. Обычная API composition не принимает экспериментальный допуск как
production qualification; reserved research qualification нельзя загрузить
через обычный `GRACE_AI_MODEL_GROUPS`. Отдельные fake-тесты обязаны доказать
эту границу, включая истёкший/чужой допуск и подмену manifest.

Конкретная модель сейчас **UNSELECTED**, доступ и тариф **UNVERIFIED**.
Q1–Q3 не закрепляют прежнюю Astra или старую candidate-policy. До запроса Q4
оператор представляет один конкретный model ID/canonical/provider endpoint,
region/privacy, явные верхние тарифы и image bounds. Значения берутся из
публичных официальных сведений/операторского предложения; актуальная metadata
проверяется только внутри отдельно разрешённого Q4 preflight. Невозможность
подтвердить параметры расходует попытку без POST, а не выбирает другую модель.
Если кандидат не укладывается в потолок, запрос Q4 не готовится.

Metadata: не более **6 GET** за кампанию — catalog/ZDR/endpoint для двух
назначений; можно разделить общий proof и сделать меньше. Только выбранный
endpoint; без дополнительных поисковых запросов, generation GET и retry.
Proof не старше24 часов, состав группы и конкретный выбор фиксированы.
Протухший proof во время кампании останавливает её, без скрытого refresh.

Модели передаются все разрешённые страницы data URL и доступный настоящий
text. PDF локально рендерится перед сборкой bundle; сетевой parser/OCR plugin,
raw PDF, remote URL, tools/search и fallback не используются. Сборка включает
PNG/text и source hashes. Производственный helper во время Q4 не обновляется:
renderer проверен локально, его production rollout остаётся отдельным этапом.

## Q3. Учёт, one-shot runner и offline evidence

Предлагаемый потолок — **2 000 000 USD micros ($2.00)** суммарно, в том числе
все provider token/image/request fees; на один запрос ≤250 000 micros ($0.25).
Это лимит предложения, а не расход, котировка или увеличение прежнего allowance.
Максимум **8 POST / 6 metadata GET**, одна попытка, без model/provider switch.
Лимиты G1–G4 ≤30000 input /16000 output tokens и HTTP180 секунд сохраняются.
До допуска проверяется сумма консервативных резервов всех восьми случаев;
она должна укладываться и в $2.00, и в существующий доступный shared budget.

Live runner использует прежние PostgreSQL budget reserve/guard/settle под
существующим operator principal; локальный fake balance не заменяет реальный
allowance. Нельзя создать новый budget row, пополнить allowance, прочитать
финансовые таблицы либо использовать parallel/off-ledger расходы. Штатная
конкурентная защита shared allowance сохраняется. После каждого POST bill
settled либо unknown/frozen; до следующего вызова предыдущий settlement обязан
быть известен. Timeout, неизвестный bill, identity/privacy/price нарушение,
ошибка settlement или первый decisive quality REJECT останавливают кампанию.
Все оставшиеся случаи получают `not_sent` с причиной, а не фиктивный FAIL/PASS.

Предлагаемые исходники, которые разрешит принятие Q1–Q3:

- `internal/documentqualification/`: manifest, corpus/scorer/campaign/replay,
  закрытая проекция и причинные fake-контроли; общие components в assistant
  только для явно ограниченного experiment scope и наблюдения metering.
- `cmd/document-reading-eval`: `check`, `replay`, `summary` без сети/ключей;
  `run` выключен по умолчанию, требует exact-source/manifest/approval/execute
  и отдельного Q4. Сборка фиксирует source SHA; default check не читает env secrets.
- `.github/workflows/document-reading-eval.yml`: manual dispatch, execute=false,
  source ancestor main, CI5/5, attempt1, production concurrency, закрытый output.
- `deployments/production/document-reading-eval.sh`: pinned root SSH, staging
  только нового bundle/binary в текущий API container, новый lock/marker/report;
  без изменения image/config/service, restart, migrations и старых markers.
- Локальные fake SSH/SCP/Docker controls и tests/fixtures; отчёт в этом же пакете.

Это прямое разрешение на создание операционных **исходников** после принятия
Q1–Q3; создание рабочих keys/secrets и запуск этих файлов в него не входят.
Нового bootstrap не предлагается: существующие dedicated SSH key и operator
UUID остаются непрочитанными до отдельного Q4. Их текущая пригодность не
объявляется доказанной по историческому B4.

Q4 предложит ровно один dispatch с ≤2 root SSH command sessions и ≤1 SCP staging
при pinned host key; SSH transport соединения, их purpose и timeout входят
в manifest. CI сначала строит bundle/check без provider key; только разрешённый
remote run читает существующий OpenRouter credential внутри API container.
Ключи, DSN, UUID и env не выводятся и не сохраняются в bundle/report. SQL
ограничен существующими budget-функциями и обязательным owner context; business
users/финансовые таблицы не читаются. Operator consent ограничен публичными
документами manifest; пользовательские consents/financial drafts не создаются.

Новый marker создаётся до remote preflight с no-clobber под общим lock.
Допуск расходуется при dispatch, включая SSH/preflight failure до marker.
Обновление run attempt, повтор dispatch, удаление marker и старый P4 запрещены.
Raw report нового запуска сохраняется приватно на VPS; тот же binary делает
replay там же. В CI выходит только allowlisted JSON: ordinal/case IDs,
source/manifest/proof hashes, counts/check codes, finite timings/tokens/cost,
settlement и итог. Произвольный model content/stderr не публикуется. Локальный
replay использует публичные fake fixtures; новый private live payload не
скачивается. Staging удаляется штатным cleanup; новые marker/report сохраняются.

Итоги: `PASS_SYNTHETIC` только при завершении всех8 случаев и полном PASS
всех обязательных проверок; `REJECT` при измеренной ошибке чтения; `INCONCLUSIVE`
при невозможности честно измерить. Transport success не равен quality PASS.
Изменение scorer/golden/prompts/source после допуска требует нового решения,
результат прежнего запуска не переоценивается молча.

## Критерии приёмки Q1–Q3

| AC | Обязательное исполняемое evidence после принятия |
| --- | --- |
| Q1.1 | `go run ./cmd/document-reading-eval -mode=check` без ключей/сети:8 cases, hashes/page counts/goldens/порядок и $2/$0.25/8POST/6GET в manifest; native/image-only различены |
| Q1.2 | Полные correct fake responses →PASS_SYNTHETIC; пропуск строки32/позиций/валюты/скидки/времени/precision, лишняя строка, fabricated unknown →REJECT; full32-native297 checks не сокращены |
| Q2.1 | Actual grouped wire на fake HTTP для обоих назначений; golden не утёк в request; экспериментальный scope недоступен production config; нет experiment grant/consent →0POST |
| Q2.2 | Text-only/missing schema/ZDR/region/price/fees, aliases, stale proof и неверная returned identity →отказ; metadata≤6, POST≤8, нет retry/fallback; все страницы переданы |
| Q3.1 | Временная PostgreSQL: reserve/guard/settle reuse, известный bill при quality REJECT учтён, unknown freeze, concurrent cap, no financial deltas; default/replay0live calls/0secret reads |
| Q3.2 | Fake SSH/SCP/Docker/workflow tests: execute=false, attempt/source/CI/manifest mismatch, spent marker, lock, refusal до SQL, cleanup, interrupted SSH, stderr/model-text injection; повторов/новых secret writes0 |
| Q3.3 | Deterministic replay и bounded public projection; изменённый report/manifest/missing bill либо unknown на unreadable case не создаёт quality PASS |
| Q3.4 | `make test`, `make test-integration`, `make test-production`, `go vet ./cmd/... ./internal/...`, `git diff --check`; AC/evidence на русском, точный head/CI, live quality честно UNVERIFIED |

## Следующее решение владельца

**Q1–Q3 приняты владельцем**; описанная offline-реализация подготовлена в том
же PR. [AC/evidence](0056-qualification-evidence.md), implementation acceptance
пока PENDING. Q4, paid/metadata calls, чтение/настройка secrets,
SSH/VPS/production SQL, merge и deploy этим решением не разрешены.

После готового diff/evidence отдельно представляются приёмка/merge и затем
Q4: exact source/ref/manifest, concrete endpoint, rates/image bounds, восемь
документов, ≤8POST/6GET/$2.00, пути новых marker/report и transport limits.
Пока source/endpoint/rates для запуска не определены, согласия на него нет.
Успешное synthetic evidence будет основанием обсуждать дальнейшую проверку
и production rollout; оно не доказывает точность на произвольных документах.

## Принятие владельцем до реализации

2026-10-25 владелец ответил **«Принять Q1–Q3 и реализовать offline»** на
PR123/head `ab2f10bf8f08ffc9b6ea4598ba27bf708f96d2e2`. Разрешена вся
описанная Q1–Q3 offline-реализация в этом же PR, включая создание runner,
manual workflow/root wrapper и fake SSH/SCP/Docker проверки.
Q4/live provider/metadata calls, чтение/настройка secrets, production
SSH/SQL, merge и deploy не разрешены. Receipt фиксируется до нового кода.
