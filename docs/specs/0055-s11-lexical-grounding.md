# SPEC-0055 / S1.1: lexical monetary eligibility и record descriptions

- Дата: 2026-10-19 (Asia/Yekaterinburg).
- Статус: **accepted**, 2026-10-19; прямое решение владельца «Утвердить S1.1, один smoke и включение при PASS».
- Implementation: **not_started** на момент принятия; разрешена реализация и один новый smoke по приведённым ниже границам. Production versions ещё не изменены.
- Основание: цель владельца — включение Statement AI для собственных production-тестов; отдельно принятые [S1](0055-record-atomic-planning.md) и [owner-test activation](0055-owner-test-activation.md).
- Baseline: production `711a13d` (PR #71), deploy `39248364426`: internal health PASS; один разрешённый smoke завершился quality FAIL на columns, cards NOT_RUN, flag false. Автоповторов нет.

## Установленные факты

Safe read-only diagnostic `39249075304` экспортировал только canonical locators
двух разрешённых public fixtures, без source literals/usage/cost/generation IDs.
[Сохранённые refs](../research/statement-quality/owner-smoke-2026-10-19-locators.json)
и [offline probe](../research/statement-quality/owner-smoke-candidate-offline.json)
связаны с run `39248364426` и исходным generator/реальным pinned helper.

На columns модель вернула **3 rows вместо 2**. Третья accountAmount selection
содержит три обычных текстовых фрагмента description и один настоящий знак
минус. Старый RequestSchema допускает любые opaque `[text_<alias>]` в денежный
enum; три нечисловых фрагмента поэтому schema-eligible. Go money validation
отклонил их; row count/turnover/blockers дали quality FAIL, финансовых writes нет.

Descriptions первых двух rows ссылаются на чужие блоки: заголовок страницы и
описание предыдущей операции. Все source spans классифицированы, поэтому coverage
сам по себе этого не обнаружил. Существующий smoke scorer не проверяет exact
record binding descriptions; его нужно **усилить**, а не ослабить ради PASS.

Это не доказательство, что aliases вызвали ошибку. Doc-order aliases — отдельная
гипотеза упрощения references; её semantic effect пока UNVERIFIED. Строгий
lexical monetary enum детерминированно исключает именно три подтверждённых
невозможных text-as-money references, но не гарантирует правильный ответ модели.

## Предлагаемая техническая редакция

### 1. Server-owned lexical class

В `RedactedSpan`/OpenAPI добавить required `lexicalClass` с двумя значениями:
`numeric_fragment` / `other`. Класс вычисляет Go из **original token**, до
redaction; printed placeholder-подобный текст никогда не создаёт numeric flag.
Это shape hint, не финансовая роль, значение, confidence или подтверждение суммы.

Нормализовать только U+2212 к ASCII minus и NBSP/NNBSP к обычному пробелу для
shape predicate. Предикат допускает отдельный знак +/−/- либо token без currency
suffix, отвечающий существующей числовой форме: необязательный sign; целые digits
или группы `1–3 digits + (space + 3 digits)+`; optional decimal comma/dot с 1–2
digits. Даты, clocks, headings, merchant words, exponent, скобки телефона и
literal `[number_s0]` имеют `other`. Ограничение длины token — 80 characters;
value/overflow/currency/clock/reconciliation по-прежнему проверяются финансовым Go.
Не применять новые округления, валюту по умолчанию или bank-specific parser.

Все прежние denial/visibility rules сохраняются: account/phone/email fragments
не становятся видимыми. Manufactured opaque placeholder получает вид
`[number_<alias>]` для hidden numeric_fragment и `[text_<alias>]` для hidden other.
**Новые raw digits/имена не передаются**; дополнительно раскрывается только факт
числовой формы скрытого fragment. Это явное изменение privacy/preview profile,
требующее нового consent. Literal placeholder-подобный текст остаётся ordinary
source text; нельзя определять класс через prefix строки.

`RequestSchema` включает в moneySelection только aliases с server-owned
`numeric_fragment`, включая отдельные знаки. Он больше не считает все opaque
text monetary candidates. `CanonicalProposal` проверяет тот же typed eligibility
для row account/transaction amounts и всех header monetary fields **до merge**,
даже если provider проигнорировал JSON Schema. Selection со знаком без числа
по-прежнему отклоняет существующий money parser. Другие fields/coverage/service
не заполняются автоматически, unknown/invalid evidence остаётся видимым blocker.

### 2. Aliases в порядке переданных source spans

Сохранить один whole-document atom и исходный порядок `Document.Spans`.
ID с индексом i в этом порядке получает `s` + lower-case base36(i), без ведущих
нулей. Canonical ID больше не сортируется лексикографически перед нумерацией.
Map остаётся биекцией, source values/geometry/IDs не меняются. Chunk ordering и
alias index теперь согласованы; это упрощает references, а не доказывает grouping.
Все source spans включены ровно один раз. Фактические input bytes измеряются после
нового prompt/schema; overflow отказывается до consent/reservation/call.

`lexicalClass` хранится в hashed plan/preview, но `Messages` сохраняет tuple
`[id,page,x,y,text]`. Модель получает lexical hint через placeholder и monetary
enum, без canonical map и без JSON service metadata из plan.

### 3. Новый version/consent/report profile

- Planner `statement-document-atom.v2.lexical-grounding`.
- Alias `statement-alias.source-order-base36.v2`.
- Redaction `statement-allowlist.v3.lexical-placeholders`.
- Prompt `statement-span-mapping.v10.lexical-records`.
- Proposal wire fields прежние; dynamic monetary enum и schema descriptions
  обновлены, SchemaHash вычисляется из новой schema. Prompt явно отличает page
  headings/description от amount, сохраняет printed record grouping и не создаёт
  transaction solely to consume остаток. Нераспознанные поля не выдумываются.
- Новый report `statement.synthetic.v3.lexical-records`; `CurrentProfile` включает
  отдельный `ScorerVersion`, а summary требует текущие versions/hash/limits.
  Legacy plans/consents/reports не разрешают новые calls/enable. Исторические
  attempts остаются читаемыми; known/unknown billing и allowance/fees сохранены.

### 4. Независимый усиленный synthetic scorer

Для обоих shipped PDF кроме прежних checks сравнить **каждую row description**
с независимым printed fixture/generator oracle, после штатной Go нормализации.
Никаких ожиданий из ответа модели. Неверное page/header/other-row description,
перестановка descriptions, лишняя/пропущенная row и plaintext-as-amount → FAIL.
Public fixture с embedded instruction проверяет буквальное сохранение description
как данных, не исполнение инструкции. Это scorer только fixed synthetic corpus,
не bank-specific production parser. Старые 17/18 checks не удаляются; после
двух description checks required totals становятся **19 / 20**.

## Лимиты и разрешение на следующий smoke

S1 bounds сохраняются: 2000 spans / 90000 messages+schema bytes;
100000 input / 16000 output, 120 s/call, product attempt 150 s, HTTP 180 s.
Default analysis/receipt transport не меняется. Общий allowance/fees не растут.

Если владелец отдельно утвердит этот контракт **и один новый smoke**: только
columns → cards, не более 2 calls / **$0.14 aggregate provider ceiling**, 300 s/run,
whole-run reserve до первого вызова. Предыдущий FAIL не стирается. Нет repair,
необъявленных повторов, перехода на другую модель или использования «оставшегося»
call как разрешения пробовать снова. При любом FAIL flag false; новый paid run
снова требует решения. Enable для owner testing — только после обоих полных
PASS усиленного scorer, CI/Delivery/deploy и actual-image/runtime/public checks.
Личные документы не участвуют. Полная AC12/S2–S5/SPEC-0056 qualification отдельно.

## Исполняемые критерии приёмки

| ID | Требование / evidence |
| --- | --- |
| L1 | Direct source shape: integer/grouped/decimal/sign, обе Unicode spaces; dates/clocks/currency/phone punctuation/literal placeholders/merchant text не numeric. Denied raw values не раскрываются даже с visibility |
| L2 | Monetary schema eligibility и Go role check совпадают; saved failed response содержит 3 impossible refs, все отвергаются до draft при known billing settlement; собственно знак eligible, но сам по себе не valid money |
| L3 | Source-order map exact bijection и inverse по всем typed selections; no source mutation/loss/duplication на S1 layouts/stress; exact byte/span caps и old consent/version rejection |
| L4 | Fake-provider только по actual delivered tuples → canonical → Go → оба fixed scorer PASS; old wrong description/title/extra-row response FAIL; before financial confirmation ноль financial writes |
| L5 | Scorer descriptions independent от model output, требуемые checks 19/20; report/summary/scorer-version mutation/legacy negatives, protected audit и billing invariants |
| L6 | OpenAPI → regenerated TS, `make test`, `make test-integration`, `make test-production`, `go vet ./cmd/... ./internal/...`, `git diff --check` |
| L7 | Отдельно разрешённый один smoke: оба strengthened cases PASS; exact-SHA CI/Delivery/deploy, actual API image/runtime flag/current policy, public/auth smoke; иначе disabled |

## Offline evidence и его границы

Real helper + network-none/read-only UID10001 probe, **0 inference / 0 SQL**:
columns 46 spans, money candidates 22→10, input estimate 7769→8138 bytes;
cards 57 spans, candidates 23→9, 8108→8479 bytes. Все source IDs сохранены
ровно один раз, оба baseline помещаются в прежний 90000 cap. Эти размеры —
prototype recipe с явным proposed prompt/schema, не утверждение о финальной
production реализации или реальном token count. 84-layout production proof
нужно повторить после принятия/реализации; semantic quality остаётся UNVERIFIED.

Воспроизведение: `sh scripts/research/probe-statement-smoke-candidate.sh /tmp/grace-smoke-candidate.json`.
Первичные schema ограничения остаются описанными в accepted S1; новые JSON Schema
features не добавляются. Tightening enum/новые aliases/lexical hints всё равно
нуждаются в actual endpoint smoke, metadata сама по себе этого не доказывает.
