package trace

import (
	"testing"
)

func TestDeviceFingerprintTracker_Init(t *testing.T) {
	tracker := NewDeviceFingerprintTracker()
	if tracker == nil {
		t.Fatal("Failed to create DeviceFingerprintTracker")
	}
}

func TestDeviceFingerprintTracker_RegisterFingerprint(t *testing.T) {
	tracker := NewDeviceFingerprintTracker()

	data := map[string]interface{}{
		"user_agent":        "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
		"screen_resolution": "1920x1080",
		"timezone":         "UTC+8",
		"language":         "zh-CN",
	}

	fingerprint, err := tracker.RegisterFingerprint(data)
	if err != nil {
		t.Fatalf("Failed to register fingerprint: %v", err)
	}

	if fingerprint == nil {
		t.Error("Fingerprint should not be nil")
	}

	retrieved, exists := tracker.GetFingerprint(fingerprint.FingerprintID)
	if !exists {
		t.Error("Fingerprint should exist")
	}

	if retrieved.UserAgent != "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36" {
		t.Errorf("Expected UserAgent mismatch")
	}
}

func TestDeviceFingerprintTracker_TrackDevice(t *testing.T) {
	tracker := NewDeviceFingerprintTracker()

	data := map[string]interface{}{
		"user_agent":        "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
		"screen_resolution": "1920x1080",
		"timezone":         "UTC+8",
		"language":         "zh-CN",
	}

	fingerprint, err := tracker.RegisterFingerprint(data)
	if err != nil {
		t.Fatalf("Failed to register fingerprint: %v", err)
	}

	result, err := tracker.TrackDevice(data)
	if err != nil {
		t.Fatalf("Failed to track device: %v", err)
	}

	if result == nil {
		t.Error("Tracking result should not be nil")
	}

	if result.FingerprintID != fingerprint.FingerprintID {
		t.Errorf("Expected FingerprintID '%s', got '%s'", fingerprint.FingerprintID, result.FingerprintID)
	}
}

func TestDeviceFingerprintTracker_CompareFingerprints(t *testing.T) {
	tracker := NewDeviceFingerprintTracker()

	data1 := map[string]interface{}{
		"user_agent":        "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
		"screen_resolution": "1920x1080",
		"timezone":         "UTC+8",
		"language":         "zh-CN",
	}

	data2 := map[string]interface{}{
		"user_agent":        "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
		"screen_resolution": "1920x1080",
		"timezone":         "UTC+8",
		"language":         "zh-CN",
	}

	fingerprint1, err := tracker.RegisterFingerprint(data1)
	if err != nil {
		t.Fatalf("Failed to register fingerprint1: %v", err)
	}

	fingerprint2, err := tracker.RegisterFingerprint(data2)
	if err != nil {
		t.Fatalf("Failed to register fingerprint2: %v", err)
	}

	comparison, err := tracker.CompareFingerprints(fingerprint1.FingerprintID, fingerprint2.FingerprintID)
	if err != nil {
		t.Fatalf("Failed to compare fingerprints: %v", err)
	}

	if comparison.SimilarityScore < 0 || comparison.SimilarityScore > 100 {
		t.Errorf("Similarity score should be between 0 and 100, got %f", comparison.SimilarityScore)
	}
}

func TestDeviceFingerprintTracker_FindRelatedFingerprints(t *testing.T) {
	tracker := NewDeviceFingerprintTracker()

	data1 := map[string]interface{}{
		"user_agent":        "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
		"screen_resolution": "1920x1080",
		"timezone":         "UTC+8",
		"language":         "zh-CN",
		"platform":         "windows",
	}

	data2 := map[string]interface{}{
		"user_agent":        "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
		"screen_resolution": "1920x1080",
		"timezone":         "UTC+8",
		"language":         "zh-CN",
		"platform":         "linux",
	}

	fingerprint1, err := tracker.RegisterFingerprint(data1)
	if err != nil {
		t.Fatalf("Failed to register fingerprint1: %v", err)
	}

	fingerprint2, err := tracker.RegisterFingerprint(data2)
	if err != nil {
		t.Fatalf("Failed to register fingerprint2: %v", err)
	}

	if fingerprint1.FingerprintID == fingerprint2.FingerprintID {
		t.Fatal("Fingerprints should have different IDs")
	}

	related := tracker.FindRelatedFingerprints(fingerprint1.FingerprintID, 0)
	if len(related) != 1 {
		t.Errorf("Expected 1 related device, got %d", len(related))
	}
}

