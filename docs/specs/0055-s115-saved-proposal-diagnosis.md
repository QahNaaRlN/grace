# SPEC-0055 / S1.15: read-only диагностика сохранённых proposals P6

- Статус: accepted (D1–D4); implementation_status: integrated_offline; execution_status: executed_read_only (one-dispatch permission consumed).
- Автор: Codex; дата: 2026-10-21, Asia/Yekaterinburg.
- Acceptance owner: владелец Grace; acceptance record: прямое «Приступай», 2026-10-21, после предъявления PR №93 и границы D1–D4/D5.
- Основание подготовки: «работаем!» после P6 FAIL и публикации evidence.
  D1–D4 приняты последующим «Приступай»; D5/SSH/read-only run не разрешены.
- База: main `868c0a4254bb32e08b07225dfb3672d9b7d5287b`, docs-only PR №92.
- Spec branch: `codex/statement-saved-proposal-diagnosis-spec`.
- Implementation: `codex/statement-saved-proposal-diagnosis`, code `de770793bba711dada87446ddcd3a442b74f5d5f`; [D1–D4 evidence](0055-s115-evidence.md).
- Final implementation head: `703b6c6ac960c21c0b7ab7bd84b3411459204c92`, integrated PR №94 / merge17e3493; [post-merge evidence](0055-pr94-postmerge.md).
- [D5 decision](0055-s115-d5-run-decision.md) отдельно принят прямым «разрешаю» на exact703b6c6/ref/private-read/CI scope; [workflow receipt](0055-s115-d5-execution-receipt.md): run39860669842 SUCCESS, permission CONSUMED; closed payload остаётся в CI.

## Проблема и пользовательский результат

[S1.14 P6 evidence](0055-s114-p6-evidence.md): один run39621334415 на source
`301afd6bedc4a19559e731471336bbc4e2f74075`,3POST/5320micros known/settled,
3not_sent, permission consumed. A/columns FAIL16/19 с description_evidence_gap
в row2; B/columns FAIL17/19 с source_coverage_incomplete; A/cards остановлен
unknown_alias в operationDate первой строки Proposal, selectionIndex1.
Все3captured snapshots matched replay, marker/report unchanged.

Точный первый canonicalization guard уже установлен — его поиск заново не цель.
Не известно, относится ли неизвестная ссылка к original source ID, отсутствующему
alias канонического формата или иному значению; какие конкретно позиции пропущены
между description evidence и какие spans действительно unmapped у B.
Ответы без private snapshot не угадываются. Новые provider calls для разбора не нужны.

Результат — проверяемые structural witnesses для этих3 наблюдений, полученные
на неизменённом saved artifact и pinned public layouts. Отчёт отличает VERIFIED
структурный факт, UNVERIFIED/mismatch и гипотезу будущего исправления. Он не
исправляет mapping, не объясняет внутреннее рассуждение модели и не даёт winner.

## Scope

**D1–D4 offline**, после принятия: pure diagnosis/replay на fixed S1.14 context,
closed projection, отдельный fixed-path verifier и memfd/SSH envelope по bounded
паттерну S1.12, default-off workflow isolation и causal/private/real-helper tests.

**D5 execution**, только после отдельного exact-source approval: один read-only
запуск verifier на VPS над сохранённым P6 report/marker. Provider calls/metadata
GET/SQL/metering=0. Принятие D1–D4 не открывает D5. «Работаем!» не принимается за
approval технического контракта, которого до этого не было на review.

Допустимые области будущего diff: pure diagnostics в statementqualification,
узкие shared predicate helpers statementextract/statementdata при доказанной
неизменности прежнего поведения, новый verifier/prepare/bootstrap/wrapper в
scripts/research, отдельный workflow job/input и tests/gates/docs. Общий refactor
financial/API/UI не входит; полный actual diff предъявляется в evidence.

## Non-goals

Inference/catalog/endpoint/ZDR/generation GET, paid retry/new campaign/authorization,
reconciliation/SQL, reserve/settle/budget/fees changes, личные PDF/owner drafts,
product endpoints/UI/OpenAPI, prompt/schema/planner/redaction/aliases/scorer или
admission repair. Не менять старые S1.11/S1.12/S1.14 readers, artifacts/markers,
consumed status, source301afd6/ref либо production model/flags/image.
D5 не использует Docker/API/provider на VPS и не устанавливает files/packages/
services. Public fixtures обрабатываются pinned helper **локально** в network=none.
Никакого исправленного proposal или изменения golden для настоящего P6.

