# SPEC-0014 evidence: Telegram production configuration

Accepted specification commit: a232cf5. Implementation: 767c026.
Production candidate and deployed SHA: 2bafa878453f9c69c136a0d9c697cd1c94c20c18.

## Verified

- make test: Go, 20 Web tests, lint, build, Jev offline checks PASS, both feature and primary checkout.
- make test-production: Linux container PASS, plus GitHub CI. Checks include an outbound API network, only Caddy published ports, rejection of public credential-file permissions, loading of a private environment before mock Docker, absence of test credentials in output.
- git diff --check: PASS.
- CI https://github.com/QahNaaRlN/grace/actions/runs/37348399683: success, including all PostgreSQL integration suites.
- Delivery https://github.com/QahNaaRlN/grace/actions/runs/37348789806: success.
- Deploy https://github.com/QahNaaRlN/grace/actions/runs/37348976487: completed/success.
- Live current link and API/Web images match deployed SHA; API/Web healthy, readiness succeeds.
- schema_migrations: 6, dirty=false; eight owner_access policies.
- Counts and JSONB checksums of accounts/events/postings/receipts/items/obligations/payments/settlements match before/after, excluding added owner_id. Snapshots remain private on VPS, financial records were not exported.
- Production API wget to Telegram JWKS succeeds. Anonymous internal accounts/me requests return 401; public Basic Auth boundary checked by deployment workflow.
- Fresh backup restored successfully into a separate temporary database before migration; test database removed.
- Runtime telegram.env: operator-owned, mode 600, loaded by deploy.sh; values never included in Git/release artifacts/evidence.

## Live acceptance (2026-10-06 Asia/Yekaterinburg)

After the SPEC-0015 JWKS compatibility repair, owner personally completed Telegram login and confirmed success. Database checks show one identity linked to an enabled admin, one active non-revoked session and one consumed invitation. No Telegram claims, session tokens or financial data were read for this check. First owner login and the accepted rollout criteria are complete; SPEC-0014 is integrated. Manual authenticated financial-screen inspection and live logout were not separately observed; authorization, isolation and logout behavior are covered by automated tests.