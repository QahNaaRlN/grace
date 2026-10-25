# SPEC-0056 S: implementation evidence

2026-10-25. [Accepted S](0056-evidence-alignment-series.md), acceptance до кода
`eda80af` по прямым решениям «согласуй…» и «До3 попыток / $2 суммарно».
Production repair ограничен VisualSchema и DocumentPrompt/version; Go validator,
scorer, goldens/corpus, model/rates, core/API/web не изменены. Runtime quality пока
не заявляется. Каждая новая попытка получит exact-source receipt и свой результат.

| AC | Исполняемое evidence |
| --- | --- |
| S1 | `go test ./internal/statementdata -run TestSEvidenceSchemaClosesObservedProtocolGaps` PASS: actual schema0..1/quotes1..8/null; обе независимые invalid classes остаются rejected unchanged validator, fractional/null control PASS. `go test ./internal/assistant -run TestSActualGroupedWireCarriesEvidenceContract` PASS: настоящий grouped wire содержит именно эту schema, strict=true, prompt v2, pinned provider/no fallback. |
| S2 | `go test ./internal/assistant ./internal/documentqualification ./cmd/document-reading-eval -count=1` PASS: fake8/replay/negative/forgery/billing/stop guards сохранены, full32по297checks. Actual DocumentRequest reserves247349micros; campaign preflight267205,8max285232,3*8max855696<2000000. |
| S3 | `python3 -B scripts/research/test-document-reading-workflow.py` PASS: q3 no-clobber/shared lock/source/attempt1/manifest/CI gates/cleanup, старые q/q2 marker/report неизменны;0 real SSH/provider calls. |
| S4 | Local make test PASS (339/339Web), go vet PASS, git diff --check PASS; exact-head/main CI фиксируются перед запуском. Remote make test-production/test-integration обязательны. |
| S5 | [Series journal](../research/statement-quality/evidence-series-2026-10-25.json), пока0из3/$0из$2. Receipt каждого dispatch связывает предыдущие validated summaries, cost и следующий reserve; stop при unknown/failed/overcap/не-evidence причине. |

Первый offline reserve test выявил full32-native price_unknown: inline повторы
новых constraints увеличивали schema сверх30k input плана. Вместо повышения
лимитов одинаковый nullable field вынесен в $defs/field и $ref, как в прежнем
DirectSchema. Форма ответа прежняя. Повтор вместимости PASS, новая actual сумма
247349; консервативный план267205 и максимальная цена35654/call не меняются.
Первый fixed old reserve assertion ожидаемо показал новую сумму; baseline test
обновлён только после реального вычисления. Rate/token/model/call/total caps прежние.

Принимаются только реальные verbatim quotes и fractional boxes; автоматического
рескейла/изобретения evidence/null-подмены после ответа нет. Null предлагает
сама модель, затем прежний scorer проверяет его по неизменному golden.
Actual page bounds/площадь/byte limits по-прежнему проверяет Go. JSON Schema не
заменяет domain validation и не гарантирует grounding или PASS_SYNTHETIC.

## Actual первая попытка и compatibility candidate

Q4.3 run40204752709: INCONCLUSIVE/HTTP400,1POST/6GET, unknownBills1, reserve30329micros.
[Полный evidence](0056-q43-execution-evidence.md). S series frozen1/3; paid продолжение
требует отдельно принятого T и authoritative reconciliation. Offline candidate v3
убирает $defs/$ref, сохраняя evidence constraints; actual reserves255817, preflight267205,
8max285232. Причина HTTP400 остаётся UNVERIFIED; это controlled compatibility change,
не доказанное исправление сервера.

Inline candidate checks: `make test` PASS (339/339Web), `go vet ./cmd/... ./internal/...`
PASS, предметные Go/fake campaign/replay/negative controls PASS, q4 fake workflow
PASS с сохранностью q/q2/q3 файлов, `git diff --check` PASS. HTTP400 reader
privacy/snapshot controls4/4 PASS. Ни новых inference, ни T SQL/metadata/settlement
операций после frozen события не выполнено.
