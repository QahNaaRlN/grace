# S1.17 E6 — разрешение, закреплённый source и единственный dispatch

Запись до dispatch: 2026-10-23T08:35:30+00:00 / 2026-10-23T13:35:30+05:00, Asia/Yekaterinburg.

- Accepted contract: [S1.17](0055-s117-alias-comparison.md), E1–E5/E6.
- Владелец принял offline реализацию и подтвердил переход: «влил PR, переходим к E6».
  Отдельное разрешение E6 дано ранее: «Даю отдельное разрешение».
- Дополнительно владелец явно ответил: «Разрешаю слить №101 после CI5/5
  и продолжить E6». PR №101 слит после канонической проверки CI5/5.
- Текущий статус: AUTHORIZED / NOT_DISPATCHED, permission UNUSED до запроса dispatch.
- PR №100 интегрирован7651668. YAML blocker исправлен [PR №101](https://github.com/QahNaaRlN/grace/pull/101),
  merge `9cf0d03322f0ab2e9875863a075231336a25e4b3`, 2026-10-23 08:34:04 UTC.
- **Закреплённый executable source:** `24f73223b9b3370ca78c41d34e89edca9a276488`.
- **Замороженный ref:** `codex/statement-s117-e6-source`, тот же exact SHA.
  Worktree `statement-s117-e6-source` clean. Ref дальше не сдвигается.
- Source является предком main9cf0d03; trees byte-identical (`git diff --quiet`).
- [Exact-source CI39924326517](https://github.com/QahNaaRlN/grace/actions/runs/39924326517)
  SUCCESS5/5: Quality gates, PostgreSQL integration, api/pdf-extractor/web image builds.
  `verify-screening-source-ci.py` подтвердил exact24f7322, полный состав и успех всех jobs.
  Это CI выбранного интегрированного source, не перенос CI на merge9cf0d03.
- Merge CI/Delivery9cf0d03 проверяются отдельно. Primary9cf0d03 clean, permissions0700 сохранены.

## Единственное разрешённое исполнение

Один workflow_dispatch `deploy-production.yml` на frozen ref выше, authorization
`s117-comparison-20261023`. До6POST / **208500 USD micros ($0.2085 provider)**,
по одному на tuple, порядок A-columns → B-columns → A-cards → B-cards → A-RU → B-RU.
Shared $10 budget, fees, remaining/uncertain и historical rows не меняются.
Provider cap не является fee-inclusive итогом.

| ID | Request → canonical | Endpoint / provider | Max micros на PDF /3PDF |
| --- | --- | --- | --- |
| A | google/gemini-3.1-flash-lite → google/gemini-3.1-flash-lite-20260507 | google-vertex/eu / Google | 53900 /161700 |
| B | inception/mercury-3 → inception/mercury-3-20261019 | inception / Inception | 15600 /46800 |

Только bundled public synthetic PDFs с accepted exact hashes, прежние limits:
input100000/output16000,120s/call,900s campaign,5s settlement, bytes90000/spans2000.
Не используются личные PDF, owner drafts, D5/P6 artifacts или финансовые записи.

| Input | Значение |
| --- | --- |
| ref | codex/statement-s117-e6-source |
| statement_comparison_s117_only | true |
| comparison_s117_authorized_sha | 24f73223b9b3370ca78c41d34e89edca9a276488 |
| statement_action | preserve |
| release_sha | empty |
| Все остальные screening/comparison/diagnose/inspect modes | false |
| Оба reconcile modes | none |
| Остальные authorized SHA inputs | empty |

Все6plans/общий reserve проверяются до первого POST; call_allowed — перед каждым
Complete. Fresh catalog/endpoint/ZDR и caps обеих пар, global AI/key/helper,
один active admin, shared budget/uncertain, manifest/source, actual image/health
и отсутствие новых artifacts **UNVERIFIED до однократного preflight**.

Общий lock, no-clobber marker до readiness, private0600 fixed artifact,
outer/inner digests и v3 profile проверяются прежним принятым runner.
Preflight refusal также расходует попытку. После dispatch, любого отказа/FAIL
или неизвестного сетевого результата **повтор не выполняется**.
Не меняются pins, source, models/flags, environment, allowance или guards;
нет retry/fallback/repair/deploy/financial writes/reconciliation.

После run входит только read-only inspection/replay своего нового fixed artifact
в рамках workflow: exact unique reservation correlation, закрытые результаты,
known/unknown cost, Complete latency, bytes/mode/mtime, image/health.
Private snapshots, free text, UUID/generation/provider IDs и ключи не публикуются.
Unavailable/missing capture остаётся UNVERIFIED; малый/неполный run не даёт winner/p95
или production qualification. [Результат E6](0055-s117-e6-evidence.md) заполняется после run.
