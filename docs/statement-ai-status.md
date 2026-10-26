# Statement AI: текущее состояние и следующее решение

Обновлено 2026-10-26, Asia/Yekaterinburg.

Текущая работа — [SPEC-0058](specs/0058-real-document-import.md), **accepted /
offline implementation verified**: простой запрос → редактируемая таблица → единая
сверка → прежний preview/confirmation. Ветка `codex/real-document-import`, source `0dad84f`; [AC/evidence](specs/0058-evidence.md).
Research SPEC-0057 дал 96 строк/768 совпадений на одном реальном PDF; новый Go parser
повторил эти проверки offline по сохранённому ответу. Эталон остаётся черновым.
Новых платных вызовов, production activation и записей в реальные финансы нет.
Photo/scan live quality и deployed usability не подтверждены.

Обнаружен и воспроизведён дефект helper image: отсутствие fallback-шрифтов
давало белые страницы для PDF без встроенных шрифтов. Исправление добавляет
закреплённый DejaVu; tests проверяют реальную непустую картинку. Старые verdicts
не пересчитываются автоматически: потеря времени в ответе остаётся отдельным фактом.

## Историческая закрытая серия SPEC-0056

**Последний Q4.5: REJECT по качеству.
Серия закрыта3/3,remaining0. PASS_SYNTHETIC не получен.**

[SPEC-0056 G1–G4](specs/0056-openrouter-document-reading.md) и Q1–Q3
интегрированы PR122/123. Прежние model/canonical/provider/8documents/criteria сохранены.
Принятый [V](specs/0056-last-attempt-portable-evidence-candidate.md), acceptance6165d81 до кода,
интегрирован PR132/main0a955148 после всех local gates и PR CI40227597292/5;
source mainCI40228543312/5 PASS. V удаляет только S1 bounds из model request schema;
prompt v3 и Go semantic validation/oracle/scorer/corpus unchanged.

[Q4.5/run40229482580](specs/0056-q45-execution-evidence.md),attempt1/count1:
**HTTP200/protocol PASS**,21/27 recognition checks PASS,6fail, nativeBinding fail.
Четыре date fields сохранили calendar date, но потеряли напечатанное время;
row2 description=null,issues nonempty. Не установлено, почему модель пропустила описание.
Остальные7cases не отправлены по decisive_quality_reject; full32/receipts live UNVERIFIED.
1POST/6GET,5631input/2117output,actual$0.005042 settled,unknownBills0.
Read-only fixed-file diagnosis1SSH/0SQL/key/provider/writes,closed projection и hashes сохранены.

Q4.3/Q4.4 original INCONCLUSIVE summaries сохранены; административные U/V settlement0
имеют directMetadataVerified=false и отдельные основания. Их полный reserve$0.062352
остаётся contingency внутри прежних$2. Q4.5 имеет обычный known settlement, policy0 не применялся.
[Journal](research/statement-quality/evidence-series-2026-10-25.json): consumed3/3,
remaining0,conservative accounted$0.067394. Q4.2 bill$0.005183 вне новой серии.
Реальный quality PASS не заявлен. Новые retry/series/model/activation/import/deploy не открыты.
[V AC/evidence](specs/0056-v-portable-evidence.md).
[История](specs/0055-development-history.md).

## Исторические S1.19–S1.22

S1.22 V1–V3 интегрирован PR120/main e9d02b2; mainCI40118781180/5 и
Delivery40119558599/4 PASS. Прямой OpenAI V4 не открывался. Данные ниже относятся
к прежним пакетам; они не заменяют актуальное состояние SPEC-0056 выше.

