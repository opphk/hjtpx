package profile

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"time"
)

type Builder struct {
	repo      *ProfileRepo
	captchaDB *sql.DB
}

func NewBuilder(repo *ProfileRepo, captchaDB *sql.DB) *Builder {
	return &Builder{
		repo:      repo,
		captchaDB: captchaDB,
	}
}

func (b *Builder) BuildProfile(ctx context.Context, identifier string, identifierType IdentifierType) (*UserProfile, error) {
	profile, err := b.repo.GetByIdentifier(ctx, identifier, identifierType)
	if err != nil {
		return nil, fmt.Errorf("failed to get existing profile: %w", err)
	}

	if profile == nil {
		profile = &UserProfile{
			Identifier:              identifier,
			IdentifierType:          identifierType,
			CaptchaTypeDistribution: make(map[string]int64),
			ActiveHours:             make(map[int]int64),
			ActiveDays:              make(map[int]int64),
			LocationDistribution:    make(map[string]int64),
			DeviceDistribution:      make(map[string]int64),
			FirstSeenAt:             time.Now(),
			LastSeenAt:              time.Now(),
			CreatedAt:               time.Now(),
			UpdatedAt:               time.Now(),
		}
	}

	if err := b.enrichFromCaptchaLogs(ctx, profile); err != nil {
		return nil, fmt.Errorf("failed to enrich from captcha logs: %w", err)
	}

	if err := b.enrichFromRiskEvents(ctx, profile); err != nil {
		return nil, fmt.Errorf("failed to enrich from risk events: %w", err)
	}

	b.calculateAggregates(profile)

	if profile.ID == 0 {
		id, err := b.repo.Create(ctx, profile)
		if err != nil {
			return nil, fmt.Errorf("failed to create profile: %w", err)
		}
		profile.ID = id
	} else {
		if err := b.repo.Update(ctx, profile); err != nil {
			return nil, fmt.Errorf("failed to update profile: %w", err)
		}
	}

	return profile, nil
}

func (b *Builder) enrichFromCaptchaLogs(ctx context.Context, profile *UserProfile) error {
	columnName := b.getIdentifierColumn(profile.IdentifierType)
	if columnName == "" {
		columnName = "ip"
	}

	query := `
		SELECT 
			COUNT(*) as total_attempts,
			COUNT(*) FILTER (WHERE result = true) as success_count,
			COUNT(*) FILTER (WHERE result = false) as fail_count,
			COALESCE(AVG(duration), 0) as avg_duration,
			COALESCE(MIN(duration), 0) as min_duration,
			COALESCE(MAX(duration), 0) as max_duration,
			MIN(created_at) as first_seen,
			MAX(created_at) as last_seen,
			COUNT(DISTINCT captcha_type) as unique_types
		FROM captcha_logs
		WHERE ` + columnName + ` = $1
	`

	var firstSeen, lastSeen sql.NullTime
	var totalAttempts, successCount, failCount int64
	var avgDuration, minDuration, maxDuration float64

	err := b.captchaDB.QueryRowContext(ctx, query, profile.Identifier).Scan(
		&totalAttempts, &successCount, &failCount,
		&avgDuration, &minDuration, &maxDuration,
		&firstSeen, &lastSeen,
	)
	if err != nil && err != sql.ErrNoRows {
		return fmt.Errorf("failed to query captcha logs: %w", err)
	}

	profile.TotalAttempts = totalAttempts
	profile.SuccessCount = successCount
	profile.FailCount = failCount

	if totalAttempts > 0 {
		profile.SuccessRate = float64(successCount) / float64(totalAttempts) * 100
	}

	profile.AvgResponseTime = avgDuration
	profile.MinResponseTime = minDuration
	profile.MaxResponseTime = maxDuration

	if firstSeen.Valid {
		profile.FirstSeenAt = firstSeen.Time
	}
	if lastSeen.Valid {
		profile.LastSeenAt = lastSeen.Time
	}

	if err := b.enrichCaptchaTypeDistribution(ctx, profile); err != nil {
		return fmt.Errorf("failed to enrich captcha type distribution: %w", err)
	}

	if err := b.enrichActiveHours(ctx, profile); err != nil {
		return fmt.Errorf("failed to enrich active hours: %w", err)
	}

	if err := b.enrichActiveDays(ctx, profile); err != nil {
		return fmt.Errorf("failed to enrich active days: %w", err)
	}

	if err := b.enrichPreferredCaptchaType(ctx, profile); err != nil {
		return fmt.Errorf("failed to enrich preferred captcha type: %w", err)
	}

	return nil
}

