# SPEC-0055 / S1.12: read-only диагностика сохранённого C6

- Дата: 2026-10-21, Asia/Yekaterinburg.
- Статус: accepted; implementation_status: implemented (R1–R4 offline PASS); diagnosis_status: completed (R5 PASS; canonicalization guard UNVERIFIED).
- Основание подготовки: «работаем дальше» после предложения разобрать сохранённый
  C6 trace без новых платных вызовов. Это разрешение подготовки и доступных чтений;
  acceptance записан ниже отдельным прямым решением владельца.
- Worktree: `statement-comparison-diagnosis-spec`, branch
  `codex/statement-comparison-diagnosis-spec`, база `8fc5b12` (documentation PR №85).
- Executed campaign source: `012f604033de9c6a75bf531b8f715323d0fda57d`.
- Зависимости: [S1.11](0055-s111-model-comparison.md),
  [C6 evidence](0055-s111-c6-evidence.md); пакет документации №85 интегрирован `4785b0c`.

## Проблема и ожидаемый результат

Единственный C6 [run39587559021](https://github.com/QahNaaRlN/grace/actions/runs/39587559021)
остановлен после одного A/columns: `qualification_protocol_failed`, до первого score.
Остальные5 tuples not_sent; known/settled, report=ledger, sharedUncertainBlock=false,
image/health preserved. Разрешение paid campaign израсходовано.

Public CI не показывает proposal presence, cost/latency или точную стадию отказа.
Artifacts run пусты (проверено GitHub API 2026-10-21). Локальный SSH не авторизован;
рабочий доступ есть у защищённого production workflow через существующий secret.
S1.9 diagnostic привязан к S1.8; его dispatch не читает C6. C6 inspector привязан
к paid runner. Повтор paid mode ради чтения запрещён.

Результат S1.12: отдельный read-only путь к фиксированному report, установление
canonicalization/merge stage или UNVERIFIED, точные recorded cost/latency и,
только при наличии canonical proposal, локальное воспроизведение merge.
Новая qualification, repair или production adoption не следуют из диагностики.

## Установленные ограничения сохранённых данных

На executed source `internal/statementqualification/comparison.go`:

1. Complete и wire decode должны успешно завершиться, иначе код ошибки был бы
   `comparison_provider_failed`. Наблюдаемый protocol FAIL находится позднее.
2. `CanonicalComparisonProposal` вызывается до присвоения `tuple.Proposal`.
   При ошибке canonicalization возвращает пустой Proposal, исходный alias proposal
   в report не попадает. Diagnostic также не присваивается для этого отказа.
3. После успешной canonicalization `tuple.Proposal` сохраняется; затем вызывается
   `MergeProposals`. Обе ошибки свёрнуты в один public code.
4. При verified report с `protocol_failed` отсутствие proposal означает ветку
   canonicalization, присутствие означает ветку merge. Это доказательство стадии
   по exact-source control flow, не знание конкретного guard.
5. Причины canonicalization: invalid plan, money selection вне numeric aliases,
   неизвестные aliases или дубли selection. Без исходного proposal выбор между
   ними остаётся UNVERIFIED. Нельзя реконструировать ответ по golden fixture.
6. Merge проверяет locators/chunk membership и header roles; при сохранённом
   proposal можно воспроизвести exact function на проверенном plan. Конкретную
   причину заявлять только при воспроизведённом совпадении.

## Ограниченный контракт

### R1. Единственный источник и безопасное чтение

Фиксированы report `/opt/grace/shared/statement-comparison-s111-comparison-20261021.json`
и marker `/opt/grace/shared/statement-comparison-s111-comparison-20261021.started`.
Никаких glob/latest/arbitrary path. Marker exact source012f604; report version,
authorization/profile/policies/proofs/digest/timestamps и все6tuples проверяются
по executed contract, с явным historical replay без требования current freshness.

Open O_RDONLY/O_NOFOLLOW, regular0600, ограниченные size/depth/arrays, strict JSON
без duplicate keys/trailing values. Лимиты: report<=524288 bytes, marker<=128 bytes, depth<=20, arrays<=5000;
при превышении stop/UNVERIFIED, без усечения.
Existing operator lock открывается read-only; никаких создания/удаления lock files.
Concurrent lock => stop. До/после hash/size/mode/mtime report и marker должны совпасть.
Несовпадение, malformed или missing data => closed unavailable/UNVERIFIED, без
partial output и raw exceptions. Авторизация inference не создаётся и не меняется.

### R2. Закрытая проекция

Вывод: fixed candidate/fixture/state/reason enums, proposalPresent bool,
stage enum canonicalization/merge/unverified, recordedCalls/completedTuples,
validated costMicros/actualMicros/latencyMs целыми числами из этого synthetic run,
known bool и recorded timestamps. Разрешено раскрыть только эти numeric accounting
поля; они не заменяют новой SQL ledger verification. Прежний результат inspector
known/settled/report=ledger остаётся датированным evidence предыдущего job.

Никаких generation/owner/reservation IDs, credentials, source/model/service/error
text или raw report. Proposal, если сохранён, экспортируется только как bounded
structural selections с существующими source locators, ordinal, closed directions
и header roles. Непустой service reason заменяется фиксированным synthetic значением
после проверки прежних bounds. Unknown scalar/role => UNVERIFIED, не догадка.
Projection должна сохранять порядок/дубли selections, чтобы не чинить ответ при чтении.
Privacy sentinel в любом forbidden text не должен попасть в stdout/stderr/artifacts.

### R3. Проверка и локальный replay

Только прежний repository columns.pdf с hash S1.11 и pinned Poppler25.04.0 helper
в network=none. Сверить document/plan/request hashes с report, сохраняя exact policy
и весь profile executed source. Локально проверить digest тем же Go hashing contract,
не подменять его Python serialization. Private Go verifier собирается из exact
diagnostic source, передаётся в bounded stdin envelope и исполняется через Linux
memfd с проверенным hash, без disk install. Он читает фиксированный report, проверяет
прежний Go digest и возвращает только закрытую проекцию; Python bootstrap закрывает
ошибки без raw output. Если memfd недоступен — stop/UNVERIFIED, без fallback upload.
Remote projection не может объявить digestVerified по одной проверке hex shape;
полный report в Actions не копируется.

Если proposal отсутствует: stage canonicalization, exact guard UNVERIFIED,
replay unavailable, без fabricated proposal. Если присутствует: через прежний
MergeProposals воспроизвести отказ, вывести closed code и bounded field/role/ordinal
локализацию при доказуемом соответствии. Ни canonicalization повторно на canonical
proposal, ни golden-created ответ не являются replay исходной модели.

### R4. Отдельный workflow и нулевой inference

Новый diagnostic mode default=false в существующем production workflow, preserve;
все остальные jobs исключены при каждом конфликте, включая C6 paid mode.
Существующие production concurrency и pinned SSH. Exact diagnostic source + CI
публикуются перед dispatch. Нет paid runner, provider/key/catalog requests, SQL,
reserve/settle/reconciliation, Docker/config/model/image/financial writes.
Remote действия ограничены чтением fixed artifacts и существующего lock;
private Go verifier передаётся через stdin/memfd без server install.
Sanitized artifacts сохраняются отдельно от private report.

### R5. Исполнение и evidence

После принятия R1–R4, интеграции спецификации и offline gates разрешается один
read-only diagnostic dispatch R5. Технический сбой не открывает paid retry;
повтор read-only возможен только с неизменными scope/source, без новых provider calls.
Отчёт сохраняет C6 FAIL/consumed, явно указывает verified stage/cost/latency и
неустановленные причины. Исправление extractor/prompt/schema/report writer/guards
и новый campaign — отдельный пакет после диагноза.

## Критерии приёмки

| ID | Исполняемая проверка |
| --- | --- |
| R1 | Reader negatives: source/path/mode/symlink/size/depth/duplicate keys/version/profile/digest/hash/lock; output closed, immutable artifacts |
| R2 | Privacy sentinel matrix во всех free-text полях; allowlist-only stdout/stderr/artifact, exact bounded numeric cost/latency, дубли не теряются |
| R3 | Real-helper cases обеих веток: missing proposal => canonicalization/guard UNVERIFIED; сохранённый proposal => exact merge error или mismatch/UNVERIFIED; digest/document/plan/request verification; никаких fabricated model outputs |
| R4 | Exhaustive actual workflow conditions, default false/exclusivity и provider/SQL/paid/deploy absence; make test, relevant production mocks, go vet, git diff --check PASS |
| R5 | Один read-only dispatch на reviewed diagnostic SHA: artifacts unchanged; стадия и numeric accounting проверены либо честно UNVERIFIED, inference=0 |

## Решение владельца

Владелец 2026-10-21 ответил «утверждаю» на S1.12 R1–R5. Приняты
**ограниченная реализация и один бесплатный read-only diagnostic dispatch**, с разрешённой закрытой публикацией numeric
cost/latency только этого synthetic C6. Production-код поведения продукта,
новый paid run и исправление причины этим решением не разрешаются.
Acceptance зафиксирован до кода; implementation worktree создан от main33cd15a.
[R1–R5 evidence](0055-s112-evidence.md): offline/exact-source CI PASS; R5 SUCCESS,
canonicalization / raw proposal missing / exact guard UNVERIFIED, providerCalls0.
