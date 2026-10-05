package auth

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

const TelegramIssuer = "https://oauth.telegram.org"

type Provider interface {
	AuthorizationURL(Flow) string
	Exchange(context.Context, string, Flow) (Identity, error)
}

type OIDCProvider struct {
	config   oauth2.Config
	verifier *oidc.IDTokenVerifier
	issuer   string
}

func NewTelegramProvider(clientID, secret, callback string) *OIDCProvider {
	return newOIDCProvider(TelegramIssuer, clientID, secret, callback)
}

func newOIDCProvider(issuer, clientID, secret, callback string) *OIDCProvider {
	keyURL := issuer + "/.well-known/jwks.json"
	keyContext := oidc.ClientContext(context.Background(), &http.Client{Timeout: 10 * time.Second, Transport: rs256KeyTransport{base: http.DefaultTransport, url: keyURL}})
	keys := oidc.NewRemoteKeySet(keyContext, keyURL)
	return &OIDCProvider{
		issuer: issuer,
		config: oauth2.Config{ClientID: clientID, ClientSecret: secret, RedirectURL: callback,
			Scopes: []string{oidc.ScopeOpenID, "profile"}, Endpoint: oauth2.Endpoint{
				AuthURL: issuer + "/auth", TokenURL: issuer + "/token", AuthStyle: oauth2.AuthStyleInHeader}},
		verifier: oidc.NewVerifier(issuer, keys, &oidc.Config{ClientID: clientID, SupportedSigningAlgs: []string{"RS256"}}),
	}
}
func (p *OIDCProvider) AuthorizationURL(flow Flow) string {
	return p.config.AuthCodeURL(flow.State, oauth2.S256ChallengeOption(flow.Verifier), oauth2.SetAuthURLParam("nonce", flow.Nonce))
}
func (p *OIDCProvider) Exchange(ctx context.Context, code string, flow Flow) (Identity, error) {
	token, err := p.config.Exchange(ctx, code, oauth2.VerifierOption(flow.Verifier), oauth2.SetAuthURLParam("client_id", p.config.ClientID))
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) || strings.Contains(err.Error(), "context deadline exceeded") {
			return Identity{}, failure("provider_timeout")
		}
		return Identity{}, failure("token_exchange")
	}
	raw, ok := token.Extra("id_token").(string)
	if !ok {
		return Identity{}, failure("id_token_missing")
	}
	idToken, err := p.verifier.Verify(ctx, raw)
	if err != nil {
		var expired *oidc.TokenExpiredError
		switch {
		case errors.As(err, &expired):
			return Identity{}, failure("token_expired")
		case strings.Contains(err.Error(), "issued by a different provider"):
			return Identity{}, failure("issuer_mismatch")
		case strings.Contains(err.Error(), "expected audience"):
			return Identity{}, failure("audience_mismatch")
		case strings.Contains(err.Error(), "failed to verify signature"):
			return Identity{}, failure("signature_verification")
		default:
			return Identity{}, failure("token_verification")
		}
	}
	if idToken.Subject == "" {
		return Identity{}, failure("subject_missing")
	}
	var claims struct {
		Nonce string `json:"nonce"`
		Name  string `json:"name"`
	}
	if err := idToken.Claims(&claims); err != nil {
		return Identity{}, failure("claims_invalid")
	}
	if claims.Nonce == "" {
		return Identity{}, failure("nonce_missing")
	}
	if claims.Nonce != flow.Nonce {
		return Identity{}, failure("nonce_mismatch")
	}
	if len(claims.Name) > 160 {
		claims.Name = ""
	}
	if idToken.Issuer != p.issuer {
		return Identity{}, failure("issuer_mismatch")
	}
	return Identity{Issuer: idToken.Issuer, Subject: idToken.Subject, Name: claims.Name}, nil
}