## Текущее поведение и доказательства

- Source301afd6 `CanonicalComparisonProposalDiagnostic` даёт closed first guard;
  `ReplayComparisonObservation` проверяет canonicalization и hashes. Он не
  объясняет alias relation, gap witnesses или unclassified source coverage.
- `descriptionIssue` в statementdata/grounding_guards.go возвращает gap, если
  последовательные selected evidence spans на одной printed line имеют
  непоследовательные Word ordinals. Это фактический predicate, не семантическая
  догадка о недостающем слове/разделителе.
- `ValidateProposal` вычисляет used/unmapped и source_coverage_incomplete;
  `NewMappingSnapshot` сохраняет реальные unmappedSpanIDs. Это источник membership,
  а не самостоятельный повторный подсчёт по произвольному обходу selections.
- `ScoreScreening` — текущий scorer19/20/23, A/cards до него не дошёл.
- Existing Python P6 inspector сверяет billing/replay, но его numeric/enum summary
  не содержит требуемые structural witnesses. Старые cost/latency факты не меняются.
- PR №92 синхронизирован наc96cc83, exact-head CI39622633167 SUCCESS5/5,
  integrated868c0a4. P6 run/job FAIL остаются отдельными от documentation CI PASS.

## Предметная модель и инварианты

### D1. Fixed artifact и unknown-alias witness

Вход только envelope `statement.comparison.observed.s114.v1`, authorization
`s114-comparison-20261021`, marker source301afd6, прежние exact candidates/profile/
3fixture hashes. До диагностики проверить strict JSON/оба digests, bounds S1.13,
восстановленные document/plan/request hashes и matched replay для3 captured tuples.
Source report не переписывается; bare v1, S1.11 и другие кампании не подходят.

Для tuple3 A/cards воспроизвести прежний unknown_alias, scope=row,
field=operationDate, ordinal1, selectionIndex1. Из исходного snapshot прочитать
ровно этот selection item в памяти, проверить отсутствие в plan.Aliases и выдать
один closed relation по фиксированному приоритету:

1. `original_span_id`: exact match одного original Document span ID;
2. `canonical_alias_syntax_absent`: каноническая syntax s+base36 без ведущих0
   (кроме s0), но exact alias отсутствует в plan;
3. `outside_alias_syntax`: не соответствует указанной grammar;
4. `UNVERIFIED`: несовпавший guard/position, недоступный snapshot или иной mismatch.

Grammar/relation — объяснение, не новые admission predicates. Не возвращать raw
alias, его hash/prefix/length, похожий alias или предложенную replacement selection.
Не делать fuzzy lookup/repair. Первый guard не объявляется единственным дефектом.

### D2. A/columns — causal description-gap witness

Для tuple1 пройти фактические canonicalizer→MergeProposals→ValidateProposal→
ScoreScreening с теми же reviewedPages/rolesConfirmed/zone, что source301afd6.
Сверить canonical proposal, saved review-derived codes/checks/correct/pass и
falseAcceptance с report; latency переносится только из recorded Call и не
пересчитывается как качество. Любое расхождение означает UNVERIFIED для diagnosis.

Получить первый witness именно production description predicate для row2:
selected left/right source ordinals, Page/Block/Line, Word ordinals и количество
промежуточных source positions, не выбранных в description. SourceOrdinal —
индекс с1 в verified Document.Spans, не raw span ID. Поле witness содержит только
эти целые числа, row ordinal и closed status. При наличии других selections на
этих positions допустим closed membership `service`, `other_field`, `unselected`.
Это не утверждение, куда span должен быть назначен финансово.

Проверка и её explanation должны использовать один shared predicate либо его
прямой проверяемый результат; расходящуюся копию description guard не вводить.
При extracted witness сначала подтвердить прежний description_evidence_gap,
валидность evidence и same printed line. Cross-line/consecutive-word near misses
не становятся новыми guards. Не выводить source/model text или literal amounts.

### D3. B/columns — verified unmapped witnesses

