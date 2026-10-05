# SPEC-0015 implementation evidence

Accepted scope: [SPEC-0015](0015-telegram-login-repair.md).

Callback failures now record only a controlled stage and a fixed message. Provider errors, tokens, cookies, codes, invitations and identities are never recorded by this diagnostic. The public failure remains generic. The token form includes client_id while preserving Basic authentication, PKCE and exact redirect URI.

Local checks on 2026-10-05 passed: `make test` (including web tests/build and offline evaluation), `go test ./internal/auth`, `go vet ./cmd/... ./internal/...`, `make test-production` in Linux Docker, and `git diff --check`. The signed-token fixture rejects invalid issuer, audience, expiration, signature, subject, mismatched nonce and missing nonce. The callback test verifies that sensitive inputs and arbitrary provider errors do not appear in logs.

Initial diagnostic release db592874a829fda8bf98e27ddebc89d1089dfc95 passed CI 37356969782, Delivery 37357363613 and Deploy production 37357530791. The server release symlink and healthy API/Web containers were verified. Owner's retry produced only the controlled stage signature_verification.

Confirmed cause: the public Telegram JWKS includes an ES256K/secp256k1 key. Parsing it with the project's go-jose v4.0.5 reproduces `unsupported elliptic curve 'secp256k1'`; go-oidc therefore cannot verify even an RS256 token. The repair filters only the public JWKS response to eligible RSA/RS256 signature-verification keys before the JOSE decoder, with a 1 MiB bound. Original key material and kid are preserved; go-oidc still controls caching/refresh and all signed-token checks. No user token was read or exported for diagnosis.

The signed-token regression fixture now includes the unsupported curve alongside a valid RSA key. It verifies valid signed tokens and still rejects all invalid-token cases. Transport tests reject malformed, oversized, empty and incompatible key sets, encryption-only keys and wrong key operations; HTTP failures remain failures. Live first owner login remains pending, so SPEC-0015 stays implementing.
