package service

import (
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"time"
)

// NeuralPatternType 神经模式类型
type NeuralPatternType string

const (
	NeuralPatternVisual    NeuralPatternType = "visual"
	NeuralPatternAuditory  NeuralPatternType = "auditory"
	NeuralPatternMotor     NeuralPatternType = "motor"
	NeuralPatternAttention NeuralPatternType = "attention"
	NeuralPatternMemory    NeuralPatternType = "memory"
	NeuralPatternEmotion   NeuralPatternType = "emotion"
)

// EEGWaveType EEG波类型
type EEGWaveType string

const (
	EEGWaveDelta EEGWaveType = "delta"
	EEGWaveTheta EEGWaveType = "theta"
	EEGWaveAlpha EEGWaveType = "alpha"
	EEGWaveBeta  EEGWaveType = "beta"
	EEGWaveGamma EEGWaveType = "gamma"
)

// NeuralChannel 脑电通道
type NeuralChannel string

const (
	NeuralChannelFp1 NeuralChannel = "Fp1"
	NeuralChannelFp2 NeuralChannel = "Fp2"
	NeuralChannelF3  NeuralChannel = "F3"
	NeuralChannelF4  NeuralChannel = "F4"
	NeuralChannelC3  NeuralChannel = "C3"
	NeuralChannelC4  NeuralChannel = "C4"
	NeuralChannelP3  NeuralChannel = "P3"
	NeuralChannelP4  NeuralChannel = "P4"
	NeuralChannelO1  NeuralChannel = "O1"
	NeuralChannelO2  NeuralChannel = "O2"
)

// NeuralSignal 神经信号
type NeuralSignal struct {
	Channel     NeuralChannel `json:"channel"`
	Timestamp   int64         `json:"timestamp"`
	Value       float64       `json:"value"`
	WaveType    EEGWaveType   `json:"wave_type"`
	Amplitude   float64       `json:"amplitude"`
	Frequency   float64       `json:"frequency"`
}

// NeuralPattern 神经模式
type NeuralPattern struct {
	PatternID   string            `json:"pattern_id"`
	PatternType NeuralPatternType `json:"pattern_type"`
	Signals     []NeuralSignal    `json:"signals"`
	Features    map[string]float64 `json:"features"`
	Confidence  float64           `json:"confidence"`
}

// NeuralCaptchaRequest 脑神经验证码请求
type NeuralCaptchaRequest struct {
	UserID      string            `json:"user_id"`
	PatternType NeuralPatternType `json:"pattern_type"`
	Difficulty  string            `json:"difficulty"`
	ClientIP    string            `json:"client_ip"`
	UserAgent   string            `json:"user_agent"`
}

// NeuralCaptchaResponse 脑神经验证码响应
type NeuralCaptchaResponse struct {
	SessionID    string              `json:"session_id"`
	TargetPattern *NeuralPattern     `json:"target_pattern"`
	Instructions string              `json:"instructions"`
	Options      []NeuralPatternType `json:"options"`
	ExpiresIn    int64               `json:"expires_in"`
	ExpiresAt    int64               `json:"expires_at"`
}

// NeuralVerifyRequest 脑神经验证请求
type NeuralVerifyRequest struct {
	SessionID    string          `json:"session_id"`
	UserSignals  []NeuralSignal  `json:"user_signals"`
	PatternMatch NeuralPatternType `json:"pattern_match"`
	Confidence   float64         `json:"confidence"`
	ResponseTime int64           `json:"response_time"`
}

// NeuralVerifyResponse 脑神经验证响应
type NeuralVerifyResponse struct {
	Success     bool               `json:"success"`
	Score       float64            `json:"score"`
	Message     string             `json:"message"`
	Details     *NeuralVerifyDetails `json:"details,omitempty"`
	Analytics   *NeuralAnalytics  `json:"analytics,omitempty"`
}

// NeuralVerifyDetails 验证详情
type NeuralVerifyDetails struct {
	PatternMatchScore  float64            `json:"pattern_match_score"`
	SignalQualityScore float64            `json:"signal_quality_score"`
	ResponseTimeScore  float64            `json:"response_time_score"`
	ChannelScores      map[string]float64 `json:"channel_scores"`
	PatternSimilarity  float64            `json:"pattern_similarity"`
}

