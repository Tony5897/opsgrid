//go:build integration

package authn_test

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Tony5897/opsgrid/internal/platform/authn"
	"github.com/Tony5897/opsgrid/internal/platform/database/dbtest"
)

func TestMain(m *testing.M) { dbtest.Main(m) }

func TestSessionLifecycle(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	db := dbtest.New(t).Open(t, "opsgrid_app", 4)
	store := authn.NewStore(db)
	sealer, err := authn.NewSealer(bytes.Repeat([]byte("s"), 32), "v1")
	if err != nil {
		t.Fatal(err)
	}

	userID, err := store.UpsertUser(ctx, authn.UserUpsert{
		Issuer:      "http://localhost:8180/realms/opsgrid",
		Subject:     "sub-owen",
		Email:       "owen@cascade.example",
		DisplayName: "Owen Park",
	})
	if err != nil || userID.String() == "" {
		t.Fatalf("upsert: %v %v", userID, err)
	}
	userID2, err := store.UpsertUser(ctx, authn.UserUpsert{
		Issuer:      "http://localhost:8180/realms/opsgrid",
		Subject:     "sub-owen",
		Email:       "owen@cascade.example",
		DisplayName: "Owen P.",
	})
	if err != nil || userID2 != userID {
		t.Fatalf("re-upsert: %v %v want %v", userID2, err, userID)
	}

	_, hash, err := authn.NewSessionID()
	if err != nil {
		t.Fatal(err)
	}
	enc, err := sealer.SealBinary([]byte("refresh-token"))
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	if err := store.CreateSession(ctx, authn.CreateSessionParams{
		UserID:            userID,
		IDHash:            hash,
		AbsoluteExpiresAt: now.Add(12 * time.Hour),
		IdleExpiresAt:     now.Add(2 * time.Hour),
		RefreshTokenEnc:   enc,
		RefreshTokenKeyID: sealer.KeyID(),
		IP:                "127.0.0.1",
		UserAgent:         "test",
	}); err != nil {
		t.Fatal(err)
	}

	live, err := store.GetLiveSession(ctx, hash, now.Add(time.Minute))
	if err != nil || live.UserID != userID {
		t.Fatalf("live: %+v %v", live, err)
	}

	if _, err := store.GetLiveSession(ctx, hash, now.Add(3*time.Hour)); !errors.Is(err, authn.ErrSessionNotFound) {
		t.Fatalf("want not found after idle expiry, got %v", err)
	}

	_, hash2, _ := authn.NewSessionID()
	enc2, _ := sealer.SealBinary([]byte("refresh-2"))
	if err := store.CreateSession(ctx, authn.CreateSessionParams{
		UserID:            userID,
		IDHash:            hash2,
		AbsoluteExpiresAt: now.Add(12 * time.Hour),
		IdleExpiresAt:     now.Add(2 * time.Hour),
		RefreshTokenEnc:   enc2,
		RefreshTokenKeyID: sealer.KeyID(),
	}); err != nil {
		t.Fatal(err)
	}
	n, err := store.DeleteSessionsForSubject(ctx, "http://localhost:8180/realms/opsgrid", "sub-owen")
	if err != nil || n < 1 {
		t.Fatalf("delete by subject: n=%d err=%v", n, err)
	}
	if _, err := store.GetLiveSession(ctx, hash2, now); !errors.Is(err, authn.ErrSessionNotFound) {
		t.Fatalf("session should be gone: %v", err)
	}
}

func TestSessionIDRotatesOnNewLogin(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	db := dbtest.New(t).Open(t, "opsgrid_app", 4)
	store := authn.NewStore(db)
	sealer, _ := authn.NewSealer(bytes.Repeat([]byte("r"), 32), "v1")
	userID, err := store.UpsertUser(ctx, authn.UserUpsert{
		Issuer: "https://issuer.example", Subject: "fixation-user", DisplayName: "Fix",
	})
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	raw1, hash1, _ := authn.NewSessionID()
	enc, _ := sealer.SealBinary([]byte("rt1"))
	if err := store.CreateSession(ctx, authn.CreateSessionParams{
		UserID: userID, IDHash: hash1,
		AbsoluteExpiresAt: now.Add(time.Hour), IdleExpiresAt: now.Add(time.Hour),
		RefreshTokenEnc: enc, RefreshTokenKeyID: "v1",
	}); err != nil {
		t.Fatal(err)
	}
	raw2, hash2, _ := authn.NewSessionID()
	if raw1 == raw2 || bytes.Equal(hash1, hash2) {
		t.Fatal("session ids must rotate")
	}
	enc2, _ := sealer.SealBinary([]byte("rt2"))
	if err := store.CreateSession(ctx, authn.CreateSessionParams{
		UserID: userID, IDHash: hash2,
		AbsoluteExpiresAt: now.Add(time.Hour), IdleExpiresAt: now.Add(time.Hour),
		RefreshTokenEnc: enc2, RefreshTokenKeyID: "v1",
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.GetLiveSession(ctx, hash1, now); err != nil {
		t.Fatalf("old session still valid until logout: %v", err)
	}
	if err := store.DeleteSession(ctx, hash1); err != nil {
		t.Fatal(err)
	}
	if _, err := store.GetLiveSession(ctx, hash1, now); !errors.Is(err, authn.ErrSessionNotFound) {
		t.Fatal("deleted old session still live")
	}
	if _, err := store.GetLiveSession(ctx, hash2, now); err != nil {
		t.Fatalf("new session: %v", err)
	}
}
