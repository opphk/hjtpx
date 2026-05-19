package service

import (
	"encoding/json"
	"fmt"
	"time"
)

type GuideStep struct {
	ID          string                 `json:"id"`
	Title       string                 `json:"title"`
	Description string                 `json:"description"`
	Type        string                 `json:"type"`
	Target      string                 `json:"target"`
	Position    string                 `json:"position"`
	Actions     []GuideAction          `json:"actions"`
	Conditions  map[string]interface{} `json:"conditions"`
	Priority    int                    `json:"priority"`
}

type GuideAction struct {
	Type     string                 `json:"type"`
	Selector string                 `json:"selector"`
	Content  string                 `json:"content"`
	Callback string                 `json:"callback"`
	Params   map[string]interface{} `json:"params"`
}

type GuideSession struct {
	ID            string                 `json:"id"`
	UserID        string                 `json:"user_id"`
	GuideID       string                 `json:"guide_id"`
	CurrentStep   int                    `json:"current_step"`
	CompletedSteps []int                `json:"completed_steps"`
	SkippedSteps  []int                  `json:"skipped_steps"`
	StartedAt     time.Time              `json:"started_at"`
	UpdatedAt     time.Time              `json:"updated_at"`
	CompletedAt   *time.Time             `json:"completed_at"`
	Context       map[string]interface{} `json:"context"`
}

type GuideContext struct {
	UserID         string                 `json:"user_id"`
	SessionID      string                 `json:"session_id"`
	Device         string                 `json:"device"`
	Browser        string                 `json:"browser"`
	OS             string                 `json:"os"`
	ScreenSize     string                 `json:"screen_size"`
	Language       string                 `json:"language"`
	Experience     string                 `json:"experience"`
	SuccessRate    float64                `json:"success_rate"`
	TotalAttempts  int                    `json:"total_attempts"`
	FailedAttempts int                    `json:"failed_attempts"`
	TimeSpent      int64                  `json:"time_spent"`
	VerificationType string               `json:"verification_type"`
}

type GuideAnalytics struct {
	SessionID     string    `json:"session_id"`
	TotalViews    int       `json:"total_views"`
	Completions   int       `json:"completions"`
	DropOffs      int       `json:"drop_offs"`
	AvgTimeSpent  float64   `json:"avg_time_spent"`
	LastUpdated   time.Time `json:"last_updated"`
}

type GuideService struct {
	guides    map[string][]*GuideStep
	sessions  map[string]*GuideSession
	analytics map[string]*GuideAnalytics
}

func NewGuideService() *GuideService {
	service := &GuideService{
		guides:    make(map[string][]*GuideStep),
		sessions:  make(map[string]*GuideSession),
		analytics: make(map[string]*GuideAnalytics),
	}

	service.initDefaultGuides()
	return service
}

func (s *GuideService) initDefaultGuides() {
	s.guides["onboarding"] = []*GuideStep{
		{
			ID:          "welcome",
			Title:       "欢迎使用",
			Description: "欢迎来到验证系统，让我们快速了解主要功能",
			Type:        "welcome",
			Target:      "",
			Position:    "center",
			Priority:    1,
		},
		{
			ID:          "slider_intro",
			Title:       "滑块验证",
			Description: "拖动滑块完成拼图，适合简单场景",
			Type:        "highlight",
			Target:      ".slider-captcha",
			Position:    "bottom",
			Priority:    2,
		},
		{
			ID:          "click_intro",
			Title:       "点选验证",
			Description: "按顺序点击目标字符，安全性更高",
			Type:        "highlight",
			Target:      ".click-captcha",
			Position:    "right",
			Priority:    3,
		},
		{
			ID:          "complete",
			Title:       "完成",
			Description: "您已了解基本功能，开始使用吧！",
			Type:        "success",
			Target:      "",
			Position:    "center",
			Priority:    4,
		},
	}

	s.guides["slider_guide"] = []*GuideStep{
		{
			ID:          "slider_target",
			Title:       "拖动滑块",
			Description: "将滑块拖动到缺口位置",
			Type:        "tooltip",
			Target:      ".slider-handle",
			Position:    "top",
			Actions: []GuideAction{
				{
					Type:     "highlight",
					Selector: ".slider-track",
				},
			},
			Priority: 1,
		},
		{
			ID:          "slider_release",
			Title:       "释放完成",
			Description: "松开鼠标完成验证",
			Type:        "tooltip",
			Target:      ".slider-track",
			Position:    "bottom",
			Priority:    2,
		},
	}

	s.guides["click_guide"] = []*GuideStep{
		{
			ID:          "click_instruction",
			Title:       "查看提示",
			Description: "根据顶部提示按顺序点击字符",
			Type:        "tooltip",
			Target:      ".click-hint",
			Position:    "top",
			Priority:    1,
		},
		{
			ID:          "click_target",
			Title:       "点击字符",
			Description: "依次点击所有目标字符",
			Type:        "highlight",
			Target:      ".click-images",
			Position:    "center",
			Actions: []GuideAction{
				{
					Type:     "pulse",
					Selector: ".click-image",
				},
			},
			Priority: 2,
		},
	}

	s.guides["error_recovery"] = []*GuideStep{
		{
			ID:          "error_detected",
			Title:       "验证失败",
			Description: "检测到异常，请重试",
			Type:        "warning",
			Target:      "",
			Position:    "center",
			Priority:    1,
		},
		{
			ID:          "suggest_retry",
			Title:       "建议重试",
			Description: "请按照正确的顺序和方法重新验证",
			Type:        "tip",
			Target:      ".captcha-container",
			Position:    "bottom",
			Actions: []GuideAction{
				{
					Type:     "enable",
					Selector: ".captcha-retry",
				},
			},
			Priority: 2,
		},
	}
}

