# SPEC-0055 / S1.17: контролируемое сравнение после ограничения alias

- Статус: accepted; accepted_scope: E1–E5 offline; implementation_status: integrated_offline (PR №100); E6 readiness blocked by workflow YAML.
- Автор: Codex; дата: 2026-10-23, Asia/Yekaterinburg.
- Владелец приёмки: владелец Grace; E1–E5 приняты 2026-10-23 прямым «да»
  на уточнение о принятии offline-контракта PR №99 (review head `cc90361`).
- Отдельное разрешение E6 получено 2026-10-23: «Даю отдельное разрешение».
  Оно не использовано: execution NOT_STARTED, source SHA ещё не выбран;
  предварительные условия E1–E5/integration/exact-source CI/readiness обязательны.
- Основание подготовки: продолжение после слияния PR №98; следующий контракт
  был предложен на review. Последующее принятие и отдельное разрешение E6
  зафиксированы выше; слияние S1.16 само по себе их не заменяло.
- База: main `81ef7e543824b327d2a43ab88838a04bb58f1ed8`.
- Ветка спецификации: `codex/statement-s116-postmerge-next`.
- Реализация: `codex/statement-s117-comparison`, commit `220284f131029812f3ab918625606aa6fc7b9e0b`; [E1–E5 evidence](0055-s117-evidence.md).
- Разрешённый live source/ref: не выбран; E6 permission unused, readiness NOT_READY.
- Владелец подтвердил слияние №100 и переход к E6; [readiness blocker](0055-s117-e6-readiness.md) выявлен до dispatch.

## Проблема и пользовательский результат

[S1.16](0055-s116-alias-schema.md) ограничила все22 selection paths точным
множеством переданных alias и сохранила отдельное числовое множество для денег.
[Offline evidence](0055-s116-evidence.md) подтверждает schema/HTTP/consent/bytes
и неизменность Go-проверок, но не соблюдение схемы реальной моделью.
Исторический [P6](0055-s114-p6-evidence.md) остаётся FAIL, разрешение consumed.

Предлагаемый результат — подготовленный offline механизм одной новой,
отдельно разрешаемой synthetic campaign на контракте v3. После отдельного
решения об исполнении он даст наблюдаемые результаты protocol/quality/cost/
latency с сохранёнными proposals и честным replay либо UNVERIFIED.
До измерения не утверждается, что enum устраняет failures, закрывает пропуски
описаний или определяет финансовую роль неразмеченных spans.

## Scope

Приняты **E1–E5 offline**: новый фиксированный context,
отдельные operator runner/summary/inspector/replay/workflow mode, проверки
совместимости и причинные offline-тесты. Использовать существующие comparison
engine, private capture, canonicalizer, scorer, бюджет и transport.
Допустимые области реализации: `internal/statementqualification`, operator CLI,
тесты, отдельные research runner/inspector/bundle scripts и workflow isolation.
Общие helpers можно переиспользовать без изменения поведения legacy entry points.

**E6 live** получил отдельное разрешение владельца. Исполнение возможно только
после приёмки и интеграции E1–E5, фиксации immutable SHA/ref, полного CI5/5
на том же source и успешного preflight. Разрешение не является готовностью
и не переносится на старые modes/markers или более широкий scope.

Принятая спецификация интегрирована [PR №99](0055-pr99-postmerge.md) в main94fd4ab
до создания нового implementation worktree. Offline реализация интегрирована №100;
до E6 выявлен YAML blocker. Исправление должно быть интегрировано до выбора
immutable source/ref и exact-source CI5/5.

## Non-goals

Prompt, schema, planner, redaction, aliases allocation, lexical classification,
scorer, финансовые guards и product API/Web не меняются. Не исправлять descriptions
или неразмеченные spans и не подменять их service автоматически. Не добавлять
repair/retry/fallback, новые модели/endpoint/fixtures, произвольные report paths,
смену production policy/flags, deployment, личные PDF или финансовые writes.
Бюджет/fees/allowance не увеличиваются и не сбрасываются. SPEC-0056 отдельно.
До готовности E1–E5 и фиксации exact source нет SSH/VPS, metadata GET, dispatch или inference.
Старые P6/D5 artifacts и consumed permissions не переиспользуются; закрытые
private-derived данные не переносятся в Git/PR/Actions artifacts.

