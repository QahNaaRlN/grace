# SPEC-0055 / S1.13: наблюдаемость canonicalization без нового inference

- Статус: accepted; implementation_status: implemented (offline); live_status: not_authorized.
- Автор: Codex; дата: 2026-10-21, Asia/Yekaterinburg.
- Acceptance owner: владелец Grace; acceptance record: 2026-10-21, прямое «одобряю, приступай» на O1–O5 offline.
- Основание подготовки: «приступай к формированию спеки». Разрешена подготовка
  документа; последующее принятие O1–O5 записано до кода. Live run не разрешён.
- Worktree: `statement-protocol-observability-spec`, branch
  `codex/statement-protocol-observability-spec`.
- База: main `f4fc56a0db711f6e44e0d87cf71495c0da2ac20d`, PR №87.
- Implementation branch: `codex/statement-protocol-observability`; code commit
  `5f0a55aa2304e73bfcfdfd1d0814e29f3d1563e6`; [O1–O5 evidence](0055-s113-evidence.md).

## Проблема и пользовательский результат

[S1.12 R5](0055-s112-evidence.md) установил canonicalization FAIL единственного
C6 A/columns. Wire decode завершился, но alias-proposal не сохранён до
`CanonicalComparisonProposal`: точный guard **UNVERIFIED**, replay невозможен.
Из старого report прочитаны cost1995micros ($0.001995 provider) и Complete
latency2931ms; остальные5 tuples not_sent, completed0. Сравнения A/B нет.

Результат S1.13: следующий отдельно разрешённый synthetic run сможет сохранить
ограниченный typed alias-proposal **до** canonicalization и закрытую причину
отказа, а offline replay подтвердит её либо явно даст UNVERIFIED. Пакет даёт
наблюдаемость протокола; он не исправляет отсутствующий исторический ответ и
не доказывает качество модели или правильность финансовых данных.

## Scope

1. Закрытая структурная диагностика тех же guards canonicalization.
2. Приватный bounded snapshot typed alias-proposal для comparison на прежних
   трёх публичных synthetic fixtures, при успешных Complete/decode/route/billing.
3. Версионированный diagnostic envelope с неизменённым v1 comparison report,
   строгим validation/digest и безопасной summary/inspection проекцией.
4. Offline replay и причинные regression/privacy/compatibility tests.
5. Evidence O1–O5, exact source и gates перед будущим отдельным live решением.

Допустимые места будущей реализации: `internal/statementextract` (общие
предикаты + отдельный comparison diagnostic entry point),
`internal/statementqualification` (comparison observer/envelope),
`cmd/statement-admin` (private writer/closed summary), существующий comparison
inspector и локальные replay/test scripts. Полный список изменённых файлов
предъявляется в implementation report; общий рефакторинг не входит.

## Non-goals

Provider inference/metadata GET, SSH/VPS dispatch, новая authorization/campaign,
paid retry, новый workflow mode, deploy/model/route/price/budget/fees changes,
reconciliation, SQL/financial writes, личные PDF/drafts, product API/UI/OpenAPI,
prompt/schema/redaction/aliases/planner/scorer changes и ослабление guards.
Не расширять S1.12 reader на произвольные пути и не переписывать его fixed v1
источник. Исправление реальной причины и запуск будущего эксперимента —
отдельные решения после принятия и offline evidence этого пакета.

## Текущее поведение и доказательства

На base f4fc56a:

- `internal/statementextract/aliases.go`: invalid plan, nonnumeric monetary
  reference, unknown alias и duplicate alias свёрнуты в `provider_protocol`.
- `internal/statementqualification/comparison.go`: результат Complete сначала
  canonicalize, и только затем присваивается `tuple.Proposal`; обе ошибки
  canonicalization/merge дают `qualification_protocol_failed`.
- `ComparisonReportVersion=statement.comparison.v1.grounding` и строгие
  `DecodeComparisonReport`/`ValidateComparisonReport` не допускают новые поля.
- S1.12 stage canonicalization установлен из verified v1 report и source012f604,
  а не из догадки о содержимом ответа. Raw bytes ответа отсутствуют.