| Вопрос | Подтверждённое состояние |
| --- | --- |
| Интеграция | S1.19 offline PR107, evidence/контракт S1.20 PR108, review fix PR109, reader PR110, отчёт PR111, принятый manual evidence PR112 и принятые S1.21 D1–D3/PR113, P1–P3/PR114, root fix/PR115, P4 отчёт/PR116 и A1–A3/PR117 и отчёт B4/PR118 и новый P4/PR119/mainad1eca3; exact-main CI40111553326/all5 и Delivery40112288231/all4 PASS |
| S1.19 | Единственный pilot run39983522098:6/6 completed, clocks `[0,4,4,0,0,4]`, итог **REJECT**, без принятого candidate |
| R4 | Единственный run40002091161/attempt1, source1a6b742, **VERIFIED**, replay6/6, все120 checks/3 pairs; разрешение **CONSUMED** |
| Точная причина REJECT | Первая пара, candidate ordinal2, вторая строка: `record_description` и `row_issues` ухудшились с base PASS до candidate FAIL; `non_clock_regression` |
| Время/precision | Candidate4/4 clocks и4/4 precision во всех трёх парах; это не причина REJECT |
| Общие проблемы | `coverage` и `blockers` FAIL во всех6 ответах; quality scores base14/20, candidate16/20,18/20,18/20, без full PASS |
| Accounting S1.19 | Сохранённые bills `[3032,3117,1437,2999,1446,1449]`, actual13480 USD micros ($0.01348), known/settled; без fresh SQL/fees/allowance proof |
| Дополнительные действия R4 | Provider/metadata/SQL calls0, persistentWritesRequested0, activationAuthorized=false; новые model calls/deploy/flags/financial writes не выполнялись |
| S1.21 | D1–D3/P1–P3/root fix приняты и интегрированы. Correct fake controls19/19,20/20,23/23; первый P4 INCONCLUSIVE, новый после B4 — **REJECT** на grouping/partition_invalid. Row-field quality UNVERIFIED |
| Разрешения | E6/Q6/S1.19/R4/P4 **CONSUMED**. P4/run40035544723/attempt1: SSH public-key refusal, report unavailable; calls/cost/settlement N/A. B4 VERIFIED/CONSUMED. Новый P4/run40109843789 REJECT/CONSUMED; repair/repeat/activation не разрешены |

R1–R3 приняты владельцем и интегрированы PR110 до R4. Exact source
`1a6b74201db92ba37be750411ff7883d8216b602`, ref `codex/statement-s120-reader-source`,
sourceCI39998179098/mainCI39999830331, Delivery40001135152 проверены.
R4 approval receipt `d4ca1af280363972bd423977a6a1dad310b59eaa` опубликован **до** dispatch.
Результат, source/manifest/approval/run/digest находятся в одном пакете;
отдельной readiness/result спецификации для каждой записи нет.

[Полная закрытая projection](research/statement-quality/s120-diagnosis-summary.json)
имеет SHA256 `ecadd5ea938f485662c2dd9d9c25181454621a99d390f13f619bdd7ef920ad6a`;
CI digest и повторные Go/Python validators совпали. Private payload не скачивался.
Коды не раскрывают конкретные потерянные слова/причины row issues или spans coverage;
эти сведения и мысли модели не выдумываются. Producer linkage не является
криптографической аттестацией происхождения файла.

**S1.20 завершён после VERIFIED.** Владелец отдельно принял
[S1.21 D1–D3](specs/0055-s121-row-scoped-selection.md): сначала envelopes/header,
затем поля отдельной строки. Offline prototype в PR113 использует прежний полный
scorer и проходит три correct public controls с fake clients; это проверка
исполнимости протокола, не качества модели. Реальных вызовов и qualification нет.
Количество calls/theoretical cap втрое выше base; будущая полезность H2 неизвестна.
Владелец отдельно ответил «Принять реализацию и слить №113» для source
f5024aa; merge05221d9 сохраняет решение. D1–D3 завершены; цель надёжного AI-импорта
не доказана. В том же документе принято дополнение P1–P3 offline для одного
измерения H2:3 пары A/cards,≤12 POST, прежний theoretical provider ceiling
646800 micros плюс fees в существующем бюджете. P4 требует отдельного допуска
на готовый source. Offline-реализация P1–P3 подготовлена в PR114: fixed adapter/CLI,
aggregate campaign, replay/closed result, manual workflow/wrapper; до отдельного P4 запуска не было.
Владелец принял реализацию и слияние PR114: merge8a3637b, exact-main CI/all5 и
Delivery/all4 PASS. Отдельно разрешённая root UUID setup CONFIGURED: один read-only
users SELECT, запись только production operator UUID secret с metadata-проверкой;
значение не выводилось, существующие secrets не читались, БД не изменялась.
По уточнению владельца grace-deploy отсутствует; root SSH/SCP source fix принят
и интегрирован PR115/main6dc08d3, exact-main CI/all5 и Delivery/all4 PASS.

