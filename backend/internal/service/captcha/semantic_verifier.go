package captcha

import (
	"context"
	"regexp"
	"strings"
	"time"
)

type SemanticVerifierService struct {
	generatorService *SemanticGeneratorService
	languageAnalyzers map[string]SemanticLanguageAnalyzer
}

type VerifySemanticCaptchaRequest struct {
	SessionID    string                   `json:"session_id"`
	Answer       string                    `json:"answer"`
	BehaviorData SemanticBehaviorData      `json:"behavior_data"`
}

type VerifySemanticCaptchaResponse struct {
	Success       bool                      `json:"success"`
	Message       string                    `json:"message"`
	Score         float64                   `json:"score"`
	Analysis      SemanticAnalysisResult    `json:"analysis,omitempty"`
}

type SemanticBehaviorData struct {
	ClickTimes      []int64  `json:"click_times"`
	ClickIntervals  []int64  `json:"click_intervals"`
	TotalTime       int64    `json:"total_time"`
	ReadTime        int64    `json:"read_time"`
	DecisionTime    int64    `json:"decision_time"`
	IsMobile        bool     `json:"is_mobile"`
	TouchEvents     int      `json:"touch_events"`
	MouseMovements  int      `json:"mouse_movements"`
}

type SemanticAnalysisResult struct {
	CorrectAnswer     string   `json:"correct_answer,omitempty"`
	AnswerMatch       bool     `json:"answer_match"`
	Confidence        float64  `json:"confidence"`
	RiskLevel         string   `json:"risk_level"`
	BehaviorScore     float64  `json:"behavior_score"`
	LanguageScore     float64  `json:"language_score"`
	TimeScore         float64  `json:"time_score"`
	ConsistencyScore  float64  `json:"consistency_score"`
	IsBot             bool     `json:"is_bot"`
	Indicators        []string `json:"risk_indicators"`
}

type SemanticLanguageAnalyzer interface {
	Analyze(text string) float64
	Validate(text string) bool
}

type ChineseAnalyzer struct{}

type EnglishAnalyzer struct{}

type JapaneseAnalyzer struct{}

type SpanishAnalyzer struct{}

func NewSemanticVerifierService(generatorService *SemanticGeneratorService) *SemanticVerifierService {
	return &SemanticVerifierService{
		generatorService: generatorService,
		languageAnalyzers: map[string]SemanticLanguageAnalyzer{
			"zh":    &ChineseAnalyzer{},
			"ja":    &JapaneseAnalyzer{},
			"es":    &SpanishAnalyzer{},
			"en":    &EnglishAnalyzer{},
		},
	}
}

func NewSemanticVerifierServiceSimple() *SemanticVerifierService {
	return &SemanticVerifierService{
		generatorService: NewSemanticGeneratorServiceSimple(),
		languageAnalyzers: map[string]SemanticLanguageAnalyzer{
			"zh":    &ChineseAnalyzer{},
			"ja":    &JapaneseAnalyzer{},
			"es":    &SpanishAnalyzer{},
			"en":    &EnglishAnalyzer{},
		},
	}
}

func (s *SemanticVerifierService) Verify(ctx context.Context, req *VerifySemanticCaptchaRequest) (*VerifySemanticCaptchaResponse, error) {
	session, err := s.generatorService.GetSession(ctx, req.SessionID)
	if err != nil {
		return &VerifySemanticCaptchaResponse{
			Success: false,
			Message: "会话不存在或已过期",
			Score:   0,
		}, nil
	}

	if time.Now().After(session.ExpiredAt) {
		return &VerifySemanticCaptchaResponse{
			Success: false,
			Message: "会话已过期",
			Score:   0,
		}, nil
	}

	if session.VerifyCount >= session.MaxAttempts {
		return &VerifySemanticCaptchaResponse{
			Success: false,
			Message: "验证次数已用完",
			Score:   0,
		}, nil
	}

	session.VerifyCount++
	s.generatorService.UpdateSession(ctx, session)

	analysis := s.analyzeVerification(req, session)

	if session.Status == "verified" {
		return &VerifySemanticCaptchaResponse{
			Success: true,
			Message: "验证码已验证通过",
			Score:   100,
			Analysis: SemanticAnalysisResult{
				CorrectAnswer: session.CorrectAnswer,
			},
		}, nil
	}

	if analysis.AnswerMatch && !analysis.IsBot {
		session.Status = "verified"
		session.RiskScore = analysis.Confidence
		s.generatorService.UpdateSession(ctx, session)

		return &VerifySemanticCaptchaResponse{
			Success: true,
			Message: "验证成功",
			Score:   analysis.Confidence,
			Analysis: SemanticAnalysisResult{
				CorrectAnswer:     session.CorrectAnswer,
				AnswerMatch:        true,
				Confidence:        analysis.Confidence,
				RiskLevel:         analysis.RiskLevel,
				BehaviorScore:     analysis.BehaviorScore,
				LanguageScore:     analysis.LanguageScore,
				TimeScore:         analysis.TimeScore,
				ConsistencyScore:  analysis.ConsistencyScore,
				IsBot:             false,
			},
		}, nil
	}

	return &VerifySemanticCaptchaResponse{
		Success: false,
		Message: "验证失败",
		Score:   analysis.Confidence,
		Analysis: SemanticAnalysisResult{
			CorrectAnswer:    session.CorrectAnswer,
			AnswerMatch:       analysis.AnswerMatch,
			Confidence:        analysis.Confidence,
			RiskLevel:         analysis.RiskLevel,
			BehaviorScore:     analysis.BehaviorScore,
			LanguageScore:     analysis.LanguageScore,
			TimeScore:         analysis.TimeScore,
			ConsistencyScore:  analysis.ConsistencyScore,
			IsBot:             analysis.IsBot,
			Indicators:        analysis.Indicators,
		},
	}, nil
}

