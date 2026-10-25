# SPEC-0056 S/V: последняя попытка3из3, Q4.5 — immutable receipt

Accepted [S](0056-evidence-alignment-series.md)/[U](0056-q43-owner-activity-reconciliation-candidate.md)/
[V1–V3](0056-last-attempt-portable-evidence-candidate.md). Владелец отдельно принял V:
«Принять V и выполнить последнюю попытку (рекомендую)». Acceptance V6165d8139e8068bc6d1f5e585abbe236c860f920
до production/ops code. Только последний слот исходной серии3/$2; model/corpus/criteria прежние.

**AUTHORIZED_ONCE**, после accepted dispatch **CONSUMED**, включая preflight/transport отказ.
Receipt immutable. Ни rerun, ни четвёртая попытка, ни новая серия/model этим не открываются.

## Exact integrated source и gates

- Source `0a955148169cd42c04667c5e510c87fb31102591`, integrated [PR132](https://github.com/QahNaaRlN/grace/pull/132).
- Tested PR headc8c241f6453b4d937baee31d6bf8aee6658a5b9c,
  [CI40227597292](https://github.com/QahNaaRlN/grace/actions/runs/40227597292),5/5 PASS.
- [Exact-source mainCI40228543312](https://github.com/QahNaaRlN/grace/actions/runs/40228543312),5/5 PASS перед публикацией/dispatch.
- Local make test (339 Web), make test-production, make test-integration,
  go vet ./cmd/... ./internal/..., git diff --check PASS; tested tree clean.
  V real SQL/owner/conflict/replay/precommit rollback/privacy/policy negatives12/12 PASS/skipped0.
- Build clean main source, Go1.24.0/CGO0/trimpath, vcs.revision=source/vcs.modified=false.
- Manifest SHA256 `9c253c87615961c925d58fb600206ce62b49dddcf7378c4bff0614a45ebc8255`.
- Corpus268d0c2d49ea0803a2b9cffd619448ed9c9304f493d0db4e567a39472a3efa80;
  protocol5b617c066794257e90c289d1ef522cd86af87d26c6cfd713d4ed3c819fc6d58f.
- V portable shape: только S1-added numeric0..1/quotes1..8 bounds сняты с request schema;
  box4/strict/nullable/required сохранены; prompt v3/Go Validate/oracle/scorer/corpus/criteria unchanged.
  Гипотеза compatibility остаётся непроверенной до live результата, PASS не обещается.

## Предыдущие две попытки и conservative accounting

[Journal](../research/statement-quality/evidence-series-2026-10-25.json) consumed2/3,remaining1.
Q4.3/run40204752709 и Q4.4/run40225129622 получили HTTP400/no generation/usage/cost,
INCONCLUSIVE; их original unknown summaries не переписаны. Q4.2 bill$0.005183 вне новой серии.
Q4.3 закрыт0 только по accepted U Activity+policy, directMetadataVerified=false,
contingency30329 сохранён. Q4.4 закрыт0 только по accepted V HTTP400 policy:
[projection](../research/statement-quality/q44-V-settlement.json), VERIFIED_OPERATION,
1SSH/0GET/0POST/1usage row, owner/snapshots unchanged, directMetadataVerified=false.
Private0600 audit SHA2531586b9f8fdb988c06f2bfd6d120f4848b573276953890473b14de4af11236.
Обе операции административные; individual API metadata bill0 не утверждается.
Reconciled spend0, **contingency30329+32023=62352micros ($0.062352)** внутри прежних$2.
Последний campaign maximum285232:0+62352+285232=347584≤2000000.
После accepted dispatch consumed3/3,remaining0 независимо от результата.

## Неизменные runtime limits

google/gemini-3.1-flash-lite / canonical20260507 / google-vertex/eu / Google;
ZDR/deny collection, no retries/fallback/switch. Input275000/output1650000micros/M;
request1000/image1micros/router_ceiling_only. ≤4images/2048px/image4200tokens,
input30000/output16000,HTTP180s,≤8POST/6metadataGET/$0.25call/$2campaign.
Actual request reserves254649, conservative267205,8max285232; caps/rates unchanged.
Cases:cards,columns,cards-photo,full32-native,full32-scan,receipt-discount,
receipt-pdf,receipt-unreadable. Full32по297checks, критерии без изменений.

Один dispatch document-reading-eval.yml через `codex/document-q45-source-0a95514`,
ref exact source, prior dispatch0 проверяется перед публикацией. Inputs authorized_source=source,
approved_manifest=manifest, approval_id=SHA этого опубликованного receipt commit,
execute_approved_reading=true. Все defaultfalse/attempt1/count1/source ancestry/CI5/5/manifest gates сохранены.

≤2pinned root SSH+≤1SCP в одном workflow с прежними secrets/timeouts S.
Новые staging /tmp/grace-document-reading-q-0a955148169cd42c04667c5e510c87fb31102591,
marker /opt/grace/shared/document-reading-spec0056-q5.started,
private report /opt/grace/shared/document-reading-spec0056-q5.private.json; shared lock прежний.
Прежние q/q2/q3/q4 files/audits сохраняются; no-clobber/cleanup/replay,
private raw остаётся VPS, наружу только validated public summary.
После отказа S допускает≤2 fixed-file read-only diagnostic SSH без SQL/key/provider calls.
Только если точный V2 HTTP400/INVALID_ARGUMENT/no generation/choices/usage/cost/noaux
паттерн подтверждён, V допускает≤1 отдельную pinned SSH для единственного unknown reserve Q4.5,
source/manifest/approval из этого receipt, report digest из validated public summary,
private0600 audit/owner/locks/precommit rollback/read-after-write как V2;0новыхmetadataGET.
Иной unknown остаётся unknown без policy0. Освобождённый reserve Q4.5 тоже остаётся contingency в$2.

Стоп на actual PASS_SYNTHETIC8/8 либо после этой единственной попытки/отказа.
No activation/import/deploy; ни один отказ не пересчитывается в PASS.
