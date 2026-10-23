# S1.17 E6 — шесть ответов canonicalization PASS, качество FAIL

- Accepted contract: [S1.17 E6](0055-s117-alias-comparison.md).
- [Разрешение и immutable source](0055-s117-e6-execution-receipt.md) записаны
  и опубликованы коммитом `734daf1eb88ad5433a746fbe65f1fc87b595a611` до dispatch.
- Exact source `24f73223b9b3370ca78c41d34e89edca9a276488`, frozen ref
  `codex/statement-s117-e6-source`; интегрирован №101, source CI39924326517 SUCCESS5/5.
- Один [workflow_dispatch39925874197](https://github.com/QahNaaRlN/grace/actions/runs/39925874197),
  создан 2026-10-23 08:36:44 UTC / 13:36:44 +05:00;
  [job120321991496](https://github.com/QahNaaRlN/grace/actions/runs/39925874197/job/120321991496),
  08:36:48–08:37:48 UTC / 13:36:48–13:37:48 +05:00.
- Итог: **EXECUTED / FAIL `qualification_quality_failed`; authorization CONSUMED.**
  Не preflight refusal и не protocol failure. Повторного dispatch/POST нет.
- Выполнен только job S1.17; остальные11 jobs skipped, в том числе deployment,
  старые comparisons, diagnostics, inspections и reconciliation.

## Наблюдаемый результат

Все6 tuples completed, каждая прошла canonicalization и имеет private capture.
Closed replay matched6/6, дополнительных provider calls при replay0.
Качественную приёмку не прошла ни одна пара: successfulA0/3, successfulB0/3,
falseAcceptances0 у обоих. Ни одна модель не квалифицирована для production.

| Порядок | Candidate | Public fixture | Quality checks | Complete latency, ms | Closed codes |
| --- | --- | --- | --- | --- | --- |
| 1 | A | columns | FAIL17/19 | 2235 | coverage, blockers; source_coverage_incomplete |
| 2 | B | columns | FAIL17/19 | 3847 | coverage, blockers; source_coverage_incomplete |
| 3 | A | cards | FAIL14/20 | 2257 | operation_clock, booking_clock, coverage, blockers; source_coverage_incomplete |
| 4 | B | cards | FAIL18/20 | 1875 | coverage, blockers; unknown validation, source_coverage_incomplete |
| 5 | A | turnover-roles-ru | FAIL17/23 | 1916 | row_issues, printed_debit_turnover, printed_credit_turnover, turnover, blockers; unknown validation |
| 6 | B | turnover-roles-ru | FAIL19/23 | 1359 | row_issues, turnover, blockers; unknown validation |

[Точная закрытая проекция Go summary + inspector](0055-s117-e6-public-summary.json)
сохраняет arrays, их порядок и повторения codes. `unknown` оставлен `unknown`:
из публичной проекции нельзя придумывать точные private validation reasons.
Raw proposals/service/source IDs и личные финансовые данные не публикуются.

Complete latency включает proof/network/decode; это не pure generation.
При n3 для каждой модели winner/p95 и общая пригодность для банков не заявляются.
Observed canonicalization6/6 не доказывает всеобщую schema compliance или качество
за пределами этих шести ответов. Схема, scorer и финансовые guards не менялись.

## Расход и accounting

| Candidate | Attempts/completed | Успешных | Provider USD micros | USD provider | Cost per success |
| --- | --- | --- | --- | --- | --- |
| A | 3/3 | 0 | 8524 | $0.008524 | undefined |
| B | 3/3 | 0 | 2373 | $0.002373 | undefined |
| Итого | 6/6 | 0 | **10897** | **$0.010897** | undefined |

Billing known; exact unique reservation из verified private report совпала с
единственной ledger row по UUID/model/provider/purpose/time/amount.
Reservation settled, reservationUncertain=false, ledgerMatchesReport=true,
sharedUncertainBlock=false. Known cost не обнулён из-за quality FAIL.
Расход ниже утверждённого208500 USD micros; provider cost не объявляется
fee-inclusive итогом. Remaining/allowance не раскрывались и не изменялись.

## Сохранность и границы

Strict Go summary и новый inspector подтвердили reportShapeVerified,
observationsVerified; snapshots captured6/6, replay matched6/6.
Marker/report0600 проверены через private_read; bytes/mode/mtime совпали
до/после read-only inspection (artifactsUnchanged=true).
Runner подтвердил `comparison_application_image_and_health_preserved`:
actual container/image совпали до/после, readiness health успешен.

Новые report/marker остаются в fixed private storage `/opt/grace/shared`.
Старые P6/D5 artifacts не читались; private/free text/UUID/generation/provider IDs
не попадали в публичные evidence. Новые финансовые записи, модель/flags,
configuration/budget/guards, deployment/reconciliation не менялись.
Исторические permissions не переиспользовались. Отдельный E6 approval израсходован;
новые inference, retry, repair или эксперимент этим результатом не открыты.

## Исполняемая проверка evidence

Exact source/ref и CI5/5 проверены до dispatch; `git diff --quiet` подтвердил
совпадение source tree и merge9cf0d03. Run metadata подтвердили тот же SHA/ref,
event workflow_dispatch, ровно один выполненный job и11 skipped.
JSON-проекции взяты только из публичных closed stdout, сверены между собой:
6completed,0successful,10897micros, settled/unique/no uncertain, matched6/6,
artifactsUnchanged и отсутствие private fields/UUID — PASS.
Source worktree/ref остались clean и на24f7322 после run.

`TMPDIR=/tmp VITEST_MAX_WORKERS=1 make test` — exit0 в evidence worktree:
Go,88 Web файлов/334 теста, ESLint/FSD/Prettier/build и offline Jev.
`git diff --check` и документные ссылки — PASS.
Это offline проверка документационного пакета; реальный E6 остаётся quality FAIL.
Logs: `/tmp/grace-s117-e6-make.log`, `/tmp/grace-s117-e6-job.log`;
в Git добавлена только принятая closed projection, не private artifact.

Возможный следующий пакет — отдельная read-only диагностика coverage/clocks/
turnover по сохранённому E6. Это предложение для отдельного контракта; сейчас
она не выполнялась, prompt/guards не ремонтировались, новые POST не разрешены.
