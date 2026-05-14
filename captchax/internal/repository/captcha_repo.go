package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"captchax/internal/model"
)

type CaptchaRepo struct {
	db *sql.DB
}

func NewCaptchaRepo(db *sql.DB) *CaptchaRepo {
	return &CaptchaRepo{db: db}
}

func (r *CaptchaRepo) Create(ctx context.Context, log *model.CaptchaLog) (int64, error) {
	query := `
		INSERT INTO captcha_logs (captcha_type, client_id, ip, user_agent, result, duration, risk_score, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id
	`
	now := time.Now()
	var userAgent interface{}
	if log.UserAgent.Valid {
		userAgent = log.UserAgent.String
	}

	var id int64
	err := r.db.QueryRowContext(ctx, query,
		log.Type, log.ClientID, log.IP, userAgent, log.Result, log.Duration, log.RiskScore, now,
	).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("failed to create captcha log: %w", err)
	}
	return id, nil
}

func (r *CaptchaRepo) GetByID(ctx context.Context, id int64) (*model.CaptchaLog, error) {
	query := `
		SELECT id, captcha_type, client_id, ip, user_agent, result, duration, risk_score, created_at
		FROM captcha_logs WHERE id = $1
	`
	log := &model.CaptchaLog{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&log.ID, &log.Type, &log.ClientID, &log.IP, &log.UserAgent,
		&log.Result, &log.Duration, &log.RiskScore, &log.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get captcha log: %w", err)
	}
	return log, nil
}

func (r *CaptchaRepo) List(ctx context.Context, filter *model.CaptchaLogFilter) ([]*model.CaptchaLog, error) {
	var conditions []string
	var args []interface{}
	argIdx := 1

	if filter.StartDate != nil {
		conditions = append(conditions, fmt.Sprintf("created_at >= $%d", argIdx))
		args = append(args, *filter.StartDate)
		argIdx++
	}
	if filter.EndDate != nil {
		conditions = append(conditions, fmt.Sprintf("created_at <= $%d", argIdx))
		args = append(args, *filter.EndDate)
		argIdx++
	}
	if filter.Type != "" {
		conditions = append(conditions, fmt.Sprintf("captcha_type = $%d", argIdx))
		args = append(args, filter.Type)
		argIdx++
	}
	if filter.ClientID != "" {
		conditions = append(conditions, fmt.Sprintf("client_id = $%d", argIdx))
		args = append(args, filter.ClientID)
		argIdx++
	}
	if filter.IP != "" {
		conditions = append(conditions, fmt.Sprintf("ip = $%d", argIdx))
		args = append(args, filter.IP)
		argIdx++
	}
	if filter.Result != nil {
		conditions = append(conditions, fmt.Sprintf("result = $%d", argIdx))
		args = append(args, *filter.Result)
		argIdx++
	}
	if filter.MinScore > 0 {
		conditions = append(conditions, fmt.Sprintf("risk_score >= $%d", argIdx))
		args = append(args, filter.MinScore)
		argIdx++
	}
	if filter.MaxScore > 0 {
		conditions = append(conditions, fmt.Sprintf("risk_score <= $%d", argIdx))
		args = append(args, filter.MaxScore)
		argIdx++
	}

	where := ""
	if len(conditions) > 0 {
		where = "WHERE " + strings.Join(conditions, " AND ")
	}

	query := fmt.Sprintf(`
		SELECT id, captcha_type, client_id, ip, user_agent, result, duration, risk_score, created_at
		FROM captcha_logs %s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d
	`, where, argIdx, argIdx+1)

	args = append(args, filter.Limit(), filter.Offset())

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list captcha logs: %w", err)
	}
	defer rows.Close()

	var logs []*model.CaptchaLog
	for rows.Next() {
		log := &model.CaptchaLog{}
		err := rows.Scan(
			&log.ID, &log.Type, &log.ClientID, &log.IP, &log.UserAgent,
			&log.Result, &log.Duration, &log.RiskScore, &log.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan captcha log: %w", err)
		}
		logs = append(logs, log)
	}
	return logs, nil
}

func (r *CaptchaRepo) CountByIP(ctx context.Context, ip string, since time.Time) (int64, error) {
	query := `SELECT COUNT(*) FROM captcha_logs WHERE ip = $1 AND created_at >= $2`
	var count int64
	err := r.db.QueryRowContext(ctx, query, ip, since).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count by IP: %w", err)
	}
	return count, nil
}

func (r *CaptchaRepo) GetStats(ctx context.Context, startDate, endDate time.Time) (*model.CaptchaLogStats, error) {
	stats := &model.CaptchaLogStats{
		ByType: make(map[string]int64),
	}

	statsQuery := `
		SELECT 
			COUNT(*) as total,
			COUNT(*) FILTER (WHERE result = true) as success,
			COUNT(*) FILTER (WHERE result = false) as fail,
			COALESCE(AVG(duration), 0) as avg_duration,
			COALESCE(AVG(risk_score), 0) as avg_risk
		FROM captcha_logs
		WHERE created_at >= $1 AND created_at <= $2
	`
	err := r.db.QueryRowContext(ctx, statsQuery, startDate, endDate).Scan(
		&stats.TotalCount, &stats.SuccessCount, &stats.FailCount,
		&stats.AvgDuration, &stats.AvgRiskScore,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get stats: %w", err)
	}

	if stats.TotalCount > 0 {
		stats.SuccessRate = float64(stats.SuccessCount) / float64(stats.TotalCount) * 100
	}

	typeQuery := `
		SELECT captcha_type, COUNT(*) 
		FROM captcha_logs
		WHERE created_at >= $1 AND created_at <= $2
		GROUP BY captcha_type
	`
	rows, err := r.db.QueryContext(ctx, typeQuery, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to get type stats: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var t string
		var count int64
		if err := rows.Scan(&t, &count); err != nil {
			return nil, fmt.Errorf("failed to scan type stat: %w", err)
		}
		stats.ByType[t] = count
	}

	return stats, nil
}

func (r *CaptchaRepo) DeleteOlderThan(ctx context.Context, before time.Time) (int64, error) {
	query := `DELETE FROM captcha_logs WHERE created_at < $1`
	result, err := r.db.ExecContext(ctx, query, before)
	if err != nil {
		return 0, fmt.Errorf("failed to delete old logs: %w", err)
	}
	affected, _ := result.RowsAffected()
	return affected, nil
}
