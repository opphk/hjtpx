package captcha

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestVideoVerifierService_checkAnswer_ExactMatch(t *testing.T) {
	verifier := NewVideoVerifierServiceSimple()

	assert.True(t, verifier.checkAnswer("举手", "举手"))
	assert.True(t, verifier.checkAnswer("挥手", "挥手"))
	assert.False(t, verifier.checkAnswer("举手", "挥手"))
}

func TestVideoVerifierService_checkAnswer_CaseInsensitive(t *testing.T) {
	verifier := NewVideoVerifierServiceSimple()

	assert.True(t, verifier.checkAnswer("举手", "举手"))
}

func TestVideoVerifierService_checkAnswer_ContainsMatch(t *testing.T) {
	verifier := NewVideoVerifierServiceSimple()

	assert.True(t, verifier.checkAnswer("举手", "举手过头"))
	assert.True(t, verifier.checkAnswer("挥手", "挥挥手"))
}

func TestVideoVerifierService_checkAnswer_AliasMatch(t *testing.T) {
	verifier := NewVideoVerifierServiceSimple()

	assert.True(t, verifier.checkAnswer("举手", "举手过头顶"))
	assert.True(t, verifier.checkAnswer("举手", "抬起手"))
	assert.True(t, verifier.checkAnswer("举手", "举手过头"))
	assert.True(t, verifier.checkAnswer("点头", "向下点头"))
	assert.True(t, verifier.checkAnswer("摇头", "向左摇头"))
	assert.True(t, verifier.checkAnswer("眨眼", "快速眨眼"))
	assert.True(t, verifier.checkAnswer("张嘴", "张开嘴巴"))
}

func TestVideoVerifierService_checkAnswer_NoMatch(t *testing.T) {
	verifier := NewVideoVerifierServiceSimple()

	assert.False(t, verifier.checkAnswer("举手", "跳舞"))
	assert.False(t, verifier.checkAnswer("挥手", "举手"))
}

func TestVideoVerifierService_analyzeBehavior_NormalBehavior(t *testing.T) {
	verifier := NewVideoVerifierServiceSimple()

	session := &VideoCaptchaSession{
		Duration: 8,
	}

	behavior := VideoBehaviorData{
		Duration:    8000,
		ViewCount:   1,
		ReplayCount: 0,
	}

	analysis := verifier.analyzeBehavior(behavior, session)
	assert.NotNil(t, analysis)
	assert.False(t, analysis.IsBot)
	assert.Less(t, analysis.RiskScore, 0.5)
}

func TestVideoVerifierService_analyzeBehavior_QuickResponse(t *testing.T) {
	verifier := NewVideoVerifierServiceSimple()

	session := &VideoCaptchaSession{
		Duration: 8,
	}

	behavior := VideoBehaviorData{
		Duration:    500,
		ViewCount:   1,
		StartTime:   0,
		AnswerTime:  200,
	}

	analysis := verifier.analyzeBehavior(behavior, session)
	assert.NotNil(t, analysis)
	assert.Contains(t, analysis.RiskIndicators, "响应时间过短")
}

func TestVideoVerifierService_analyzeBehavior_NoView(t *testing.T) {
	verifier := NewVideoVerifierServiceSimple()

	session := &VideoCaptchaSession{
		Duration: 8,
	}

	behavior := VideoBehaviorData{
		Duration:    5000,
		ViewCount:   0,
		ReplayCount: 0,
	}

	analysis := verifier.analyzeBehavior(behavior, session)
	assert.NotNil(t, analysis)
	assert.Contains(t, analysis.RiskIndicators, "未观看视频")
	assert.Greater(t, analysis.RiskScore, 0.2)
}

func TestVideoVerifierService_analyzeBehavior_FewViews(t *testing.T) {
	verifier := NewVideoVerifierServiceSimple()

	session := &VideoCaptchaSession{
		Duration: 8,
	}

	behavior := VideoBehaviorData{
		Duration:    5000,
		ViewCount:   1,
		ReplayCount: 0,
	}

	analysis := verifier.analyzeBehavior(behavior, session)
	assert.NotNil(t, analysis)
	assert.Contains(t, analysis.RiskIndicators, "视频观看次数过少")
}

