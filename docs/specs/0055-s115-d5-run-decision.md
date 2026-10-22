# S1.15 / D5 — кандидат одного read-only запуска над saved P6

- Статус: **candidate; execution BLOCKED_PENDING_EXPLICIT_APPROVAL / NOT_EXECUTED**.
- Дата подготовки: 2026-10-23, Asia/Yekaterinburg; acceptance owner: владелец Grace.
- Основание: поручение актуализировать разработку и следующий этап. Первоначальное поручение разрешало
  подготовку решения; исполнение остаётся закрытым после автоматического отказа ниже.
- Контракт: [accepted S1.15](0055-s115-saved-proposal-diagnosis.md), D1–D4 integrated;
  [offline evidence](0055-s115-evidence.md), [post-merge №94](0055-pr94-postmerge.md).
- Предлагаемый verifier SHA: `703b6c6ac960c21c0b7ab7bd84b3411459204c92`.
- Предлагаемый ref: `codex/statement-saved-proposal-diagnosis`, сохранён на этом SHA.
- Source CI: [39639515724](https://github.com/QahNaaRlN/grace/actions/runs/39639515724),5/5 PASS;
  source интегрирован PR №94, merge17e3493; полный tree identical подтверждён.
  Merge CI39664307328=5/5 и Delivery39665622283=4/4 PASS.
- Наличие и доступность private saved artifacts на2026-10-23: UNVERIFIED, VPS не читался.
- Effective execution approval: **отсутствует**. Ответ «влил PR, продолжить работу» был первоначально истолкован как approval (local commit7610661), но автоматическая проверка отклонила dispatch до запуска процесса: требуется явное согласование production workflow с SSH-чтением приватного P6 и передачей закрытого результата через CI. Первоначальная интерпретация отозвана этой записью; отдельный прямой ответ владельца на точное действие ещё нужен.

## Предлагаемое решение владельца

Разрешить **один** dispatch `deploy-production.yml` с единственным выбранным job
`diagnose-statement-s115` на exact SHA/ref выше. Это fixed-host read-only diagnosis
того же private P6 artifact, не новая comparison campaign.

Ожидаемый результат — закрытая проекция `statement.saved-proposal-diagnosis.v1`:

1. A/columns: production/scorer replay и structural witness первого row2 description gap.
2. B/columns: actual unmapped membership, count/positions/lexical class, максимум128 witnesses.
3. A/cards: exact unknown_alias guard/position и relation к исходным span IDs / syntax aliases.
4. Остальные3tuples: not_applicable/not_sent, без выдуманной diagnosis.

Для каждой issue VERIFIED либо честный UNVERIFIED; mismatch integrity/context
закрывает весь отчёт без witnesses. Alias/ID/text/amount, приватные snapshots,
model service reasons и arbitrary exceptions не публикуются.

## Ограничения исполнения после отдельного approval

- Фиксированный host: `grace-deploy@195.133.195.190`, pinned known_hosts.
- Только `/opt/grace/shared/statement-comparison-s114-comparison-20261021.json`
  и соответствующий `.started`; marker source остаётся
  `301afd6bedc4a19559e731471336bbc4e2f74075` — это historical P6 source,
  он отличается от verifier703b6c6. Ни один artifact/marker не меняется.
- Public contexts готовятся локально из exact3 fixtures через pinned Poppler25.04.0,
  network=none. Перед SSH: exact source/ref + CI5/5, полный manifest
  verifier/context/SOURCE_SHA/bootstrap/wrapper; drift даёт отказ, без замены SHA.
- На VPS Python `-B`, sealed memfd и verifier env={}; fixed0600 O_NOFOLLOW reads,
  before/after hash/size/mode/mtime совпадают. Atime не является критерием.
- Binary<=16MiB, context<=524288bytes, envelope<=32MiB, stdout<=65536bytes,
  remote45s/SSH75s. stderr и failed-process stdout не выводятся.
- Provider inference=0, catalog/endpoint/ZDR/generation metadata GET=0, SQL=0;
  reserve/settle/budget/fees/ledger writes=0. Application image/model/flags не меняются.
- Persistent writes requested verifier/bootstrap=0. Это не обещание отсутствия OS/SSH logs.
- Нет runtime health/readiness probes, Docker/API/key/config/environment changes
  на VPS. Недоступность artifact/permissions/metric остаётся UNVERIFIED.
- Один dispatch; отказ/timeout/UNVERIFIED не разрешает повтор, chmod, repair,
  новое сохранение snapshot, иной source/path или новое обращение к модели.

## Зафиксированные inputs кандидата

Это inputs кандидата одного dispatch, ещё не разрешённого к исполнению. До его выполнения
исполнитель повторно проверяет ref/SHA, ancestry и CI exact source. Дрейф требует
нового решения владельца, а не тихого обновления параметров.

```json
{
  "release_sha": "",
  "statement_action": "preserve",
  "reconcile_screening_usage": "none",
  "reconcile_screening_s16_usage": "none",
  "statement_screening_only": false,
  "statement_screening_s16_only": false,
  "statement_screening_s18_only": false,
  "diagnose_statement_s19_only": false,
  "statement_comparison_s111_only": false,
  "comparison_authorized_sha": "",
  "diagnose_statement_s112_only": false,
  "diagnosis_authorized_sha": "",
  "statement_comparison_s114_only": false,
  "comparison_s114_authorized_sha": "",
  "inspect_statement_only": false,
  "diagnose_statement_s115_only": true,
  "diagnosis_s115_authorized_sha": "703b6c6ac960c21c0b7ab7bd84b3411459204c92"
}
```

Контракт modes проверен32768 combinations. Inputs этого кандидата выбирают
только D5 job; deploy/paid/reconciliation/старые diagnostic jobs не выбираются.

## Evidence и дальнейшее решение

После разрешённого run сохранить source SHA/ref, CI/run/job, полный hash manifest,
closed output и artifactsUnchanged. Прочитанный private report не переносить в
Git/Actions. Не объявлять VERIFIED структурный факт model-quality PASS/winner или
owner financial acceptance. P6 остаётся FAIL/permission consumed/known settled cost.

Выводы становятся основанием отдельного bounded fix candidate; prompt/planner/
alias/scorer repair, новый paid run, SPEC-0056 и deploy остаются отдельными gates.

## Отказ автоматической проверки — 2026-10-23

Попытка вызвать exec_command для единственного workflow dispatch отклонена
на CreateProcess до выполнения shell. Dispatch_count=0, SSH/VPS/provider/metadata/
SQL/environment writes=0. Автоматическая проверка сочла «продолжить работу»
недостаточно явным approval именно чтения private P6 через SSH и CI projection.
Альтернативного tool/API/прямого SSH, retry или indirect execution не выполнялось.
Это отказ разрешения на запуск, а не diagnostic FAIL и не результат replay P6.

Необходимый ответ владельца должен явно разрешить один D5 production workflow
на `703b6c6ac960c21c0b7ab7bd84b3411459204c92` / указанном ref, чтение private P6
на fixed VPS и публикацию только закрытой numeric/enum projection через CI,
при прежних provider/metadata/SQL/persistent artifact/environment writes=0.
После такого ответа перед единственным dispatch снова проверяются exact ref/CI.
