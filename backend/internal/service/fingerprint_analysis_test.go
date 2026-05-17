package service

import (
	"testing"
	"time"
)

func TestFingerprintDatabase_AddFingerprint(t *testing.T) {
	db := NewFingerprintDatabase()

	fp := &FingerprintAnalysis{
		FingerprintID:    "test_fp_001",
		CanvasHash:      "canvas123",
		WebGLHash:       "webgl456",
		UserAgent:       "Mozilla/5.0",
		ScreenResolution: "1920x1080",
		Timezone:        "Asia/Shanghai",
		Language:        "zh-CN",
	}

	db.AddFingerprint(fp)

	retrieved, exists := db.GetFingerprint("test_fp_001")
	if !exists {
		t.Fatal("Expected fingerprint to exist after adding")
	}

	if retrieved.CanvasHash != "canvas123" {
		t.Errorf("Expected CanvasHash to be 'canvas123', got '%s'", retrieved.CanvasHash)
	}

	if retrieved.RequestCount != 1 {
		t.Errorf("Expected RequestCount to be 1, got %d", retrieved.RequestCount)
	}

	if retrieved.FirstSeen.IsZero() {
		t.Error("Expected FirstSeen to be set")
	}
}

func TestFingerprintDatabase_CalculateSimilarity(t *testing.T) {
	db := NewFingerprintDatabase()

	fp1 := &FingerprintAnalysis{
		FingerprintID:    "fp1",
		CanvasHash:      "hash1",
		WebGLHash:       "webgl1",
		UserAgent:       "Mozilla/5.0",
		ScreenResolution: "1920x1080",
		Timezone:        "UTC",
		Language:        "en-US",
		Platform:        "Win32",
	}

	fp2 := &FingerprintAnalysis{
		FingerprintID:    "fp2",
		CanvasHash:      "hash1",
		WebGLHash:       "webgl1",
		UserAgent:       "Mozilla/5.0",
		ScreenResolution: "1920x1080",
		Timezone:        "UTC",
		Language:        "en-US",
		Platform:        "Win32",
	}

	fp3 := &FingerprintAnalysis{
		FingerprintID:    "fp3",
		CanvasHash:      "hash2",
		WebGLHash:       "webgl2",
		UserAgent:       "Chrome/90.0",
		ScreenResolution: "1366x768",
		Timezone:        "America/New_York",
		Language:        "en-US",
		Platform:        "MacIntel",
	}

	similarity1 := db.CalculateSimilarity(fp1, fp2)
	if similarity1 != 100 {
		t.Errorf("Expected 100%% similarity for identical fingerprints, got %.2f%%", similarity1)
	}

	similarity2 := db.CalculateSimilarity(fp1, fp3)
	if similarity2 >= 50 {
		t.Errorf("Expected lower similarity for different fingerprints, got %.2f%%", similarity2)
	}

	nilSimilarity := db.CalculateSimilarity(nil, fp1)
	if nilSimilarity != 0 {
		t.Errorf("Expected 0 similarity for nil fingerprint, got %.2f%%", nilSimilarity)
	}
}

func TestFingerprintDatabase_FindSimilarFingerprints(t *testing.T) {
	db := NewFingerprintDatabase()

	fps := []*FingerprintAnalysis{
		{
			FingerprintID:    "fp1",
			CanvasHash:      "hash1",
			WebGLHash:       "webgl1",
			UserAgent:       "Mozilla/5.0",
			ScreenResolution: "1920x1080",
		},
		{
			FingerprintID:    "fp2",
			CanvasHash:      "hash1",
			WebGLHash:       "webgl1",
			UserAgent:       "Mozilla/5.0",
			ScreenResolution: "1920x1080",
		},
		{
			FingerprintID:    "fp3",
			CanvasHash:      "hash2",
			WebGLHash:       "webgl2",
			UserAgent:       "Chrome/90.0",
			ScreenResolution: "1366x768",
		},
	}

	for _, fp := range fps {
		db.AddFingerprint(fp)
	}

	similar := db.FindSimilarFingerprints("fp1", 70)

	if len(similar) < 1 {
		t.Error("Expected at least 1 similar fingerprint")
	}

	hasFp2 := false
	for _, s := range similar {
		if s.FingerprintID == "fp2" {
			hasFp2 = true
			if s.Similarity < 70 {
				t.Errorf("Expected similarity >= 70%% for fp2, got %.2f%%", s.Similarity)
			}
		}
	}
	if !hasFp2 {
		t.Error("Expected fp2 to be similar to fp1")
	}
}