func TestDeviceFingerprintTracker_DetectSuspiciousGroups(t *testing.T) {
	tracker := NewDeviceFingerprintTracker()

	data1 := map[string]interface{}{
		"user_agent":        "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
		"screen_resolution": "1920x1080",
		"timezone":         "UTC+8",
		"language":         "zh-CN",
	}

	data2 := map[string]interface{}{
		"user_agent":        "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
		"screen_resolution": "1920x1080",
		"timezone":         "UTC+8",
		"language":         "zh-CN",
	}

	_, err := tracker.RegisterFingerprint(data1)
	if err != nil {
		t.Fatalf("Failed to register fingerprint1: %v", err)
	}

	_, err = tracker.RegisterFingerprint(data2)
	if err != nil {
		t.Fatalf("Failed to register fingerprint2: %v", err)
	}

	suspiciousGroups := tracker.DetectSuspiciousGroups()
	t.Logf("Suspicious groups found: %d", len(suspiciousGroups))
}

func TestDeviceFingerprintTracker_GetAllDeviceGroups(t *testing.T) {
	tracker := NewDeviceFingerprintTracker()

	groupID := "test_group"
	tracker.deviceGroups[groupID] = []string{"fingerprint1", "fingerprint2", "fingerprint3"}

	groups := tracker.GetAllDeviceGroups()
	if len(groups) != 1 {
		t.Errorf("Expected 1 device group, got %d", len(groups))
	}

	if groups[0].GroupID != groupID {
		t.Errorf("Expected GroupID '%s', got '%s'", groupID, groups[0].GroupID)
	}
}

func TestDeviceFingerprintTracker_GetFingerprintsByUserID(t *testing.T) {
	tracker := NewDeviceFingerprintTracker()

	data := map[string]interface{}{
		"user_agent": "Mozilla/5.0",
	}

	fingerprint, err := tracker.RegisterFingerprint(data)
	if err != nil {
		t.Fatalf("Failed to register fingerprint: %v", err)
	}

	err = tracker.AssociateUser(fingerprint.FingerprintID, "user1")
	if err != nil {
		t.Fatalf("Failed to associate user: %v", err)
	}

	fingerprints := tracker.GetFingerprintsByUserID("user1")
	if len(fingerprints) != 1 {
		t.Errorf("Expected 1 fingerprint for user1, got %d", len(fingerprints))
	}
}

func TestDeviceFingerprintTracker_GetFingerprintHistory(t *testing.T) {
	tracker := NewDeviceFingerprintTracker()

	data := map[string]interface{}{
		"user_agent": "Mozilla/5.0",
	}

	fingerprint, err := tracker.RegisterFingerprint(data)
	if err != nil {
		t.Fatalf("Failed to register fingerprint: %v", err)
	}

	history := tracker.GetFingerprintHistory(fingerprint.FingerprintID)
	if len(history) != 1 {
		t.Errorf("Expected 1 history entry, got %d", len(history))
	}
}

func TestDeviceFingerprintTracker_GetAllFingerprints(t *testing.T) {
	tracker := NewDeviceFingerprintTracker()

	data := map[string]interface{}{
		"user_agent": "Mozilla/5.0",
	}

	_, err := tracker.RegisterFingerprint(data)
	if err != nil {
		t.Fatalf("Failed to register fingerprint: %v", err)
	}

	fingerprints := tracker.GetAllFingerprints()
	if len(fingerprints) != 1 {
		t.Errorf("Expected 1 fingerprint, got %d", len(fingerprints))
	}
}

func TestDeviceFingerprintTracker_GetTrackerStatistics(t *testing.T) {
	tracker := NewDeviceFingerprintTracker()

	data := map[string]interface{}{
		"user_agent": "Mozilla/5.0",
	}

	_, err := tracker.RegisterFingerprint(data)
	if err != nil {
		t.Fatalf("Failed to register fingerprint: %v", err)
	}

	stats := tracker.GetTrackerStatistics()
	if stats == nil {
		t.Error("Statistics should not be nil")
	}

	if stats.TotalFingerprints != 1 {
		t.Errorf("Expected TotalFingerprints=1, got %d", stats.TotalFingerprints)
	}
}

func TestDeviceFingerprintTracker_ExportImportFingerprints(t *testing.T) {
	tracker := NewDeviceFingerprintTracker()

	data := map[string]interface{}{
		"user_agent": "Mozilla/5.0",
	}

	_, err := tracker.RegisterFingerprint(data)
	if err != nil {
		t.Fatalf("Failed to register fingerprint: %v", err)
	}

	exported, err := tracker.ExportFingerprints()
	if err != nil {
		t.Fatalf("Failed to export fingerprints: %v", err)
	}

	if len(exported) == 0 {
		t.Error("Exported data should not be empty")
	}

	newTracker := NewDeviceFingerprintTracker()
	err = newTracker.ImportFingerprints(exported)
	if err != nil {
		t.Fatalf("Failed to import fingerprints: %v", err)
	}

	fingerprints := newTracker.GetAllFingerprints()
	if len(fingerprints) != 1 {
		t.Errorf("Expected 1 fingerprint after import, got %d", len(fingerprints))
	}
}
