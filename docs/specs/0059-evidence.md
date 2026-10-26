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
  CI/Delivery/deploy and post-merge checks are pending.
- A5/A6 **UNVERIFIED**: no real inference has been performed. The uploaded owner
  draft is source-only, with no extraction/reading consent or financial link;
  shared usage has no unsettled reservations. Private baseline and comparison
  helper are outside Git; negative controls detect an omitted free row and a
  fractional-minor monetary mismatch. No quality result is inferred from these
  controls or earlier PDF success.

## Live bounds

Owner-approved grant available 1/1: same supplied photo through production UI,
Gemini 3.1 Flash Lite / Google Vertex EU, ≤1 provider POST, ≤$0.04 total including
reserved fees, ≤3 metadata GET for fresh receipt proof, no financial writes and no
retry after protocol/quality failure. Only receipt membership may change;
analysis/statement remain unchanged. Qualification is temporary ≤24h, quality
UNVERIFIED until comparison, no automatic renewal. Unknown billing retains the
reserve and blocks further POST; no administrative zero settlement.
