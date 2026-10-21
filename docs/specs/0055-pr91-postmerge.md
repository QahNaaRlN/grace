# PR №91: post-merge evidence S1.14 P1–P5

Дата: 2026-10-21, Asia/Yekaterinburg. Основание продолжения: «идем дальше»
после ready-for-review реализации P1–P5. [PR №91](https://github.com/QahNaaRlN/grace/pull/91)
слит с expected head `301afd6bedc4a19559e731471336bbc4e2f74075`.
Merge: `68ee7eefad1aa36f12faa1cc820a152644ad60c0`.
[Accepted S1.14](0055-s114-observed-comparison.md), [implementation evidence](0055-s114-evidence.md).
Acceptance a22f94d предшествует code9db80af.

## Post-merge local gates

- Primary main обновлён fast-forward до68ee7ee, status clean до/после checks.
- `TMPDIR=/tmp VITEST_MAX_WORKERS=1 make test` PASS;
  `/tmp/grace-pr91-postmerge-make.log`.
- `TMPDIR=/tmp make test-production` PASS;
  `/tmp/grace-pr91-postmerge-production.log`.
- `go vet ./cmd/... ./internal/... ./scripts/research/s113-replay` и
  `git diff --check` PASS.
- Первый `TMPDIR=/tmp make test-integration` в primary **FAIL до первого теста**:
  Go test container UID10001 не мог прочитать `go.mod` через readonly bind mount.
  Primary directory mode0700, read-only probe подтвердил Permission denied.
  Журнал `/tmp/grace-pr91-postmerge-integration.log`. Runtime/test code не менялся.
- Повтор `TMPDIR=/tmp make test-integration` в clean source worktree
  `statement-s114-p6-source` на exact68ee7ee **PASS всего suite**, включая настоящий
  helper/stdin replay6/6, S1.14 SQL reservation/uncertain и все финансовые suites;
  `/tmp/grace-pr91-postmerge-integration-source.log`. Primary permissions сохранены.
- Документационный worktree: `npm ci --prefix web`, затем
  `TMPDIR=/tmp VITEST_MAX_WORKERS=1 make test` PASS;
  `/tmp/grace-s114-p6-decision-make.log`. Только документация менялась.

## Exact source и незарегистрированный main CI

[Implementation CI39612267914](https://github.com/QahNaaRlN/grace/actions/runs/39612267914)
SUCCESS5/5 именно на `301afd6bedc4a19559e731471336bbc4e2f74075`.
GitHub не зарегистрировал push-CI/Delivery на merge68ee7ee. Нормальный push пустого
integration commit `c23f4c110b61f35cc11bcd9edf485d70b82f1260`, затем leased API
fast-forward на same-tree empty commit
`fd68235e4f79eec7a3153807da2c8e9a1a6bb1d1` также не показали runs.
Root cause не установлен; CI workflow active. Отсутствие runs — **UNVERIFIED**,
не PASS, не объясняется доказанной причиной и не заменяется CI другого SHA.
Дальнейших trigger attempts или workflow edits нет. Delivery не является deploy.

Primary mainfd68235 clean. Команды проверили:

```sh
git merge-base --is-ancestor 301afd6bedc4a19559e731471336bbc4e2f74075 main
git diff 301afd6bedc4a19559e731471336bbc4e2f74075 68ee7eefad1aa36f12faa1cc820a152644ad60c0 --exit-code
git diff 68ee7eefad1aa36f12faa1cc820a152644ad60c0 fd68235e4f79eec7a3153807da2c8e9a1a6bb1d1 --exit-code
```

Все PASS:301afd6 интегрирован, trees301afd6/68ee7ee/c23f4c1/fd68235 одинаковы.
**Предлагаемый executable source P6 — именно301afd6 с его собственным CI5/5**,
ref `codex/statement-observed-comparison` frozen, original implementation worktree
clean. Это явное предложение до owner approval; более новые main SHA не заявлены
как прошедшие CI. Ранее подготовленный source ref `codex/statement-s114-p6-source`
на68ee7ee не используется и не авторизован; worktree сохранён без force cleanup.

## Следующий gate и границы

P1–P5 integrated/offline, local post-merge gates PASS. Main push-CI/Delivery остаются
отдельно UNVERIFIED. [P6 decision](0055-s114-p6-run-decision.md) — draft на301afd6,
owner acceptance отсутствует, live_status NOT_AUTHORIZED/NOT_STARTED.

Предложение соблюдает accepted P6 gate: implementation review/integration,
post-merge local checks и exact executable-source CI5/5. До исполнения нужны
прямое owner approval на301afd6 и обязательный runtime preflight. Ни main merge,
ни empty commits, ни source ref не являются paid authorization.
Provider metadata/inference, SSH/VPS, workflow dispatch/deploy=0.
Старый C6 FAIL/consumed, exact guard UNVERIFIED; реальные ledger/markers/reports,
production model/flags/image не менялись.
