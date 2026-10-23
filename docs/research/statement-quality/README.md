# Statement AI: проработка устойчивости и расширенного корпуса

Актуальная работа и следующий gate — [единая status-страница](../../statement-ai-status.md).
Ниже исторический design study2026-10-19; его baseline/production snapshot
не является проверкой текущего runtime.

- Дата: 2026-10-19 (Asia/Yekaterinburg).
- Статус: design study accepted, 2026-10-19; implementation `not_started`.
- Acceptance record: владелец ответил «Принимаю» на результат проработки / draft PR #68, revision `c63e7cd3f7ca355ee7470036919b948d0078b73f`. Приняты выводы и последовательность S1–S5; конкретные технические редакции и платные campaigns сохраняют свои gates.
- Основание: владелец поручил «Приступай к проработке» устойчивости обоих layouts и более широкого корпуса.
- Baseline: `8232976`, draft SPEC-0056 / PR #67; product code `7be572c`, draft PR #66. CI `39218772264` завершён PASS для всех пяти jobs; это не model qualification.
- Исполнено сейчас: 32 offline planner probes, 0 model calls, 0 database writes; публичные synthetic PDF и layout-only mutations.
- На историческом срезе2026-10-19: production `1ada384`, Statement AI false. Billing reconciliation завершена ранее; в этой проработке production не менялся.
- Связи: [SPEC-0055](../../specs/0055-bank-statement-pdf-import.md), [activation evidence](../../specs/0055-activation-evidence.md), [draft SPEC-0056](../../specs/0056-statement-model-selection.md).

## Что установлено, а что остаётся гипотезой

Три разных свойства нельзя смешивать: все source spans присутствуют в плане;
поля одной операции доступны вместе; модель правильно классифицировала каждый
span. Нынешний planner обеспечивает первое, но не гарантирует второе; live
результаты отдельно показали проблемы третьего.

| Наблюдение | Evidence | Вывод / граница |
| --- | --- | --- |
| v7 + merge fix: финансовые поля cards верны, четыре фрагмента не классифицированы | Live `39217497119`, read-only diagnosis `39218054190` | Source coverage FAIL: повторная currency и три подписи полей. Это model mapping/classification failure |
| v8: лишняя неполная строка | Live `39218319463` | Model quality FAIL, v8 отклонён. Конкретная причинная цепочка лишней строки не доказана |
| Исходные columns: v7 один chunk, v8 два, но core каждого ряда целиком в одном chunk | Offline evidence ниже | Само увеличение числа chunks не доказывает потерю/разрыв операции |
| Исходные cards: обе версии по два chunks; core обоих рядов атомарен | Offline evidence ниже | Разрыв core на исходном cards PDF не воспроизведён; нельзя объявлять его причиной live лишней строки |
| Cards + четыре служебных prefix words: v7 делит core второй операции | Offline v7 `layout_prefix_4` | Подтверждённый дефект устойчивости planner к нефинансовому префиксу |
| Cards + 12 prefix words: обе версии делят core первой операции | Offline v7/v8 `layout_prefix_12` | Небольшие изменения контекста двигают byte boundary через вертикальный record |
| Слишком крупная единая полоса (32/64 prefix words) | `statement_atomic_block_limit`, 8 cases | Safe preflight refusal; никакого усечения или inference |

Из 32 probes: 24 плана успешно построены; в четырёх из них core одной
вертикальной операции разделён. Во всех 24 планы сохраняют каждый source span
ровно один раз и messages+schema ≤9000 bytes. Это не модельная accuracy и не
оценка частоты ошибок в реальных банках.

Предыдущий bank integration FAIL `fixture plan split unexpectedly 2` —
ограничение текущего fake-provider harness на один вызов. Для columns офлайн
проверка не показала разрыва core; такой FAIL сам по себе не доказывает дефект
финансового извлечения. Harness нужно расширить на реальные multi-chunk paths,
сохранив конечные проверки полей/coverage/confirmation, а не просто удалить assert.

## Как воспроизведено

```sh
./scripts/research/probe-statement-chunks.sh /tmp/grace-statement-study
```

Нужны Docker/Compose, Go, Python 3 и история обоих commit refs. Script
отказывает, если product tree отличается от baseline `7be572c`; это
воспроизведение снимка, а не benchmark любого будущего HEAD.

