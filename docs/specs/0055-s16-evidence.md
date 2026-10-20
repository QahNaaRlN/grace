# SPEC-0055 / S1.6: подтверждения реализации и единственного прогона

Дата: 2026-10-21, Asia/Yekaterinburg. [Принятый контракт](0055-s16-diagnostic-screening.md),
утверждение «Утвердить S1.6 и один ограниченный smoke»; commit `9a9268d`
предшествует исполняемым изменениям. [PR №79](https://github.com/QahNaaRlN/grace/pull/79).
Исполняемый SHA: `9e89ae08b7ec0b2b60d642b7f2c3d29a7f52598f`.

| ID | Исполняемое подтверждение |
| --- | --- |
| N1 | `sh scripts/test-statement-screening-s16.sh` PASS: preflight failure не расходует marker; failure/success расходуют его один раз; повтор, другой SHA и report без marker не делают новый вызов. Старые S1.4 report/marker проверены sha256sum и сохранены. Прежний `scripts/test-statement-screening.sh` также PASS |
| N2 | `python3 scripts/research/test-screening-workflow-modes.py` PASS: все 128 сочетаний реальных job conditions взаимоисключены; S1.6 по умолчанию false; deploy/inspection/старый smoke/settlement не исполняются совместно. Сборка закрепляет SOURCE_SHA, binary/PDF/runner manifest; перед оплатой требуется CI этого SHA |
| N3 | `scripts/test-bank-statements.sh` exit0: настоящий закреплённый PDF helper + fake-provider columns19/19, cards20/20, RU23/23 и отрицательные случаи; SQL lifecycle/restore PASS. `git diff --exit-code 29a28a7 HEAD -- internal cmd api web deployments/production/statement-screening.sh scripts/ops-reconcile-statement-screening.py` exit0: Go policy, schema/scorer, финансовый core и старые ops не менялись |
| N4 | `make test-production` повторно exit0; первое выполнение FAIL на загрузке в локальный тестовый S3, причина не установлена, неизменённый повтор PASS. Новый inspector privacy test PASS; `go vet ./cmd/... ./internal/...`, `git diff --check` PASS. `TMPDIR=/var/tmp/grace-quality-next-tests VITEST_MAX_WORKERS=1 make test` exit0: Go, 88 web files / 334 tests, lint/FSD/форматирование/сборка/Jev PASS; [CI точного SHA 39494867634](https://github.com/QahNaaRlN/grace/actions/runs/39494867634) — все 5 jobs PASS, включая полный integration и три image builds |
| N5 | [Единственный run 39496554051](https://github.com/QahNaaRlN/grace/actions/runs/39496554051) выполнен: **FAIL** до первой оценки, `qualification_provider_failed_selected_model`. HTTP200, private correlation сохранён; report0600 и source/profile совпали. Reservation unique/uncertain, sharedUncertainBlock=true. Image/health preserved. Старый smoke, deploy и inspection jobs skipped; повторов и активации нет |

## Проверки приватности и граница изменений

`python3 scripts/research/test-screening-s16-inspection.py`: output содержит только
фиксированные флаги/счётчики; ID, точный расход и произвольный error sentinel не
выводятся. Inspector читает только фиксированный private report и подходящую
reservation плюс общий uncertain flag; ключ, исходный PDF и данные владельца не
читает, SQL-записей не делает. Runtime/image и health проверяются runner до/после.

Локальные журналы: `/tmp/grace-s16-make.log`, `/tmp/grace-s16-bank.log`,
`/tmp/grace-s16-production.log` (исходный FAIL),
`/tmp/grace-s16-production-retry.log` (PASS), `/tmp/grace-s16-ci-watch.log`.

До live run внешних inference не было. В единственном run отправлен первый запрос; calls=0 в отчёте означает отсутствие подтверждённого billing, не нулевой расход. Развёртывание приложения, смена модели/flags,
автоматическая сверка неизвестного расхода, новый corpus и повтор не разрешены.

## Единственный удалённый запуск

Workflow [39496554051](https://github.com/QahNaaRlN/grace/actions/runs/39496554051)
отправлен ровно один раз после PASS exact-source CI. Source `9e89ae0`.
Точные SHA local/remote сверены; локальный dispatch receipt исключает повтор
из этой сессии. 2026-10-20 20:59:30 UTC / 2026-10-21 01:59:30 +05:00: первая columns-ответная
metadata отклонена проверкой `endpoint.Model != in.Policy.Model`. Это не quality
FAIL модели: scorer не запускался, cards и RU не отправлялись. Подтверждены
HTTP200 и private correlation, но actualMicros неизвестен, recordedCalls=0,
scoredCases=0. Серверная inspection подтвердила reportPrivate/expectedSource/
expectedVersion/expectedCandidate=true, reservationUnique/Uncertain=true,
reservationSettled=false, sharedUncertainBlock=true.

Код указывает точный guard текущего запуска. Он не сохраняет фактическую строку
selected endpoint model; нельзя объявлять доказанной замену alias на canonical
или считать установленной историческую причину S1.4. Top-level model/requested,
strategy/attempt/non-BYOK/pipeline прошли предшествующие guards, но provider/count
проверки после selected_model ещё не пройдены. GET generation может помочь
сверить billing, но не восстанавливает отсутствующий chat envelope.

Никакого автоматического settlement, zero-cost предположения или paid retry.
Runtime-флаг/production policy не переключались; shared ledger блокирует новые
платные запросы. Ручной импорт сохраняется. Для восстановления учёта подготовлен
[отдельный контракт](0055-s16-reconciliation.md) без inference и без изменения
transport/alias допуска. Исторический S1.4 report и его reconciliation неизменны.
