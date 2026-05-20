package service

import (
	"testing"
	"time"
)

func TestNewCanvasFingerprintService(t *testing.T) {
	service := NewCanvasFingerprintService()
	if service == nil {
		t.Error("NewCanvasFingerprintService returned nil")
	}
}

func TestCanvasFingerprintService_GenerateFingerprint(t *testing.T) {
	service := NewCanvasFingerprintService()

	fingerprint, err := service.GenerateFingerprint("test-canvas-data", "TestAgent/1.0")
	if err != nil {
		t.Errorf("GenerateFingerprint failed: %v", err)
	}
	if fingerprint == nil {
		t.Error("GenerateFingerprint returned nil")
	}
	if fingerprint.Hash == "" {
		t.Error("Fingerprint hash should not be empty")
	}
}

func TestCanvasFingerprintService_GenerateDifferentFingerprints(t *testing.T) {
	service := NewCanvasFingerprintService()

	fp1, _ := service.GenerateFingerprint("data1", "Mozilla/5.0")
	fp2, _ := service.GenerateFingerprint("data2", "Mozilla/5.0")

	if fp1.Hash == fp2.Hash {
		t.Error("Different canvas data should produce different fingerprints")
	}
}

func TestCanvasFingerprintService_StoreFingerprint(t *testing.T) {
	service := NewCanvasFingerprintService()

	fingerprint := &CanvasFingerprint{
		ID:        "test-canvas-fp",
		Hash:      "abc123hash",
		UserAgent: "TestAgent",
		CreatedAt: time.Now(),
	}

	err := service.StoreFingerprint(fingerprint)
	if err != nil {
		t.Errorf("StoreFingerprint failed: %v", err)
	}
}

func TestCanvasFingerprintService_GetFingerprint(t *testing.T) {
	service := NewCanvasFingerprintService()

	fp, err := service.GetFingerprint("test-canvas-fp")
	if err == nil {
		t.Error("Should return error for non-existent fingerprint")
	}
	if fp != nil {
		t.Error("Should return nil for non-existent fingerprint")
	}

	fingerprint := &CanvasFingerprint{
		ID:        "test-canvas-fp-2",
		Hash:      "def456hash",
		UserAgent: "TestAgent",
		CreatedAt: time.Now(),
	}
	service.StoreFingerprint(fingerprint)

	fp, err = service.GetFingerprint("test-canvas-fp-2")
	if err != nil {
		t.Errorf("GetFingerprint failed: %v", err)
	}
	if fp == nil {
		t.Error("Should return fingerprint after storing")
	}
}

func TestCanvasFingerprintService_DeleteFingerprint(t *testing.T) {
	service := NewCanvasFingerprintService()

	fingerprint := &CanvasFingerprint{
		ID:        "test-canvas-delete",
		Hash:      "deletehash",
		UserAgent: "TestAgent",
		CreatedAt: time.Now(),
	}
	service.StoreFingerprint(fingerprint)

	err := service.DeleteFingerprint("test-canvas-delete")
	if err != nil {
		t.Errorf("DeleteFingerprint failed: %v", err)
	}
}

func TestCanvasFingerprintService_ListFingerprints(t *testing.T) {
	service := NewCanvasFingerprintService()

	fps, err := service.ListFingerprints(0, 10)
	if err != nil {
		t.Errorf("ListFingerprints failed: %v", err)
	}
	if fps == nil {
		t.Error("ListFingerprints should return empty slice, not nil")
	}
}

func TestCanvasFingerprintService_GetBrowserInfo(t *testing.T) {
	service := NewCanvasFingerprintService()

	browser := service.GetBrowserInfo("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	if browser == nil {
		t.Error("GetBrowserInfo returned nil")
	}
}

func TestCanvasFingerprintService_GetBrowserInfoUnknown(t *testing.T) {
	service := NewCanvasFingerprintService()

	browser := service.GetBrowserInfo("UnknownBrowser/1.0")
	if browser == nil {
		t.Error("GetBrowserInfo should return BrowserInfo for unknown browser")
	}
}

func TestCanvasFingerprintService_AnalyzeFingerprint(t *testing.T) {
	service := NewCanvasFingerprintService()

	analysis := service.AnalyzeFingerprint("test-hash-123", "Mozilla/5.0")
	if analysis == nil {
		t.Error("AnalyzeFingerprint returned nil")
	}
}

func TestCanvasFingerprintService_GetStatistics(t *testing.T) {
	service := NewCanvasFingerprintService()

	stats := service.GetStatistics()
	if stats == nil {
		t.Error("GetStatistics returned nil")
	}
}

func TestCanvasFingerprintService_GetUniqueBrowserCount(t *testing.T) {
	service := NewCanvasFingerprintService()

	count := service.GetUniqueBrowserCount()
	if count < 0 {
		t.Error("Unique browser count should not be negative")
	}
}

func TestCanvasFingerprintService_GetUniquePlatformCount(t *testing.T) {
	service := NewCanvasFingerprintService()

	count := service.GetUniquePlatformCount()
	if count < 0 {
		t.Error("Unique platform count should not be negative")
	}
}

