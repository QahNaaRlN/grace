# SPEC-0015: Safe diagnosis and repair of Telegram first-login failures

- Статус: integrated
- Implementation commits: db59287 (diagnostics), 799c702 (mixed-JWKS repair).
- Live acceptance: owner confirmed successful login on 2026-10-06 Asia/Yekaterinburg; database checks confirmed an admin identity, active session and consumed invitation.
- Acceptance owner: owner
- Acceptance record: owner requested complete Telegram setup and authorized publication/deployment after CI; first live login returned Grace authentication_failed after Telegram success. This is corrective work within accepted SPEC-0013 security contracts.
- Implementation branch: codex/telegram-login-repair

## Scope

Add finite, safe callback failure stages for browser/state, flow, token exchange, missing ID token, signed-token claims/signature, nonce and invitation/session persistence. Logs must contain only a fixed message and a controlled stage, never token, code, state, cookie, invitation, provider response/error text, user identity or secrets. Keep the public callback error generic and fail-closed. Include client_id in the token form as in the Telegram contract while retaining Basic client authentication and PKCE.

Use these stages to find and repair the live first-login incompatibility. Every repair preserves signature, issuer, audience, expiry, nonce, browser binding, invite-only access, RLS and CSRF. No weakening checks or manual identity/session insertion. Any required security-contract change needs explicit review rather than an automatic workaround.

## Acceptance

- [x] Tests prove safe failure stages and no sensitive values in logs; invalid credentials/tokens still fail closed.
- [x] Token fixture requires Basic authentication, client_id, PKCE and exact redirect URI.
- [x] make test, make test-production, git diff --check; CI/Delivery before live update.
- [x] Live failure stage identified; owner first login succeeds after the corrective update, with Telegram consent performed by owner.

## Rollout

### Confirmed compatibility repair

The live callback reached signature_verification. Parsing Telegram's public JWKS with the exact project dependency reproduces `unsupported elliptic curve 'secp256k1'`: a key for ES256K prevents the library from decoding the whole key set, including the required RS256 key. Before the library decodes this public JWKS, select only RSA keys explicitly marked RS256 and intended for signatures. Keep the original key material and kid, HTTPS endpoint, bounded response size, timeout, remote refresh/cache and RS256 cryptographic verification. Malformed responses, absence of eligible keys and invalid signatures fail closed. Tests must prove mixed-algorithm compatibility and rejection behavior. This is within the accepted corrective scope and does not change the authentication security contract.

Use the established protected server environment and release pipeline. Reuse the existing unconsumed owner invitation; do not rotate or consume it merely for diagnostics. Production and interactive results must be recorded honestly; stay implementing while live acceptance is pending.
