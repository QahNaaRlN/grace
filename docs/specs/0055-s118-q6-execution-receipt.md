# S1.18 / Q6 — разрешение и receipt единственного dispatch

Запись до dispatch: 2026-10-23 11:48:30 UTC / 16:48:30 +05:00, Asia/Yekaterinburg.

- Контракт: [accepted S1.18 Q6](0055-s118-saved-quality-diagnosis.md),
  [конкретное предложение](0055-s118-q6-run-decision.md), [точные inputs](0055-s118-q6-inputs.json).
- Effective execution approval: прямой ответ владельца **«Разрешаю один Q6»**
  на отдельный вопрос о source `5fdd0f93380b72f6f3bdad03715f9a83744cace5`,
  ref `codex/statement-s118-q6-source`, одном workflow_dispatch с SSH-чтением
  только сохранённого E6 на VPS Grace и закрытым результатом через CI,
  без платных вызовов, SQL, финансовых записей и повторов.
- При публикации этой записи: **AUTHORIZED / NOT_DISPATCHED**, permission UNUSED
  до единственного запроса dispatch; после запроса — CONSUMED независимо от итога.
- Source/ref local и remote exact5fdd0f9, frozen worktree clean; ref не двигается.
  Source интегрирован №104 в main438256b; ancestry и byte-identical trees PASS.
- [Exact-source CI39945454718](https://github.com/QahNaaRlN/grace/actions/runs/39945454718)
  повторно проверен перед dispatch: SUCCESS5/5, canonical full jobs/SHA check PASS.
  Merge CI39947122855 отдельно5/5, Delivery39948123236 отдельно4/4 PASS.
- [Локальный readiness manifest](0055-s118-q6-local-SHA256SUMS) подготовлен до
  approval; runtime job создаёт/проверяет свой полный manifest перед SSH.
  Его будущие fingerprints не подменяются локальными.

## Разрешённый scope

Ровно один job `diagnose-statement-s118` в `deploy-production.yml` на frozen ref.
Все остальные modes=false; actual conditions для inputs выбирают только Q6.
Логические boolean inputs при gh CLI transport передаются строками true/false;
семантика inputs не меняется. Никакого fallback/ref/source/path/mode switch.

Чтение только fixed E6 `/opt/grace/shared/statement-comparison-s117-comparison-20261023.json`
и `.started`; artifactSource остаётся24f7322, verifierSource5fdd0f9 отдельно.
Private artifact не копируется с VPS. На VPS sealed memfd/env={}, fixed0600/
O_NOFOLLOW reads и before/after inode/bytes/hash/size/mode/mtime; stdout closed.
Binary<=16MiB/context<=524288/envelope<=32MiB/output<=65536bytes, remote45s/SSH75s.
Нет provider/metadata/SQL/API/Docker/metering calls или financial/ledger/runtime
writes, новых health probes/permissions/install/config/image/flag изменений.
Закрытые witnesses остаются только stdout разрешённого CI job; не в Git/PR/
Actions artifacts. Manifest/source — отдельный разрешённый artifact.

Один dispatch потребляет permission также при preflight failure, timeout или
UNVERIFIED. Автоматического retry/repair нет. После исполнения записать run/job,
точный SHA/ref, статусы6 tuples и safe counts, artifactsUnchanged и ограничения.
Исторический E6 quality FAIL и его CONSUMED permission не меняются.