func TestCanvasFingerprintService_GetFingerprintCount(t *testing.T) {
	service := NewCanvasFingerprintService()

	count := service.GetFingerprintCount()
	if count < 0 {
		t.Error("Fingerprint count should not be negative")
	}
}

func TestCanvasFingerprintService_UpdateConfig(t *testing.T) {
	service := NewCanvasFingerprintService()

	service.UpdateConfig(&CanvasConfig{
		EnableDetailedAnalysis: true,
		SimilarityThreshold:    0.9,
	})

	if !service.config.EnableDetailedAnalysis {
		t.Error("Config not updated correctly")
	}
}

func TestCanvasFingerprintService_GetConfig(t *testing.T) {
	service := NewCanvasFingerprintService()

	config := service.GetConfig()
	if config == nil {
		t.Error("GetConfig returned nil")
	}
}

func TestCanvasFingerprintService_ClearCache(t *testing.T) {
	service := NewCanvasFingerprintService()

	service.ClearCache()
}

func TestCanvasFingerprintService_GetCacheSize(t *testing.T) {
	service := NewCanvasFingerprintService()

	size := service.GetCacheSize()
	if size < 0 {
		t.Error("Cache size should not be negative")
	}
}

func TestCanvasFingerprintService_CompareFingerprints(t *testing.T) {
	service := NewCanvasFingerprintService()

	fp1 := &CanvasFingerprint{
		ID:        "compare-1",
		Hash:      "hash123",
		UserAgent: "Mozilla/5.0",
		CreatedAt: time.Now(),
	}
	fp2 := &CanvasFingerprint{
		ID:        "compare-2",
		Hash:      "hash123",
		UserAgent: "Mozilla/5.0",
		CreatedAt: time.Now(),
	}

	similarity, err := service.CompareFingerprints(fp1, fp2)
	if err != nil {
		t.Errorf("CompareFingerprints failed: %v", err)
	}
	if similarity < 0.0 || similarity > 1.0 {
		t.Error("Similarity should be between 0 and 1")
	}
}

func TestCanvasFingerprintService_FindSimilarFingerprints(t *testing.T) {
	service := NewCanvasFingerprintService()

	fp := &CanvasFingerprint{
		ID:        "similar-1",
		Hash:      "target-hash",
		UserAgent: "Mozilla/5.0",
		CreatedAt: time.Now(),
	}
	service.StoreFingerprint(fp)

	similar, err := service.FindSimilarFingerprints("target-hash", 0.8)
	if err != nil {
		t.Errorf("FindSimilarFingerprints failed: %v", err)
	}
	if similar == nil {
		t.Error("FindSimilarFingerprints should return slice, not nil")
	}
}

func TestCanvasFingerprintService_DetectFingerprintSpoofing(t *testing.T) {
	service := NewCanvasFingerprintService()

	result := service.DetectFingerprintSpoofing("test-hash", "Mozilla/5.0", "Windows NT 10.0")
	if result == nil {
		t.Error("DetectFingerprintSpoofing returned nil")
	}
}

func TestCanvasFingerprintService_GetFingerprintAge(t *testing.T) {
	service := NewCanvasFingerprintService()

	age := service.GetFingerprintAge("test-hash")
	if age < 0 {
		t.Error("Fingerprint age should not be negative")
	}
}

func TestCanvasFingerprintService_GetRecentFingerprints(t *testing.T) {
	service := NewCanvasFingerprintService()

	fps, err := service.GetRecentFingerprints(10)
	if err != nil {
		t.Errorf("GetRecentFingerprints failed: %v", err)
	}
	if fps == nil {
		t.Error("GetRecentFingerprints should return slice, not nil")
	}
}

func TestCanvasFingerprintService_ExportFingerprints(t *testing.T) {
	service := NewCanvasFingerprintService()

	export, err := service.ExportFingerprints("json")
	if err != nil {
		t.Errorf("ExportFingerprints failed: %v", err)
	}
	if len(export) == 0 {
		t.Error("Export should return data")
	}
}

func TestCanvasFingerprintService_ImportFingerprints(t *testing.T) {
	service := NewCanvasFingerprintService()

	data := []byte(`[{"ID": "import-1", "Hash": "hash-import", "UserAgent": "TestAgent"}]`)

	count, err := service.ImportFingerprints(data)
	if err != nil {
		t.Errorf("ImportFingerprints failed: %v", err)
	}
	if count != 1 {
		t.Errorf("Should import 1 fingerprint, got %d", count)
	}
}

func TestCanvasFingerprintService_CleanupOldFingerprints(t *testing.T) {
	service := NewCanvasFingerprintService()

	cleaned, err := service.CleanupOldFingerprints(30 * 24 * time.Hour)
	if err != nil {
		t.Errorf("CleanupOldFingerprints failed: %v", err)
	}
	if cleaned < 0 {
		t.Error("Cleaned count should not be negative")
	}
}

func TestCanvasFingerprintService_GetFingerprintByHash(t *testing.T) {
	service := NewCanvasFingerprintService()

	fp, err := service.GetFingerprintByHash("non-existent-hash")
	if err == nil {
		t.Error("Should return error for non-existent hash")
	}
	if fp != nil {
		t.Error("Should return nil for non-existent hash")
	}
}

