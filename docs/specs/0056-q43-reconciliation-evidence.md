# SPEC-0056 T — generation time mismatch, продолжение BLOCKED

2026-10-25. [Принятый T](0056-q43-billing-reconciliation-candidate.md),
приёмка владельца сохранена commit `9e9bd21` **до реализации** нового ops reader.
[Q4.3](0056-q43-execution-evidence.md) и [journal](../research/statement-quality/evidence-series-2026-10-25.json).

Первый pinned SSH проверил только два запечатанных q3 файла:
**generation_correlation_absent**, snapshot VERIFIED/unchanged. Владелец затем
передал один Activity generation ID в ответ на узкий запрос evidence. Второй SSH
сверил snapshot, прочитал ровно одну связанную usage row, сделал единственный
разрешённый authenticated metadata GET и сохранил private audit.

Результат metadata: **UNVERIFIED / generation_time_mismatch**. `created_at`
не находится в фиксированном интервале Q4.3. Стоимость этого другого по времени
события не переносится на Q4.3. Authoritative Q4.3 cost остаётся неизвестной;
settlement NOT_EXECUTED, оставшиеся2 попытки BLOCKED по T1/S3.

| Критерий | Исполняемое evidence / результат |
| --- | --- |
| Accepted contract перед ops code | `git show 9e9bd21 -- docs/specs/0056-q43-billing-reconciliation-candidate.md` |
| Fixed q3 source/manifest/receipt/digests | [Первый reader](../research/statement-quality/q43-billing-correlation-reader.py), [projection](../research/statement-quality/q43-T-correlation-inspection.json): snapshotVerified/Unchanged=true |
| Отсутствие reliable stored correlation | Strict gen-* в явных ID fields/embedded raw JSON, без угадывания из message/request_id; остановка до key/SQL/GET |
| Узкий Activity ID владельца | Принят как кандидат для проверки; точный ID остаётся приватным, source `explicit_owner_activity_reply` записан в audit |
| Ровно одна usage row по UUID из report | [Metadata reader](../research/statement-quality/q43-billing-metadata-reader.py) сверил row id/owner/purpose/model/provider/reserved30329/time/status uncertain/actual null/generation null; другие rows не выбирались |
| ≤3SSH / ≤1metadataGET / без POST | Всего2SSH,1GET,1SQL read/одна row; key прочитан только внутри API container; paidPOST0, ledger writes0 |
| Exact generation route/time/error/cost | [Metadata projection](../research/statement-quality/q43-T-metadata-inspection.json): generation_time_mismatch; Q4.3 actualMicros=null; subsequent finish/cost admission не выполнен |
| Private audit0600 | `/opt/grace/shared/document-reading-spec0056-q3.reconciliation.json`, status UNVERIFIED/committed=false, metadata/row before сохранены; SHA256 `374579f1c99cceb90725e9450a247bf3677bd6302029249668025a652f13d332` |
| Conditional T2 settlement/audit after | NOT_EXECUTED: VERIFIED bill отсутствует, writer не запускался и не реализован в этом изменении |
| Условное T3 resume | BLOCKED: consumed1/3, remaining2, $2 total прежний; historical unknownBills1/reserve30329micros сохранены |
| Correlation controls | `python3 -B docs/research/statement-quality/q43-billing-correlation-reader-test.py`: 8/8 PASS; privacy, missing/ambiguous ID, tamper/change/duplicate refusal |
| Metadata controls | `python3 -B docs/research/statement-quality/q43-billing-metadata-reader-test.py`: 9/9 PASS; identity/time/route/finish/cost/ceil, provider exception, exact row, private audit, one GET, no apply, changed report refusal |

Первый исполненный reader SHA256 `7b31b21a136410f9e1010a060cb8da0926e255d69821dc9ec60e372f8c2e718e`.
Второй исполненный reader SHA256 `75a027ee1de74ea16f4be68ec0f7528e12850331ab3585e1173c760db0c34c1f`;
после исполнения локально уточнён счётчик SQLReads на ветке отказа второго чтения
и комментарий; фактический отказ по времени сделал ровно один SQL read.

Report object `c6f93f6246c9e37612c7861865fdab606b700184130aa8d4427e79fa9fad5d4f`;
report file `7f8ed93cf6862972cccdce3486fd81bed58db2e3dbd92f8821891f52cc05dee7`;
marker `730afbd8b27061c0e31e06ebb92d925190580aee6793b70e1031d5da475c3b8c`.
Наружу не выведены response literals, generation/request IDs, row UUID/owner/key/DSN.
Никаких updates ai_usage, budget, financial state или деплоя не выполнено.

Нужна owner-supplied строка Activity именно за Q4.3:
2026-10-25T14:58:49Z..14:59:44Z (19:58:49..19:59:44 Asia/Yekaterinburg),
`google/gemini-3.1-flash-lite`, HTTP400: время/model/authoritative cost и ID,
либо подтверждение OpenRouter об отсутствии списания именно за это событие.
Разрешённый metadata GET исчерпан; второй этим T не открыт. Отсутствие события
в Activity само по себе ещё не подменяет подтверждённую стоимость.

Offline compatibility candidate остаётся гипотезой по HTTP400; причина не доказана.
Исходный summary unknownBills1 не переписан, quality checks не вычислены.
