# SPEC-0056 U — одна policy reconciliation исполнена

2026-10-25. [Принятый U](0056-q43-owner-activity-reconciliation-candidate.md),
acceptance commit `423e7ff` до ops implementation. [Journal](../research/statement-quality/evidence-series-2026-10-25.json).

Один разрешённый U pinned SSH закрыл **ровно один** Q4.3 reserve30329micros:
status settled/actual0, basis **owner_accepted_policy_and_activity**,
**directMetadataVerified=false**. Это исполненная операция по принятому исключению,
не per-generation metadata proof bill0. Q4.2 не переучтён, его ID не использован.
Historical Q4.3 report/summary unknownBills1 сохранены. Остаток серии2из3;
contingency30329micros ($0.030329) остаётся внутри прежних $2.

| AC | Исполняемое evidence |
| --- | --- |
| U1 accepted до code | `git show 423e7ff -- docs/specs/0056-q43-owner-activity-reconciliation-candidate.md` |
| Exact source/request без auxiliary services | `git show a49542e:internal/assistant/llm_provider.go`: SHA256 f5bb9a8b01b76fb3c2ede779b916f356ffd8b57f058ce627a8f1d14cacaa6c41; current builder identical; explicit body model/messages/max_tokens/stream/provider/response_format/reasoning, no tools/plugins/native file parsing; fallback=false |
| Fixed report/marker/T audit/row binding | [Writer](../research/statement-quality/q43-owner-activity-settle.py), SHA256 2adcd3eab5e3dc655a39fd7caa0a789434e2e2bf9ca5e2bf3805dc9d95e00e25; marker/report/T audit matched fixed hashes before transaction and before COMMIT |
| Owner-scoped exact row | UUID только из одного successful reserve sealed report; owner/policy30329/time/status сверены с единственной row/T audit; operator+budget+row locks, grace_financial/grace.owner_id и existing grace_ai_settle0 |
| Rollback до COMMIT / read-after-write | Открытая SQL transaction ожидает Python snapshot/after-row checks перед COMMIT; затем повторное чтение той же row, owner/immutable fields unchanged |
| Separate private audit | `/opt/grace/shared/document-reading-spec0056-q3.activity-settlement.json`,0600, before/after/committed=true, basis U/directMetadataVerified=false; SHA256 ce361cc90481153d32d704315ffc34a5c3a5ead469d0133928e4da0eb42ea2b1 |
| Closed projection / actual limits | [Result](../research/statement-quality/q43-U-settlement.json): VERIFIED_OPERATION,1target row, owner/snapshot unchanged, actual0/contingency30329; U1SSH,0GET/POST/key reads, T+U total3SSH/1GET |
| U3 offline and real SQL controls | `./scripts/research/test-q43-owner-settlement.sh`: **11/11 PASS, skipped0**. Actual migration13 function/RLS in disposable PostgreSQL network=none/no ports; commit, wrong/stale owner, conflict, idempotency, precommit snapshot rollback, neighbor row/budget unchanged. Unit privacy/audit/replay/refusal controls;0live SSH/HTTP/key reads |
| U2 original result and cap preserved | Original summary remains unknownBills1; journal unresolved0 only after accepted administrative settlement, contingency30329; reconciled0+30329+2*285232=600793micros<$2 |

Запрос owner Activity screenshot сохранён только как hash и закрытые наблюдения в
принятом U; raw screenshot/key label/ID не публикуются. Старые report/marker/T audit
не перезаписаны. Нет изменений финансовых записей, allowance, функций/migrations,
других usage rows, production policy/deploy или activation/import.

## Gates и следующий run

PR130 интегрирован `a8752028463d19351c9eddcb0ee40b71fe398b47` после exact head
`709e8bbfacd7095ba8efbe9e8b1a74f5fbd76e58`, [CI40222097044](https://github.com/QahNaaRlN/grace/actions/runs/40222097044)5/5 PASS.
Все локальные gates PR130 PASS. Для U source перед следующим dispatch обязательны
новые make test/production/integration/vet/diff и exact head/main CI5/5,
clean manifest/binary/опубликованный receipt. Q4.4 пока NOT_DISPATCHED;
actual PASS_SYNTHETIC не заявляется.

## Результат продолжения

После U/PR131 и опубликованного receipt выполнен Q4.4/attempt2из3:
[INCONCLUSIVE/HTTP400/unknown bill](0056-q44-execution-evidence.md).
Source a875202 прошёл mainCI40223309245/5, U head d692de9 — CI40224022530/5.
Последний слот сохранён, серия снова frozen; U не распространяется на новый bill.
