# SPEC-0055 / S1.19: один эксперимент о полноте напечатанного времени

- Статус: **accepted** (C1–C4 offline); implementation_status: **integrated offline**.
- Автор: Codex; дата: 2026-10-23, Asia/Yekaterinburg.
- Владелец принял изменение программы прямым «Хорошо, измени дальнейшую работу
  и продолжай» после ретроспективы.
- Acceptance record: 2026-10-23, прямое «Принять, интегрировать и продолжить offline»
  на C1–C4 PR106, review head `c9b621dc088402d8ef419e9427f818f64db02ff8`.
  Владелец разрешил зафиксировать принятие, слить PR после CI5/5 и продолжить
  offline-реализацию. Live pilot требует отдельного exact-source разрешения.
- Spec base: main `1a5bbfbefecd6e8983a8c290f6e8c650c62351dc`, [PR105](0055-pr105-postmerge.md).
- Implementation base: main `db65225c6d2cfc9052ccf4f7310ced61c5ba79d8`, PR106;
  [C1–C4 offline evidence](0055-s119-evidence.md).
- Spec branch: `codex/statement-s119-repair-spec`, PR №106 (merged).
- Implementation branch: `codex/statement-s119-clock-pilot`; code `4668d1309aea156d96b03938b56fd43a30a4470f`.
- Предыдущая редакция о clocks/claims/coverage заменена до принятия. Реализация
  начата после принятия и интеграции спецификации PR106/main db65225.

## Проблема, гипотеза и полезный результат

[Q6](0055-s118-q6-evidence.md) установил у A/cards четыре совпадения date и UTC,
но четыре расхождения time/precision. Это не объяснение рассуждения модели.
A — historical Gemini 3.1 Flash Lite / Google EU; B, columns и RU не исследуются
в новом платном пилоте. [Исторический E6](0055-s117-e6-evidence.md) остаётся FAIL.

**H1:** отдельный шаг поиска часов рядом с уже выбранной датой увеличит число
правильных operation/booking clock selections на неизменном public cards PDF
относительно v12 без ухудшения остальных проверок этого документа.

Изменяется одна вещь: clock-only appendix в system message. Результат исследования
— решение сохранить или отклонить этот вариант, либо зафиксировать отсутствие
информативного сравнения. Отрицательный результат завершает пакет так же честно,
как положительный. Передача prompt и успех fake не считаются подтверждением H1.

## Scope и non-goals

Один пакет C1–C4: фиксированные base/candidate requests, причинная проверка
измерителя, подготовка одного ограниченного парного пилота и итоговое решение.
Offline реализация и её отчёт ведутся вместе; receipts и final evidence входят
в тот же исследовательский пакет без отдельных спецификаций для каждой записи.
Отдельными решениями остаются принятие контракта и запуск на exact source.

Не входят исправления coverage/overlap/turnover, service catch-all, автоматический
repair, подбор моделей/вариантов prompt, расширение corpus или лимитов,
изменение financial guards/scorer, production prompt/profile/consent/API/UI/БД,
финансовые записи, registry SPEC-0056, deployment и production model switch.
Старые private E6/P6/D5 payloads, closed stdout и их локальные копии не читаются.
E6/Q6 permissions остаются CONSUMED. Новый пилот пока **NOT_AUTHORIZED / NOT_RUN**.

## C1. Ровно два запроса и один изменяемый фактор

Base — текущие `BuildComparisonPlan` / `Messages` / `RequestSchema` для candidate A
и public `cards.pdf`, hash `d4dd02153724c1599d4d89eb073e8cfaaf3fbce1696a4d691b69d150d0ccdce6`.
Для base используется exact v12 SystemPrompt из указанной базы. Candidate — тот
же prompt, затем два LF и literal appendix ниже (без завершающего LF):

```text
CLOCK SELECTION PROCEDURE: For each record, first locate the printed operation date and booking date separately. Then, before emitting JSON, inspect the adjacent lines in that same field and record for a printed clock. When both date and clock are printed, select their aliases together in the corresponding date array, in source order; a date-only selection is incomplete in that case. Preserve the printed precision. If no clock is printed, keep the field date-only. Never borrow a clock from another record or field, invent a clock or timezone, or copy operation evidence into an unprinted booking field. Perform this check silently and return only the existing JSON object.
```

Version: `statement-span-mapping.v13.clock-selection-candidate`.
SHA256 UTF-8 bytes (без завершающего LF):

