package trace

import (
	"testing"

	"github.com/hjtpx/hjtpx/internal/model"
)

func TestUserProfileBuilder_Init(t *testing.T) {
	builder := NewUserProfileBuilder()
	if builder == nil {
		t.Fatal("Failed to create UserProfileBuilder")
	}
}

func TestUserProfileBuilder_GetOrCreateProfile(t *testing.T) {
	builder := NewUserProfileBuilder()

	profile := builder.GetOrCreateProfile("user123")
	if profile == nil {
		t.Fatal("Profile should not be nil")
	}

	if profile.UserID != "user123" {
		t.Errorf("Expected UserID 'user123', got '%s'", profile.UserID)
	}
}

func TestUserProfileBuilder_UpdateProfileWithTrace(t *testing.T) {
	builder := NewUserProfileBuilder()
	profile := builder.GetOrCreateProfile("user123")

	traceData := &model.TraceData{
		Points: []model.TracePoint{
			{X: 0, Y: 0, Timestamp: 0},
			{X: 10, Y: 10, Timestamp: 100},
			{X: 20, Y: 20, Timestamp: 200},
		},
	}

	err := builder.UpdateProfileWithTrace("user123", traceData)
	if err != nil {
		t.Fatalf("Failed to update profile: %v", err)
	}

	profile, _ = builder.GetProfile("user123")
	if len(profile.SessionHistory) != 1 {
		t.Errorf("Expected 1 session, got %d", len(profile.SessionHistory))
	}
}

func TestUserProfileBuilder_DetectAnomalousBehavior(t *testing.T) {
	builder := NewUserProfileBuilder()
	builder.GetOrCreateProfile("user123")

	for i := 0; i < 5; i++ {
		traceData := &model.TraceData{
			Points: []model.TracePoint{
				{X: 0, Y: 0, Timestamp: int64(i * 100)},
				{X: 10, Y: 10, Timestamp: int64(i*100 + 100)},
			},
		}
		err := builder.UpdateProfileWithTrace("user123", traceData)
		if err != nil {
			t.Fatalf("Failed to update profile: %v", err)
		}
	}

	anomalousTrace := &model.TraceData{
		Points: []model.TracePoint{
			{X: 0, Y: 0, Timestamp: 500},
			{X: 1000, Y: 1000, Timestamp: 550},
		},
	}

	result, err := builder.DetectAnomalousBehavior("user123", anomalousTrace)
	if err != nil {
		t.Fatalf("Failed to detect anomalous behavior: %v", err)
	}

	if result == nil {
		t.Error("Detection result should not be nil")
	}
}

func TestUserProfileBuilder_GetProfileStatistics(t *testing.T) {
	builder := NewUserProfileBuilder()
	builder.GetOrCreateProfile("user123")

	stats, err := builder.GetProfileStatistics("user123")
	if err != nil {
		t.Fatalf("Failed to get statistics: %v", err)
	}

	if stats == nil {
		t.Error("Statistics should not be nil")
	}
}

func TestUserProfileBuilder_AddDeviceFingerprint(t *testing.T) {
	builder := NewUserProfileBuilder()
	builder.GetOrCreateProfile("user123")

	builder.AddDeviceFingerprint("user123", "fingerprint123")

	profile, _ := builder.GetProfile("user123")
	if len(profile.DeviceFingerprints) != 1 {
		t.Errorf("Expected 1 device fingerprint, got %d", len(profile.DeviceFingerprints))
	}
}

func TestUserProfileBuilder_RemoveProfile(t *testing.T) {
	builder := NewUserProfileBuilder()
	builder.GetOrCreateProfile("user123")

	err := builder.RemoveProfile("user123")
	if err != nil {
		t.Fatalf("Failed to remove profile: %v", err)
	}

	_, exists := builder.GetProfile("user123")
	if exists {
		t.Error("Profile should be removed")
	}
}

func TestUserProfileBuilder_GetAllProfiles(t *testing.T) {
	builder := NewUserProfileBuilder()

	builder.GetOrCreateProfile("user1")
	builder.GetOrCreateProfile("user2")

	profiles := builder.GetAllProfiles()
	if len(profiles) != 2 {
		t.Errorf("Expected 2 profiles, got %d", len(profiles))
	}
}

func TestUserProfileBuilder_GetProfileCount(t *testing.T) {
	builder := NewUserProfileBuilder()

	builder.GetOrCreateProfile("user1")
	builder.GetOrCreateProfile("user2")

	count := builder.GetProfileCount()
	if count != 2 {
		t.Errorf("Expected 2 profiles, got %d", count)
	}
}

func TestUserProfileBuilder_ExportImportProfile(t *testing.T) {
	builder := NewUserProfileBuilder()
	builder.GetOrCreateProfile("user123")

	data, err := builder.ExportProfile("user123")
	if err != nil {
		t.Fatalf("Failed to export profile: %v", err)
	}

	if len(data) == 0 {
		t.Error("Exported data should not be empty")
	}

	newBuilder := NewUserProfileBuilder()
	err = newBuilder.ImportProfile(data)
	if err != nil {
		t.Fatalf("Failed to import profile: %v", err)
	}

	profiles := newBuilder.GetAllProfiles()
	if len(profiles) != 1 {
		t.Errorf("Expected 1 profile after import, got %d", len(profiles))
	}
}

func TestUserProfileBuilder_AddLocation(t *testing.T) {
	builder := NewUserProfileBuilder()
	builder.GetOrCreateProfile("user123")

	builder.AddLocation("user123", "192.168.1.1", "CN", "Beijing", "Beijing")

	profile, _ := builder.GetProfile("user123")
	if len(profile.LocationHistory) != 1 {
		t.Errorf("Expected 1 location, got %d", len(profile.LocationHistory))
	}
}

func TestUserProfileBuilder_MergeProfiles(t *testing.T) {
	builder := NewUserProfileBuilder()
	builder.GetOrCreateProfile("user1")
	builder.GetOrCreateProfile("user2")

	err := builder.MergeProfiles("user1", "user2")
	if err != nil {
		t.Fatalf("Failed to merge profiles: %v", err)
	}

	_, exists := builder.GetProfile("user1")
	if exists {
		t.Error("Source profile should be removed")
	}

	_, exists = builder.GetProfile("user2")
	if !exists {
		t.Error("Target profile should exist")
	}
}
