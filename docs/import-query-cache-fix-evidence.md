# Import navigation regression — SPEC-0017

Scope: repair the accepted SPEC-0017 review workflow; no new financial behavior
or API changes. On 2026-10-08, opening an import row and then `/accounts`
reproduced `e.map is not a function` in production.

The import form cached API list envelopes under the same query keys used by
the main screens for item arrays. Accounts, categories, and financial events
now use separate `import-options` child keys. Existing prefix invalidation
continues to refresh those queries after account creation and confirmation.

| Acceptance criterion | Executable evidence |
|---|---|
| SPEC-0017 AC8: review UI coexists with financial screens | `ImportsPage.test.tsx`, `keeps home list caches compatible when reviewing imports`: cold and preloaded caches, account/category/event options, then array consumption by main screens |
| Draft edits retain separate publication | Existing date-only, revision-refresh, upload, and confirmation tests remain passing |
| Contract and financial logic unchanged | Only query keys and regression tests change; generated schema has no content changes |

Validation on 2026-10-08:

- `bash scripts/test-windows.sh unit`: PASS (`make test`, 41 web tests,
  lint, production build, Go packages, offline Jev checks, `go vet ./...`).
- `git diff --check`: PASS.
- Initial sandbox run could not connect to local test servers. Re-running with
  local test-server access through the project Windows adapter passed.
- Production deployment and navigation verification: pending.

No source financial data, import decisions, credentials, or statements are
included in this change.
