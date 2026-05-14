package repository

import (
	"context"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/opphk/captcha-system/internal/model"
)

type AttemptRepository struct {
	*BaseRepository
}

func NewAttemptRepository(db *sqlx.DB) *AttemptRepository {
	return &AttemptRepository{
		BaseRepository: NewBaseRepository(db),
	}
}

func (r *AttemptRepository) Create(ctx context.Context, attempt *model.Attempt) error {
	attempt.CreatedAt = time.Now()
	query := `INSERT INTO attempts (challenge_id, session_id, user_answer, is_valid, response_time_ms,
			  ip_address, user_agent, fingerprint, risk_score, created_at)
			  VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`
	_, err := r.db.ExecContext(ctx, query,
		attempt.ChallengeID,
		attempt.SessionID,
		attempt.UserAnswer,
		attempt.IsValid,
		attempt.ResponseTimeMs,
		attempt.IPAddress,
		attempt.UserAgent,
		attempt.Fingerprint,
		attempt.RiskScore,
		attempt.CreatedAt,
	)
	return err
}

func (r *AttemptRepository) GetBySessionID(ctx context.Context, sessionID string) ([]model.Attempt, error) {
	var attempts []model.Attempt
	query := `SELECT id, challenge_id, session_id, user_answer, is_valid, response_time_ms,
			  ip_address, user_agent, fingerprint, risk_score, created_at
			  FROM attempts WHERE session_id = $1 ORDER BY created_at DESC`
	err := r.db.SelectContext(ctx, &attempts, query, sessionID)
	if err != nil {
		return nil, err
	}
	return attempts, nil
}

func (r *AttemptRepository) CountBySessionID(ctx context.Context, sessionID string) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM attempts WHERE session_id = $1`
	err := r.db.GetContext(ctx, &count, query, sessionID)
	return count, err
}

func (r *AttemptRepository) GetByChallengeID(ctx context.Context, challengeID string) ([]model.Attempt, error) {
	var attempts []model.Attempt
	query := `SELECT id, challenge_id, session_id, user_answer, is_valid, response_time_ms,
			  ip_address, user_agent, fingerprint, risk_score, created_at
			  FROM attempts WHERE challenge_id = $1 ORDER BY created_at DESC`
	err := r.db.SelectContext(ctx, &attempts, query, challengeID)
	if err != nil {
		return nil, err
	}
	return attempts, nil
}
