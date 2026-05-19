package model

import (
	"encoding/json"
	"strings"
	"time"
)

type FeedbackType string

const (
	FeedbackTypeVerification  FeedbackType = "verification"
	FeedbackTypeError         FeedbackType = "error"
	FeedbackTypeUX            FeedbackType = "ux"
	FeedbackTypeAccessibility FeedbackType = "accessibility"
	FeedbackTypePerformance   FeedbackType = "performance"
	FeedbackTypeSecurity      FeedbackType = "security"
	FeedbackTypeGeneral       FeedbackType = "general"
)

type FeedbackSeverity string

const (
	FeedbackSeverityLow      FeedbackSeverity = "low"
	FeedbackSeverityMedium   FeedbackSeverity = "medium"
	FeedbackSeverityHigh     FeedbackSeverity = "high"
	FeedbackSeverityCritical FeedbackSeverity = "critical"
)

type FeedbackStatus string

const (
	FeedbackStatusPending   FeedbackStatus = "pending"
	FeedbackStatusReviewed  FeedbackStatus = "reviewed"
	FeedbackStatusResolved  FeedbackStatus = "resolved"
	FeedbackStatusDismissed FeedbackStatus = "dismissed"
)

type VerificationFeedback struct {
	ID            uint             `json:"id" gorm:"primaryKey;autoIncrement"`
	SessionID     string           `json:"session_id" gorm:"size:100;index:idx_feedback_session"`
	UserID        uint             `json:"user_id" gorm:"index:idx_feedback_user"`
	ApplicationID uint             `json:"application_id" gorm:"index:idx_feedback_app"`
	FeedbackType  FeedbackType     `json:"feedback_type" gorm:"size:50;index:idx_feedback_type"`
	Category      string           `json:"category" gorm:"size:100"`
	Content       string           `json:"content" gorm:"type:text"`
	Severity      FeedbackSeverity `json:"severity" gorm:"size:20"`
	Status        FeedbackStatus   `json:"status" gorm:"size:20;index:idx_feedback_status"`
	Rating        int              `json:"rating" gorm:"default:0"`
	Success       bool             `json:"success" gorm:"default:false"`
	ResponseTime  int64            `json:"response_time"`
	Environment   string           `json:"environment" gorm:"size:50"`
	BrowserInfo   string           `json:"browser_info" gorm:"size:200"`
	IPAddress     string           `json:"ip_address" gorm:"size:50"`
	ContactEmail  string           `json:"contact_email" gorm:"size:100"`
	ScreenshotURL string           `json:"screenshot_url" gorm:"size:500"`
	Metadata      string           `json:"metadata" gorm:"type:text"`
	AdminNotes    string           `json:"admin_notes" gorm:"type:text"`
	ReviewedBy    uint             `json:"reviewed_by"`
	ReviewedAt    *time.Time       `json:"reviewed_at"`
	ResolvedAt    *time.Time       `json:"resolved_at"`
	CreatedAt     time.Time        `json:"created_at" gorm:"index:idx_feedback_created"`
	UpdatedAt     time.Time        `json:"updated_at"`
}

type FeedbackResponse struct {
	ID           uint      `json:"id"`
	FeedbackID   uint      `json:"feedback_id" gorm:"index"`
	AdminID      uint      `json:"admin_id"`
	AdminName    string    `json:"admin_name" gorm:"size:100"`
	Content      string    `json:"content" gorm:"type:text"`
	ResponseType string    `json:"response_type" gorm:"size:50"`
	CreatedAt    time.Time `json:"created_at"`
}

type FeedbackSearchParams struct {
	SessionID       string
	UserID          uint
	ApplicationID   uint
	FeedbackType    FeedbackType
	Category        string
	Severity        FeedbackSeverity
	Status          FeedbackStatus
	Rating          int
	MinRating       int
	MaxRating       int
	Success         *bool
	MinResponseTime int64
	MaxResponseTime int64
	ContactEmail    string
	StartDate       time.Time
	EndDate         time.Time
	SearchText      string
	Page            int
	PageSize        int
	SortBy          string
	SortOrder       string
}

type FeedbackListResult struct {
	Total      int64                  `json:"total"`
	Page       int                    `json:"page"`
	PageSize   int                    `json:"page_size"`
	TotalPages int                    `json:"total_pages"`
	Items      []VerificationFeedback `json:"items"`
}

