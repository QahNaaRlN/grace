# SPEC-0056 Q4.2: фактический прогон — REJECT

2026-10-25, Asia/Yekaterinburg. [Accepted R](0056-q4-pricing-repair.md),
[implementation AC/evidence](0056-q4-pricing-repair-evidence.md),
[immutable exact-source receipt](0056-q42-owner-receipt.md).
Разрешение владельца «Принять и довести до прогона» исполнено:
repair → gates → merge → один новый Q4.2. Grant **CONSUMED**.

## Итог

**REJECT**, первый case `cards`, reason `quality_reject`, единственный
recognition check `protocol` FAIL. Модель была вызвана и вернула ответ;
metadata/pricing preflight теперь пройден. Ответ не прошёл DirectJSON/VisualReading
protocol validation; из публичной projection нельзя различить конкретную ошибку
JSON/структуры/валидации. Поля и строки этим case не были оценены:
это не доказательство конкретной ошибки amount/date/description.
Остальные7cases штатно `not_sent` после decisive quality reject.

Workflow success означает успешные transport/report/replay/cleanup gates,
а не распознавание PASS. Campaign decision остаётся REJECT. Качество всей восьмёрки,
full32 native/scan и чеков не измерено; production qualification/activation
и financial import не разрешены. `importReady=false`, `activationAuthorized=false`.

| Факт | Actual evidence |
| --- | --- |
| Run | [40191109474](https://github.com/QahNaaRlN/grace/actions/runs/40191109474), workflow_dispatch, attempt1, completed/success |
| Время | created2026-10-25T11:58:44Z (16:58:44+05:00), updated11:59:44Z; job56s; campaign9531ms |
| Counters |1POST /6metadataGET,≤8/6; repeat/rerun0; exact-source dispatch count1 |
| Bill |5183USD micros = **$0.005183**, settled; unknown bills0, overCap=false |
| Первый ответ | input5551/output2216tokens, duration6702ms; `cards` rejected/quality_reject/protocol FAIL |
| Остальные | columns, cards-photo, full32-native, full32-scan, receipt-discount, receipt-pdf, receipt-unreadable: not_sent/decisive_quality_reject/not_reserved |
| Evidence | [validated public summary](../research/statement-quality/q42-40191109474.summary.json), [run identity](../research/statement-quality/q42-40191109474.run.json) |

## Exact-source цепочка

- Contract acceptance до кода: `e0fbfa4b0d3d0fbc7174dd7eae3b1cbee9aa49fe`.
- [PR126](https://github.com/QahNaaRlN/grace/pull/126) merged2026-10-25T11:47:05Z;
  tested head `3ca3d64a788c6109a5f774f475e9317efc6e670b`,
  [PR CI40189521830](https://github.com/QahNaaRlN/grace/actions/runs/40189521830)5/5 PASS.
- Source `b1d310ed8746d4c1159d0b345c4acc3a7d8d3fdf`, tree равен tested head;
  clean main build, vcs.revision совпадает, vcs.modified=false.
- [Main CI40190313336](https://github.com/QahNaaRlN/grace/actions/runs/40190313336)5/5 PASS,
  включая make test (339Web), make test-production и make test-integration.
  Local make test/go vet/diff check PASS до публикации repair.
- [Delivery40191042047](https://github.com/QahNaaRlN/grace/actions/runs/40191042047)4/4 PASS;
  deployment не выполнялся.
- Manifest `28f1e68838743a597203a938e0a36b80bf60c459d24e26614e86c08f307aded5`.
- Corpus `268d0c2d49ea0803a2b9cffd619448ed9c9304f493d0db4e567a39472a3efa80`.
- Immutable owner receipt commit `75ebae2390d1fef818e5da8da9b3257e0316b909`
  опубликован до dispatch; source branch `codex/document-q42-source-b1d310e`
  указывает exact source. SHA-only dispatch не использовался.
- Model `google/gemini-3.1-flash-lite`, canonical
  `google/gemini-3.1-flash-lite-20260507`, endpoint `google-vertex/eu`/Google.
  Ceilings275000/1650000micros/M, request1000/image1micros; mode router_ceiling_only.
  Corpus/prompts/schema/scorer и порядок не менялись; reserve/limits прежние по R.

## Проверка projection и хранение

Скачан только artifact `document-reading-q-closed-result`, без private raw payload.
Локальный binary из clean exact source выполнил:

```sh
/tmp/grace-q42-b1d310e-eval -mode=validate-public \
  -bundle=/tmp/grace-q42-b1d310e-bundle \
  -authorized-source=b1d310ed8746d4c1159d0b345c4acc3a7d8d3fdf \
  < /tmp/grace-q42-40191109474-public/document-reading-summary.json
```

PASS. Workflow wrapper до projection выполняет summary через actual offline Replay
на VPS; successful summary/validation gates сохранены в run. Локально private Replay
не выполнялся и report не выгружался. Projection sha256
`0bbb94fe7989e914ff92f820ef649e84288a7a6f9522369e43e30398b0580cf8`;
private report object digest из projection
`444feb9ddf89d4b8a13c532a6638a65fdfab271e6932747dd4fa8693ab2b7329`.
Эти hashes связывают артефакты; не являются независимой криптографической аттестацией.

Raw report `/opt/grace/shared/document-reading-spec0056-q2.private.json` и
consumed marker `/opt/grace/shared/document-reading-spec0056-q2.started` остаются VPS.
Штатный cleanup вызван wrapper; отдельная remote проверка удаления staging
не выполнялась. Container identity до/после совпала по guard.
Старые Q4 marker/report не сбрасывались; первоначальный run40186488958 остаётся
INCONCLUSIVE0POST/3GET/$0 и CONSUMED. Дополнительных root SSH/SQL/key probes,
private payload downloads, повторов POST и financial writers не было.

R1–R5 выполнены repair/CI; R6 выполнен фактическим bounded run и public evidence.
Приёмка качества/активация не заявляются. Новый live grant не открыт;
дальнейший точный разбор protocol failure возможен отдельным read-only пакетом
по сохранённому report, без повтора распознавания.

## Последующий read-only разбор

По отдельному запросу владельца «разбор REJECT» установлены два нарушения
evidence:23 boxes вне0..1 и два непустых поля второй строки без цитат.
[Причина, causal controls и границы вывода](../research/statement-quality/q42-reject-diagnosis.md).
Исходный REJECT/counters/bill/grant не изменены; новых provider calls0.
