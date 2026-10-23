# S1.19 — offline C1–C4 local_verified, live NOT_AUTHORIZED

- Контракт: [accepted C1–C4](0055-s119-evidence-selection-candidate.md).
- Owner acceptance: прямое «Принять, интегрировать и продолжить offline» на
  PR106/c9b621d; разрешены фиксация принятия, merge после CI5/5 и offline работа.
- Implementation code: `b9809d8319de4d6e0f45de79faf8abc0f2e5d6a4`; branch `codex/statement-s119-clock-pilot`.
- База: main `db65225c6d2cfc9052ccf4f7310ced61c5ba79d8`.
  Принятый контракт интегрирован **до** implementation worktree.
- Статус: **local_verified offline / integration pending**.
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

## Границы и следующее решение

Default product prompt/profile/consent, schema/planner/aliases/redaction,
financial validators/scorer, API/OpenAPI/Web/migrations и старый deploy workflow
не изменены. Exact-price restriction включается только новым research adapter;
legacy comparison default сохраняется. Production constructors не выбирают его.

External inference/metadata, SSH/VPS, production SQL, deploy, owner financial
writes0. Private E6/P6/D5/closed stdout не читались. Ни одного нового workflow
запуска не было; old E6 FAIL/Q6 SUCCESS и consumed permissions сохранены.

Offline реализация ещё не интегрирована. После review/integration и exact-source
CI5/5 в этом же пакете предъявляются source/ref/manifest/inputs для отдельного
одного pilot. Новый draft/readiness PR для каждой записи не требуется.
Ни live quality PASS, ни winner, ни production activation здесь не заявляются.
