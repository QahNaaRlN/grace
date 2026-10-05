# SPEC-0015 implementation evidence

Accepted scope: [SPEC-0015](0015-telegram-login-repair.md).

Callback failures now record only a controlled stage and a fixed message. Provider errors, tokens, cookies, codes, invitations and identities are never recorded by this diagnostic. The public failure remains generic. The token form includes client_id while preserving Basic authentication, PKCE and exact redirect URI.

Local checks on 2026-10-05 passed: `make test` (including web tests/build and offline evaluation), `go test ./internal/auth`, `go vet ./cmd/... ./internal/...`, `make test-production` in Linux Docker, and `git diff --check`. The signed-token fixture rejects invalid issuer, audience, expiration, signature, subject, mismatched nonce and missing nonce. The callback test verifies that sensitive inputs and arbitrary provider errors do not appear in logs.

Live status before deployment: no Telegram identities or sessions; one valid unused owner invitation. Owner confirms an initial attempt without invitation followed by an attempt with it. The failure cause remains unconfirmed. CI/Delivery, deployment and an owner-assisted retry are pending. SPEC-0015 remains implementing until live acceptance succeeds.
