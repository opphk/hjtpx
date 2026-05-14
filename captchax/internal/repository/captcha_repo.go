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

func (r *CaptchaRepo) ListOptimized(ctx context.Context, filter *model.CaptchaLogFilter) ([]*model.CaptchaLog, error) {
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

func (r *CaptchaRepo) GetStatsFromMaterialized(ctx context.Context, startDate, endDate time.Time) (*model.CaptchaLogStats, error) {
	stats := &model.CaptchaLogStats{
		ByType: make(map[string]int64),
	}

	query := `
		SELECT 
			COALESCE(SUM(total_count), 0) as total,
			COALESCE(SUM(success_count), 0) as success,
			COALESCE(SUM(fail_count), 0) as fail,
			COALESCE(AVG(avg_duration), 0) as avg_duration,
			COALESCE(AVG(avg_risk_score), 0) as avg_risk
		FROM mv_captcha_daily_stats
		WHERE stat_date >= $1 AND stat_date <= $2
	`
	err := r.db.QueryRowContext(ctx, query, startDate, endDate).Scan(
		&stats.TotalCount, &stats.SuccessCount, &stats.FailCount,
		&stats.AvgDuration, &stats.AvgRiskScore,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get stats from materialized view: %w", err)
	}

	if stats.TotalCount > 0 {
		stats.SuccessRate = float64(stats.SuccessCount) / float64(stats.TotalCount) * 100
	}

	typeQuery := `
		SELECT captcha_type, SUM(total_count)
		FROM mv_captcha_daily_stats
		WHERE stat_date >= $1 AND stat_date <= $2
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

func (r *CaptchaRepo) GetHourlyTrend(ctx context.Context, startDate, endDate time.Time) ([]model.HourlyStat, error) {
	query := `
		SELECT 
			DATE_TRUNC('hour', created_at) as hour,
			COUNT(*) as total_count,
			COUNT(*) FILTER (WHERE result = true) as success_count,
			COUNT(*) FILTER (WHERE result = false) as fail_count
		FROM captcha_logs
		WHERE created_at >= $1 AND created_at < $2 + INTERVAL '1 day'
		GROUP BY DATE_TRUNC('hour', created_at)
		ORDER BY hour
	`
	rows, err := r.db.QueryContext(ctx, query, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to get hourly trend: %w", err)
	}
	defer rows.Close()

	var stats []model.HourlyStat
	for rows.Next() {
		var s model.HourlyStat
		if err := rows.Scan(&s.Hour, &s.TotalCount, &s.SuccessCount, &s.FailCount); err != nil {
			return nil, fmt.Errorf("failed to scan hourly stat: %w", err)
		}
		stats = append(stats, s)
	}
	return stats, nil
}

func (r *CaptchaRepo) CountByIPWithWindow(ctx context.Context, ip string, windowStart, windowEnd time.Time) (int64, error) {
	query := `SELECT COUNT(*) FROM captcha_logs WHERE ip = $1 AND created_at >= $2 AND created_at <= $3`
	var count int64
	err := r.db.QueryRowContext(ctx, query, ip, windowStart, windowEnd).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count by IP window: %w", err)
	}
	return count, nil
}

func (r *CaptchaRepo) GetIPStats(ctx context.Context, limit int) ([]*model.IPStats, error) {
	query := `
		SELECT 
			ip,
			COUNT(*) AS total_attempts,
			COUNT(*) FILTER (WHERE result = true) AS success_count,
			COUNT(*) FILTER (WHERE result = false) AS fail_count,
			AVG(risk_score) AS avg_risk_score,
			MAX(risk_score) AS max_risk_score,
			MIN(created_at) AS first_seen,
			MAX(created_at) AS last_seen,
			COUNT(DISTINCT client_id) AS unique_clients
		FROM captcha_logs
		WHERE created_at >= NOW() - INTERVAL '30 days'
		GROUP BY ip
		HAVING COUNT(*) > 10
		ORDER BY total_attempts DESC
		LIMIT $1
	`
	rows, err := r.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get IP stats: %w", err)
	}
	defer rows.Close()

	var stats []*model.IPStats
	for rows.Next() {
		s := &model.IPStats{}
		if err := rows.Scan(
			&s.IP, &s.TotalAttempts, &s.SuccessCount, &s.FailCount,
			&s.AvgRiskScore, &s.MaxRiskScore, &s.FirstSeen, &s.LastSeen, &s.UniqueClients,
		); err != nil {
			return nil, fmt.Errorf("failed to scan IP stats: %w", err)
		}
		stats = append(stats, s)
	}
	return stats, nil
}

func (r *CaptchaRepo) RefreshMaterializedStats(ctx context.Context) error {
	query := `SELECT refresh_captcha_stats()`
	_, err := r.db.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to refresh materialized stats: %w", err)
	}
	return nil
}

func (r *CaptchaRepo) ArchiveOldLogs(ctx context.Context, before time.Time, batchSize int) (int64, error) {
	query := `SELECT * FROM archive_captcha_logs($1, $2, true)`
	var archived, remaining, execTime int64
	err := r.db.QueryRowContext(ctx, query, before, batchSize).Scan(&archived, &remaining, &execTime)
	if err != nil {
		return 0, fmt.Errorf("failed to archive old logs: %w", err)
	}
	return archived, nil
}

func (r *CaptchaRepo) GetArchiveStats(ctx context.Context) (*model.ArchiveStats, error) {
	query := `SELECT * FROM get_archive_stats()`
	stats := &model.ArchiveStats{}
	err := r.db.QueryRowContext(ctx, query).Scan(
		&stats.TableName, &stats.ActiveCount, &stats.ArchivedCount,
		&stats.ArchiveRatio, &stats.OldestActive, &stats.NewestArchived, &stats.LastArchiveAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get archive stats: %w", err)
	}
	return stats, nil
}