func (s *SemanticVerifierService) analyzeVerification(req *VerifySemanticCaptchaRequest, session *SemanticCaptchaSession) SemanticAnalysisResult {
	result := SemanticAnalysisResult{}

	normalizedAnswer := strings.ToUpper(strings.TrimSpace(req.Answer))
	normalizedCorrect := strings.ToUpper(strings.TrimSpace(session.CorrectAnswer))

	result.AnswerMatch = normalizedAnswer == normalizedCorrect

	if !result.AnswerMatch {
		result.Confidence = 0
		result.RiskLevel = "low"
		result.IsBot = false
		result.Indicators = []string{"incorrect_answer"}
		return result
	}

	result.TimeScore = s.analyzeTimeBehavior(&req.BehaviorData)

	result.BehaviorScore = s.analyzeBehaviorPattern(&req.BehaviorData)

	result.LanguageScore = s.analyzeLanguagePattern(req.Answer, session.Language)

	result.ConsistencyScore = s.analyzeConsistency(&req.BehaviorData)

	result.Confidence = (result.TimeScore + result.BehaviorScore + result.LanguageScore + result.ConsistencyScore) / 4

	if result.Confidence >= 80 {
		result.RiskLevel = "low"
		result.IsBot = false
	} else if result.Confidence >= 50 {
		result.RiskLevel = "medium"
		result.IsBot = false
	} else {
		result.RiskLevel = "high"
		result.IsBot = true
	}

	result.Indicators = s.generateRiskIndicators(result)

	return result
}

func (s *SemanticVerifierService) analyzeTimeBehavior(data *SemanticBehaviorData) float64 {
	if data.TotalTime == 0 {
		return 50
	}

	minReadTime := int64(500)
	maxReadTime := int64(120000)

	score := 100.0

	if data.ReadTime < minReadTime {
		score -= 50
	} else if data.ReadTime < minReadTime*2 {
		score -= 25
	}

	if data.DecisionTime < 100 {
		score -= 30
	} else if data.DecisionTime < 500 {
		score -= 10
	}

	totalTime := data.TotalTime
	if totalTime < 1000 {
		score -= 40
	} else if totalTime < 2000 {
		score -= 20
	} else if totalTime > maxReadTime {
		score -= 10
	}

	if score < 0 {
		score = 0
	}

	return score
}

func (s *SemanticVerifierService) analyzeBehaviorPattern(data *SemanticBehaviorData) float64 {
	score := 100.0

	if data.TouchEvents == 0 && data.MouseMovements == 0 && !data.IsMobile {
		score -= 30
	}

	if data.IsMobile && data.TouchEvents == 0 {
		score -= 40
	}

	if len(data.ClickIntervals) > 0 {
		avgInterval := int64(0)
		for _, interval := range data.ClickIntervals {
			avgInterval += interval
		}
		avgInterval /= int64(len(data.ClickIntervals))

		if avgInterval < 50 {
			score -= 30
		} else if avgInterval < 200 {
			score -= 10
		}
	}

	variance := s.calculateIntervalVariance(data.ClickIntervals)
	if variance < 10 {
		score -= 20
	}

	if score < 0 {
		score = 0
	}

	return score
}

func (s *SemanticVerifierService) calculateIntervalVariance(intervals []int64) float64 {
	if len(intervals) < 2 {
		return 100
	}

	sum := int64(0)
	for _, interval := range intervals {
		sum += interval
	}
	mean := float64(sum) / float64(len(intervals))

	varianceSum := 0.0
	for _, interval := range intervals {
		diff := float64(interval) - mean
		varianceSum += diff * diff
	}

	return varianceSum / float64(len(intervals)-1)
}

func (s *SemanticVerifierService) analyzeLanguagePattern(answer string, language string) float64 {
	analyzer, exists := s.languageAnalyzers[language]
	if !exists {
		analyzer = &EnglishAnalyzer{}
	}

	if !analyzer.Validate(answer) {
		return 50
	}

	score := analyzer.Analyze(answer)

	return score * 100
}