- Base v12: `31c0496472f618aa9622918ab0013bbf87e8b5a59903bfd8fcbe40954a123819`.
- Candidate: `c417773a3ff5fd8c855b0701ca8be30606f07e910e0dff47eb2972a4cc18a187`.

Эти fingerprints закрепляются в реализации и evidence.
При другой base identity отказ, а не автоматическая адаптация. Policy, schema,
compact user tuples, aliases, порядок, document/plan hashes и transport options
одинаковы. Request hashes различаются только system content. Candidate имеет
отдельную research identity и не записывается в product Plan/consent.

Builder принимает Document и сам вызывает штатный BuildComparisonPlan с точной A
policy; не вводит второй permissive validator произвольного Plan. Считает actual
JSON messages + schema для каждого варианта. Предел90000 bytes/2000 spans,
input100000/output16000 и timeout120s прежние; превышение — полный отказ без
обрезания, chunking или добавления фактов. Никаких generic prompt overrides.
Рабочий SystemPrompt/CurrentProfile, frozen E6/Q6 refs и legacy validators неизменны.

## C2. Измеритель, которому можно доверять

Каждый typed raw proposal проходит actual canonicalization → merge →
ValidateProposal → `ScreeningChecks("cards", review)`. Нельзя создавать Review,
issues или итоговый score вручную. Golden доступен assertions/измерителю,
но не fake или request builder. Невалидная структура/протокол не получает ноль
баллов: сравнение недоступно, дальнейшие вызовы останавливаются.

Primary metric — четыре actual scorer checks: operation_clock/booking_clock
для двух строк, по одному boolean на поле. Дополнительно для каждого поля
сравнивается precision с `local_datetime` по public golden; UTC остаётся пустым.
Эти derived comparisons не меняют scorer version/его20 checks. Помимо primary
metric сохраняются все исходные checks с ordinal/row/code/PASS и blockers.
Для сравнимого target нужны ровно2 rows и4 соответствующих trace entries.
При другом row count исходный quality FAIL сохраняется; target=N/A, не0/4.
Quality campaign может завершить заданные вызовы, но pilot будет INCONCLUSIVE
из-за несопоставимости. Только при2 rows исходный scorer обязан дать20 checks.

Offline causal pair на настоящем pinned Poppler: корректный alias proposal и
его копия, из которой убран только clock из четырёх date selections. Оба проходят
всю цепочку; полный вариант даёт4/4 clock checks, урезанный0/4. Не замазывать
возникший coverage FAIL. Ещё два отрицательных контроля: чужие часы и попытка
придумать время при date-only source. Shared currency/финансовые guards прежние.

Источники — public cards/columns PDF и synthetic граничные случаи. Golden/corpus
не редактируются. Существующий S1.18 helper gate повторно выполнен при пересмотре
программы; [evidence](../research/statement-quality/s119-offline-review.md).
Его завершающий near-miss создаёт Review напрямую: это ограниченная проверка
scorer, поэтому она **не заменяет** новый сквозной causal pair C2.

## C3. Один заранее определённый парный пилот

После offline gates, интеграции реализации, CI5/5 и отдельного разрешения на
exact source — один pilot, **3 пары / максимум6 POST**, только A/cards:

| Пара | Порядок |
| --- | --- |
| 1 | base, candidate |
| 2 | candidate, base |
| 3 | base, candidate |

Запросы независимы, без conversation state, retry и адаптации после ответов.
Внутри пары отличаются только инструкции C1. Sampling options одинаковы и
фиксируются в manifest; никаких утверждений о полной детерминированности модели.
Известный quality FAIL позволяет завершить заранее заданную последовательность;
protocol/route/unknown billing/timeout/settlement failure прекращает весь pilot.

Предлагаемый provider cap:6 ×53900 = **323400 USD micros ($0.3234)** по нынешней
repo policy A (input275000/output1650000 micros за миллион и bounds100000/16000).
Это расчёт конфигурации, не свежая цена и не разрешение на расход. Перед запуском
нужны fresh exact-model/route/capability/ZDR proof и прежний shared budget с fees.
Иная цена/модель/route — отказ, не незаметное расширение cap. Новое разрешение
не обнуляет прошлый расход и не увеличивает allowance.

Вызовы используют существующие transport, route/billing guards, budget reserve/
settle и fixed public fixture. Product constructor не выбирает candidate.
Операторский путь фиксирован только на этот pilot; не создавать универсальный
framework или добавлять очередной boolean mode в monolithic deploy workflow.
Если нужен workflow, это один отдельный research job без deploy и соседних modes.
Source CI validator и защита one-shot переиспользуются, legacy jobs не меняются.
Конкретный source/ref, inputs, manifest и способ исполнения предъявляются до
отдельного live approval; исходящий live HTTP/SSH этим offline acceptance не разрешён.