type FeedbackStatistics struct {
	TotalFeedback        int64               `json:"total_feedback"`
	PendingFeedback      int64               `json:"pending_feedback"`
	ReviewedFeedback     int64               `json:"reviewed_feedback"`
	ResolvedFeedback     int64               `json:"resolved_feedback"`
	AvgRating            float64             `json:"avg_rating"`
	AvgResponseTime      float64             `json:"avg_response_time"`
	TypeDistribution     map[string]int64    `json:"type_distribution"`
	SeverityDistribution map[string]int64    `json:"severity_distribution"`
	StatusDistribution   map[string]int64    `json:"status_distribution"`
	SuccessRate          float64             `json:"success_rate"`
	TrendByDay           []DailyFeedbackStat `json:"trend_by_day"`
}

type DailyFeedbackStat struct {
	Date            time.Time `json:"date"`
	TotalFeedback   int64     `json:"total_feedback"`
	AvgRating       float64   `json:"avg_rating"`
	AvgResponseTime float64   `json:"avg_response_time"`
	SuccessCount    int64     `json:"success_count"`
	FailureCount    int64     `json:"failure_count"`
}

type VerificationFeedbackInput struct {
	SessionID     string           `json:"session_id" binding:"required"`
	FeedbackType  FeedbackType     `json:"feedback_type" binding:"required"`
	Category      string           `json:"category"`
	Content       string           `json:"content" binding:"required"`
	Severity      FeedbackSeverity `json:"severity"`
	Rating        int              `json:"rating" binding:"min=1,max=5"`
	Success       bool             `json:"success"`
	Environment   string           `json:"environment"`
	BrowserInfo   string           `json:"browser_info"`
	ContactEmail  string           `json:"contact_email" binding:"omitempty,email"`
	ScreenshotURL string           `json:"screenshot_url"`
}

type FeedbackResponseInput struct {
	Content      string `json:"content" binding:"required"`
	ResponseType string `json:"response_type" binding:"required"`
}

type UserFeedbackPreferences struct {
	UserID               uint      `json:"user_id" gorm:"primaryKey"`
	EnableEmail          bool      `json:"enable_email" gorm:"default:true"`
	EnablePush           bool      `json:"enable_push" gorm:"default:true"`
	NotificationLevel    string    `json:"notification_level" gorm:"size:20;default:'basic'"`
	DigestFrequency      string    `json:"digest_frequency" gorm:"size:20;default:'weekly'"`
	FeedbackHistoryDays  int       `json:"feedback_history_days" gorm:"default:90"`
	AutoResolveThreshold int       `json:"auto_resolve_threshold" gorm:"default:5"`
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`
}

type HelpDocument struct {
	ID              uint       `json:"id" gorm:"primaryKey;autoIncrement"`
	Title           string     `json:"title" gorm:"size:200"`
	Slug            string     `json:"slug" gorm:"size:200;uniqueIndex"`
	Content         string     `json:"content" gorm:"type:text"`
	Category        string     `json:"category" gorm:"size:100;index:idx_help_category"`
	Tags            string     `json:"tags" gorm:"size:500"`
	Language        string     `json:"language" gorm:"size:10;default:'zh-CN'"`
	Version         string     `json:"version" gorm:"size:20"`
	Priority        int        `json:"priority" gorm:"default:0"`
	IsPublished     bool       `json:"is_published" gorm:"default:false"`
	IsFeatured      bool       `json:"is_featured" gorm:"default:false"`
	ViewCount       int        `json:"view_count" gorm:"default:0"`
	HelpfulCount    int        `json:"helpful_count" gorm:"default:0"`
	NotHelpfulCount int        `json:"not_helpful_count" gorm:"default:0"`
	RelatedDocs     string     `json:"related_docs" gorm:"type:text"`
	Metadata        string     `json:"metadata" gorm:"type:text"`
	AuthorID        uint       `json:"author_id"`
	PublishedAt     *time.Time `json:"published_at"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

type HelpDocumentFeedback struct {
	ID         uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	DocumentID uint      `json:"document_id" gorm:"index:idx_doc_feedback"`
	UserID     uint      `json:"user_id" gorm:"index"`
	Helpful    bool      `json:"helpful"`
	Comment    string    `json:"comment" gorm:"type:text"`
	IPAddress  string    `json:"ip_address" gorm:"size:50"`
	CreatedAt  time.Time `json:"created_at"`
}

type ErrorContext struct {
	ErrorCode    string                 `json:"error_code"`
	ErrorMessage string                 `json:"error_message"`
	Details      string                 `json:"details"`
	Suggestions  []string               `json:"suggestions"`
	HelpLinks    []string               `json:"help_links"`
	ContextData  map[string]interface{} `json:"context_data"`
}

type UXMetrics struct {
	SessionID         string  `json:"session_id"`
	StartTime         int64   `json:"start_time"`
	EndTime           int64   `json:"end_time"`
	TotalDuration     int64   `json:"total_duration"`
	ClickCount        int     `json:"click_count"`
	ErrorCount        int     `json:"error_count"`
	RetryCount        int     `json:"retry_count"`
	SuccessRate       float64 `json:"success_rate"`
	SatisfactionScore float64 `json:"satisfaction_score"`
	NetPromoterScore  int     `json:"net_promoter_score"`
}

func (f *VerificationFeedback) SetMetadata(data map[string]interface{}) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}
	f.Metadata = string(jsonData)
	return nil
}

