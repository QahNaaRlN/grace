# SPEC-0055 / S1.19: offline-кандидат инструкций выбора evidence

- Статус: **draft**; implementation_status: **not_started**.
- Автор: Codex; дата: 2026-10-23, Asia/Yekaterinburg.
- Acceptance owner: владелец Grace; acceptance record: **отсутствует**.
- Основание: продолжение после сообщения «Слил PR» о №105;
  [post-merge №105](0055-pr105-postmerge.md).
- База: main `1a5bbfbefecd6e8983a8c290f6e8c650c62351dc`.
- Spec branch: `codex/statement-s119-repair-spec`.
- Этот PR содержит контракт для рассмотрения. Реализация R1–R5 начинается
  после явного принятия и интеграции принятой спецификации в main.

## Проблема и ожидаемый результат

[Q6](0055-s118-q6-evidence.md) успешно объяснил сохранённый E6, но не исправил
его quality FAIL. Все 6 tuples, 22 failed checks и 23 reason entries VERIFIED;
15 public unknown сопоставлены. Публичные результаты позволяют выделить три
семейства: потеря времени/точности, повторное использование evidence и пропуски
coverage. Они не доказывают, что дополнительная инструкция улучшит модель.

Предлагается подготовить **один research-only кандидат prompt** и offline
доказательства его точной передачи, размера и совместимости с прежними Go
проверками. Результат — готовый к отдельному рассмотрению запрос и регрессионный
пакет. Действующий product prompt и все live campaign paths сохраняются.
Эффект на Gemini/Mercury остаётся **UNVERIFIED**; offline fake не является моделью.

Такой объём выбран потому, что S1.18 установил типы ошибок, а качество ответа на
новые инструкции ещё не измерено. Подключение кандидата к runtime, даже без
деплоя, меняло бы общий prompt/profile и идентичность согласия до проверки гипотезы.

## Доказанная исходная точка

Используются только public repository source и safe summary Q6:

| Факт | Что он позволяет утверждать |
| --- | --- |
| A/cards:4 совпадения date и UTC,4 расхождения time и precision | Нужна проверка полноты date+clock selection; причина выбора моделью неизвестна |
| Columns A/B и cards A/B:6/7/1/10 unmapped spans | Есть конкретные пропуски; их финансовая/service роль из счётчика не следует |
| 13 entries `overlapping_evidence` в tuples4–6 | Есть пересечение evidence; entries включают row/blocker повторы, это не13 независимых дефектов |
| 2 `invalid_source_rows`; A/RU: 2 расхождения printed amount при2 совпадениях currency | RU остаётся отдельным риском; числовое расхождение не устанавливает выбранную финансовую роль |

`internal/statementextract/plan.go` уже требует clock spans, полноты покрытия,
раздельных ролей и service с reason. Поэтому кандидат уточняет процедуру
самопроверки, а не вводит прежде отсутствовавшую финансовую проверку.
`Messages` отправляет system prompt и compact tuples; `InputBytes` считает
messages + actual request schema. `aliases.go` проверяет exact plan identity.
`internal/statementdata/review.go` уже предъявляет single-use claims к датам,
суммам, description и service; currency/labels/direction evidence имеют отдельное
существующее правило sharing. Coverage считается по действительному used set.
`internal/statementqualification/quality_checks.go` сравнивает компоненты дат и
денег с независимым golden; scorer остаётся источником PASS/FAIL.

## Scope и границы

После принятия разрешаются R1–R5: чистый research-only builder в
`scripts/research/s119-request/`, offline Go/HTTP/helper tests, локальный test
script и подключение его к `make test-production`, implementation evidence.
Файлы candidate builder имеют package без `main`; исполняемого runner нет.
Изменения `internal/assistant` и `internal/statementqualification` допускаются
только в тестах. Рабочие зависимости на candidate package отсутствуют.

Не входят: переключение `statementextract.SystemPrompt`/`PromptVersion`,
изменение schema/planner/alias/redaction, financial guards/scorer или прежнего corpus/golden,
автоматическое дополнение или перенос IDs, service catch-all, repair/retry,
финансовые записи, API/UI/БД, новый report/campaign/authorization/workflow input,
модели/цены/budget, registry SPEC-0056, deployment или production flags.

Запрещены external inference/metadata, workflow_dispatch, SSH/VPS и повторное
чтение любых private E6/P6/D5 artifacts, closed stdout или сохранённых witnesses,
включая локальные копии. Не создаются fixtures из private-derived материала.
Оба прежних разрешения E6/Q6 остаются CONSUMED. Нового live этапа в S1.19 нет.

## Контракт R1–R5

### R1. Один кандидат и независимая identity

Фиксированный candidate version:
`statement-span-mapping.v13.evidence-selection-candidate`.
Candidate system message — **точный текущий SystemPrompt v12** на базе выше,
затем два LF и literal appendix R2. Базовый prompt закрепляется digest в тесте;
при его изменении builder должен отказать, а не молча получить новую основу.
Другие variants, параметризуемый prompt, owner text и поиск лучшей версии не входят.

Фиксированные SHA256 UTF-8 bytes (без завершающего LF):

