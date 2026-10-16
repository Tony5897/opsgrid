package authn

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"sync"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

// OIDCConfig configures the confidential BFF client (ADR-006).
type OIDCConfig struct {
	// IssuerURL is the iss claim browsers and tokens use (e.g. http://localhost:8180/realms/opsgrid).
	IssuerURL string
	// ProviderURL is where the API discovers endpoints (may be the internal
	// Keycloak host when IssuerURL is the public hostname).
	ProviderURL  string
	ClientID     string
	ClientSecret string
	RedirectURL  string
}

// TokenBundle is the result of a successful code exchange.
type TokenBundle struct {
	Issuer       string
	Subject      string
	Email        string
	DisplayName  string
	RefreshToken string
	Expiry       time.Time
}

// LogoutClaims are the verified back-channel logout token fields we use.
type LogoutClaims struct {
	Issuer  string
	Subject string
}

// Provider wraps go-oidc / oauth2 for the BFF. Discovery is lazy so the API
// can boot when Keycloak is down (failure matrix: existing sessions keep
// working; only new logins fail).
type Provider struct {
	cfg OIDCConfig

	mu       sync.Mutex
	provider *oidc.Provider
	oauth    oauth2.Config
	verifier *oidc.IDTokenVerifier
}

// NewProvider validates config and returns a Provider. IdP discovery runs on
// first Warm / login / callback / logout-token verify (not at construction).
func NewProvider(cfg OIDCConfig) (*Provider, error) {
	if cfg.IssuerURL == "" || cfg.ClientID == "" || cfg.RedirectURL == "" {
		return nil, errors.New("authn: OIDC issuer, client id and redirect URL are required")
	}
	if cfg.ProviderURL == "" {
		cfg.ProviderURL = cfg.IssuerURL
	}
	return &Provider{cfg: cfg}, nil
}

// Warm eagerly discovers the IdP (optional; used by tests that need a ready client).
func (p *Provider) Warm(ctx context.Context) error {
	return p.ensure(ctx)
}

func (p *Provider) ensure(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.provider != nil {
		return nil
	}
	discCtx := ctx
	if p.cfg.ProviderURL != p.cfg.IssuerURL {
		discCtx = oidc.InsecureIssuerURLContext(ctx, p.cfg.IssuerURL)
	}
	prov, err := oidc.NewProvider(discCtx, p.cfg.ProviderURL)
	if err != nil {
		return fmt.Errorf("authn: OIDC discovery: %w", err)
	}
	oauth := oauth2.Config{
		ClientID:     p.cfg.ClientID,
		ClientSecret: p.cfg.ClientSecret,
		RedirectURL:  p.cfg.RedirectURL,
		Endpoint:     prov.Endpoint(),
		// Regular refresh tokens (use.refresh.tokens on the client) — not
		// offline_access. Offline tokens need extra Keycloak grants and exceed
		// our 12h absolute / 2h idle session model (ADR-013).
		Scopes: []string{oidc.ScopeOpenID, "profile", "email"},
	}
	// Prefer server-side token calls against the provider host when discovery
	// returned public (browser) hostnames that the API container cannot reach.
	oauth.Endpoint.TokenURL = rewriteHost(oauth.Endpoint.TokenURL, p.cfg.ProviderURL)
	oauth.Endpoint.AuthURL = rewriteHost(oauth.Endpoint.AuthURL, p.cfg.IssuerURL)

	p.provider = prov
	p.oauth = oauth
	p.verifier = prov.Verifier(&oidc.Config{ClientID: p.cfg.ClientID})
	return nil
}

func rewriteHost(endpoint, base string) string {
	eu, err1 := url.Parse(endpoint)
	bu, err2 := url.Parse(base)
	if err1 != nil || err2 != nil || eu.Host == "" || bu.Host == "" {
		return endpoint
	}
	eu.Scheme = bu.Scheme
	eu.Host = bu.Host
	return eu.String()
}