func TestFingerprintDatabase_RemoveFingerprint(t *testing.T) {
	db := NewFingerprintDatabase()

	fp := &FingerprintAnalysis{
		FingerprintID: "fp_to_remove",
		CanvasHash:    "hash123",
	}

	db.AddFingerprint(fp)

	_, exists := db.GetFingerprint("fp_to_remove")
	if !exists {
		t.Fatal("Fingerprint should exist before removal")
	}

	db.RemoveFingerprint("fp_to_remove")

	_, exists = db.GetFingerprint("fp_to_remove")
	if exists {
		t.Error("Fingerprint should not exist after removal")
	}
}

func TestFingerprintDatabase_GetStats(t *testing.T) {
	db := NewFingerprintDatabase()

	initialStats := db.GetStats()
	if initialStats.TotalFingerprints != 0 {
		t.Errorf("Expected 0 initial fingerprints, got %d", initialStats.TotalFingerprints)
	}

	fps := []*FingerprintAnalysis{
		{FingerprintID: "fp1", IsKnownBot: true, AnomalyScore: 80},
		{FingerprintID: "fp2", IsKnownVPN: true, AnomalyScore: 50},
		{FingerprintID: "fp3", AnomalyScore: 30},
	}

	for _, fp := range fps {
		db.AddFingerprint(fp)
	}

	stats := db.GetStats()

	if stats.TotalFingerprints != 3 {
		t.Errorf("Expected 3 fingerprints, got %d", stats.TotalFingerprints)
	}
	if stats.BotFingerprints != 1 {
		t.Errorf("Expected 1 bot fingerprint, got %d", stats.BotFingerprints)
	}
	if stats.VPNFingerprints != 1 {
		t.Errorf("Expected 1 VPN fingerprint, got %d", stats.VPNFingerprints)
	}
	if stats.HighRiskCount != 1 {
		t.Errorf("Expected 1 high risk fingerprint, got %d", stats.HighRiskCount)
	}
}

func TestFingerprintDatabase_CleanupOldData(t *testing.T) {
	db := NewFingerprintDatabase()

	oldFp := &FingerprintAnalysis{
		FingerprintID: "old_fp",
		CanvasHash:    "hash",
		RequestCount:  1,
	}
	db.AddFingerprint(oldFp)

	db.mu.Lock()
	db.fingerprints["old_fp"].LastSeen = time.Now().Add(-48 * time.Hour)
	db.mu.Unlock()

	removed := db.CleanupOldData(24 * time.Hour)

	if removed != 1 {
		t.Errorf("Expected 1 removed fingerprint, got %d", removed)
	}

	_, exists := db.GetFingerprint("old_fp")
	if exists {
		t.Error("Old fingerprint should have been removed")
	}

	newFp := &FingerprintAnalysis{
		FingerprintID: "new_fp",
		CanvasHash:    "hash",
		RequestCount:  1,
	}
	db.AddFingerprint(newFp)

	db.mu.Lock()
	db.fingerprints["new_fp"].LastSeen = time.Now().Add(-48 * time.Hour)
	db.mu.Unlock()

	removed = db.CleanupOldData(24 * time.Hour)

	_, exists = db.GetFingerprint("new_fp")
	if !exists {
		t.Error("New fingerprint with high request count should not be removed")
	}
}

func TestFingerprintDatabase_GetCluster(t *testing.T) {
	db := NewFingerprintDatabase()

	fps := []*FingerprintAnalysis{
		{FingerprintID: "fp1", UserAgent: "Mozilla/5.0", CanvasHash: "hash1"},
		{FingerprintID: "fp2", UserAgent: "Mozilla/5.0", CanvasHash: "hash1"},
		{FingerprintID: "fp3", UserAgent: "Chrome/90.0", CanvasHash: "hash2"},
	}

	for _, fp := range fps {
		db.AddFingerprint(fp)
	}

	clusters := db.GetAllClusters()
	if len(clusters) == 0 {
		t.Error("Expected at least one cluster")
	}

	for _, cluster := range clusters {
		if cluster.ClusterID == "" {
			t.Error("Cluster ID should not be empty")
		}
		if cluster.Size == 0 {
			t.Error("Cluster size should not be zero")
		}
	}
}

