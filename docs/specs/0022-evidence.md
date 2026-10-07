# SPEC-0022 implementation evidence

Accepted specification: [0022-email-authentication-profile.md](0022-email-authentication-profile.md).
Branch: `codex/email-profile`. Verified locally 2026-10-07.
Integrated in PR #10 and deployed as release
`54053633129560c99206c08d39855b547a87eb65`. Production email delivery and the
owner's personally completed email sign-in are verified. Telegram login is disabled.

| Criterion | Executable evidence | Result |
| --- | --- | --- |
| AC1 OTP, invitation, expiry, attempts, one use and browser binding | `go test ./internal/auth -run TestEmail`; `./scripts/test-authentication.sh` | Passed. Real PostgreSQL migration login; browser cookie and digest verification; unit expiry/attempt/replay checks. |
| AC2 Existing owner UUID/role/finances retained; email collision refused | `./scripts/test-authentication.sh` | Passed. Wrong mailbox cannot consume bound recovery grant; same owner and account balances survive migration; another user cannot claim that email. |
| AC3 Names, avatar limits and own profile | `go test ./internal/auth`; `./scripts/test-authentication.sh`; `cd web && npm test -- --run` | Passed. PNG normalization, SVG/pixel-limit refusal, own avatar isolation; names persist. |
| AC4 Telegram entry points disabled in email mode | `go test ./internal/auth -run TestEmailMode`; `cd web && npm test -- --run` | Passed. Login/callback refuse Telegram; existing session opens app without requesting a new code. |
| AC5 TLS and secrets; persistent rate limit | `go test ./internal/auth -run 'TestSMTP|TestMailDelivery'`; `go test ./internal/platform/config`; `./scripts/test-authentication.sh` | Passed. SMTP without STARTTLS receives no credentials or message; generic delivery error; cooldown persists across repository instances. |
| AC6 Common gates, contract, rollout safety | `make test`; `git diff --check`; `sh scripts/test-email-rollout.sh` | Passed locally and after integration. 29 Web tests; regenerated API types; Linux rollback test without network; remote CI and Delivery passed. |
| AC7 Actual production mailbox delivery and owner migration | Production email request + owner-entered OTP; protected `prepare-owner-email` operator command; read-only database verification | Passed. Owner confirmed delivery and sign-in; email identity belongs to the original owner UUID with admin role. Fingerprints of all eight owner financial tables match before and after. |

Remote evidence: [PR CI](https://github.com/QahNaaRlN/grace/actions/runs/37615148767),
[main CI](https://github.com/QahNaaRlN/grace/actions/runs/37615749959),
[Delivery](https://github.com/QahNaaRlN/grace/actions/runs/37616140412),
[deployment](https://github.com/QahNaaRlN/grace/actions/runs/37616378947).
SMTP was subsequently installed using the owner's explicitly authorized root
access; deployment itself ran as grace-deploy. A verified backup preceded the
email-mode switch and the bundled readiness/public-entrypoint gates passed.
Postbox DKIM records both show Ok; SMTP authentication was verified over TLS
from the workstation and VPS without printing credentials.

Executable anonymous production checks: `GET /api/auth/methods` returns
email=true and telegram=false; GET /api/auth/me, /api/auth/profile,
/api/auth/profile/avatar and /api/v1/accounts return 401; POST /api/auth/login
returns 503; GET /api/auth/callback returns 303 to the login failure page.
None of these requests returns a Basic authentication challenge. The browser
profile displays the verified email after the owner completes the code flow.

Local browser evidence on synthetic data: profile names saved, refreshed and
displayed; sanitized avatar loaded at 256×256; mobile 390px viewport had no horizontal
overflow. Avatar processing/upload was verified through the API; automated native
file chooser was unavailable because of the browser extension's file-access setting.
This does not constitute production smoke acceptance.

An additive migration 000008 retains all financial tables and owner IDs.
SMTP secrets and the owner's actual address are excluded from source control.
The CLI prepares a bound 24-hour grant and prints no invitation secret.

Operational sequence and current prerequisites:
[email-authentication-rollout.md](../email-authentication-rollout.md).