func TestVideoVerifierService_analyzeBehavior_TooManyReplays(t *testing.T) {
	verifier := NewVideoVerifierServiceSimple()

	session := &VideoCaptchaSession{
		Duration: 8,
	}

	behavior := VideoBehaviorData{
		Duration:    5000,
		ViewCount:   1,
		ReplayCount: 5,
	}

	analysis := verifier.analyzeBehavior(behavior, session)
	assert.NotNil(t, analysis)
	assert.Contains(t, analysis.RiskIndicators, "视频重复播放次数过多")
}

func TestVideoVerifierService_analyzeBehavior_LongDuration(t *testing.T) {
	verifier := NewVideoVerifierServiceSimple()

	session := &VideoCaptchaSession{
		Duration: 5,
	}

	behavior := VideoBehaviorData{
		Duration:    60000,
		ViewCount:   1,
		ReplayCount: 0,
	}

	analysis := verifier.analyzeBehavior(behavior, session)
	assert.NotNil(t, analysis)
	assert.Contains(t, analysis.RiskIndicators, "响应时间过长")
}

func TestVideoVerifierService_analyzeBehavior_HighRisk(t *testing.T) {
	verifier := NewVideoVerifierServiceSimple()

	session := &VideoCaptchaSession{
		Duration: 8,
	}

	behavior := VideoBehaviorData{
		Duration:    200,
		ViewCount:   0,
		ReplayCount: 5,
		StartTime:   0,
		AnswerTime:  100,
	}

	analysis := verifier.analyzeBehavior(behavior, session)
	assert.NotNil(t, analysis)
	assert.True(t, analysis.IsBot)
	assert.Greater(t, analysis.RiskScore, 0.5)
	assert.Contains(t, analysis.RiskIndicators, "综合风险评分过高")
}

func TestVideoVerifierService_calculateScore(t *testing.T) {
	verifier := NewVideoVerifierServiceSimple()

	session := &VideoCaptchaSession{
		Difficulty: 2,
	}

	riskAnalysis := &VideoRiskAnalysis{
		RiskScore:  0.1,
		Confidence: 0.9,
	}

	score := verifier.calculateScore(riskAnalysis, session)
	assert.Greater(t, score, 0.7)
	assert.LessOrEqual(t, score, 1.0)
}

func TestVideoVerifierService_calculateScore_LowConfidence(t *testing.T) {
	verifier := NewVideoVerifierServiceSimple()

	session := &VideoCaptchaSession{
		Difficulty: 2,
	}

	riskAnalysis := &VideoRiskAnalysis{
		RiskScore:  0.2,
		Confidence: 0.3,
	}

	score := verifier.calculateScore(riskAnalysis, session)
	assert.Less(t, score, 0.8)
}

func TestVideoVerifierService_calculateScore_HighDifficulty(t *testing.T) {
	verifier := NewVideoVerifierServiceSimple()

	session := &VideoCaptchaSession{
		Difficulty: 3,
	}

	riskAnalysis := &VideoRiskAnalysis{
		RiskScore:  0.0,
		Confidence: 1.0,
	}

	score := verifier.calculateScore(riskAnalysis, session)
	assert.Equal(t, 0.9, score)
}

func TestVideoVerifierService_calculateScore_NilRiskAnalysis(t *testing.T) {
	verifier := NewVideoVerifierServiceSimple()

	session := &VideoCaptchaSession{
		Difficulty: 2,
	}

	score := verifier.calculateScore(nil, session)
	assert.Equal(t, 0.85, score)
}

func TestVideoVerifierService_calculateScore_HighRisk(t *testing.T) {
	verifier := NewVideoVerifierServiceSimple()

	session := &VideoCaptchaSession{
		Difficulty: 2,
	}

	riskAnalysis := &VideoRiskAnalysis{
		RiskScore:  0.9,
		Confidence: 0.1,
	}

	score := verifier.calculateScore(riskAnalysis, session)
	assert.Less(t, score, 0.5)
}

func TestVideoVerifierService_CheckSessionValid_NotFound(t *testing.T) {
	verifier := NewVideoVerifierServiceSimple()

	valid, message := verifier.CheckSessionValid(nil, "nonexistent")
	assert.False(t, valid)
	assert.Equal(t, "会话不存在", message)
}
