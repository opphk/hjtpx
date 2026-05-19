package service

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"sync"
	"time"

	"github.com/hjtpx/hjtpx/pkg/models"
)

// CaptchaType 验证码类型
type CaptchaType string

const (
	CaptchaTypeSlider     CaptchaType = "slider"
	CaptchaTypeClick      CaptchaType = "click"
	CaptchaTypeIcon       CaptchaType = "icon"
	CaptchaType3D         CaptchaType = "3d"
	CaptchaTypeSemantic   CaptchaType = "semantic"
	CaptchaTypeEmoji      CaptchaType = "emoji"
	CaptchaTypeVoice      CaptchaType = "voice"
	CaptchaTypeMath       CaptchaType = "math"
	CaptchaTypeColor      CaptchaType = "color"
	CaptchaTypePhrase     CaptchaType = "phrase"
)

// VerificationStep 验证步骤
type VerificationStep struct {
	StepID         string      `json:"step_id"`
	CaptchaType    CaptchaType `json:"captcha_type"`
	Difficulty     string      `json:"difficulty"`
	Required       bool        `json:"required"`
	Order          int         `json:"order"`
	SessionID      string      `json:"session_id"`
	Data          interface{} `json:"data"`
	Completed      bool        `json:"completed"`
	Passed         bool        `json:"passed"`
	Score          float64     `json:"score"`
}

// VerificationFlow 验证流程
type VerificationFlow struct {
	FlowID         string              `json:"flow_id"`
	Steps          []*VerificationStep `json:"steps"`
	CurrentStep    int                 `json:"current_step"`
	Status         string              `json:"status"`
	Strategy       string              `json:"strategy"`
	RequiredPassed int                 `json:"required_passed"`
	TotalSteps     int                 `json:"total_steps"`
	PassedSteps    int                 `json:"passed_steps"`
	FailedSteps    int                 `json:"failed_steps"`
	CreatedAt      time.Time           `json:"created_at"`
	ExpiresAt      time.Time           `json:"expires_at"`
	UserID         string              `json:"user_id"`
	ClientIP       string              `json:"client_ip"`
	UserAgent      string              `json:"user_agent"`
	Fingerprint    string              `json:"fingerprint"`
	RiskScore      float64             `json:"risk_score"`
}

// ComboVerificationService 组合验证服务
type ComboVerificationService struct {
	flows         map[string]*VerificationFlow
	difficultySvc *EnhancedAdaptiveDifficultyService
	mu            sync.RWMutex
}

// NewComboVerificationService 创建组合验证服务
func NewComboVerificationService(difficultySvc *EnhancedAdaptiveDifficultyService) *ComboVerificationService {
	return &ComboVerificationService{
		flows:         make(map[string]*VerificationFlow),
		difficultySvc: difficultySvc,
	}
}

// CreateFlowRequest 创建验证流程请求
type CreateFlowRequest struct {
	UserID      string  `json:"user_id"`
	ClientIP    string  `json:"client_ip"`
	UserAgent   string  `json:"user_agent"`
	Fingerprint string  `json:"fingerprint"`
	RiskScore   float64 `json:"risk_score"`
	Strategy    string  `json:"strategy"`
}

// CreateFlowResponse 创建验证流程响应
type CreateFlowResponse struct {
	FlowID     string              `json:"flow_id"`
	Steps      []*VerificationStep `json:"steps"`
	CurrentStep int                 `json:"current_step"`
	Strategy    string              `json:"strategy"`
	Required    int                 `json:"required_passed"`
	ExpiresAt   int64               `json:"expires_at"`
}

// CreateFlow 创建验证流程
func (s *ComboVerificationService) CreateFlow(ctx context.Context, req *CreateFlowRequest) (*CreateFlowResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	flowID := generateFlowID()
	
	riskLevel := s.assessRiskLevel(req.RiskScore, req.UserID)
	steps := s.generateSteps(flowID, riskLevel, req.UserID)
	
	strategy := req.Strategy
	if strategy == "" {
		strategy = determineStrategy(riskLevel)
	}
	
	requiredPassed := calculateRequiredPassed(len(steps), strategy)
	
	flow := &VerificationFlow{
		FlowID:         flowID,
		Steps:          steps,
		CurrentStep:    0,
		Status:         "active",
		Strategy:       strategy,
		RequiredPassed: requiredPassed,
		TotalSteps:     len(steps),
		PassedSteps:    0,
		FailedSteps:    0,
		CreatedAt:      time.Now(),
		ExpiresAt:      time.Now().Add(5 * time.Minute),
		UserID:         req.UserID,
		ClientIP:       req.ClientIP,
		UserAgent:      req.UserAgent,
		Fingerprint:    req.Fingerprint,
		RiskScore:      req.RiskScore,
	}

	s.flows[flowID] = flow

	return &CreateFlowResponse{
		FlowID:     flowID,
		Steps:      steps,
		CurrentStep: 0,
		Strategy:    strategy,
		Required:    requiredPassed,
		ExpiresAt:   flow.ExpiresAt.Unix(),
	}, nil
}

