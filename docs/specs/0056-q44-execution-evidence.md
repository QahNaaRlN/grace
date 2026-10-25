# Q4.4 / S attempt2: INCONCLUSIVE, серия снова frozen

2026-10-26, Asia/Yekaterinburg (run2026-10-25T19:15:05Z..19:16:01Z).
[Accepted S](0056-evidence-alignment-series.md), [accepted U](0056-q43-owner-activity-reconciliation-candidate.md),
[immutable receipt](0056-q44-owner-receipt.md), [journal](../research/statement-quality/evidence-series-2026-10-25.json).

**INCONCLUSIVE / billing_requires_review**, новый POST вернул HTTP400/
INVALID_ARGUMENT без generation ID/usage/cost. Quality checks0, реальное
распознавание всей восьмёрки не измерено, PASS_SYNTHETIC не получен.
Workflow success означает успешные transport/replay/public projection gates.

| Факт | Evidence |
| --- | --- |
| Run | [40225129622](https://github.com/QahNaaRlN/grace/actions/runs/40225129622),workflow_dispatch/attempt1,completed/success,created19:15:05Z/00:15:05+05,updated19:16:01Z/00:16:01+05 |
| Source |a8752028463d19351c9eddcb0ee40b71fe398b47, integrated PR130; PR head709e8bb/CI40222097044/5, source mainCI40223309245/5 PASS |
| U gates |U acceptance423e7ff до code; tested headd692de9/CI40224022530/5 и local make test339Web/production/integration/vet/diff PASS; PR131 merge200e719f1cf43254e27e37a88201ae311b976d29 до dispatch; model runtime/CLI/wrapper/workflow identical to Q4.4 source |
| Manifest |1639957c4398c15733643b315b7a185c82fc9fdfad2c981517c1cf2ad012f3fc; clean source/Go1.24.0/CGO0/vcs.modified=false; Go1.27.3 manifest parity PASS |
| Approval |1c4e940a90ed1b4586d0dea14d16312a4c060a6b опубликован до dispatch; exact branchref codex/document-q44-source-a875202; dispatch count1/no rerun |
| Counters |1POST/6metadataGET, wire3091ms, POST1262ms; остальные7not_sent/billing_requires_review |
| Billing |unknownBills1,knownCostMicros0 **не доказывает bill0**, reserve32023micros ($0.032023),settlement unknown,overCap=false |
| Series |consumed2/3, remaining1 **BLOCKED** по новому unknown; U закрывал только Q4.3 и не распространяется автоматически на Q4.4; contingency Q4.3 30329micros сохранён |

Скачан только [public summary](../research/statement-quality/q44-40225129622.summary.json),
принят exact-source Go1.24 CLI `-mode=validate-public` с source-bound bundle.
[Run identity](../research/statement-quality/q44-40225129622.run.json).
Private raw/replay остаются VPS; прежние reports/markers не менялись.

## Один разрешённый read-only diagnosis

S разрешает≤2 дополнительных fixed-file SSH после отказа. Выполнен1:
[Reader](../research/statement-quality/q44-wire-reader.py),
[projection](../research/statement-quality/q44-40225129622.diagnosis.json),
[offline tests](../research/statement-quality/q44-wire-reader-test.py)5/5 PASS.
SQL/key/provider calls0, remote persistent writes requested0, snapshots unchanged.
Reader SHA2562ef8afdbff1862969b5975bf3dbd2219662254324145f55832df8fe71ffd98ee.

- Report object21d4d71e413bc3f8631d369db1ed344e7f9fc6f13c210cc58a80f08a2a3b7963.
- Report filecb657a28762be9c9338c5f7871172d8f0a7d91470de22ebbba4d2098b070842d.
- Marker42e4e23a7c70d18cac6dd870b52d540a6041ce3d48e7941d2ab559e449c726df.
- Response641bytes,SHA256c876b2c8e82268919c4060e1384c37f0a94af976a28f6039bc8cdb67dc71c13e.
- HTTP400,transportFailure=false,JSON/error object valid,typed status INVALID_ARGUMENT.
- Closed flag confirms generic “Request contains an invalid argument”; fieldViolationPaths пусты.
- Generation correlation/usage/cost отсутствуют. Private literals/IDs/UUID/key наружу не выведены.

Inline без $defs/$ref не устранил отказ. Нельзя считать конкретный keyword
доказанной причиной: provider не указал поле/constraint. Model/output quality
не оценивались. Следующая compatibility гипотеза — передавать только исходную
shape schema, сохранив prompt v3 и все прежние Go validation/quality criteria.
Это меняет S1 и требует отдельного accepted-контракта; [draft V](0056-last-attempt-portable-evidence-candidate.md).
Автоматического settlement/new POST после нового freeze не выполнено.
