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
R5 пока **NOT_RUN**: новых remote/model calls ещё нет. Исходный C6 FAIL и
consumed authorization не изменяются. Новые implementation exact SHA/CI
записываются перед разрешённым read-only dispatch.

## Критерии и исполняемые проверки

| ID | Evidence |
| --- | --- |
| R1 | `go test ./scripts/research/s112-diagnose -count=1` PASS: фиксированные source/report/marker; O_RDONLY/O_NOFOLLOW/O_NONBLOCK, regular0600, duplicate/depth/array/size limits; tampered source/profile/proof/digest/fixture/document/plan/request/order/state/cost отвергаются. Existing read-only lock: concurrent holder, missing/symlink/FIFO/public/oversize files отвергаются. До/после same inode/size/mode/mtime/SHA256; никакого report/marker write |
| R2 | Те же Go tests: generation/reservation/unknown alias/role/direction/locator/service sentinel не экспортируются; service reason заменён фиксированным, selections order/duplicates сохранены. `python3 scripts/research/test-statement-s112-diagnosis.py` PASS: fixed host/pinned SSH, closed failures, no memfd fallback; реальный executable проверил memfd identity и все4 kernel seals, directory bytes/mtime unchanged, stderr sentinel не выведен |
| R3 | `scripts/research/prepare-statement-s112.sh --self-test` PASS; `TestS112RealHelperStages` также PASS внутри `scripts/test-bank-statements.sh`. Pinned columns.pdf/Poppler25.04.0, network=none. Настоящий `RunComparison` с offline fake достигает canonicalization и merge failures: отсутствие raw proposal даёт canonicalization / exact guard UNVERIFIED; unknown role закрывает проекцию; retained sanitized proposal с несовпадающим merge даёт UNVERIFIED. Golden-created live response/reason не используется |
| R4 | `python3 scripts/research/test-screening-workflow-modes.py`:8192 actual job-condition combinations PASS, default=false/exclusive mode; reviewed exact diagnostic SHA и CI всех5jobs до SSH. `go test -race ./scripts/research/s112-diagnose -count=1`, `go vet ./cmd/... ./internal/... ./scripts/research/s112-diagnose`, `go vet -tags=integration ./scripts/research/s112-diagnose`, `make test-production`, `scripts/test-bank-statements.sh`, `git diff --check` PASS. `TMPDIR=/tmp VITEST_MAX_WORKERS=1 make test` PASS (Go, 88 Web files/334 tests, lint/FSD/format/build, offline Jev); `prepare --prepare` выдал6452bytes/46public spans |
| R5 | NOT_RUN. Исполнение возможно только после implementation exact-source CI5/5; один read-only dispatch уже принят владельцем вместе с R1–R5. Provider inference всегда0 |

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
