package authn_test

import (
	"bytes"
	"testing"

	"github.com/Tony5897/opsgrid/internal/platform/authn"
)

func TestSealerRoundTrip(t *testing.T) {
	t.Parallel()
	key := bytes.Repeat([]byte("k"), 32)
	s, err := authn.NewSealer(key, "v1")
	if err != nil {
		t.Fatal(err)
	}
	plain := []byte("refresh-token-value")
	blob, err := s.SealBinary(plain)
	if err != nil {
		t.Fatal(err)
	}
	got, err := s.OpenBinary(blob)
	if err != nil || !bytes.Equal(got, plain) {
		t.Fatalf("got %q err %v", got, err)
	}
	enc, err := s.SealString(plain)
	if err != nil {
		t.Fatal(err)
	}
	got, err = s.OpenString(enc)
	if err != nil || !bytes.Equal(got, plain) {
		t.Fatalf("string round-trip: %q %v", got, err)
	}
}

func TestSealerRejectsWrongKeyID(t *testing.T) {
	t.Parallel()
	key := bytes.Repeat([]byte("k"), 32)
	s1, _ := authn.NewSealer(key, "v1")
	s2, _ := authn.NewSealer(key, "v2")
	blob, err := s1.SealBinary([]byte("x"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s2.OpenBinary(blob); err == nil {
		t.Fatal("expected key id mismatch")
	}
}

func TestParseEncryptionKey(t *testing.T) {
	t.Parallel()
	// "opsgrid-dev-session-key-32bytes!"
	const b64 = "b3BzZ3JpZC1kZXYtc2Vzc2lvbi1rZXktMzJieXRlcyE="
	key, err := authn.ParseEncryptionKey(b64)
	if err != nil || len(key) != 32 {
		t.Fatalf("key=%v err=%v", key, err)
	}
}

func TestNewSessionIDHashedOnly(t *testing.T) {
	t.Parallel()
	raw, hash, err := authn.NewSessionID()
	if err != nil || raw == "" || len(hash) != 32 {
		t.Fatalf("raw=%q hash=%d err=%v", raw, len(hash), err)
	}
	if !bytes.Equal(hash, authn.HashSessionID(raw)) {
		t.Fatal("hash mismatch")
	}
	if bytes.Contains(hash, []byte(raw)) {
		t.Fatal("raw leaked into hash")
	}
}