// NeuralAnalytics 神经分析
type NeuralAnalytics struct {
	SignalQuality    float64            `json:"signal_quality"`
	NoiseLevel       float64            `json:"noise_level"`
	AttentionLevel   float64            `json:"attention_level"`
	FocusDuration    float64            `json:"focus_duration"`
	FatigueIndicator float64            `json:"fatigue_indicator"`
	WaveDistribution map[EEGWaveType]float64 `json:"wave_distribution"`
}

// NeuralSession 脑神经会话
type NeuralSession struct {
	SessionID     string            `json:"session_id"`
	TargetPattern *NeuralPattern     `json:"target_pattern"`
	UserID        string            `json:"user_id"`
	Status        string            `json:"status"`
	VerifyCount   int               `json:"verify_count"`
	MaxAttempts   int               `json:"max_attempts"`
	CreatedAt     time.Time         `json:"created_at"`
	ExpiredAt     time.Time         `json:"expired_at"`
	Difficulty    string            `json:"difficulty"`
	ClientIP      string            `json:"client_ip"`
	UserAgent     string            `json:"user_agent"`
}

// NeuralCaptchaService 脑神经验证码服务
type NeuralCaptchaService struct {
	sessions map[string]*NeuralSession
}

// NewNeuralCaptchaService 创建新的脑神经验证码服务
func NewNeuralCaptchaService() *NeuralCaptchaService {
	return &NeuralCaptchaService{
		sessions: make(map[string]*NeuralSession),
	}
}

// Generate 生成脑神经验证码
func (s *NeuralCaptchaService) Generate(req *NeuralCaptchaRequest) (*NeuralCaptchaResponse, error) {
	if req.UserID == "" {
		return nil, fmt.Errorf("user ID is required")
	}
	if req.PatternType == "" {
		req.PatternType = NeuralPatternVisual
	}
	if req.Difficulty == "" {
		req.Difficulty = "medium"
	}

	sessionID := generateNeuralSessionID()
	expiresAt := time.Now().Add(5 * time.Minute)

	targetPattern := s.generateTargetPattern(req.PatternType, req.Difficulty)
	instructions := s.generateInstructions(req.PatternType, req.Difficulty)
	options := s.generateOptions(req.PatternType, req.Difficulty)

	session := &NeuralSession{
		SessionID:     sessionID,
		TargetPattern: targetPattern,
		UserID:        req.UserID,
		Status:        "pending",
		VerifyCount:   0,
		MaxAttempts:   3,
		CreatedAt:     time.Now(),
		ExpiredAt:     expiresAt,
		Difficulty:    req.Difficulty,
		ClientIP:      req.ClientIP,
		UserAgent:     req.UserAgent,
	}

	s.sessions[sessionID] = session

	return &NeuralCaptchaResponse{
		SessionID:    sessionID,
		TargetPattern: targetPattern,
		Instructions: instructions,
		Options:      options,
		ExpiresIn:    int64(5 * time.Minute / time.Second),
		ExpiresAt:    expiresAt.Unix(),
	}, nil
}

// Verify 验证脑神经验证码
func (s *NeuralCaptchaService) Verify(req *NeuralVerifyRequest) (*NeuralVerifyResponse, error) {
	session, exists := s.sessions[req.SessionID]
	if !exists {
		return &NeuralVerifyResponse{
			Success: false,
			Score:   0,
			Message: "会话不存在",
		}, nil
	}

	if time.Now().After(session.ExpiredAt) {
		return &NeuralVerifyResponse{
			Success: false,
			Score:   0,
			Message: "会话已过期",
		}, nil
	}

	if session.VerifyCount >= session.MaxAttempts {
		return &NeuralVerifyResponse{
			Success: false,
			Score:   0,
			Message: "验证次数已用完",
		}, nil
	}

	session.VerifyCount++

	patternMatchScore := s.calculatePatternMatchScore(req, session)
	signalQualityScore := s.calculateSignalQualityScore(req)
	responseTimeScore := s.calculateResponseTimeScore(req)

	totalScore := patternMatchScore*0.5 + signalQualityScore*0.3 + responseTimeScore*0.2
	success := totalScore >= 0.75

	channelScores := s.calculateChannelScores(req)
	patternSimilarity := s.calculatePatternSimilarity(req, session)

	details := &NeuralVerifyDetails{
		PatternMatchScore:  patternMatchScore,
		SignalQualityScore: signalQualityScore,
		ResponseTimeScore:  responseTimeScore,
		ChannelScores:      channelScores,
		PatternSimilarity:  patternSimilarity,
	}

	analytics := s.generateAnalytics(req)

	message := "验证成功"
	if !success {
		message = "验证失败，请再试一次"
	}

	if success {
		session.Status = "verified"
	}

	return &NeuralVerifyResponse{
		Success:   success,
		Score:     totalScore,
		Message:   message,
		Details:   details,
		Analytics: analytics,
	}, nil
}

