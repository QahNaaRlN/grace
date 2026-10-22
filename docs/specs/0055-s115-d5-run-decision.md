# S1.15 / D5 — кандидат одного read-only запуска над saved P6

- Статус: **accepted execution decision; one dispatch authorized / NOT_EXECUTED**.
- Дата подготовки: 2026-10-23, Asia/Yekaterinburg; acceptance owner: владелец Grace.
- Основание: поручение актуализировать разработку и следующий этап. Первоначальное поручение разрешало
  подготовку решения; исполнение отдельно открыто последующим ответом владельца ниже.
- Контракт: [accepted S1.15](0055-s115-saved-proposal-diagnosis.md), D1–D4 integrated;
  [offline evidence](0055-s115-evidence.md), [post-merge №94](0055-pr94-postmerge.md).
- Предлагаемый verifier SHA: `703b6c6ac960c21c0b7ab7bd84b3411459204c92`.
- Предлагаемый ref: `codex/statement-saved-proposal-diagnosis`, сохранён на этом SHA.
- Source CI: [39639515724](https://github.com/QahNaaRlN/grace/actions/runs/39639515724),5/5 PASS;
  source интегрирован PR №94, merge17e3493; полный tree identical подтверждён.
  Merge CI39664307328=5/5 и Delivery39665622283=4/4 PASS.
- Наличие и доступность private saved artifacts на2026-10-23: UNVERIFIED, VPS не читался.
- Approval record: 2026-10-23, прямое «влил PR, продолжить работу» после предъявления PR №95 и конкретного D5 source703b6c6/ref/границ. Разрешён ровно один описанный dispatch; это отдельный ответ владельца, а не вывод из merge/CI/D1–D4 acceptance.

## Принятое решение владельца

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

Это inputs единственного разрешённого dispatch. До его выполнения
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
