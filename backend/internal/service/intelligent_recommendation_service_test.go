package service

import (
	"testing"
)

func TestNewIntelligentRecommendationService(t *testing.T) {
	service := NewIntelligentRecommendationService()
	if service == nil {
		t.Error("NewIntelligentRecommendationService 返回了 nil")
	}
}

func TestGetRecommendation_Success(t *testing.T) {
	service := NewIntelligentRecommendationService()
	
	rec := service.GetRecommendation("user-123")
	
	if rec == nil {
		t.Error("GetRecommendation 返回了 nil")
	}
	if rec.UserID != "user-123" {
		t.Errorf("期望用户ID为 user-123, 实际得到 %s", rec.UserID)
	}
}

func TestGetRecommendation_EmptyUserID(t *testing.T) {
	service := NewIntelligentRecommendationService()
	
	rec := service.GetRecommendation("")
	
	if rec == nil {
		t.Error("GetRecommendation 应该处理空用户ID")
	}
}

func TestGetRecommendation_Concurrent(t *testing.T) {
	service := NewIntelligentRecommendationService()
	done := make(chan bool, 10)
	
	for i := 0; i < 10; i++ {
		go func(userID string) {
			rec := service.GetRecommendation(userID)
			if rec == nil {
				t.Errorf("并发获取推荐失败")
			}
			done <- true
		}("user-" + string(rune('0'+i)))
	}
	
	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestUpdateRecommendation_Success(t *testing.T) {
	service := NewIntelligentRecommendationService()
	
	rec := &RecommendationResult{
		UserID:        "user-123",
		Difficulty:    "medium",
		CaptchaType:   "slider",
		Confidence:    0.85,
	}
	
	err := service.UpdateRecommendation(rec)
	if err != nil {
		t.Errorf("UpdateRecommendation 失败: %v", err)
	}
}

func TestGetCaptchaTypeDistribution(t *testing.T) {
	service := NewIntelligentRecommendationService()
	
	distribution := service.GetCaptchaTypeDistribution("user-123")
	
	if distribution == nil {
		t.Error("GetCaptchaTypeDistribution 返回了 nil")
	}
}

func TestRecommendationResult_Fields(t *testing.T) {
	result := &RecommendationResult{
		UserID:        "user-123",
		Difficulty:    "medium",
		CaptchaType:   "slider",
		Confidence:    0.85,
		Factors:       []string{"history", "device"},
	}
	
	if result.UserID != "user-123" {
		t.Errorf("期望用户ID为 user-123, 实际得到 %s", result.UserID)
	}
	if result.Difficulty != "medium" {
		t.Errorf("期望难度为 medium, 实际得到 %s", result.Difficulty)
	}
	if result.CaptchaType != "slider" {
		t.Errorf("期望验证码类型为 slider, 实际得到 %s", result.CaptchaType)
	}
	if result.Confidence != 0.85 {
		t.Errorf("期望置信度为 0.85, 实际得到 %f", result.Confidence)
	}
}
