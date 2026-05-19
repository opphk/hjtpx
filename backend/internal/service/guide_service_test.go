package service

import (
	"testing"
)

func TestGuideServiceGetGuide(t *testing.T) {
	service := NewGuideService()

	tests := []struct {
		name    string
		guideID string
		wantErr bool
	}{
		{
			name:    "Get onboarding guide",
			guideID: "onboarding",
			wantErr: false,
		},
		{
			name:    "Get slider guide",
			guideID: "slider_guide",
			wantErr: false,
		},
		{
			name:    "Get click guide",
			guideID: "click_guide",
			wantErr: false,
		},
		{
			name:    "Get error recovery guide",
			guideID: "error_recovery",
			wantErr: false,
		},
		{
			name:    "Get non-existent guide",
			guideID: "nonexistent",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			steps, err := service.GetGuide(tt.guideID)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetGuide() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && len(steps) == 0 {
				t.Errorf("GetGuide() returned empty steps for %s", tt.guideID)
			}
		})
	}
}

func TestGuideServiceGetAllGuides(t *testing.T) {
	service := NewGuideService()

	guides := service.GetAllGuides()
	if len(guides) < 4 {
		t.Errorf("GetAllGuides() returned %d guides, want at least 4", len(guides))
	}
}

func TestGuideServiceStartSession(t *testing.T) {
	service := NewGuideService()

	session, err := service.StartSession("user123", "onboarding", &GuideContext{
		UserID:     "user123",
		Device:     "desktop",
		Browser:    "chrome",
		Language:   "zh-CN",
	})
	if err != nil {
		t.Errorf("StartSession() error = %v", err)
		return
	}

	if session == nil {
		t.Error("StartSession() returned nil session")
	}

	if session.UserID != "user123" {
		t.Errorf("StartSession() UserID = %s, want user123", session.UserID)
	}

	if session.GuideID != "onboarding" {
		t.Errorf("StartSession() GuideID = %s, want onboarding", session.GuideID)
	}

	if session.CurrentStep != 0 {
		t.Errorf("StartSession() CurrentStep = %d, want 0", session.CurrentStep)
	}
}

func TestGuideServiceCompleteStep(t *testing.T) {
	service := NewGuideService()

	session, _ := service.StartSession("user123", "onboarding", nil)

	err := service.CompleteStep(session.ID, 0)
	if err != nil {
		t.Errorf("CompleteStep() error = %v", err)
		return
	}

	updated, _ := service.GetSession(session.ID)
	if updated.CompletedSteps[0] != 0 {
		t.Errorf("CompleteStep() did not record completed step")
	}
}

func TestGuideServiceSkipStep(t *testing.T) {
	service := NewGuideService()

	session, _ := service.StartSession("user123", "onboarding", nil)

	err := service.SkipStep(session.ID, 0, "test_skip")
	if err != nil {
		t.Errorf("SkipStep() error = %v", err)
		return
	}

	updated, _ := service.GetSession(session.ID)
	if len(updated.SkippedSteps) != 1 || updated.SkippedSteps[0] != 0 {
		t.Errorf("SkipStep() did not record skipped step")
	}
}

func TestGuideServiceCompleteSession(t *testing.T) {
	service := NewGuideService()

	session, _ := service.StartSession("user123", "onboarding", nil)

	err := service.CompleteSession(session.ID)
	if err != nil {
		t.Errorf("CompleteSession() error = %v", err)
		return
	}

	updated, _ := service.GetSession(session.ID)
	if updated.CompletedAt == nil {
		t.Error("CompleteSession() did not set CompletedAt")
	}
}

func TestGuideServiceGetNextStep(t *testing.T) {
	service := NewGuideService()

	session, _ := service.StartSession("user123", "onboarding", nil)

	step, err := service.GetNextStep(session.ID)
	if err != nil {
		t.Errorf("GetNextStep() error = %v", err)
		return
	}

	if step == nil {
		t.Error("GetNextStep() returned nil step")
	}
}

func TestGuideServiceGetPersonalizedGuide(t *testing.T) {
	service := NewGuideService()

	tests := []struct {
		name    string
		context *GuideContext
		wantErr bool
	}{
		{
			name: "New user",
			context: &GuideContext{
				UserID:        "user1",
				TotalAttempts: 0,
			},
			wantErr: false,
		},
		{
			name: "High failure rate slider user",
			context: &GuideContext{
				UserID:         "user2",
				TotalAttempts:  10,
				FailedAttempts: 7,
				VerificationType: "slider",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			guide, err := service.GetPersonalizedGuide(tt.context)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetPersonalizedGuide() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && guide == "" {
				t.Error("GetPersonalizedGuide() returned empty guide")
			}
		})
	}
}

func TestGuideServiceSaveAndRestoreProgress(t *testing.T) {
	service := NewGuideService()

	session, _ := service.StartSession("user123", "onboarding", nil)
	service.CompleteStep(session.ID, 0)

	data, err := service.SaveProgress(session.ID)
	if err != nil {
		t.Errorf("SaveProgress() error = %v", err)
		return
	}

	session2, _ := service.StartSession("user456", "onboarding", nil)
	err = service.RestoreProgress(session2.ID, data)
	if err != nil {
		t.Errorf("RestoreProgress() error = %v", err)
	}
}

func TestGuideServiceExportAnalytics(t *testing.T) {
	service := NewGuideService()

	_, _ = service.StartSession("user1", "onboarding", nil)
	_, _ = service.StartSession("user2", "slider_guide", nil)

	analytics, err := service.ExportAnalytics()
	if err != nil {
		t.Errorf("ExportAnalytics() error = %v", err)
		return
	}

	if len(analytics) < 2 {
		t.Errorf("ExportAnalytics() returned %d analytics, want at least 2", len(analytics))
	}
}
