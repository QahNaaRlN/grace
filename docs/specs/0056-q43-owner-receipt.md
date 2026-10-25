# SPEC-0056 S: попытка1/3, Q4.3 — exact-source receipt

2026-10-25, Asia/Yekaterinburg. Основание: [accepted S](0056-evidence-alignment-series.md),
владелец «согласуй, затем прогон до положительного результата» и
«До3 попыток / $2 суммарно». Acceptance до кода eda80af.
Полномочия охватывают repair/gates/merge/до3 bounded runs без новых подтверждений.

**AUTHORIZED_ONCE**, после accepted dispatch **CONSUMED**, в том числе при
preflight/SSH failure. Этот receipt immutable; остальные grants/failed results сохраняются.

## Source, gates, manifest

- PR129 integrated после exact-head CI40202818845/all5 PASS.
- Tested head db8c4256314d81497acb1a8476157346b608c44c.
- Integrated source `a49542e2db7522adf30171ef2cf4b2198016fa67`, main CI40203746305/all5 PASS.
- Manifest SHA256 `8f378b10d708075d5a3badcb4cf3f7d016c967f5ff86c8854c0c8b541e1de3da`.
- Corpus digest268d0c2d49ea0803a2b9cffd619448ed9c9304f493d0db4e567a39472a3efa80.
- Binary clean build, vcs.revision=source, vcs.modified=false; workflow перепроверяет manifest.
- DocumentPrompt v2 и aligned VisualSchema; validator/scorer/goldens/corpus unchanged.

## Series budget до dispatch

Новых consumed attempts0/3; known spend0micros/$2; unknown bills0.
Следующая максимальная reservation285232micros≤2000000remaining.
Максимальная сумма трёх healthy cap-compliant campaigns855696micros<$2.
Это reservation bound; anomalous bill/price/identity/privacy/unknown settlement
останавливает всю серию и фиксируется без retry/top-up.
Исторические Q4/Q4.2 CONSUMED и не относятся к новым3slots; не переиспользуются.
Текущий [series journal](../research/statement-quality/evidence-series-2026-10-25.json)
после dispatch получит runID и fact counters; до следующей попытки требуется
validated summary, settled bills и проверка sum(spent)+nextMax≤2000000.

Model google/gemini-3.1-flash-lite, canonical google/gemini-3.1-flash-lite-20260507;
endpoint google-vertex/eu, provider Google, ZDR/deny collection.
Input275000/output1650000micros/M; request1000/image1micros, router_ceiling_only;
image bound4200tokens,≤4images/2048px. Input≤30000/output≤16000;
≤8POST/6metadataGET/$0.25call/$2campaign, HTTP180s, no retries/fallback/switch.
Actual reserve247349micros, conservative campaign267205,8max285232.
Порядок8: cards,columns,cards-photo,full32-native,full32-scan,receipt-discount,
receipt-pdf,receipt-unreadable. Full32по297checks, oracle неизменный.

## Dispatch и transport

Один workflow_dispatch document-reading-eval.yml через branch ref на exact source;
inputs authorized_source=source, approved_manifest=manifest,
approval_id=SHA этого опубликованного receipt commit, execute_approved_reading=true.
Source ancestry main, exact CI5/5, attempt1/count1/defaultfalse gates обязательны.
Повтор dispatch/rerun запрещён; следующий slot только по accepted S после diagnosis/repair.

≤2 pinned root SSH commands +≤1SCP, существующие dedicated secrets;
ConnectTimeout15s, ServerAlive30s/CountMax60, workflow40min, remote1810s,
campaign30min/grant35min/summary60s. Нет deployment/restart/container config/image
changes, key/bootstrap/migrations/произвольных SQL/financial writers.

Новые paths:
- staging /tmp/grace-document-reading-q-a49542e2db7522adf30171ef2cf4b2198016fa67;
- marker /opt/grace/shared/document-reading-spec0056-q3.started;
- private report /opt/grace/shared/document-reading-spec0056-q3.private.json;
- общий /opt/grace/shared/statement-screening.lock.

Старые q/q2 marker/report сохраняются; no-clobber, raw/replay остаются VPS,
наружу только validated allowlisted summary, cleanup удаляет staging.
После REJECT allowed S read-only diagnosis≤2fixed-file SSH sessions без новых
provider/SQL calls; private payload не скачивается, сохраняются только closed flags.

Каждый POST settled либо unknown/frozen; после decisive quality REJECT кампания
останавливается, остальные not_sent. Новый evidence repair не меняет oracle.
PASS_SYNTHETIC требует actual8/8; после него серия останавливается немедленно.
Ни PASS, ни этот receipt не разрешают production activation/import/deployment.
