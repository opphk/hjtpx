package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/hjtpx/hjtpx/internal/model"
	"github.com/hjtpx/hjtpx/pkg/database"
	"gorm.io/gorm"
)

var (
	ErrFeedbackNotFound    = errors.New("feedback not found")
	ErrInvalidFeedbackType = errors.New("invalid feedback type")
	ErrInvalidSeverity     = errors.New("invalid severity")
	ErrInvalidRating       = errors.New("rating must be between 1 and 5")
	ErrDocumentNotFound    = errors.New("document not found")
)

type FeedbackService struct {
	db *gorm.DB
}

func NewFeedbackService() *FeedbackService {
	return &FeedbackService{
		db: database.DB,
	}
}

type VerificationResultFeedback struct {
	Success      bool                `json:"success"`
	Message      string              `json:"message"`
	ErrorCode    string              `json:"error_code,omitempty"`
	HelpDocID    uint                `json:"help_doc_id,omitempty"`
	Suggestions  []string            `json:"suggestions,omitempty"`
	NextStep     string              `json:"next_step,omitempty"`
	RetryAllowed bool                `json:"retry_allowed"`
	RetryCount   int                 `json:"retry_count"`
	MaxRetries   int                 `json:"max_retries"`
	ResponseTime int64               `json:"response_time"`
	ErrorContext *model.ErrorContext `json:"error_context,omitempty"`
}

type UserExperienceFeedback struct {
	SessionID              string           `json:"session_id"`
	Metrics                *model.UXMetrics `json:"metrics"`
	SatisfactionScore      float64          `json:"satisfaction_score"`
	ImprovementSuggestions []string         `json:"improvement_suggestions"`
	SuccessRate            float64          `json:"success_rate"`
	AverageDuration        int64            `json:"average_duration"`
}

func (s *FeedbackService) SubmitVerificationFeedback(ctx context.Context, userID uint, appID uint, input *model.VerificationFeedbackInput) (*model.VerificationFeedback, error) {
	if err := s.validateFeedbackInput(input); err != nil {
		return nil, err
	}

	feedback := &model.VerificationFeedback{
		SessionID:     input.SessionID,
		UserID:        userID,
		ApplicationID: appID,
		FeedbackType:  input.FeedbackType,
		Category:      input.Category,
		Content:       input.Content,
		Severity:      input.Severity,
		Rating:        input.Rating,
		Success:       input.Success,
		Environment:   input.Environment,
		BrowserInfo:   input.BrowserInfo,
		ContactEmail:  input.ContactEmail,
		ScreenshotURL: input.ScreenshotURL,
		Status:        model.FeedbackStatusPending,
		CreatedAt:     time.Now(),
	}

	if err := s.db.WithContext(ctx).Create(feedback).Error; err != nil {
		return nil, fmt.Errorf("failed to create feedback: %w", err)
	}

	return feedback, nil
}

func (s *FeedbackService) GetVerificationResult(ctx context.Context, success bool, errorCode string, sessionID string) *VerificationResultFeedback {
	feedback := &VerificationResultFeedback{
		Success:      success,
		RetryAllowed: true,
		RetryCount:   0,
		MaxRetries:   3,
		ResponseTime: time.Now().UnixMilli(),
	}

	if success {
		feedback.Message = s.getSuccessMessage(sessionID)
		feedback.NextStep = "continue"
		return feedback
	}

	feedback.Message = s.getErrorMessage(errorCode)
	feedback.ErrorCode = errorCode
	feedback.ErrorContext = s.getErrorContext(errorCode)
	feedback.Suggestions = s.getSuggestionsForError(errorCode)
	feedback.HelpDocID = s.findHelpDocument(errorCode)
	feedback.NextStep = s.getNextStep(errorCode)

	return feedback
}