func (s *GuideService) GetGuide(guideID string) ([]*GuideStep, error) {
	steps, exists := s.guides[guideID]
	if !exists {
		return nil, fmt.Errorf("guide not found: %s", guideID)
	}

	result := make([]*GuideStep, len(steps))
	copy(result, steps)
	return result, nil
}

func (s *GuideService) GetAllGuides() []string {
	guideIDs := make([]string, 0, len(s.guides))
	for id := range s.guides {
		guideIDs = append(guideIDs, id)
	}
	return guideIDs
}

func (s *GuideService) CreateGuide(guideID string, steps []*GuideStep) error {
	if guideID == "" {
		return fmt.Errorf("guide ID is required")
	}
	if len(steps) == 0 {
		return fmt.Errorf("guide must have at least one step")
	}

	s.guides[guideID] = steps
	return nil
}

func (s *GuideService) UpdateGuide(guideID string, steps []*GuideStep) error {
	if _, exists := s.guides[guideID]; !exists {
		return fmt.Errorf("guide not found: %s", guideID)
	}
	s.guides[guideID] = steps
	return nil
}

func (s *GuideService) DeleteGuide(guideID string) error {
	if _, exists := s.guides[guideID]; !exists {
		return fmt.Errorf("guide not found: %s", guideID)
	}
	delete(s.guides, guideID)
	return nil
}

func (s *GuideService) StartSession(userID string, guideID string, context *GuideContext) (*GuideSession, error) {
	if _, exists := s.guides[guideID]; !exists {
		return nil, fmt.Errorf("guide not found: %s", guideID)
	}

	sessionID := fmt.Sprintf("session_%s_%d", userID, time.Now().UnixNano())

	session := &GuideSession{
		ID:             sessionID,
		UserID:         userID,
		GuideID:        guideID,
		CurrentStep:    0,
		CompletedSteps: []int{},
		SkippedSteps:   []int{},
		StartedAt:      time.Now(),
		UpdatedAt:      time.Now(),
		Context:        s.contextToMap(context),
	}

	s.sessions[sessionID] = session

	analytics := &GuideAnalytics{
		SessionID:   sessionID,
		TotalViews:  1,
		Completions: 0,
		DropOffs:    0,
		LastUpdated: time.Now(),
	}
	s.analytics[sessionID] = analytics

	return session, nil
}

func (s *GuideService) GetSession(sessionID string) (*GuideSession, error) {
	session, exists := s.sessions[sessionID]
	if !exists {
		return nil, fmt.Errorf("session not found: %s", sessionID)
	}
	return session, nil
}

func (s *GuideService) GetSessionByUser(userID string) (*GuideSession, error) {
	for _, session := range s.sessions {
		if session.UserID == userID && session.CompletedAt == nil {
			return session, nil
		}
	}
	return nil, fmt.Errorf("no active session found for user: %s", userID)
}

