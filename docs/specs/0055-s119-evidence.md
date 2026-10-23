# S1.19 — offline C1–C4 integrated, live NOT_AUTHORIZED

- Контракт: [accepted C1–C4](0055-s119-evidence-selection-candidate.md).
- Owner acceptance: прямое «Принять, интегрировать и продолжить offline» на
  PR106/c9b621d; разрешены фиксация принятия, merge после CI5/5 и offline работа.
- Implementation code: `4668d1309aea156d96b03938b56fd43a30a4470f`; branch `codex/statement-s119-clock-pilot`.
- База: main `db65225c6d2cfc9052ccf4f7310ced61c5ba79d8`.
  Принятый контракт интегрирован **до** implementation worktree.
- Статус: **integrated offline; post-merge gates PASS**.
  Live pilot **NOT_AUTHORIZED / NOT_RUN**, эффект H1 **UNVERIFIED**.

## Интеграция принятого контракта

[PR106](https://github.com/QahNaaRlN/grace/pull/106) слит агентом по прямому
разрешению 2026-10-23 13:16:35 UTC / 18:16:35 +05:00.
Final acceptance head4a0ecaaee81fc760738493c54d75c2fc49b86c49:
[CI39958410490](https://github.com/QahNaaRlN/grace/actions/runs/39958410490)
SUCCESS5/5, exact SHA/full jobs проверены canonical validator до merge.
Merge db65225:
[CI39959863612](https://github.com/QahNaaRlN/grace/actions/runs/39959863612)
SUCCESS5/5; [Delivery39961227991](https://github.com/QahNaaRlN/grace/actions/runs/39961227991)
SUCCESS4/4, exact SHA/full jobs проверены отдельно. Delivery не является deploy.
Primary clean/mode0700, новая implementation worktree от exact merge.

## AC и исполняемое evidence

| AC | Реализация и доказательство | Итог |
| --- | --- | --- |
| C1 | `BuildClockRequest` сам строит A plan; два exact prompt SHA; actual HTTP body через прежний provider; одинаковые policy/schema/user;90000/90001 и2000/2001 boundaries | PASS offline |
| C2 | Actual pinned Poppler → delivered alias proposal → canonicalization → merge → validation → scorer; полный4/4 vs удалённые часы0/4; чужие часы/date-only/invented alias negatives; precision отдельно | PASS offline |
| C3 offline | Exact A proof/price и reserve до POST; base/candidate/candidate/base/base/candidate;6 calls/cap323400; early-stop; known/unknown settlement; fixed one-shot включая другой SHA, fail и dangling symlink | PASS offline |
| C3 storage |32768-byte typed snapshots без усечения; <=1MiB report; digest + replay completed snapshots; duplicate/case/unknown/trailing/oversize rejection; только closed summary | PASS offline |
| C3 live | Новый research workflow default false; точный Source/CI5/5/main ancestry, manifest и отдельное разрешение до SSH/inference | NOT_AUTHORIZED / NOT_RUN |
| C4 | KEEP / REJECT / ceiling tie / incomplete / N/A / non-clock regression matrices; report cannot substitute a forged score; прекращение после одного pilot | PASS judge, live decision NOT_RUN |

Ключевые команды:

- `go test ./internal/statementextract -run '^TestClockRequestSingleFactorAndIdentity$' -count=1 -v`
  — PASS; log `/tmp/grace-s119-public-fingerprints.log`.
- `go test ./internal/statementqualification -run '^TestClock' -count=1 -v`
  — PASS; log `/tmp/grace-s119-clock-matrix.log`.
- `TestClockActualHTTPAndExactPrice` в `internal/assistant`: actual localhost
  serializer; новая/lower price не достигает POST; protocol/unknown останавливают
  sequence; никакой fake не объявляется подтверждением H1.
- `./scripts/test-statement-clock-s119.sh` — PASS,
  `/tmp/grace-s119-causal.log`; этот gate включён в `make test-production`.
- `TestClockMeterSQL` в `make test-integration` — PASS на disposable PostgreSQL:
  purpose/model/provider, cap323400, known settlement, uncertainty block и
  неизменность ранее settled row; production SQL отсутствует.
- `TestClockCLIClosedRefusalAndDefault` — PASS: default check не требует keys,
  wrong source не запускает pilot, preflight refusal сохраняет только closed
  cause; произвольные provider strings в stdout не попадают.

## Наблюдаемая offline причинность

Положительный public cards fake: **20/20**, clocks **4/4**.
Его копия, из которой удалены только четыре clock selections: **14/20**,
clocks **0/4**, coverage failure сохранён. Original proposal не мутирует.
Оба проходят реальный pipeline; готовые Review/issues не фабрикуются.
Старый activation fake переиспользован переносом без изменения алгоритма в
общий `_test.go`, прежние integration tests остаются PASS.

Date-only columns сохраняет precision=date; invented alias отвергается старым
canonicalizer. Это проверяет grounding и измеритель, а не улучшение модели.
Wrong original currency при4/4 clocks даёт REJECT по non-clock regression.
При несопоставимом числе строк target=N/A, решение INCONCLUSIVE; нулевой score
для недоступного результата не изобретается.

## Публичные fingerprints запроса

- Base input **9730 bytes**, candidate **10411 bytes** (messages+schema).
- Document: `ff6f0157ec84d773ed788efd67f4635e111425b5146ee2624e6e45aab538a4e9`.
- Plan: `f0f25fe36340438e13ec088a90230a6499041cb26d82424b800c5b45bd86a4db`.
- Base request: `c77a84f39a9b2acd57af1b27386b6e265f4742825c10d85babe975507708a6f9`.
- Candidate request: `ca1cb2bc4385214566d62d917d13dd36220b2d464a98f74aebe7558ac13b8ecd`.
- Prompt SHA256 — exact accepted значения из C1.

Файл s119-cards-document.json получен только из shipped public PDF существующим
pinned helper; новый integration test проверяет полное равенство реальной
экстракции этому fixture и его hash. Корпус/исходный PDF/golden не менялись.

## Общие gates

- `TMPDIR=/tmp VITEST_MAX_WORKERS=1 make test` — PASS, exit0:
  Go;88 Web файлов/334 tests; lint/FSD/Prettier/build/offline Jev.
  Log `/tmp/grace-s119-make-test.log`.
- `TMPDIR=/tmp make test-production` — PASS, exit0;
  `/tmp/grace-s119-production.log`, включая все прежние diagnostic/one-shot gates.
- `TMPDIR=/tmp make test-integration` — PASS, exit0;
  `/tmp/grace-s119-integration.log`, все disposable SQL/helper/API suites.
  Сообщения о запрещённых downgrade — ожидаемые отрицательные проверки.
- После завершения CLI refusal/summary: `go test ./cmd/... ./internal/...`,
  `go vet ./cmd/... ./internal/...` — PASS;
  `/tmp/grace-s119-final-go.log`, `/tmp/grace-s119-final-vet.log`.
- `go test -race ./cmd/statement-clock-pilot ./internal/statementqualification ./internal/statementextract ./internal/assistant`
  — PASS; `/tmp/grace-s119-final-race.log`.
- Точная Go1.24.0 из CI: `GOTOOLCHAIN=go1.24.0 go test ./cmd/statement-clock-pilot ./internal/statementextract ./internal/statementqualification ./internal/assistant`
  — PASS; `/tmp/grace-s119-go124.log`.
- `git diff --check` и actual workflow YAML parser (4 files) — PASS.

## Точность статусов учёта

Успех `Settle` с неизвестной суммой означает запись uncertainty, а не known
settlement. Private/public fields разделены: `settlementRecorded` фиксирует
успешную запись, `reservationSettled` true только при известной actual сумме.
В unknown сценарии: settlementRecorded=true, reservationSettled=false,
billingKnown=false; SQL независимо подтверждает status=uncertain и блокировку.
Отдельный regression test не позволяет назвать такую резервацию settled.

После этой правки: focused Go tests и точная Go1.24.0 — PASS
(`/tmp/grace-s119-accounting-status-tests.log`, `/tmp/grace-s119-accounting-status-go124.log`),
race qualification/CLI — PASS (`/tmp/grace-s119-accounting-status-race.log`),
полный vet — PASS. Request fingerprints/guards/SQL adapter не изменены.

## Границы и следующее решение

Default product prompt/profile/consent, schema/planner/aliases/redaction,
financial validators/scorer, API/OpenAPI/Web/migrations и старый deploy workflow
не изменены. Exact-price restriction включается только новым research adapter;
legacy comparison default сохраняется. Production constructors не выбирают его.

External inference/metadata, SSH/VPS, production SQL, deploy, owner financial
writes0. Private E6/P6/D5/closed stdout не читались. Ни одного нового workflow
запуска не было; old E6 FAIL/Q6 SUCCESS и consumed permissions сохранены.

Offline реализация интегрирована PR107. Ниже зафиксированы source/ref/manifest/inputs
того же исследовательского пакета; отдельная спецификация или readiness PR
для этой записи не создаются.
Ни live quality PASS, ни winner, ни production activation здесь не заявляются.


## PR107: интеграция и подготовка единственного пилота

Владелец слил [PR107](https://github.com/QahNaaRlN/grace/pull/107)
2026-10-23 15:43:34 UTC / 20:43:34 +05:00.
Merge `84266bd72ac10d569099e922cb826a8a01330252`;
final head `8da7f45b105105d6596032850893bb690febebd8`.
`git diff --exit-code <head> <merge>` PASS; head является предком merge.
Primary fast-forward до merge, clean; frozen E6/Q6 refs сохранены.

Exact source пилота — **`8da7f45b105105d6596032850893bb690febebd8`**,
опубликованный fixed ref **`codex/statement-s119-pilot-source`**.
Он больше не передвигается. Его [CI39965496366](https://github.com/QahNaaRlN/grace/actions/runs/39965496366)
повторно проверен canonical `verify-screening-source-ci.py`: completed/success,
exact SHA и все5 jobs PASS. [Merge CI39980740312](https://github.com/QahNaaRlN/grace/actions/runs/39980740312)
SUCCESS5/5, exact SHA/full jobs проверены canonical validator отдельно от head.
[Delivery39981953935](https://github.com/QahNaaRlN/grace/actions/runs/39981953935)
SUCCESS4/4, exact merge SHA и все4 jobs проверены отдельно. Это публикация
образов, не deployment.

### Manifest и точные inputs (ещё не отправлены)

[Manifest](../research/statement-quality/s119-pilot-manifest.json) получен
из clean frozen source worktree следующими командами (только check):

```sh
source_sha=8da7f45b105105d6596032850893bb690febebd8
bundle="/tmp/grace-s119-local-manifest-$source_sha"
mkdir -p "$bundle"
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GOTOOLCHAIN=go1.24.0 go build -trimpath -ldflags "-X main.sourceSHA=$source_sha" -o "$bundle/grace-clock-pilot" ./cmd/statement-clock-pilot
"$bundle/grace-clock-pilot" -mode=check > "$bundle/MANIFEST.json"
```

Его SHA256:
`2e4f3bfb766041086fa70009dbfdf5400603a3d85f15f4d4bb40e381811a25bf`.
Local bundle6/6 checksums PASS. CI соберёт свой binary и опубликует только
MANIFEST/SOURCE_SHA/SHA256SUMS; локальный binary digest не считается digest CI.
Ни `-mode=run`, ни metadata/provider/SSH/VPS preflight не выполнялись.

Workflow: `.github/workflows/statement-clock-pilot.yml`, один job `clock-pilot`;
ref `codex/statement-s119-pilot-source`, inputs:

```json
{
  "authorized_source": "8da7f45b105105d6596032850893bb690febebd8",
  "execute_approved_pilot": true
}
```

Это предъявленные параметры будущего dispatch, **не разрешение и не запуск**.
До отдельного решения `execute_approved_pilot` не отправляется;
workflow default остаётся false. Read-only GitHub list:0 запусков этого workflow.
GitHub workflow id394220010 active; environment production существует,
protection_rules=[] (только GitHub metadata, не состояние VPS).

### Объём запрашиваемого разрешения

Один workflow_dispatch, одна identity `s119-clock-pilot`, максимум6 POST:
base/candidate/candidate/base/base/candidate,3 пары только A/cards.
Repo policy: `google/gemini-3.1-flash-lite-20260507`, Google/`google-vertex/eu`,
input/output100000/16000,120s на вызов; provider cap323400 USD micros **$0.3234**.
Прежние fees и shared allowance сохраняются; разрешение не пополняет бюджет.
Fresh exact model/route/capability/ZDR/price и доступный бюджет проверит только
разрешённый job. Сейчас их runtime состояние **UNVERIFIED**; несовпадение — отказ.

Разрешённый job загрузит только новый public bundle на Grace VPS и исполнит
research binary в существующем API container, с записью отдельного one-shot
marker/private report0600 и reserve/settle в существующем metering ledger.
Изменения финансовых объектов не входят в pilot. Фиксированные файлы:
`/opt/grace/shared/statement-s119-clock-pilot.started` и
`/opt/grace/shared/statement-s119-clock-pilot.json`;
temp bundle `/tmp/grace-statement-clock-s119-<source>` удаляется wrapper.
Private proposals разбираются внутри того же job; GitHub получает закрытый
summary и безопасный manifest, без payloads/идентификаторов.

Marker/report общие для всех SHA этого pilot. Preflight refusal тоже расходует
one-shot; протокол/route/timeout/unknown billing/settlement error останавливает
весь pilot. После KEEP_FOR_FURTHER_REVIEW / REJECT / INCONCLUSIVE — отчёт и стоп,
без rerun, другого prompt, модели, deployment или production activation.
Старые private E6/P6/D5 и их локальные копии остаются закрыты.

**Pre-dispatch receipt: NOT_AUTHORIZED / NOT_RUN.** Owner approval/run id,
actual calls/cost/settlement и H1 decision пока отсутствуют и не выдумываются.


### Локальные post-merge gates

`TMPDIR=/tmp VITEST_MAX_WORKERS=1 make test` на clean primary84266bd — PASS,
exit0: Go,88 Web files/334 tests, lint/FSD/Prettier/build/offline Jev.
Log `/tmp/grace-s119-postmerge-test.log`.

Первый `make test-production` на primary — FAIL: Docker helper uid10001
не может читать fixture под primary0700, `TestS118VerifierRealHelperContext`,
permission denied; log `/tmp/grace-s119-postmerge-production.log`.
Права primary не ослаблялись. Повтор из отдельной worktree того же merge
`statement-s119-pilot-evidence` — PASS, exit0, включая pinned helper,
one-shot и остальные production/deployment/backup regression gates.
Log `/tmp/grace-s119-postmerge-worktree-production.log`.
Эти local gates не являются обращениями к production VPS.

`TMPDIR=/tmp make test-integration` из этой worktree — PASS, exit0:
все disposable PostgreSQL/API/helper suites, включая clock meter и uncertainty.
Log `/tmp/grace-s119-postmerge-worktree-integration.log`.
Ожидаемые отказные downgrade checks не скрывались.

`go vet ./cmd/... ./internal/...` — PASS, exit0;
`/tmp/grace-s119-postmerge-vet.log`. `git diff --check` — PASS.
Primary clean84266bd/mode0700; frozen source worktree clean8da7f45;
E6 ref24f7322 и Q6 ref5fdd0f9 не изменены.

Подготовка публикуется в ветке `codex/statement-s119-pilot-evidence` того же
пакета; отдельного readiness PR нет. После разрешённого pilot его approval,
run и итоговое evidence будут добавлены здесь же для одного итогового review.
Никакой запуск этим receipt не произведён: **NOT_AUTHORIZED / NOT_RUN**.