## Текущее поведение и доказательства

- PR №98 интегрировал planner `statement-document-atom.v3.alias-enum`, schema
  hash `1a6e755395da69883666f3eb61471ad8c4257f7d465451a4ad0050fbd253c3c7`.
  [Post-merge №98](0055-pr98-postmerge.md) отделяет source/head/merge CI.
- Current Go profile зависит от этого planner/schema; old v2 consent и
  qualification profile несовместимы с новым запросом. Старые inspectors
  закреплены за v2 и намеренно отвергают v3. Нельзя просто запустить прежний P6
  на новом binary или переписать PROFILE старого inspector.
- S1.14 и S1.15 реализуют private snapshot, закрытый diagnostic и replay;
  historical readers/source сохраняются. До этой реализации нового campaign identity не было.
- Доступность, actual identity, цены, runtime readiness, remaining/uncertain
  и actual image/health сейчас UNVERIFIED; в offline реализации не читаются.

## Предметная модель и инварианты

### E1. Новый закрытый context и прежний контракт извлечения

Предлагаемые идентификаторы, сами по себе не являющиеся разрешением:

- authorization: `s117-comparison-20261023`;
- private outer envelope: `statement.comparison.observed.s117.v1`;
- ledger model: `statement-comparison.s117`, provider: `fixed-A-B`, purpose: `statement`;
- CLI summary mode: `comparison-s117-summary`;
- bundle prefix: `/tmp/grace-statement-comparison-s117-<exact40hexSHA>/`.

Inner report сохраняет прежнюю структуру `statement.comparison.v1.grounding`,
но новую authorization и фактический profile S1.16. Outer/inner digests вычисляются
по исходным значениям. Новый fixed decoder выбирается по новой outer version.
Несовместимые auth/version/profile/digest combinations отвергаются без JSON repair,
переписывания hashes или нормализации в старый context.

Legacy entry points/PROFILE сохраняют своё прежнее поведение. Новый inspector
должен проверять настоящий v3 contract; нельзя временно monkeypatch старый PROFILE
или заменять поля отчёта перед validation. Повторно используются общие функции
только с закрытым ожидаемым context; пользователь не выбирает profile/authorization.
Private capture до canonicalization, closed diagnostic, snapshot bounds и
приоритет причин ошибки остаются как в accepted S1.13/S1.14.

### E2. Однократность и изоляция operator mode

Новые фиксированные файлы в `/opt/grace/shared`:
`statement-comparison-s117-comparison-20261023.started` и
`statement-comparison-s117-comparison-20261023.json`; режим0600.
Preflight/exit files также принадлежат только новому identity. Общий exclusive
lock между всеми modes сохраняется. No-clobber marker создаётся до первой
возможной отправки и остаётся после любого исхода. Existing/orphan/symlink,
concurrent/mixed mode или старый marker не разрешают продолжение.

Предлагаются default-false `statement_comparison_s117_only` и отдельный
`comparison_s117_authorized_sha`; `statement_action=preserve`.
Ровно один mode допустим; matrix tests должны исключать deploy, все paid/
reconciliation/diagnostic соседние jobs и SSH при ambiguous inputs.
Новый режим не сбрасывает старые markers и не использует их имена.
Ref, exact40hex SHA, SOURCE_SHA, весь SHA256SUMS bundle и все5 CI jobs
проверяются до исполнения. Pending/skipped/failed CI не означает готовность.

Один будущий approval покрывает один dispatch, включая возможный preflight
refusal. Отказ не даёт права повторить dispatch, переставить tuple, выбрать
другой source/model или ремонтировать окружение в пределах этого разрешения.

### E3. Пары, порядок, лимиты и остановки

Предложено повторно использовать только прежние source pins; это не утверждение
о текущем каталоге или ценах:

| ID | Request model → exact canonical | Endpoint / display provider | Максимум USD micros / документ |
| --- | --- | --- | --- |
| A | google/gemini-3.1-flash-lite → google/gemini-3.1-flash-lite-20260507 | google-vertex/eu / Google | 53900 |
| B | inception/mercury-3 → inception/mercury-3-20261019 | inception / Inception | 15600 |