Сохраняются private bounded typed proposals до canonicalization и outcomes всех
отправленных вызовов, чтобы разбор завершился внутри того же разрешённого job.
Фиксированная identity pilot — `s119-clock-pilot`; marker/report отдельны от
E6/Q6 и общие для всех source SHA этого одного разрешения. Существующие
storage0600/O_NOFOLLOW/no-clobber/digest правила сохраняются; никакого arbitrary
path. Snapshot не более32768 bytes на ответ,6 snapshots; весь private report
не более1048576 bytes. Capture failure/overflow — остановка, не усечение.
В Git только safe
counts/status/digests. Новая неисчерпанная identity не заимствует E6/Q6 authority.
Отказ preflight тоже расходует one-shot; повтор требует нового решения.

## C4. Решение и обязательная остановка

При полном валидном pilot:

- **KEEP_FOR_FURTHER_REVIEW**: candidate получил4/4 clocks и правильную precision
  во всех3 ответах; хотя бы один base ответ хуже; candidate ни в одной паре не
  ухудшил primary count и не превратил base-PASS любого из16 остальных checks в FAIL.
- **REJECT**: candidate не достиг указанного target, ухудшил любой такой check
  или выбрал неверную precision. Дополнительные baseline failures не скрываются.
- **INCONCLUSIVE**: оба варианта уже идеальны по target либо campaign неполна
  из-за protocol/route/billing/refusal. Нет переноса баллов между неполными парами.

Порядок решения: неполнота/protocol refusal/target=N/A → INCONCLUSIVE;
иначе ухудшение non-clock checks или недостигнутый candidate target → REJECT;
иначе strict gain → KEEP_FOR_FURTHER_REVIEW, иначе ceiling tie → INCONCLUSIVE.
Все predicates вычисляются на полной сравнимой campaign, не на её удобной части.
Три ответа — три повторения на одном PDF, а не12 независимых samples или p95.
Даже KEEP не квалифицирует модель, другие форматы, новый corpus или production.

**Предел итераций: один candidate, один pilot.** После любого исхода работа над
этим вариантом завершается отчётом. Нет v14, новых моделей, второго pilot или
расширения scope по инерции. Следующий подход требует отдельного решения.

## Приёмка и исполняемое evidence

| AC | Требуемое evidence |
| --- | --- |
| C1 | Exact two requests/hashes/bytes; HTTP body через существующий serializer на localhost; unchanged schema/policy/user; oversize отказ |
| C2 | Сквозной real-helper4/4→0/4 causal pair; precision/чужие часы/date-only negatives; judge matrices для всех C4 исходов |
| C3 offline | Six-call order/cap/one-shot/early-stop проверены fake transport и fake meter; proof до POST; private capture bounded; production isolation |
| C3 live | Отдельное exact-source разрешение, один run, known/settled accounting, no retry; сейчас NOT_AUTHORIZED / NOT_RUN |
| C4 | Решение вычислено без ручной подмены; полный отчёт о всех попытках, limitations и остановке |

В implementation evidence указываются реальные команды тестов, source/exit
status. Обязательны `make test`, `make test-production`, `make test-integration`,
`go vet ./cmd/... ./internal/...`, `git diff --check` и предметные race/HTTP/helper
проверки изменённых модулей. Existing тесты переиспользуются; не дублировать их
только ради номера S1.19. Offline PASS не заполняет строку C3 live.

## Данные, API, UI, ошибки и rollback

API/OpenAPI/generated types/UI, PostgreSQL schema и финансовые записи не меняются.
Runtime model/flags/images не переключаются; manual import сохраняется.
Операторские metering/marker/report writes входят только в отдельно разрешённый
pilot, не являются импортом финансов. До live approval внешних вызовов0.
Rollback — убрать research candidate/entrypoint; product state не мигрируется.
Ошибки не превращаются в PASS или основание автоматического повтора.

## Открытое решение

C1–C4 offline приняты и интегрированы PR107.
[C1–C4 evidence](0055-s119-evidence.md). Пилот имеет отдельный будущий exact-source gate. Один
[актуальный status](../statement-ai-status.md) ссылается на этот документ;
история сохранена отдельно и не задаёт следующие шаги.
