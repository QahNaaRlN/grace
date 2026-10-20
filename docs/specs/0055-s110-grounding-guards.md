# SPEC-0055 / S1.10: непрерывность описания и роли книжных остатков

- Дата: 2026-10-21, Asia/Yekaterinburg.
- Статус: accepted; implementation_status: local_verified (G1–G5 PASS).
- Acceptance record: 2026-10-21, прямое «Утвердить S1.10 и offline-реализацию».
  Разрешены G1–G5, external inference=0; без deployment/model switch.
- Worktree: `statement-grounding-guards`, ветка `codex/statement-grounding-guards`.
- Основание: [S1.9](0055-s19-evidence.md), [точный диагноз](../research/statement-quality/s18-saved-trace-diagnosis.md).

## Результат реализации

[G1–G5 evidence](0055-s110-evidence.md): Go guards и retained read/preview/confirm
проверены unit/real-helper/SQL/HTTP, без внешних inference. PR №82, интеграция и
CI head отдельно. S1.8 historical FAIL сохранён; улучшение LLM не заявляется.

## Проблема и пользовательский результат

S1.8 пропустил дефис описания, оставив его в service. Общая source coverage
получила matched, row_issues прошёл: пропуск обнаружил только golden scorer.
Для личной выписки golden описания нет. Выбор двух дефисов заголовка как
остатков существующий money parser отклонил; этот отказ исправлять не нужно.
Но `ValidateProposal` проверяет у книжного остатка наличие label и специальный
запрет available, а общей проверки смысла opening/closing label, аналогичной
TurnoverRoleBlockers, пока нет. Число под чужой подписью требует отдельной защиты.

Результат: неполное описание внутри одной печатной строки и неподтверждённая
роль распознанного книжного остатка становятся явными blockers до финансового
подтверждения. Исходная разметка и значения сохраняются для ручного исправления.
Это пакет проверки достоверности, не обещание улучшить ответы LLM или получить
новый PASS. Современные модели сравниваются позднее при неизменных новых guards.

## Scope

1. Две общие детерминированные проверки `internal/statementdata`.
2. Одинаковое применение при новой разметке, чтении retained review,
   preview и confirmation через существующий `applyStatementFlags` /
   `statementPlanGuards`. Проверки не пишут review при чтении.
3. Версионирование qualification/scorer при изменении правил validation;
   явное отделение исторического replay S1.9 от текущего профиля.
4. Synthetic real-helper, fake-provider, SQL/HTTP и owner-scope regressions.
   Никаких новых внешних inference или операторских dispatch.

## Non-goals

Не входят: автоматическое возвращение пропущенных фрагментов, удаление
неудобных предложений модели, подстановка остатков/арифметика вместо evidence,
расширение корпуса банков, OCR, новые модели/provider/catalog refresh, prompt,
request schema, aliases/redaction/planner/лимиты/цены/бюджет/fees, платный smoke,
переключение production policy/flags и deployment. Сохранённый FAIL S1.8 остаётся
историческим FAIL; исправленный вручную synthetic proposal не является качеством LLM.

## Текущее поведение и технические точки

- `internal/statementdata/layout.go:ground`: для description oneLine=false;
  выбор Word1,2,4,5 одной линии разрешён, Word3 может быть отнесён в service.
- `internal/statementdata/review.go:ValidateProposal`: service учитывается в
  source coverage; row.Description собирается только из выбранных fragments.
- `internal/statementdata/turnover_roles.go`: положительное подтверждение роли
  и геометрической близости есть для оборотов, действует также на retained evidence.
- `internal/imports/statement_repository.go:applyStatementFlags` вызывается для
  чтения; `statement_guards.go:statementPlanGuards` перечитывает source внутри
  транзакции preview/confirmation. Кешированный matched не является доверием.

## Предметные инварианты

### Непрерывность description

Для последовательных выбранных spans одной и той же Page/Block/Line их Word
должны следовать подряд. Пропуск Word внутри выбранного интервала добавляет
`description_evidence_gap` в Row.Issues и `row_N:description_evidence_gap` в blockers.
Проверяется каждый сегмент description отдельно; переход на другую печатную
линию, блок или страницу сам по себе не является разрывом. Геометрия и исходный
порядок остаются проверяемыми прежним grounding; IDs не сортируются и не дополняются.

Если отсутствует надёжная сохранённая description evidence/геометрия при
непустой исходной source-description, добавить `description_evidence_unverified`,
не достраивать её из итогового пользовательского текста. У retained evidence
используются сохранённые Page/Block/Line/Word исходных spans; поля с нулевой или
противоречивой геометрией не доказывают непрерывность. Новые коды дедуплицируются.

Проверка ловит внутренние пропуски, включая `-`, `+`, слова и placeholders.
Она не доказывает наличие первого/последнего слова или целой пропущенной линии:
эти случаи явно остаются вне гарантии. Пустое unknown description не заполняется.
Не включать соседнюю сумму/дату в description автоматически. Существующие
ручные правки итоговых import rows остаются отдельными от source mapping;
проверяется source evidence, не совпадение отредактированного merchant text.

### Роли opening/closing book balance

Проверять каждый успешно распознанный non-nil book balance по его value и label
Evidence, а не по совпадению сумм. Missing/unrecognized/borrowed label даёт
`openingBookBalance:book_balance_role_unverified` либо соответствующий closing code.
BookBalance.Status становится incomplete; parsed values/evidence сохраняются.
При nil после ошибочного money parsing прежний ambiguous_money сохраняется;
не добавлять фиктивный value ради проверки роли и не снимать старые blockers.