Владелец отдельно разрешил один P4 на023ad38. [Run40035544723](https://github.com/QahNaaRlN/grace/actions/runs/40035544723),
attempt1, завершён failure на SSH staging: public-key authentication refusal/exit255.
[Проверенная закрытая projection](research/statement-quality/s121-pilot-summary.json):
**INCONCLUSIVE / row_pilot_report_unavailable**, replayVerified=false,
calls/cost/settlement=null (N/A). Качество модели не измерено; это инфраструктурный
отказ, не REJECT H2. Локальная root-аутентификация при UUID setup не доказывала
пригодность CI credential. Точная причина отказа ключа не установлена.
[Run/artifact/hash evidence](research/statement-quality/s121-pilot-execution.json)
и [public manifest](research/statement-quality/s121-pilot-manifest.json) сохранены
в том же S1.21 пакете. Допуск **CONSUMED**, повторов/дополнительного SSH нет.
Отчёт отдельно принят владельцем и интегрирован PR116/main ee7c0ec;
exact-main CI40038147418/all5 и Delivery40039113963/all4 PASS, без deploy.
В том же S1.21 A1–A3 приняты и интегрированы PR117/main2059b49,
source CI40104638874/all5, main CI40105683044/all5 и Delivery40106473989/all4 PASS.
Владелец отдельно разрешил **один B4** на source41ad69f, receipt опубликован
в PR117 до bootstrap. Bootstrap CONFIGURED: новый dedicated key/secret,
installSSH1/rollbackSSH0. [Run40107076865](https://github.com/QahNaaRlN/grace/actions/runs/40107076865),
attempt1: **VERIFIED**, один auth-only CI root SSH; все шаги success, включая cleanup.
[Closed result](research/statement-quality/s121-access-result.json),
[manifest](research/statement-quality/s121-access-manifest.json) и
[run/hash evidence](research/statement-quality/s121-access-execution.json) проверены.
CI manifest побайтно совпал с approved; root key/secret сохраняются по назначению.
Допуск B4 **CONSUMED**, отчёт принят и интегрирован PR118/main4e21431;
main CI40108414891/all5 и Delivery40109156224/all4 PASS. B4 подтвердил только
аутентификацию; последующее выполнение pilot отражено ниже.
Новых provider/model-metadata/SQL/budget calls или deploy не было.
Владелец затем отдельно разрешил один новый P4 на source9202641, frozen ref
`codex/statement-s121-row-pilot-after-access-source`; receipt в PR118 до dispatch.
[Run40109843789](https://github.com/QahNaaRlN/grace/actions/runs/40109843789),
attempt1, completed/success: **quality decision REJECT**. Base completed14/20;
candidate rejected на grouping/`partition_invalid`, до row-field вызовов.
Ordinal3–6 not_sent/row_pilot_decisive_reject. Это отказ candidate в первой паре,
преимущество H2 не подтверждено; качество row-field extraction не измерено.
[Closed summary](research/statement-quality/s121-pilot-after-access-summary.json),
[public manifest](research/statement-quality/s121-pilot-after-access-manifest.json),
[run/hash evidence](research/statement-quality/s121-pilot-after-access-execution.json).
Actual2 POST/3 metadata GET, known bills2, actual5509 USD micros ($0.005509),
BillingKnown/ReservationSettled/SettlementRecorded=true. ReplayVerified=true:
2 measurement positions +4 refusal-consistency suffix positions, не6 completed answers.
Closed summary не раскрывает конкретный дефект partition; aliases не выдумываются.
Новый допуск **CONSUMED**, отчёт принят и интегрирован PR119/mainad1eca3; repair/repeat/activation не открыты.
Старый P4/run40035544723 остаётся INCONCLUSIVE/accounting N/A; он не перезаписан.
Владелец поручил сменить курс. [S1.22 V1–V3](specs/0055-s122-direct-page-reading.md)
предлагает прямое чтение PNG страниц+текста одной vision-моделью: обычные строки
и дословные quotes, deterministic binding/старые financial guards после распознавания.
Candidate GPT-6 Astra пока UNVERIFIED;4 cases включают image-only control и full32.
Контракт **accepted V1–V3 offline**, implementation **accepted / integrated PR120**, main `e9d02b2`; [offline evidence](specs/0055-s122-offline-evidence.md). Четыре fake controls PASS, full32:32 строки/3 страницы/297 recognition checks; importReady=false. Paid V4/keys/deploy не открыты; merge PR120 отдельно принят и выполнен.
Возможный будущий V4:≤8 POST, предлагаемый ceiling $10.40, новый явный бюджетный допуск.
Старый grouping-подход закрыт REJECT; фейки/зелёный CI не заменяют измерения качества.
Даже успешный новый corpus не докажет real/full import/owner acceptance автоматически.

Manual path и deterministic Go guards независимы от AI. Исправление актуальности
owner review PR109 интегрировано. [Полный ручной UI-сценарий](specs/0055-manual-import-ui-evidence.md)
проверен Codex и отчёт принят владельцем/интегрирован PR112 на публичном columns.pdf без LLM: purchase+own transfer,
confirmation/reload/повтор файла, events2/postings3 и точные остатки.
Это ограниченное synthetic evidence; owner functional acceptance и надёжный
AI-импорт ещё не доказаны.
