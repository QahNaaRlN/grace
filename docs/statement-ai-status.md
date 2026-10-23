# Statement AI: текущее состояние и следующее решение

Обновлено после интеграции отчёта R4 PR111 и локальной проверки полного ручного импорта, 2026-10-23 UTC.
[История](specs/0055-development-history.md); [полный S1.20 пакет](specs/0055-s120-clock-reject-diagnosis.md).

| Вопрос | Подтверждённое состояние |
| --- | --- |
| Интеграция | S1.19 offline PR107, evidence/контракт S1.20 PR108, review fix PR109, reader PR110 и принятый отчёт PR111/main9899182; source/main CI5/5, Delivery4/4 и post-merge gates PASS |
| S1.19 | Единственный pilot run39983522098:6/6 completed, clocks `[0,4,4,0,0,4]`, итог **REJECT**, без принятого candidate |
| R4 | Единственный run40002091161/attempt1, source1a6b742, **VERIFIED**, replay6/6, все120 checks/3 pairs; разрешение **CONSUMED** |
| Точная причина REJECT | Первая пара, candidate ordinal2, вторая строка: `record_description` и `row_issues` ухудшились с base PASS до candidate FAIL; `non_clock_regression` |
| Время/precision | Candidate4/4 clocks и4/4 precision во всех трёх парах; это не причина REJECT |
| Общие проблемы | `coverage` и `blockers` FAIL во всех6 ответах; quality scores base14/20, candidate16/20,18/20,18/20, без full PASS |
| Accounting S1.19 | Сохранённые bills `[3032,3117,1437,2999,1446,1449]`, actual13480 USD micros ($0.01348), known/settled; без fresh SQL/fees/allowance proof |
| Дополнительные действия R4 | Provider/metadata/SQL calls0, persistentWritesRequested0, activationAuthorized=false; новые model calls/deploy/flags/financial writes не выполнялись |
| Разрешения | E6/Q6/S1.19/R4 CONSUMED. Повторы, repair, новая версия prompt/model или pilot не открыты |

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

**Исследовательский пакет остановлен после VERIFIED.** Следующее решение должно
отдельно определить исправление сохранности описания и общих coverage/blockers.
Новый prompt, model call или ремонт candidate автоматически не начат.

Manual path и deterministic Go guards независимы от AI. Исправление актуальности
owner review PR109 интегрировано. [Полный ручной UI-сценарий](specs/0055-manual-import-ui-evidence.md)
проверен Codex на публичном columns.pdf без LLM: purchase+own transfer,
confirmation/reload/повтор файла, events2/postings3 и точные остатки.
Это ограниченное synthetic evidence; owner functional acceptance и надёжный
AI-импорт ещё не доказаны.