// GetFlow 获取验证流程
func (s *ComboVerificationService) GetFlow(ctx context.Context, flowID string) (*VerificationFlow, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	flow, exists := s.flows[flowID]
	if !exists {
		return nil, fmt.Errorf("flow not found")
	}

	if time.Now().After(flow.ExpiresAt) {
		return nil, fmt.Errorf("flow expired")
	}

	return flow, nil
}

// VerifyStepRequest 验证步骤请求
type VerifyStepRequest struct {
	FlowID     string      `json:"flow_id"`
	StepID     string      `json:"step_id"`
	CaptchaType string      `json:"captcha_type"`
	Answer     interface{} `json:"answer"`
	BehaviorData []models.BehaviorData `json:"behavior_data"`
}

// VerifyStepResponse 验证步骤响应
type VerifyStepResponse struct {
	Success         bool              `json:"success"`
	Message         string            `json:"message"`
	StepResult      *VerificationStep `json:"step_result"`
	CurrentStep     int               `json:"current_step"`
	RemainingSteps  int               `json:"remaining_steps"`
	PassedCount     int               `json:"passed_count"`
	FailedCount     int               `json:"failed_count"`
	FlowCompleted   bool              `json:"flow_completed"`
	FlowSuccess     bool              `json:"flow_success"`
	NextStep        *VerificationStep `json:"next_step,omitempty"`
}

// VerifyStep 验证单个步骤
func (s *ComboVerificationService) VerifyStep(ctx context.Context, req *VerifyStepRequest) (*VerifyStepResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	flow, exists := s.flows[req.FlowID]
	if !exists {
		return nil, fmt.Errorf("flow not found")
	}

	if time.Now().After(flow.ExpiresAt) {
		return nil, fmt.Errorf("flow expired")
	}

	if flow.Status == "completed" || flow.Status == "failed" {
		return nil, fmt.Errorf("flow already %s", flow.Status)
	}

	var step *VerificationStep
	for _, s := range flow.Steps {
		if s.StepID == req.StepID {
			step = s
			break
		}
	}

	if step == nil {
		return nil, fmt.Errorf("step not found")
	}

	if step.Completed {
		return nil, fmt.Errorf("step already completed")
	}

	success, score := s.verifyCaptchaAnswer(req.CaptchaType, req.Answer, req.BehaviorData)
	
	step.Completed = true
	step.Passed = success
	step.Score = score

	if success {
		flow.PassedSteps++
	} else {
		flow.FailedSteps++
	}

	flow.CurrentStep++

	flowCompleted := flow.CurrentStep >= len(flow.Steps)
	flowSuccess := false

	if flowCompleted {
		flowSuccess = flow.PassedSteps >= flow.RequiredPassed
		if flowSuccess {
			flow.Status = "completed"
		} else {
			flow.Status = "failed"
		}
	}

	var nextStep *VerificationStep
	if !flowCompleted {
		nextStep = flow.Steps[flow.CurrentStep]
	}

	return &VerifyStepResponse{
		Success:        success,
		Message:        mapResultMessage(success, step.CaptchaType),
		StepResult:     step,
		CurrentStep:    flow.CurrentStep,
		RemainingSteps: len(flow.Steps) - flow.CurrentStep,
		PassedCount:    flow.PassedSteps,
		FailedCount:    flow.FailedSteps,
		FlowCompleted:  flowCompleted,
		FlowSuccess:    flowSuccess,
		NextStep:       nextStep,
	}, nil
}

// VerifyAllRequest 验证所有步骤请求
type VerifyAllRequest struct {
	FlowID     string                 `json:"flow_id"`
	Answers    []StepAnswer           `json:"answers"`
	BehaviorData []models.BehaviorData `json:"behavior_data"`
}

// StepAnswer 步骤答案
type StepAnswer struct {
	StepID     string      `json:"step_id"`
	CaptchaType string      `json:"captcha_type"`
	Answer     interface{} `json:"answer"`
}