Начальный конечный словарь, после существующей нормализации регистра/пунктуации:

| Role | Явные положительные labels |
| --- | --- |
| openingBookBalance | opening book balance; opening balance; начальный остаток; входящий остаток; остаток на начало периода |
| closingBookBalance | closing book balance; closing balance; конечный остаток; исходящий остаток; остаток на конец периода |

Label должен быть локально связан с value: та же страница и существующая
геометрическая политика turnoverLabelNearValue, без заимствования с другого
заголовка/страницы. При полном layout близкий БИК/КПП/ИНН/BIC/KPP/INN/SWIFT
отклоняет роль по прежнему правилу оборотов. Available balance, обороты,
SYNTHETIC BANK и числовые идентификаторы не подтверждают book balance.
Неизвестные синонимы остаются unverified, не принимаются по LLM confidence
или rolesConfirmed=true. Когда printed book balance не доказан, поле можно
оставить unknown; строки импорта сохраняются для обычного ручного review.

### Повторная проверка и финансовое подтверждение

Новые проверки применяются к свежему и сохранённому Review. Очищенный в старом
JSON blockers или прежний matched не обходят проверку. Preview и confirmation
проверяют актуальную source revision внутри существующей owner-scoped транзакции;
при новых blockers подтверждение отклоняется, postings/events не появляются.
Чтение не переписывает persisted review/rows/revision. Published financial
records автоматически не пересматриваются и не исправляются этим пакетом.

Недостаточная retained evidence даёт видимый unverified и ручной путь
повторной разметки/загрузки; удалённый layout не восстанавливается догадкой.
Точное поведение должно быть покрыто SQL tests для expiry и старого review.

## Версии и исторические reports

Правила validation меняются: обновить `ScorerVersion` и `ScreeningScorerVersion`
соответственно на `statement-fixed-corpus.v3.grounding` и
`statement-screening.v2.grounding19-20-23`. Счёт checks19/20/23 и сами денежные,
датные и description equality assertions не ослабляются. Версионирование
qualification не меняет согласие на передачу данных; новых отправок нет.

Старые qualification reports не получают текущий PASS. S1.9 original D5
остаётся воспроизводимым на зафиксированном source ecfa850; private report,
projection/replay и старые inspector pins не переписываются. При текущем новом
профиле replay v1 явно возвращает UNVERIFIED из-за несовместимости версии,
а не пересчитывает исторические 17/19 как будто это тот же контракт. В тестах
сохранить старые данные и отдельно доказать новые blockers фактического trace.
Оператор не должен обращаться к provider ради такой проверки.

Публичный summary добавляет только закрытые новые reason codes, включая
безопасно разобранный row_N prefix; raw model text/IDs/errors не выводятся.
Новые данные не присваивают модели qualified status и не разрешают activation.

## API, UI, данные, миграция и rollback

Новые endpoints/DTO/SQL columns не нужны: используются существующие Issues /
Reconciliation.Blockers и manual review. OpenAPI проверить на совместимость;
если API-контракт всё же меняется, остановить расширение scope и вернуть draft
на review. Денежные суммы, валюты, source mapping, evidence и импортируемые
строки не изменяются guard-функциями. Текущий Web отображает новые коды через
человеческие подписи существующих сообщений, без бизнес-правил в React.

Чтение/preview не выполняют backfill; migration отсутствует. Runtime deployment
не входит в пакет. Rollback кода снимает новые проверки в этом релизе, но не
переписывает review, audit, historical reports или финансовые записи. Rollback
не разрешает подтверждать ранее заблокированный импорт без проверки владельца.

## Ошибки и негативные сценарии

Проверить внутренний пропуск в service и полностью unmapped; пропуск слова и
пунктуации; корректный split signed amount; несколько линий/страниц/колонок;
поддельный bank heading, available label и swapped opening/closing; соседний
идентификатор с тем же числом; отсутствующий layout, corrupted/partial retained
Evidence; старый cached matched; owner isolation, revision race и ручные edits.
Все ошибки остаются видимы. Никаких auto-repair, retries или финансовых writes.

## Критерии приёмки и исполняемые проверки

| ID | Наблюдаемое подтверждение |
| --- | --- |
| G1 | Unit + real-helper: внутренний description gap даёт row issue/blocker; известный S1.8 counterexample обнаруживается общим validator. Полные descriptions и законные переносы/колонки/знаки суммы PASS; source proposal неизменен |
| G2 | Opening/closing role matrix EN/RU: положительные labels/geometry PASS, bank/available/wrong role/page/identifier/unknown FAIL; значения и исходные evidence не меняются; прежние ambiguous_money сохранены |
| G3 | SQL/HTTP: новый и legacy review блокируются в read/preview/confirm; forged matched/rolesConfirmed и отсутствующий layout не дают обход; no-write read и zero postings/events при отказе, owner/revision/manual-edit regressions PASS |
| G4 | Новый profile не принимает старые reports; historical S1.9 artifacts неизменны, replay v1 честно UNVERIFIED; canonical public summary sanitization PASS. Fake-provider full columns19/cards20/RU23 PASS, сильные scorer assertions сохранены |
| G5 | make test, make test-production, scripts/test-bank-statements.sh, go vet и git diff --check PASS; external model requests=0, deployment/model switch=0 |

## Решение владельца

Утверждена реализация G1–G5 как один offline-пакет.
Принятие фиксируется отдельным коммитом до production-кода. Новые
model calls, smoke budget, deployment и production model switch не входят.
Сравнение современных моделей — следующий контракт с обновлённым профилем,
свежей metadata и отдельным бюджетом; SPEC-0056 registry пока остаётся draft.
