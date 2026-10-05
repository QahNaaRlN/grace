# SPEC-0016: Implementation evidence

Accepted scope: [SPEC-0016](0016-spreadsheet-import-review.md), acceptance commit
5911963. Implementation commit a03fef6; cross-platform golden correction 7f3b404.

## Behavior delivered

`internal/importreview` and `cmd/import-review` review a bounded local snapshot
without provider/database/network wiring. Versioned input/output JSON schemas,
a synthetic ten-sheet example and [operator instructions](../import-review.md)
are included. Output preserves raw typed values, formulas, provenance,
snapshot-local IDs, normalized domains, controlled issues and reconciliation
candidates. It never merges records or publishes financial facts.

Money uses exact decimal-to-int64 arithmetic with overflow/precision rejection.
Dates retain confirmed timezone and precision; date-only/approximate serial
fractions do not invent an instant. Plans, reported debt, future payment
requirements and unavailable documents remain distinct. Bank balance controls
are independently checked with overflow-safe integer arithmetic. Unmapped
accounts/categories/funding remain explicit issues rather than fake entities.

CLI writes only to a newly created operator directory outside Git checkouts,
never overwrites, and uses Unix 0700/0600. Windows requires a private parent
ACL. Standard output contains counts and published=false; errors use controlled
codes. Real snapshots/reports remain outside Git; *.private.json is ignored.

## Executable verification

- `go test ./internal/importreview ./cmd/import-review`: PASS. Fixtures cover
  exact numeric/text/grouped amounts, overflow/subminor rejection, serial/local
  dates, timezone, null/zero, missing IDs, aliases, multiple recipients,
  date conflicts, computed empty/nonempty rows, unknown sheets, stale formula
  literals, separate debt/plan requirements, incomplete/mismatched balances,
  unavailable documents, input/resource limits, safe output/no overwrite and
  malicious source text as inert evidence. JSON/Markdown golden checks pass.
- `go test -cover ./internal/importreview ./cmd/import-review`: 85.2% each.
- Linux Docker `--network none` with Go 1.24:
  `go test -race ./internal/importreview ./cmd/import-review`: PASS, including
  CLI offline execution and Unix private permissions.
- `make SHELL=bash test` and `go vet ./cmd/... ./internal/...` in feature tree:
  PASS, including 20 Web tests, lint/build and Jev offline evaluation.
- Input and output schemas validate synthetic fixtures and private data using
  the existing Ajv validator with the shared draft-7 keyword subset adapter.
- `git diff --check`: PASS.
- Post-merge main `make SHELL=bash test`, `go vet ./cmd/... ./internal/...`
  and `git diff --check`: PASS after the Windows golden line-ending correction.
  Implementation commits are reachable from main; SPEC-0016 is integrated.

## Private source dry run

The previously captured owner-authorized workbook was converted locally to the
v1 contract, merging identical overlapping read coordinates; conflicting reads
would be rejected. No fresh provider access or document download was needed.
Final run reviewed 373 records across all ten sheet profiles, including 191
transaction rows and every one of the 50 additional rows. It retained four text
money values, two missing IDs, 41 reconciliation candidates and explicit date
conflicts/no-candidate rows. Wider secondary rules may retain extra date-conflict
candidates even when another candidate matches the date; none are confirmed.

Reports show missing canonical mappings as issues because no account/category/
facility assignment was invented. All results were written to the private
operator workspace; no amounts, personal records or PDF files are committed.
Financial ledger, Google workbook and production deployment are unchanged.
