# SPEC-0056 S/U: попытка2из3, Q4.4 — exact-source receipt

Основание: [accepted S](0056-evidence-alignment-series.md), acceptance eda80af,
и [accepted U](0056-q43-owner-activity-reconciliation-candidate.md), acceptance423e7ff.
Владелец разрешил оставшиеся2из3 после одной policy reconciliation Q4.3,
с прежними8 документами/model/criteria и совокупным $2. Новая серия не создаётся.

**AUTHORIZED_ONCE**, после accepted dispatch **CONSUMED**, включая preflight/
transport отказ. Receipt immutable. Workflow rerun/повтор этого source запрещены.

## Проверенный source

- Q4.4 source `a8752028463d19351c9eddcb0ee40b71fe398b47`, integrated PR130,
  tested head709e8bbfacd7095ba8efbe9e8b1a74f5fbd76e58/
  [PR CI40222097044](https://github.com/QahNaaRlN/grace/actions/runs/40222097044)5/5 PASS.
- [Source main CI40223309245](https://github.com/QahNaaRlN/grace/actions/runs/40223309245)5/5 PASS.
- U implementation PR131, tested headd692de9f0b98f510a2e9aa8a45f5cc43d10096f6/
  [CI40224022530](https://github.com/QahNaaRlN/grace/actions/runs/40224022530)5/5 PASS,
  merge `200e719f1cf43254e27e37a88201ae311b976d29`. Model runtime/CLI/wrapper/workflow идентичны Q4.4 source;
  source является ancestor main. U добавляет ops helper/docs и не меняет runtime.
- Local U make test339Web/test-production/test-integration/vet/diff PASS;
  isolated U SQL/privacy controls11/11 PASS/skipped0. PR130 local gates PASS.
- Binary build из clean source, Go1.24.0/CGO0, vcs.revision=source,
  vcs.modified=false. Offline manifest совпал также при Go1.27.3 build.
- Manifest SHA256 `1639957c4398c15733643b315b7a185c82fc9fdfad2c981517c1cf2ad012f3fc`.
- Corpus268d0c2d49ea0803a2b9cffd619448ed9c9304f493d0db4e567a39472a3efa80;
  protocol e15ab2b29344ab9445537ab204842d012fb11329f23ad1ef26146a4babacdf80.
- DocumentPrompt v3/inline VisualSchema0..1/quotes1..8; oracle/validator/scorer/
  corpus/rates/model unchanged. Изменение compatibility остаётся гипотезой по HTTP400.

## Предыдущая попытка и бюджет

Consumed1/3: Q4.3 run40204752709/attempt1, decisionINCONCLUSIVE/HTTP400,
1POST/6GET, historical unknownBills1. [Summary](../research/statement-quality/q43-40204752709.summary.json)
SHA2561075c23708c10ee7f650b619bf9a652d3260d5b8cfb750becc4882f6d9e9182b,
reportc6f93f6246c9e37612c7861865fdab606b700184130aa8d4427e79fa9fad5d4f.
Original summary не переписан, Q4/Q4.2/все прежние grants сохраняются CONSUMED.

[U result](../research/statement-quality/q43-U-settlement.json), file SHA256
8590d323008943b1512e488605276e3656101c0be6e476cbf48ad39769ceadab:
VERIFIED_OPERATION/settled0, basis owner_accepted_policy_and_activity,
directMetadataVerified=false; private audit SHA256
ce361cc90481153d32d704315ffc34a5c3a5ead469d0133928e4da0eb42ea2b1.
Q4.2 bill не переучтён. Административное закрытие допускается только accepted U.

Reconciled spend0, **contingency30329micros** остаётся внутри прежних2000000.
Остаток для новых campaigns1969671micros; следующий max285232:
0+30329+285232=315561≤2000000, обе оставшиеся max600793≤2000000.
Текущий [journal](../research/statement-quality/evidence-series-2026-10-25.json)
сохраняет historical unknown и отдельную U reconciliation; unresolved0 по U.
После accepted dispatch это будет consumed2/3, останется максимум1попытка.
Новый unknown/price/identity/privacy/settlement failure снова закрывает серию;
U не разрешает автоматическое settled0 следующего HTTP400.

## Неизменные runtime limits и transport

google/gemini-3.1-flash-lite / canonical20260507 / google-vertex/eu / Google;
ZDR/deny collection, no fallback/retries/switch. Input275000/output1650000micros/M;
request1000/image1micros, router_ceiling_only. ≤4images/2048px/image4200tokens,
input30000/output16000, HTTP180s, ≤8POST/6GET/$0.25call/$2campaign.
Actual request reserves255817micros; conservative267205,8max285232.
Порядок: cards,columns,cards-photo,full32-native,full32-scan,receipt-discount,
receipt-pdf,receipt-unreadable. Full32по297checks; критерии не меняются.

Один dispatch document-reading-eval.yml через
`codex/document-q44-source-a875202`, ref указывает exact source, prior dispatch0.
Inputs authorized_source=source, approved_manifest=manifest, approval_id=SHA
этого опубликованного receipt commit, execute_approved_reading=true.
Все attempt1/count1/source ancestry/CI5/5/manifest/defaultfalse gates сохранены.

≤2pinned root SSH+≤1SCP через прежние secrets/timeouts S.
Новые staging `/tmp/grace-document-reading-q-a8752028463d19351c9eddcb0ee40b71fe398b47`,
marker `/opt/grace/shared/document-reading-spec0056-q4.started`, private report
`/opt/grace/shared/document-reading-spec0056-q4.private.json`, shared lock прежний.
Старые q/q2/q3 и reconciliation audits сохраняются; no-clobber/cleanup/replay,
private raw остаётся VPS, наружу только validated public summary.
После quality REJECT допустимы ≤2 дополнительных fixed-file read-only SSH по S;
без SQL/key/provider calls и без выгрузки private payload.

Стоп на первом actual PASS_SYNTHETIC8/8, иначе оставшийся1slot только при
разрешённом evidence repair и всех gates. Activation/import/deploy не открываются.
