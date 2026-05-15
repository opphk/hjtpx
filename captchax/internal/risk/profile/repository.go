package profile

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type ProfileRepo struct {
	db *sql.DB
}

func NewProfileRepo(db *sql.DB) *ProfileRepo {
	return &ProfileRepo{db: db}
}

func (r *ProfileRepo) Create(ctx context.Context, profile *UserProfile) (int64, error) {
	captchaDistJSON, err := json.Marshal(profile.CaptchaTypeDistribution)
	if err != nil {
		captchaDistJSON = []byte("{}")
	}

	activeHoursJSON, err := json.Marshal(profile.ActiveHours)
	if err != nil {
		activeHoursJSON = []byte("{}")
	}

	activeDaysJSON, err := json.Marshal(profile.ActiveDays)
	if err != nil {
		activeDaysJSON = []byte("{}")
	}

	locationDistJSON, err := json.Marshal(profile.LocationDistribution)
	if err != nil {
		locationDistJSON = []byte("{}")
	}

	deviceDistJSON, err := json.Marshal(profile.DeviceDistribution)
	if err != nil {
		deviceDistJSON = []byte("{}")
	}

	query := `
		INSERT INTO user_profiles (
			identifier, identifier_type, ip, device_fingerprint, cookie_id, session_id,
			total_attempts, success_count, fail_count, success_rate,
			avg_response_time, min_response_time, max_response_time,
			preferred_captcha_type, captcha_type_distribution,
			active_hours, active_days,
			location_distribution, device_distribution,
			total_risk_events, high_risk_events, last_risk_event_at,
			first_seen_at, last_seen_at, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24, $25, $26)
		RETURNING id
	`

	now := time.Now()
	var lastRiskEventAt interface{}
	if profile.LastRiskEventAt != nil {
		lastRiskEventAt = profile.LastRiskEventAt
	}

	var id int64
	err = r.db.QueryRowContext(ctx, query,
		profile.Identifier, profile.IdentifierType, profile.IP, profile.DeviceFingerprint,
		profile.CookieID, profile.SessionID,
		profile.TotalAttempts, profile.SuccessCount, profile.FailCount, profile.SuccessRate,
		profile.AvgResponseTime, profile.MinResponseTime, profile.MaxResponseTime,
		profile.PreferredCaptchaType, captchaDistJSON,
		activeHoursJSON, activeDaysJSON,
		locationDistJSON, deviceDistJSON,
		profile.TotalRiskEvents, profile.HighRiskEvents, lastRiskEventAt,
		profile.FirstSeenAt, profile.LastSeenAt, now, now,
	).Scan(&id)

	if err != nil {
		return 0, fmt.Errorf("failed to create profile: %w", err)
	}

	return id, nil
}

