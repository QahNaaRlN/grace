# SPEC-0018 implementation evidence

Accepted specification: [SPEC-0018](0018-public-telegram-entrypoint.md).
Branch: `codex/public-telegram-entrypoint`. Date: 2026-10-06.
Implementation commits: 0d7138f and release-promotion correction 9640c51.
Production release: f9875f525b6be06eb7cf08700e67201c539e845b, verified 2026-10-06.

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
| AC2 | Same proxy smoke; `make test` auth/server/Web regressions and CI PostgreSQL integration | `/api/auth/me`, `/api/v1/accounts`: 401 JSON unauthenticated, no Basic challenge; session/ownership tests PASS |
| AC3 | `make test-production` initial Compose config and published-port checks; only postgres_password test file exists | Compose config validates without Basic hash. Published-port/network check is included in Linux gate; credentials/runtime code unchanged |
| AC4 | Deploy and VPS release-promotion checks call the bundled anonymous checker | CI/Delivery/deploy and fresh live HTTPS smoke PASS |
| AC5 | `make test`, `go vet ./...`, `git diff --check`, Linux `make test-production` and `make test-integration` in CI | PASS; promotion regression exercises actual deploy.sh against real Caddy |

Local `make test` completed all Go packages, 20 Web tests, lint, production Web
build and offline Jev validation (49 v1 and 72 v2 cases). OpenAPI is unchanged;
generated-file line-ending noise from the build was restored.

The complete production script cannot run unchanged on Windows: its existing
backup test relies on a Unix symlink (`no current Grace release` on Git Bash).
The new proxy regression was executed separately with local Docker/PowerShell
and the exact shared shell checker. CI runs the full script on Ubuntu.
The full Linux production checks passed in PR and exact-main CI.

## Production baseline

Before repair, fresh HTTPS requests to `/`, `/login` and `/api/auth/me` return
401 with `WWW-Authenticate: Basic realm="restricted"`. The reported minute
delay was not independently reproduced. No arbitrary timeout changes were made.

## Integration gate

Automatic approval review initially rejected publication. On 2026-10-06 the
owner explicitly authorized publishing to QahNaaRlN/grace, merging after checks
and deploying graceapp.ru. [PR #4](https://github.com/QahNaaRlN/grace/pull/4)
removed Basic Auth. Its first rollout automatically rolled back because the
legacy VPS health loop still expected 401 from `/`.
[PR #5](https://github.com/QahNaaRlN/grace/pull/5) replaced that loop with the
bundled public/session-protected checker and added a release-promotion regression.
The internal check pins localhost via curl --resolve and verifies TLS.

## Successful production verification

- Exact release SHA: `f9875f525b6be06eb7cf08700e67201c539e845b`.
- [Main CI](https://github.com/QahNaaRlN/grace/actions/runs/37389117231): PASS,
  including quality, production bundle/promotion and PostgreSQL regressions.
- [Delivery](https://github.com/QahNaaRlN/grace/actions/runs/37389572696): PASS,
  both exact-SHA images published.
- [Production deploy](https://github.com/QahNaaRlN/grace/actions/runs/37389737686):
  PASS, including pre-deploy backup, internal health/promotion and public smoke.
- Independent `scripts/check-public-entrypoint.sh https://graceapp.ru`: PASS
  after completed deploy. Fresh `/`, `/login`, `/api/meta` return 200; fresh
  `/api/auth/me`, `/api/v1/accounts` return 401 JSON; none has a Basic challenge.
- Browser after final successful deploy: Telegram button and invitation-code
  field are visible without old login/password. Local screenshot:
  ignored `bin/live-login.png`. No session, invitation or financial data used.
- Fresh login HTML response: HTTP 200 in 0.566450 seconds. This measures HTTP
  response, not the duration of another person's Telegram authorization.

The minute-long invitation incident was not independently reproduced. The
confirmed external Basic Auth barrier and its obsolete release health check are
removed; no arbitrary OIDC timeout or retry change was made.