func TestCanvasFingerprintService_GetFingerprintsByUserAgent(t *testing.T) {
	service := NewCanvasFingerprintService()

	fps, err := service.GetFingerprintsByUserAgent("Mozilla/5.0", 0, 10)
	if err != nil {
		t.Errorf("GetFingerprintsByUserAgent failed: %v", err)
	}
	if fps == nil {
		t.Error("GetFingerprintsByUserAgent should return slice, not nil")
	}
}

func TestCanvasFingerprintService_GetFingerprintsByBrowser(t *testing.T) {
	service := NewCanvasFingerprintService()

	fps, err := service.GetFingerprintsByBrowser("Chrome", 0, 10)
	if err != nil {
		t.Errorf("GetFingerprintsByBrowser failed: %v", err)
	}
	if fps == nil {
		t.Error("GetFingerprintsByBrowser should return slice, not nil")
	}
}

func TestCanvasFingerprintService_GetFingerprintsByPlatform(t *testing.T) {
	service := NewCanvasFingerprintService()

	fps, err := service.GetFingerprintsByPlatform("Windows", 0, 10)
	if err != nil {
		t.Errorf("GetFingerprintsByPlatform failed: %v", err)
	}
	if fps == nil {
		t.Error("GetFingerprintsByPlatform should return slice, not nil")
	}
}

func TestCanvasFingerprintService_VerifyFingerprint(t *testing.T) {
	service := NewCanvasFingerprintService()

	result, err := service.VerifyFingerprint("test-hash", "Mozilla/5.0", map[string]interface{}{})
	if err != nil {
		t.Errorf("VerifyFingerprint failed: %v", err)
	}
	if result == nil {
		t.Error("VerifyFingerprint should return result")
	}
}

func TestCanvasFingerprint_Fields(t *testing.T) {
	fp := &CanvasFingerprint{
		ID:        "test-id",
		Hash:      "test-hash",
		UserAgent: "Mozilla/5.0",
		CreatedAt: time.Now(),
	}

	if fp.ID != "test-id" {
		t.Errorf("ID should be test-id, got %s", fp.ID)
	}
	if fp.Hash != "test-hash" {
		t.Errorf("Hash should be test-hash, got %s", fp.Hash)
	}
}

func TestCanvasConfig_Fields(t *testing.T) {
	config := &CanvasConfig{
		EnableDetailedAnalysis: true,
		SimilarityThreshold:    0.95,
		MaxCacheSize:         10000,
	}

	if !config.EnableDetailedAnalysis {
		t.Error("EnableDetailedAnalysis should be true")
	}
	if config.SimilarityThreshold != 0.95 {
		t.Errorf("SimilarityThreshold should be 0.95, got %f", config.SimilarityThreshold)
	}
}

func TestCanvasFingerprintAnalysis_Fields(t *testing.T) {
	analysis := &CanvasFingerprintAnalysis{
		Hash:            "test-hash",
		UserAgent:      "Mozilla/5.0",
		Browser:        "Chrome",
		Platform:       "Windows",
		IsUnique:       true,
		SimilarCount:   0,
		CreatedAt:      time.Now(),
		LastSeenAt:     time.Now(),
		Confidence:     0.95,
	}

	if analysis.Hash != "test-hash" {
		t.Errorf("Hash should be test-hash, got %s", analysis.Hash)
	}
	if !analysis.IsUnique {
		t.Error("IsUnique should be true")
	}
}

func TestBrowserInfo_Fields(t *testing.T) {
	info := &BrowserInfo{
		Name:    "Chrome",
		Version: "120.0.0.0",
		Platform: "Windows",
	}

	if info.Name != "Chrome" {
		t.Errorf("Name should be Chrome, got %s", info.Name)
	}
	if info.Version != "120.0.0.0" {
		t.Errorf("Version should be 120.0.0.0, got %s", info.Version)
	}
}

func TestCanvasSpoofingDetection_Fields(t *testing.T) {
	detection := &CanvasSpoofingDetection{
		Hash:           "test-hash",
		Detected:       true,
		SpoofingType:   "canvas-tampering",
		Confidence:     0.85,
		Details:        "Canvas data manipulation detected",
	}

	if !detection.Detected {
		t.Error("Detected should be true")
	}
	if detection.SpoofingType != "canvas-tampering" {
		t.Errorf("SpoofingType should be canvas-tampering, got %s", detection.SpoofingType)
	}
}

func TestCanvasFingerprintService_GetAllBrowserTypes(t *testing.T) {
	service := NewCanvasFingerprintService()

	browsers := service.GetAllBrowserTypes()
	if browsers == nil {
		t.Error("GetAllBrowserTypes should return slice, not nil")
	}
}

func TestCanvasFingerprintService_GetAllPlatformTypes(t *testing.T) {
	service := NewCanvasFingerprintService()

	platforms := service.GetAllPlatformTypes()
	if platforms == nil {
		t.Error("GetAllPlatformTypes should return slice, not nil")
	}
}
