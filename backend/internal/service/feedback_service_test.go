package service

import (
	"context"
	"testing"
	"time"

	"github.com/hjtpx/hjtpx/internal/model"
)

func TestNewFeedbackService(t *testing.T) {
	service := NewFeedbackService()
	if service == nil {
		t.Error("NewFeedbackService 返回了 nil")
	}
}

func TestSubmitVerificationFeedback(t *testing.T) {
	service := NewFeedbackService()
	ctx := context.Background()

	testCases := []struct {
		name    string
		input   *model.VerificationFeedbackInput
		wantErr bool
	}{
		{
			name: "有效反馈-验证类型",
			input: &model.VerificationFeedbackInput{
				SessionID:    "session-123",
				FeedbackType: model.FeedbackTypeVerification,
				Category:     "slider",
				Content:      "验证体验很好",
				Severity:     model.FeedbackSeverityLow,
				Rating:       5,
				Success:      true,
			},
			wantErr: false,
		},
		{
			name: "有效反馈-错误类型",
			input: &model.VerificationFeedbackInput{
				SessionID:    "session-456",
				FeedbackType: model.FeedbackTypeError,
				Category:     "timeout",
				Content:      "验证超时",
				Severity:     model.FeedbackSeverityMedium,
				Rating:       2,
				Success:      false,
			},
			wantErr: false,
		},
		{
			name: "有效反馈-UX类型",
			input: &model.VerificationFeedbackInput{
				SessionID:    "session-789",
				FeedbackType: model.FeedbackTypeUX,
				Category:     "usability",
				Content:      "界面可以更友好",
				Severity:     model.FeedbackSeverityLow,
				Rating:       3,
				Success:      true,
			},
			wantErr: false,
		},
		{
			name: "无效评分-超过最大值",
			input: &model.VerificationFeedbackInput{
				SessionID:    "session-invalid",
				FeedbackType: model.FeedbackTypeVerification,
				Content:      "测试反馈",
				Rating:       6,
			},
			wantErr: true,
		},
		{
			name: "无效评分-负数",
			input: &model.VerificationFeedbackInput{
				SessionID:    "session-invalid",
				FeedbackType: model.FeedbackTypeVerification,
				Content:      "测试反馈",
				Rating:       -1,
			},
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := service.SubmitVerificationFeedback(ctx, 1, 1, tc.input)
			if (err != nil) != tc.wantErr {
				t.Errorf("SubmitVerificationFeedback() error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}

func TestGetVerificationResult(t *testing.T) {
	service := NewFeedbackService()

	t.Run("成功验证结果", func(t *testing.T) {
		result := service.GetVerificationResult(context.Background(), true, "", "session-123")

		if !result.Success {
			t.Error("验证成功时 Success 应该为 true")
		}
		if result.Message == "" {
			t.Error("成功消息不应为空")
		}
		if result.NextStep != "continue" {
			t.Errorf("NextStep 应为 'continue', 实际为 %s", result.NextStep)
		}
		if !result.RetryAllowed {
			t.Error("成功时应该允许重试")
		}
	})

	t.Run("失败验证结果-session_expired", func(t *testing.T) {
		result := service.GetVerificationResult(context.Background(), false, "session_expired", "session-123")

		if result.Success {
			t.Error("验证失败时 Success 应该为 false")
		}
		if result.ErrorCode != "session_expired" {
			t.Errorf("ErrorCode 应为 'session_expired', 实际为 %s", result.ErrorCode)
		}
		if result.Message == "" {
			t.Error("错误消息不应为空")
		}
		if result.ErrorContext == nil {
			t.Error("ErrorContext 不应为 nil")
		}
		if len(result.Suggestions) == 0 {
			t.Error("应该提供建议")
		}
		if result.NextStep != "refresh" {
			t.Errorf("NextStep 应为 'refresh', 实际为 %s", result.NextStep)
		}
	})

	t.Run("失败验证结果-invalid_position", func(t *testing.T) {
		result := service.GetVerificationResult(context.Background(), false, "invalid_position", "session-123")

		if result.ErrorCode != "invalid_position" {
			t.Errorf("ErrorCode 应为 'invalid_position', 实际为 %s", result.ErrorCode)
		}
		if result.NextStep != "retry" {
			t.Errorf("NextStep 应为 'retry', 实际为 %s", result.NextStep)
		}
		if len(result.Suggestions) == 0 {
			t.Error("应该提供建议")
		}
	})

	t.Run("失败验证结果-timeout", func(t *testing.T) {
		result := service.GetVerificationResult(context.Background(), false, "timeout", "session-123")

		if result.ErrorCode != "timeout" {
			t.Errorf("ErrorCode 应为 'timeout', 实际为 %s", result.ErrorCode)
		}
		if result.NextStep != "retry" {
			t.Errorf("NextStep 应为 'retry', 实际为 %s", result.NextStep)
		}
	})

	t.Run("失败验证结果-risk_detected", func(t *testing.T) {
		result := service.GetVerificationResult(context.Background(), false, "risk_detected", "session-123")

		if result.ErrorCode != "risk_detected" {
			t.Errorf("ErrorCode 应为 'risk_detected', 实际为 %s", result.ErrorCode)
		}
		if result.NextStep != "wait" {
			t.Errorf("NextStep 应为 'wait', 实际为 %s", result.NextStep)
		}
	})

	t.Run("失败验证结果-attempts_exceeded", func(t *testing.T) {
		result := service.GetVerificationResult(context.Background(), false, "attempts_exceeded", "session-123")

		if result.ErrorCode != "attempts_exceeded" {
			t.Errorf("ErrorCode 应为 'attempts_exceeded', 实际为 %s", result.ErrorCode)
		}
		if result.NextStep != "contact_support" {
			t.Errorf("NextStep 应为 'contact_support', 实际为 %s", result.NextStep)
		}
	})

	t.Run("未知错误代码", func(t *testing.T) {
		result := service.GetVerificationResult(context.Background(), false, "unknown_error", "session-123")

		if result.ErrorCode != "unknown_error" {
			t.Errorf("ErrorCode 应为 'unknown_error', 实际为 %s", result.ErrorCode)
		}
		if result.NextStep != "retry" {
			t.Errorf("NextStep 应为 'retry', 实际为 %s", result.NextStep)
		}
	})
}

func TestOptimizeErrorMessage(t *testing.T) {
	service := NewFeedbackService()
	ctx := context.Background()

	t.Run("优化错误消息-默认语言", func(t *testing.T) {
		message, suggestions, err := service.OptimizeErrorMessage(ctx, "session_expired", nil)

		if err != nil {
			t.Errorf("OptimizeErrorMessage() error = %v", err)
		}
		if message == "" {
			t.Error("消息不应为空")
		}
		if len(suggestions) == 0 {
			t.Error("应该提供建议")
		}
		if len(suggestions) > 3 {
			t.Errorf("建议数量应不超过 3 个, 实际为 %d", len(suggestions))
		}
	})

	t.Run("优化错误消息-英文", func(t *testing.T) {
		userContext := map[string]interface{}{
			"language": "en-US",
		}
		message, suggestions, err := service.OptimizeErrorMessage(ctx, "session_expired", userContext)

		if err != nil {
			t.Errorf("OptimizeErrorMessage() error = %v", err)
		}
		if message == "" {
			t.Error("消息不应为空")
		}
		if len(suggestions) == 0 {
			t.Error("应该提供建议")
		}
	})

	t.Run("优化错误消息-invalid_position", func(t *testing.T) {
		message, suggestions, err := service.OptimizeErrorMessage(ctx, "invalid_position", nil)

		if err != nil {
			t.Errorf("OptimizeErrorMessage() error = %v", err)
		}
		if message == "" {
			t.Error("消息不应为空")
		}
		if len(suggestions) < 2 {
			t.Errorf("应该提供多个建议, 实际为 %d", len(suggestions))
		}
	})
}

func TestLocalizeMessage(t *testing.T) {
	service := NewFeedbackService()

	t.Run("中文消息", func(t *testing.T) {
		message := service.localizeMessage("session_expired", "zh-CN")
		if message == "" {
			t.Error("消息不应为空")
		}
	})

	t.Run("英文消息", func(t *testing.T) {
		message := service.localizeMessage("session_expired", "en-US")
		if message == "" {
			t.Error("消息不应为空")
		}
		if message == service.localizeMessage("session_expired", "zh-CN") {
			t.Error("中英文消息应该不同")
		}
	})
}

func TestGenerateImprovementSuggestions(t *testing.T) {
	service := NewFeedbackService()

	t.Run("低成功率建议", func(t *testing.T) {
		suggestions := service.generateImprovementSuggestions(0.5, 3.0)
		if len(suggestions) == 0 {
			t.Error("应该提供改进建议")
		}
	})

	t.Run("低评分建议", func(t *testing.T) {
		suggestions := service.generateImprovementSuggestions(0.9, 2.0)
		if len(suggestions) == 0 {
			t.Error("应该提供改进建议")
		}
	})

	t.Run("优秀体验建议", func(t *testing.T) {
		suggestions := service.generateImprovementSuggestions(0.95, 4.5)
		if len(suggestions) == 0 {
			t.Error("优秀体验也应该提供建议")
		}
	})
}

func TestCalculateUserSatisfaction(t *testing.T) {
	service := NewFeedbackService()
	ctx := context.Background()

	t.Run("计算用户满意度-无反馈", func(t *testing.T) {
		feedback, err := service.CalculateUserSatisfaction(ctx, 9999)
		if err != nil {
			t.Errorf("CalculateUserSatisfaction() error = %v", err)
		}
		if feedback == nil {
			t.Error("反馈不应为 nil")
		}
		if feedback.SatisfactionScore != 0 {
			t.Errorf("满意度应为 0, 实际为 %f", feedback.SatisfactionScore)
		}
	})
}

func TestValidateFeedbackInput(t *testing.T) {
	service := NewFeedbackService()

	t.Run("有效输入", func(t *testing.T) {
		input := &model.VerificationFeedbackInput{
			SessionID:    "session-123",
			FeedbackType: model.FeedbackTypeVerification,
			Content:      "测试反馈",
			Severity:     model.FeedbackSeverityLow,
			Rating:       5,
		}
		err := service.validateFeedbackInput(input)
		if err != nil {
			t.Errorf("validateFeedbackInput() error = %v", err)
		}
	})

	t.Run("无效反馈类型", func(t *testing.T) {
		input := &model.VerificationFeedbackInput{
			SessionID:    "session-123",
			FeedbackType: "invalid_type",
			Content:      "测试反馈",
		}
		err := service.validateFeedbackInput(input)
		if err != ErrInvalidFeedbackType {
			t.Errorf("validateFeedbackInput() 应返回 ErrInvalidFeedbackType, 实际为 %v", err)
		}
	})

	t.Run("无效严重程度", func(t *testing.T) {
		input := &model.VerificationFeedbackInput{
			SessionID:    "session-123",
			FeedbackType: model.FeedbackTypeVerification,
			Content:      "测试反馈",
			Severity:     "invalid_severity",
		}
		err := service.validateFeedbackInput(input)
		if err != ErrInvalidSeverity {
			t.Errorf("validateFeedbackInput() 应返回 ErrInvalidSeverity, 实际为 %v", err)
		}
	})

	t.Run("无效评分-超过最大值", func(t *testing.T) {
		input := &model.VerificationFeedbackInput{
			SessionID:    "session-123",
			FeedbackType: model.FeedbackTypeVerification,
			Content:      "测试反馈",
			Rating:       6,
		}
		err := service.validateFeedbackInput(input)
		if err != ErrInvalidRating {
			t.Errorf("validateFeedbackInput() 应返回 ErrInvalidRating, 实际为 %v", err)
		}
	})

	t.Run("无效评分-负数", func(t *testing.T) {
		input := &model.VerificationFeedbackInput{
			SessionID:    "session-123",
			FeedbackType: model.FeedbackTypeVerification,
			Content:      "测试反馈",
			Rating:       -1,
		}
		err := service.validateFeedbackInput(input)
		if err != ErrInvalidRating {
			t.Errorf("validateFeedbackInput() 应返回 ErrInvalidRating, 实际为 %v", err)
		}
	})
}

func TestGenerateSlug(t *testing.T) {
	service := NewFeedbackService()

	t.Run("生成slug-基本标题", func(t *testing.T) {
		slug := service.generateSlug("测试标题")
		if slug == "" {
			t.Error("slug 不应为空")
		}
		if slug == "测试标题" {
			t.Error("slug 应该是小写且无空格")
		}
	})

	t.Run("生成slug-带特殊字符", func(t *testing.T) {
		slug := service.generateSlug("测试: 标题/帮助")
		if slug == "" {
			t.Error("slug 不应为空")
		}
	})
}

func TestGetUserPreferences(t *testing.T) {
	service := NewFeedbackService()
	ctx := context.Background()

	t.Run("获取用户偏好-新用户", func(t *testing.T) {
		prefs, err := service.GetUserPreferences(ctx, 99999)
		if err != nil {
			t.Errorf("GetUserPreferences() error = %v", err)
		}
		if prefs == nil {
			t.Error("偏好不应为 nil")
		}
		if !prefs.EnableEmail {
			t.Error("新用户默认应启用邮件通知")
		}
	})
}

func TestGetErrorContext(t *testing.T) {
	service := NewFeedbackService()

	t.Run("session_expired错误上下文", func(t *testing.T) {
		ctx := service.getErrorContext("session_expired")
		if ctx == nil {
			t.Error("错误上下文不应为 nil")
		}
		if ctx.ErrorCode != "session_expired" {
			t.Errorf("ErrorCode 应为 'session_expired', 实际为 %s", ctx.ErrorCode)
		}
		if len(ctx.Suggestions) == 0 {
			t.Error("应该提供建议")
		}
		if len(ctx.HelpLinks) == 0 {
			t.Error("应该提供帮助链接")
		}
	})

	t.Run("invalid_position错误上下文", func(t *testing.T) {
		ctx := service.getErrorContext("invalid_position")
		if ctx == nil {
			t.Error("错误上下文不应为 nil")
		}
		if ctx.ErrorCode != "invalid_position" {
			t.Errorf("ErrorCode 应为 'invalid_position', 实际为 %s", ctx.ErrorCode)
		}
	})
}

func TestGetSuggestionsForError(t *testing.T) {
	service := NewFeedbackService()

	t.Run("session_expired建议", func(t *testing.T) {
		suggestions := service.getSuggestionsForError("session_expired")
		if len(suggestions) == 0 {
			t.Error("应该提供建议")
		}
		if len(suggestions) < 2 {
			t.Error("应该提供多个建议")
		}
	})

	t.Run("timeout建议", func(t *testing.T) {
		suggestions := service.getSuggestionsForError("timeout")
		if len(suggestions) == 0 {
			t.Error("应该提供建议")
		}
	})

	t.Run("未知错误建议", func(t *testing.T) {
		suggestions := service.getSuggestionsForError("unknown")
		if len(suggestions) == 0 {
			t.Error("应该提供默认建议")
		}
	})
}

func TestGetNextStep(t *testing.T) {
	service := NewFeedbackService()

	testCases := []struct {
		errorCode string
		expected  string
	}{
		{"session_expired", "refresh"},
		{"invalid_position", "retry"},
		{"timeout", "retry"},
		{"risk_detected", "wait"},
		{"server_error", "wait"},
		{"attempts_exceeded", "contact_support"},
		{"unknown", "retry"},
	}

	for _, tc := range testCases {
		t.Run(tc.errorCode, func(t *testing.T) {
			step := service.getNextStep(tc.errorCode)
			if step != tc.expected {
				t.Errorf("getNextStep(%s) = %s, expected %s", tc.errorCode, step, tc.expected)
			}
		})
	}
}

func TestVerificationFeedbackModel(t *testing.T) {
	t.Run("设置元数据", func(t *testing.T) {
		feedback := &model.VerificationFeedback{}
		data := map[string]interface{}{
			"key1": "value1",
			"key2": 123,
		}
		err := feedback.SetMetadata(data)
		if err != nil {
			t.Errorf("SetMetadata() error = %v", err)
		}

		retrieved, err := feedback.GetMetadata()
		if err != nil {
			t.Errorf("GetMetadata() error = %v", err)
		}
		if retrieved["key1"] != "value1" {
			t.Error("元数据值不匹配")
		}
	})

	t.Run("标记为已审核", func(t *testing.T) {
		feedback := &model.VerificationFeedback{}
		err := feedback.MarkAsReviewed(1)
		if err != nil {
			t.Errorf("MarkAsReviewed() error = %v", err)
		}
		if feedback.Status != model.FeedbackStatusReviewed {
			t.Errorf("Status 应为 %s, 实际为 %s", model.FeedbackStatusReviewed, feedback.Status)
		}
		if feedback.ReviewedBy != 1 {
			t.Errorf("ReviewedBy 应为 1, 实际为 %d", feedback.ReviewedBy)
		}
		if feedback.ReviewedAt == nil {
			t.Error("ReviewedAt 不应为 nil")
		}
	})

	t.Run("标记为已解决", func(t *testing.T) {
		feedback := &model.VerificationFeedback{}
		err := feedback.MarkAsResolved()
		if err != nil {
			t.Errorf("MarkAsResolved() error = %v", err)
		}
		if feedback.Status != model.FeedbackStatusResolved {
			t.Errorf("Status 应为 %s, 实际为 %s", model.FeedbackStatusResolved, feedback.Status)
		}
		if feedback.ResolvedAt == nil {
			t.Error("ResolvedAt 不应为 nil")
		}
	})

	t.Run("IsResolved检查", func(t *testing.T) {
		feedback := &model.VerificationFeedback{
			Status: model.FeedbackStatusResolved,
		}
		if !feedback.IsResolved() {
			t.Error("IsResolved() 应返回 true")
		}

		feedback.Status = model.FeedbackStatusPending
		if feedback.IsResolved() {
			t.Error("IsResolved() 应返回 false")
		}
	})

	t.Run("IsPending检查", func(t *testing.T) {
		feedback := &model.VerificationFeedback{
			Status: model.FeedbackStatusPending,
		}
		if !feedback.IsPending() {
			t.Error("IsPending() 应返回 true")
		}

		feedback.Status = model.FeedbackStatusResolved
		if feedback.IsPending() {
			t.Error("IsPending() 应返回 false")
		}
	})
}

func TestHelpDocumentModel(t *testing.T) {
	t.Run("设置标签列表", func(t *testing.T) {
		doc := &model.HelpDocument{}
		tags := []string{"tag1", "tag2", "tag3"}
		err := doc.SetTagsList(tags)
		if err != nil {
			t.Errorf("SetTagsList() error = %v", err)
		}

		retrieved, err := doc.GetTagsList()
		if err != nil {
			t.Errorf("GetTagsList() error = %v", err)
		}
		if len(retrieved) != len(tags) {
			t.Errorf("标签数量应为 %d, 实际为 %d", len(tags), len(retrieved))
		}
	})

	t.Run("设置相关文档", func(t *testing.T) {
		doc := &model.HelpDocument{}
		ids := []uint{1, 2, 3}
		err := doc.SetRelatedDocs(ids)
		if err != nil {
			t.Errorf("SetRelatedDocs() error = %v", err)
		}

		retrieved, err := doc.GetRelatedDocs()
		if err != nil {
			t.Errorf("GetRelatedDocs() error = %v", err)
		}
		if len(retrieved) != len(ids) {
			t.Errorf("文档ID数量应为 %d, 实际为 %d", len(ids), len(retrieved))
		}
	})

	t.Run("增加查看次数", func(t *testing.T) {
		doc := &model.HelpDocument{ViewCount: 5}
		doc.IncrementViewCount()
		if doc.ViewCount != 6 {
			t.Errorf("ViewCount 应为 6, 实际为 %d", doc.ViewCount)
		}
	})
}

func TestUXMetricsModel(t *testing.T) {
	t.Run("计算持续时间", func(t *testing.T) {
		now := time.Now().UnixMilli()
		metrics := &model.UXMetrics{
			StartTime: now - 1000,
			EndTime:   now,
		}
		duration := metrics.CalculateDuration()
		if duration < 999 || duration > 1001 {
			t.Errorf("Duration 应约为 1000, 实际为 %d", duration)
		}
	})

	t.Run("检查良好体验", func(t *testing.T) {
		metrics := &model.UXMetrics{
			SuccessRate: 0.85,
			RetryCount:  1,
		}
		if !metrics.IsGoodExperience() {
			t.Error("IsGoodExperience() 应返回 true")
		}

		metrics.SuccessRate = 0.5
		metrics.RetryCount = 5
		if metrics.IsGoodExperience() {
			t.Error("IsGoodExperience() 应返回 false")
		}
	})
}

func TestErrorContextModel(t *testing.T) {
	t.Run("创建错误上下文", func(t *testing.T) {
		ctx := model.NewErrorContext("test_error", "测试错误", "详细信息")
		if ctx.ErrorCode != "test_error" {
			t.Errorf("ErrorCode 应为 'test_error', 实际为 %s", ctx.ErrorCode)
		}
		if ctx.ErrorMessage != "测试错误" {
			t.Errorf("ErrorMessage 应为 '测试错误', 实际为 %s", ctx.ErrorMessage)
		}
	})

	t.Run("添加建议", func(t *testing.T) {
		ctx := model.NewErrorContext("test", "测试", "")
		ctx.AddSuggestion("建议1")
		ctx.AddSuggestion("建议2")
		if len(ctx.Suggestions) != 2 {
			t.Errorf("建议数量应为 2, 实际为 %d", len(ctx.Suggestions))
		}
	})

	t.Run("添加帮助链接", func(t *testing.T) {
		ctx := model.NewErrorContext("test", "测试", "")
		ctx.AddHelpLink("/help/test")
		if len(ctx.HelpLinks) != 1 {
			t.Errorf("帮助链接数量应为 1, 实际为 %d", len(ctx.HelpLinks))
		}
	})

	t.Run("添加上下文数据", func(t *testing.T) {
		ctx := model.NewErrorContext("test", "测试", "")
		ctx.AddContext("key", "value")
		if ctx.ContextData["key"] != "value" {
			t.Error("上下文数据值不匹配")
		}
	})
}

func TestUserFeedbackPreferencesModel(t *testing.T) {
	t.Run("ShouldSendEmail", func(t *testing.T) {
		prefs := &model.UserFeedbackPreferences{EnableEmail: true}
		if !prefs.ShouldSendEmail() {
			t.Error("ShouldSendEmail() 应返回 true")
		}

		prefs.EnableEmail = false
		if prefs.ShouldSendEmail() {
			t.Error("ShouldSendEmail() 应返回 false")
		}
	})

	t.Run("ShouldSendPush", func(t *testing.T) {
		prefs := &model.UserFeedbackPreferences{EnablePush: true}
		if !prefs.ShouldSendPush() {
			t.Error("ShouldSendPush() 应返回 true")
		}

		prefs.EnablePush = false
		if prefs.ShouldSendPush() {
			t.Error("ShouldSendPush() 应返回 false")
		}
	})
}
