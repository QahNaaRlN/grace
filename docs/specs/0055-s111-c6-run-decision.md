# S1.11 / C6: решение об одном synthetic comparison

- Статус: accepted execution decision; owner approval: 2026-10-21, прямое «разрешаю»
  в ответ на конкретное предложение одного C6 на source012f604, до6POST/$0.2085 provider.
- Подготовлено: 2026-10-21, Asia/Yekaterinburg.
- Основание: [accepted S1.11](0055-s111-model-comparison.md), offline C1–C5
  реализованы в [PR №84](https://github.com/QahNaaRlN/grace/pull/84).
- Точный разрешённый source SHA: `012f604033de9c6a75bf531b8f715323d0fda57d`,
  merge №84. [Post-merge gates](0055-pr84-postmerge.md) обязательны перед запуском.
- Source ref: `codex/statement-comparison-c6-source`, закреплён на exact SHA выше;
  worktree `statement-comparison-c6-source` сохраняется чистым.
- Live status: EXECUTED / FAIL `qualification_protocol_failed`; authorization consumed.
  Один вызов A/columns, остальные5 не отправлены. Retry не разрешён;
  [результат и ограничения evidence](0055-s111-c6-evidence.md).

## Принятое решение

Разрешить **ровно одну** кампанию C6, authorization
`s111-comparison-20261021`, на указанном exact source при успешных local/CI gates
и runtime preflight. Две пары:

| ID | Request alias → canonical | Exact endpoint | Max provider/document | Max provider/3 documents |
| --- | --- | --- | --- | --- |
| A | google/gemini-3.1-flash-lite → google/gemini-3.1-flash-lite-20260507 | google-vertex/eu / Google | 53900 micros | 161700 micros |
| B | inception/mercury-3 → inception/mercury-3-20261019 | inception / Inception | 15600 micros | 46800 micros |

Порядок: A-columns → B-columns → A-cards → B-cards → A-turnover-roles-ru →
B-turnover-roles-ru, не более **6 POST / 208500 USD micros ($0.2085 provider)**.
На tuple один вызов, input100000/output16000, 120s/call и 900s inference context;
финализация settlement отдельно ограничена5s. Общий прежний $10 ledger,
исторические расходы и fees не сбрасываются и не увеличиваются.

Только три прежних публичных synthetic PDF с hashes из accepted S1.11;
planner/prompt/schema/redaction/aliases и scorer19/20/23 с S1.10 guards не меняются.
Inception допускается исключительно для этих публичных fixtures; EU residency
для него не заявляется, личных PDF/финансовых записей провайдеру нет.

## Что уже проверено и что остаётся preflight

[Свежий public snapshot](../research/statement-quality/model-shortlist-c6-2026-10-21.json):
2026-10-21 15:38:47 +05:00 / 10:38:47 UTC, только GET, providerCalls=0.
A/B status0, metadata profile compatible и exact pairs в ZDR list; цены и
canonical identity совпадают с accepted pins. Это не live wire/quality и не
доказательство residency всей цепочки. Baseline Gemini2.5/EU снова отсутствует,
замены route/третьей модели не предлагается. Старый snapshot не перезаписан.

Runtime marker/report, single active admin, global AI/key, actual image/health,
helper readiness, текущий shared remaining/uncertain и реальные metadata
**UNVERIFIED до preflight**. До первого POST runner проверяет точный source,
CI всех5jobs, полный bundle hashes, отсутствие marker/orphan/symlink и fresh
catalog/endpoint/ZDR proofs обеих пар, все6plans и общий reserve. Любой отказ
не даёт права менять route/model/limits/budget или повторять другой smoke.
`grace_statement_call_allowed` проверяется перед каждым Complete.

## Остановки и сохранение результата

Known quality/grounding FAIL сохраняется и позволяет закончить остальные
заранее разрешённые tuples. Transport/protocol/identity/privacy/cost/timeout,
unknown billing, settlement failure или shared block останавливают всю кампанию.
Неиспользованные tuples не переносятся, retry/fallback/repair отсутствуют.

Marker расходуется до первой возможной отправки и не удаляется после FAIL.
Известные расходы всех attempts учитываются точно; unknown не становится0 и
блокирует новый paid work до отдельного решения о reconciliation. Не запускать
старые S1.4/S1.6/S1.8 inputs, не выполнять automatic generation GET/recovery.
Private report сохраняет exact reservation UUID, proofs, traces/diagnostics,
стоимость и latency; read-only inspector сверяет его с одной ledger row.
Public CI выводит только closed codes/bools/counts.

Application image/health сравниваются до/после; deployment, переключение
production flags/model/policy, owner drafts и финансовые записи не входят.
Даже обе пары3/3 не получают production qualification/activation. При n=3
никакого p95 или обещания качества всех банков. Все исторические FAIL сохраняются.

## Исполнение после отдельного owner approval

До approval никаких dispatch нет. После approval оператор ещё раз проверяет,
что source ref равен указанному exact source и его CI5/5 PASS. Вызов существующего
`Deploy production` workflow выбирает **только**
`statement_comparison_s111_only=true`, `comparison_authorized_sha` равен полному
SHA выше; остальные modes false/none, statement_action=preserve.
Workflow выбирает `--ref codex/statement-comparison-c6-source`, который закреплён
на этом SHA. Если ref сдвинулся, несовпадение должно остановить запуск; новое
решение о source предъявляется владельцу, а не подставляется молча. Слияние
этого документа в main само по себе не меняет закреплённый executable source.

Завершение: PASS3/3 обеих пар либо честный FAIL/incomplete; exact billing state,
image/health evidence и сводная таблица quality/cost/latency для владельца.
Owner financial acceptance и production adoption остаются отдельными решениями.

## Запись решения владельца

Владелец ответил «разрешаю» на следующее конкретное предложение:
«Разрешаю один C6 на 012f604033de9c6a75bf531b8f715323d0fda57d,
не более6 вызовов/$0.2085 provider, при обязательном preflight и прежнем shared
бюджете, без deployment/model switch/retry».