- Base v12: `31c0496472f618aa9622918ab0013bbf87e8b5a59903bfd8fcbe40954a123819`.
- Candidate v13: `17c661e349a7c94bb0b8fe217ee9319640c8ac56261caaf6a7f03b38a38de8c9`.

Appendix берётся из блока R2 без внешних code fences и завершающего LF.
Hashes вычислены из public source и literal текста; это не quality evidence.

Чистый builder получает текущий Plan, созданный штатным BuildPlan/
BuildScreeningPlan/BuildComparisonPlan из публичного документа. Он не загружает
файлы, env или credentials и не вызывает providers. Принимается один chunk с
Index 1, empty Context, от 1 до 2000 spans; plan должен иметь текущие версии,
полную source-order alias/atom/chunk identity и прежний limits/bytes contract. Policy — только точная текущая product, screening
или одна из двух comparison policies; произвольная модель/цена не допускается.
Проверка не допускает изготовленного plan с одной лишь правильной строкой версии.
Не вводить новый permissive validator в product code: research-only validation
может использовать независимую проверку текущей формы с mutation tests.

Выход — typed research request: candidate version, base prompt SHA256,
candidate prompt SHA256, base plan hash, base request hash, candidate request
hash, policy, messages, actual schema, input bytes. SHA256 prompt считается от UTF-8
bytes; hashes plan/request — прежним `statementextract.Hash` contract.
Оба request hash используют одну форму `{Policy, Messages, Schema}` из Q6;
различаются только system content. Число bytes — JSON messages + schema,
как в текущем InputBytes; это не wire HTTP total и не измеренные tokens.

Policy, user message, schema и все IDs/значения byte-identical базовому request.
Builder не меняет Plan.PromptVersion/PlanHash/consent и не выдаёт candidate за
valid product plan. Input и source остаются неизменными. Повторный вызов
детерминирован; результат deep copy без общего изменяемого backing storage.
Лимиты 90000 bytes / 2000 spans / 100000 input bound / 16000 output bound сохраняются.
Если candidate bytes превышают 90000, возвращается error без partial request;
увеличивать лимиты, обрезать prompt/spans или разбивать документ нельзя.

### R2. Точный appendix: clocks, claims и coverage

Текст английский, как действующий model prompt; спецификация и отчёты русские.
Appendix не запрашивает chain of thought, дополнительный ответ или новые keys:

```text
Before emitting the existing JSON object, check these selections silently. Return only that JSON object; do not add an explanation, audit object, confidence, values or calculated facts.
CLOCKS: For each operationDate and bookingDate, preserve every printed date and clock fragment belonging to that field in that record, including a clock on an adjacent line. Keep the printed precision. A printed date without a clock stays date-only. Never copy a neighboring record's clock, invent a clock or timezone, or reuse operationDate evidence to fill an unprinted bookingDate. Distinct printed operation and booking fields must use their own evidence.
CLAIMS: Dates, monetary values, descriptions and service spans must not claim the same alias twice across those selections or records. A selected financial field must not also appear in service. Only the existing shared currency, heading/label and direction-evidence roles may reuse evidence as permitted by Go. Sharing a currency never permits sharing a date, amount or description. Never delete evidence from a valid field merely to hide a conflict.
COVERAGE: Check the supplied aliases against the union of your actual field and service selections. Classify a residual alias only when its printed location and content establish its role. Keep all description fragments and internal separators in their own record. Use service with a specific truthful reason only for genuinely unused service content. Never place all residual aliases in service, invent a transaction, borrow a label or assign a financial role merely to obtain full coverage. Leave genuinely uncertain evidence visibly unresolved; an incomplete result must remain incomplete under Go validation.
```

Это инструкция модели, а не новый Go алгоритм и не гарантия её соблюдения.
Старые запреты на bank identifiers, доступный/book balance, turnover labels,
суммы из row totals, injection и передачу приватных значений остаются в prefix.
Исправление выбора printed turnover amounts не заявляется результатом S1.19.

### R3. Wire evidence и отсутствие runtime подключения

Тест передаёт результат R1 через существующий `ReasoningProvider.Complete` /
OpenRouter HTTP serializer на **локальный httptest endpoint**, с synthetic
credential и зафиксированным test policy. Проверяются фактические messages,
exact schema/enum, response_format, limits, no tools/no fallback/ZDR options и
прежний режим DisableReasoning. Создание альтернативного HTTP serializer
или обход transport guards ради теста запрещены.

Проверка должна падать, если appendix исчез, вместо candidate отправлен base
prompt, schema/IDs изменены или bytes считают только старый prompt. Тест endpoint
не оценивает prompt как модель: ответ заранее определён или построен только
по доставленным tuples и geometry, без golden/private lookup. Он подтверждает
wire contract, не улучшение извлечения. Никакого production endpoint по умолчанию,
внешнего DNS/HTTP или вызова CurrentPolicy с реальными metadata.

Нет hook/interface/config, позволяющего product reasoner, CLI или campaign
выбрать candidate. `CurrentProfile`, historical report formats/validators,
active consent, workflow defaults/dispatch modes, frozen refs E6/Q6 и их bytes
сохраняются. Мутации candidate identity не должны делать его допустимым
product plan/consent или разрешением E6/Q6. Нет миграции/переподписания evidence.

