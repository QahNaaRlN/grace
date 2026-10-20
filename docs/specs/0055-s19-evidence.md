# SPEC-0055 / S1.9: подтверждения бесплатной диагностики

Дата: 2026-10-21, Asia/Yekaterinburg.
[Принятый контракт](0055-s19-saved-trace-diagnosis.md).
Acceptance `6d1f369` предшествует executable source
`ecfa85078e0fae6d9f9e519f5e72b0126728fb45`.
[PR №81](https://github.com/QahNaaRlN/grace/pull/81).

| ID | Исполняемое подтверждение |
| --- | --- |
| D1 | `python3 scripts/research/test-statement-s19-diagnosis.py` PASS: strict JSON/duplicate keys/depth, source/profile/pin/schema/locator/size/mode/symlink, private sentinel и замена service reason; неизвестная причина закрывается unknown. File hashes/size/mode/mtime неизменны, existing lock открыт read-only, конкурентный lock отклонён; при отказе только фиксированный unavailable |
| D2 | `python3 scripts/research/test-screening-workflow-modes.py` PASS: 2048 сочетаний реальных job conditions. S1.9 default=false, при нём ровно один диагностический job либо ни одного; конфликт не запускает платные/inspect/reconcile/deploy jobs. Remote script не имеет subprocess/SQL/provider-network пути |
| D3 | `scripts/research/replay-statement-s19.sh --self-test` PASS: pinned real helper, network=none у helper и runner. Correct19/19, separator18/19, label18/19, combined17/19; сохранённый score и source hash при подмене отклонены. Те же counterexamples и TestS19RetainedReplay пройдены в финальном `scripts/test-bank-statements.sh`, подключены к существующему CI integration |
| D4 | `TMPDIR=/var/tmp/grace-quality-next-tests VITEST_MAX_WORKERS=1 make test` exit0: Go, 88 файлов/334 Web tests, lint/FSD/Prettier/build/Jev. `make test-production`, `scripts/test-bank-statements.sh`, `go vet ./cmd/... ./internal/... ./scripts/research/s19-replay`, `git diff --check` PASS. Старые activation19/20 и RU23/23, SQL lifecycle/restore/destructive-down guard сохранены. [CI source39510485510](https://github.com/QahNaaRlN/grace/actions/runs/39510485510): на срезе перед публикацией retained regression Quality и три image builds PASS, PostgreSQL integration pending; это не общий CI PASS. Финальные head checks — в PR №81 |
| D5 | [Read-only workflow39510628212](https://github.com/QahNaaRlN/grace/actions/runs/39510628212) SUCCESS; provider inference=0, artifactsUnchanged=true, replayMatchesSaved=true. Подтверждены пропущенный дефис строки2 и два header ambiguous_money; все paid/inspect/reconcile/deploy jobs skipped |

## Границы изменений

`git diff --exit-code 5e9116e -- internal cmd api migrations web deployments/production scripts/research/inspect-statement-screening-s18.py` exit0.
Прежние Go transport/prompt/schema/scorer/planner/цены, Web/API, миграции,
product consent и операторы S1.4/S1.6/S1.8 неизменны. Меняется только workflow
взаимоисключения, добавляются read-only reader и отдельный локальный Go replay,
privacy/helper tests и документация. Public API продукта не расширен.

В первой редакции real-helper теста было подтверждение только страницы1;
исправлено на все страницы документа. Удаление подписи оборота заменено в
контрпримере на available-balance label, чтобы отделить blocker от turnover.
Это исправления fixture теста, validator/scorer не изменялись. Первоначальный
запуск с пустой проекцией ожидаемо завершился s19_replay_unverified после PASS
контрпримеров; отдельный self-test затем завершён exit0.

Журналы: `/tmp/grace-s19-make.log`, `/tmp/grace-s19-production.log`,
`/tmp/grace-s19-bank.log`, `/tmp/grace-s19-bank-final.log`, `/tmp/grace-s19-vet.log`,
`/tmp/grace-s19-helper-positive.log`, `/tmp/grace-s19-diagnostic-watch.log`.
Частный исходный report не копируется. Сохранённый FAIL S1.8 и consumed marker
не изменяются; новая диагностика не даёт права на inference или model activation.

## D5: фактическое исполнение

Dispatch **03:47:30 +05:00** / 2026-10-20 22:47:30 UTC, завершение
**03:48:32 +05:00** / 22:48:32 UTC. Единственный запуск, source `ecfa850`.
На VPS выполнен только Python reader через stdin: O_RDONLY/O_NOFOLLOW,
existing exclusive lock без создания/усечения, immutable report/marker.
Ни SQL, ни Docker mutations, ни provider key/network на VPS не использовались.
Pinned helper и Go replay запускались на GitHub runner с network=none.

Projection/replay скачаны из artifact statement-s19-sanitized и сохранены в
[исследовании](../research/statement-quality/s18-saved-trace-diagnosis.md).
Private original report не скачивался. Known cost и ledger в S1.9 не читались
и не изменялись; статус billing taken from S1.8 evidence, не новая сверка.

Результат columns19/17 воспроизведён. Строка2 потеряла дефис; оба book balance
содержали знак `-` из заголовка банка и были отклонены как ambiguous_money.
Тест TestS19RetainedReplay закрепляет фактический trace и проверяет, что все
три дефиса были видимы модели буквально и имели прежний numeric_fragment.
Локальный повтор только replay на sanitized projection также PASS; нового
remote dispatch не было. Исторический S1.8 FAIL не исправлен этим пакетом.

Source CI относится к исполняемому diagnostic source ecfa850; после запуска
добавлены только retained synthetic regression, документация и его CI wiring.
Окончательный head CI фиксируется отдельно. Merge/deployment не выполнялись.
