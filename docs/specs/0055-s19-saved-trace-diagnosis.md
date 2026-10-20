# SPEC-0055 / S1.9: диагностика сохранённого S1.8 без inference

- Дата: 2026-10-21, Asia/Yekaterinburg.
- Статус: draft; implementation_status: not_started.
- Acceptance record: отсутствует; требуется review нового operator/workflow path.
- Worktree: `statement-trace-diagnosis`; ветка `codex/statement-trace-diagnosis`.
- База: main `5e9116e9c064c5129049414a060f340acc000700`, PR №80.

## Проблема и пользовательский результат

[S1.8](0055-s18-evidence.md) дал columns17/19 с record_description/blockers.
Точный дефект не установлен. Полный синтетический trace и validation сохранены
приватно, однако самостоятельного read-only workflow для их диагностики нет.
Существующий ordinary inspect читает старые statement.synthetic reports;
inspector S1.8 выдаёт агрегаты, запускается после paid runner и не выдаёт trace.
Повторное включение paid mode ради чтения report недопустимо.

Результат: воспроизвести текущими Go validation/scorer сохранённое предложение
на прежнем синтетическом PDF, назвать конкретные поля/ссылки и причины отказа
либо зафиксировать UNVERIFIED при недостаточных данных. Это диагностика
исторического результата, а не новая qualification или разрешение activation.

## Scope и non-goals

Scope: самостоятельный строго read-only operator path для фиксированного S1.8,
закрытая проекция locator/reason metadata, локальный replay и отчёт о причине.
Разрешение реализации должно включать один диагностический dispatch после
offline privacy/exclusivity gates; повторное чтение допустимо при техническом
сбое, без запуска screening. Число provider inference calls всегда ноль.

Не входят: модельные POST/GET metadata, provider key, catalog refresh, budget
reservation/settlement, снятие uncertain, изменение marker/report, paid runner,
production config/model/flags/image, личные PDF/drafts, финансовые записи,
prompt/planner/scorer/transport/redaction/schema и новый платный smoke.
Исправление выявленного дефекта потребует отдельного принятого контракта.

## Текущее поведение и доказательства

[Статический разбор](../research/statement-quality/s18-saved-trace-diagnosis.md).
Все проверки кроме одного record_description и общего blockers прошли.
Оба row_issues прошли: текущих данных недостаточно, чтобы связать blockers
с ошибкой grounding описания. Два публичных unknown маскируют причины, а не
неизвестный расход. Billing был known/settled при чтении 03:10:03 +05:00.
Cards/RU не отправлены. Исторический FAIL и consumed authorization сохраняются.

## Предметная модель и инварианты

1. Только `/opt/grace/shared/statement-screening-s18-gemini31-identity-20261021.json`
   и соответствующий `.started`. Нет glob, произвольного пути или выбора
   последнего report. Marker должен содержать exact source
   `f450e9d629a4543821a21b8f7b3aa679788872c4`.
2. Чтение O_RDONLY/O_NOFOLLOW, regular0600, marker<=128 bytes, report<=524288;
   ограниченные arrays/depth, строгий JSON без duplicate keys/trailing values.
   Ошибки дают закрытый unavailable/UNVERIFIED без raw exception или partial output.
3. Проверить version v3, прежние profile/policy/pin, timestamps, форму единственного
   scored columns и corpus fixture/document/plan hashes. Не выдавать FAIL report
   за PASS qualification; ограничение свежести activation к историческому replay
   не применять, source identity и исходные timestamps остаются обязательными.
4. Вывод по явному allowlist: фиксированные case/field/role enums, ordinal,
   canonical source locators, direction debit/credit, замкнутые reason codes,
   bool/counts и проверенные hashes синтетического корпуса. Row/header причины
   разбирать на закрытую пару field/ordinal+reason; произвольный текст заменить
   unknown и count, без печати оригинала. Не экспортировать service.reason,
   source/model/error text, generation/owner/reservation IDs, tokens или cost.
   Порядок selections и численность строк сохранять, без молчаливого усечения.
5. Проекция содержит все структурные selections rows/header/labels/service для
   воспроизведения. Service reason при replay заменить фиксированным synthetic
   значением: нынешний validator проверяет только непустоту/длину; обе проверки
   отдельно подтвердить при чтении. При неизвестном scalar/schema — UNVERIFIED.
6. Локально извлечь только прежний репозиторный columns.pdf pinned helper
   Poppler25.04.0 в network=none, сверить fixture/document/plan hashes. Все locators
   должны существовать в этом документе; иначе stop/UNVERIFIED. Сохранить
   порядок/дубли выбора, затем прежние MergeProposals, ValidateProposal и
   ScoreScreening. Replay не вызывает reasoner/meter и не пишет в PostgreSQL.
   Не менять exported API продукта ради диагностического helper.
7. Сверить replay с сохранёнными checks/correct/codes/validation/coverage.
   Разобрать expected/actual description по известному публичному fixture локально;
   независимо разобрать каждый blocker. При несовпадении не делать вывод о причине.
8. Новый отдельный workflow mode default=false, statement_action=preserve;
   другие smoke/inspect/reconcile/deploy jobs исключены при любом конфликте.
   Сохранить production concurrency и блокировку конкурентных операторов.
   Использовать защищённый SSH, Python через stdin; не копировать private report,
   не читать provider key, не устанавливать binary и не запускать приложение.
   Все remote действия — чтение файлов; без SQL, Docker mutations и provider сети.
9. После чтения проверить неизменность report/marker (hash/size/mode), вывести
   только boolean. Сохранить прежние artifacts и отчёт о consumed authorization;
   диагностический dispatch не создаёт нового права на inference.

## API, UI, данные и rollback

Публичный API/OpenAPI/UI и migrations не меняются. В репозитории добавляются
только оператор диагностики, offline replay/helper, тесты и evidence.
Приватные исходные artifacts остаются на VPS. Локальная sanitized projection
не содержит личных данных или billing. Rollback — возврат нового operator path;
старые report/marker/ledger и исторические результаты не изменяются.

## Ошибки и негативные сценарии

Missing/symlink/public/oversize/неверный source/version/profile/hash, дубли JSON,
malformed locator/scalar, неверный corpus или несовпадение replay => stop и
UNVERIFIED. Отсутствие report не означает, что вызова не было. Unknown reason
не превращается в guessed code. Privacy sentinel в любом текстовом поле не
должен появиться ни в stdout, ни в stderr или Actions logs.

## Критерии приёмки и исполняемые проверки

| ID | Проверка |
| --- | --- |
| D1 | Offline fixtures: неправильные paths/modes/symlinks/source/JSON/hashes/schema отклоняются; sentinel privacy, bounded output и artifact immutability PASS |
| D2 | Все сочетания фактических workflow job conditions взаимоисключены; новый mode default=false; ни paid runner, ни deploy/reconcile/старый inspect не вызываются |
| D3 | Real-helper replay на synthetic counterexamples воспроизводит record_description и независимые blockers; no reasoner/meter/SQL/provider network; прежние scorer и validator неизменны |
| D4 | make test, затронутые production mocks, go vet, git diff --check PASS; source diff подтверждает отсутствие продуктовых изменений |
| D5 | Один read-only dispatch: exact source/report/fixtures verified, неизменность artifacts PASS; replay совпал либо результат честно UNVERIFIED; причина по конкретным selections/reasons без нового inference |

## Открытое решение владельца

Утвердить S1.9: реализацию D1–D4 и read-only D5. Платных вызовов, деплоя,
переключения модели и исправления prompt/scorer/transport это не разрешает.
До принятия production/operator implementation не начинается.