### R4. Причинные offline контрпримеры

Источник — прежние public columns/cards/turnover-roles-ru PDF и небольшие новые
**synthetic** документы для граничных случаев; hashes прежнего corpus/golden
не меняются. Для PDF используется настоящий pinned Poppler helper без сети.
Ответы fake используют только aliases действительно доставленного chunk;
golden доступен только assertions. Каждый pair меняет один selection/fact и
проходит actual canonicalization → merge → ValidateProposal → ScoreScreening,
без подделки Review/issues или отредактированных saved reports.

| Семейство | Положительный и отрицательный контроль |
| --- | --- |
| Clock | Полный date+clock сохраняет time/precision; удаление clock воспроизводит соответствующий score FAIL. Date-only без печатных часов не получает выдуманного времени; соседняя строка и чужой clock не становятся успехом |
| Claims | Раздельные evidence для двух печатных полей проходят; один date/amount/description в двух claimed fields либо field+service даёт прежний overlap blocker. Повтор currency в header/rows остаётся допустимым |
| Coverage | Корректно классифицированный публичный пример имеет matched coverage; удаление одного selection даёт точный unmapped count/coverage FAIL. Уверенно заданный service reason проверяется лишь как форма: matched coverage не доказывает истинность классификации |
| Financial regression | Чужой turnover label, identifier вместо суммы, available вместо book, пропуск hidden numeric fragment, конфликт currency и description gap не становятся financial acceptance. Нельзя понижать критерии ради positive case |

На трёх неизменных PDF положительные вручную подготовленные alias proposals
должны проходить прежние 19/20/23 checks. Они показывают достижимость правильной
разметки и прохождение transport/guards; не показывают, что кандидат её генерирует.
Если правильный пример не проходит guard, фиксируется FAIL и отдельное решение;
guard/scorer нельзя ремонтировать внутри S1.19. Синтетический fixture вне трёх
hashes не объявляется qualified через fixed-corpus scorer: для него evidence —
actual validation и независимые assertions, отдельно от19/20/23.

В evidence отдельно указать, где Go отвергает ошибку, где её обнаруживает только
scorer и где автоматического доказательства семантики нет. Residual финансовый
span в service может закрыть coverage без доказательства роли: это ограничение
должно остаться явным, новая service семантическая защита здесь не добавляется.

### R5. Проверки, отчёт и следующий gate

Предлагаемые исполняемые проверки (ещё **не реализованы/не выполнены**):

- `go test ./scripts/research/s119-request` — R1/R2, hashes/deep-copy/mutation,
  candidate exact90000/90001 bytes и2000/2001 spans, invalid plan, legacy isolation.
- `go test ./internal/assistant -run S119` — R3 actual localhost HTTP wire.
- `./scripts/test-statement-s119-candidate.sh` — R4 pinned helper и guards/scorer;
  добавить в `make test-production`; отсутствие helper не считается PASS/skip.
- `make test`, `make test-production`, `make test-integration`,
  `go test -race ./internal/statementdata ./internal/statementextract ./internal/statementqualification ./scripts/research/s119-request`,
  `go vet ./cmd/... ./internal/... ./scripts/research/s119-request`, `git diff --check`.

Отчёт содержит R1–R5 с командами/exit status, точными source/digests/sizes на
трёх PDF, positive/negative outcomes и source isolation diff. До выполнения
статусы **NOT_RUN**, live effect **UNVERIFIED**. В Git только public/synthetic
материал. Не читать или публиковать private artifacts для «проверки улучшения».

## Приёмка и завершение

| AC | Исполняемое evidence после отдельного принятия |
| --- | --- |
| R1 | Детерминированная отдельная request identity, точные границы и отказ на mutation/oversize |
| R2 | Literal appendix и pinned v12 prefix, exact wire/body tests |
| R3 | Настоящий serializer на localhost; runtime source/profile/consent/workflows unchanged |
| R4 | Причинные good/bad pairs на real helper и synthetic cases, прежние 19/20/23, guards без изменений |
| R5 | Полный bundle PASS и честный отчёт об ограничениях; внешних запросов0 |

Публикация/слияние draft не означает принятие R1–R5. Владелец отдельно принимает
этот контракт; затем принятую версию интегрируют до implementation worktree.
Окончание реализации — offline candidate ready for review, не model qualified.
Продвижение candidate в active prompt/profile/consent и новый платный experiment
требуют отдельного контракта и решения; в этом пакете таких этапов нет.

## Данные, API, UI и rollback

Финансовое состояние, PostgreSQL/OpenAPI/generated types/UI не изменяются.
Ручной импорт доступен по прежнему контракту. Candidate не сохраняется в owner
state и не принимает private documents. Ошибки builder закрывают offline output;
ошибки проверок не нормализуются в PASS и не разрешают retry.
Rollback research-пакета — удаление builder/tests/test hook; product/ledger
восстанавливать не требуется. Исходный E6 quality FAIL и Q6 SUCCESS сохраняются.
