package repository

import (
	"context"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/opphk/captcha-system/internal/model"
)

type StatsRepository struct {
	*BaseRepository
}

func NewStatsRepository(db *sqlx.DB) *StatsRepository {
	return &StatsRepository{
		BaseRepository: NewBaseRepository(db),
	}
}

func (r *StatsRepository) GetTotalChallenges(ctx context.Context) (int64, error) {
	var count int64
	query := `SELECT COUNT(*) FROM challenges`
	err := r.db.GetContext(ctx, &count, query)
	return count, err
}

func (r *StatsRepository) GetTotalAttempts(ctx context.Context) (int64, error) {
	var count int64
	query := `SELECT COUNT(*) FROM attempts`
	err := r.db.GetContext(ctx, &count, query)
	return count, err
}

func (r *StatsRepository) GetSuccessRate(ctx context.Context) (float64, error) {
	var result struct {
		Total   int64 `db:"total"`
		Success int64 `db:"success"`
	}
	query := `SELECT COUNT(*) as total, SUM(CASE WHEN is_valid THEN 1 ELSE 0 END) as success FROM attempts`
	err := r.db.GetContext(ctx, &result, query)
	if err != nil {
		return 0, err
	}
	if result.Total == 0 {
		return 0, nil
	}
	return float64(result.Success) / float64(result.Total), nil
}

func (r *StatsRepository) GetAvgResponseTime(ctx context.Context) (int64, error) {
	var avg int64
	query := `SELECT COALESCE(AVG(response_time_ms), 0) FROM attempts`
	err := r.db.GetContext(ctx, &avg, query)
	return avg, err
}

func (r *StatsRepository) GetBlockedSessionsCount(ctx context.Context) (int64, error) {
	var count int64
	query := `SELECT COUNT(*) FROM sessions WHERE blocked_until > $1`
	err := r.db.GetContext(ctx, &count, query, time.Now())
	return count, err
}

func (r *StatsRepository) GetAttemptsByDate(ctx context.Context, startDate, endDate time.Time) ([]model.Attempt, error) {
	var attempts []model.Attempt
	query := `SELECT id, challenge_id, session_id, user_answer, is_valid, response_time_ms,
              ip_address, user_agent, fingerprint, risk_score, created_at
              FROM attempts WHERE created_at BETWEEN $1 AND $2 ORDER BY created_at DESC`
	err := r.db.SelectContext(ctx, &attempts, query, startDate, endDate)
	return attempts, err
}