// AuthCodeURL builds the authorization redirect with PKCE, state and nonce.
// Call Warm first (handlers do).
func (p *Provider) AuthCodeURL(state, nonce, challenge string) string {
	return p.oauth.AuthCodeURL(state,
		oauth2.SetAuthURLParam("code_challenge", challenge),
		oauth2.SetAuthURLParam("code_challenge_method", "S256"),
		oauth2.SetAuthURLParam("nonce", nonce),
	)
}

// Exchange redeems an authorization code and verifies the ID token.
func (p *Provider) Exchange(ctx context.Context, code, verifier, nonce string) (TokenBundle, error) {
	if err := p.ensure(ctx); err != nil {
		return TokenBundle{}, err
	}
	tok, err := p.oauth.Exchange(ctx, code, oauth2.VerifierOption(verifier))
	if err != nil {
		return TokenBundle{}, fmt.Errorf("authn: token exchange: %w", err)
	}
	rawID, ok := tok.Extra("id_token").(string)
	if !ok || rawID == "" {
		return TokenBundle{}, errors.New("authn: token response missing id_token")
	}
	idt, err := p.verifier.Verify(ctx, rawID)
	if err != nil {
		return TokenBundle{}, fmt.Errorf("authn: id_token verify: %w", err)
	}
	if idt.Nonce != nonce {
		return TokenBundle{}, errors.New("authn: nonce mismatch")
	}
	var claims struct {
		Email             string `json:"email"`
		EmailVerified     bool   `json:"email_verified"`
		Name              string `json:"name"`
		PreferredUsername string `json:"preferred_username"`
	}
	if err := idt.Claims(&claims); err != nil {
		return TokenBundle{}, fmt.Errorf("authn: id_token claims: %w", err)
	}
	display := claims.Name
	if display == "" {
		display = claims.PreferredUsername
	}
	email := ""
	if claims.EmailVerified && claims.Email != "" {
		email = claims.Email
	}
	return TokenBundle{
		Issuer:       idt.Issuer,
		Subject:      idt.Subject,
		Email:        email,
		DisplayName:  display,
		RefreshToken: tok.RefreshToken,
		Expiry:       tok.Expiry,
	}, nil
}

// EndSessionURL builds the RP-initiated logout redirect.
func (p *Provider) EndSessionURL(ctx context.Context, postLogoutRedirect string) (string, error) {
	if err := p.ensure(ctx); err != nil {
		return "", err
	}
	var claims struct {
		EndSessionEndpoint string `json:"end_session_endpoint"`
	}
	if err := p.provider.Claims(&claims); err != nil {
		return "", err
	}
	if claims.EndSessionEndpoint == "" {
		return "", errors.New("authn: IdP has no end_session_endpoint")
	}
	u, err := url.Parse(claims.EndSessionEndpoint)
	if err != nil {
		return "", err
	}
	// Browser must hit the public issuer host.
	u, err = url.Parse(rewriteHost(u.String(), p.cfg.IssuerURL))
	if err != nil {
		return "", err
	}
	q := u.Query()
	q.Set("client_id", p.cfg.ClientID)
	if postLogoutRedirect != "" {
		q.Set("post_logout_redirect_uri", postLogoutRedirect)
	}
	u.RawQuery = q.Encode()
	return u.String(), nil
}

// VerifyLogoutToken validates an OIDC back-channel logout token using go-oidc's
// dedicated logout verifier (events claim, no nonce, aud, exp, jti).
func (p *Provider) VerifyLogoutToken(ctx context.Context, raw string) (LogoutClaims, error) {
	if err := p.ensure(ctx); err != nil {
		return LogoutClaims{}, err
	}
	lt, err := p.verifier.VerifyLogout(ctx, raw)
	if err != nil {
		return LogoutClaims{}, fmt.Errorf("authn: logout token verify: %w", err)
	}
	// We match sessions by (issuer, subject). sid-only tokens need an idp_sid
	// column (not yet); Keycloak's configured client includes sub.
	if lt.Subject == "" {
		return LogoutClaims{}, errors.New("authn: logout token without sub is not supported yet")
	}
	return LogoutClaims{Issuer: lt.Issuer, Subject: lt.Subject}, nil
}