func (s *FeedbackService) getSuccessMessage(sessionID string) string {
	messages := []string{
		"验证成功，感谢您的使用！",
		"验证通过，祝您有愉快的体验！",
		"验证完成，您的身份已确认。",
		"成功！您已完成验证流程。",
	}
	return messages[time.Now().UnixNano()%int64(len(messages))]
}

func (s *FeedbackService) getErrorMessage(errorCode string) string {
	errorMessages := map[string]string{
		"session_expired":     "会话已过期，请刷新页面后重试",
		"invalid_position":    "验证位置不正确，请重新尝试",
		"timeout":             "验证超时，请检查网络连接后重试",
		"risk_detected":       "检测到异常行为，请稍后重试",
		"server_error":        "服务器繁忙，请稍后重试",
		"browser_not_support": "您的浏览器不支持此验证方式",
		"network_error":       "网络连接不稳定，请检查网络",
		"attempts_exceeded":   "验证次数已达上限，请稍后再试",
	}

	if msg, ok := errorMessages[errorCode]; ok {
		return msg
	}
	return "验证失败，请重试"
}

func (s *FeedbackService) getErrorContext(errorCode string) *model.ErrorContext {
	context := model.NewErrorContext(errorCode, s.getErrorMessage(errorCode), "")

	switch errorCode {
	case "session_expired":
		context.AddSuggestion("请刷新页面或清除浏览器缓存")
		context.AddHelpLink("/help/verification/session-expired")
		context.AddContext("expires_in", "5 minutes")
	case "invalid_position":
		context.AddSuggestion("请仔细查看验证图片并点击正确位置")
		context.AddHelpLink("/help/verification/correct-position")
		context.AddContext("tolerance", "5 pixels")
	case "timeout":
		context.AddSuggestion("请确保网络连接稳定")
		context.AddHelpLink("/help/troubleshooting/network")
		context.AddContext("timeout_value", "30 seconds")
	case "risk_detected":
		context.AddSuggestion("请更换网络环境或设备后重试")
		context.AddHelpLink("/help/troubleshooting/risk-detected")
		context.AddContext("risk_level", "high")
	case "server_error":
		context.AddSuggestion("服务器可能正在维护，请稍后再试")
		context.AddHelpLink("/help/troubleshooting/server")
		context.AddContext("error_type", "500")
	}

	return context
}

func (s *FeedbackService) getSuggestionsForError(errorCode string) []string {
	suggestions := map[string][]string{
		"session_expired": {
			"刷新页面重新开始验证",
			"清除浏览器缓存和Cookie",
			"检查系统时间是否准确",
		},
		"invalid_position": {
			"仔细查看示例图片",
			"缓慢且准确地点击目标位置",
			"避免使用自动填充工具",
		},
		"timeout": {
			"检查网络连接",
			"更换到更稳定的网络",
			"关闭可能占用带宽的应用",
		},
		"risk_detected": {
			"使用常用的设备和网络",
			"避免使用VPN或代理",
			"等待一段时间后再尝试",
		},
		"server_error": {
			"等待几分钟后重试",
			"查看系统状态页面",
			"联系技术支持",
		},
	}

	if sugs, ok := suggestions[errorCode]; ok {
		return sugs
	}
	return []string{"请稍后重试", "如有问题请联系客服"}
}

func (s *FeedbackService) findHelpDocument(errorCode string) uint {
	var doc model.HelpDocument
	err := s.db.Where("slug LIKE ? AND is_published = ?", "%"+errorCode+"%", true).
		Order("priority DESC").
		First(&doc).Error

	if err != nil {
		return 0
	}
	return doc.ID
}

func (s *FeedbackService) getNextStep(errorCode string) string {
	nextSteps := map[string]string{
		"session_expired":   "refresh",
		"invalid_position":  "retry",
		"timeout":           "retry",
		"risk_detected":     "wait",
		"server_error":      "wait",
		"attempts_exceeded": "contact_support",
	}

	if step, ok := nextSteps[errorCode]; ok {
		return step
	}
	return "retry"
}

