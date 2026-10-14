package authn

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
)

// Sealer encrypts and decrypts small blobs with AES-256-GCM. Used for IdP
// refresh tokens at rest and the short-lived OIDC handshake cookie.
type Sealer struct {
	aead  cipher.AEAD
	keyID string
}

// NewSealer builds a Sealer from a 32-byte key and a non-empty key id used for
// rotation. Ciphertext is: keyID length-prefixed UTF-8 + nonce + sealed data,
// then base64-url encoded when written to cookies; for bytea columns the raw
// binary form (without base64) is stored via SealBinary / OpenBinary.
func NewSealer(key []byte, keyID string) (*Sealer, error) {
	if len(key) != 32 {
		return nil, fmt.Errorf("authn: encryption key must be 32 bytes, got %d", len(key))
	}
	if keyID == "" {
		return nil, errors.New("authn: encryption key id is required")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return &Sealer{aead: aead, keyID: keyID}, nil
}

// KeyID returns the envelope key identifier.
func (s *Sealer) KeyID() string { return s.keyID }

// SealBinary encrypts plaintext. Layout: 1-byte keyID len | keyID | nonce | ciphertext.
func (s *Sealer) SealBinary(plaintext []byte) ([]byte, error) {
	keyIDLen := len(s.keyID)
	if keyIDLen == 0 || keyIDLen > 255 {
		return nil, errors.New("authn: key id length must be 1..255")
	}
	nonce := make([]byte, s.aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	out := make([]byte, 0, 1+keyIDLen+len(nonce)+len(plaintext)+s.aead.Overhead())
	out = append(out, byte(keyIDLen))
	out = append(out, s.keyID...)
	out = append(out, nonce...)
	out = append(out, s.aead.Seal(nil, nonce, plaintext, nil)...)
	return out, nil
}

// OpenBinary decrypts a SealBinary blob. keyID in the envelope must match this sealer.
func (s *Sealer) OpenBinary(blob []byte) ([]byte, error) {
	if len(blob) < 1 {
		return nil, errors.New("authn: ciphertext too short")
	}
	n := int(blob[0])
	if len(blob) < 1+n+s.aead.NonceSize() {
		return nil, errors.New("authn: ciphertext truncated")
	}
	keyID := string(blob[1 : 1+n])
	if keyID != s.keyID {
		return nil, fmt.Errorf("authn: unknown encryption key id %q", keyID)
	}
	rest := blob[1+n:]
	nonce, ct := rest[:s.aead.NonceSize()], rest[s.aead.NonceSize():]
	return s.aead.Open(nil, nonce, ct, nil)
}

// SealString seals and base64-url-encodes for cookies.
func (s *Sealer) SealString(plaintext []byte) (string, error) {
	b, err := s.SealBinary(plaintext)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// OpenString decodes a SealString value.
func (s *Sealer) OpenString(encoded string) ([]byte, error) {
	b, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return nil, errors.New("authn: invalid ciphertext encoding")
	}
	return s.OpenBinary(b)
}

// ParseEncryptionKey decodes a base64 (Std or Raw) 32-byte session key.
func ParseEncryptionKey(s string) ([]byte, error) {
	if s == "" {
		return nil, errors.New("authn: empty encryption key")
	}
	key, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		key, err = base64.RawStdEncoding.DecodeString(s)
	}
	if err != nil {
		return nil, errors.New("authn: encryption key must be base64")
	}
	if len(key) != 32 {
		return nil, fmt.Errorf("authn: encryption key must decode to 32 bytes, got %d", len(key))
	}
	return key, nil
}