func (s *SemanticVerifierService) analyzeConsistency(data *SemanticBehaviorData) float64 {
	score := 100.0

	if len(data.ClickTimes) >= 2 {
		firstClickTime := data.ClickTimes[0]
		for i := 1; i < len(data.ClickTimes); i++ {
			timeDiff := data.ClickTimes[i] - data.ClickTimes[i-1]
			if timeDiff < 0 || timeDiff > 30000 {
				score -= 20
			}
		}

		if data.ReadTime > 0 && firstClickTime > data.ReadTime+data.TotalTime {
			score -= 15
		}
	}

	if score < 0 {
		score = 0
	}

	return score
}

func (s *SemanticVerifierService) generateRiskIndicators(result SemanticAnalysisResult) []string {
	indicators := []string{}

	if result.TimeScore < 50 {
		indicators = append(indicators, "异常快速答题")
	}

	if result.BehaviorScore < 50 {
		indicators = append(indicators, "行为模式异常")
	}

	if result.LanguageScore < 50 {
		indicators = append(indicators, "语言验证异常")
	}

	if result.ConsistencyScore < 50 {
		indicators = append(indicators, "行为一致性异常")
	}

	if len(indicators) == 0 {
		indicators = append(indicators, "验证通过")
	}

	return indicators
}

func (a *ChineseAnalyzer) Analyze(text string) float64 {
	chineseCharCount := 0
	for _, r := range text {
		if r >= 0x4e00 && r <= 0x9fa5 {
			chineseCharCount++
		}
	}

	if len(text) == 0 {
		return 0.5
	}

	ratio := float64(chineseCharCount) / float64(len([]rune(text)))

	if ratio > 0.5 {
		return 1.0
	} else if ratio > 0.2 {
		return 0.7
	}
	return 0.3
}

func (a *ChineseAnalyzer) Validate(text string) bool {
	if len(text) == 0 {
		return false
	}

	hasValidChar := false
	for _, r := range text {
		if (r >= 0x4e00 && r <= 0x9fa5) || (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') {
			hasValidChar = true
		}
	}

	return hasValidChar
}

func (a *EnglishAnalyzer) Analyze(text string) float64 {
	englishCharCount := 0
	for _, r := range text {
		if (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') {
			englishCharCount++
		}
	}

	if len(text) == 0 {
		return 0.5
	}

	ratio := float64(englishCharCount) / float64(len([]rune(text)))

	if ratio > 0.7 {
		return 1.0
	} else if ratio > 0.3 {
		return 0.7
	}
	return 0.3
}

func (a *EnglishAnalyzer) Validate(text string) bool {
	if len(text) == 0 {
		return false
	}

	pattern := regexp.MustCompile(`^[A-Za-z]$`)
	return pattern.MatchString(text)
}

func (a *JapaneseAnalyzer) Analyze(text string) float64 {
	japaneseCharCount := 0
	for _, r := range text {
		if (r >= 0x3040 && r <= 0x309f) || (r >= 0x30a0 && r <= 0x30ff) {
			japaneseCharCount++
		}
	}

	if len(text) == 0 {
		return 0.5
	}

	ratio := float64(japaneseCharCount) / float64(len([]rune(text)))

	if ratio > 0.5 {
		return 1.0
	} else if ratio > 0.2 {
		return 0.7
	}
	return 0.3
}

func (a *JapaneseAnalyzer) Validate(text string) bool {
	if len(text) == 0 {
		return false
	}

	hasValidChar := false
	for _, r := range text {
		if (r >= 0x3040 && r <= 0x309f) || (r >= 0x30a0 && r <= 0x30ff) || (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') {
			hasValidChar = true
		}
	}

	return hasValidChar
}

func (a *SpanishAnalyzer) Analyze(text string) float64 {
	spanishCharCount := 0
	for _, r := range text {
		if (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || r == 'ñ' || r == 'Ñ' || r == 'á' || r == 'é' || r == 'í' || r == 'ó' || r == 'ú' || r == 'ü' || r == 'Á' || r == 'É' || r == 'Í' || r == 'Ó' || r == 'Ú' || r == 'Ü' {
			spanishCharCount++
		}
	}

	if len(text) == 0 {
		return 0.5
	}

	ratio := float64(spanishCharCount) / float64(len([]rune(text)))

	if ratio > 0.7 {
		return 1.0
	} else if ratio > 0.3 {
		return 0.7
	}
	return 0.3
}

func (a *SpanishAnalyzer) Validate(text string) bool {
	if len(text) == 0 {
		return false
	}

	hasValidChar := false
	for _, r := range text {
		if (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || r == 'ñ' || r == 'Ñ' || r == 'á' || r == 'é' || r == 'í' || r == 'ó' || r == 'ú' || r == 'ü' {
			hasValidChar = true
		}
	}

	return hasValidChar
}
