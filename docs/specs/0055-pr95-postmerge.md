# PR №95 — post-merge и граница исполнения D5

Дата: 2026-10-23, Asia/Yekaterinburg. Основание: «влил PR, продолжить работу».

- [PR №95](https://github.com/QahNaaRlN/grace/pull/95) MERGED,
  merge `403f69d9019cf3ef65db2dc126fe7e274bfabe48`,
  2026-10-23 02:40:34 +05:00 / 2026-10-22 21:40:34 UTC.
- Primary main fast-forward17e3493→403f69d; clean до/после. Actual diff только8 docs.
  `git merge-base --is-ancestor 703b6c6 HEAD` и `git diff --exit-code 703b6c6 HEAD -- . ':(exclude)docs'` PASS:
  runtime/workflow/financial/API/Web tree по-прежнему совпадает с frozen source.
- Remote/local source ref `codex/statement-saved-proposal-diagnosis` остаётся
  `703b6c6ac960c21c0b7ab7bd84b3411459204c92`; source worktree clean.
- [Source CI39639515724](https://github.com/QahNaaRlN/grace/actions/runs/39639515724)
  повторно получен/проверен canonical exact-source validator: SUCCESS5/5.
  Это CI703b6c6, не CI merge403f69d.
- [Merge CI39859170136](https://github.com/QahNaaRlN/grace/actions/runs/39859170136)
  на403f69d на момент capture:4/5 PASS (quality и3 image builds),
  PostgreSQL integration выполняется; полный SUCCESS заранее не заявляется.
  Для frozen source сохраняется подтверждённый [PR94 post-merge](0055-pr94-postmerge.md).
- Workflow history с2026-10-21 16:50UTC повторно проверен: operator runs отсутствуют.

## Историческая подготовка D5 и отказ разрешения

Зафиксированы exact inputs из [D5 candidate](0055-s115-d5-run-decision.md).
Actual workflow matrix32768 combinations PASS; `selected(inputs)` выбирает ровно
`diagnose-statement-s115`, без paid/deploy/reconciliation/старых diagnostic jobs.

Ответ владельца первоначально истолкован как разрешение D5; local commit7610661
сделан до попытки dispatch. Automatic approval review **отклонил CreateProcess**:
«продолжить работу» не признано явным разрешением production workflow с SSH-чтением
private P6 artifact и передачей closed projection через CI. Эта первоначальная
интерпретация скорректирована commit68d5f7a. Effective execution approval отсутствует.

Shell/dispatch не выполнялся: dispatch_count=0, SSH/VPS/provider/metadata/SQL/
persistent artifact/environment writes=0. Не было повторного dispatch, другого
API/tool, прямого SSH, обхода проверки, chmod, смены path/source или repair.
Отказ разрешения — не diagnostic FAIL; actual P6 witnesses остаются UNVERIFIED.

## Исторический execution gate (впоследствии закрыт)

Нужно явное подтверждение одного D5 на703b6c6/ref, включающего private P6 read
на fixed VPS и передачу только allowlisted numeric/enum projection через CI.
Provider/metadata/SQL/writes=0 и no-retry ограничения исходного кандидата сохраняются.
До такого ответа D5 BLOCKED_PENDING_EXPLICIT_APPROVAL / NOT_EXECUTED.

## Локальные gates

`TMPDIR=/tmp VITEST_MAX_WORKERS=1 make test` — PASS,
log `/tmp/grace-s115-d5-make.log`; `git diff --check` — PASS.
Нового локального full integration для docs-only пакета не заявляется;
полный integration frozen source подтверждён source CI5/5. Никаких runtime probes.

## Последующее явное разрешение и workflow execution

Владелец прямо ответил «разрешаю» на конкретный private-read/CI scope и exact703b6c6.
Approval commit e5961a4; [run39860669842](0055-s115-d5-execution-receipt.md) завершён
workflow SUCCESS, только D5 job; остальные10 jobs skipped. Прежний
BLOCKED_PENDING_EXPLICIT_APPROVAL выше — historical capture, не текущий gate.
Один dispatch использован. Closed payload сохранён только в CI: перенос в Git
отклонён отдельной automatic review до CreateProcess и не выполнялся.

Post-merge [CI39859170136](https://github.com/QahNaaRlN/grace/actions/runs/39859170136)
на403f69d впоследствии завершён5/5 SUCCESS, повторно проверен canonical validator;
[Delivery39860348646](https://github.com/QahNaaRlN/grace/actions/runs/39860348646)
на том же head завершён4/4 SUCCESS, все jobs checked. Это публикация images, не deploy.