func (s *FeedbackService) OptimizeErrorMessage(ctx context.Context, errorCode string, userContext map[string]interface{}) (string, []string, error) {
	baseMessage := s.getErrorMessage(errorCode)

	suggestions := s.getSuggestionsForError(errorCode)

	if userContext != nil {
		if lang, ok := userContext["language"].(string); ok && lang != "" {
			baseMessage = s.localizeMessage(errorCode, lang)
			suggestions = s.localizeSuggestions(errorCode, lang)
		}
	}

	if len(suggestions) > 3 {
		suggestions = suggestions[:3]
	}

	return baseMessage, suggestions, nil
}

func (s *FeedbackService) localizeMessage(errorCode, language string) string {
	if language == "en-US" {
		localized := map[string]string{
			"session_expired":  "Session expired, please refresh the page",
			"invalid_position": "Incorrect verification position",
			"timeout":          "Verification timeout",
			"risk_detected":    "Abnormal behavior detected",
			"server_error":     "Server is busy, please try again later",
		}
		if msg, ok := localized[errorCode]; ok {
			return msg
		}
	}
	return s.getErrorMessage(errorCode)
}

func (s *FeedbackService) localizeSuggestions(errorCode, language string) []string {
	suggestions := s.getSuggestionsForError(errorCode)

	if language == "en-US" {
		enSuggestions := map[string][]string{
			"session_expired": {
				"Refresh the page",
				"Clear browser cache",
				"Check system time",
			},
			"invalid_position": {
				"Look at the example carefully",
				"Click accurately",
				"Avoid autofill",
			},
		}
		if sugs, ok := enSuggestions[errorCode]; ok {
			return sugs
		}
	}
	return suggestions
}

func (s *FeedbackService) GetHelpDocument(ctx context.Context, slug string) (*model.HelpDocument, error) {
	var doc model.HelpDocument
	err := s.db.Where("slug = ? AND is_published = ?", slug, true).First(&doc).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrDocumentNotFound
		}
		return nil, err
	}

	s.db.Model(&doc).UpdateColumn("view_count", doc.ViewCount+1)

	return &doc, nil
}

func (s *FeedbackService) GetHelpDocuments(ctx context.Context, category string, language string) ([]model.HelpDocument, error) {
	var docs []model.HelpDocument
	query := s.db.Where("is_published = ?", true)

	if category != "" {
		query = query.Where("category = ?", category)
	}
	if language != "" {
		query = query.Where("language = ?", language)
	}

	err := query.Order("priority DESC, view_count DESC").Find(&docs).Error
	if err != nil {
		return nil, err
	}

	return docs, nil
}

func (s *FeedbackService) SearchHelpDocuments(ctx context.Context, keyword string, category string) ([]model.HelpDocument, error) {
	var docs []model.HelpDocument
	query := s.db.Where("is_published = ?", true)

	if keyword != "" {
		searchTerm := "%" + keyword + "%"
		query = query.Where("title LIKE ? OR content LIKE ? OR tags LIKE ?", searchTerm, searchTerm, searchTerm)
	}
	if category != "" {
		query = query.Where("category = ?", category)
	}

	err := query.Order("priority DESC, helpful_count DESC").Limit(20).Find(&docs).Error
	if err != nil {
		return nil, err
	}

	return docs, nil
}

func (s *FeedbackService) CreateHelpDocument(ctx context.Context, doc *model.HelpDocument) error {
	doc.Slug = s.generateSlug(doc.Title)
	doc.CreatedAt = time.Now()
	doc.UpdatedAt = time.Now()

	if err := s.db.WithContext(ctx).Create(doc).Error; err != nil {
		return fmt.Errorf("failed to create help document: %w", err)
	}
	return nil
}