func (b *Builder) getIdentifierColumn(identifierType IdentifierType) string {
	switch identifierType {
	case IdentifierTypeIP:
		return "ip"
	case IdentifierTypeDevice:
		return "client_id"
	case IdentifierTypeCookie:
		return "client_id"
	case IdentifierTypeSession:
		return "client_id"
	default:
		return "ip"
	}
}

func (b *Builder) enrichCaptchaTypeDistribution(ctx context.Context, profile *UserProfile) error {
	columnName := b.getIdentifierColumn(profile.IdentifierType)

	query := `
		SELECT captcha_type, COUNT(*) as count
		FROM captcha_logs
		WHERE ` + columnName + ` = $1
		GROUP BY captcha_type
	`

	rows, err := b.captchaDB.QueryContext(ctx, query, profile.Identifier)
	if err != nil {
		return fmt.Errorf("failed to query captcha type distribution: %w", err)
	}
	defer rows.Close()

	distribution := make(map[string]int64)
	for rows.Next() {
		var captchaType string
		var count int64
		if err := rows.Scan(&captchaType, &count); err != nil {
			return fmt.Errorf("failed to scan captcha type: %w", err)
		}
		distribution[captchaType] = count
	}

	profile.CaptchaTypeDistribution = distribution
	return nil
}

func (b *Builder) enrichActiveHours(ctx context.Context, profile *UserProfile) error {
	columnName := b.getIdentifierColumn(profile.IdentifierType)

	query := `
		SELECT EXTRACT(HOUR FROM created_at) as hour, COUNT(*) as count
		FROM captcha_logs
		WHERE ` + columnName + ` = $1
		GROUP BY EXTRACT(HOUR FROM created_at)
	`

	rows, err := b.captchaDB.QueryContext(ctx, query, profile.Identifier)
	if err != nil {
		return fmt.Errorf("failed to query active hours: %w", err)
	}
	defer rows.Close()

	activeHours := make(map[int]int64)
	for rows.Next() {
		var hour int
		var count int64
		if err := rows.Scan(&hour, &count); err != nil {
			return fmt.Errorf("failed to scan active hour: %w", err)
		}
		activeHours[hour] = count
	}

	profile.ActiveHours = activeHours
	return nil
}

func (b *Builder) enrichActiveDays(ctx context.Context, profile *UserProfile) error {
	columnName := b.getIdentifierColumn(profile.IdentifierType)

	query := `
		SELECT EXTRACT(DOW FROM created_at) as day, COUNT(*) as count
		FROM captcha_logs
		WHERE ` + columnName + ` = $1
		GROUP BY EXTRACT(DOW FROM created_at)
	`

	rows, err := b.captchaDB.QueryContext(ctx, query, profile.Identifier)
	if err != nil {
		return fmt.Errorf("failed to query active days: %w", err)
	}
	defer rows.Close()

	activeDays := make(map[int]int64)
	for rows.Next() {
		var day int
		var count int64
		if err := rows.Scan(&day, &count); err != nil {
			return fmt.Errorf("failed to scan active day: %w", err)
		}
		activeDays[day] = count
	}

	profile.ActiveDays = activeDays
	return nil
}

