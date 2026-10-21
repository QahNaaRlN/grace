# SPEC-0055 / S1.14: offline implementation evidence

Дата: 2026-10-21, Asia/Yekaterinburg.
[Accepted S1.14](0055-s114-observed-comparison.md): владелец принял **только P1–P5**
прямым «принимаю». Acceptance commit `a22f94d79db1733ebe930b8b5a9ff01d02e766ff`
предшествует production-коду. [PR №90](https://github.com/QahNaaRlN/grace/pull/90)
интегрирован в main `bc7aaf27f382d8aa300d510e758f1c7e9fe8543a`;
[accepted-head CI39609572384](https://github.com/QahNaaRlN/grace/actions/runs/39609572384)
SUCCESS5/5. Это CI спецификации, не реализации.

Implementation worktree `statement-observed-comparison`, branch
`codex/statement-observed-comparison`, создан от локально интегрированного accepted
main a22f94d; subsequent remote merge bc7aaf2 содержит тот же acceptance commit.
Code commit `9db80afc57c42fd283e1b6daf46d9e1efe027d01`; следующий commit содержит
только это evidence, spec/index status updates. Exact итоговый head и его CI
фиксируются в implementation PR/handoff после завершения.

Статус: implemented offline / local gates PASS.
P6=NOT_AUTHORIZED, live source approval отсутствует. Provider metadata/inference,
SSH/VPS/workflow dispatch/deploy=0. Historical C6 FAIL/consumed, exact guard
UNVERIFIED, production model/flags/image и реальные ledger/artifacts не менялись.

## P1–P5 и исполняемое evidence

| ID | Результат и команда |
| --- | --- |
| P1 | PASS: `go test ./internal/statementqualification -run S114 -count=1`. Общая prepare/run/validation логика с закрытым private context; S1.14 authorization и outer version проверяются до admission по исходным inner/outer digests. Cross-auth/version, wrong digest, missing snapshot/forged diagnostic и новый bare v1 отвергаются. Legacy validation/summary/observed decoder остаются закреплены за S1.11; `DecodeComparisonReport` как прежде только синтаксический, admission всегда требует validation |
| P2 | PASS: `scripts/test-statement-comparison-s114.sh`; real shell runner с offline Docker stubs. Wrong SHA, no-clobber на same/other SHA, success/failure, symlink/orphan preflight/exit, общий exclusive lock, forged manifest и сохранность старого C6/older artifacts. `python3 scripts/research/test-screening-workflow-modes.py`:16384 комбинации actual job predicates; новый mode default-off, mixed inputs не выбирают соседние/deploy jobs; отдельный exact SHA input, CI5/5 и manifest обоих Go binaries/scripts/PDF |
| P3 | PASS: тот же Go S114 table сравнивает с legacy runner behavior/calls/order/hashes/reserve/settle/actual на protocol/unknown/duplicate/route/quality/overflow/settlement cases. Положительные/guard реальные fixtures ниже; прежние HTTP transport tests проходят в full Go suite. Model/policy/prompt/schema/scorer/caps не менялись |
| P4 | PASS: `python3 scripts/research/test-comparison-s114.py`: настоящий Go engine→private writer→CLI→fixed Python inspector; read-only SQL stubs требуют exact UUID, единственную row, S1.14 model label, known cost и правильный status. Corruption/private sentinel/mode/symlink/duplicate JSON/old ledger/wrong reservation/cost cases отвергаются без raw output. Реальный replay CLI при недоступном helper честно UNVERIFIED. Bytes/mode/mtime сохранены. Real helper и SQL доказательства ниже |
| P5 | `make test`, `make test-production`, полный `make test-integration`, race/vet/diff PASS. Exact implementation-head CI5/5 остаётся отдельным незавершённым gate до live review |

## Настоящий helper, CLI и PostgreSQL

`TMPDIR=/tmp scripts/test-bank-statements.sh` после последней tagged-test правки
PASS (`/tmp/grace-s114-bank-cli-final.log`):

- `TestS114RealHelperReplay`: network=none/read-only, pinned Poppler25.04.0,
  delivered-alias fake, прежние columns/cards/RU scores19/20/23; common replay6/6.
  **Настоящий built CLI** получает private envelope через stdin в том же isolated
  container и даёт matched6/6. Три causal alias guards дают matched для captured
  tuple, остальные unavailable_capture. ProviderCalls0, private strings не выводятся.
- Дополнительный test CLI process потребовал test-runner pids-limit32 вместо16;
  первоначальный усиленный probe FAIL сохранён в `/tmp/grace-s114-bank-cli.log`,
  повтор после изменения test container PASS. Production/helper limits не менялись.
- `TestS114StatementMeterSQL`: disposable migrated PostgreSQL, настоящий
  `s114ComparisonMeter`, owner-scoped reserve/call_allowed/settle. Одна новая
  reservation model=`statement-comparison.s114`, provider=`fixed-A-B`, purpose
  statement,208500 reserved,42 metered на causal protocol FAIL, settled exact UUID.
  Историческая тестовая S1.11 row byte-for-byte unchanged. Unknown response
  сохраняет NULL actual/uncertain и блокирует следующий reserve; без principal
  reserve запрещён. Только fixture-owned rows очищаются; реальный ledger не читался.
- Existing bank/SQL/retained/S1.10/S1.12/S1.13 regressions, restore и запрет downgrade
  с потерей mapping PASS. Ожидаемый SQL downgrade refusal — часть положительного gate.

Fake scores и42micros — только synthetic test evidence, не результат новой модели
или новая запись реального провайдера.

## Реализованный контракт и scope review

`statement.comparison.observed.s114.v1` оборачивает прежнюю report schema с новым
fixed authorization и прежними S1.13 bounded observations. Legacy entry points не
принимают новый context. Общий canonicalizer, admission predicates, planner/prompt/
schema/redaction/alias/scorer versions, profiles/prices/caps/fees не изменены.
Replay stdin принимает только S1.14; absolute-file offline mode сохраняет legacy
validation и unavailable_legacy. Summary/replay возвращаются до config/key/SQL/
provider constructors.

Новый fixed runner использует прежний shared lock, separate marker/report0600,
no-clobber на writes и новый bundle prefix. Новый workflow job допускается только
изолированно, при exact source и CI5/5. Manifest включает admin, replay binary,
три PDFs, SOURCE_SHA, runner и оба inspector scripts. Read-only replay использует
private stdin в existing api container и existing Unix helper socket, без raw
копии в Actions. Inspector выводит closed observations, проверенные числовые
metrics и результат replay; failures не печатают exceptions/payload/IDs.

Runtime diff: comparison core/CLI, fixed operator runner/inspector/replay,
workflow input/job isolation. Остальное — tests/gates/docs. Product API/OpenAPI,
Web, migrations, SQL functions/financial core, runtime policy и реальная среда
не менялись. Legacy Python wrappers используют общие validation helpers и прежние
fixed constants; arbitrary campaign/path/provider flags не добавлялись.

## Gates и журналы

- `TMPDIR=/tmp VITEST_MAX_WORKERS=1 make test` PASS: Go,88 Web files/334 tests,
  lint/FSD/format/build/Jev; `/tmp/grace-s114-make.log`.
- `TMPDIR=/tmp make test-production` PASS; `/tmp/grace-s114-production.log`.
- `TMPDIR=/tmp make test-integration` PASS; `/tmp/grace-s114-integration.log`.
- `TMPDIR=/tmp scripts/test-bank-statements.sh` final PASS; журнал выше.
- `go test -race ./internal/statementextract ./internal/statementqualification`
  PASS; `/tmp/grace-s114-race.log`.
- `go vet ./cmd/... ./internal/... ./scripts/research/s113-replay` и
  `go vet -tags=integration ./cmd/statement-admin ./internal/statementqualification` PASS.
- `git diff --check` PASS. Targeted inspector/shell/workflow logs:
  `/tmp/grace-s114-inspector.log`, `/tmp/grace-s114-shell.log`, `/tmp/grace-s114-workflow.log`.

## Следующий gate

Local full integration и scope review завершены. Завершить exact final-head CI5/5 и предъявить
implementation PR. После интеграции повторить post-merge gates. Только затем
возможно отдельное предложение P6 на конкретном source/ref, с прежними pairs,
до6POST/$0.2085 provider и обязательным preflight. P1–P5 acceptance, этот report
и merge не разрешают P6, metadata/SSH/dispatch, model switch или deployment.
