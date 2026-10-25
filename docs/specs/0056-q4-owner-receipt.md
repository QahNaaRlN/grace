# SPEC-0056 Q4: отдельное разрешение владельца

Дата: 2026-10-25, Asia/Yekaterinburg. Статус: **AUTHORIZED_ONCE** до первого
workflow dispatch; при dispatch, включая preflight/SSH failure, **CONSUMED**.
Основание: прямое сообщение владельца «даю отдельное разрешение» после merge
PR124 и отчёта об отдельном разрешении на реальное распознавание Q4.
Это разрешение исполняет [accepted Q-контракт](0056-openrouter-document-qualification.md)
и не изменяет исходники, scorer, corpus, prompts/schema или продуктовую policy.

## Зафиксированный допуск

- Exact source/ref: `9200b731a0dea8f171e31aaf988d038073ad577f`, интегрированный main PR124.
- Source CI: [40185076058](https://github.com/QahNaaRlN/grace/actions/runs/40185076058), completed/success, все5 jobs PASS.
- Manifest SHA256: `10200604c2a134577b3d22eed84fd106d7d951b1aececa0ec47f978dbf85653a`.
- Corpus digest: `268d0c2d49ea0803a2b9cffd619448ed9c9304f493d0db4e567a39472a3efa80`.
- Model: `google/gemini-3.1-flash-lite`; canonical proposal `google/gemini-3.1-flash-lite-20260507`.
- Endpoint: `google-vertex/eu`, provider `Google`, region `eu`; ZDR и deny data collection обязательны.
- Ceilings: input275000 /output1650000 USD micros за миллион tokens ($0.275/$1.65).
- Дополнительные request/image fees≤0; отсутствие явных rates либо превышение ceilings →отказ.
- Images≤4, каждая сторона≤2048px, резерв4200 input tokens/image.
- Input≤30000/output≤16000; provider HTTP timeout180s, no retries/fallback/model/provider switch.
- Максимум8POST /6metadataGET, ≤250000 USD micros/call ($0.25), ≤2000000 total ($2.00).
- Offline actual request reserves=246235micros ($0.246235); реальный существующий shared budget
  проверяется только штатным preflight. Никакого увеличения allowance/новых budget rows.

Порядок восьми публичных synthetic документов: cards, columns, cards-photo,
full32-native, full32-scan, receipt-discount, receipt-pdf, receipt-unreadable.
Все source/page/text/golden hashes и page bounds уже связаны указанным manifest.
Частные документы, пользовательские consents и финансовые записи исключены.

## Исполнение и transport

Разрешён ровно один dispatch `.github/workflows/document-reading-eval.yml`
на ref exact source выше, `execute_approved_reading=true`, attempt1.
`approval_id` — SHA этого опубликованного receipt commit. Предварительная
проверка GitHub показала0 Q4 dispatch для source. Исторические grants не используются.
Повтор dispatch, rerun/attempt2, удаление или изменение marker запрещены.

CI до production staging сверяет exact source, main ancestry, CI5/5 и manifest.
Metadata≤6GET и все проверки ключа/endpoint/schema/context/ZDR/region/fees/
identity/общего бюджета выполняются существующим preflight внутри одного run.
Невозможность подтвердить параметры →INCONCLUSIVE/отказ, без альтернативной модели.

Pinned host `root@195.133.195.190`, dedicated существующий SSH key и operator UUID
используются только workflow через существующие GitHub production secrets.
≤2 root SSH command sessions: создать новый staging каталог; выполнить pinned wrapper.
≤1 SCP staging transfer. ConnectTimeout15s, ServerAliveInterval30s/CountMax60.
Workflow timeout40min, remote timeout1810s (TERM, kill10s), campaign30min;
grant35min, summary60s. Secrets/UUID/DSN/env значения не выводятся и не сохраняются.
Новых keys/secrets/bootstrap, container image/config/service changes, рестартов,
migrations и произвольных production SQL запросов нет.

Новые paths:
- staging `/tmp/grace-document-reading-q-9200b731a0dea8f171e31aaf988d038073ad577f`;
- lock `/opt/grace/shared/statement-screening.lock`;
- marker `/opt/grace/shared/document-reading-spec0056-q.started`;
- private raw report `/opt/grace/shared/document-reading-spec0056-q.private.json`.

Marker создаётся no-clobber до remote preflight. Raw report и replay остаются
на VPS; наружу выходит только allowlisted summary. Staging удаляет штатный
cleanup; marker/report сохраняются. Дополнительные root SSH diagnostics и
скачивание private payload этим допуском не разрешены.

После каждого POST previous bill settled либо unknown/frozen. Timeout,
identity/privacy/price/billing/settlement нарушение или первый decisive quality
REJECT останавливают кампанию; остальные cases not_sent. Неполный запуск не
объявляется PASS. Даже PASS_SYNTHETIC не разрешает production qualification,
activation, deployment или финансовый импорт; importReady=false.

## Evidence

Dispatch/run ID, фактические counters/cost/settlement и итог будут записаны
отдельно после исполнения. Этот receipt immutable и сам по себе не является
результатом распознавания или evidence работоспособности production доступа.