func TestFingerprintAnalyzer_AnalyzeFingerprint(t *testing.T) {
	analyzer := NewFingerprintAnalyzer()

	data := map[string]interface{}{
		"canvas_hash":        "canvas_test_hash",
		"webgl_hash":        "webgl_test_hash",
		"audio_hash":        "audio_test_hash",
		"user_agent":        "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
		"screen_resolution":  "1920x1080",
		"timezone":           "Asia/Shanghai",
		"language":           "zh-CN",
		"platform":           "Win32",
		"hardware_concurrency": float64(8),
		"device_memory":      float64(16),
	}

	fp, anomaly, err := analyzer.AnalyzeFingerprint(data)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if fp == nil {
		t.Fatal("Expected fingerprint to be returned")
	}

	if fp.CanvasHash != "canvas_test_hash" {
		t.Errorf("Expected CanvasHash 'canvas_test_hash', got '%s'", fp.CanvasHash)
	}

	if fp.HardwareConcurrency != 8 {
		t.Errorf("Expected HardwareConcurrency 8, got %d", fp.HardwareConcurrency)
	}

	if anomaly == nil {
		t.Fatal("Expected anomaly result to be returned")
	}
}

func TestFingerprintAnalyzer_DetectBotIndicators(t *testing.T) {
	analyzer := NewFingerprintAnalyzer()

	testCases := []struct {
		name       string
		userAgent  string
		expectBot  bool
	}{
		{"Chrome on Windows", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36", false},
		{"Headless Chrome", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) HeadlessChrome/91.0.4472.0 Safari/537.36", true},
		{"Puppeteer", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.0 Safari/537.36 puppeteer", true},
		{"Selenium", "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:89.0) Gecko/20100101 Firefox/89.0 selenium", true},
		{"Playwright", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.0 Safari/537.36 playwright", true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			data := map[string]interface{}{
				"user_agent": tc.userAgent,
			}

			fp, _, _ := analyzer.AnalyzeFingerprint(data)

			if tc.expectBot && !fp.IsKnownBot {
				t.Errorf("Expected bot to be detected for %s", tc.name)
			}
			if !tc.expectBot && fp.IsKnownBot {
				t.Errorf("Did not expect bot detection for %s", tc.name)
			}
		})
	}
}

func TestFingerprintAnalyzer_CalculateConfidence(t *testing.T) {
	analyzer := NewFingerprintAnalyzer()

	testCases := []struct {
		name        string
		data        map[string]interface{}
		minExpected float64
	}{
		{
			name: "Complete data",
			data: map[string]interface{}{
				"canvas_hash":        "hash1",
				"webgl_hash":        "hash2",
				"audio_hash":        "hash3",
				"font_hash":         "hash4",
				"user_agent":        "Mozilla/5.0",
				"screen_resolution":  "1920x1080",
			},
			minExpected: 0.8,
		},
		{
			name: "Partial data",
			data: map[string]interface{}{
				"canvas_hash": "hash1",
				"user_agent":  "Mozilla/5.0",
			},
			minExpected: 0.2,
		},
		{
			name:        "Empty data",
			data:        map[string]interface{}{},
			minExpected: 0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fp, _, _ := analyzer.AnalyzeFingerprint(tc.data)

			if fp.Confidence < tc.minExpected {
				t.Errorf("Expected confidence >= %.2f, got %.2f", tc.minExpected, fp.Confidence)
			}
		})
	}
}

func TestFingerprintAnalyzer_GetSimilarFingerprints(t *testing.T) {
	analyzer := NewFingerprintAnalyzer()

	data1 := map[string]interface{}{
		"user_agent":        "Mozilla/5.0",
		"canvas_hash":       "common_hash",
		"webgl_hash":        "webgl1",
		"screen_resolution": "1920x1080",
	}

	data2 := map[string]interface{}{
		"user_agent":        "Mozilla/5.0",
		"canvas_hash":       "common_hash",
		"webgl_hash":        "webgl2",
		"screen_resolution": "1920x1080",
	}

	data3 := map[string]interface{}{
		"user_agent":        "Chrome/90.0",
		"canvas_hash":       "different_hash",
		"webgl_hash":        "webgl3",
		"screen_resolution": "1366x768",
	}

	fp1, _, _ := analyzer.AnalyzeFingerprint(data1)
	fp2, _, _ := analyzer.AnalyzeFingerprint(data2)
	fp3, _, _ := analyzer.AnalyzeFingerprint(data3)

	similar1 := analyzer.GetSimilarFingerprints(fp1.FingerprintID, 60)
	hasFp2 := false
	for _, s := range similar1 {
		if s.FingerprintID == fp2.FingerprintID && s.Similarity >= 60 {
			hasFp2 = true
			break
		}
	}
	if !hasFp2 {
		t.Error("Expected fp1 and fp2 to be similar")
	}

	similar3 := analyzer.GetSimilarFingerprints(fp3.FingerprintID, 60)
	for _, s := range similar3 {
		if s.FingerprintID == fp1.FingerprintID && s.Similarity >= 60 {
			t.Error("Did not expect fp3 to be similar to fp1")
		}
	}
}

