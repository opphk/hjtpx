package service

import (
	"testing"
)

func TestNewIntelligentRecommendationService(t *testing.T) {
	svc := NewIntelligentRecommendationService()
	if svc == nil {
		t.Error("NewIntelligentRecommendationService 返回了 nil")
	}
}

func TestGetRecommendation_Success(t *testing.T) {
	svc := NewIntelligentRecommendationService()

	req := &RecommendationRequest{
		UserID: "user-123",
	}

	rec := svc.GetRecommendation(req)

	if rec == nil {
		t.Error("GetRecommendation 返回了 nil")
	}
}

func TestGetRecommendation_EmptyUserID(t *testing.T) {
	svc := NewIntelligentRecommendationService()

	req := &RecommendationRequest{
		UserID: "",
	}

	rec := svc.GetRecommendation(req)

	if rec == nil {
		t.Error("GetRecommendation 应该处理空用户ID")
	}
}

func TestGetRecommendation_Concurrent(t *testing.T) {
	svc := NewIntelligentRecommendationService()
	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func(userID string) {
			req := &RecommendationRequest{UserID: userID}
			rec := svc.GetRecommendation(req)
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

func TestRecommendationResult_Fields(t *testing.T) {
	result := &RecommendationResult{
		RecommendedMethod:  "slider",
		Confidence:        0.85,
		AlternativeMethods: []AlternativeRecommendation{},
		Reasoning:         "test reasoning",
	}

	if result.RecommendedMethod != "slider" {
		t.Errorf("期望推荐方法为 slider, 实际得到 %s", result.RecommendedMethod)
	}
	if result.Confidence != 0.85 {
		t.Errorf("期望置信度为 0.85, 实际得到 %f", result.Confidence)
	}
}

func TestRecommendationRequest_Fields(t *testing.T) {
	ctx := &RecommendationContext{
		SessionID: "session-456",
	}

	req := &RecommendationRequest{
		UserID:         "user-123",
		Context:        ctx,
	}

	if req.UserID != "user-123" {
		t.Errorf("期望用户ID为 user-123, 实际得到 %s", req.UserID)
	}
	if req.Context.SessionID != "session-456" {
		t.Errorf("期望会话ID为 session-456, 实际得到 %s", req.Context.SessionID)
	}
}

func TestAlternativeRecommendation_Fields(t *testing.T) {
	alt := &AlternativeRecommendation{
		Method: "click",
		Score:  0.75,
		Reason: "user-friendly",
		Pros:   []string{"easy"},
		Cons:   []string{"slower"},
	}

	if alt.Method != "click" {
		t.Errorf("期望方法为 click, 实际得到 %s", alt.Method)
	}
	if alt.Score != 0.75 {
		t.Errorf("期望分数为 0.75, 实际得到 %f", alt.Score)
	}
}

func TestRecommendationContext_Fields(t *testing.T) {
	ctx := &RecommendationContext{
		Action:    "login",
		SessionID: "session-123",
	}

	if ctx.Action != "login" {
		t.Errorf("期望动作为 login, 实际得到 %s", ctx.Action)
	}
	if ctx.SessionID != "session-123" {
		t.Errorf("期望会话ID为 session-123, 实际得到 %s", ctx.SessionID)
	}
}