Только columns/cards/turnover-roles-ru с прежними exact fixture hashes.
Порядок A-columns → B-columns → A-cards → B-cards → A-RU → B-RU.
На tuple максимум1POST, всего до6POST / **208500 USD micros ($0.2085 provider)**.
A максимум161700, B46800; прежние input100000/output16000,120s/call,
900s campaign inference context и отдельные5s settlement. Размеры/span bounds
S1.16 не меняются. Все6plans и total reserve проверяются до первого POST.
Прежний shared $10 ledger, реальные remaining, uncertain, fees и costs сохраняются;
provider ceiling не объявляется fee-inclusive итогом. Inception только synthetic;
EU residency всей цепочки не заявляется.

Перед каждым Complete проверяется прежний call_allowed. Known deterministic
quality/grounding FAIL позволяет пройти оставшиеся заранее разрешённые tuples.
Protocol/transport/identity/privacy/cost/timeout, unknown billing, settlement
failure или shared block останавливают campaign. Нельзя пропускать A ради B
после protocol failure, делать дополнительные пробные POST или переносить
неиспользованный лимит в новый run. Fresh catalog/endpoint/ZDR proof обеих пар
и цены в пределах прежних caps — обязательный preflight будущего E6.
Любое несовпадение требует возврата владельцу, без автоматической замены pins.

### E4. Артефакт, accounting и read-only объяснение

Прежний private writer сохраняет новый outer envelope0600. В stdout/stderr,
CI logs/artifacts и evidence попадают только принятые closed enums/bools/counts
и числовая cost/latency сводка. Ни raw alias-proposal, ни free text, UUID,
source IDs, provider/generation IDs или ключи публично не передаются.
Snapshot<=32768 bytes/tuple, artifact<=524288 bytes; overflow отбрасывает snapshot
целиком и не меняет billing. Capture не объявляется crash-durable.

Одна новая reservation с точным новым ledger model. Inspector сопоставляет
единственный reservation UUID из verified report с amount/status/time; не ищет
«последнюю подходящую» строку. Accounting и artifact integrity имеют отдельные
результаты; известный расход не становится0 из-за capture/storage failure.
Historical rows/reports не переписываются, unknown bill блокирует новые calls.

Будущее E6 включает только read-only inspection/replay своего нового fixed
artifact, bundled public PDFs и helper. Replay не создаёт model/key/SQL/config
constructors; unavailable/missing/partial capture честно UNVERIFIED.
Bytes/mode/mtime marker и report должны совпасть до/после inspection.
Никакого отдельного recovery или повторного чтения старого D5 по этому контракту.

### E5. Offline evidence до решения об исполнении

Реальный путь fake Complete → capture → canonical guard → private writer →
строгий Go summary → новый inspector → replay. Fakes получают только actual
wire tuples; oracle публичных fixtures независим от ответов модели. Проверить
все6 profile/plan/request hashes S1.16, scorer19/20/23 и causal guard failures.
Current v3 и frozen v2 fixtures проверяются раздельно; старые validators не ослабляются.

Проверки: no-clobber при том же/другом SHA, orphan/symlink, concurrent modes,
preflight refusal, failure after marker, known/unknown/settlement, overflow,
corrupt digest, exact SQL correlation, private sentinels на каждом публичном
выходе и отсутствие соседних workflow jobs. SQL — только disposable DB;
реальный pinned helper — network-none. Fake PASS не является live evidence.

### E6. Отдельный execution gate

Владелец дал отдельное разрешение на один E6 заранее; оно ещё не использовано.
Готовность остаётся обязательной: приёмка offline E1–E5, интеграция, фиксация
immutable source SHA/ref и exact-source CI5/5. Перед dispatch эти evidence
и конкретный SHA/ref должны быть записаны в execution receipt. Scope и caps
остаются прежними; изменение scope или подмена уже закреплённого source
возвращаются на review владельцу.

Сейчас live SHA отсутствует; merge81ef7e5 — база, не executable S1.17.
Dispatch-команда не подготовлена, execution NOT_STARTED. Повторное общее
разрешение E6 не запрашивается; предварительные условия не обходятся.

