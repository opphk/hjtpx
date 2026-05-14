package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/opphk/captcha-system/internal/model"
)

type AdminRepository struct {
	*BaseRepository
}

func NewAdminRepository(db *sqlx.DB) *AdminRepository {
	return &AdminRepository{
		BaseRepository: NewBaseRepository(db),
	}
}

func (r *AdminRepository) GetByUsername(ctx context.Context, username string) (*model.AdminUser, error) {
	var admin model.AdminUser
	query := `SELECT id, username, password_hash, role, is_active, last_login_at, created_at, updated_at
              FROM admins WHERE username = $1`
	err := r.db.GetContext(ctx, &admin, query, username)
	if err != nil {
		return nil, err
	}
	return &admin, nil
}

func (r *AdminRepository) GetByID(ctx context.Context, id int) (*model.AdminUser, error) {
	var admin model.AdminUser
	query := `SELECT id, username, password_hash, role, is_active, last_login_at, created_at, updated_at
              FROM admins WHERE id = $1`
	err := r.db.GetContext(ctx, &admin, query, id)
	if err != nil {
		return nil, err
	}
	return &admin, nil
}

func (r *AdminRepository) Create(ctx context.Context, admin *model.AdminUser) error {
	admin.CreatedAt = time.Now()
	admin.UpdatedAt = time.Now()
	query := `INSERT INTO admins (username, password_hash, role, is_active, created_at, updated_at)
              VALUES ($1, $2, $3, $4, $5, $6)`
	_, err := r.db.ExecContext(ctx, query,
		admin.Username,
		admin.PasswordHash,
		admin.Role,
		admin.IsActive,
		admin.CreatedAt,
		admin.UpdatedAt,
	)
	return err
}

func (r *AdminRepository) UpdatePassword(ctx context.Context, id int64, passwordHash string) error {
	query := `UPDATE admins SET password_hash = $1, updated_at = $2 WHERE id = $3`
	_, err := r.db.ExecContext(ctx, query, passwordHash, time.Now(), id)
	return err
}

func (r *AdminRepository) UpdateLastLogin(ctx context.Context, userID int64) error {
	query := `UPDATE admins SET last_login_at = $1 WHERE id = $2`
	_, err := r.db.ExecContext(ctx, query, time.Now(), userID)
	return err
}

func (r *AdminRepository) GetChallenges(ctx context.Context, page, size int, challengeType string) ([]*model.Challenge, int64, error) {
	offset := (page - 1) * size

	var total int64
	var countQuery string
	var selectQuery string
	var args []interface{}

	if challengeType != "" {
		countQuery = `SELECT COUNT(*) FROM challenges WHERE type = $1`
		err := r.db.GetContext(ctx, &total, countQuery, challengeType)
		if err != nil {
			return nil, 0, err
		}

		selectQuery = `SELECT id, challenge_id, type, difficulty, data, solution, expires_at, created_at, updated_at
                      FROM challenges WHERE type = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`
		args = []interface{}{challengeType, size, offset}
	} else {
		countQuery = `SELECT COUNT(*) FROM challenges`
		err := r.db.GetContext(ctx, &total, countQuery)
		if err != nil {
			return nil, 0, err
		}

		selectQuery = `SELECT id, challenge_id, type, difficulty, data, solution, expires_at, created_at, updated_at
                      FROM challenges ORDER BY created_at DESC LIMIT $1 OFFSET $2`
		args = []interface{}{size, offset}
	}

	var challenges []*model.Challenge
	err := r.db.SelectContext(ctx, &challenges, selectQuery, args...)
	if err != nil {
		return nil, 0, err
	}

	return challenges, total, nil
}

func (r *AdminRepository) GetAttempts(ctx context.Context, page, size int) ([]*model.Attempt, int64, error) {
	offset := (page - 1) * size

	var total int64
	countQuery := `SELECT COUNT(*) FROM attempts`
	err := r.db.GetContext(ctx, &total, countQuery)
	if err != nil {
		return nil, 0, err
	}

	selectQuery := `SELECT id, challenge_id, session_id, user_answer, is_valid, response_time_ms,
                    ip_address, user_agent, fingerprint, risk_score, created_at
                    FROM attempts ORDER BY created_at DESC LIMIT $1 OFFSET $2`

	var attempts []*model.Attempt
	err = r.db.SelectContext(ctx, &attempts, selectQuery, size, offset)
	if err != nil {
		return nil, 0, err
	}

	return attempts, total, nil
}

func (r *AdminRepository) UpdateConfig(ctx context.Context, key string, value []byte) error {
	query := `INSERT INTO config (key, value, updated_at) VALUES ($1, $2, $3)
              ON CONFLICT (key) DO UPDATE SET value = $2, updated_at = $3`
	_, err := r.db.ExecContext(ctx, query, key, value, time.Now())
	return err
}

func (r *AdminRepository) GetConfig(ctx context.Context, key string) (string, error) {
	var value string
	query := `SELECT value FROM config WHERE key = $1`
	err := r.db.GetContext(ctx, &value, query, key)
	if err != nil {
		return "", err
	}
	return value, nil
}