Исходники: [probe shell](../../../scripts/research/probe-statement-chunks.sh),
[Go probe](../../../scripts/research/statement-quality-probe.go).
Результаты: [v7](evidence/v7.json), [v8 overlay](evidence/v8.json),
[сводка](evidence/summary.json), [provenance](evidence/provenance.json).

Реальный Poppler helper из Dockerfile проекта работает network=none/read-only,
UID 10001. Клиент также network=none; нет ключей, API сервиса или PostgreSQL.
Второй Go build использует `-overlay` только для исторического `plan.go` из
`8a83cdc`; production-файлы не переписываются. PDF SHA-256 и версии extractor/
prompt записаны в отчётах. Все контейнеры/volume этого probe удаляются его trap.

Для каждого из двух публичных PDF добавлены варианты с 0/4/8/12/16/24/32/64
нефинансовыми словами перед layout. Вариант 0 — настоящий extracted PDF;
остальные — in-memory layout mutations, а не новые PDF и не новые банки.
Core oracle независимо привязан к уникальным printed operation/booking dates,
account/FX amounts и clocks этих synthetic источников. Проверка не охватывает
весь record, description, currency binding или модельное понимание смысла.

## Предлагаемый первый пакет: устойчивый план без новых модельных вызовов

Сначала устранить структурную нестабильность, затем менять формат ответа и
сравнивать модели. Иначе результаты benchmark будут смешивать разные причины.

1. **Record atoms.** До byte packing выделять консервативные геометрические
   единицы: горизонтальную строку таблицы, либо целый вертикальный блок. Признаки
   — расположение/отступы/разрывы между полосами, без bank-specific parser и без
   доверия финансовому смыслу текста. Поля модели не должны создавать границу
   атома после consent. Вся source карта и границы входят в versioned plan hash.
2. **Неуверенная граница.** Одна геометрическая эвристика не является доказательством
   структуры для всех банков. Если безопасная единица не помещается или граница
   неоднозначна, вернуть явный preflight/manual-review outcome. Не резать record
   ради cap, не выбрасывать spans и не склеивать partial rows по похожим суммам.
   Continuation через страницу — отдельный обязательный negative/positive case.
3. **Стабильность.** Новые prompt/schema версии могут менять packing атомов,
   но не их границы внутри record. Сначала учитывать полный prompt/schema/
   context budget, затем класть целые атомы. Context headings допустимо повторять
   как read-only references; владение source spans остаётся однозначным.
4. **Проверка.** Все current 32 probes: либо core atomic, либо явный отказ до
   inference; ноль успешно принятых планов с разорванным core. Добавить stress
   около 9000 bytes, перестановку колонок, несколько колонок страницы и
   continuation. Успех synthetic probe не делает эвристику универсальной.
5. **Multi-chunk harness.** Fake reasoner получает только реально доступные в
   данном chunk IDs, возвращает chunk-specific proposals. Проверяются rows,
   clocks/FX, totals/coverage, отсутствие duplicates и ноль финансовых writes
   до confirmation. Фиксированный assert «ровно один chunk» заменяется этими
   причинными свойствами, а не отключением теста.

Это изменение технического контракта/plan version для отдельного review.
В текущей проработке planner не изменён и product tests не переписаны.
Границы следующего шага и проверяемый контракт описаны в
[S1 technical candidate](../../specs/0055-record-atomic-planning.md); точный
[Offline comparison](atoms.md) выполнен: геометрические кандидаты отклонены;
новая техническая редакция S1 предлагает whole-document/aliases с явно
увеличенными caps. Реализация не начата, принятие этой редакции отсутствует.

## Второй пакет: явный ответ по каждому source span

Проблему четырёх пропусков не решать автоматическим превращением остатка в
service. Такой остаток может содержать пропущенную финансовую строку.

Рекомендуемый кандидат wire format для исследования — **required classification
на каждый main source locator** с явным `unresolved`, вместе с field/record
ссылками. Возможные реализации сравнить offline: required object slots либо
фиксированный slot vector. Для сокращения payload допустимы короткие локальные
aliases, если Go хранит взаимно однозначную alias→canonical locator map и не
принимает неизвестные/дублирующиеся aliases. Реализованный способ ещё не выбран.