Будущий preflight включает SSH/VPS, fresh proofs, global AI/key/helper,
единственного active admin, shared budget/uncertain, actual image/health и
проверку manifest/source. Такой доступ должен явно входить в отдельное решение
об исполнении, как и public synthetic передачи провайдерам.
После run фиксируются run/job/source IDs, tuple states, guard/capture/replay,
known cost и latency Complete (с proof/network/decode), billing и image/health.
При incomplete/малой выборке не объявлять победителя, p95 или qualification
всех банков. Production adoption — отдельный пакет.

## API, UI и OpenAPI

Продуктовые endpoints/DTO/UI не меняются; TS regeneration не требуется.
Новые интерфейсы только private operator context/summary и workflow inputs.
При необходимости product API изменения вернуть scope на review.

## Данные, миграция и rollback

Новых SQL миграций/financial writes нет. Offline — disposable fixtures.
После отдельного E6 — новая reservation и новые private artifacts в прежней
storage/retention boundary. Rollback запрещает новый mode, но не удаляет
markers, reports или сведения о расходе. Deployment rollback не разрешён.

## Ошибки и негативные сценарии

Stale source/profile, wrong auth/version, partial/pending CI, mixed modes,
marker/orphan/symlink, отсутствующие admin/key/helper/budget/proofs, цена выше cap,
unknown billing, settlement/storage failure и unavailable replay дают закрытый
отказ либо UNVERIFIED. Отказ не преобразуется в PASS или новое разрешение.

## Критерии приёмки и исполняемые проверки

E1–E5 приняты; текущее выполнение отражено в [implementation evidence](0055-s117-evidence.md). E6 отдельно разрешён,
но prerequisites не выполнены и запуск не выполнялся.

| ID | Требование | Исполняемое evidence |
| --- | --- | --- |
| E1 | Новый fixed context и v3 profile; legacy не изменён | Go decoder/digest/profile cross-product negatives, текущие и frozen fixtures раздельно; report нельзя relabel/rehash |
| E2 | Однократность и isolation | Shell/workflow exhaustive matrix, same/different SHA marker, symlink/orphan/lock/concurrent tests; ambiguous inputs→0SSH/POST/deploy |
| E3 | Exact policies/schema/order/caps и accounting | Fake HTTP + actual tuples/schema,6plans before reserve, one reserve/settle, continuation/stop, known/unknown и bounds |
| E4 | Capture→writer→Go summary→inspector→replay с privacy | Real helper + disposable SQL, digest/overflow/error/immutability/sentinel tests, scorer19/20/23 и replay6/6 на fake cases |
| E5 | Полная offline регрессия | `make test`, `make test-production`, `make test-integration`, affected race, `go vet ./cmd/... ./internal/...`, `git diff --check`; implementation report по каждому критерию и exact-head CI5/5 |
| E6 | Один отдельно разрешённый execution либо preflight refusal | После выполнения gate: exact source/run/job IDs, до6POST/$0.2085, states/quality/diagnostics/replay/billing/image evidence; без retry |

## Открытые решения

1. Offline E1–E5 приняты. После интеграции этой принятой спеки в main —
   реализация интегрирована №100; YAML blocker исправляется до E6.
2. Отдельное разрешение E6 получено и не использовано. Exact source/ref,
   integrated offline evidence, CI5/5 и preflight ещё отсутствуют; запуск не готов.
3. Полнота description и семантика неразмеченных spans остаются отдельным
   кандидатом. Этот контракт не разрешает их prompt/guard repair.

## Документационная проверка принятия

`TMPDIR=/tmp VITEST_MAX_WORKERS=1 make test` — PASS в spec worktree:
Go,88 Web files/334 tests, ESLint/FSD/Prettier/build и offline Jev;
log `/tmp/grace-s117-acceptance-make.log`. `git diff --check`, ссылки
документов и отсутствие runtime diff относительно main81ef7e5 — PASS.
Это проверка docs-only acceptance, не выполнение новых E1–E5.
Исторический docs-only result не является implementation evidence. Текущая реализация
описана [отдельно](0055-s117-evidence.md); E6 не исполнялся, разрешение не использовано.