func (s *FeedbackService) generateSlug(title string) string {
	slug := strings.ToLower(title)
	replacer := strings.NewReplacer(" ", "-", "/", "-", ":", "")
	slug = replacer.Replace(slug)
	slug = strings.Trim(slug, "-")

	var existingCount int64
	s.db.Model(&model.HelpDocument{}).Where("slug LIKE ?", slug+"%").Count(&existingCount)
	if existingCount > 0 {
		slug = fmt.Sprintf("%s-%d", slug, existingCount+1)
	}

	return slug
}

func (s *FeedbackService) RecordHelpDocumentFeedback(ctx context.Context, docID uint, userID uint, helpful bool, comment string, ipAddress string) error {
	feedback := &model.HelpDocumentFeedback{
		DocumentID: docID,
		UserID:     userID,
		Helpful:    helpful,
		Comment:    comment,
		IPAddress:  ipAddress,
		CreatedAt:  time.Now(),
	}

	tx := s.db.WithContext(ctx).Begin()

	if err := tx.Create(feedback).Error; err != nil {
		tx.Rollback()
		return err
	}

	updateField := "helpful_count"
	if !helpful {
		updateField = "not_helpful_count"
	}

	if err := tx.Model(&model.HelpDocument{}).Where("id = ?", docID).
		Update(updateField, gorm.Expr(updateField+" + 1")).Error; err != nil {
		tx.Rollback()
		return err
	}

	tx.Commit()
	return nil
}

func (s *FeedbackService) GetUserExperienceMetrics(ctx context.Context, sessionID string) (*model.UXMetrics, error) {
	var feedbackCount int64
	s.db.Model(&model.VerificationFeedback{}).
		Where("session_id = ?", sessionID).
		Count(&feedbackCount)

	var avgRating float64
	s.db.Model(&model.VerificationFeedback{}).
		Where("session_id = ? AND rating > 0", sessionID).
		Select("COALESCE(AVG(rating), 0)").
		Scan(&avgRating)

	metrics := &model.UXMetrics{
		SessionID:         sessionID,
		StartTime:         time.Now().Add(-time.Hour).UnixMilli(),
		EndTime:           time.Now().UnixMilli(),
		TotalDuration:     time.Now().Add(-time.Hour).UnixMilli(),
		ClickCount:        0,
		ErrorCount:        int(feedbackCount),
		RetryCount:        0,
		SuccessRate:       0,
		SatisfactionScore: avgRating,
		NetPromoterScore:  0,
	}

	return metrics, nil
}

func (s *FeedbackService) CalculateUserSatisfaction(ctx context.Context, userID uint) (*UserExperienceFeedback, error) {
	var feedbacks []model.VerificationFeedback
	err := s.db.Where("user_id = ?", userID).
		Order("created_at DESC").
		Limit(100).
		Find(&feedbacks).Error
	if err != nil {
		return nil, err
	}

	totalFeedback := len(feedbacks)
	if totalFeedback == 0 {
		return &UserExperienceFeedback{}, nil
	}

	var totalRating float64
	successCount := 0
	var totalDuration int64

	for _, fb := range feedbacks {
		totalRating += float64(fb.Rating)
		if fb.Success {
			successCount++
		}
		totalDuration += fb.ResponseTime
	}

	avgRating := totalRating / float64(totalFeedback)
	successRate := float64(successCount) / float64(totalFeedback)
	avgDuration := totalDuration / int64(totalFeedback)

	feedback := &UserExperienceFeedback{
		SatisfactionScore: avgRating,
		SuccessRate:       successRate,
		AverageDuration:   avgDuration,
	}

	feedback.ImprovementSuggestions = s.generateImprovementSuggestions(successRate, avgRating)

	return feedback, nil
}

func (s *FeedbackService) generateImprovementSuggestions(successRate float64, avgRating float64) []string {
	suggestions := make([]string, 0)

	if successRate < 0.7 {
		suggestions = append(suggestions, "验证成功率较低，建议检查网络环境和设备")
	}
	if avgRating < 3.0 {
		suggestions = append(suggestions, "用户体验有待提升，建议简化验证流程")
	}
	if successRate >= 0.9 && avgRating >= 4.0 {
		suggestions = append(suggestions, "您的验证体验非常好，感谢您的支持！")
	}

	return suggestions
}