Go обязан проверять равенство множеств main source IDs и подтверждённых
classifications, согласованность classifications с реальными field selections,
отсутствие overlaps и неизменность исходных значений. Required slots позволяют Go отклонить отсутствующий ответ, но не гарантируют
поведение модели или истинность роли: модель всё ещё может ошибочно объявить
сумму служебной. Такие ошибки проверяются golden oracle; неоднозначные numeric/
date candidates должны оставаться видимыми и блокировать автоматическое принятие,
а не скрываться blanket service reason.

Проверять required cardinality/types и duplicate object keys на стороне Go:
одного `DisallowUnknownFields` недостаточно для отсутствующих полей или
повторного JSON key. Нулевые/неполные/лишние slots, unknown alias и conflicting
bindings отклоняются до формирования финансового draft.

Проверка первичных источников 2026-10-19: [Gemini structured outputs](https://ai.google.dev/gemini-api/docs/structured-output)
описывает object required/additionalProperties и array minItems/maxItems, но
поддерживает лишь подмножество JSON Schema и может отвергать сложные схемы.
[OpenRouter structured outputs](https://openrouter.ai/docs/guides/features/structured-outputs)
указывает зависимость от endpoint и различную строгость enforcement. Это
поддерживает feasibility исследования, но не доказывает исполнение новой схемы
конкретным `google-vertex/eu`; actual compatibility пока UNVERIFIED.

Финансовые grounding/decimal/date/coverage/reconciliation guards и отдельное
owner confirmation сохраняются. У новой схемы отдельные prompt/schema/plan
versions: старое consent не переиспользуется. Первый вариант — один model pass
на chunk; repair loop или LLM structure pass не добавляются скрыто.

Перед выбором wire format измерить offline input/schema и conservative output
size для каждого corpus case при ≤2000 output tokens. Byte bound не доказывает
реальный token count. Неподтверждённые ограничения endpoint schema пометить
UNVERIFIED и проверить официальной metadata/документацией до оплачиваемого run.
Увеличение output cap либо новый двухэтапный inference требует отдельного
принятого контракта и полного нового budget/consent plan.

## Расширенный corpus: конкретный manifest

[Manifest](corpus-manifest.json) содержит 24 случая: 8 development, 8 holdout,
8 negative/unsupported. Только два исходных EN PDF уже существуют; остальные
22 fixture ещё не созданы. Golden expectations — кандидаты на независимый review,
не утверждённые эталоны и не выполненный benchmark.

Development включает оба layouts и RU/EN, повторные currencies/labels,
несколько строк description и packing boundaries. Holdout отделён до настройки
prompt/схемы: новые расположения/порядок колонок, плотность, pagination и
балансовые роли. Negative cases требуют ожидаемой неопределённости или отказа:
missing currency/date, конфликт на 1 minor, неоднозначный record, несколько
счетов, credit statement, scan, encrypted PDF.

Golden records создаются из независимого источника fixture до model run:
row identity, точные minor units/currency/direction, оба clocks с precision/
timezone, balances/turnovers и роли source spans. Нельзя брать «эталон» из
оцениваемой модели, из её self-confidence или объявлять все unused IDs service.
Один набор значений при неверной группировке строк обязан FAIL.

## Критерии стабильности и бюджетная выполнимость

- Для положительных случаев: каждый критический field/row/column check верен,
  нет пропущенных/лишних строк, нет выдуманных значений; source coverage полон.
- Для negative: ожидаемый blocker/abstention или ранний unsupported; отсутствие
  ложного принятия и финансовых writes. Нулевой результат не считается accuracy
  на положительном случае.
- Для каждого text-inference case — три заранее назначенных повтора, все
  результаты сохраняются. Не заменять FAIL удачным повтором и не дописывать
  попытки до первого PASS. Это наблюдаемая повторяемость на данном корпусе,
  не статистическая гарантия всех банков.
- Метрики раздельны: row/field/column exact match, source classification,
  omissions/extra rows, false acceptance/abstention, per-case latency и exact
  billing; финальные суммы и totals сами по себе недостаточны.

У manifest 22 потенциальных text-inference cases и два offline-only отказа.
22 × 3 = **66 scheduled document attempts** на одного кандидата. Одни
16 положительных cases × 3 повтора требуют минимум **48 model calls** для
полной проверки модели, даже при одном call на attempt. Шесть text-negative
cases могут отказать до сети, поэтому 66 не объявляется нижней границей calls.
Уже 48 больше proposed per-run cap 32 из SPEC-0056: один capped smoke
не может представлять полную qualification.
Multi-account/credit cases консервативно учтены как возможные inference cases:
способность отвергнуть их до модели пока не доказана. Только scan/encrypted
отнесены к offline-only. Нельзя заранее вычитать неустановленные paid calls.

Кандидат campaign policy для review: каждый text case после нового planner
вмещается в ≤3 calls; всего ≤198 calls и theoretical provider ceiling ≤1.584 USD
на одну exact model/endpoint пару при прежних ценовых ceilings. Это UNVERIFIED
working envelope до создания corpus и его preflight, не измеренная стоимость. Shards имеют
≤32 calls / ≤0.256 USD / ≤120 s, как proposed SPEC-0056; весь manifest/порядок/
число повторов и **общий** campaign ceiling фиксируются заранее. Это минимум
семь shards при worst-case 198 calls, не обещание их завершения за 120 s.
Timeout/incomplete campaign не получает PASS; новых paid retries автоматически нет.

Если какой-то case не помещается в declared envelope или весь campaign не
помещается в фактический остаток общего бюджета с fees, отказать **до первого
model call** и пересмотреть план/лимиты. Не удалять трудные cases ради попадания в cap.
Общий бюджет не повышается. Не проверять
сразу три модели полными campaigns: сначала offline gates и малый screening,
только затем отдельно разрешённый complete campaign выбранной пары. Эти новые
campaign ceilings — предложения, разрешения тратить их в этой работе нет.

Follow-up 2026-10-19: [S1 comparison](atoms.md) предлагает другой profile
(один whole-document call, theoretical ceiling 0.07 USD). Приведённые выше
198 calls / 1.584 USD относятся к прежнему 0.008 USD/call envelope S0. Для
нового profile они не являются разрешённым или проверенным budget: если все
22 text cases действительно укладываются в один call, 66 scheduled attempts
дают theoretical provider ceiling 4.62 USD, плюс действующие fees. Corpus ещё
не создан полностью, shared remaining не проверялся этой работой. Требуются
новый aggregate preflight и отдельное решение по campaign; allowance не растёт.

Частичные shards не дают `qualified`. Campaign report связывает corpus/golden
hashes, все запланированные attempts (включая failures), route, версии planner/
extractor/prompt/schema/redaction, свежие metadata и полный billing audit.
Production usage/remaining/token/generation/cost values остаются приватными.

## Пакеты и границы дальнейшей работы

| Пакет | Результат | Gate |
| --- | --- | --- |
| S0: эта проработка | Offline evidence + root-cause distinctions + 24-case manifest + protocol candidates | accepted владельцем 2026-10-19; 0 model calls |
| S1: planner/harness | Целые record atoms, безопасный отказ и правдивый multi-chunk integration | техническая редакция accepted; offline evidence до моделей |
| S2: coverage protocol | Выбранный bounded response format, проверенные aliases/classifications/unresolved | принятые schema/plan/consent изменения; fake-provider guards |
| S3: corpus | 24 реально созданных fixture, независимые golden expectations, holdout frozen | fixture/render/extractor/scorer review; пока 22 planned |
| S4: bounded evaluation | Предварительный screening, затем отдельно утверждённый complete campaign | конкретная пара, aggregate cap и actual shared budget; никаких скрытых retries |
| S5: registry/enable | Только qualified пары допускаются в SPEC-0056; отдельный release/activation и owner check | full evidence, CI/Delivery, новый runtime smoke; сейчас not_started |

API/OpenAPI compatibility, миграции plan/consent/audit и rollback обсуждаются
в технической редакции до реализации S1/S2; публичные DTO меняются только через
OpenAPI и сгенерированные типы. Финансовое ядро и подтверждения не расширяются.
Двухисходный fixture smoke, зелёный CI или более низкая цена не заменяют S3/S4.

## Следующее поручение — 2026-10-20

Добавлены [регрессия ролей банковских реквизитов и подбор современных моделей](next-quality-package.md).
[S1.3](../../specs/0055-s13-turnover-roles.md) accepted прямым поручением
«оформи спеку и вперед», [реализация local_verified](../../specs/0055-s13-evidence.md), не deployed. Добавлен новый RU
development fixture с independent golden; старый 24-case manifest и его
исторические счётчики не переписаны. [Shortlist metadata](model-shortlist-2026-10-20.md)
не является qualification. Model calls = 0; полного нового campaign нет.
