# SPEC-0056 V: implementation и AC evidence

Accepted [V1–V3](0056-last-attempt-portable-evidence-candidate.md), human2026-10-26,
acceptance commit6165d8139e8068bc6d1f5e585abbe236c860f920 **до production/ops кода**.

| AC | Реализация и executable evidence |
| --- | --- |
| V1 shape/semantic separation | VisualSchema удаляет только S1 box numeric0..1 и quote1..8 schema bounds; box4/required/nullable/strict сохранены. DocumentPrompt v3, Go Validate/oracle/scorer/corpus unchanged. `go test ./internal/statementdata ./internal/assistant ./internal/documentqualification ./cmd/document-reading-eval`: causal actual grouped wire, fractional/null positive и integer/empty evidence negative controls. Actual reserve254649, conservative267205,8max285232 прежние. |
| V1 новая identity | Wrapper фиксирует q5; workflow Q4.5 defaultfalse/attempt1/count1/no-clobber. `python3 -B scripts/research/test-document-reading-workflow.py` сохраняет q/q2/q3/q4 files. |
| V2 bounded policy operation | `./scripts/research/test-q44-policy-settlement.sh`12/12 PASS/skipped0 до ops: real PostgreSQL commit/rollback/owner/conflict/replay/neighbor/budget checks, private audit/public privacy, policy negatives. `q44-policy-settle.py` фиксирует source/manifest/approval/report/file/body hashes и32023reserve/window; sealed unique reserve UUID only, one row/owner from row/locks/precommit rollback/read-after-write. No HTTP/key/financial SQL/functions/migrations/config changes. |
| V2 actual Q4.4 | [Projection](../research/statement-quality/q44-V-settlement.json) VERIFIED_OPERATION,1SSH/0GET/0POST,1row/owner/snapshots unchanged; private0600 audit2531586b9f8fdb988c06f2bfd6d120f4848b573276953890473b14de4af11236. Administrative actual0, basis owner_accepted_http400_policy/directMetadataVerified=false. Original summary unknown preserved, contingency30329+32023=62352 in original$2. |
| V3 last-slot boundary before dispatch | [Journal](../research/statement-quality/evidence-series-2026-10-25.json) consumed2/3,remaining1. Last maximum285232+contingency62352=347584≤2000000. New integrated source/mainCI/manifest/published immutable receipt required before one Q4.5 dispatch. After dispatch remaining0 regardless result; no retry/fourth/new series/model/activation/import/deploy. |

Full local make test (339 Web)/test-production/test-integration/vet/diff PASS.
Tested headc8c241f6453b4d937baee31d6bf8aee6658a5b9c, PR CI40227597292/5 PASS;
merged PR132/main0a955148169cd42c04667c5e510c87fb31102591; exact-source mainCI40228543312/5 PASS.
Clean source Go1.24.0/CGO0 build/vcs.modified=false, new manifest9c253c87615961c925d58fb600206ce62b49dddcf7378c4bff0614a45ebc8255.
[Immutable Q4.5 receipt](0056-q45-owner-receipt.md) publishes the last-slot grant before dispatch.
Q4.4 outcome INCONCLUSIVE is preserved; this repair does not claim quality PASS.

## Последний live Q4.5: V path завершён, quality PASS не получен

[Immutable receipt](0056-q45-owner-receipt.md) commit e2e7daabf1ca002ad3b80ef6ed96782acb139c89
опубликован после mainCI5/5 до единственного dispatch run40229482580/attempt1.
[Execution evidence](0056-q45-execution-evidence.md): HTTP200/protocol PASS,
REJECT/quality_reject,1POST/6GET,27checks/6fail,nativeBinding fail,$0.005042 settled,
unknown0. V compatibility trial прошёл protocol, но model quality не прошла.
S read-only diagnosis5/5 controls PASS,1SSH/0SQL/key/provider/writes,snapshots unchanged.
Серия закрыта3/3,remaining0,contingency62352+known5042=67394≤2000000.
Q4.5 conditional policy0 не применялся: это200 с известным bill.
Новый repair/retry/fourth/new series/model/activation/import/deploy не открыт.
