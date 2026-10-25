# SPEC-0056 T: один Q4.3 billing reconcile и условное продолжение серии

- Статус: **draft / awaiting owner acceptance**; ops execution NOT_STARTED.
- Основание: frozen S после run40204752709/attempt1, HTTP400, usage.cost отсутствует.
- [Accepted S](0056-evidence-alignment-series.md) и [journal](../research/statement-quality/evidence-series-2026-10-25.json).

S прямо закрывает серию при unknown bill. Новая операция нужна для сверки одного
AI usage резерва30329micros ($0.030329); она не разрешена прежним read-only diagnosis
и не должна молча превращать отсутствие cost в0. Production/ops код этой операции
может быть реализован и выполнен только после принятия T согласно AGENTS.md.

## Фиксированная identity

Source a49542e2db7522adf30171ef2cf4b2198016fa67, receipt689e0952b85353dd7aee95834ab0b4a01dced6bb,
manifest8f378b10d708075d5a3badcb4cf3f7d016c967f5ff86c8854c0c8b541e1de3da,
report object c6f93f6246c9e37612c7861865fdab606b700184130aa8d4427e79fa9fad5d4f.
Report file7f8ed93cf6862972cccdce3486fd81bed58db2e3dbd92f8821891f52cc05dee7,
marker730afbd8b27061c0e31e06ebb92d925190580aee6793b70e1031d5da475c3b8c.
UTC window2026-10-25T14:58:49Z..14:59:44Z; storedPOST1/GET6, unknownBills1.

## T1. Ограниченная authoritative inspection

До3 pinned root SSH command sessions всего: read-only inspection, conditional
apply, verify (apply+verify можно в одной session). Без SCP/deploy/restart/key setup.
Читать только fixed q3 marker/report, ровно одну ai_usage строку по UUID успешного
reserve из запечатанного report, и существующий OpenRouter key только внутри
контейнера API для одного authenticated metadata GET. Значения key/owner/DSN/
row UUID/generation/request ID/raw response наружу не выводятся.

Сверить model/provider/purpose=statement/reserved30329/time window и uncertain
status с report/marker; owner UUID берётся только из этой единственной bound row
для tenant context и должен совпасть до/после transaction. Отказ при отсутствии/
дубликате/расхождении; выбор другого owner или reservation не допускается.
Из error envelope извлечь generation correlation только по строгому gen-* формату.
Если надёжной correlation нет — без guess/search и без изменения ledger остановиться,
запросить owner-supplied узкое Activity evidence для этого события. Общие выписки
пользователя, расходы/счета/проводки/consents не читаются.

Если correlation есть —≤1GET `/api/v1/generation?id=<saved-correlation>` через
существующий key, direct TLS/no redirects/15s/≤1MiB. Никаких POST/fallback/probes.
Проверить exact ID, model alias/canonical, non-BYOK, timestamp interval, error
finish и finite authoritative total_cost (ceil USD micros)≤30329. Для ошибки до
upstream provider identity может отсутствовать только при finish=error,0tokens,
total_cost=0; это отдельная подтверждённая неисполненная generation, не admission
неизвестного provider для inference. Иначе требуется Google match либо отказ.
Общая публичная pricing policy сама по себе не заменяет per-request evidence.

Private audit0600 `/opt/grace/shared/document-reading-spec0056-q3.reconciliation.json`
содержит report hash/metadata/row before и VERIFIED conditions, не меняет исходные
report/marker. Наружу только закрытые booleans/cost/hash/статус. При404/no cost/
identity mismatch/отсутствии correlation — UNVERIFIED и никакого apply.

## T2. Условный settlement

Только после T1 VERIFIED/authoritative cost и неизменности report/row: tenant-scoped
transaction, общий budget lock, lock ровно одной row, owner context и существующий
`grace_ai_settle(reservation, verifiedCost, original-or-verified-generation-marker)`.
Проверить idempotent replay/conflicting-cost refusal и read-after-write той же row.
Сохранить audit before/after, actual cost и committed flag. Не изменять функции,
модели данных/migrations, allowance/лимиты/budget rows, финансовые записи, другие
usage rows. Никакого принудительного settled0 на основании одного HTTP400.
Если settlement failure/неоднозначность — rollback и stop без paid calls.

## T3. Оставшиеся попытки S

После подтверждённого settlement и отсутствия unresolved bill этого run владелец
разрешает продолжить **оставшиеся2из3**, не новую серию. $2 total прежний;
Q4.3 остаётся consumed и historical summary unknown не переписывается.
Journal отдельно хранит original unknown и authoritative reconciliation/cost.
Перед следующим dispatch проверяются сумма spent+next maximal reserve≤$2,
остаток slots, полный набор gates/новыйsource/manifest/receipt и bound evidence repair.
Все S model/corpus/oracle/criteria/price/privacy/stop правила сохраняются.
Новый неизвестный счёт либо исчерпание остатка снова останавливает работу.

## Что требуется принять

Принять T и разрешить указанный bounded inspect→только VERIFIED settle→resume
оставшихся2 попыток S/$2, без повторного запроса тех же операций. Отсутствующее
authoritative billing evidence остаётся блокировкой; положительный исход не обещается.

Официальный reference для единственного GET: [generation metadata](https://openrouter.ai/docs/api/api-reference/generations/get-generation);
общая [pricing policy](https://openrouter.ai/pricing) не заменяет per-request cost proof.
