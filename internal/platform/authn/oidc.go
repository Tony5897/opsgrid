package authn

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
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

// Provider wraps go-oidc / oauth2 for the BFF.
type Provider struct {
	cfg      OIDCConfig
	provider *oidc.Provider
	oauth    oauth2.Config
	verifier *oidc.IDTokenVerifier
}

// NewProvider discovers the IdP and builds an OAuth2 config. When IssuerURL
// and ProviderURL differ (Docker hostname split), discovery uses ProviderURL
// while token verification expects IssuerURL.
func NewProvider(ctx context.Context, cfg OIDCConfig) (*Provider, error) {
	if cfg.IssuerURL == "" || cfg.ClientID == "" || cfg.RedirectURL == "" {
		return nil, errors.New("authn: OIDC issuer, client id and redirect URL are required")
	}
	if cfg.ProviderURL == "" {
		cfg.ProviderURL = cfg.IssuerURL
	}
	discCtx := ctx
	if cfg.ProviderURL != cfg.IssuerURL {
		discCtx = oidc.InsecureIssuerURLContext(ctx, cfg.IssuerURL)
	}
	p, err := oidc.NewProvider(discCtx, cfg.ProviderURL)
	if err != nil {
		return nil, fmt.Errorf("authn: OIDC discovery: %w", err)
	}
	oauth := oauth2.Config{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		RedirectURL:  cfg.RedirectURL,
		Endpoint:     p.Endpoint(),
		Scopes:       []string{oidc.ScopeOpenID, "profile", "email", "offline_access"},
	}
	// Prefer server-side token calls against the provider host when discovery
	// returned public (browser) hostnames that the API container cannot reach.
	oauth.Endpoint.TokenURL = rewriteHost(oauth.Endpoint.TokenURL, cfg.ProviderURL)
	oauth.Endpoint.AuthURL = rewriteHost(oauth.Endpoint.AuthURL, cfg.IssuerURL)

	return &Provider{
		cfg:      cfg,
		provider: p,
		oauth:    oauth,
		verifier: p.Verifier(&oidc.Config{ClientID: cfg.ClientID}),
	}, nil
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
func (p *Provider) AuthCodeURL(state, nonce, challenge string) string {
	return p.oauth.AuthCodeURL(state,
		oauth2.SetAuthURLParam("code_challenge", challenge),
		oauth2.SetAuthURLParam("code_challenge_method", "S256"),
		oauth2.SetAuthURLParam("nonce", nonce),
	)
}

// Exchange redeems an authorization code and verifies the ID token.
func (p *Provider) Exchange(ctx context.Context, code, verifier, nonce string) (TokenBundle, error) {
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
	if claims.EmailVerified || claims.Email != "" {
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
func (p *Provider) EndSessionURL(postLogoutRedirect string) (string, error) {
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

// VerifyLogoutToken validates an OIDC back-channel logout token.
func (p *Provider) VerifyLogoutToken(ctx context.Context, raw string) (LogoutClaims, error) {
	return parseLogoutJWT(ctx, p.provider, raw, p.cfg.ClientID, p.cfg.IssuerURL)
}

// parseLogoutJWT verifies the JWT signature via the provider JWKS, then
// enforces back-channel logout claims (events, no nonce, aud/azp).
func parseLogoutJWT(ctx context.Context, provider *oidc.Provider, raw, clientID, issuer string) (LogoutClaims, error) {
	verifier := provider.Verifier(&oidc.Config{
		ClientID:          clientID,
		SkipClientIDCheck: false,
		SkipIssuerCheck:   false,
		SkipExpiryCheck:   false,
	})
	// Logout tokens fail standard ID token verification when "nonce" is
	// required absent — go-oidc does not require nonce on Verify. The
	// events claim distinguishes logout tokens; we reject ID tokens that
	// lack it.
	idt, err := verifier.Verify(ctx, raw)
	if err != nil {
		// Some IdPs set azp instead of a single aud; retry with skip client
		// check and validate aud/azp manually from raw claims.
		verifier = provider.Verifier(&oidc.Config{
			SkipClientIDCheck: true,
			SkipIssuerCheck:   false,
		})
		idt, err = verifier.Verify(ctx, raw)
		if err != nil {
			return LogoutClaims{}, fmt.Errorf("authn: logout token verify: %w", err)
		}
	}
	if idt.Issuer != issuer && !strings.EqualFold(idt.Issuer, issuer) {
		return LogoutClaims{}, fmt.Errorf("authn: logout token issuer mismatch")
	}
	var claims struct {
		Aud    any            `json:"aud"`
		Azp    string         `json:"azp"`
		Events map[string]any `json:"events"`
		Nonce  string         `json:"nonce"`
		Sub    string         `json:"sub"`
	}
	if err := idt.Claims(&claims); err != nil {
		return LogoutClaims{}, err
	}
	if claims.Nonce != "" {
		return LogoutClaims{}, errors.New("authn: logout token must not contain nonce")
	}
	if _, ok := claims.Events["http://schemas.openid.net/event/backchannel-logout"]; !ok {
		return LogoutClaims{}, errors.New("authn: logout token missing backchannel-logout event")
	}
	if claims.Sub == "" {
		return LogoutClaims{}, errors.New("authn: logout token missing sub")
	}
	if !audienceIncludes(claims.Aud, clientID) && claims.Azp != clientID {
		return LogoutClaims{}, errors.New("authn: logout token audience mismatch")
	}
	return LogoutClaims{Issuer: idt.Issuer, Subject: claims.Sub}, nil
}

func audienceIncludes(aud any, clientID string) bool {
	switch v := aud.(type) {
	case string:
		return v == clientID
	case []any:
		for _, a := range v {
			if s, ok := a.(string); ok && s == clientID {
				return true
			}
		}
	case []string:
		for _, s := range v {
			if s == clientID {
				return true
			}
		}
	}
	// Fallback: re-marshal unusual shapes
	b, err := json.Marshal(aud)
	if err != nil {
		return false
	}
	return strings.Contains(string(b), clientID)
}
