# Q4.5 / V last attempt3: REJECT по качеству, серия закрыта

2026-10-26 Asia/Yekaterinburg; run2026-10-25T20:10:07Z..20:11:05Z
(01:10:07..01:11:05+05). Accepted [S](0056-evidence-alignment-series.md)/
[V](0056-last-attempt-portable-evidence-candidate.md), [immutable receipt](0056-q45-owner-receipt.md).

**REJECT/quality_reject**, protocol PASS/HTTP200. Portable schema устранила
наблюдавшийся400 на этой попытке. Это подтверждает работу V protocol path,
но не устанавливает единственный unsupported keyword прежнего provider compiler.
PASS_SYNTHETIC не получен; workflow success подтверждает transport/replay/public validation gates.

| Факт | Evidence |
| --- | --- |
| Run | [40229482580](https://github.com/QahNaaRlN/grace/actions/runs/40229482580),workflow_dispatch,attempt1,count1,completed/success |
| Source |0a955148169cd42c04667c5e510c87fb31102591,integrated PR132; tested headc8c241f6453b4d937baee31d6bf8aee6658a5b9c/CI40227597292/5 PASS,exact-source mainCI40228543312/5 PASS |
| Local gates |make test339Web/test-production/test-integration/vet/diff PASS; V SQL12/12/skipped0 PASS до ops, worktree clean; clean source Go1.24.0/CGO0/vcs.modified=false build |
| Approval |e2e7daabf1ca002ad3b80ef6ed96782acb139c89 опубликован до dispatch; source branch codex/document-q45-source-0a95514,prior dispatch0 |
| Manifest |9c253c87615961c925d58fb600206ce62b49dddcf7378c4bff0614a45ebc8255; corpus268d0c2d49ea0803a2b9cffd619448ed9c9304f493d0db4e567a39472a3efa80/protocol5b617c066794257e90c289d1ef522cd86af87d26c6cfd713d4ed3c819fc6d58f |
| Outcome |cards rejected,27checks/6fail,recognition=false,nativeBinding=fail; остальные7not_sent/decisive_quality_reject |
| Calls/cost |1POST/6metadataGET,wire6761ms/POST5880ms,5631input/2117output,known actual5042micros=$0.005042,settled,unknownBills0,overCap=false |
| Series |consumed3/3,remaining0,statusclosed_exhausted_quality_reject; no fourth/retry/rerun/new model/series |

Скачан только [public summary](../research/statement-quality/q45-40229482580.summary.json),
принят exact-source Go1.24 CLI `-mode=validate-public` с одобренным bundle.
[Run identity](../research/statement-quality/q45-40229482580.run.json).
Private raw/replay остаются VPS, прежние files/audits не переписаны.

## Разбор REJECT без новых model calls

| Проверка | Результат |
| --- | --- |
| 8header checks + rowCount |9/9 PASS |
| Row1 |operationDate/bookingDate FAIL; остальные7/9 PASS |
| Row2 |operationDate/bookingDate/description/issues FAIL; остальные5/9 PASS |
| Native source binding |fail, отдельно от recognition criteria |

Один разрешённый S fixed-file read-only SSH, [reader](../research/statement-quality/q45-quality-reader.py),
[projection](../research/statement-quality/q45-40229482580.diagnosis.json),
[offline privacy/snapshot tests](../research/statement-quality/q45-quality-reader-test.py)5/5 PASS.
SQL/key/provider calls0,remote persistent writes requested0,snapshots unchanged.
Reader SHA256cfceb50943feac723f3e1f7b78263f1f3f47375a7259f5ceb3139a9633c48965.
Report object48404d81a3f1c373db0d6b40f31f5b59a4aca3585c4b0fda551aa0cd405d4d3f;
report file8f3e1baf5ce884037ca82f1624908c932e3b9dcaa43346949c7123347d514667;
markerd08091939aa81e0f771ea0b7ae55992511d41b5ea6878ae10617a1adf6140c13;
responseac2aa6513ebdc5547d02c0c60827afd033eccea6057eb1d1e1ae742160021f65.

Во всех4 date fields model value содержит правильную calendar date, но **не содержит clock**;
совпадение hour/minute отсутствует, added seconds нет. Полные ожидаемые printed date+clock
также не найдены как цельная нормализованная строка в соответствующих evidence quotes.
Это потеря напечатанной временной точности, а не неверный календарный день.
Row1 description полный/правильный, issues=[]; row2 description=null и issues nonempty,
в закрытой vocabulary найден missing. Public golden row2 содержит untrusted instruction literal,
который должен сохраняться как source data; почему модель выбрала null, по этим данным не установлено.
Нельзя утверждать ни исполнение этой инструкции, ни доказанную safety-фильтрацию.
Ни рескейл/дописанное время, ни заполнение description из golden, ни ослабление scorer не применялись.

## Закрытый accounting и границы

[Journal](../research/statement-quality/evidence-series-2026-10-25.json):
known/reconciled spend5042micros,unresolved bills0,heldReserve0;
Q4.3/Q4.4 policy contingency62352micros сохранён,administrative basis отдельно от metadata.
Итого conservative accounted67394micros=$0.067394≤$2. Q4.2 bill вне новой серии не переучитывался.
Q4.5 settled обычным usage path, V conditional HTTP400 policy0 не применялся.
Оригинальные неизвестные счета в прошлых summaries сохранены с отдельными U/V reconciliation.

Серия исчерпана3/3, **remaining0/resumeAllowed=false**. Остальные7 документов/full32/receipt quality
в live не измерены, UNVERIFIED. Новая серия/model/repair/retry/rerun/activation/import/deploy не открываются.
