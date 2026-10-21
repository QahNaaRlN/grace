# SPEC-0055 / S1.12: implementation и diagnostic evidence

Дата: 2026-10-21, Asia/Yekaterinburg.
[Accepted S1.12](0055-s112-saved-comparison-diagnosis.md): прямое «утверждаю»
приняло R1–R5. Acceptance commit `cf8c57efedf75af0c2efaa5f672279329a778e14`
предшествует коду; спецификация интегрирована [PR №86](https://github.com/QahNaaRlN/grace/pull/86),
main `33cd15a7dfd34257a634e2c76831be60b5d7c4b4`.
[Spec CI39590043589](https://github.com/QahNaaRlN/grace/actions/runs/39590043589):5/5 PASS.
PR №85 с C6 evidence интегрирован `4785b0c66d5fe8e279556e7d34dad6b63d215b41`.

Implementation worktree `statement-comparison-diagnosis`, branch
`codex/statement-comparison-diagnosis`, создан от main33cd15a.
R1–R5 выполнены. Implementation exact source
`f202f857b103416694553f6f4efda19a31d2eeb8`, ref `codex/statement-comparison-diagnosis`.
[CI39592301585](https://github.com/QahNaaRlN/grace/actions/runs/39592301585):5/5 PASS
до dispatch. [R5 run39593210426](https://github.com/QahNaaRlN/grace/actions/runs/39593210426)
**SUCCESS**; новых provider calls0. Исходный C6 FAIL/consumed не изменяются.

## Критерии и исполняемые проверки

| ID | Evidence |
| --- | --- |
| R1 | `go test ./scripts/research/s112-diagnose -count=1` PASS: фиксированные source/report/marker; O_RDONLY/O_NOFOLLOW/O_NONBLOCK, regular0600, duplicate/depth/array/size limits; tampered source/profile/proof/digest/fixture/document/plan/request/order/state/cost отвергаются. Existing read-only lock: concurrent holder, missing/symlink/FIFO/public/oversize files отвергаются. До/после same inode/size/mode/mtime/SHA256; никакого report/marker write |
| R2 | Те же Go tests: generation/reservation/unknown alias/role/direction/locator/service sentinel не экспортируются; service reason заменён фиксированным, selections order/duplicates сохранены. `python3 scripts/research/test-statement-s112-diagnosis.py` PASS: fixed host/pinned SSH, closed failures, no memfd fallback; реальный executable проверил memfd identity и все4 kernel seals, directory bytes/mtime unchanged, stderr sentinel не выведен |
| R3 | `scripts/research/prepare-statement-s112.sh --self-test` PASS; `TestS112RealHelperStages` также PASS внутри `scripts/test-bank-statements.sh`. Pinned columns.pdf/Poppler25.04.0, network=none. Настоящий `RunComparison` с offline fake достигает canonicalization и merge failures: отсутствие raw proposal даёт canonicalization / exact guard UNVERIFIED; unknown role закрывает проекцию; retained sanitized proposal с несовпадающим merge даёт UNVERIFIED. Golden-created live response/reason не используется |
| R4 | `python3 scripts/research/test-screening-workflow-modes.py`:8192 actual job-condition combinations PASS, default=false/exclusive mode; reviewed exact diagnostic SHA и CI всех5jobs до SSH. `go test -race ./scripts/research/s112-diagnose -count=1`, `go vet ./cmd/... ./internal/... ./scripts/research/s112-diagnose`, `go vet -tags=integration ./scripts/research/s112-diagnose`, `make test-production`, `scripts/test-bank-statements.sh`, `git diff --check` PASS. `TMPDIR=/tmp VITEST_MAX_WORKERS=1 make test` PASS (Go, 88 Web files/334 tests, lint/FSD/format/build, offline Jev); `prepare --prepare` выдал6452bytes/46public spans |
| R5 | PASS: [job119199055924](https://github.com/QahNaaRlN/grace/actions/runs/39593210426/job/119199055924), exact f202f85/CI5/5; digestVerified=true, artifactsUnchanged=true, providerCalls=0. Stage canonicalization, raw proposal отсутствует, exactGuard=UNVERIFIED; recorded cost1995micros/latency2931ms. Другие8jobs skipped, paid/deploy не выполнялись |

Локальные журналы: `/tmp/grace-s112-helper.log`, `/tmp/grace-s112-production.log`,
`/tmp/grace-s112-bank.log`, `/tmp/grace-s112-make.log`.

## Реализованные границы

Новый CLI/operator находится только в `scripts/research/s112-diagnose`.
`prepare` извлекает только pinned columns.pdf через локальный Unix socket;
remote default читает фиксированные artifacts и lock, сверяет полный прежний Go
digest и report contract; `replay` исполняется локально над sanitized projection.
Отсутствующий raw alias proposal не восстанавливается и не заменяется golden.

Локальный wrapper передаёт bounded binary/context envelope фиксированному SSH host.
Python bootstrap проверяет SHA256 executable, создаёт sealed memfd, запускает
verifier с пустым environment и подавленным stderr; filesystem install/fallback нет.
Полный private report остаётся на VPS. Ни SQL, ни Docker/server mutations, ни
provider key/catalog/inference, ни reserve/settle/reconciliation не вызываются.
Cost/latency — только записанные numeric значения synthetic C6; это не новая
ledger inspection, прежний known/settled вывод остаётся датированным evidence C6.

Workflow mode `diagnose_statement_s112_only` выключен по умолчанию;
`diagnosis_authorized_sha` по умолчанию пуст. Режим исключает остальные8jobs,
конфликтующие inputs ничего не запускают. Production concurrency сохранена.
SSH key конфигурируется только после offline/source gates; remote command
не копирует файлы на VPS. Artifacts содержат только закрытые projection/result.

Продуктовые Go-модули, OpenAPI/DTO/SQL/UI, prompt/schema/planner/redaction/scorer,
финансовый core и предыдущие reports/markers не меняются. S1.8 FAIL17/19,
S1.9 trace16/19 и C6 protocol FAIL сохраняются; paid retry/adoption не открыты.

## Фактический R5 и граница диагноза

2026-10-21 **16:37:47 +05:00 / 11:37:47 UTC**:
`verified=true`, `digestVerified=true`, `artifactsUnchanged=true`,
`stage=canonicalization`, `exactGuard=UNVERIFIED`,
`replay=unavailable_missing_raw_proposal`. Один diagnostic dispatch;
все8 остальных jobs skipped, включая paid C6 и Deploy owner VPS.
Read-only SSH/memfd и локальный replay завершились SUCCESS;
artifact `statement-s112-sanitized` содержит только projection/result.

Из прежнего C6 report, сверенного полным Go digest и pinned-helper hashes:

| Поле | Подтверждённое значение |
| --- | --- |
| Campaign time | 2026-10-21T10:51:32.22918582Z → 10:51:35.191103488Z |
| A/columns | attempted / protocol_failed, proposalPresent=false |
| Остальные5 | not_sent / campaign_not_started, proposalPresent=false |
| Recorded calls / completed tuples | 1 / 0 |
| Billing known / actualMicros | true / 1995 = **$0.001995 provider**, без пересчёта fees |
| Recorded Complete latency | **2931ms**, включает CurrentProof/network/decode внутри Complete; не чистое generation time и не p95 |
| Columns document hash | `14fedad79043a4f191719c8b26d3f74c5e8e555e0fcc507a90132603005d88a1` |
| A plan hash | `eb48a78739b0a33ded5398bada96eab5060fa4646f4b6f8e8301e8154c9801e2` |
| A request hash | `9a4655cb1fe0efab7ec4f0ae25cc9acb3c2a0da4856cb5ff83a6e53a945ceb13` |

Это исторические значения C6, не новая ledger verification/settlement.
Прежний inspector known/unique/settled/report=ledger/sharedUncertainBlock=false
остаётся evidence исходного C6 job. R5 не обращался к SQL или provider.

Control flow executed source012f604 доказывает стадию canonicalization, поскольку
canonical proposal не присвоен до отказа. Exact alias selection и причина между
invalid plan / nonnumeric money alias / unknown alias / duplicate selection
не сохранены; выбирать одну из них по golden PDF или править guards нельзя.
Quality comparison A/B по-прежнему UNVERIFIED, победителя нет. Наблюдаемость
протокольного отказа и возможное исправление требуют нового принятого пакета;
S1.12 не открывает writer changes, paid retry или production adoption.
