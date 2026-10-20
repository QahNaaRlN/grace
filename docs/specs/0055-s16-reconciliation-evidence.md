# SPEC-0055 / S1.6: восстановление учёта

Дата: 2026-10-21, Asia/Yekaterinburg. [Принятый контракт](0055-s16-reconciliation.md).
Утверждение владельца «Утвердить сверку и восстановление учёта» записано в
`dd7223f` до реализации. Исполняемый коммит оператора:
`500c17a504d64c81ab7381dea473d1f1db0ba406`; [draft PR №79](https://github.com/QahNaaRlN/grace/pull/79).

| ID | Исполняемое подтверждение |
| --- | --- |
| R1 | `python3 scripts/research/test-screening-s16-reconciliation.py` PASS: match/mismatch, неизвестный/нечисловой/отрицательный расход, ceil/ceiling, точная alias/canonical пара, профиль, ID/UUID/status/marker/time, приватность 0600 и запрет symlink. Полный main с подменёнными GET/SQL: apply без inspect запрещён; inspect не пишет SQL; apply пишет один раз; replay не пишет; конфликт отвергается; report/marker неизменны |
| R2 | [Read-only run 39498418859](https://github.com/QahNaaRlN/grace/actions/runs/39498418859) SUCCESS, 2026-10-20 21:12:21 UTC / 2026-10-21 02:12:21 +05:00. metadataMatched/unknownBillingVerified=true; private audit сохранён; reservation остаётся uncertain. Generation и каталог получены GET, без inference |
| R3 | [Apply run 39498531555](https://github.com/QahNaaRlN/grace/actions/runs/39498531555) SUCCESS, 2026-10-20 21:13:09 UTC / 2026-10-21 02:13:09 +05:00. Повторно проверены metadata/report/row и прежний inspect audit. Existing owner-scoped grace_ai_settle под ledger/row locks; after-COMMIT row exact/settled. Независимое чтение: reservationUnique=true, reservationSettled=true, reservationUncertain=false, sharedUncertainBlock=false. Конфликтный replay покрыт offline. [Read-only replay 39498648789](https://github.com/QahNaaRlN/grace/actions/runs/39498648789) SUCCESS, 21:14:00 UTC / 02:14:00 +05:00: idempotentReplay=true, SQL settlement повторно не исполняется |
| R4 | `make test-production` exit0; workflow isolation — все 512 сочетаний PASS, inspector privacy PASS, `git diff --check` PASS. `TMPDIR=/var/tmp/grace-quality-next-tests VITEST_MAX_WORKERS=1 make test` exit0: Go, 88 web files / 334 tests, lint/FSD/форматирование/сборка/Jev PASS |

## Приватность и сохранённые ограничения

ID берётся только из фиксированного private `failure.unverifiedCorrelationId`.
Ключ читается в память сервера из существующего secret; нового секрета нет.
GET generation/catalog ограничены размером/временем, redirects запрещены.
Расход округлён вверх в integer USD micros и проверен по потолку одного вызова.
ID, точная стоимость, owner и usage UUID публично не выводятся.

Private 0600 audit: `/opt/grace/shared/statement-reconciliation-s16-gemini31-diag-20261021.json`:
report digest/name, catalog digest и точная mapping, metadata, usage before/after,
verified/committed timestamps. Исходный report и `.started` не перезаписываются;
исторические S1.4 report/audit не затрагиваются. Временный operator script
удалён workflow EXIT trap (шаг завершён успешно).

Это восстановление metering, а не положительный результат smoke.
Исходный report по-прежнему содержит actualMicros=null и passed=false; расход
доказан отдельным reconciliation audit и settled ledger. FAIL `selected_model`,
consumed authorization и quality UNVERIFIED сохраняются. Cards/RU не отправлены.
Значение выбранного endpoint model в исходном ответе не сохранено; billing
metadata не доказывает причину этого transport failure.

Во всех runs платные smoke и deploy jobs skipped. Новых inference=0,
смены production policy/model/flags, allowance/fees, deployment и правок финансовых
данных не было. Проверка `git diff --exit-code 29a28a7 500c17a -- internal cmd api web deployments/production/statement-screening.sh scripts/ops-reconcile-statement-screening.py`
exit0. Основной checkout clean; работа остаётся в feature-worktree/PR.

После live run добавлена offline-регрессия длинного Decimal: точный ceil без потери младших знаков; чрезмерная отрицательная экспонента отвергается. Повторная операция на сервере не нужна и не выполнялась.

Локальные журналы: `/tmp/grace-s16-recovery-inspect.log`,
`/tmp/grace-s16-recovery-apply.log`, `/tmp/grace-s16-recovery-replay.log`, `/tmp/grace-s16-recovery-test.log`,
`/tmp/grace-s16-recovery-production.log`.
Следующий отдельный scope: совместимость `selected_model`; этот контракт
не разрешает изменение transport, новый smoke или adoption модели.