func TestFingerprintAnalyzer_ExtendedAnalysis(t *testing.T) {
	analyzer := NewFingerprintAnalyzer()

	data := map[string]interface{}{
		"canvas_hash":        "test_hash",
		"webgl_hash":        "webgl_hash",
		"user_agent":        "Mozilla/5.0",
		"webrtc_ips":        []interface{}{"192.168.1.1", "10.0.0.1"},
		"connection_type":   "wifi",
		"request_interval":  float64(0.5),
		"request_paths":     []interface{}{"/api/v1/captcha", "/api/v1/verify"},
	}

	extended, err := analyzer.AnalyzeWithExtendedMetrics(data)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if extended == nil {
		t.Fatal("Expected extended analysis to be returned")
	}

	if extended.BaseAnalysis == nil {
		t.Error("Expected base analysis to be set")
	}

	if extended.AccuracyScore <= 0 {
		t.Error("Expected positive accuracy score")
	}

	if extended.PredictionScore <= 0 {
		t.Error("Expected positive prediction score")
	}
}

func TestGenerateFingerprintID(t *testing.T) {
	data1 := map[string]interface{}{
		"user_agent":        "Mozilla/5.0",
		"canvas_hash":       "hash1",
		"screen_resolution": "1920x1080",
	}

	data2 := map[string]interface{}{
		"user_agent":        "Mozilla/5.0",
		"canvas_hash":       "hash1",
		"screen_resolution": "1920x1080",
	}

	data3 := map[string]interface{}{
		"user_agent":        "Chrome/90.0",
		"canvas_hash":       "hash1",
		"screen_resolution": "1920x1080",
	}

	id1 := generateFingerprintID(data1)
	id2 := generateFingerprintID(data2)
	id3 := generateFingerprintID(data3)

	if id1 == id2 {
		t.Log("IDs for same data may be similar (includes random component)")
	}

	if len(id1) < 10 {
		t.Errorf("Expected ID length >= 10, got %d", len(id1))
	}

	if id1 == id3 {
		t.Error("Different data should produce different IDs")
	}
}

func TestAnomalyResult_Severity(t *testing.T) {
	db := NewFingerprintDatabase()

	fps := []*FingerprintAnalysis{
		{FingerprintID: "high_risk", AnomalyScore: 85},
		{FingerprintID: "medium_risk", AnomalyScore: 55},
		{FingerprintID: "low_risk", AnomalyScore: 25},
	}

	for _, fp := range fps {
		db.AddFingerprint(fp)
	}

	highAnomaly := db.DetectAnomaly("high_risk")
	if highAnomaly.Severity != "high" {
		t.Errorf("Expected 'high' severity, got '%s'", highAnomaly.Severity)
	}

	mediumAnomaly := db.DetectAnomaly("medium_risk")
	if mediumAnomaly.Severity != "medium" {
		t.Errorf("Expected 'medium' severity, got '%s'", mediumAnomaly.Severity)
	}

	lowAnomaly := db.DetectAnomaly("low_risk")
	if lowAnomaly.Severity != "low" {
		t.Errorf("Expected 'low' severity, got '%s'", lowAnomaly.Severity)
	}
}

func TestExportImport(t *testing.T) {
	db := NewFingerprintDatabase()

	originalFp := &FingerprintAnalysis{
		FingerprintID: "export_test",
		CanvasHash:   "export_hash",
		WebGLHash:    "webgl_export",
	}
	db.AddFingerprint(originalFp)

	exportData, err := db.ExportData()
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	if len(exportData) == 0 {
		t.Error("Expected non-empty export data")
	}

	newDb := NewFingerprintDatabase()
	err = newDb.ImportData(exportData)
	if err != nil {
		t.Fatalf("Import failed: %v", err)
	}

	imported, exists := newDb.GetFingerprint("export_test")
	if !exists {
		t.Fatal("Expected imported fingerprint to exist")
	}

	if imported.CanvasHash != "export_hash" {
		t.Errorf("Expected CanvasHash 'export_hash', got '%s'", imported.CanvasHash)
	}
}

func TestDatabase_ConcurrentAccess(t *testing.T) {
	db := NewFingerprintDatabase()

	done := make(chan bool)

	for i := 0; i < 10; i++ {
		go func(idx int) {
			for j := 0; j < 100; j++ {
				fp := &FingerprintAnalysis{
					FingerprintID:    "concurrent_fp",
					CanvasHash:      "hash",
				}
				db.AddFingerprint(fp)
				db.GetFingerprint("concurrent_fp")
				db.GetStats()
			}
			done <- true
		}(i)
	}

	for i := 0; i < 10; i++ {
		<-done
	}

	stats := db.GetStats()
	if stats.TotalFingerprints == 0 {
		t.Error("Expected fingerprints to exist after concurrent access")
	}
}
