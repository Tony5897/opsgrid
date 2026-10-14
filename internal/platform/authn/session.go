package authn

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"net/http"
	"time"

	"github.com/google/uuid"
)

// Cookie names use the __Host- prefix (Secure, Path=/, no Domain) per ADR-006/013/019.
const (
	SessionCookieName = "__Host-opsgrid_session"
	OIDCCookieName    = "__Host-oidc"
)

// Session is a server-side BFF session (ADR-013).
type Session struct {
	IDHash            []byte
	UserID            uuid.UUID
	CreatedAt         time.Time
	LastSeenAt        time.Time
	AbsoluteExpiresAt time.Time
	IdleExpiresAt     time.Time
	RefreshTokenEnc   []byte
	RefreshTokenKeyID string
	IP                *string
	UserAgent         *string
}

// Principal is the authenticated caller attached to the request context.
type Principal struct {
	UserID      uuid.UUID
	SessionHash []byte
}

// NewSessionID returns a 256-bit random session secret and its SHA-256 hash.
// Only the hash is stored; only the secret is placed in the cookie.
func NewSessionID() (raw string, hash []byte, err error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", nil, err
	}
	raw = base64.RawURLEncoding.EncodeToString(buf)
	sum := sha256.Sum256([]byte(raw))
	return raw, sum[:], nil
}

// HashSessionID hashes a cookie value the same way NewSessionID does.
func HashSessionID(raw string) []byte {
	sum := sha256.Sum256([]byte(raw))
	return sum[:]
}

// Alive reports whether the session is within absolute and idle windows.
func (s Session) Alive(now time.Time) bool {
	return now.Before(s.AbsoluteExpiresAt) && now.Before(s.IdleExpiresAt)
}

// SetSessionCookie writes the Host-prefixed session cookie.
func SetSessionCookie(w http.ResponseWriter, raw string, maxAge time.Duration) {
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    raw,
		Path:     "/",
		MaxAge:   int(maxAge.Seconds()),
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})
}

// ClearSessionCookie expires the session cookie.
func ClearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})
}

// SetOIDCCookie writes the short-lived OIDC handshake cookie.
func SetOIDCCookie(w http.ResponseWriter, value string, maxAge time.Duration) {
	http.SetCookie(w, &http.Cookie{
		Name:     OIDCCookieName,
		Value:    value,
		Path:     "/",
		MaxAge:   int(maxAge.Seconds()),
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})
}

// ClearOIDCCookie expires the handshake cookie.
func ClearOIDCCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     OIDCCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})
}
