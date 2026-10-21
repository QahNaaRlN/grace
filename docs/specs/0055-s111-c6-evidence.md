# SPEC-0055 / S1.11 C6: единственный live run

Дата: 2026-10-21, Asia/Yekaterinburg. Основание: [accepted S1.11](0055-s111-model-comparison.md)
и отдельное прямое «разрешаю» владельца на [exact decision](0055-s111-c6-run-decision.md).
Approval записан до dispatch в commit `604bf828cbd95d92231247f28f38aa6b0e1bff3a`.
Executable source: `012f604033de9c6a75bf531b8f715323d0fda57d`,
ref `codex/statement-comparison-c6-source`; source не заменялся documentation head.

**Результат: FAIL `qualification_protocol_failed`, до первого score.** Разрешение
`s111-comparison-20261021` израсходовано; повторов нет и они не разрешены.
Ни одна пара не завершила сравнение. Quality, cost efficiency и latency comparison
остаются UNVERIFIED; победитель и production qualification не заявляются.

## Исполнение и наблюдаемое evidence

[Workflow run39587559021](https://github.com/QahNaaRlN/grace/actions/runs/39587559021),
[job119180142751](https://github.com/QahNaaRlN/grace/actions/runs/39587559021/job/119180142751).
Workflow называется `Deploy production`, но активен только job
`S1.11 separately authorized synthetic comparison`; остальные7jobs, включая
Deploy owner VPS и старые screening/inspection/reconciliation modes, skipped.
Dispatch ровно один, source SHA совпал. До него в этой source branch dispatch runs отсутствовали.
Input: `statement_comparison_s111_only=true`, `comparison_authorized_sha` = source выше;
остальные modes false/none, `statement_action=preserve`.

Source CI5/5 и exact bundle checks PASS. В 10:51:06 UTC source gate вывел
`screening_exact_source_ci_all_jobs_passed`; execute стартовал 10:51:18 UTC.
В 10:51:35 UTC / **15:51:35 +05:00** campaign завершился с exit1:
`error=qualification_protocol_failed`, `comparisonPassed=false`,
`activationAuthorized=false`, `reportShapeVerified=true`, `billingKnown=true`.

| Порядок | Candidate / fixture | State | Reason | Score |
| --- | --- | --- | --- | --- |
| 1 | A / columns | attempted | protocol_failed | отсутствует |
| 2 | B / columns | not_sent | campaign_not_started | отсутствует |
| 3 | A / cards | not_sent | campaign_not_started | отсутствует |
| 4 | B / cards | not_sent | campaign_not_started | отсутствует |
| 5 | A / turnover-roles-ru | not_sent | campaign_not_started | отсутствует |
| 6 | B / turnover-roles-ru | not_sent | campaign_not_started | отсутствует |

Для всех6 `falseAcceptance=false`. Это закрытый report field, а не доказательство
качества непросчитанных результатов. A = Gemini 3.1 Flash Lite / Google EU,
B = Mercury 3 / Inception, exact pins и caps неизменны из принятого решения.

## Accounting и сохранность приложения

Read-only inspector в том же job подтвердил:

- `s111InspectionReadOnly=true`, `reportShapeVerified=true`, `reportedPassed=false`;
- `recordedCalls=1`, `completedTuples=0`, `successfulA=0`, `successfulB=0`;
- `billingKnown=true`, `reservationUnique=true`, `reservationSettled=true`;
- `reservationUncertain=false`, `ledgerMatchesReport=true`, `sharedUncertainBlock=false`;
- `activationAuthorized=false`, `error=qualification_protocol_failed`.

Runner вывел `comparison_application_image_and_health_preserved`. Это фактическое
runtime evidence этого запуска, отдельно от прежних offline mocks.
Deployment/model/production policy switch и финансовые записи не выполнялись.
Максимум решения был6POST/$0.2085 provider; фактически recordedCalls=1.
Точная стоимость учтена и сверена inspector с ledger, однако публичный CI намеренно
не выводит сумму и latency. Они **UNVERIFIED для этого отчёта**, не заменены нулём
или cap. Shared uncertain block не установлен.

Private report сохранён операторским runner:
`/opt/grace/shared/statement-comparison-s111-comparison-20261021.json`.
Read-only локальный SSH probe не смог прочитать файл: `Permission denied
(publickey,password)`. Ключи/секреты не читались, server state не менялся.
Публичный closed summary не различает точный отказ canonicalization и merge;
конкретный guard и private trace остаются **UNVERIFIED**. Исправление причины
и новый paid run требуют отдельного ограниченного решения; автоматического retry нет.

## Acceptance criteria и ограничения

| Критерий | Evidence / итог |
| --- | --- |
| C1–C5 | Integrated offline PASS в PR №84; [evidence](0055-s111-evidence.md), [post-merge](0055-pr84-postmerge.md). Этот live FAIL их не заменяет |
| C6 source/gates | Exact source012f604, CI5/5/bundle PASS в job выше; один separately authorized dispatch |
| C6 live comparison | FAIL protocol до score; честный incomplete A0/3, B0/3, remaining5 not_sent. Успешного сравнения нет |
| C6 accounting | Known/settled/unique, report=ledger, uncertain=false, shared block=false; exact amount/latency публично UNVERIFIED |
| C6 application | Image/health preserved; deployment job skipped; activationAuthorized=false |
| C6 one-shot | Один dispatch/recordedCall; marker израсходован, повтор не выполнялся |

Исторические S1.8 FAIL17/19 и S1.9 trace16/19 сохранены. Никаких изменений
production-кода, prompt/schema/scorer/limits/routes, новых DTO или пересчёта старых
FAIL в PASS в этом документационном пакете нет. Следующее полезное действие —
отдельный read-only разбор сохранённого private trace; оно не означает разрешения
новой кампании или смены production-модели.

## Финальная локальная проверка документационного пакета

`TMPDIR=/tmp VITEST_MAX_WORKERS=1 make test` на integration checkout с тем же
executable source012f604: PASS (Go, 88 Web files/334 tests, lint/FSD/format/build,
offline Jev evaluation). Журнал `/tmp/grace-c6-final-make.log`; новых inference нет.
`git diff --check` в docs worktree и integration checkout: PASS.
Относительные Markdown links изменённых документов проверены: PASS.
Primary checkout и pinned source worktree остались чистыми. Код этого PR не меняется;
accepted specification и критерии приведены выше.
