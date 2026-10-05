package auth

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/go-jose/go-jose/v4"
)

func TestOIDCValidationAndPKCE(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	wrongKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	var raw string
	f := Flow{State: randomToken(), Verifier: randomToken(), Nonce: randomToken()}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/.well-known/jwks.json" {
			// Telegram publishes secp256k1 alongside RSA. go-jose cannot
			// decode that curve, but it must still verify the RSA signature.
			_ = json.NewEncoder(w).Encode(map[string]any{"keys": []any{jose.JSONWebKey{Key: &key.PublicKey, KeyID: "test", Algorithm: "RS256", Use: "sig"}, map[string]string{"kty": "EC", "alg": "ES256K", "crv": "secp256k1", "kid": "unrelated", "x": "invalid", "y": "invalid"}}})
			return
		}
		if r.URL.Path != "/token" {
			t.Error("unexpected endpoint, including UserInfo", r.URL.Path)
			w.WriteHeader(404)
			return
		}
		id, secret, ok := r.BasicAuth()
		if !ok || id != "client" || secret != "secret" {
			t.Error("client authentication missing")
		}
		_ = r.ParseForm()
		if r.Form.Get("code_verifier") != f.Verifier || r.Form.Get("grant_type") != "authorization_code" || r.Form.Get("client_id") != "client" || r.Form.Get("redirect_uri") != "https://grace.test/api/auth/callback" {
			t.Error("token exchange does not match Telegram contract")
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "opaque", "token_type": "Bearer", "id_token": raw})
	}))
	defer server.Close()
	p := newOIDCProvider(server.URL, "client", "secret", "https://grace.test/api/auth/callback")
	u, err := url.Parse(p.AuthorizationURL(f))
	if err != nil {
		t.Fatal(err)
	}
	if u.Query().Get("state") != f.State || u.Query().Get("nonce") != f.Nonce || u.Query().Get("code_challenge_method") != "S256" || u.Query().Get("code_challenge") == "" || u.Query().Get("scope") != "openid profile" {
		t.Fatalf("unsafe authorization URL: %s", u)
	}
	for _, test := range []string{"valid", "issuer", "audience", "expired", "nonce", "missing_nonce", "signature", "subject"} {
		t.Run(test, func(t *testing.T) {
			claims := map[string]any{"iss": server.URL, "aud": "client", "sub": "stable-subject", "exp": time.Now().Add(time.Hour).Unix(), "iat": time.Now().Unix(), "nonce": f.Nonce, "name": "Alice"}
			signingKey := key
			switch test {
			case "issuer":
				claims["iss"] = "https://wrong.test"
			case "audience":
				claims["aud"] = "other"
			case "expired":
				claims["exp"] = time.Now().Add(-time.Hour).Unix()
			case "nonce":
				claims["nonce"] = "wrong"
			case "missing_nonce":
				delete(claims, "nonce")
			case "signature":
				signingKey = wrongKey
			case "subject":
				claims["sub"] = ""
			}
			signer, err := jose.NewSigner(jose.SigningKey{Algorithm: jose.RS256, Key: signingKey}, (&jose.SignerOptions{}).WithHeader("kid", "test"))
			if err != nil {
				t.Fatal(err)
			}
			payload, _ := json.Marshal(claims)
			signed, err := signer.Sign(payload)
			if err != nil {
				t.Fatal(err)
			}
			raw, err = signed.CompactSerialize()
			if err != nil {
				t.Fatal(err)
			}
			identity, err := p.Exchange(context.Background(), "code", f)
			if test == "valid" {
				if err != nil || identity.Subject != "stable-subject" {
					t.Fatalf("valid token rejected: %#v %v", identity, err)
				}
			} else if err == nil {
				t.Fatal("invalid token accepted")
			} else {
				want := map[string]string{"issuer": "issuer_mismatch", "audience": "audience_mismatch", "expired": "token_expired", "nonce": "nonce_mismatch", "missing_nonce": "nonce_missing", "signature": "signature_verification", "subject": "subject_missing"}[test]
				if failureStage(err) != want {
					t.Fatalf("unsafe or incorrect failure stage: %s", failureStage(err))
				}
			}
		})
	}
}