- [PR №87](https://github.com/QahNaaRlN/grace/pull/87) интегрирован f4fc56a;
  [main CI39595212708](https://github.com/QahNaaRlN/grace/actions/runs/39595212708)
  и [Delivery39596315641](https://github.com/QahNaaRlN/grace/actions/runs/39596315641)
  SUCCESS. Это baseline, не evidence ещё не реализованного S1.13.

## Предметная модель и инварианты

### O1. Закрытая диагностика с прежними решениями guards

Отдельный comparison entry point возвращает canonical proposal, прежнюю ошибку
и optional diagnostic. Общие проверяемые предикаты остаются единственными:
не вводить расходящуюся копию canonicalizer для объяснений. Legacy/product
`CanonicalProposal` и прежний comparison entry point сохраняют сигнатуры,
`provider_protocol`, nil/error semantics и каноническое содержимое результата.
Публичные HTTP errors не получают новую детализацию.

Diagnostic содержит только:

| Поле | Закрытый контракт |
| --- | --- |
| version | `statement.canonicalization-diagnostic.v1` |
| stage | `canonicalization` |
| code | `invalid_plan`, `unknown_alias`, `duplicate_alias`, `non_numeric_money_selection` |
| scope | `plan`, `header`, `row`, `header_label`, `service` |
| field | фиксированные поля Proposal, `spans`, либо `plan` |
| ordinal | row/service ordinal с1;0 для header/plan |
| selectionIndex | индекс с1 внутри selection;0 для plan |
| role | для header_label:5 известных balance/turnover roles или `unknown`; иначе пусто |

Приоритет: invalid plan → monetary selections → остальные alias selections.
Monetary порядок совпадает с текущим кодом: opening/closing/available balance,
debit/credit turnover, затем account/transaction amount каждой строки.
Для проверяемого monetary alias отсутствие alias в плане даёт unknown_alias,
существующий nonnumeric — non_numeric_money_selection; оба по-прежнему FAIL.
Повтор numeric alias внутри selection диагностируется на прежнем alias guard.
Остальные поля обходятся в фиксированном порядке текущих row/header/service
полей; header label keys — в лексикографическом порядке только для устойчивого
выбора диагностического поля. Raw key никогда не попадает в публичный role.

Выбор детерминированного diagnostic не меняет acceptance или возвращаемый
canonical proposal. Дубли **между** selections, неизвестный direction или header
role не становятся новыми canonicalization guards: текущие дальнейшие validators
сохраняют ответственность. Nil/empty selections, порядок, дубли и исходный объект
не нормализуются. При PASS diagnostic отсутствует. Merge/grounding/provider errors
не переименовываются в одну из этих четырёх причин.

### O2. Bounded typed snapshot до canonicalization

Snapshot создаётся только операторским comparison путём после успешного
Complete, typed decode и прежних проверок route/known billing; до вызова
canonicalization и до возможного уничтожения исходного proposal. Не сохраняются
HTTP bytes/envelope, provider raw errors или документы. Включаются исходные
rows/header/service Proposal с alias strings и free-text service reason **только
в приватном artifact**, deep copy без изменения объекта reasoner.
Это capture в памяти до проверки, с записью при прежней финализации campaign;
новый crash-durable journal и гарантия сохранности при убийстве процесса не входят.

Eligibility фиксирована:3 прежних fixture hashes, exact comparison policies,
profile, document/plan/request hashes. Нельзя включить capture через product API,
owner request или конфигурацию для личного PDF. Для иных путей storage отсутствует.

Лимиты snapshot: JSON<=32768bytes/tuple, rows<=5000, service<=2000,
selection<=2000 elements, alias/direction string<=64bytes, header role key<=128bytes,
service reason<=2000bytes, depth<=20. Сохраняются nil/empty, порядок массивов и все дубли; порядок JSON object keys
не является частью typed Proposal.
При превышении любого лимита snapshot **целиком отсутствует**, никаких первых N
элементов/обрезанных strings; captureStatus=`unavailable_limit`. Диагностический
код, если получен прежним guard, сохраняется независимо от snapshot.

Capture status: `not_reached`, `captured`, `unavailable_limit`, `unavailable_report_limit`.
Not_sent/provider-failed-before-successful-Complete не получают snapshot.
Если known route/billing success достигнут, provider failure stage не фабрикуется
из ошибки capture. Capture не влияет на call count, продолжение/остановку campaign,
known/uncertain, reserve/settle или paid caps. Потеря snapshot не становится PASS.

### O3. Envelope и приватность без миграции v1

Предлагается новый private envelope `statement.comparison.observed.v1`:
`version`, `report` (прежний v1 ComparisonReport), `observations` (ровно6 строк
в прежнем tuple order), `digest`. Observation содержит tuple ordinal, stage,
captureStatus, optional snapshot и optional closed diagnostic O1.
Observation stage: `not_reached`, `canonicalization_passed`,
`canonicalization_failed`. Not_sent и provider/billing/route отказ до допуска
proposal всегда not_reached/not_reached без snapshot/diagnostic; canonical FAIL
всегда имеет O1 diagnostic, PASS — никогда. CaptureStatus captured эквивалентен
наличию полного snapshot; unavailable statuses snapshot исключают. Ни stage,
ни captureStatus не заменяют прежние state/reason основного tuple.
Внешний digest по прежнему Go hashing contract покрывает весь envelope с пустым
внешним digest; внутренний v1 digest проверяется независимо. Нельзя дописывать
новые поля в `statement.comparison.v1.grounding` или молча менять его digest.

Private writer/summary/существующий comparison inspector явно различают bare v1
и новый envelope. V1 остаётся строго валиден по прежнему контракту и означает
`observationUnavailableLegacy`, а не отсутствие model call или PASS. V1 artifacts,
markers, UUID/generation IDs, ledger и historical source hashes не мигрируют.
S1.12 fixed reader продолжает читать прежний C6 v1 без изменений.

Общий artifact<=524288bytes, strict JSON без duplicate keys/trailing values,
bounded arrays/depth. Если snapshot(s) делают envelope слишком большим, удалять
их целиком в обратном tuple order, с `unavailable_report_limit`, пока размер
допустим; report, diagnostic и accounting не обрезать. Если базовый report и
закрытые observations сами превышают предел, artifact verification UNVERIFIED;
не выдавать частичный файл за валидный или делать известный расход unknown/0.
Такой отказ записи проверяется offline; повтор provider call запрещён.

Diagnostic envelope пишется существующим operator private writer с0600; никаких
новых storage backend/DB fields. Новых имён campaign/authorization или разрешения
перезаписать consumed marker/report пакет не создаёт. Прежний workflow default-off,
exact-source gate, exclusivity, one-shot и accounting сохраняются.

Публичные stdout/stderr/Actions/artifacts получают только O1 enums/ordinals,
captureStatus/bools/counts. Никаких alias strings (в том числе неизвестных), raw
role/service/model/source/error text, proposal, owner/reservation/generation IDs.
Числовая cost/latency проекция не расширяется этим пакетом: остаётся в отдельно
принятом S1.12 reader фиксированного C6. Новый envelope не даёт writer consent
или production activation. Privacy sentinel должен отсутствовать во всех
публичных каналах независимо от ошибки validation/serialization/replay.

### O4. Offline replay и неизменность поведения

Replay получает только offline test envelope и pinned public fixture/plan.
Проверяет оба digests, profile/policies/hashes и snapshot bounds. При captured
snapshot повторяет тот же canonicalizer и сопоставляет error+closed diagnostic;
при PASS дополнительно сверяет canonical proposal с прежним результатом.
При любом mismatch — UNVERIFIED, без repair/retry и без вывода исходного snapshot.
Отсутствующий/legacy/overflow snapshot означает replay unavailable, не guessed PASS.

Нужны причинные cases всех4codes, их сочетаний/приоритетов и положительные
near-misses: numeric alias, nil/empty selections, duplicate across fields,
unknown role/direction, несколько ошибочных labels. Baseline expected outcomes
фиксируются из поведения f4fc56a; не сравнивать новый код только с самим собой.
Fakes получают только доставленные tuples/aliases, не golden ответы. Настоящий
pinned helper в network=none на columns/cards/RU подтверждает hash/profile и
неизменность C1–C5/S1.10 guards. Отдельно сохраняются старые S1.8/S1.12 regressions.
Никаких model/key/catalog/SQL/server calls в replay или tests этого observer.

### O5. Offline gates и evidence

Implementation report предъявляет accepted spec/acceptance commit, actual diff,
exact source, O1–O5 matrix и исполняемые результаты. Минимум `make test`,
`make test-production`, затронутый real-helper/bank suite, `go vet`,
`git diff --check`; exact implementation head CI всех5jobs PASS до рассмотрения
нового live решения. CI может использовать прежние disposable integration DB;
observer/replay не вызывает SQL. Synthetic fake calls не считаются live quality.

## API и OpenAPI

Публичный HTTP API/DTO/OpenAPI не меняется; TypeScript regeneration не требуется.
Новая версия — private operator artifact, не product consent или HTTP response.

## Данные, миграция и rollback

Только операторский private envelope будущего отдельно разрешённого run.
Миграций PostgreSQL и финансовых записей нет. Legacy artifacts неизменны.
Rollback к прежнему writer убирает новые diagnostics; v1 embedded report можно
читать отдельно только после строгой проверки envelope. Не удалять snapshot,
marker или сведения о расходе для восстановления разрешения campaign.

## UI

Изменений пользовательского UI нет.

## Ошибки и негативные сценарии

Invalid plan/unknown alias/duplicate/nonnumeric, multi-error ordering, unknown
labels/direction, oversized/unicode strings, depth/duplicate JSON keys, corrupted
inner/outer digest, observation/tuple mismatch, false stage/captureStatus,
legacy v1 без snapshot, partial artifact, reader privacy failures, snapshot
mutation после Complete и envelope size overflow должны давать проверяемое
закрытое состояние. Не читать provider key для объяснения локальной ошибки.
Исторический C6 остаётся canonicalization FAIL / exact guard UNVERIFIED.

## Критерии приёмки

| ID | Критерий и evidence будущей реализации |
| --- | --- |
| O1 | Таблица4guards+priority/near-miss cases: legacy error/decision/canonical output неизменны; diagnostic deterministic/closed; исходный proposal не мутируется |
| O2 | Настоящий RunComparison с causal offline reasoner: snapshot существует до canonical FAIL, deep copy; route/billing/provider/not_sent boundaries; limits дают whole-snapshot unavailable и не меняют calls/accounting/stop policy |
| O3 | V1 bare и новый envelope независимо валидируются; inner/outer digest, tuple consistency, unknown fields/duplicates/size negatives; byte-preservation старых artifacts и closed public privacy sentinel matrix PASS |
| O4 | Pinned real-helper replay всех4causes и positive cases; mismatch/legacy/overflow => UNVERIFIED; прежние fake quality19/20/23 и S1.10/retained regressions PASS; live inference0 |
| O5 | make test, production mocks, relevant helper/bank suite, vet/diff PASS; implementation exact-head CI5/5, evidence и scope review; новых dispatch/provider metadata/inference/deploy0 |

## Исполняемые проверки

Точные команды новых test packages/CLI фиксируются в implementation evidence,
до live decision. Обязательные существующие команды перечислены в O5;
проверки должны реально вызывать общий canonicalizer, comparison runner и
writer/summary/inspector, а не вручную созданные признаки успешного capture.
Для offline envelope используются только явно помеченные synthetic fixtures;
они не подменяют отсутствующий исходный C6 response.

## Решение владельца

Владелец прямо принял **S1.13 O1–O5 только offline**, включая private typed snapshot
с указанными лимитами, новый envelope и закрытую диагностику без изменения guards.
Выбранные version/limits/error priority включены в принятие; несовместимое
изменение возвращается на review до реализации.

Новый платный эксперимент не включён и не спроектирован как автоматическое
продолжение: позднее потребуется отдельный контракт campaign/call count/caps,
новая authorization, exact source и отдельное прямое разрешение владельца.
Оригинальный C6 `s111-comparison-20261021` остаётся consumed.
На момент принятия: implementation_status=not_started, live_status=not_authorized.