func (s *FeedbackService) GetFeedbackStatistics(ctx context.Context, startDate, endDate time.Time) (*model.FeedbackStatistics, error) {
	var stats model.FeedbackStatistics

	s.db.Model(&model.VerificationFeedback{}).
		Where("created_at BETWEEN ? AND ?", startDate, endDate).
		Count(&stats.TotalFeedback)

	s.db.Model(&model.VerificationFeedback{}).
		Where("created_at BETWEEN ? AND ? AND status = ?", startDate, endDate, model.FeedbackStatusPending).
		Count(&stats.PendingFeedback)

	s.db.Model(&model.VerificationFeedback{}).
		Where("created_at BETWEEN ? AND ? AND status = ?", startDate, endDate, model.FeedbackStatusReviewed).
		Count(&stats.ReviewedFeedback)

	s.db.Model(&model.VerificationFeedback{}).
		Where("created_at BETWEEN ? AND ? AND status = ?", startDate, endDate, model.FeedbackStatusResolved).
		Count(&stats.ResolvedFeedback)

	s.db.Model(&model.VerificationFeedback{}).
		Where("created_at BETWEEN ? AND ? AND rating > 0", startDate, endDate).
		Select("COALESCE(AVG(rating), 0)").
		Scan(&stats.AvgRating)

	s.db.Model(&model.VerificationFeedback{}).
		Where("created_at BETWEEN ? AND ?", startDate, endDate).
		Select("COALESCE(AVG(response_time), 0)").
		Scan(&stats.AvgResponseTime)

	stats.TypeDistribution = make(map[string]int64)
	var typeStats []struct {
		Type  string
		Count int64
	}
	s.db.Model(&model.VerificationFeedback{}).
		Where("created_at BETWEEN ? AND ?", startDate, endDate).
		Select("feedback_type, COUNT(*) as count").
		Group("feedback_type").
		Scan(&typeStats)
	for _, ts := range typeStats {
		stats.TypeDistribution[ts.Type] = ts.Count
	}

	stats.SeverityDistribution = make(map[string]int64)
	var severityStats []struct {
		Severity string
		Count    int64
	}
	s.db.Model(&model.VerificationFeedback{}).
		Where("created_at BETWEEN ? AND ?", startDate, endDate).
		Select("severity, COUNT(*) as count").
		Group("severity").
		Scan(&severityStats)
	for _, ss := range severityStats {
		stats.SeverityDistribution[ss.Severity] = ss.Count
	}

	stats.StatusDistribution = make(map[string]int64)
	stats.StatusDistribution["pending"] = stats.PendingFeedback
	stats.StatusDistribution["reviewed"] = stats.ReviewedFeedback
	stats.StatusDistribution["resolved"] = stats.ResolvedFeedback

	var successCount int64
	s.db.Model(&model.VerificationFeedback{}).
		Where("created_at BETWEEN ? AND ? AND success = ?", startDate, endDate, true).
		Count(&successCount)
	if stats.TotalFeedback > 0 {
		stats.SuccessRate = float64(successCount) / float64(stats.TotalFeedback) * 100
	}

	return &stats, nil
}