func (r *AdminRepository) GetLogs(ctx context.Context, level string, page, size int) ([]*model.Log, int64, error) {
	offset := (page - 1) * size

	var total int64
	var countQuery string
	var selectQuery string
	var args []interface{}

	if level != "" && level != "all" {
		countQuery = `SELECT COUNT(*) FROM logs WHERE level = $1`
		err := r.db.GetContext(ctx, &total, countQuery, level)
		if err != nil {
			return nil, 0, err
		}

		selectQuery = `SELECT id, level, message, metadata, ip_address, user_id, created_at
                      FROM logs WHERE level = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`
		args = []interface{}{level, size, offset}
	} else {
		countQuery = `SELECT COUNT(*) FROM logs`
		err := r.db.GetContext(ctx, &total, countQuery)
		if err != nil {
			return nil, 0, err
		}

		selectQuery = `SELECT id, level, message, metadata, ip_address, user_id, created_at
                      FROM logs ORDER BY created_at DESC LIMIT $1 OFFSET $2`
		args = []interface{}{size, offset}
	}

	var logs []*model.Log
	err := r.db.SelectContext(ctx, &logs, selectQuery, args...)
	if err != nil {
		return nil, 0, err
	}

	return logs, total, nil
}

func (r *AdminRepository) CreateLog(ctx context.Context, log *model.Log) error {
	log.CreatedAt = time.Now()
	query := `INSERT INTO logs (level, message, metadata, ip_address, user_id, created_at)
              VALUES ($1, $2, $3, $4, $5, $6)`
	_, err := r.db.ExecContext(ctx, query,
		log.Level,
		log.Message,
		log.Metadata,
		log.IPAddress,
		log.UserID,
		log.CreatedAt,
	)
	return err
}

func (r *StatsRepository) GetByDate(ctx context.Context, date time.Time) (*model.DailyStats, error) {
	var stats model.DailyStats
	dateStr := date.Format("2006-01-02")

	query := `
		SELECT
			$1 as date,
			COALESCE((SELECT COUNT(*) FROM challenges WHERE DATE(created_at) = $1), 0) as challenge_count,
			COALESCE((SELECT COUNT(*) FROM attempts WHERE DATE(created_at) = $1), 0) as attempt_count,
			COALESCE((SELECT COUNT(*) FROM attempts WHERE DATE(created_at) = $1 AND is_valid = true), 0) as success_count,
			COALESCE((SELECT CASE WHEN COUNT(*) > 0 THEN CAST(SUM(CASE WHEN is_valid THEN 1 ELSE 0 END) AS FLOAT) / COUNT(*) ELSE 0 END FROM attempts WHERE DATE(created_at) = $1), 0) as success_rate,
			COALESCE((SELECT AVG(response_time_ms) FROM attempts WHERE DATE(created_at) = $1), 0) as avg_response_time_ms
	`

	err := r.db.GetContext(ctx, &stats, query, dateStr)
	if err != nil {
		return nil, fmt.Errorf("failed to get stats by date: %w", err)
	}

	stats.Date = dateStr
	return &stats, nil
}

func (r *StatsRepository) GetRange(ctx context.Context, startDate, endDate time.Time) ([]*model.DailyStats, error) {
	query := `
		SELECT
			DATE(created_at) as date,
			COUNT(*) as challenge_count,
			0 as attempt_count,
			0 as success_count,
			0 as success_rate,
			0 as avg_response_time_ms
		FROM challenges
		WHERE DATE(created_at) BETWEEN $1 AND $2
		GROUP BY DATE(created_at)
		ORDER BY date DESC
	`

	var stats []*model.DailyStats
	err := r.db.SelectContext(ctx, &stats, query, startDate.Format("2006-01-02"), endDate.Format("2006-01-02"))
	if err != nil {
		return nil, fmt.Errorf("failed to get stats range: %w", err)
	}

	return stats, nil
}

func (r *StatsRepository) GetTotal(ctx context.Context) (*model.TotalStats, error) {
	var stats model.TotalStats

	query := `
		SELECT
			COALESCE((SELECT COUNT(*) FROM challenges), 0) as total_challenges,
			COALESCE((SELECT COUNT(*) FROM attempts), 0) as total_attempts,
			COALESCE((SELECT COUNT(*) FROM attempts WHERE is_valid = true), 0) as success_count,
			COALESCE((SELECT CASE WHEN COUNT(*) > 0 THEN CAST(SUM(CASE WHEN is_valid THEN 1 ELSE 0 END) AS FLOAT) / COUNT(*) ELSE 0 END FROM attempts), 0) as success_rate,
			COALESCE((SELECT AVG(response_time_ms) FROM attempts), 0) as avg_response_time_ms,
			COALESCE((SELECT COUNT(*) FROM sessions WHERE blocked_until > NOW()), 0) as blocked_sessions
	`

	err := r.db.GetContext(ctx, &stats, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get total stats: %w", err)
	}

	return &stats, nil
}
