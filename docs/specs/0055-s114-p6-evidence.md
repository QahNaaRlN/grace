# S1.14 / P6: live evidence — FAIL protocol, exact guard replayed

Дата: 2026-10-21, Asia/Yekaterinburg.
[Принятое решение P6](0055-s114-p6-run-decision.md): владелец ответил «+» на
конкретные source/caps/preflight/no-retry условия. Acceptance commit
`1194a234f519fba03019074db5c220b4beb8a772` записан и опубликован **до dispatch**.

Source: `301afd6bedc4a19559e731471336bbc4e2f74075`, frozen ref
`codex/statement-observed-comparison`, unchanged после run; integrated ancestry
и exact-source [CI39612267914](https://github.com/QahNaaRlN/grace/actions/runs/39612267914)5/5
проверены до отправки и повторно workflow gate.

**Результат P6: FAIL `qualification_protocol_failed`, 3POST из максимум6.**
Campaign/dispatch authorization `s114-comparison-20261021` израсходована;
retry, новая campaign, model/route/guard/prompt repair не разрешены.

- [Workflow39621334415](https://github.com/QahNaaRlN/grace/actions/runs/39621334415)
  final failure; source301afd6, event workflow_dispatch, создан14:56:50 UTC.
- [S1.14 job119295262027](https://github.com/QahNaaRlN/grace/actions/runs/39621334415/job/119295262027)
  FAIL. Exact source/CI, operator build/manifest и pinned SSH stages SUCCESS.
- Execute stage14:57:22–14:57:51 UTC /19:57:22–19:57:51 +05:00; это окно stage,
  не вымышленный exact campaign start/finish из private report.
- Все9 соседних jobs SKIPPED, включая Deploy owner VPS, old C6, reconcile,
  screening/diagnosis/inspect. Повторного dispatch или rerun не было.

## Результаты по tuple

| Порядок | Tuple | Результат | Observation/capture | Replay | Complete latency |
| --- | --- | --- | --- | --- | --- |
| 1 | A/columns | quality FAIL16/19, row_2:description_evidence_gap; codes record_description/row_issues/blockers | canonicalization_passed / captured | matched | 2325ms |
| 2 | B/columns | quality FAIL17/19, source_coverage_incomplete; codes coverage/blockers | canonicalization_passed / captured | matched | 4037ms |
| 3 | A/cards | protocol FAIL до scorer: unknown_alias, scope row, field operationDate, proposal ordinal1, selectionIndex1, role empty | canonicalization_failed / captured | matched с тем же diagnostic | 2264ms |
| 4 | B/cards | not_sent / campaign_not_started | not_reached / not_reached | unavailable_capture | unavailable |
| 5 | A/turnover-roles-ru | not_sent / campaign_not_started | not_reached / not_reached | unavailable_capture | unavailable |
| 6 | B/turnover-roles-ru | not_sent / campaign_not_started | not_reached / not_reached | unavailable_capture | unavailable |

A — Gemini3.1 Flash Lite/Google EU, B — Mercury3/Inception, exact policies прежние.
Known quality FAIL первых двух tuples позволил продолжить заранее разрешённый
порядок; protocol FAIL третьего остановил всю campaign, без отдельного B/cards.
`unknown_alias` означает ссылку на alias вне допустимого plan; raw alias не
публикуется. Ordinal1 относится к первой строке returned Proposal, не к номеру
строки PDF. Snapshot/closed diagnostic позволили точно воспроизвести этот guard.
Это первый reported guard в детерминированном порядке, не доказательство отсутствия
других дефектов в непрошедшем proposal.

Обе пары остаются **quality UNVERIFIED / unqualified**: A completed1 из3,
B completed1 из3; successfulA=0, successfulB=0, recorded falseAcceptances=0.
Это не PASS полноты/качества; полного сравнения или winner нет. Никакого p95.
Complete latency включает fresh proof/network/decode; не pure generation.

## Recorded cost и accounting

| Candidate | Recorded calls | Known cost USD micros | USD | Latencies |
| --- | --- | --- | --- | --- |
| A | 2 | 4203 | $0.004203 | 2325,2264ms |
| B | 1 | 1117 | $0.001117 | 4037ms |
| Total | 3 | **5320** | **$0.00532** | n=3, без p95 |

Это public closed numeric projection metered provider cost, не утверждение о
fee-inclusive invoice. Per-tuple split A cost в public summary отсутствует и не
восстанавливается догадкой. Actual<=208500 provider micros; прежний shared budget,
fees и historical expenses не сбрасывались/не увеличивались.

Inspector подтвердил:

- billingKnown=true, reservationUnique=true, reservationSettled=true;
- ledgerMatchesReport=true, reservationUncertain=false, sharedUncertainBlock=false;
- exact unique reservation UUID коррелирован внутри private inspector, не опубликован;
- reportedPassed=false, recordedCalls3, completedTuples2, error qualification_protocol_failed.

Actual numeric remaining shared budget публично не получен и не угадывается.
Known billing не превращается в0 из-за quality/protocol FAIL; reconciliation и
новых generation metadata GET не выполнялось.

## Snapshot, replay, privacy и runtime

`reportShapeVerified=true`, `observationsVerified=true`, оба digest/context
проверены exact-source Go binary; snapshots captured для3 отправленных tuples,
включая pre-canonicalization failure A/cards. Snapshot bytes остались private.
Replay providerCalls0: все3 captured matched;3 not_sent unavailable_capture.
`artifactsUnchanged=true` подтверждает report/marker bytes/mode/mtime до/после
read-only inspection. Marker/report fixed identity присутствуют; no-clobber и
one-shot разрешение сохранены после FAIL.

Log содержит `comparison_application_image_and_health_preserved`: container/image
identity совпали до/после, readiness до/после успешен. Deployment/model/policy/
flags/financial writes не выполнялись. SSH/VPS и fresh metadata использовались
только внутри разрешённого workflow/preflight; новая ledger reservation/settlement
— разрешённый metering, не запись финансового импорта.

Public Actions/Markdown не получают alias strings, proposal/service text,
reservation/owner/generation IDs или keys. Использованы только closed summaries.
Private artifact не скачивался и не копировался в Actions artifacts. Старый C6
остаётся FAIL/consumed/exact guard UNVERIFIED; новый guard не объясняет утраченный
исторический ответ и не исправляет его.

## Локальная проверка evidence и границы

Closed outputs отдельно сверены:3 recorded calls,2 completed, sum4203+1117=5320,
3 matched replay, settled/unique/ledger match, no uncertain block и unchanged files.
Локальная копия только public summaries: `/tmp/grace-s114-p6-public-summary.json`;
private response/report не читался вне разрешённого fixed inspector/replay.
`TMPDIR=/tmp VITEST_MAX_WORKERS=1 make test` документационного worktree PASS
(`/tmp/grace-s114-p6-evidence-make.log`); links и `git diff --check` PASS.
CI документационного PR фиксируется отдельно; source301afd6 CI5/5 уже проверен.
Execution result FAIL не переименовывается в PASS из-за успешного replay/accounting.

## Следующее решение

P6 execution завершён; permission consumed. Автоматических новых calls, recovery,
fix, prompt/scorer change, model switch или deployment нет. Возможный следующий
пакет — отдельно согласованный offline разбор сохранённых3 proposals и двух
quality failures; его scope/acceptance здесь не открываются. Main push-CI/Delivery
limitation из [post-merge record](0055-pr91-postmerge.md) сохранена отдельно;
она не меняет verified exact source301afd6 текущего run.
