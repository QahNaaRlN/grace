# SPEC-0056 R: pricing repair — implementation evidence

Дата: 2026-10-25. Принятый контракт: [SPEC-0056 R](0056-q4-pricing-repair.md).
Приёмка спецификации и условный допуск до одного Q4.2: ответ владельца
«Принять и довести до прогона»; receipt `e0fbfa4b0d3d0fbc7174dd7eae3b1cbee9aa49fe`
предшествует implementation diff. Реализация проверена локально; CI/merge/live
фиксируются далее отдельно. Реальное качество этим отчётом не заявляется.

Request pricing декодируется с различением отсутствующего ключа и explicit null.
Только experiment group с положительным cap и явным `router_ceiling_only`
может использовать omission; production constructor этот режим отвергает.
Полный cap резервируется и отправляется в `provider.max_price.request`.
Image rate по-прежнему обязателен и проверяется; candidate ceilings1000/1micros.
Новый fixed q2 marker/report сохраняет старую consumed identity; общий lock прежний.
Scorer, corpus/order, golden, prompts/schema, production config и financial core
не изменены. Нет реальных provider calls в локальных проверках.

| AC | Исполняемое evidence / результат |
| --- | --- |
| R1 | `go test ./internal/assistant -run 'TestR(RequestFeeEvidenceAndWireCaps|FeeEvidenceCannotEnterProductionOrLosePositiveCap)'`: PASS. Omitted research разрешён, strict omission/null/empty/nonstring/negative/nonfinite/overcap и invalid mode дают0POST; ordinary production mode отвергнут до network. |
| R2 | Тот же actual grouped wire test: request0.001/image0.000001, один pinned provider, fallback=false, полный fee reserve. `go test ./internal/documentqualification -run 'TestR(ActualCandidateConservativeReservations|ResearchOmittedRequestCampaignAndStrictReplay)'`: PASS. Actual request reserves254267micros; campaign preflight267205;8 maximal285232. Strict saved-wire omission остаётся INCONCLUSIVE0POST/3GET; image missing/invalid/overcap отвергнут. |
| R3 | `go test ./internal/documentqualification ./cmd/document-reading-eval`: PASS. Новый cap-only positive fake campaign8POST/6GET/8settled →PASS_SYNTHETIC, full32 native/scan по297checks; replay и projection проходят. `TestRCapOnlyFailuresStopWithoutFollowingPOST`: quality/timeout/wrong model/unknown cost/overrun/ZDR/text-only останавливают дальнейшие POST, known quality bill settled, unknown bill frozen. Прежние forgery/negative/source/manifest/secret guards проходят. |
| R4 | `python3 -B scripts/research/test-document-reading-workflow.py`: PASS. Реальные workflow blocks/wrapper/CLI с fake transport, q2 spent/lock/no-clobber/source/attempt1/cleanup, старые q marker/report неизменны;0 real SSH/provider calls. |
| R5 | Локальные `make test` PASS (включая339/339 Web), `go vet ./cmd/... ./internal/...` PASS, `git diff --check` PASS. Exact-head CI обязан исполнить также `make test-production` и `make test-integration`; только после5/5 и main source CI допускается receipt/dispatch. Результаты CI пока PENDING. |
| R6 | Q4.2 NOT_DISPATCHED; условный grant пока unspent. После gates новый receipt свяжет exact source/manifest/caps и разрешение; один dispatch, verdict/cost/counters будут отдельным runtime evidence. |

Лимиты остаются≤8POST/6GET/$2 total/$0.25 call. Request omission означает
неподтверждённую metadata цену под операторским/router ceiling, а не цену0.
Ни offline PASS, ни будущий PASS_SYNTHETIC не активируют production profile
и не разрешают финансовый import. Повтор после расходования grant не разрешён.