// VerifyAllResponse 验证所有步骤响应
type VerifyAllResponse struct {
	Success      bool                   `json:"success"`
	Message      string                 `json:"message"`
	Results      []*VerificationStep    `json:"results"`
	PassedCount  int                    `json:"passed_count"`
	FailedCount  int                    `json:"failed_count"`
	RequiredPassed int                   `json:"required_passed"`
	Score        float64                `json:"score"`
}

// VerifyAll 验证所有步骤
func (s *ComboVerificationService) VerifyAll(ctx context.Context, req *VerifyAllRequest) (*VerifyAllResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	flow, exists := s.flows[req.FlowID]
	if !exists {
		return nil, fmt.Errorf("flow not found")
	}

	if time.Now().After(flow.ExpiresAt) {
		return nil, fmt.Errorf("flow expired")
	}

	passedCount := 0
	failedCount := 0
	totalScore := 0.0

	for _, answer := range req.Answers {
		var step *VerificationStep
		for _, s := range flow.Steps {
			if s.StepID == answer.StepID {
				step = s
				break
			}
		}

		if step == nil {
			continue
		}

		success, score := s.verifyCaptchaAnswer(answer.CaptchaType, answer.Answer, req.BehaviorData)
		step.Completed = true
		step.Passed = success
		step.Score = score

		if success {
			passedCount++
		} else {
			failedCount++
		}
		totalScore += score
	}

	flow.PassedSteps = passedCount
	flow.FailedSteps = failedCount
	flow.CurrentStep = len(flow.Steps)

	success := passedCount >= flow.RequiredPassed
	if success {
		flow.Status = "completed"
	} else {
		flow.Status = "failed"
	}

	var avgScore float64
	if len(req.Answers) > 0 {
		avgScore = totalScore / float64(len(req.Answers))
	}

	return &VerifyAllResponse{
		Success:      success,
		Message:      mapResultMessage(success, ""),
		Results:      flow.Steps,
		PassedCount:  passedCount,
		FailedCount:  failedCount,
		RequiredPassed: flow.RequiredPassed,
		Score:        avgScore,
	}, nil
}

// DeleteFlow 删除验证流程
func (s *ComboVerificationService) DeleteFlow(ctx context.Context, flowID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.flows, flowID)
	return nil
}

// CleanupExpiredFlows 清理过期流程
func (s *ComboVerificationService) CleanupExpiredFlows(ctx context.Context) int {
	s.mu.Lock()
	defer s.mu.Unlock()

	count := 0
	now := time.Now()
	for flowID, flow := range s.flows {
		if now.After(flow.ExpiresAt) {
			delete(s.flows, flowID)
			count++
		}
	}
	return count
}

// 评估风险等级
func (s *ComboVerificationService) assessRiskLevel(riskScore float64, userID string) string {
	if riskScore >= 80 {
		return "critical"
	} else if riskScore >= 60 {
		return "high"
	} else if riskScore >= 40 {
		return "medium"
	}
	return "low"
}

// 生成验证步骤
func (s *ComboVerificationService) generateSteps(flowID string, riskLevel, userID string) []*VerificationStep {
	var steps []*VerificationStep
	var captchaTypes []CaptchaType

	switch riskLevel {
	case "critical":
		captchaTypes = []CaptchaType{CaptchaType3D, CaptchaTypeSemantic, CaptchaTypeVoice}
	case "high":
		captchaTypes = []CaptchaType{CaptchaType3D, CaptchaTypeClick, CaptchaTypeMath}
	case "medium":
		captchaTypes = []CaptchaType{CaptchaTypeSlider, CaptchaTypeIcon}
	default:
		captchaTypes = []CaptchaType{CaptchaTypeSlider}
	}

	if s.difficultySvc != nil {
		difficulty := s.difficultySvc.GetDifficulty(userID)
		for i, ct := range captchaTypes {
			step := &VerificationStep{
				StepID:      generateStepID(flowID, i),
				CaptchaType: ct,
				Difficulty:  string(difficulty),
				Required:    i < len(captchaTypes)-1,
				Order:       i + 1,
				SessionID:   generateSessionID(),
			}
			steps = append(steps, step)
		}
	} else {
		for i, ct := range captchaTypes {
			step := &VerificationStep{
				StepID:      generateStepID(flowID, i),
				CaptchaType: ct,
				Difficulty:  "medium",
				Required:    i < len(captchaTypes)-1,
				Order:       i + 1,
				SessionID:   generateSessionID(),
			}
			steps = append(steps, step)
		}
	}

	return steps
}

