package captcha

import (
	"context"
	"time"
)

type EmojiVerifierService struct {
	generatorService *EmojiGeneratorService
}

type VerifyEmojiCaptchaRequest struct {
	SessionID    string   `json:"session_id"`
	SelectedEmojis []string `json:"selected_emojis"`
	// 行为分析数据
	BehaviorData EmojiBehaviorData `json:"behavior_data"`
}

type VerifyEmojiCaptchaResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

type EmojiBehaviorData struct {
	ClickTimes    []int64  `json:"click_times"`    // 每个点击的时间戳
	ClickIntervals []int64 `json:"click_intervals"` // 点击间隔（毫秒）
	TotalTime     int64    `json:"total_time"`     // 总耗时（毫秒）
	IsMobile      bool     `json:"is_mobile"`      // 是否为移动设备
}

func NewEmojiVerifierService(generatorService *EmojiGeneratorService) *EmojiVerifierService {
	return &EmojiVerifierService{
		generatorService: generatorService,
	}
}

func NewEmojiVerifierServiceSimple() *EmojiVerifierService {
	return &EmojiVerifierService{
		generatorService: NewEmojiGeneratorServiceSimple(),
	}
}

func (s *EmojiVerifierService) Verify(ctx context.Context, req *VerifyEmojiCaptchaRequest) (*VerifyEmojiCaptchaResponse, error) {
	// 获取会话
	session, err := s.generatorService.GetSession(ctx, req.SessionID)
	if err != nil {
		return &VerifyEmojiCaptchaResponse{
			Success: false,
			Message: "会话不存在或已过期",
		}, nil
	}

	// 检查会话状态
	if time.Now().After(session.ExpiredAt) {
		return &VerifyEmojiCaptchaResponse{
			Success: false,
			Message: "会话已过期",
		}, nil
	}

	// 检查验证次数
	if session.VerifyCount >= session.MaxAttempts {
		return &VerifyEmojiCaptchaResponse{
			Success: false,
			Message: "验证次数已用完",
		}, nil
	}

	// 更新验证次数
	session.VerifyCount++
	s.generatorService.UpdateSession(ctx, session)

	// 检查选中表情是否正确
	if len(req.SelectedEmojis) != len(session.TargetEmojis) {
		return &VerifyEmojiCaptchaResponse{
			Success: false,
			Message: "选择的表情数量不正确",
		}, nil
	}

	// 验证表情序列
	isCorrect := true
	for i, emoji := range session.TargetEmojis {
		if i >= len(req.SelectedEmojis) || req.SelectedEmojis[i] != emoji {
			isCorrect = false
			break
		}
	}

	if !isCorrect {
		return &VerifyEmojiCaptchaResponse{
			Success: false,
			Message: "表情序列不正确",
		}, nil
	}

	// 分析行为数据（简单分析，实际项目可以集成更复杂的行为分析）
	_ = s.analyzeBehavior(&req.BehaviorData)

	// 验证成功
	session.Status = "verified"
	s.generatorService.UpdateSession(ctx, session)

	return &VerifyEmojiCaptchaResponse{
		Success: true,
		Message: "验证成功",
	}, nil
}

func (s *EmojiVerifierService) analyzeBehavior(data *EmojiBehaviorData) map[string]interface{} {
	result := make(map[string]interface{})

	// 简单分析，实际项目可以集成机器学习模型
	if data.TotalTime > 0 {
		// 总耗时分析
		result["total_time"] = data.TotalTime
	}

	// 点击间隔分析
	if len(data.ClickIntervals) > 0 {
		avgInterval := int64(0)
		for _, interval := range data.ClickIntervals {
			avgInterval += interval
		}
		avgInterval /= int64(len(data.ClickIntervals))
		result["avg_click_interval"] = avgInterval
	}

	return result
}