func (s *GuideService) CompleteStep(sessionID string, stepIndex int) error {
	session, exists := s.sessions[sessionID]
	if !exists {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	if stepIndex < 0 || stepIndex >= len(s.guides[session.GuideID]) {
		return fmt.Errorf("invalid step index: %d", stepIndex)
	}

	session.CompletedSteps = append(session.CompletedSteps, stepIndex)
	session.CurrentStep = stepIndex + 1
	session.UpdatedAt = time.Now()

	if analytics, exists := s.analytics[sessionID]; exists {
		analytics.TotalViews++
		analytics.LastUpdated = time.Now()
	}

	return nil
}

func (s *GuideService) SkipStep(sessionID string, stepIndex int, reason string) error {
	session, exists := s.sessions[sessionID]
	if !exists {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	session.SkippedSteps = append(session.SkippedSteps, stepIndex)
	session.CurrentStep = stepIndex + 1
	session.UpdatedAt = time.Now()

	if session.Context == nil {
		session.Context = make(map[string]interface{})
	}
	session.Context["skip_reason_"+fmt.Sprintf("%d", stepIndex)] = reason

	return nil
}

func (s *GuideService) CompleteSession(sessionID string) error {
	session, exists := s.sessions[sessionID]
	if !exists {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	now := time.Now()
	session.CompletedAt = &now
	session.UpdatedAt = now

	if analytics, exists := s.analytics[sessionID]; exists {
		analytics.Completions++
		analytics.LastUpdated = now

		totalSteps := len(session.CompletedSteps)
		if totalSteps > 0 {
			analytics.AvgTimeSpent = float64(time.Since(session.StartedAt).Milliseconds()) / float64(totalSteps)
		}
	}

	return nil
}

func (s *GuideService) AbandonSession(sessionID string, reason string) error {
	session, exists := s.sessions[sessionID]
	if !exists {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	session.UpdatedAt = time.Now()
	if session.Context == nil {
		session.Context = make(map[string]interface{})
	}
	session.Context["abandon_reason"] = reason

	if analytics, exists := s.analytics[sessionID]; exists {
		analytics.DropOffs++
		analytics.LastUpdated = time.Now()
	}

	return nil
}

func (s *GuideService) GetAnalytics(sessionID string) (*GuideAnalytics, error) {
	analytics, exists := s.analytics[sessionID]
	if !exists {
		return nil, fmt.Errorf("analytics not found for session: %s", sessionID)
	}
	return analytics, nil
}

func (s *GuideService) GetPersonalizedGuide(context *GuideContext) (string, error) {
	if context.TotalAttempts == 0 {
		return "onboarding", nil
	}

	failureRate := 0.0
	if context.TotalAttempts > 0 {
		failureRate = float64(context.FailedAttempts) / float64(context.TotalAttempts)
	}

	if failureRate > 0.5 {
		if context.VerificationType == "slider" {
			return "slider_guide", nil
		}
		return "click_guide", nil
	}

	if context.FailedAttempts > 0 && context.FailedAttempts < 3 {
		return "error_recovery", nil
	}

	return "", fmt.Errorf("no specific guide needed")
}

func (s *GuideService) SaveProgress(sessionID string) ([]byte, error) {
	session, exists := s.sessions[sessionID]
	if !exists {
		return nil, fmt.Errorf("session not found: %s", sessionID)
	}

	data, err := json.Marshal(session)
	if err != nil {
		return nil, err
	}

	return data, nil
}

func (s *GuideService) RestoreProgress(sessionID string, data []byte) error {
	var session GuideSession
	if err := json.Unmarshal(data, &session); err != nil {
		return err
	}

	s.sessions[sessionID] = &session
	return nil
}

func (s *GuideService) GetNextStep(sessionID string) (*GuideStep, error) {
	session, exists := s.sessions[sessionID]
	if !exists {
		return nil, fmt.Errorf("session not found: %s", sessionID)
	}

	if session.CompletedAt != nil {
		return nil, fmt.Errorf("session already completed")
	}

	guideSteps, exists := s.guides[session.GuideID]
	if !exists {
		return nil, fmt.Errorf("guide not found: %s", session.GuideID)
	}

	if session.CurrentStep >= len(guideSteps) {
		return nil, fmt.Errorf("all steps completed")
	}

	nextStep := guideSteps[session.CurrentStep]

	for _, skipped := range session.SkippedSteps {
		if skipped == session.CurrentStep {
			session.CurrentStep++
			if session.CurrentStep >= len(guideSteps) {
				return nil, fmt.Errorf("all steps completed")
			}
			nextStep = guideSteps[session.CurrentStep]
		}
	}

	return nextStep, nil
}

func (s *GuideService) contextToMap(context *GuideContext) map[string]interface{} {
	if context == nil {
		return make(map[string]interface{})
	}

	return map[string]interface{}{
		"device":           context.Device,
		"browser":           context.Browser,
		"os":               context.OS,
		"screen_size":      context.ScreenSize,
		"language":          context.Language,
		"experience":        context.Experience,
		"success_rate":      context.SuccessRate,
		"total_attempts":    context.TotalAttempts,
		"failed_attempts":   context.FailedAttempts,
		"time_spent":        context.TimeSpent,
		"verification_type": context.VerificationType,
	}
}

func (s *GuideService) ExportAnalytics() (map[string]*GuideAnalytics, error) {
	result := make(map[string]*GuideAnalytics)
	for id, analytics := range s.analytics {
		result[id] = analytics
	}
	return result, nil
}
