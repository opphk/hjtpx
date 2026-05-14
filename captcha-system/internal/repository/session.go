package repository

import (
	"context"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/opphk/captcha-system/internal/model"
)

type SessionRepository struct {
	*BaseRepository
}

func NewSessionRepository(db *sqlx.DB) *SessionRepository {
	return &SessionRepository{
		BaseRepository: NewBaseRepository(db),
	}
}

func (r *SessionRepository) Create(ctx context.Context, session *model.Session) error {
	session.CreatedAt = time.Now()
	session.UpdatedAt = time.Now()
	query := `INSERT INTO sessions (session_id, fingerprint, ip_address, risk_score, attempt_count, expires_at, created_at, updated_at)
			  VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`
	_, err := r.db.ExecContext(ctx, query,
		session.SessionID,
		session.Fingerprint,
		session.IPAddress,
		session.RiskScore,
		session.AttemptCount,
		session.ExpiresAt,
		session.CreatedAt,
		session.UpdatedAt,
	)
	return err
}

func (r *SessionRepository) GetBySessionID(ctx context.Context, sessionID string) (*model.Session, error) {
	var session model.Session
	query := `SELECT id, session_id, fingerprint, ip_address, risk_score, attempt_count, blocked_until, expires_at, created_at, updated_at
			  FROM sessions WHERE session_id = $1`
	err := r.db.GetContext(ctx, &session, query, sessionID)
	if err != nil {
		return nil, err
	}
	return &session, nil
}

func (r *SessionRepository) UpdateAttemptCount(ctx context.Context, sessionID string, count int) error {
	query := `UPDATE sessions SET attempt_count = $1, updated_at = $2 WHERE session_id = $3`
	_, err := r.db.ExecContext(ctx, query, count, time.Now(), sessionID)
	return err
}

func (r *SessionRepository) IncrementAttemptCount(ctx context.Context, sessionID string) error {
	query := `UPDATE sessions SET attempt_count = attempt_count + 1, updated_at = $1 WHERE session_id = $2`
	_, err := r.db.ExecContext(ctx, query, time.Now(), sessionID)
	return err
}

func (r *SessionRepository) Delete(ctx context.Context, sessionID string) error {
	query := `DELETE FROM sessions WHERE session_id = $1`
	_, err := r.db.ExecContext(ctx, query, sessionID)
	return err
}