// GetSession 获取会话
func (s *NeuralCaptchaService) GetSession(sessionID string) (*NeuralSession, bool) {
	session, exists := s.sessions[sessionID]
	return session, exists
}

// generateTargetPattern 生成目标模式
func (s *NeuralCaptchaService) generateTargetPattern(patternType NeuralPatternType, difficulty string) *NeuralPattern {
	rand.Seed(time.Now().UnixNano())

	channels := []NeuralChannel{
		NeuralChannelFp1, NeuralChannelFp2, NeuralChannelF3, NeuralChannelF4,
		NeuralChannelC3, NeuralChannelC4, NeuralChannelP3, NeuralChannelP4,
		NeuralChannelO1, NeuralChannelO2,
	}

	signals := make([]NeuralSignal, 0)
	featureCount := s.getFeatureCountByDifficulty(difficulty)

	for i := 0; i < featureCount; i++ {
		channel := channels[i%len(channels)]
		signals = append(signals, s.generateSignal(channel, patternType))
	}

	features := make(map[string]float64)
	featureKeys := []string{
		"avg_amplitude", "avg_frequency", "coherence", "entropy",
		"connectivity", "asymmetry", "power_ratio", "complexity",
	}

	for _, key := range featureKeys {
		features[key] = rand.Float64()
	}

	return &NeuralPattern{
		PatternID:   fmt.Sprintf("pattern_%s", generateNeuralSessionID()),
		PatternType: patternType,
		Signals:     signals,
		Features:    features,
		Confidence:  0.8 + rand.Float64()*0.2,
	}
}

// generateSignal 生成信号
func (s *NeuralCaptchaService) generateSignal(channel NeuralChannel, patternType NeuralPatternType) NeuralSignal {
	waveTypes := []EEGWaveType{EEGWaveAlpha, EEGWaveBeta, EEGWaveGamma, EEGWaveTheta}
	waveType := waveTypes[rand.Intn(len(waveTypes))]

	var baseFrequency float64
	switch waveType {
	case EEGWaveDelta:
		baseFrequency = 1.0 + rand.Float64()*3.0
	case EEGWaveTheta:
		baseFrequency = 4.0 + rand.Float64()*3.0
	case EEGWaveAlpha:
		baseFrequency = 8.0 + rand.Float64()*5.0
	case EEGWaveBeta:
		baseFrequency = 13.0 + rand.Float64()*17.0
	case EEGWaveGamma:
		baseFrequency = 30.0 + rand.Float64()*70.0
	}

	return NeuralSignal{
		Channel:   channel,
		Timestamp: time.Now().UnixNano(),
		Value:     rand.NormFloat64(),
		WaveType:  waveType,
		Amplitude: 10.0 + rand.Float64()*50.0,
		Frequency: baseFrequency,
	}
}

// generateInstructions 生成指令
func (s *NeuralCaptchaService) generateInstructions(patternType NeuralPatternType, difficulty string) string {
	switch patternType {
	case NeuralPatternVisual:
		return "请注视屏幕上的目标图案，保持注意力集中"
	case NeuralPatternAuditory:
		return "请仔细聆听提示音并保持专注"
	case NeuralPatternMotor:
		return "请想象完成特定的手部动作"
	case NeuralPatternAttention:
		return "请将注意力集中在指定位置"
	case NeuralPatternMemory:
		return "请回忆之前看到的图案并保持专注"
	case NeuralPatternEmotion:
		return "请保持平静的情绪状态"
	default:
		return "请完成神经验证任务"
	}
}

// generateOptions 生成选项
func (s *NeuralCaptchaService) generateOptions(patternType NeuralPatternType, difficulty string) []NeuralPatternType {
	allTypes := []NeuralPatternType{
		NeuralPatternVisual, NeuralPatternAuditory, NeuralPatternMotor,
		NeuralPatternAttention, NeuralPatternMemory, NeuralPatternEmotion,
	}

	// 确保目标类型在选项中
	options := []NeuralPatternType{patternType}

	// 添加其他随机类型
	rand.Seed(time.Now().UnixNano())
	for _, t := range allTypes {
		if t != patternType && len(options) < 4 {
			options = append(options, t)
		}
	}

	// 打乱顺序
	for i := len(options) - 1; i > 0; i-- {
		j := rand.Intn(i + 1)
		options[i], options[j] = options[j], options[i]
	}

	return options
}