Для tuple2 так же проверить canonical/merge/validation/scorer equivalence17/19.
Unmapped membership взять из actual ValidateProposal result через existing
NewMappingSnapshot, а не из guessed service classification или golden selectors.
Выдать totalUnmapped и source-order witnesses: SourceOrdinal, Page/Block/Line/Word,
closed lexical class `numeric`/`other` из текущего verified redacted plan.
Не публиковать raw source IDs, texts, aliases, values или model service reasons.

Не классифицировать unmapped token как «служебный», «финансовый» или «безопасно
игнорируемый» по эвристике. Membership — факт; требуемая business роль остаётся
UNVERIFIED до отдельного решения. Любой ID, отсутствующий в verified layout/plan,
даёт mismatch. At most128 witnesses; превышение означает diagnosis_limit_exceeded/
UNVERIFIED, без частичного списка с заявленным VERIFIED. Exact total count может
быть отдельно verified; оригинальный artifact не усекать/не менять.

### D4. Closed verifier, privacy и offline gates

Новый closed output `statement.saved-proposal-diagnosis.v1`: fixed context,
contextVerified/artifactsUnchanged, providerCalls=0 и6tuple rows в исходном
порядке:1description_gap,2unmapped,3alias_relation,4–6not_applicable. Для первых3
status=VERIFIED/UNVERIFIED, только D1/D2/D3 enums/ordinals/counts. Для tuples4–6
только not_applicable/not_sent, никакой вымышленной diagnosis. Output<=65536bytes;
unknown field/code, ambiguous/duplicate keys, overflow и private sentinel дают
closed failure/UNVERIFIED без raw exception. Per-issue reason только
snapshot_unavailable/context_mismatch/guard_mismatch/replay_mismatch/
diagnosis_limit_exceeded/unavailable_runtime; при VERIFIED reason отсутствует.
Один issue UNVERIFIED не скрывает остальные verified facts: wrapper передаёт
closed projection только при contextVerified=true и artifactsUnchanged=true.
Context/file integrity failure возвращает отдельный closed diagnostic_unavailable
без witnesses; не выдавать частичный объект за проверенный context.
Строгий schema/allowlist на Go boundary
и wrapper; failure process stdout/stderr не переадресуется в public logs.

Verifier читает только fixed `/opt/grace/shared/statement-comparison-s114-comparison-20261021.json`
и соответствующий `.started`: O_NOFOLLOW, regular file0600, report<=524288bytes,
marker<=128bytes; before/after content hash/size/mode/mtime совпадают. Не проверять
неизменность atime как условие read-only. persistentWritesRequested0 относится
к verifier/bootstrap, не обещает отсутствие физических OS/SSH log writes. Reject symlink/orphan/source mismatch,
mutation/partial file/legacy version. Нет arbitrary artifact path/campaign flags.

Public contexts готовятся локально pinned Poppler25.04.0 в network=none из exact
public fixtures. На VPS только bounded stdin envelope, sealed memfd verifier,
env={}, fixed file reads и closed stdout, по отдельным S1.15 wrapper/bootstrap;
существующие S1.12 scripts не расширяются и не меняют свой fixed source.
Binary<=16MiB, public context<=524288bytes, envelope<=32MiB, remote timeout45s,
SSH timeout75s, stderr discarded. Никаких инициированных verifier/bootstrap
persistent file writes, Docker/API/SQL/key/
config/provider constructors. Частично недоступная метрика остаётся UNVERIFIED.

Новый default-false `diagnose_statement_s115_only`/exact verifier SHA input и
отдельный job. Все modes взаимоисключающие; mixed inputs выбирают0 execution jobs.
До SSH обязательны exact verifier source/ref, CI всех5jobs и полный binary/context/
bootstrap/wrapper hash manifest. Old paid/new comparison/deploy/reconcile jobs
не выбираются. Это инфраструктура D5, не разрешение запуска.

Tests: causal delivered-alias fakes для3alias relations, настоящая gap/unmapped
validation, first-guard ordering, consecutive/cross-line/service-membership near
misses,128/129 witness boundary, corrupted hashes/context/version, public sentinel
на success/error/timeout/replay paths, file mutation/mode/symlink и env poison.
Offline synthetic control может восстановить **тестовый** omitted span для
локализации predicate; настоящий saved P6 snapshot никогда не repair/normalize.
Real helper подтверждает hashes и causal results; fake outcomes не live quality.
Прежние decisions/output/error semantics, arithmetic и scorer versions сохраняются.

