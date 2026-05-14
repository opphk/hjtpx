package model

import (
	"database/sql"
	"time"
)

type CaptchaLog struct {
	ID         int64          `json:"id"`
	Type       string         `json:"captcha_type"`
	ClientID   string         `json:"client_id"`
	IP         string         `json:"ip"`
	UserAgent  sql.NullString `json:"-"`
	Result     bool           `json:"result"`
	Duration   int            `json:"duration"`
	RiskScore  int            `json:"risk_score"`
	CreatedAt  time.Time     `json:"created_at"`
}

type CaptchaLogDTO struct {
	ID         int64  `json:"id"`
	Type       string `json:"captcha_type"`
	ClientID   string `json:"client_id"`
	IP         string `json:"ip"`
	UserAgent  string `json:"user_agent,omitempty"`
	Result     bool   `json:"result"`
	Duration   int    `json:"duration"`
	RiskScore  int    `json:"risk_score"`
	CreatedAt  string `json:"created_at"`
}

func (c *CaptchaLog) ToDTO() *CaptchaLogDTO {
	dto := &CaptchaLogDTO{
		ID:        c.ID,
		Type:      c.Type,
		ClientID:  c.ClientID,
		IP:        c.IP,
		Result:    c.Result,
		Duration:  c.Duration,
		RiskScore: c.RiskScore,
		CreatedAt: c.CreatedAt.Format(time.RFC3339),
	}
	if c.UserAgent.Valid {
		dto.UserAgent = c.UserAgent.String
	}
	return dto
}

type CreateCaptchaLogRequest struct {
	Type      string `json:"captcha_type" binding:"required,oneof=slider click puzzle"`
	ClientID  string `json:"client_id" binding:"required,max=64"`
	IP        string `json:"ip" binding:"required,max=45"`
	UserAgent string `json:"user_agent"`
	Result    bool   `json:"result"`
	Duration  int    `json:"duration" binding:"min=0"`
	RiskScore int    `json:"risk_score" binding:"min=0,max=100"`
}

type CaptchaLogFilter struct {
	StartDate  *time.Time
	EndDate    *time.Time
	Type       string
	ClientID   string
	IP         string
	Result     *bool
	MinScore   int
	MaxScore   int
	Page       int
	PageSize   int
}

func (f *CaptchaLogFilter) Offset() int {
	if f.Page <= 0 {
		f.Page = 1
	}
	if f.PageSize <= 0 {
		f.PageSize = 20
	}
	return (f.Page - 1) * f.PageSize
}

func (f *CaptchaLogFilter) Limit() int {
	if f.PageSize <= 0 {
		f.PageSize = 20
	}
	if f.PageSize > 100 {
		f.PageSize = 100
	}
	return f.PageSize
}

type CaptchaLogStats struct {
	TotalCount     int64            `json:"total_count"`
	SuccessCount   int64            `json:"success_count"`
	FailCount      int64            `json:"fail_count"`
	SuccessRate    float64          `json:"success_rate"`
	AvgDuration    float64          `json:"avg_duration"`
	AvgRiskScore   float64          `json:"avg_risk_score"`
	ByType         map[string]int64 `json:"by_type"`
	ByHour         []HourlyStat     `json:"by_hour"`
}

type HourlyStat struct {
	Hour         time.Time `json:"hour"`
	TotalCount   int64     `json:"total_count"`
	SuccessCount int64     `json:"success_count"`
	FailCount    int64     `json:"fail_count"`
}

type IPStats struct {
	IP            string    `json:"ip"`
	TotalAttempts int64     `json:"total_attempts"`
	SuccessCount  int64     `json:"success_count"`
	FailCount     int64     `json:"fail_count"`
	AvgRiskScore  float64   `json:"avg_risk_score"`
	MaxRiskScore  float64   `json:"max_risk_score"`
	FirstSeen     time.Time `json:"first_seen"`
	LastSeen      time.Time `json:"last_seen"`
	UniqueClients int64     `json:"unique_clients"`
}

type ArchiveStats struct {
	TableName       string     `json:"table_name"`
	ActiveCount     int64      `json:"active_count"`
	ArchivedCount   int64      `json:"archived_count"`
	ArchiveRatio    float64    `json:"archive_ratio"`
	OldestActive    *time.Time `json:"oldest_active"`
	NewestArchived  *time.Time `json:"newest_archived"`
	LastArchiveAt   *time.Time `json:"last_archive_at"`
}

type TableSize struct {
	SchemaName     string `json:"schema_name"`
	TableName      string `json:"table_name"`
	TotalSize      string `json:"total_size"`
	TotalBytes     int64  `json:"total_bytes"`
	TableSize      string `json:"table_size"`
	TableBytes     int64  `json:"table_bytes"`
	IndexesSize    string `json:"indexes_size"`
	IndexesBytes   int64  `json:"indexes_bytes"`
	LiveTuples     int64  `json:"n_live_tup"`
	DeadTuples     int64  `json:"n_dead_tup"`
	LastVacuum     *time.Time `json:"last_vacuum"`
	LastAutovacuum *time.Time `json:"last_autovacuum"`
	LastAnalyze    *time.Time `json:"last_analyze"`
}

type IndexUsage struct {
	SchemaName  string `json:"schemaname"`
	TableName   string `json:"tablename"`
	IndexName   string `json:"indexname"`
	IdxScan     int64  `json:"idx_scan"`
	IdxTupRead  int64  `json:"idx_tup_read"`
	IdxTupFetch int64  `json:"idx_tup_fetch"`
	IndexSize   string `json:"index_size"`
}

type CleanupJobLog struct {
	ID               int64      `json:"id"`
	JobName          string     `json:"job_name"`
	JobType          string     `json:"job_type"`
	StartedAt        time.Time  `json:"started_at"`
	CompletedAt      *time.Time `json:"completed_at"`
	Status           string     `json:"status"`
	RecordsProcessed int64      `json:"records_processed"`
	RecordsDeleted   int64      `json:"records_deleted"`
	ExecutionTimeMs  int64      `json:"execution_time_ms"`
	ErrorMessage     string     `json:"error_message"`
	CreatedAt        time.Time  `json:"created_at"`
}

type PoolMetrics struct {
	MaxOpenConns     int   `json:"max_open_conns"`
	OpenConns        int   `json:"open_conns"`
	IdleConns        int   `json:"idle_conns"`
	WaitCount        int64 `json:"wait_count"`
	WaitDuration     int64 `json:"wait_duration_ms"`
	MaxIdleClosed    int64 `json:"max_idle_closed"`
	MaxLifetimeClosed int64 `json:"max_lifetime_closed"`
}