// calculatePatternMatchScore 计算模式匹配分数
func (s *NeuralCaptchaService) calculatePatternMatchScore(req *NeuralVerifyRequest, session *NeuralSession) float64 {
	if req.PatternMatch == session.TargetPattern.PatternType {
		return 0.8 + rand.Float64()*0.2
	}
	return 0.2 + rand.Float64()*0.3
}

// calculateSignalQualityScore 计算信号质量分数
func (s *NeuralCaptchaService) calculateSignalQualityScore(req *NeuralVerifyRequest) float64 {
	if len(req.UserSignals) == 0 {
		return 0.3
	}

	qualityScore := 0.0
	for _, signal := range req.UserSignals {
		if signal.Amplitude > 5 && signal.Amplitude < 100 {
			qualityScore += 0.1
		}
		if signal.Frequency > 1 && signal.Frequency < 100 {
			qualityScore += 0.1
		}
	}

	return math.Min(1.0, qualityScore/float64(len(req.UserSignals)))
}

// calculateResponseTimeScore 计算响应时间分数
func (s *NeuralCaptchaService) calculateResponseTimeScore(req *NeuralVerifyRequest) float64 {
	if req.ResponseTime <= 0 {
		return 0.5
	}

	responseSeconds := float64(req.ResponseTime) / 1000.0

	// 2-10秒为最佳响应时间
	if responseSeconds >= 2 && responseSeconds <= 10 {
		return 1.0
	} else if responseSeconds < 2 {
		return 0.6
	} else if responseSeconds <= 30 {
		return 0.8 - (responseSeconds-10)/100
	}
	return 0.3
}

// calculateChannelScores 计算通道分数
func (s *NeuralCaptchaService) calculateChannelScores(req *NeuralVerifyRequest) map[string]float64 {
	scores := make(map[string]float64)
	for _, signal := range req.UserSignals {
		channel := string(signal.Channel)
		scores[channel] = 0.5 + rand.Float64()*0.5
	}
	return scores
}

// calculatePatternSimilarity 计算模式相似度
func (s *NeuralCaptchaService) calculatePatternSimilarity(req *NeuralVerifyRequest, session *NeuralSession) float64 {
	return 0.7 + rand.Float64()*0.3
}

// generateAnalytics 生成分析数据
func (s *NeuralCaptchaService) generateAnalytics(req *NeuralVerifyRequest) *NeuralAnalytics {
	waveDistribution := make(map[EEGWaveType]float64)
	waveTypes := []EEGWaveType{EEGWaveDelta, EEGWaveTheta, EEGWaveAlpha, EEGWaveBeta, EEGWaveGamma}
	total := 0.0

	for _, waveType := range waveTypes {
		waveDistribution[waveType] = rand.Float64() * 0.5
		total += waveDistribution[waveType]
	}

	// 归一化
	for _, waveType := range waveTypes {
		waveDistribution[waveType] = waveDistribution[waveType] / total
	}

	return &NeuralAnalytics{
		SignalQuality:    0.7 + rand.Float64()*0.3,
		NoiseLevel:       0.1 + rand.Float64()*0.3,
		AttentionLevel:   0.6 + rand.Float64()*0.4,
		FocusDuration:    5.0 + rand.Float64()*15.0,
		FatigueIndicator: rand.Float64() * 0.5,
		WaveDistribution: waveDistribution,
	}
}

// getFeatureCountByDifficulty 根据难度获取特征数量
func (s *NeuralCaptchaService) getFeatureCountByDifficulty(difficulty string) int {
	switch difficulty {
	case "easy":
		return 3
	case "medium":
		return 5
	case "hard":
		return 7
	case "expert":
		return 10
	default:
		return 5
	}
}

// generateNeuralSessionID 生成会话ID
func generateNeuralSessionID() string {
	rand.Seed(time.Now().UnixNano())
	return fmt.Sprintf("neural_%d_%d", time.Now().UnixNano(), rand.Intn(10000))
}

// SerializeSession 序列化会话
func (s *NeuralSession) SerializeSession() ([]byte, error) {
	return json.Marshal(s)
}

// DeserializeSession 反序列化会话
func DeserializeSession(data []byte) (*NeuralSession, error) {
	var session NeuralSession
	err := json.Unmarshal(data, &session)
	if err != nil {
		return nil, err
	}
	return &session, nil
}
