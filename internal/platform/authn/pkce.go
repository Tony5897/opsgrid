package authn

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"
)

const oidcHandshakeTTL = 10 * time.Minute

// handshake is sealed into the __Host-oidc cookie between /auth/login and /auth/callback.
type handshake struct {
	State        string    `json:"state"`
	Nonce        string    `json:"nonce"`
	CodeVerifier string    `json:"cv"`
	ReturnTo     string    `json:"rt"`
	ExpiresAt    time.Time `json:"exp"`
}

func newHandshake(returnTo string, now time.Time) (handshake, string, error) {
	state, err := randomURLToken(32)
	if err != nil {
		return handshake{}, "", err
	}
	nonce, err := randomURLToken(32)
	if err != nil {
		return handshake{}, "", err
	}
	verifier, err := randomURLToken(64)
	if err != nil {
		return handshake{}, "", err
	}
	return handshake{
		State:        state,
		Nonce:        nonce,
		CodeVerifier: verifier,
		ReturnTo:     returnTo,
		ExpiresAt:    now.Add(oidcHandshakeTTL),
	}, pkceChallenge(verifier), nil
}

func pkceChallenge(verifier string) string {
	sum := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

func randomURLToken(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func (s *Sealer) sealHandshake(h handshake) (string, error) {
	b, err := json.Marshal(h)
	if err != nil {
		return "", err
	}
	return s.SealString(b)
}

func (s *Sealer) openHandshake(encoded string, now time.Time) (handshake, error) {
	b, err := s.OpenString(encoded)
	if err != nil {
		return handshake{}, err
	}
	var h handshake
	if err := json.Unmarshal(b, &h); err != nil {
		return handshake{}, fmt.Errorf("authn: corrupt oidc cookie: %w", err)
	}
	if !now.Before(h.ExpiresAt) {
		return handshake{}, fmt.Errorf("authn: oidc handshake expired")
	}
	return h, nil
}