func (b *Builder) enrichPreferredCaptchaType(ctx context.Context, profile *UserProfile) error {
	if len(profile.CaptchaTypeDistribution) == 0 {
		profile.PreferredCaptchaType = ""
		return nil
	}

	var maxType string
	var maxCount int64
	for captchaType, count := range profile.CaptchaTypeDistribution {
		if count > maxCount {
			maxCount = count
			maxType = captchaType
		}
	}

	profile.PreferredCaptchaType = maxType
	return nil
}

func (b *Builder) enrichFromRiskEvents(ctx context.Context, profile *UserProfile) error {
	columnName := b.getIdentifierColumn(profile.IdentifierType)

	query := `
		SELECT 
			COUNT(*) as total_risk_events,
			COUNT(*) FILTER (WHERE risk_score >= 70) as high_risk_events,
			MAX(created_at) as last_risk_event
		FROM captcha_logs
		WHERE ` + columnName + ` = $1 AND risk_score > 0
	`

	var lastRiskEvent sql.NullTime
	var totalRiskEvents, highRiskEvents int64

	err := b.captchaDB.QueryRowContext(ctx, query, profile.Identifier).Scan(
		&totalRiskEvents, &highRiskEvents, &lastRiskEvent,
	)
	if err != nil && err != sql.ErrNoRows {
		return fmt.Errorf("failed to query risk events: %w", err)
	}

	profile.TotalRiskEvents = totalRiskEvents
	profile.HighRiskEvents = highRiskEvents

	if lastRiskEvent.Valid {
		profile.LastRiskEventAt = &lastRiskEvent.Time
	}

	return nil
}

func (b *Builder) calculateAggregates(profile *UserProfile) {
	if profile.TotalAttempts == 0 {
		return
	}

	profile.SuccessRate = float64(profile.SuccessCount) / float64(profile.TotalAttempts) * 100

	if profile.AvgResponseTime == 0 && profile.MaxResponseTime > 0 {
		profile.AvgResponseTime = (profile.MinResponseTime + profile.MaxResponseTime) / 2
	}

	if profile.PreferredCaptchaType == "" && len(profile.CaptchaTypeDistribution) > 0 {
		var maxType string
		var maxCount int64
		for captchaType, count := range profile.CaptchaTypeDistribution {
			if count > maxCount {
				maxCount = count
				maxType = captchaType
			}
		}
		profile.PreferredCaptchaType = maxType
	}

	if len(profile.ActiveHours) > 0 {
		hours := make([]int, 0, len(profile.ActiveHours))
		for hour := range profile.ActiveHours {
			hours = append(hours, hour)
		}
		sort.Ints(hours)
	}
}

func (b *Builder) RefreshProfile(ctx context.Context, identifier string, identifierType IdentifierType) (*UserProfile, error) {
	return b.BuildProfile(ctx, identifier, identifierType)
}

func (b *Builder) BatchBuildProfiles(ctx context.Context, identifiers []string, identifierType IdentifierType) ([]*UserProfile, error) {
	profiles := make([]*UserProfile, 0, len(identifiers))

	for _, identifier := range identifiers {
		profile, err := b.BuildProfile(ctx, identifier, identifierType)
		if err != nil {
			continue
		}
		profiles = append(profiles, profile)
	}

	return profiles, nil
}

func (b *Builder) RefreshAllProfiles(ctx context.Context, batchSize int) (int, error) {
	columnName := b.getIdentifierColumn(IdentifierTypeIP)

	query := fmt.Sprintf(`
		SELECT DISTINCT %s
		FROM captcha_logs
		WHERE created_at >= NOW() - INTERVAL '30 days'
	`, columnName)

	rows, err := b.captchaDB.QueryContext(ctx, query)
	if err != nil {
		return 0, fmt.Errorf("failed to query distinct identifiers: %w", err)
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var identifier string
		if err := rows.Scan(&identifier); err != nil {
			continue
		}

		_, err := b.BuildProfile(ctx, identifier, IdentifierTypeIP)
		if err != nil {
			continue
		}

		count++

		if batchSize > 0 && count >= batchSize {
			break
		}
	}

	return count, nil
}