func (r *ProfileRepo) Update(ctx context.Context, profile *UserProfile) error {
	captchaDistJSON, err := json.Marshal(profile.CaptchaTypeDistribution)
	if err != nil {
		captchaDistJSON = []byte("{}")
	}

	activeHoursJSON, err := json.Marshal(profile.ActiveHours)
	if err != nil {
		activeHoursJSON = []byte("{}")
	}

	activeDaysJSON, err := json.Marshal(profile.ActiveDays)
	if err != nil {
		activeDaysJSON = []byte("{}")
	}

	locationDistJSON, err := json.Marshal(profile.LocationDistribution)
	if err != nil {
		locationDistJSON = []byte("{}")
	}

	deviceDistJSON, err := json.Marshal(profile.DeviceDistribution)
	if err != nil {
		deviceDistJSON = []byte("{}")
	}

	query := `
		UPDATE user_profiles SET
			ip = $2, device_fingerprint = $3, cookie_id = $4, session_id = $5,
			total_attempts = $6, success_count = $7, fail_count = $8, success_rate = $9,
			avg_response_time = $10, min_response_time = $11, max_response_time = $12,
			preferred_captcha_type = $13, captcha_type_distribution = $14,
			active_hours = $15, active_days = $16,
			location_distribution = $17, device_distribution = $18,
			total_risk_events = $19, high_risk_events = $20, last_risk_event_at = $21,
			last_seen_at = $22, updated_at = $23
		WHERE id = $1
	`

	var lastRiskEventAt interface{}
	if profile.LastRiskEventAt != nil {
		lastRiskEventAt = profile.LastRiskEventAt
	}

	_, err = r.db.ExecContext(ctx, query,
		profile.ID,
		profile.IP, profile.DeviceFingerprint, profile.CookieID, profile.SessionID,
		profile.TotalAttempts, profile.SuccessCount, profile.FailCount, profile.SuccessRate,
		profile.AvgResponseTime, profile.MinResponseTime, profile.MaxResponseTime,
		profile.PreferredCaptchaType, captchaDistJSON,
		activeHoursJSON, activeDaysJSON,
		locationDistJSON, deviceDistJSON,
		profile.TotalRiskEvents, profile.HighRiskEvents, lastRiskEventAt,
		profile.LastSeenAt, time.Now(),
	)

	if err != nil {
		return fmt.Errorf("failed to update profile: %w", err)
	}

	return nil
}

func (r *ProfileRepo) GetByID(ctx context.Context, id int64) (*UserProfile, error) {
	query := `
		SELECT id, identifier, identifier_type, ip, device_fingerprint, cookie_id, session_id,
			total_attempts, success_count, fail_count, success_rate,
			avg_response_time, min_response_time, max_response_time,
			preferred_captcha_type, captcha_type_distribution,
			active_hours, active_days,
			location_distribution, device_distribution,
			total_risk_events, high_risk_events, last_risk_event_at,
			first_seen_at, last_seen_at, created_at, updated_at
		FROM user_profiles WHERE id = $1
	`

	profile := &UserProfile{}
	var captchaDistJSON, activeHoursJSON, activeDaysJSON []byte
	var locationDistJSON, deviceDistJSON []byte
	var lastRiskEventAt sql.NullTime

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&profile.ID, &profile.Identifier, &profile.IdentifierType, &profile.IP,
		&profile.DeviceFingerprint, &profile.CookieID, &profile.SessionID,
		&profile.TotalAttempts, &profile.SuccessCount, &profile.FailCount, &profile.SuccessRate,
		&profile.AvgResponseTime, &profile.MinResponseTime, &profile.MaxResponseTime,
		&profile.PreferredCaptchaType, &captchaDistJSON,
		&activeHoursJSON, &activeDaysJSON,
		&locationDistJSON, &deviceDistJSON,
		&profile.TotalRiskEvents, &profile.HighRiskEvents, &lastRiskEventAt,
		&profile.FirstSeenAt, &profile.LastSeenAt, &profile.CreatedAt, &profile.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get profile: %w", err)
	}

	if lastRiskEventAt.Valid {
		profile.LastRiskEventAt = &lastRiskEventAt.Time
	}

	json.Unmarshal(captchaDistJSON, &profile.CaptchaTypeDistribution)
	json.Unmarshal(activeHoursJSON, &profile.ActiveHours)
	json.Unmarshal(activeDaysJSON, &profile.ActiveDays)
	json.Unmarshal(locationDistJSON, &profile.LocationDistribution)
	json.Unmarshal(deviceDistJSON, &profile.DeviceDistribution)

	return profile, nil
}

