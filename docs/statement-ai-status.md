# Statement AI: текущее состояние и следующее решение

Обновлено после принятия отчёта P4/PR116 и принятия A1–A3 offline, 2026-10-24 Asia/Yekaterinburg.
[История](specs/0055-development-history.md); [полный S1.20 пакет](specs/0055-s120-clock-reject-diagnosis.md).

| Вопрос | Подтверждённое состояние |
| --- | --- |
| Интеграция | S1.19 offline PR107, evidence/контракт S1.20 PR108, review fix PR109, reader PR110, отчёт PR111, принятый manual evidence PR112 и принятые S1.21 D1–D3/PR113, P1–P3/PR114, root fix/PR115, P4 отчёт/PR116/main ee7c0ec; exact-main CI40038147418/all5 и Delivery40039113963/all4 PASS |
| S1.19 | Единственный pilot run39983522098:6/6 completed, clocks `[0,4,4,0,0,4]`, итог **REJECT**, без принятого candidate |
| R4 | Единственный run40002091161/attempt1, source1a6b742, **VERIFIED**, replay6/6, все120 checks/3 pairs; разрешение **CONSUMED** |
| Точная причина REJECT | Первая пара, candidate ordinal2, вторая строка: `record_description` и `row_issues` ухудшились с base PASS до candidate FAIL; `non_clock_regression` |
| Время/precision | Candidate4/4 clocks и4/4 precision во всех трёх парах; это не причина REJECT |
| Общие проблемы | `coverage` и `blockers` FAIL во всех6 ответах; quality scores base14/20, candidate16/20,18/20,18/20, без full PASS |
| Accounting S1.19 | Сохранённые bills `[3032,3117,1437,2999,1446,1449]`, actual13480 USD micros ($0.01348), known/settled; без fresh SQL/fees/allowance proof |
| Дополнительные действия R4 | Provider/metadata/SQL calls0, persistentWritesRequested0, activationAuthorized=false; новые model calls/deploy/flags/financial writes не выполнялись |
| S1.21 | D1–D3/P1–P3/root fix приняты и интегрированы. Correct fake controls19/19,20/20,23/23; единственный P4 **INCONCLUSIVE**, H2 UNVERIFIED |
| Разрешения | E6/Q6/S1.19/R4/P4 **CONSUMED**. P4/run40035544723/attempt1: SSH public-key refusal, report unavailable; calls/cost/settlement N/A. Повтор/credential repair/activation не разрешён |

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
В том же S1.21 принят **A1–A3 offline / implementation prepared for review**: offline source
helper/dedicated key/auth-only workflow и одна secret reference нового pilot.
Общий deploy key сохраняется. Прямое принятие владельца зафиксировано receipt1921406 до кода. Helper/auth workflow
и fake controls подготовлены; рабочие ключи не создавались, live B4 не выполнялся.
Actual bootstrap/auth-only B4 и новый paid P4 требуют отдельных exact-source
допусков; принятие A1–A3 разрешает только offline source и fake tests.
Масштабирование обязательно до product-интеграции, но сначала полезно выяснить,
работает ли H2 на двух строках; больше fake/96-row planner не заменяют измерения
реальной модели. После любого результата — остановка, без автоматического повтора.

Manual path и deterministic Go guards независимы от AI. Исправление актуальности
owner review PR109 интегрировано. [Полный ручной UI-сценарий](specs/0055-manual-import-ui-evidence.md)
проверен Codex и отчёт принят владельцем/интегрирован PR112 на публичном columns.pdf без LLM: purchase+own transfer,
confirmation/reload/повтор файла, events2/postings3 и точные остатки.
Это ограниченное synthetic evidence; owner functional acceptance и надёжный
AI-импорт ещё не доказаны.