type ProfileUpdate struct {
	Identifier      string
	IdentifierType  IdentifierType
	IP              string
	DeviceFingerprint string
	CookieID        string
	SessionID       string
	CaptchaType     string
	ResponseTime    int64
	Success         bool
	RiskScore       int
	Location        string
	Device          string
	CreatedAt       time.Time
}

func (b *Builder) UpdateProfileFromEvent(ctx context.Context, update *ProfileUpdate) error {
	profile, err := b.repo.GetByIdentifier(ctx, update.Identifier, update.IdentifierType)
	if err != nil {
		return fmt.Errorf("failed to get profile: %w", err)
	}

	if profile == nil {
		profile = &UserProfile{
			Identifier:              update.Identifier,
			IdentifierType:          update.IdentifierType,
			CaptchaTypeDistribution: make(map[string]int64),
			ActiveHours:             make(map[int]int64),
			ActiveDays:              make(map[int]int64),
			LocationDistribution:    make(map[string]int64),
			DeviceDistribution:      make(map[string]int64),
			FirstSeenAt:             update.CreatedAt,
			LastSeenAt:              update.CreatedAt,
		}
	}

	if update.IP != "" {
		profile.IP = update.IP
	}
	if update.DeviceFingerprint != "" {
		profile.DeviceFingerprint = update.DeviceFingerprint
	}
	if update.CookieID != "" {
		profile.CookieID = update.CookieID
	}
	if update.SessionID != "" {
		profile.SessionID = update.SessionID
	}

	profile.TotalAttempts++

	if update.Success {
		profile.SuccessCount++
	} else {
		profile.FailCount++
	}

	if profile.TotalAttempts > 0 {
		profile.SuccessRate = float64(profile.SuccessCount) / float64(profile.TotalAttempts) * 100
	}

	if profile.TotalAttempts == 1 {
		profile.MinResponseTime = float64(update.ResponseTime)
		profile.MaxResponseTime = float64(update.ResponseTime)
	} else {
		profile.AvgResponseTime = (profile.AvgResponseTime*float64(profile.TotalAttempts-1) + float64(update.ResponseTime)) / float64(profile.TotalAttempts)
		if float64(update.ResponseTime) < profile.MinResponseTime {
			profile.MinResponseTime = float64(update.ResponseTime)
		}
		if float64(update.ResponseTime) > profile.MaxResponseTime {
			profile.MaxResponseTime = float64(update.ResponseTime)
		}
	}

	if update.CaptchaType != "" {
		profile.CaptchaTypeDistribution[update.CaptchaType]++
		preferredType := profile.PreferredCaptchaType
		preferredCount := int64(0)
		if count, ok := profile.CaptchaTypeDistribution[preferredType]; ok {
			preferredCount = count
		}
		if profile.CaptchaTypeDistribution[update.CaptchaType] > preferredCount {
			profile.PreferredCaptchaType = update.CaptchaType
		}
	}

	hour := update.CreatedAt.Hour()
	profile.ActiveHours[hour]++
	day := int(update.CreatedAt.Weekday())
	profile.ActiveDays[day]++

	if update.Location != "" {
		profile.LocationDistribution[update.Location]++
	}

	if update.Device != "" {
		profile.DeviceDistribution[update.Device]++
	}

	if update.RiskScore > 0 {
		profile.TotalRiskEvents++
		if update.RiskScore >= 70 {
			profile.HighRiskEvents++
		}
		profile.LastRiskEventAt = &update.CreatedAt
	}

	profile.LastSeenAt = update.CreatedAt
	profile.UpdatedAt = time.Now()

	if profile.ID == 0 {
		id, err := b.repo.Create(ctx, profile)
		if err != nil {
			return fmt.Errorf("failed to create profile: %w", err)
		}
		profile.ID = id
	} else {
		if err := b.repo.Update(ctx, profile); err != nil {
			return fmt.Errorf("failed to update profile: %w", err)
		}
	}

	return nil
}
