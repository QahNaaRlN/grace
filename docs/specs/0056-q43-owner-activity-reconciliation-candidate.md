# SPEC-0056 U: один Q4.3 settlement по owner Activity и HTTP400 policy

- Статус: **accepted U1–U3**; ops code NOT_STARTED.
- Приёмка владельца 2026-10-25: «Принять U и продолжить (рекомендую)»
  в ответ на отдельный вопрос о settlement0, contingency30329 и оставшихся2/$2.
- Родители: [accepted S](0056-evidence-alignment-series.md), [accepted T](0056-q43-billing-reconciliation-candidate.md).
- Это исключение только для Q4.3; прежние модель/8 документов/качество/≤3 attempts/$2 не расширяются.

## Проверенные факты и предел вывода

Q4.3: run40204752709/attempt1, sourcea49542e2db7522adf30171ef2cf4b2198016fa67,
manifest8f378b10d708075d5a3badcb4cf3f7d016c967f5ff86c8854c0c8b541e1de3da,
reportc6f93f6246c9e37612c7861865fdab606b700184130aa8d4427e79fa9fad5d4f.
Один POST получил HTTP400/error JSON642bytes, без generation ID/usage/cost;
остальные7 документов не отправлены. Одна usage row uncertain/reserve30329micros.
[T evidence](0056-q43-reconciliation-evidence.md):2SSH/1GET/одна row read,0ledger writes.

Владелец предоставил Activity screenshot и прямо сообщил: «Это единственный запрос
за 25-е число». SHA256 исходного изображения
`f6a39d60ce4429ed9f9bc5fadd77e36ac7f15d23ff32484dabfa8a340820ace3`.
Видимая единственная строка:25Oct16:59, Gemini3.1FlashLite/GoogleVertex,
5551input/2216output, $0.00518, finish=stop. Она совпадает с уже settled
[Q4.2](0056-q42-execution-evidence.md):11:59UTC/16:59+05,5551/2216,$0.005183.
Это не новый bill Q4.3; повторно учитывать его нельзя.

[OpenRouter pricing](https://openrouter.ai/pricing) сообщает, что errored attempts
не оплачиваются за model tokens; уже выполненные auxiliary services могут
оплачиваться. [Errors docs](https://github.com/OpenRouterTeam/docs/blob/main/api_reference/errors-and-debugging.mdx)
отличают HTTP400 от ошибок после начала обработки/HTTP200. В source Q4.3 используется
прямой chat-completions input нормализованных страниц, без web search/file-parser
plugins, fallback или native PDF parsing service. Проверить это по source/actual
request builder до apply, не читать другие операции/owner financial data.

Совокупность HTTP400 без generation/usage, owner-supplied Activity и policy
поддерживает административное закрытие без списания. Это **не direct per-request
metadata proof bill0**. U явно меняет основание settlement только этого резерва;
source evidence/исторический unknown не переписываются в metadata VERIFIED.

## U1. Один ограниченный settlement

Разрешить **один дополнительный pinned root SSH** (третий из исходных3 T sessions),
без новых metadata GET/POST, SCP, key/env/financial reads, deploy/restart/migrations.
Читать только fixed q3 marker/report, существующий T audit SHA256
`374579f1c99cceb90725e9450a247bf3677bd6302029249668025a652f13d332` и одну
usage row по успешному reserve UUID из sealed report. Проверить все прежние digests,
source/manifest/receipt и row owner/model/provider/purpose/reserve/time/status.
Owner берётся только из этой bound row; не допускается выбор другого владельца.

Через общий operator lock и tenant-scoped transaction с budget lock/row lock
вызвать существующий `grace_ai_settle` с actual0 и отдельным стабильным marker
`owner-activity-http400:<q3-report-digest>`, **не** с generation ID Q4.2.
Read-after-write проверяет тот же owner/row, settled/actual0/marker и прежние
policy/reserve/created fields. Конфликт/изменённый snapshot/row — rollback и stop.
Idempotent replay только при exact actual0/marker; никакой другой row/function/
allowance/budget/financial mutation не разрешается.

Отдельный private audit0600
`/opt/grace/shared/document-reading-spec0056-q3.activity-settlement.json` хранит
report/T-audit hashes, source owner Activity image hash, billingBasis
`owner_accepted_policy_and_activity`, directMetadataVerified=false, before/after,
actual0/committed. Исходные report/marker/T-audit сохраняются. Наружу только
closed flags/status/cost/hash, без IDs/owner/key/raw response или screenshot.

## U2. Остаток серии с сохранённым запасом

В journal отдельно записать `closed_by_owner_policy`, settlement0 и basis U;
historical summary unknownBills1 остаётся прежним. Не выдавать это за прямое
подтверждение OpenRouter. **30329micros остаются консервативным запасом внутри
исходных $2** до конца серии, даже после освобождения ledger reservation.

Допустить оставшиеся2из3 попыток S. Перед каждым dispatch:
reconciled spend + contingency30329 + next maximal reserve≤2000000;
для двух попыток worst plan30329+2*285232=600793micros ($0.600793).
Только clean integrated source, exact source CI5/5, новый manifest/receipt и
one-shot workflow. Новый unknown bill снова останавливает серию; U не делает
автоматический settled0 для следующих HTTP400. Стоп на actual PASS_SYNTHETIC8/8
либо прежних лимитах/не-evidence причине. Production activation/import закрыты.

## U3. Проверка перед исполнением

Acceptance commit до ops implementation. Offline privacy/tamper/row-owner/conflict/
idempotency controls и изолированный SQL тест существующей settlement функции.
Перед merge все gates S (make test/production/integration, vet, diff, exact CI5/5).
После исполнения отдельный audit projection/AC evidence; отсутствие VERIFIED
операции или конфликты не допускают новых POST.

## Принятое решение владельца

Владелец принял U1–U3: закрыть только Q4.3 без списания на явно указанном основании
policy+owner Activity, сохранить $0.030329 запаса в серии и довести оставшиеся
2 попытки до PASS либо прежнего stop. Новых GET/дополнительных attempts/$ нет.
Предоставление screenshot само по себе не считается принятием этого исключения.
