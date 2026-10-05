# SPEC-0018 implementation evidence

Accepted specification: [SPEC-0018](0018-public-telegram-entrypoint.md).
Branch: `codex/public-telegram-entrypoint`. Date: 2026-10-06.
Production publication and post-merge verification remain pending.

## Change

Production Caddy no longer requires browser Basic Auth; Compose no longer reads
or mounts its shared password hash. Existing Telegram sessions, invite-only
identity binding, CSRF/Origin and RLS are unchanged. Deployment checks public
Web/login/meta and session-protected me/accounts with fresh anonymous requests.
Stored operator secrets are retained for rollback; no private invitation is used.

## Acceptance evidence

| Criterion | Executable evidence | Result |
| --- | --- | --- |
| AC1 | Actual `caddy:2-alpine`, production Caddyfile with only test upstream addresses, `scripts/check-public-entrypoint.sh http://127.0.0.1:<ephemeral-port>` | `/`, `/login`, `/api/meta`: 200, correct content type/body, no Basic challenge |
| AC2 | Same proxy smoke; `make test` auth/server/Web regressions | `/api/auth/me`, `/api/v1/accounts`: 401 JSON unauthenticated, no Basic challenge; auth unit tests pass. PostgreSQL integration remains a CI gate |
| AC3 | `make test-production` initial Compose config and published-port checks; only postgres_password test file exists | Compose config validates without Basic hash. Published-port/network check is included in Linux gate; credentials/runtime code unchanged |
| AC4 | `.github/workflows/deploy-production.yml` calls shared anonymous checker | Code ready; CI/Delivery and live HTTPS smoke pending |
| AC5 | `make test` via Git Bash, `go vet ./...`, `git diff --check` | PASS. Linux `make test-production` pending CI |

Local `make test` completed all Go packages, 20 Web tests, lint, production Web
build and offline Jev validation (49 v1 and 72 v2 cases). OpenAPI is unchanged;
generated-file line-ending noise from the build was restored.

The complete production script cannot run unchanged on Windows: its existing
backup test relies on a Unix symlink (`no current Grace release` on Git Bash).
The new proxy regression was executed separately with local Docker/PowerShell
and the exact shared shell checker. CI runs the full script on Ubuntu.
No claim of successful production rollout is made before deployment.

## Production baseline

Before repair, fresh HTTPS requests to `/`, `/login` and `/api/auth/me` return
401 with `WWW-Authenticate: Basic realm="restricted"`. The reported minute
delay was not independently reproduced. No arbitrary timeout changes were made.

## Integration gate

Automatic approval review rejected a direct push to main and then feature-branch
publication, requesting explicit owner authorization for GitHub egress. The patch
is committed locally; no PR or production release has been published. Publication,
merge and deployment require explicit owner authorization. After authorized integration, record exact SHA, CI/Delivery/deploy
run links and anonymous live smoke here; only then mark SPEC-0018 integrated.
