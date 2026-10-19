# SPEC-0056: автоматический выбор модели для банковской выписки

- Статус: draft
- Автор: Codex по поручению владельца
- Дата: 2026-10-19 (Asia/Yekaterinburg)
- Acceptance owner: владелец Grace
- Direction record: «Хорошо» после предложения автоматизировать выбор из проверенных моделей; направление согласовано, конкретный контракт ниже ещё не принят
- Acceptance record: отсутствует для этой редакции
- Implementation status: not_started
- Specification branch: `codex/statement-model-selection`
- Implementation branch / commit: отсутствуют
- Review baseline: `7be572c`, integrated [PR #66](https://github.com/QahNaaRlN/grace/pull/66), основан на main `1ada384`; собственный diff этой ветки содержит только документацию поверх baseline
- Baseline CI: historical `12ef5a0`, run `39175297738`, 2026-10-19 — все пять jobs PASS; проверки новой редакции PR #66 отдельны. PR #66 integrated `06169e5`; production всё ещё на `1ada384`, этот draft SPEC-0056 не принят
- Зависимость: принятые [SPEC-0055](0055-bank-statement-pdf-import.md), [технический контракт](0055-technical-contract.md), существующие statement consent и общий usage ledger

## Проблема и пользовательский результат

Владелец не должен вручную подбирать модель для каждого банковского PDF.
Grace автоматически выбирает подходящую проверенную модель, показывает её и
предельную стоимость до передачи данных, затем сохраняет выбор на весь attempt.
Результат остаётся проверяемым черновиком. Финансовые расчёты и запись выполняют
Go и существующий confirmation flow.

Автовыбор не лечит неизвестный расход предыдущего запроса. При uncertain usage
новые inference calls блокируются независимо от выбранной модели.

## Текущее поведение и доказательства — 2026-10-19

`internal/statementextract/plan.go: Policy.Valid` допускает только Gemini 2.5 Flash
через точный `google-vertex/eu`. `internal/assistant/policy_verifier.go` обновляет
metadata закреплённой пары, но не выбирает другую. Consent связывает конкретную
policy, source state, redaction, prompt/schema и maximum plan cost.

Production `1ada384` развёрнут, Statement AI false. На кандидате PR #66 columns
synthetic smoke PASS, cards FAIL. Расход provider failure точно сверён
2026-10-19 12:38:04 +05:00 (run `39217226634`). После сверки v7 не прошёл
coverage, v8 дал регрессию и отклонён. [Activation evidence](0055-activation-evidence.md) отделяет эти результаты
от общего benchmark и owner acceptance. Ни Gemini, ни другой кандидат не получают
qualified status только из факта наличия в каталоге. На момент draft пригодный
для автоматического выбора пул отсутствует.

## Scope

1. Серверный реестр не более трёх явно разрешённых text-only model/endpoint пар
   для purpose `statement`, с ценовыми ceilings и версионированным quality evidence.
2. Детерминированный подбор до consent: privacy/capability/quality eligibility,
   свежая metadata, стоимость полного bounded plan, доступность общего бюджета.
3. Сохранённое объяснение выбора и конкретная policy в preview/consent/attempt.
4. Один закреплённый маршрут на attempt, безопасный отказ при изменении условий,
   ручной review и раздельное финансовое подтверждение.
5. Offline/fake-provider проверки до любой платной qualification.

## Non-goals

- Выбор модели самой LLM, произвольная маршрутизация по всему публичному каталогу,
  использование `:auto`/бесплатных/непроверенных aliases.
- Незаметный fallback после consent, переключение провайдера после отправки
  первого чанка, смешивание ответов разных моделей внутри attempt.
- Увеличение общего бюджета/fees, обход uncertain ledger, обнуление расхода,
  автоматическая reconciliation без authoritative evidence.
- Изменение receipt/analysis/Jev policies, подключение TypeSafe, OCR/vision,
  кредитных условий, финансовой арифметики или подтверждения проводок.
- Самостоятельное production enable. Полный AC12 SPEC-0055 сохраняется отдельно.

## Связанная проработка качества — 2026-10-19

Владелец отдельно поручил проработку устойчивости обоих formats и расширенного
корпуса и принял её результат 2026-10-19 («Принимаю», PR #68). Это принятие
исследования и последовательности пакетов, а не конкретной редакции SPEC-0056,
реестра моделей или платного campaign. [Design study](../research/statement-quality/README.md) содержит
32 выполненных offline probes (0 model calls), подтверждённую нестабильность
record packing, 24-case corpus manifest (2 existing / 22 planned) и этапы
planner → coverage protocol → corpus → bounded evaluation → registry.
Дополнительный model-selection код не реализован; статус этого контракта draft.

Follow-up 2026-10-19: [S1 candidate](0055-record-atomic-planning.md) после
[280 offline cases](../research/statement-quality/atoms.md) предлагает один
whole-document call с aliases и bounds 100000/16000 вместо 10000/2000.
Его theoretical provider ceiling — 70000 micros/call при прежней pinned policy.
Старые per-run/campaign envelopes этого draft нельзя считать утверждённым
бюджетом нового profile; до SPEC-0056 acceptance их нужно пересчитать по
принятому planner и preflight corpus. Реестр моделей и paid runs не разрешены.

## Реестр и qualification

Реестр — защищённая операторская конфигурация с неизменяемыми версиями;
пользовательская сессия не может добавлять модели, повышать ceilings или
помечать кандидата как проверенного. Публичная metadata используется только
для верификации записей реестра, не для автоматического включения новых моделей.
Оператор управляет допуском; пользователю не требуется выбирать модель вручную.

Запись содержит exact model ID и endpoint tag, ожидаемое provider display name,
text-only/no-tools/reasoning-disabled профиль, ceilings, версию privacy policy,
qualification report digest и состояния `unqualified`, `qualified`, `disabled`.
Каждая пара модель/endpoint квалифицируется отдельно. Региональный tag не
выдаётся за гарантию юридической резидентности всей цепочки.

Eligibility требует одновременно:

- Явный operator allow-list, `qualified`, срок evidence не старше 7 суток;
  непротиворечивые model/endpoint identities и отсутствие отзыва qualification.
- Evidence относится к текущим prompt/schema/redaction/extractor versions и
  транспортному профилю. Изменение этих версий требует новой qualification.
- Свежая metadata, не старше 24 часов: exact route доступен, поддерживает strict
  structured output, response format и token cap; подтверждены цены в ceilings.
- Запрос enforce-ит `zdr=true`, `data_collection=deny`, `only=[exact endpoint]`,
  `allow_fallbacks=false`, без tools/plugins/raw PDF/images. Requested и actual
  model/provider проверяются; совместимость не предполагается по имени бренда.
- Прежние границы SPEC-0055: ≤10000 input bound, ≤2000 output tokens, ≤32 calls,
  messages+schema ≤9000 bytes, один call ≤30 s, attempt ≤120 s. Input/output price
  ceilings не выше соответственно 300000/2500000 micros за 1M tokens.

При отсутствии immutable upstream revision фиксируются доступные identity и
metadata digest. Это не гарантия неизменности поведения провайдера: evidence
истекает, изменение наблюдаемой identity или провал regression отзывает допуск.

Минимальный versioned qualification corpus: оба layouts columns/cards на русском
и английском (не менее четырёх synthetic документов). Вместе они проверяют
operation/booking clocks, account/FX currencies, отрицательные/положительные
строки, заголовки/переносы, missing book balances и available balance отдельно.
Golden negative cases проверяют честное abstention при отсутствующей валюте,
дате или противоречивых источниках. Literal values и роли задаёт golden corpus,
не оцениваемая модель и не другая LLM.

Допуск требует 100% обязательных golden checks по числу/идентичности строк,
деньгам, валютам, направлениям, отдельным часам, source grounding/coverage,
book/available roles и корректным unknown/blockers в negative cases; ноль
выдуманных значений и пропущенных финансовых строк. Прогон только двух старых
smoke cases недостаточен для допуска. Description/omission/abstention metrics,
latency и billing также сохраняются; ограничения корпуса явно обозначаются.

Любая платная qualification — отдельный явно запущенный bounded run, не фоновая
часть пользовательского preview. Предлагаемый потолок одного такого run:
≤32 calls, ≤256000 micros ($0.256), целиком в прежнем общем бюджете с fees,
без пополнения лимита. До сетевого inference резервируется весь план; если весь
корпус выбранных кандидатов не помещается, run отказывает до первого call.
Весь qualification run последовательный, ≤120 s; неполное выполнение не
получает PASS. Это верхние границы, не обещание завершения 32 calls за 120 s.
Повтор — отдельный run, а не скрытое увеличение этого ceiling. Истёкший/неполный
report не становится qualified. Per-run cap — граница shard, не обещание
полной qualification за один run. Complete campaign manifest и общий cap
утверждаются отдельно; частичные shards не дают qualified. Design study
предлагает три повтора на 22 text cases: это 66 scheduled document attempts;
одни 16 positive cases требуют минимум 48 calls, поэтому один 32-call run
недостаточен. Negative cases могут отказать до сети. Proposed ceiling полного
campaign — ≤198 calls / ≤1.584 USD на одну пару при ≤3 calls/case; это UNVERIFIED working
envelope до corpus preflight и кандидат для review, не разрешённые расходы. Нельзя автоматически разбить
неутверждённый большой benchmark на серию разрешённых малых runs. В этой документационной работе paid calls нет;
старый unknown billing уже сверён; любой новый unknown billing блокирует run.

## Выбор и стабильность плана

1. Проверить principal/document ownership, global/statement flag, общий ledger
   и актуальную operator policy. При uncertain usage — `usage_requires_review`;
   другие модели не просматриваются как способ обойти блокировку.
2. Получить eligible пары и построить для каждой полный план по тому же source
   state и redaction. Вариант с превышением bounds, времени или стоимости
   исключается. Metadata timeout закрывает этот вариант; никакой probe inference.
3. Среди подходящих выбрать минимальный `maximumPlanMicros`, вычисленный Go
   целочисленно из ceilings. При равенстве — меньшую p95 latency из закреплённого
   qualification report, затем лексикографические exact model ID и endpoint tag.
   Для одинакового набора входных снимков выбор воспроизводим. Качество и
   приватность — обязательные gates, цена не компенсирует их провал.
4. Сохранить selection/version/report digests, причины исключения в безопасных
   кодах и победителя; сформировать preview. Все расходы выражены integer USD
   micros; это единицы metering, не валюта финансовой книги пользователя.
5. Consent связывает owner/document/redacted state, model/endpoint, ceilings,
   selection algorithm version, qualification/privacy digests и plan hash.
   Ровно этот plan атомарно резервируется в общем ledger перед первым call.
6. Перед исполнением перепроверить выбранную пару и consent. Отзыв/истечение
   qualification, смена цены/identity/privacy/source state или превышение плана
   отменяет возможность исполнения: новый preview и новое согласие. Обновление
   других кандидатов не меняет уже закреплённый plan автоматически.
7. На всех чанках одна policy. HTTP/timeout/protocol/route/quality failure
   прекращает attempt, сохраняет подтверждённый billing отдельно от непригодного
   ответа, оставляет manual review. При неизвестном расходе — uncertain и общий
   запрет дальнейших платных calls до точной сверки.

Конкурирующие attempts не могут зарезервировать один остаток дважды. Изменение
remaining между preview и запуском даёт budget error до inference. Повтор с
прежним idempotency key возвращает прежний результат/состояние без платного
replay и нового выбора. Пользовательский повтор после ошибки — новый preview,
новое согласие и новая reservation только после settlement предыдущего attempt.
Ранее учтённый расход не исключается из общего бюджета.

## API и UI

OpenAPI остаётся единственным client contract. Реализация расширит существующий
statement preview безопасным selection summary: способ `automatic`, exact
model/provider, maximum plan cost и краткая причина выбора; внутренний реестр,
ключи и чужие usage records наружу не возвращаются. Consent/attempt contract
получает effective selection/version hashes, сохраняя ownership/RLS/idempotency.
Конкретные DTO/ошибки добавляются в `api/openapi.yaml` с регенерацией TypeScript,
без ручных дубликатов. Старые previews не мигрируют в новое согласие молча.

На телефоне перед согласием: «Модель подобрана автоматически», модель/провайдер,
предел стоимости и передаваемые данные. Основная кнопка разрешает этот plan.
Если подходящих моделей нет, доступно ручное сопоставление; если нужен новый
plan, UI объясняет причину и запрашивает новое согласие. Селектор в React не
вычисляет рейтинг/стоимость и не меняет policy. Даже успешное извлечение не
создаёт Posting: перед записью остаётся существующий финансовый preview.

## Данные, архитектура и rollback

Selector — чистая Go-политика в capability `statementextract`; получение
metadata/реестра и композиция транспорта через interfaces, без нового сервиса
или общего слоя controllers/services/repositories. Финансовые модули не зависят
от selector. Immutable effective selection и digests сохраняются с существующим
preview/consent/attempt; только metering разрешено писать в общий usage ledger.
При реализации нужны аддитивная миграция, tenant isolation и сохранение истории.

Qualification reports — synthetic only, защищённое storage; CI/Git не получают
production budget/remaining/cost/token/generation values или private provider
payload. Конфигурация реестра защищена operator/root с mode 0400/0600;
квалификация не является разрешением на передачу личного PDF.

Отдельный selection flag выключен по умолчанию. Rollback выключает автовыбор
и сохраняет manual path. Закреплённый старый маршрут SPEC-0055 допустим лишь
при собственной действующей policy, flag, consent и отсутствии budget freeze;
rollback не запускает его автоматически. Usage/consents/history не удаляются,
неизвестные расходы не обнуляются. Downgrade с несовместимыми сохранёнными
selection records обязан отказать, а не потерять audit.

## Критерии приёмки и исполняемые проверки

Все строки пока `NOT_RUN` для SPEC-0056; тесты ниже — требования к реализации,
не отчёт о существующем поведении.

| AC | Наблюдаемое поведение | Будущая executable evidence |
| --- | --- | --- |
| AC1 | Из нескольких qualified моделей выбирается самый дешёвый полный план; ties воспроизводимы | Pure selector table tests: разный порядок registry, стоимость, latency, lexical tie |
| AC2 | Дешёвый, но unqualified/stale/unsafe/unsupported вариант не выбирается; пустой пул → manual | Fake metadata и registry tests; ноль provider calls для всех отказов |
| AC3 | Price/identity/version/source change не выполняется под старым consent | Existing preview/consent integration с изменениями между preview и attempt; ноль inference |
| AC4 | На всех чанках один маршрут, нет fallback/paid replay | Fake provider call trace: сбой чанка, replay одного key, повтор с новым plan/consent |
| AC5 | Unknown usage блокирует любые модели; конкурентные reservations не превышают общий лимит | PostgreSQL integration: uncertain → выбор/запуск запрещён, shared-ledger races, известный/неизвестный billing при protocol failure |
| AC6 | Qualification связана с golden corpus и версиями, не с публичной популярностью | Offline scorer tests: columns/cards RU/EN, один критический дефект → FAIL, истечение/revoke/смена версии → ineligible |
| AC7 | Выбор не меняет grounding/coverage/деньги и не создаёт финансовую запись | Real helper → fake provider → SQL review/confirmation, negatives, owner isolation |
| AC8 | UI показывает конкретную модель/предел до consent и понятный ручной путь | Mobile web tests; прежние consent при новой модели не переиспользуются; manual работает без AI |
| AC9 | Совместимый rollout/rollback сохраняет audit; OpenAPI и TS согласованы | Migration/restore/rollback tests; generated contract diff; общий regression suite |

Общие implementation gates: `make test`, `make test-integration`,
`make test-production`, `go vet ./cmd/... ./internal/...`, `git diff --check`.
Live qualification подтверждается отдельно с run ID, corpus/report digests и
приватным exact billing evidence. Успех fake-provider suite не означает model
quality, owner acceptance или production enable.

## Порядок и решения для review

1. Принять этот конкретный draft и интегрировать его; направление «автовыбор»
   само по себе не меняет accepted pinned policy SPEC-0055.
2. Offline registry/selector + API/consent + ledger/rollback tests в отдельном
   implementation worktree. Никаких paid calls или production flag changes.
3. Старый расход сверён. Далее — отдельно запущенная qualification
   разрешённых пар в прежнем бюджете; при отсутствии прошедших кандидатов
   автовыбор честно недоступен. Конкретные дополнительные model IDs не считаются
   разрешёнными по этому draft и добавляются через operator-reviewed registry.
4. После executable evidence — CI/Delivery и отдельный production enable с
   owner functional check. До этого SPEC-0055 manual path сохраняется.

Предлагаемые решения: quality TTL 7 суток, metadata TTL 24 часа, максимум три
пары в реестре, ранжирование по plan ceiling/latency/identity и qualification run
≤32 calls / $0.256. Поддерживаемые capabilities и bounds выше обязательны;
этот draft не содержит измерений качества новых моделей и не повышает бюджет.
