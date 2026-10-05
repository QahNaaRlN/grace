# SPEC-0014: Production configuration for Telegram login

- Статус: integrated
- Дата: 2026-10-05
- Acceptance owner: owner
- Acceptance record: owner requested Telegram configuration and explicitly authorized production publication/deployment after CI. This corrective scope makes the accepted SPEC-0013 usable on that target, without changing authentication or financial contracts.
- Implementation branch: codex/telegram-production-configuration
- Implementation commit: 767c026
- Verification: local and post-merge checks, CI/Delivery/deployment, migration and first owner login PASS; see [evidence](0014-evidence.md).

## Problem and result

Production API uses only the internal backend network; it cannot reach Telegram token/JWKS endpoints. The deployment workflow also does not load operator-owned Telegram credentials. Login must work after deployment and subsequent releases without opening API or PostgreSQL host ports.

## Scope and invariants

Connect API to the existing edge network for outbound HTTPS. Keep PostgreSQL on internal backend and publish only Caddy ports. Load optional /opt/grace/shared/secrets/telegram.env in deploy.sh before Compose operations; require a private readable operator-owned file and never print credentials. Missing file preserves locked defaults. Credentials remain on VPS and outside GitHub/release artifacts. Preserve Basic Auth, WireGuard, financial arithmetic, identity/session contracts and migration guards.

## Non-goals

Removing Basic Auth, public registration, a second identity provider, modifying host firewall or WireGuard, broad network/system upgrades.

## Acceptance and executable checks

- [x] Rendered API configuration has an outbound network; API/Postgres host ports stay private.
- [x] deploy.sh reads the protected optional Telegram environment without logging values; subsequent releases preserve operator configuration.
- [x] make test, make test-production, git diff --check; post-merge checks.
- [x] CI/Delivery succeeds for the exact deployment SHA; production API reaches Telegram JWKS after deployment.
- [x] Database migration, anonymous session rejection and first owner login are checked; interactive Telegram consent is performed by the owner.

## Rollout and rollback

Use the verified fresh production backup/restore drill before migration 000006. Deploy exact published images after CI. Existing application rollback preserves the database; never bypass the multi-owner down guard. Operator file changes and code deployment are separate; secrets are not embedded into release artifacts.