func (s *FeedbackService) GetFeedbackList(ctx context.Context, params *model.FeedbackSearchParams) (*model.FeedbackListResult, error) {
	if params.Page < 1 {
		params.Page = 1
	}
	if params.PageSize < 1 || params.PageSize > 100 {
		params.PageSize = 20
	}

	var feedbacks []model.VerificationFeedback
	var total int64

	query := s.db.Model(&model.VerificationFeedback{})

	if params.SessionID != "" {
		query = query.Where("session_id = ?", params.SessionID)
	}
	if params.UserID > 0 {
		query = query.Where("user_id = ?", params.UserID)
	}
	if params.ApplicationID > 0 {
		query = query.Where("application_id = ?", params.ApplicationID)
	}
	if params.FeedbackType != "" {
		query = query.Where("feedback_type = ?", params.FeedbackType)
	}
	if params.Category != "" {
		query = query.Where("category = ?", params.Category)
	}
	if params.Severity != "" {
		query = query.Where("severity = ?", params.Severity)
	}
	if params.Status != "" {
		query = query.Where("status = ?", params.Status)
	}
	if params.MinRating > 0 {
		query = query.Where("rating >= ?", params.MinRating)
	}
	if params.MaxRating > 0 {
		query = query.Where("rating <= ?", params.MaxRating)
	}
	if params.Success != nil {
		query = query.Where("success = ?", *params.Success)
	}
	if params.ContactEmail != "" {
		query = query.Where("contact_email LIKE ?", "%"+params.ContactEmail+"%")
	}
	if !params.StartDate.IsZero() {
		query = query.Where("created_at >= ?", params.StartDate)
	}
	if !params.EndDate.IsZero() {
		query = query.Where("created_at <= ?", params.EndDate)
	}
	if params.SearchText != "" {
		search := "%" + params.SearchText + "%"
		query = query.Where("content LIKE ? OR category LIKE ?", search, search)
	}

	query.Count(&total)

	sortBy := "created_at"
	if params.SortBy != "" {
		sortBy = params.SortBy
	}
	sortOrder := "DESC"
	if params.SortOrder != "" {
		sortOrder = params.SortOrder
	}

	offset := (params.Page - 1) * params.PageSize
	err := query.Order(fmt.Sprintf("%s %s", sortBy, sortOrder)).
		Offset(offset).
		Limit(params.PageSize).
		Find(&feedbacks).Error
	if err != nil {
		return nil, err
	}

	totalPages := int(total) / params.PageSize
	if int(total)%params.PageSize > 0 {
		totalPages++
	}

	return &model.FeedbackListResult{
		Total:      total,
		Page:       params.Page,
		PageSize:   params.PageSize,
		TotalPages: totalPages,
		Items:      feedbacks,
	}, nil
}

func (s *FeedbackService) UpdateFeedbackStatus(ctx context.Context, feedbackID uint, status model.FeedbackStatus, adminID uint, notes string) error {
	updates := map[string]interface{}{
		"status": status,
	}

	if notes != "" {
		updates["admin_notes"] = notes
	}

	if status == model.FeedbackStatusReviewed {
		now := time.Now()
		updates["reviewed_by"] = adminID
		updates["reviewed_at"] = now
	} else if status == model.FeedbackStatusResolved {
		now := time.Now()
		updates["resolved_at"] = now
	}

	err := s.db.WithContext(ctx).
		Model(&model.VerificationFeedback{}).
		Where("id = ?", feedbackID).
		Updates(updates).Error

	if err != nil {
		return fmt.Errorf("failed to update feedback status: %w", err)
	}
	return nil
}

func (s *FeedbackService) validateFeedbackInput(input *model.VerificationFeedbackInput) error {
	validTypes := map[model.FeedbackType]bool{
		model.FeedbackTypeVerification:  true,
		model.FeedbackTypeError:         true,
		model.FeedbackTypeUX:            true,
		model.FeedbackTypeAccessibility: true,
		model.FeedbackTypePerformance:   true,
		model.FeedbackTypeSecurity:      true,
		model.FeedbackTypeGeneral:       true,
	}

	if !validTypes[input.FeedbackType] {
		return ErrInvalidFeedbackType
	}

	validSeverities := map[model.FeedbackSeverity]bool{
		model.FeedbackSeverityLow:      true,
		model.FeedbackSeverityMedium:   true,
		model.FeedbackSeverityHigh:     true,
		model.FeedbackSeverityCritical: true,
	}

	if input.Severity != "" && !validSeverities[input.Severity] {
		return ErrInvalidSeverity
	}

	if input.Rating < 0 || input.Rating > 5 {
		return ErrInvalidRating
	}

	return nil
}

