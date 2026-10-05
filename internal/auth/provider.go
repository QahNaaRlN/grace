package auth

import (
	"context"
	"errors"
	"net/http"
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
	keyContext := oidc.ClientContext(context.Background(), &http.Client{Timeout: 10 * time.Second})
	keys := oidc.NewRemoteKeySet(keyContext, issuer+"/.well-known/jwks.json")
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
	token, err := p.config.Exchange(ctx, code, oauth2.VerifierOption(flow.Verifier))
	if err != nil {
		return Identity{}, ErrDenied
	}
	raw, ok := token.Extra("id_token").(string)
	if !ok {
		return Identity{}, ErrDenied
	}
	idToken, err := p.verifier.Verify(ctx, raw)
	if err != nil || idToken.Subject == "" {
		return Identity{}, ErrDenied
	}
	var claims struct {
		Nonce string `json:"nonce"`
		Name  string `json:"name"`
	}
	if err := idToken.Claims(&claims); err != nil || claims.Nonce != flow.Nonce {
		return Identity{}, ErrDenied
	}
	if len(claims.Name) > 160 {
		claims.Name = ""
	}
	if idToken.Issuer != p.issuer {
		return Identity{}, errors.New("invalid identity issuer")
	}
	return Identity{Issuer: idToken.Issuer, Subject: idToken.Subject, Name: claims.Name}, nil
}
