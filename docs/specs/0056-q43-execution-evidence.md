# Q4.3 / S attempt1: INCONCLUSIVE, серия frozen

2026-10-25, Asia/Yekaterinburg. [Accepted S](0056-evidence-alignment-series.md),
[immutable receipt](0056-q43-owner-receipt.md), [series journal](../research/statement-quality/evidence-series-2026-10-25.json).

Первая новая попытка [run40204752709](https://github.com/QahNaaRlN/grace/actions/runs/40204752709)
завершилась **INCONCLUSIVE / billing_requires_review**. Первый `cards` дал HTTP400;
usage/cost отсутствуют. Сохранённая ошибка не подтверждает конкретный неподдержанный
keyword/параметр. Утверждать причиной именно $ref или numeric bounds нельзя.
Полей/строк модельного ответа и quality checks в этом run не измерено.

| Факт | Evidence |
| --- | --- |
| Source |a49542e2db7522adf30171ef2cf4b2198016fa67, integrated PR129; tested head db8c4256314d81497acb1a8476157346b608c44c, tree identical |
| Gates | [PR CI40202818845](https://github.com/QahNaaRlN/grace/actions/runs/40202818845)5/5 и [main CI40203746305](https://github.com/QahNaaRlN/grace/actions/runs/40203746305)5/5 PASS; local make test339Web/vet/workflow PASS |
| Manifest |8f378b10d708075d5a3badcb4cf3f7d016c967f5ff86c8854c0c8b541e1de3da; source clean build/vcs.modified=false |
| Approval |689e0952b85353dd7aee95834ab0b4a01dced6bb, опубликован до accepted dispatch; branchref точныйsource |
| Run |workflow_dispatch,attempt1,completed/success; created14:58:49Z /19:58:49+05:00, updated14:59:44Z; job51s |
| Counters |1POST/6metadataGET; actual wire duration2906ms; остальные7not_sent/billing_requires_review |
| Billing |knownCostMicros0, **unknownBills1**, reserved30329micros=$0.030329; settlement unknown, overCap=false |
| Серия |1/3 attempts CONSUMED, остальные2 frozen; ноль известных списаний **не означает фактический bill0** |
| Quality |0checks, не PASS/не recognition REJECT; importReady=false, activationAuthorized=false |

Получен только [public summary](../research/statement-quality/q43-40204752709.summary.json),
он принят exact-source CLI `-mode=validate-public` с source-bound bundle.
[Run identity](../research/statement-quality/q43-40204752709.run.json).
Wrapper успешно получил private Replay/public projection; это не quality PASS.
Исходные report/marker не перезаписываются ради изменения billing/result.

## Read-only разбор

Два разрешённых fixed-file SSH reader sessions, provider/SQL calls0,
requested persistent remote writes0. [Diagnosis](../research/statement-quality/q43-40204752709.diagnosis.json)
и [reader](../research/statement-quality/q43-wire-reader.py). Наружу только closed
flags/paths/counts/hashes, без error/body/model literals, UUID, keys или DSN.
Оба snapshot совпали внутри и между sessions:

- Report object digest c6f93f6246c9e37612c7861865fdab606b700184130aa8d4427e79fa9fad5d4f.
- Report file7f8ed93cf6862972cccdce3486fd81bed58db2e3dbd92f8821891f52cc05dee7.
- Marker730afbd8b27061c0e31e06ebb92d925190580aee6793b70e1031d5da475c3b8c.
- HTTP400 response642bytes, SHA256281722f9e49cac3f8d58d79d47d450f0fb11928c32b41b031633d6c8e6adc5fe.
- errorCode400, JSON valid, transportFailure=false, usage/cost отсутствуют.
- Final reader SHA2563d0420a129390516e7a27578b1ddce9b8cb0e607ce9314c66b8cd77d91a71605.

Provider adapter `internal/assistant/llm_provider.go` на non-200 возвращает
unbilled response; MeteredCompletion/SQLBudget сохраняют uncertain reservation.
Отсутствие authoritative cost поэтому честно замораживает дальнейшие calls.
Общая OpenRouter policy об ошибках не является доказательством конкретного списания.

## Подготовленный следующий шаг

В рамках S подготовлен **offline compatibility candidate**: inline VisualSchema
без $defs/$ref, прежние0..1/quotes1..8 и более компактный prompt v3, чтобы остаться
в30k input. Это исключает один новый dialect feature, но причина HTTP400 им ещё
не доказана. Candidate проверяется без платных calls; q4 identity сохраняет q/q2/q3.
Никаких изменений validator/scorer/golden/model/financial core нет.

[T принят владельцем](0056-q43-billing-reconciliation-candidate.md), commit9e9bd21.
Первый bounded T1 inspection подтвердил отсутствие generation correlation в
запечатанной ошибке. [Результат T](0056-q43-reconciliation-evidence.md):
1SSH,0metadataGET/SQL/key reads/writes, settlement NOT_EXECUTED, resume BLOCKED.
Сохраняются только оставшиеся2 попытки в исходных $2; historical unknown не изменён.