// 验证验证码答案
func (s *ComboVerificationService) verifyCaptchaAnswer(captchaType string, answer interface{}, behaviorData []models.BehaviorData) (bool, float64) {
	// 模拟验证逻辑
	switch CaptchaType(captchaType) {
	case CaptchaTypeSlider:
		return verifySliderAnswer(answer)
	case CaptchaTypeClick:
		return verifyClickAnswer(answer)
	case CaptchaTypeIcon:
		return verifyIconAnswer(answer)
	case CaptchaType3D:
		return verify3DAnswer(answer)
	case CaptchaTypeSemantic:
		return verifySemanticAnswer(answer)
	case CaptchaTypeEmoji:
		return verifyEmojiAnswer(answer)
	case CaptchaTypeVoice:
		return verifyVoiceAnswer(answer)
	case CaptchaTypeMath:
		return verifyMathAnswer(answer)
	case CaptchaTypeColor:
		return verifyColorAnswer(answer)
	case CaptchaTypePhrase:
		return verifyPhraseAnswer(answer)
	default:
		return false, 0
	}
}

// 模拟各类型验证码验证
func verifySliderAnswer(answer interface{}) (bool, float64) {
	if ans, ok := answer.(map[string]interface{}); ok {
		if x, ok := ans["x"].(float64); ok {
			targetX := 150.0
			if math.Abs(x-targetX) <= 10 {
				return true, 100
			} else if math.Abs(x-targetX) <= 20 {
				return true, 80
			}
		}
	}
	return false, 0
}

func verifyClickAnswer(answer interface{}) (bool, float64) {
	if points, ok := answer.([]interface{}); ok && len(points) >= 2 {
		return true, 100
	}
	return false, 0
}

func verifyIconAnswer(answer interface{}) (bool, float64) {
	if points, ok := answer.([]interface{}); ok && len(points) >= 2 {
		return true, 100
	}
	return false, 0
}

func verify3DAnswer(answer interface{}) (bool, float64) {
	if ans, ok := answer.(map[string]interface{}); ok {
		if _, exists := ans["rotation"]; exists {
			return true, 90 + rand.Float64()*10
		}
	}
	return false, 0
}

func verifySemanticAnswer(answer interface{}) (bool, float64) {
	if ans, ok := answer.(string); ok && len(ans) >= 3 {
		return true, 95 + rand.Float64()*5
	}
	return false, 0
}

func verifyEmojiAnswer(answer interface{}) (bool, float64) {
	if emojis, ok := answer.([]interface{}); ok && len(emojis) > 0 {
		return true, 100
	}
	return false, 0
}

func verifyVoiceAnswer(answer interface{}) (bool, float64) {
	if ans, ok := answer.(string); ok && len(ans) >= 4 {
		return true, 90 + rand.Float64()*10
	}
	return false, 0
}

func verifyMathAnswer(answer interface{}) (bool, float64) {
	if ans, ok := answer.(float64); ok && ans > 0 {
		return true, 100
	}
	return false, 0
}

func verifyColorAnswer(answer interface{}) (bool, float64) {
	if points, ok := answer.([]interface{}); ok && len(points) >= 2 {
		return true, 100
	}
	return false, 0
}

func verifyPhraseAnswer(answer interface{}) (bool, float64) {
	if points, ok := answer.([]interface{}); ok && len(points) >= 2 {
		return true, 100
	}
	return false, 0
}

// 工具函数
func generateFlowID() string {
	return fmt.Sprintf("flow_%d_%d", time.Now().UnixNano(), rand.Intn(10000))
}

func generateStepID(flowID string, index int) string {
	return fmt.Sprintf("%s_step_%d", flowID, index)
}

func generateSessionID() string {
	return fmt.Sprintf("sess_%d_%d", time.Now().UnixNano(), rand.Intn(10000))
}

func determineStrategy(riskLevel string) string {
	switch riskLevel {
	case "critical":
		return "all"
	case "high":
		return "majority"
	default:
		return "any"
	}
}

func calculateRequiredPassed(totalSteps int, strategy string) int {
	switch strategy {
	case "all":
		return totalSteps
	case "majority":
		return totalSteps/2 + 1
	case "any":
		return 1
	default:
		return totalSteps
	}
}

func mapResultMessage(success bool, captchaType CaptchaType) string {
	if success {
		return "验证成功"
	}
	return "验证失败，请重试"
}

// ExportFlowJSON 导出流程为JSON
func (f *VerificationFlow) ExportFlowJSON() (string, error) {
	data, err := json.Marshal(f)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// ImportFlowJSON 从JSON导入流程
func ImportFlowJSON(jsonData string) (*VerificationFlow, error) {
	var flow VerificationFlow
	err := json.Unmarshal([]byte(jsonData), &flow)
	if err != nil {
		return nil, err
	}
	return &flow, nil
}