### D5. Отдельно разрешаемое read-only исполнение

После acceptance D1–D4, реализации, review/integration и exact-head CI5/5 предъявить
owner конкретный verifier SHA/ref и один fixed-host read-only run над **тем же P6**.
До отдельного exact-source approval D5=NOT_AUTHORIZED, SSH/VPS=0. Даже бесплатный
read не открывается старым P6 permission, оно consumed.

После approval: один dispatch, только fixed source/read/artifact signature/replay,
providerCalls0, SQL0, persistent artifact/environment writes0, no image/flags/financial changes. Не проводить
новый health/metadata probe под видом диагностики. После отказа не менять permissions,
пути, snapshot, environment или source; read-only failure остаётся UNVERIFIED.
Результат: VERIFIED structural facts либо честные limitations по каждой из3issues,
source/run/job/CI references, artifactUnchanged и отдельная human interpretation.

## API и OpenAPI

Product API/DTO/OpenAPI/Web не меняются. Новая версия только operator projection.
TypeScript regeneration не требуется. Private read не является import consent.

## Данные, миграция и rollback

PostgreSQL/финансовых миграций, rows или ledger writes нет. Saved P6 report/marker
и consumed permission неизменны, private snapshot не переносится в Git/Actions.
Ephemeral local public contexts и remote memfd исчезают после run; persistent
new private report/storage backend не вводится. Rollback отключает новый diagnostic
mode; исходный P6 result/guard/cost и existing readers не удаляются.

## UI

Пользовательский UI и owner draft не меняются.

## Ошибки и негативные сценарии

Missing/legacy/oversized snapshot, wrong campaign/marker SHA, digest/hash/scorer
mismatch, invalid indices/layout, overflow witnesses, ambiguous alias relation,
symlink/mutation, missing memfd/read permission, timeout и malicious text дают
closed UNVERIFIED. Не фабриковать replacement IDs, missing words или service roles.
Каждый verification level показывается отдельно от quality verdict и hypotheses.

## Критерии приёмки

| ID | Evidence будущей реализации |
| --- | --- |
| D1 | Реальные canonicalizer/observer fakes и pinned helper воспроизводят3alias relations; exact saved guard/position, context/digests/hashes checked, no raw alias output |
| D2 | Actual validation/scorer equivalence; causal row2 gap witness и near misses; неизменные legacy guard decisions/output, без raw literal/repair |
| D3 | Actual unmapped membership, deterministic positions/counts/lexical class; overlap/service/golden independence и128/129 limit, без guessed business classification |
| D4 | Full writer→reader→closed projection→wrapper tests, privacy/error/mutation/env negatives, workflow matrix/exact source/manifest и real-helper PASS; local gates/CI5/5 до D5 review |
| D5 | Только после отдельного approval: один fixed-host read-only run на exact verifier SHA; providerCalls0/SQL0/persistentWritesRequested0, artifact/marker unchanged,3diagnoses VERIFIED либо honest UNVERIFIED; P6 FAIL/consumed сохранён |

## Исполняемые проверки

```sh
TMPDIR=/tmp VITEST_MAX_WORKERS=1 make test
TMPDIR=/tmp make test-production
TMPDIR=/tmp scripts/test-bank-statements.sh
go test -race ./internal/statementdata ./internal/statementextract ./internal/statementqualification
go vet ./cmd/... ./internal/... ./scripts/research/...
git diff --check
```

Новые verifier/real-helper/privacy/workflow tests подключаются к обязательным
production/integration gates; точные команды/результаты и actual diff записываются
в implementation evidence. Если future diff затронет SQL path вопреки non-goals —
вернуть scope на review, не запускать live SQL или молча расширять suite/пакет.

## Открытые решения

1. D1–D4 приняты владельцем; разрешена только offline-реализация и её проверки.
2. D5 отдельно разрешён и выполнен один раз на703b6c6, workflow SUCCESS; [receipt](0055-s115-d5-execution-receipt.md). Permission consumed.
3. Payload результата доступен только в закрытом CI artifact; Git publication не разрешена. Следующий fix/paid run требует отдельного решения, ничего не repair автоматически.