func (r *ProfileRepo) GetByIdentifier(ctx context.Context, identifier string, identifierType IdentifierType) (*UserProfile, error) {
	query := `
		SELECT id, identifier, identifier_type, ip, device_fingerprint, cookie_id, session_id,
			total_attempts, success_count, fail_count, success_rate,
			avg_response_time, min_response_time, max_response_time,
			preferred_captcha_type, captcha_type_distribution,
			active_hours, active_days,
			location_distribution, device_distribution,
			total_risk_events, high_risk_events, last_risk_event_at,
			first_seen_at, last_seen_at, created_at, updated_at
		FROM user_profiles
		WHERE identifier = $1 AND identifier_type = $2
	`

	profile := &UserProfile{}
	var captchaDistJSON, activeHoursJSON, activeDaysJSON []byte
	var locationDistJSON, deviceDistJSON []byte
	var lastRiskEventAt sql.NullTime

	err := r.db.QueryRowContext(ctx, query, identifier, identifierType).Scan(
		&profile.ID, &profile.Identifier, &profile.IdentifierType, &profile.IP,
		&profile.DeviceFingerprint, &profile.CookieID, &profile.SessionID,
		&profile.TotalAttempts, &profile.SuccessCount, &profile.FailCount, &profile.SuccessRate,
		&profile.AvgResponseTime, &profile.MinResponseTime, &profile.MaxResponseTime,
		&profile.PreferredCaptchaType, &captchaDistJSON,
		&activeHoursJSON, &activeDaysJSON,
		&locationDistJSON, &deviceDistJSON,
		&profile.TotalRiskEvents, &profile.HighRiskEvents, &lastRiskEventAt,
		&profile.FirstSeenAt, &profile.LastSeenAt, &profile.CreatedAt, &profile.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get profile by identifier: %w", err)
	}

	if lastRiskEventAt.Valid {
		profile.LastRiskEventAt = &lastRiskEventAt.Time
	}

	json.Unmarshal(captchaDistJSON, &profile.CaptchaTypeDistribution)
	json.Unmarshal(activeHoursJSON, &profile.ActiveHours)
	json.Unmarshal(activeDaysJSON, &profile.ActiveDays)
	json.Unmarshal(locationDistJSON, &profile.LocationDistribution)
	json.Unmarshal(deviceDistJSON, &profile.DeviceDistribution)

	return profile, nil
}

func (r *ProfileRepo) List(ctx context.Context, filter *ProfileFilter) ([]*UserProfile, error) {
	var conditions []string
	var args []interface{}
	argIdx := 1

	if filter.Identifier != "" {
		conditions = append(conditions, fmt.Sprintf("identifier = $%d", argIdx))
		args = append(args, filter.Identifier)
		argIdx++
	}

	if filter.IdentifierType != "" {
		conditions = append(conditions, fmt.Sprintf("identifier_type = $%d", argIdx))
		args = append(args, filter.IdentifierType)
		argIdx++
	}

	if filter.TrustLevel != "" {
		trustCondition := r.buildTrustLevelCondition(filter.TrustLevel)
		conditions = append(conditions, fmt.Sprintf("(%s)", trustCondition))
		argIdx += 3
	}

	if filter.DateFrom != nil {
		conditions = append(conditions, fmt.Sprintf("first_seen_at >= $%d", argIdx))
		args = append(args, *filter.DateFrom)
		argIdx++
	}

	if filter.DateTo != nil {
		conditions = append(conditions, fmt.Sprintf("first_seen_at <= $%d", argIdx))
		args = append(args, *filter.DateTo)
		argIdx++
	}

	where := ""
	if len(conditions) > 0 {
		where = "WHERE " + strings.Join(conditions, " AND ")
	}

	query := fmt.Sprintf(`
		SELECT id, identifier, identifier_type, ip, device_fingerprint, cookie_id, session_id,
			total_attempts, success_count, fail_count, success_rate,
			avg_response_time, min_response_time, max_response_time,
			preferred_captcha_type, captcha_type_distribution,
			active_hours, active_days,
			location_distribution, device_distribution,
			total_risk_events, high_risk_events, last_risk_event_at,
			first_seen_at, last_seen_at, created_at, updated_at
		FROM user_profiles %s
		ORDER BY last_seen_at DESC
		LIMIT $%d OFFSET $%d
	`, where, argIdx, argIdx+1)

	args = append(args, filter.Limit(), filter.Offset())

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list profiles: %w", err)
	}
	defer rows.Close()

	var profiles []*UserProfile
	for rows.Next() {
		profile := &UserProfile{}
		var captchaDistJSON, activeHoursJSON, activeDaysJSON []byte
		var locationDistJSON, deviceDistJSON []byte
		var lastRiskEventAt sql.NullTime

		err := rows.Scan(
			&profile.ID, &profile.Identifier, &profile.IdentifierType, &profile.IP,
			&profile.DeviceFingerprint, &profile.CookieID, &profile.SessionID,
			&profile.TotalAttempts, &profile.SuccessCount, &profile.FailCount, &profile.SuccessRate,
			&profile.AvgResponseTime, &profile.MinResponseTime, &profile.MaxResponseTime,
			&profile.PreferredCaptchaType, &captchaDistJSON,
			&activeHoursJSON, &activeDaysJSON,
			&locationDistJSON, &deviceDistJSON,
			&profile.TotalRiskEvents, &profile.HighRiskEvents, &lastRiskEventAt,
			&profile.FirstSeenAt, &profile.LastSeenAt, &profile.CreatedAt, &profile.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan profile: %w", err)
		}

		if lastRiskEventAt.Valid {
			profile.LastRiskEventAt = &lastRiskEventAt.Time
		}

		json.Unmarshal(captchaDistJSON, &profile.CaptchaTypeDistribution)
		json.Unmarshal(activeHoursJSON, &profile.ActiveHours)
		json.Unmarshal(activeDaysJSON, &profile.ActiveDays)
		json.Unmarshal(locationDistJSON, &profile.LocationDistribution)
		json.Unmarshal(deviceDistJSON, &profile.DeviceDistribution)

		profiles = append(profiles, profile)
	}

	return profiles, nil
}

