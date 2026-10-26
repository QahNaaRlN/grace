# SPEC-0059 evidence

Accepted contract: [0059-real-receipt-photo.md](0059-real-receipt-photo.md), acceptance
recorded before implementation and integrated by PR141/main `c965958`.
Date: 2026-10-26. This report contains no owner photo, receipt contents, private
baseline, generation identifier or credentials.

## Offline implementation

Only `internal/assistant/model_groups.go` changes production behavior: a separate
`receipt_router_ceiling_only` operator opt-in for receipt/document_reading, one
image, positive request ceiling ≤1000 USD micros. An omitted request price is
unknown, not free. Existing plain/research modes and other purpose/profile bounds
remain distinct. Present invalid request prices still fail. The same existing
reservation and provider wire code includes the whole request/image ceilings.
No prompt, receipt schema, API/DTO, financial write policy or DB migration changed.

- A1 PASS: `TestReceiptProductRequestCeilingAdmissionAndWire`,
  `TestReceiptRequestCeilingCannotExpandOtherProfiles`,
  `TestReceiptMissingRequestPriceRequiresOperatorOptIn`. Omission requires an
  explicit bounded opt-in; null/empty/number/negative/malformed/over-cap reject.
  Missing image price, ZDR and excessive token pricing reject before POST.
  Other purposes/profiles, multi-image opt-in, zero/excessive cap and experiment
  admission reject. Existing `TestPlain*` and `TestR*` also pass.
- A2 PASS: same admission/wire test uses actual receipt prompt/schema and actual
  grouped adapter with fake transport: positive ceilings fully reserved, explicit
  provider and request/image caps sent, fallback false, exactly 3 metadata GET and
  1 fake POST. Preparation sends no POST; receipt schema is retained.
- A3 PASS: `./scripts/test-document-reading.sh`; real PostgreSQL
  `TestS56ReceiptDocumentLifecycle/image_receipt_ceiling` alongside legacy image
  and PDF cases. Missing request price + bounded opt-in flows through upload,
  source preview, consent, single fake extraction, idempotent replay, zero
  postings before confirmation, isolated synthetic confirmation/replay, image
  deletion and known settled usage. SQL reserve equals displayed reading plan;
  request cap is present on actual fake-provider wire. Other document/SQL/corpus
  cases in the script pass. Production financial writes=0 at this stage.
- A4 local PASS: `TMPDIR=/var/tmp/grace0058-tests make test` (341 Web tests/91 files,
  Go, lint, generated API/build, offline Jev/corpus), `make test-production`,
  `go vet ./cmd/... ./internal/...`, `git diff --check`. External TMPDIR is needed
  because this host's `/tmp/.git` marker correctly triggers privacy guards.
  Post-merge local make test/vet/diff checks also PASS on main dab06a9.
  PR142 CI40324354649/all5, main CI40325834019/all5, Delivery40327297193/all4
  and Deploy40327460999 succeeded for exact runtime dab06a9.
- A5 observed: new product reading returned all9 rows and persisted/reloaded.
  Private pre-inference reference/comparator were sealed by SHA256; comparison
  before any manual edits gives **79/80**, with exactly one issuedAt contract
  mismatch. All item names/quantities/prices/totals, seller, calendar date,
  currency, fiscal document number and printed sourceText clock matched.
  Unprinted discount/brand/package/broadGroup values remained null. The model
  returned the correct local clock as HH:MM in issuedAt, where accepted contract
  requires null without timezone. This is RAW_CONTRACT_MISMATCH, not full PASS.
  No manual edits were made; current fields still equal immutable extraction.
- A6 PASS for execution/accounting: exactly one receipt usage row, reserve29642
  USDmicros, actual3055 USDmicros ($0.003055), settled; unsettled0.
  Financial event/posting/receipt deltas0. Source trial closed after mismatch;
  grant CONSUMED1/1, no retry. No claims about unrelated photos or final real
  financial confirmation are made.

## Production execution and closed grant

One ordinary authenticated product browser path: existing owner photo upload →
source viewer → model/cost reading preview (one page, displayed maximum$0.0296) →
explicit consent → one inference → extracted draft/reload. The private reference
was never sent to the provider. Runtime API/Web/PDF helper use exact main
`dab06a967c255ec1c3c2862aeda0e19858b1b5d1`; operator configuration changed only
receipt membership to the accepted Gemini3.1/Google Vertex EU and fee evidence.
Password auth, AI enable, analysis/statement groups and purchased allowance were
preserved. Protected before-config files are retained for rollback.

The standard Deploy production workflow verified a DB backup, migrations,
internal health and public smoke before receipt configuration was applied.
Only API was then recreated with the same SHA; public entrypoint/auth smoke PASS.
Fresh receipt capability proof used the shared three-GET resolver and the
actual product request retained its receipt prompt/schema and existing
reservation/no-fallback/no-retry path. Inference ran once on 2026-10-26;
known settlement completed at13:09:16 UTC (18:09:16 Asia/Yekaterinburg).

The temporary first-test qualification was closed/expired after the raw
contract mismatch; group revision spec0059-v1-closed. API remains healthy,
analysis/statement unchanged. This prevents new receipt inference under the
consumed grant. The extracted draft/photo remain available for manual review.

## Discovered product gaps

1. The receipt decoder accepts a clock-only issuedAt string and only the later
   financial validator rejects it. The UI shows a date-only form, hiding the raw
   clock; editing the date can clear it, but this is not an automatic repair.
2. A correctly recognized free service row cannot be saved unchanged: Go draft
   validation and receipts.ValidateCommand, OpenAPI item minima and SQL CHECKs
   all require positive unit/line amounts. The positive overall purchase total
   and exact sum are not the problem. The row was not silently removed.

[Draft R](0059-receipt-normalization-candidate.md) proposes bounded deterministic
candidate normalization with original evidence retained, nonnegative child item
amounts under a positive purchase total, and verification using this saved
response plus public SQL fixtures. It is not accepted/implemented by this report.
No new model calls are needed or authorized by the candidate.


## Accepted R follow-up — integrated and deployed

The candidate R referenced above was subsequently accepted by the owner and
integrated in PR144 before implementation. PR145/main `da621d9` repairs only the
deterministic candidate/receipt rules and source text display. The existing real
response replays as candidate80/80 while its original79/80 extraction remains
unchanged. All9rows are retained, including the free row. No new provider POST,
AI usage or real financial writes were performed. The original inference grant
remains consumed and its known settlement is unchanged.

The same receipt route is qualified until2026-11-02 22:55 Asia/Yekaterinburg under
accepted R, with previous ceilings and separate per-reading consent/financial
confirmation; no automatic renewal. [Final R verification](0059-normalization-evidence.md)
records exact-SHA CI/Delivery/deploy, backup, migration32, UI and SQL evidence.
