# SPEC-0055 / S1.4: один screening современной модели

- Дата: 2026-10-20, Asia/Yekaterinburg.
- Статус: accepted; implementation_status: local_verified (C1–C5 PASS), C6 FAIL provider-stage; billing unknown, shared ledger uncertain.
- [Implementation evidence](0055-s14-evidence.md).
- Acceptance: «работаем» в ответ на предложение следующего отдельно согласованного
  smoke: Gemini 3.1 Flash Lite / Google EU, 3 public cases / provider cap $0.1617.
- Ветка: codex/statement-candidate-smoke; baseline 7c4ef83; S1.3 local_verified.
- [S1.3](0055-s13-turnover-roles.md), [shortlist](../research/statement-quality/model-shortlist-2026-10-20.md).

## Результат и граница допуска

Подготовить и выполнить один bounded screening без замены production policy,
без нового product release и без финансовых writes. Новая exact пара:
`google/gemini-3.1-flash-lite` / `google-vertex/eu`, display `Google`;
ceilings 275000 input / 1650000 output USD micros per 1M. На документ:
100000 input / 16000 output → 53900 provider micros. Whole-run ceiling
161700 micros ($0.1617), прежние fees/shared allowance, максимум 3 calls.
Первая ошибка прекращает run; неиспользованные calls не переносятся в повторы.
Никакого автоматического enable при PASS; SPEC-0056 registry остаётся draft.

## Конкретный контракт

1. Отдельные screening builder/canonicalizer и reasoner constructor допускают
   ровно эту пару с этими ceilings. Product Policy.Valid, persisted consent,
   обычный StatementReasoner/transport и activation summary по-прежнему её
   отвергают. Общая реализация planner/schema/aliases переиспользуется без
   подмены identity кандидата на baseline. Candidate не подключается к HTTP API.
2. Новый candidate runner выполняет columns → cards → turnover-roles-ru, только
   shipped hashes; все layouts и планы проверяются до общей reservation.
   Baseline scorer 19/20 не ослабляется. RU scorer отдельно проверяет 2 rows,
   money/currency/direction/dates/descriptions, отсутствие invented booking/FX и
   book fields, available 0, точные turnovers, coverage и Go blockers.
3. Report имеет отдельный version/profile/scorer и не проходит production
   ValidateReport/summary даже при PASS. Сохраняет все planned case hashes,
   plan/document hashes, все вызовы, generation IDs, exact billing и latency,
   canonical responses. Неполное выполнение — FAIL, все failures сохраняются.
4. Те же safeguards: whole-run reserve через grace_ai_reserve, settle exact
   через grace_ai_settle; invalid/unknown billing → uncertain и общий запрет
   дальнейших paid calls. Zero-call failure после reservation settles 0.
   No retry/repair/fallback/tools/images/raw PDF. Exact model/provider response
   проверяет прежний OpenRouter transport; reasoning disabled, strict schema,
   ZDR=true, data_collection=deny, only exact EU endpoint.
5. Preflight read-only: fresh official endpoint metadata + ZDR exact pair,
   profile/caps/cost ceilings и helper fixtures. До smoke подтвердить наличие
   единственного active admin и shared reserve с fees (reserve authoritative).
   Metadata/price/route mismatch, uncertain ledger или нехватка бюджета дают
   отказ до inference. Secrets и usage values не выводятся в публичный CI log.
6. CLI statement-admin: отдельные screening-preflight/screening modes, fixtures
   из закрытого операторского каталога, bounded reads и hash check до extraction.
   При preflight нет reservation/inference. Activation modes сохраняются.
   Remote runner исполняет exact static binary + 3 public PDF во временном
   каталоге текущего API container, используя его existing key/socket/DB config.
   Не меняет image, config, services, Statement AI flag и пользовательские drafts.
7. Запуск через существующий GitHub deployment workflow с новым explicit
   screening-only input (default false): отдельная job, обычный deploy и inspect
   взаимоисключены с screening. Exact SHA checkout/build/hash; serialized production
   concurrency; remote exclusive lock и one-shot marker до процесса smoke.
   Workflow rerun с тем же authorization ID не делает paid replay. Full report
   хранится приватно на VPS, CI получает только fixed result/check codes.
   Временные binary/PDF очищаются; report/marker остаются для аудита.

## Проверки

| ID | Evidence |
| --- | --- |
| C1 | Product rejects candidate; screening accepts only exact bounded pair; stale/tampered plans rejected |
| C2 | Fake HTTP request pins route/model/ZDR/deny/reasoning/caps; route failure retains known billing, unknown remains uncertain |
| C3 | Pinned helper 3-case fake run PASS, RU negatives FAIL, no scorer relaxation; reserve once/settle once, stop without retry |
| C4 | Wrong hashes/order/caps/unknown billing/duplicate authorization cannot cause paid replay or activation |
| C5 | make test, affected integration, activation/workflow guards, go vet, diff check |
| C6 | Ровно один live run <=3 calls / <=161700 provider micros, full report retained; FAIL не заменяется повтором |

420 s/run, 120 s/call, прежние document/transport bounds. Проверка 3 fixed cases
не означает full corpus qualification или доказанную эффективность на всех банках.
После результата — отдельное решение по дальнейшему corpus/production adoption.