func (r *ProfileRepo) buildTrustLevelCondition(level TrustLevel) string {
	switch level {
	case TrustLevelTrusted:
		return "success_rate >= 90 AND (total_risk_events = 0 OR high_risk_events * 1.0 / NULLIF(total_risk_events, 0) < 0.1)"
	case TrustLevelSuspicious:
		return "NOT (success_rate >= 90 AND (total_risk_events = 0 OR high_risk_events * 1.0 / NULLIF(total_risk_events, 0) < 0.1)) AND NOT (success_rate < 50 OR high_risk_events * 1.0 / NULLIF(total_risk_events, 0) > 0.5)"
	case TrustLevelHighRisk:
		return "success_rate < 50 OR high_risk_events * 1.0 / NULLIF(total_risk_events, 0) > 0.5"
	default:
		return "1=1"
	}
}

func (r *ProfileRepo) Count(ctx context.Context, filter *ProfileFilter) (int64, error) {
	var conditions []string
	var args []interface{}
	argIdx := 1

	if filter.Identifier != "" {
		conditions = append(conditions, fmt.Sprintf("identifier = $%d", argIdx))
		args = append(args, filter.Identifier)
		argIdx++
	}

	if filter.IdentifierType != "" {
		conditions = append(conditions, fmt.Sprintf("identifier_type = $%d", argIdx))
		args = append(args, filter.IdentifierType)
		argIdx++
	}

	if filter.TrustLevel != "" {
		trustCondition := r.buildTrustLevelCondition(filter.TrustLevel)
		conditions = append(conditions, fmt.Sprintf("(%s)", trustCondition))
		argIdx += 3
	}

	where := ""
	if len(conditions) > 0 {
		where = "WHERE " + strings.Join(conditions, " AND ")
	}

	query := fmt.Sprintf("SELECT COUNT(*) FROM user_profiles %s", where)

	var count int64
	err := r.db.QueryRowContext(ctx, query, args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count profiles: %w", err)
	}

	return count, nil
}

