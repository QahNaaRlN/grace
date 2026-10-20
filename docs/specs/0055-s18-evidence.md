# SPEC-0055 / S1.8: подтверждения реализации и единственного smoke

Дата: 2026-10-21, Asia/Yekaterinburg. [Принятый контракт](0055-s18-identity-screening.md).
Acceptance `819a20e` предшествует executable source
`f450e9d629a4543821a21b8f7b3aa679788872c4`.
[PR №80](https://github.com/QahNaaRlN/grace/pull/80).

| ID | Исполняемое подтверждение |
| --- | --- |
| S1 | `sh scripts/test-statement-screening-s18.sh` PASS: preflight отказ не расходует marker; failure и success расходуют его один раз; repeat/other SHA/orphan report/dangling report и marker symlink не делают новый вызов. Старые mock S1.4/S1.6 artifacts проверены SHA-256 и сохранены, report0600 и отсутствие private cost в stdout подтверждены |
| S2 | `python3 scripts/research/test-screening-workflow-modes.py` PASS, 1024 сочетания фактических job conditions. S1.8 default=false, конфликты не запускают платный или deploy job. `test-screening-source-ci.py` PASS: exact SHA, ровно пять completed/success jobs, missing/duplicate/other/skipped/failed/pending rejected. Отдельный CLI run с Python -O подтверждает, что runtime gate не зависит от assert. В workflow связаны оба gates: source CI и post-merge №79; binary/PDF/runner/SOURCE_SHA связаны manifest |
| S3 | `python3 scripts/research/test-screening-s18-inspection.py` PASS: report v3/pin/profile, catalog proof/time/classes, per-call cap и aggregate cost, exact UUID/marker/owner/reservation; malformed/wrong/missing/conflicting данные отвергаются. READ ONLY transactions содержат только SELECT, files не меняются; no ID/owner/cost/text output, chmod644 и symlink rejected. Python inspector проверяет структуру, полная qualification/scorer/digest проверяется существующим Go screening-summary |
| S4 | `TMPDIR=/var/tmp/grace-quality-next-tests VITEST_MAX_WORKERS=1 make test` exit0: Go, 88 файлов/334 Web tests, lint/FSD/форматирование/сборка/Jev PASS. `make test-production`, `scripts/test-bank-statements.sh` exit0, реальный helper + fake-provider columns19/19, cards20/20, RU23/23, negatives и SQL lifecycle/restore PASS. `go vet ./cmd/... ./internal/...`, `git diff --check` PASS. [CI exact source 39504592172](https://github.com/QahNaaRlN/grace/actions/runs/39504592172) SUCCESS, все 5/5 jobs, включая полный PostgreSQL integration и три image builds |
| S5 | [Единственный live run 39505958576](https://github.com/QahNaaRlN/grace/actions/runs/39505958576) выполнен один раз после обоих CI gates: **FAIL qualification_quality_failed**, columns17/19, codes record_description/blockers; cards/RU не отправлялись. Один verified call с pinned_canonical и один scored case. Billing known, row unique/settled, ledgerMatchesReport=true, sharedUncertainBlock=false. Image/container и health preserved; все прочие smoke/inspect/reconcile/deploy jobs skipped |

## Границы

`git diff --exit-code 938b370 f450e9d -- internal cmd api migrations web deployments/production/statement-screening-s16.sh scripts/research/inspect-statement-screening-s16.py scripts/ops-reconcile-statement-screening-s16.py` exit0.
Go transport/schema/scorer/planner/цены/лимиты, Web/API, миграции и старые
операторы неизменны. Новая реализация ограничена operator/workflow/inspector/tests.
После перехода с f75dd00 на f450e9d усилен только CI helper: явный отказ вместо
Python assert; прежний CI отменён новым head, paid inference не было.

Post-merge №79: [CI main 39502669662](https://github.com/QahNaaRlN/grace/actions/runs/39502669662)
5/5 PASS; Delivery39503724048 SUCCESS; локальные gates PASS, дерево PR/merge
совпадает. Публикация образов не является деплоем.

Local logs: `/tmp/grace-s18-make.log`, `/tmp/grace-s18-production.log`,
`/tmp/grace-s18-bank.log`, `/tmp/grace-s18-vet.log`,
`/tmp/grace-s18-ci-gate-cli.log`, `/tmp/grace-s18-ci-watch.log`.
Полный corpus/adoption остаются отдельными этапами. PASS трёх synthetic layouts
не доказывает качество на личной выписке. Модель/flags/config/image не переключаются.


## Единственный live run

Workflow dispatch создан **2026-10-21 03:09:06 +05:00** / 2026-10-20 22:09:06 UTC.
Результат **03:10:01 +05:00**, независимое чтение report/ledger **03:10:03 +05:00**.
SHA `f450e9d629a4543821a21b8f7b3aa679788872c4`, source marker и private0600
подтверждены. До dispatch проверены чистота local worktree и remote head;
локальный exclusive receipt исключает второй dispatch этой сессии.

Go summary: `screeningPassed=false`, `qualification_quality_failed`;
columns checks19/correct17, codes `record_description`, `blockers`, coverage=[].
Validation содержит два `unknown`: это закрытая публичная классификация причин
проверки, **не неизвестный billing**. Точный дефект описания/validation ещё
не разобран; нельзя объявлять причиной пропущенный дефис или ослаблять scorer
на основе одного общего кода. Полные trace/reasons остаются в private report.

Read-only inspector: expectedSource/version/candidate/profile=true,
billingKnown=true, recordedCalls=1/scoredCases=1; callProofsVerified и
callCostsVerified=true, selectedModelClasses=[pinned_canonical].
reservationUnique/Settled/ledgerMatchesReport=true, reservationUncertain=false,
sharedUncertainBlock=false. `reportedPassed=false`; diagnosticPresent=false,
потому что отказ на scorer, а не provider guard. ID известного billed call
сохранён приватно; отсутствие failure correlation не означает утрату billing.

Это подтверждение совместимости exact canonical **в текущем S1.8**, а не
восстановление утраченной строки ответа S1.6. Качество columns — FAIL17/19;
cards/RU live UNVERIFIED (не запускались), полный корпус/adoption не пройдены.
Whole-run authorization consumed, неиспользованные вызовы не переносятся.
Никаких retry, auto-settlement неизвестного, нового inference, model/flag switch
или deployment после первой попытки нет. Известный расход уже учтён; блокировки
uncertain нет на момент независимого чтения. Ручной импорт не затронут.

Private report/marker: `/opt/grace/shared/statement-screening-s18-gemini31-identity-20261021.json`
и `.started`, исходники не переписываются. Сумма/IDs/owner в публичную документацию
не включены. Журналы: `/tmp/grace-s18-live.log`, `/tmp/grace-s18-live-watch.log`,
`/tmp/grace-s18-ci-ready.json`, `/tmp/grace-s18-dispatch-attempt.json`.
Следующий шаг — бесплатный разбор сохранённого синтетического trace/reasons;
изменения prompt/scorer/transport и новый платный запуск требуют отдельного
контракта. S1.8 таких исправлений не разрешает.

## Интеграция

PR №80 слит владельцем 2026-10-21 03:24:55 +05:00, `5e9116e`.
[Post-merge evidence](0055-pr80-postmerge.md): local make test/vet/diff PASS,
main CI39507934933 все5/5 PASS. Delivery39508797773 SUCCESS.
Implementation integrated; live FAIL17/19 сохранён. Новых inference/деплоя нет.