func (f *VerificationFeedback) GetMetadata() (map[string]interface{}, error) {
	if f.Metadata == "" {
		return make(map[string]interface{}), nil
	}
	var data map[string]interface{}
	err := json.Unmarshal([]byte(f.Metadata), &data)
	return data, err
}

func (f *VerificationFeedback) SetTags(tags []string) {
	f.Metadata = strings.Join(tags, ",")
}

func (f *VerificationFeedback) GetTags() []string {
	if f.Metadata == "" {
		return []string{}
	}
	return strings.Split(f.Metadata, ",")
}

func (f *VerificationFeedback) IsResolved() bool {
	return f.Status == FeedbackStatusResolved
}

func (f *VerificationFeedback) IsPending() bool {
	return f.Status == FeedbackStatusPending
}

func (f *VerificationFeedback) MarkAsReviewed(reviewerID uint) error {
	now := time.Now()
	f.ReviewedBy = reviewerID
	f.ReviewedAt = &now
	f.Status = FeedbackStatusReviewed
	return nil
}

func (f *VerificationFeedback) MarkAsResolved() error {
	now := time.Now()
	f.ResolvedAt = &now
	f.Status = FeedbackStatusResolved
	return nil
}

func (d *HelpDocument) SetRelatedDocs(ids []uint) error {
	jsonData, err := json.Marshal(ids)
	if err != nil {
		return err
	}
	d.RelatedDocs = string(jsonData)
	return nil
}

func (d *HelpDocument) GetRelatedDocs() ([]uint, error) {
	if d.RelatedDocs == "" {
		return []uint{}, nil
	}
	var ids []uint
	err := json.Unmarshal([]byte(d.RelatedDocs), &ids)
	return ids, err
}

func (d *HelpDocument) IncrementViewCount() {
	d.ViewCount++
}

func (d *HelpDocument) SetTagsList(tags []string) error {
	jsonData, err := json.Marshal(tags)
	if err != nil {
		return err
	}
	d.Tags = string(jsonData)
	return nil
}

func (d *HelpDocument) GetTagsList() ([]string, error) {
	if d.Tags == "" {
		return []string{}, nil
	}
	var tags []string
	err := json.Unmarshal([]byte(d.Tags), &tags)
	return tags, err
}

func (p *UserFeedbackPreferences) ShouldSendEmail() bool {
	return p.EnableEmail
}

func (p *UserFeedbackPreferences) ShouldSendPush() bool {
	return p.EnablePush
}

func NewErrorContext(code, message, details string) *ErrorContext {
	return &ErrorContext{
		ErrorCode:    code,
		ErrorMessage: message,
		Details:      details,
		Suggestions:  make([]string, 0),
		HelpLinks:    make([]string, 0),
		ContextData:  make(map[string]interface{}),
	}
}

func (e *ErrorContext) AddSuggestion(suggestion string) {
	e.Suggestions = append(e.Suggestions, suggestion)
}

func (e *ErrorContext) AddHelpLink(link string) {
	e.HelpLinks = append(e.HelpLinks, link)
}

func (e *ErrorContext) AddContext(key string, value interface{}) {
	e.ContextData[key] = value
}

func (m *UXMetrics) CalculateDuration() int64 {
	if m.EndTime > m.StartTime {
		return m.EndTime - m.StartTime
	}
	return time.Now().UnixMilli() - m.StartTime
}

func (m *UXMetrics) IsGoodExperience() bool {
	return m.SuccessRate >= 0.8 && m.RetryCount <= 2
}
