# SPEC-0022 implementation evidence

Accepted specification: [0022-email-authentication-profile.md](0022-email-authentication-profile.md).
Branch: `codex/email-profile`. Verified locally 2026-10-07.
Implementation is ready for review; production email delivery and owner sign-in
are still pending. No production authentication switch has been performed.

| Criterion | Executable evidence | Result |
| --- | --- | --- |
| AC1 OTP, invitation, expiry, attempts, one use and browser binding | `go test ./internal/auth -run TestEmail`; `./scripts/test-authentication.sh` | Passed. Real PostgreSQL migration login; browser cookie and digest verification; unit expiry/attempt/replay checks. |
| AC2 Existing owner UUID/role/finances retained; email collision refused | `./scripts/test-authentication.sh` | Passed. Wrong mailbox cannot consume bound recovery grant; same owner and account balances survive migration; another user cannot claim that email. |
| AC3 Names, avatar limits and own profile | `go test ./internal/auth`; `./scripts/test-authentication.sh`; `cd web && npm test -- --run` | Passed. PNG normalization, SVG/pixel-limit refusal, own avatar isolation; names persist. |
| AC4 Telegram entry points disabled in email mode | `go test ./internal/auth -run TestEmailMode`; `cd web && npm test -- --run` | Passed. Login/callback refuse Telegram; existing session opens app without requesting a new code. |
| AC5 TLS and secrets; persistent rate limit | `go test ./internal/auth -run 'TestSMTP|TestMailDelivery'`; `go test ./internal/platform/config`; `./scripts/test-authentication.sh` | Passed. SMTP without STARTTLS receives no credentials or message; generic delivery error; cooldown persists across repository instances. |
| AC6 Common gates, contract, rollout safety | `make test`; `git diff --check`; `sh scripts/test-email-rollout.sh` | Passed. 29 Web tests; API types regenerated; rollback mode tested in Linux container with network disabled. Remote CI pending publication. |
| AC7 Actual production mailbox delivery and owner migration | Production email request + owner-entered OTP; protected `prepare-owner-email` operator command | Pending provider key, deployment and owner verification. |

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
