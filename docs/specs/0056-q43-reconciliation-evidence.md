# SPEC-0056 T — correlation отсутствует, продолжение BLOCKED

2026-10-25. [Принятый T](0056-q43-billing-reconciliation-candidate.md),
приёмка владельца сохранена commit `9e9bd21` **до реализации** нового ops reader.
[Q4.3](0056-q43-execution-evidence.md) и [journal](../research/statement-quality/evidence-series-2026-10-25.json).

Первый из ≤3 разрешённых pinned SSH выполнил фиксированный reader по двум
запечатанным q3 файлам. Итог: **UNVERIFIED / generation_correlation_absent**.
Надёжного generation ID в сохранённой ошибке нет. Authoritative cost отсутствует;
settlement NOT_EXECUTED, оставшиеся2 попытки BLOCKED по T1/S3.

| Критерий | Исполняемое evidence / результат |
| --- | --- |
| Accepted contract перед ops code | `git show 9e9bd21 -- docs/specs/0056-q43-billing-reconciliation-candidate.md` |
| Только fixed q3 marker/report; source/manifest/receipt/digests | [Reader](../research/statement-quality/q43-billing-correlation-reader.py), [закрытая projection](../research/statement-quality/q43-T-correlation-inspection.json): snapshotVerified/Unchanged=true |
| Reliable correlation до key/SQL/metadata | Strict gen-* в явных ID fields, embedded raw JSON; отсутствие correlation; без угадывания из message/request_id |
| ≤3SSH / ≤1 metadataGET, без POST | Фактически1SSH,0GET,0POST; sqlCalls/keyReads/persistentWritesRequested0 |
| Отсутствующая correlation останавливает без settlement | resumeAllowed=false, settlementApplied=false; исходные unknownBills1/reserve30329micros сохраняются |
| Проверка ровно одной usage row / authoritative metadata | NOT_EXECUTED: остановка раньше SQL/key/GET по T1 |
| Conditional T2 transaction/audit before-after | NOT_EXECUTED: нет VERIFIED cost, writer не запускался и не реализован в этом изменении |
| Условное возобновление T3 | BLOCKED: consumed1/3; remaining2; $2 total прежний; paid calls после freeze0 |
| Негативные/приватные controls | `python3 -B docs/research/statement-quality/q43-billing-correlation-reader-test.py`: 8/8 PASS; explicit/nested ID, message non-correlation, ambiguity, privacy, tamper/change/duplicate refusal |

Reader SHA256 `7b31b21a136410f9e1010a060cb8da0926e255d69821dc9ec60e372f8c2e718e`.
Report object `c6f93f6246c9e37612c7861865fdab606b700184130aa8d4427e79fa9fad5d4f`;
report file `7f8ed93cf6862972cccdce3486fd81bed58db2e3dbd92f8821891f52cc05dee7`;
marker `730afbd8b27061c0e31e06ebb92d925190580aee6793b70e1031d5da475c3b8c`.
Файлы совпадают с предыдущей diagnosis; серверных persistent writes не запрошено.
Наружу не выведены response literals, generation/request IDs, UUID/owner/key/DSN.

Нужны owner-supplied сведения Activity ровно для единственного Q4.3 запроса:
2026-10-25T14:58:49Z..14:59:44Z (19:58:49..19:59:44 Asia/Yekaterinburg),
`google/gemini-3.1-flash-lite`, HTTP400: generation ID/authoritative total_cost
либо подтверждение OpenRouter об отсутствии списания именно за это событие.
Общая pricing policy или пустое usage.cost не доказывают bill0. Полученное evidence
потребует проверки identity и соответствия T; текущий результат не объявляется PASS.

Offline compatibility candidate остаётся гипотезой по HTTP400; причина не доказана.
Исходный summary unknownBills1 не переписан, quality checks не вычислены.