func (r *ProfileRepo) Delete(ctx context.Context, id int64) error {
	query := `DELETE FROM user_profiles WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete profile: %w", err)
	}
	return nil
}

func (r *ProfileRepo) DeleteOlderThan(ctx context.Context, before time.Time) (int64, error) {
	query := `DELETE FROM user_profiles WHERE last_seen_at < $1`
	result, err := r.db.ExecContext(ctx, query, before)
	if err != nil {
		return 0, fmt.Errorf("failed to delete old profiles: %w", err)
	}

	affected, _ := result.RowsAffected()
	return affected, nil
}

func (r *ProfileRepo) GetStats(ctx context.Context) (*ProfileStats, error) {
	stats := &ProfileStats{}

	query := `
		SELECT 
			COUNT(*) as total_profiles,
			COUNT(*) FILTER (WHERE success_rate >= 90) as trusted_profiles,
			COUNT(*) FILTER (WHERE success_rate < 50) as high_risk_profiles,
			COALESCE(SUM(total_attempts), 0) as total_verifications,
			COALESCE(AVG(success_rate), 0) as avg_success_rate,
			COALESCE(AVG(avg_response_time), 0) as avg_response_time
		FROM user_profiles
	`

	err := r.db.QueryRowContext(ctx, query).Scan(
		&stats.TotalProfiles,
		&stats.TrustedProfiles,
		&stats.HighRiskProfiles,
		&stats.TotalVerifications,
		&stats.AvgSuccessRate,
		&stats.AvgResponseTime,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get profile stats: %w", err)
	}

	return stats, nil
}

type ProfileStats struct {
	TotalProfiles      int64   `json:"total_profiles"`
	TrustedProfiles    int64   `json:"trusted_profiles"`
	HighRiskProfiles   int64   `json:"high_risk_profiles"`
	SuspiciousProfiles int64   `json:"suspicious_profiles"`
	TotalVerifications int64   `json:"total_verifications"`
	AvgSuccessRate     float64 `json:"avg_success_rate"`
	AvgResponseTime    float64 `json:"avg_response_time"`
}

func (r *ProfileRepo) IncrementAttempts(ctx context.Context, identifier string, identifierType IdentifierType, success bool, responseTime int64, captchaType string) error {
	var successIncrement, failIncrement string
	if success {
		successIncrement = "success_count = success_count + 1"
		failIncrement = ""
	} else {
		successIncrement = ""
		failIncrement = "fail_count = fail_count + 1"
	}

	incrementClause := successIncrement
	if failIncrement != "" {
		if incrementClause != "" {
			incrementClause += ", "
		}
		incrementClause += failIncrement
	}

	query := fmt.Sprintf(`
		UPDATE user_profiles SET
			%s,
			total_attempts = total_attempts + 1,
			avg_response_time = (avg_response_time * total_attempts + %d) / (total_attempts + 1),
			last_seen_at = NOW(),
			updated_at = NOW()
		WHERE identifier = $1 AND identifier_type = $2
	`, incrementClause, responseTime)

	_, err := r.db.ExecContext(ctx, query, identifier, identifierType)
	if err != nil {
		return fmt.Errorf("failed to increment attempts: %w", err)
	}

	return nil
}

func (r *ProfileRepo) AddRiskEvent(ctx context.Context, identifier string, identifierType IdentifierType, highRisk bool) error {
	var highRiskIncrement string
	if highRisk {
		highRiskIncrement = ", high_risk_events = high_risk_events + 1"
	}

	query := fmt.Sprintf(`
		UPDATE user_profiles SET
			total_risk_events = total_risk_events + 1,
			last_risk_event_at = NOW(),
			updated_at = NOW()
			%s
		WHERE identifier = $1 AND identifier_type = $2
	`, highRiskIncrement)

	_, err := r.db.ExecContext(ctx, query, identifier, identifierType)
	if err != nil {
		return fmt.Errorf("failed to add risk event: %w", err)
	}

	return nil
}
