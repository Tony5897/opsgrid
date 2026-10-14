package authn

import (
	"context"
	"errors"
	"fmt"
	"net/netip"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/Tony5897/opsgrid/internal/platform/database"
)

// Store persists users and sessions via InSystemTx (no RLS on these tables).
type Store struct {
	db *database.DB
}

// NewStore returns a Store backed by db.
func NewStore(db *database.DB) *Store {
	return &Store{db: db}
}

// UserUpsert is the identity written at login.
type UserUpsert struct {
	Issuer      string
	Subject     string
	Email       string // empty clears nothing; blank email is stored as NULL
	DisplayName string
}

// UpsertUser inserts or updates the user keyed by (issuer, subject).
func (s *Store) UpsertUser(ctx context.Context, in UserUpsert) (uuid.UUID, error) {
	var id uuid.UUID
	err := s.db.InSystemTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		var email any
		if in.Email != "" {
			email = in.Email
		}
		return tx.QueryRow(ctx, `
			INSERT INTO app.users (issuer, subject, email, display_name)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (issuer, subject) DO UPDATE SET
				email = EXCLUDED.email,
				display_name = EXCLUDED.display_name,
				updated_at = now()
			RETURNING id`,
			in.Issuer, in.Subject, email, in.DisplayName,
		).Scan(&id)
	})
	if err != nil {
		return uuid.Nil, fmt.Errorf("authn: upsert user: %w", err)
	}
	return id, nil
}

// CreateSessionParams are inputs for CreateSession.
type CreateSessionParams struct {
	UserID            uuid.UUID
	IDHash            []byte
	AbsoluteExpiresAt time.Time
	IdleExpiresAt     time.Time
	RefreshTokenEnc   []byte
	RefreshTokenKeyID string
	IP                string
	UserAgent         string
}

// CreateSession inserts a new session row.
func (s *Store) CreateSession(ctx context.Context, p CreateSessionParams) error {
	return s.db.InSystemTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		var ip any
		if p.IP != "" {
			if addr, err := netip.ParseAddr(p.IP); err == nil {
				ip = addr.String()
			}
		}
		var ua any
		if p.UserAgent != "" {
			ua = p.UserAgent
		}
		_, err := tx.Exec(ctx, `
			INSERT INTO app.sessions (
				id_hash, user_id, absolute_expires_at, idle_expires_at,
				refresh_token_enc, refresh_token_key_id, ip, user_agent
			) VALUES ($1, $2, $3, $4, $5, $6, $7::inet, $8)`,
			p.IDHash, p.UserID, p.AbsoluteExpiresAt, p.IdleExpiresAt,
			p.RefreshTokenEnc, p.RefreshTokenKeyID, ip, ua,
		)
		if err != nil {
			return fmt.Errorf("authn: create session: %w", err)
		}
		return nil
	})
}

// ErrSessionNotFound is returned when no live session matches the hash.
var ErrSessionNotFound = errors.New("authn: session not found")

// GetLiveSession loads a session by hash if it is still within both expiry windows.
func (s *Store) GetLiveSession(ctx context.Context, idHash []byte, now time.Time) (Session, error) {
	var sess Session
	err := s.db.InSystemTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		var ip *netip.Addr
		var ua *string
		err := tx.QueryRow(ctx, `
			SELECT id_hash, user_id, created_at, last_seen_at,
			       absolute_expires_at, idle_expires_at,
			       refresh_token_enc, refresh_token_key_id, ip, user_agent
			FROM app.sessions
			WHERE id_hash = $1
			  AND absolute_expires_at > $2
			  AND idle_expires_at > $2`,
			idHash, now,
		).Scan(
			&sess.IDHash, &sess.UserID, &sess.CreatedAt, &sess.LastSeenAt,
			&sess.AbsoluteExpiresAt, &sess.IdleExpiresAt,
			&sess.RefreshTokenEnc, &sess.RefreshTokenKeyID, &ip, &ua,
		)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrSessionNotFound
		}
		if err != nil {
			return err
		}
		if ip != nil {
			v := ip.String()
			sess.IP = &v
		}
		sess.UserAgent = ua
		return nil
	})
	if err != nil {
		return Session{}, err
	}
	return sess, nil
}

// TouchSession extends idle expiry and last_seen_at when the previous touch
// is older than minInterval (throttled writes per ADR-013).
func (s *Store) TouchSession(ctx context.Context, idHash []byte, now time.Time, idleTTL, minInterval time.Duration) error {
	return s.db.InSystemTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		tag, err := tx.Exec(ctx, `
			UPDATE app.sessions
			SET last_seen_at = $2,
			    idle_expires_at = $3
			WHERE id_hash = $1
			  AND last_seen_at <= $4
			  AND absolute_expires_at > $2
			  AND idle_expires_at > $2`,
			idHash, now, now.Add(idleTTL), now.Add(-minInterval),
		)
		if err != nil {
			return fmt.Errorf("authn: touch session: %w", err)
		}
		_ = tag
		return nil
	})
}

// DeleteSession removes one session by hash. Missing rows are not an error.
func (s *Store) DeleteSession(ctx context.Context, idHash []byte) error {
	return s.db.InSystemTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `DELETE FROM app.sessions WHERE id_hash = $1`, idHash)
		return err
	})
}

// DeleteSessionsForSubject deletes all sessions for the user with (issuer, subject).
func (s *Store) DeleteSessionsForSubject(ctx context.Context, issuer, subject string) (int64, error) {
	var n int64
	err := s.db.InSystemTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		tag, err := tx.Exec(ctx, `
			DELETE FROM app.sessions s
			USING app.users u
			WHERE s.user_id = u.id
			  AND u.issuer = $1
			  AND u.subject = $2`,
			issuer, subject,
		)
		if err != nil {
			return err
		}
		n = tag.RowsAffected()
		return nil
	})
	return n, err
}

// IsUniqueViolation reports a PostgreSQL unique_violation.
func IsUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