func (s *FeedbackService) AddFeedbackResponse(ctx context.Context, feedbackID uint, input *model.FeedbackResponseInput) (*model.FeedbackResponse, error) {
	var feedback model.VerificationFeedback
	if err := s.db.First(&feedback, feedbackID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrFeedbackNotFound
		}
		return nil, err
	}

	response := &model.FeedbackResponse{
		FeedbackID:   feedbackID,
		Content:      input.Content,
		ResponseType: input.ResponseType,
		CreatedAt:    time.Now(),
	}

	if err := s.db.WithContext(ctx).Create(response).Error; err != nil {
		return nil, fmt.Errorf("failed to add response: %w", err)
	}

	return response, nil
}

func (s *FeedbackService) GetUserPreferences(ctx context.Context, userID uint) (*model.UserFeedbackPreferences, error) {
	var prefs model.UserFeedbackPreferences
	err := s.db.Where("user_id = ?", userID).First(&prefs).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			prefs = model.UserFeedbackPreferences{
				UserID:              userID,
				EnableEmail:         true,
				EnablePush:          true,
				NotificationLevel:   "basic",
				DigestFrequency:     "weekly",
				FeedbackHistoryDays: 90,
			}
			s.db.Create(&prefs)
			return &prefs, nil
		}
		return nil, err
	}
	return &prefs, nil
}

func (s *FeedbackService) UpdateUserPreferences(ctx context.Context, userID uint, prefs *model.UserFeedbackPreferences) error {
	prefs.UserID = userID
	prefs.UpdatedAt = time.Now()

	err := s.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Assign(prefs).
		FirstOrCreate(&model.UserFeedbackPreferences{}).Error

	if err != nil {
		return fmt.Errorf("failed to update preferences: %w", err)
	}
	return nil
}

func (s *FeedbackService) GetRecommendedHelpDocuments(ctx context.Context, errorCode string, category string) ([]model.HelpDocument, error) {
	var docs []model.HelpDocument
	query := s.db.Where("is_published = ? AND is_featured = ?", true, true)

	if category != "" {
		query = query.Where("category = ?", category)
	}

	if errorCode != "" {
		query = query.Where("slug LIKE ? OR tags LIKE ?", "%"+errorCode+"%", "%"+errorCode+"%")
	}

	err := query.Order("priority DESC, helpful_count DESC").Limit(5).Find(&docs).Error
	if err != nil {
		return nil, err
	}

	return docs, nil
}

func (s *FeedbackService) GenerateUserGuide(ctx context.Context, userID uint, language string) (string, error) {
	var prefs *model.UserFeedbackPreferences
	prefs, err := s.GetUserPreferences(ctx, userID)
	if err != nil {
		prefs = &model.UserFeedbackPreferences{}
	}

	var docs []model.HelpDocument
	if language == "en-US" {
		docs, _ = s.GetHelpDocuments(ctx, "getting-started", "en-US")
	} else {
		docs, _ = s.GetHelpDocuments(ctx, "getting-started", "zh-CN")
	}

	if len(docs) == 0 {
		if language == "en-US" {
			return "Welcome! Please visit our help center for guides.", nil
		}
		return "欢迎使用！请访问我们的帮助中心获取指南。", nil
	}

	var guide strings.Builder
	if language == "en-US" {
		guide.WriteString("# User Guide\n\n")
	} else {
		guide.WriteString("# 用户指南\n\n")
	}

	for _, doc := range docs {
		guide.WriteString(fmt.Sprintf("## %s\n", doc.Title))
		guide.WriteString(fmt.Sprintf("%s\n\n", doc.Content))
	}

	return guide.String(), nil
}
