# SPEC-0056 Q4.2: exact-source receipt

Дата: 2026-10-25, Asia/Yekaterinburg. **AUTHORIZED_ONCE** до одного dispatch;
после первого accepted dispatch, включая preflight/SSH failure, **CONSUMED**.
Основание: владелец принял [SPEC-0056 R](0056-q4-pricing-repair.md) и выбрал
«Принять и довести до прогона», прямо разрешив repair → gates → merge → один Q4.2.
Acceptance receipt до кода: `e0fbfa4b0d3d0fbc7174dd7eae3b1cbee9aa49fe`.
Этот документ связывает условное разрешение с полученными после CI source/manifest;
не создаёт нового разрешения и не расширяет его.

## Источник и воспроизводимость

- PR126: https://github.com/QahNaaRlN/grace/pull/126 — MERGED после CI5/5.
- Tested repair head `3ca3d64a788c6109a5f774f475e9317efc6e670b`;
  CI https://github.com/QahNaaRlN/grace/actions/runs/40189521830,5/5 PASS.
- Exact integrated source: `b1d310ed8746d4c1159d0b345c4acc3a7d8d3fdf`; tree равен tested repair head.
- Main exact-source CI: https://github.com/QahNaaRlN/grace/actions/runs/40190313336,
  completed/success,5/5 PASS, включая make test/production/integration.
- Manifest SHA256 `28f1e68838743a597203a938e0a36b80bf60c459d24e26614e86c08f307aded5`.
- Corpus digest `268d0c2d49ea0803a2b9cffd619448ed9c9304f493d0db4e567a39472a3efa80`.
- Binary собран из clean exact source, CGO_ENABLED=0, trimpath, sourceSHA bind;
  build info vcs.revision совпадает, vcs.modified=false.

## Единственная кампания

Model `google/gemini-3.1-flash-lite`, canonical
`google/gemini-3.1-flash-lite-20260507`, endpoint `google-vertex/eu`, provider Google.
Token ceilings275000/1650000 USD micros/M; image token upper bound4200/image.
RequestFeeMicros1000 ($0.001/request), ImageFeeMicros1 ($0.000001/image).
`requestFeeEvidence=router_ceiling_only`: только absent request key, positive
operator/router cap полностью в reserve и max_price; это не evidence нулевой цены.
Explicit malformed/null/overcap request или непроверенная image цена отвергаются.

Порядок: cards, columns, cards-photo, full32-native, full32-scan,
receipt-discount, receipt-pdf, receipt-unreadable. Corpus/prompts/schema/goldens/
scorer неизменны; full32 по297 recognition checks. Synthetic public documents only.
≤4images,≤2048px/side;30000 input/16000 output;≤8POST/6metadataGET,
≤$0.25/call/$2.00 total; conservative preflight267205micros,8max285232micros.
Штатный shared ledger проверяет реальный remaining allowance; пополнений/новых rows нет.
No retries/fallback/model/provider switch. HTTP180s, proof≤24h.

Один workflow_dispatch `.github/workflows/document-reading-eval.yml` через
branch ref, закреплённый exact source выше; authorized_source=source,
approved_manifest=manifest, approval_id=SHA этого опубликованного receipt commit,
execute_approved_reading=true, attempt1. Source ancestry main и CI5/5 проверяет workflow.
Rerun/attempt2/second accepted dispatch запрещены. Grant расходуется независимо
от результата. Предыдущий Q4 run40186488958 остаётся CONSUMED и не повторяется.

## Transport и хранение

Только существующие dedicated production SSH key/operator owner ID/OpenRouter key
через действующий workflow; ≤2 pinned root SSH command sessions и≤1SCP staging.
Host root@195.133.195.190; ConnectTimeout15s, ServerAliveInterval30s/CountMax60;
job40min, remote1810s (TERM/kill10s), campaign30min, grant35min, summary60s.

- Staging `/tmp/grace-document-reading-q-b1d310ed8746d4c1159d0b345c4acc3a7d8d3fdf`.
- Shared lock `/opt/grace/shared/statement-screening.lock`.
- Новый no-clobber marker `/opt/grace/shared/document-reading-spec0056-q2.started`.
- Новый private raw report `/opt/grace/shared/document-reading-spec0056-q2.private.json`.
- Старые q marker/report не изменяются/не удаляются.

Raw report/replay остаются VPS; наружу только allowlisted public summary.
Staging удаляет штатный cleanup; marker/report остаются. Дополнительный SSH
diagnosis/private payload download этим receipt не открыт. Нет deployment,
container image/config/service changes/restart, keys/bootstrap/migrations,
финансовых writers или произвольного SQL чтения.

Каждый POST имеет settled known bill либо unknown/frozen. Первый decisive
quality REJECT/timeout/identity/privacy/price/billing/settlement failure останавливает
кампанию; остальные not_sent. Verdict только PASS_SYNTHETIC/REJECT/INCONCLUSIVE
из actual replay, без подмены partial успехом. Даже PASS_SYNTHETIC не означает
production qualification/activation или разрешения financial import.
Результат и расход grant записываются отдельно после запуска; этот receipt immutable.
