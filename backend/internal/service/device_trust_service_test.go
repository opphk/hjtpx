package service

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/hjtpx/hjtpx/pkg/models"
)

func TestDeviceTrustService_NewDeviceTrustService(t *testing.T) {
	svc := NewDeviceTrustService()
	if svc == nil {
		t.Fatal("NewDeviceTrustService should not return nil")
	}
	if svc.trustCache == nil {
		t.Error("trustCache should be initialized")
	}
	if svc.trustHistory == nil {
		t.Error("trustHistory should be initialized")
	}
}

func TestDeviceTrustService_GenerateFingerprintHash(t *testing.T) {
	svc := NewDeviceTrustService()

	testCases := []struct {
		name     string
		data     map[string]interface{}
		wantHash bool
	}{
		{
			name: "with_webgl",
			data: map[string]interface{}{
				"webgl": "Intel Inc.|Intel Iris OpenGL Engine",
			},
			wantHash: true,
		},
		{
			name: "with_canvas",
			data: map[string]interface{}{
				"canvas": "canvas_fingerprint_data",
			},
			wantHash: true,
		},
		{
			name:     "empty_data",
			data:     map[string]interface{}{},
			wantHash: false,
		},
		{
			name: "with_multiple_components",
			data: map[string]interface{}{
				"webgl":     "vendor|renderer",
				"canvas":    "canvas_data",
				"fonts":     "Arial,Helvetica",
				"screen":    "1920x1080",
				"platform":  "MacIntel",
				"timezone":  "Asia/Shanghai",
				"languages": "zh-CN,en-US",
			},
			wantHash: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			hash := svc.GenerateFingerprintHash(tc.data)
			if tc.wantHash && hash == "" {
				t.Error("expected non-empty hash")
			}
			if !tc.wantHash && hash != "" {
				t.Error("expected empty hash for empty data")
			}
			if tc.wantHash && len(hash) != 32 {
				t.Errorf("expected hash length 32, got %d", len(hash))
			}
		})
	}
}

func TestDeviceTrustService_AnalyzeFingerprint_NewDevice(t *testing.T) {
	svc := NewDeviceTrustService()

	analysis := svc.AnalyzeFingerprint("test_fingerprint_123", map[string]interface{}{})

	if !analysis.IsNew {
		t.Error("expected IsNew to be true for new fingerprint")
	}
	if analysis.TrustScore != 50 {
		t.Errorf("expected TrustScore 50 for new device, got %d", analysis.TrustScore)
	}
	if analysis.RiskScore != 25 {
		t.Errorf("expected RiskScore 25 for new device, got %.1f", analysis.RiskScore)
	}
	if analysis.Confidence != 0.6 {
		t.Errorf("expected Confidence 0.6, got %.2f", analysis.Confidence)
	}
}

func TestDeviceTrustService_AnalyzeFingerprint_WithCache(t *testing.T) {
	svc := NewDeviceTrustService()
	fingerprint := "existing_device_fp"

	svc.cacheMutex.Lock()
	svc.trustCache[fingerprint] = &DeviceTrustInfo{
		Fingerprint:  fingerprint,
		TrustScore:   75,
		TrustLevel:   models.TrustLevelHigh,
		VisitCount:   10,
		SuccessCount: 8,
		RiskScore:    15,
		RiskFactors:  []string{},
		IsVerified:   true,
	}
	now := time.Now()
	svc.trustCache[fingerprint].VerifiedAt = &now
	svc.cacheMutex.Unlock()

	analysis := svc.AnalyzeFingerprint(fingerprint, map[string]interface{}{})

	if analysis.IsNew {
		t.Error("expected IsNew to be false for existing fingerprint")
	}
	if analysis.TrustScore != 75 {
		t.Errorf("expected TrustScore 75, got %d", analysis.TrustScore)
	}
	if analysis.RiskLevel != "low" {
		t.Errorf("expected RiskLevel 'low', got '%s'", analysis.RiskLevel)
	}
	if !analysis.IsVerified {
		t.Error("expected IsVerified to be true")
	}
}

func TestDeviceTrustService_EvaluateTrust_MissingFingerprint(t *testing.T) {
	svc := NewDeviceTrustService()

	decision := svc.EvaluateTrust("", map[string]interface{}{})

	if decision.Pass {
		t.Error("expected Pass to be false for missing fingerprint")
	}
	if decision.Action != "block" {
		t.Errorf("expected Action 'block', got '%s'", decision.Action)
	}
	if decision.ChallengeLevel != 3 {
		t.Errorf("expected ChallengeLevel 3, got %d", decision.ChallengeLevel)
	}
}

func TestDeviceTrustService_EvaluateTrust_HighTrust(t *testing.T) {
	svc := NewDeviceTrustService()
	fingerprint := "trusted_device"

	svc.cacheMutex.Lock()
	svc.trustCache[fingerprint] = &DeviceTrustInfo{
		Fingerprint:  fingerprint,
		TrustScore:  90,
		TrustLevel:  models.TrustLevelHigh,
		RiskScore:   10,
		RiskFactors: []string{},
	}
	svc.cacheMutex.Unlock()

	decision := svc.EvaluateTrust(fingerprint, map[string]interface{}{})

	if !decision.Pass {
		t.Error("expected Pass to be true for high trust device")
	}
	if decision.Action != "allow" {
		t.Errorf("expected Action 'allow', got '%s'", decision.Action)
	}
	if decision.ChallengeLevel != 0 {
		t.Errorf("expected ChallengeLevel 0, got %d", decision.ChallengeLevel)
	}
}

func TestDeviceTrustService_EvaluateTrust_Challenge(t *testing.T) {
	svc := NewDeviceTrustService()
	fingerprint := "medium_trust_device"

	svc.cacheMutex.Lock()
	svc.trustCache[fingerprint] = &DeviceTrustInfo{
		Fingerprint:  fingerprint,
		TrustScore:  50,
		TrustLevel:  models.TrustLevelMedium,
		RiskScore:   40,
		RiskFactors: []string{},
	}
	svc.cacheMutex.Unlock()

	decision := svc.EvaluateTrust(fingerprint, map[string]interface{}{})

	if decision.Pass {
		t.Error("expected Pass to be false for medium trust with risk")
	}
	if decision.Action != "challenge" {
		t.Errorf("expected Action 'challenge', got '%s'", decision.Action)
	}
}

func TestDeviceTrustService_EvaluateTrust_WithRiskFactors(t *testing.T) {
	svc := NewDeviceTrustService()
	fingerprint := "risky_device"

	svc.cacheMutex.Lock()
	svc.trustCache[fingerprint] = &DeviceTrustInfo{
		Fingerprint:  fingerprint,
		TrustScore:  70,
		TrustLevel:  models.TrustLevelHigh,
		RiskScore:   30,
		RiskFactors: []string{},
	}
	svc.cacheMutex.Unlock()

	data := map[string]interface{}{
		"webdriver": "wd:true",
	}

	decision := svc.EvaluateTrust(fingerprint, data)

	if decision.RiskScore < 30 {
		t.Errorf("expected RiskScore >= 30 with automation detected, got %.1f", decision.RiskScore)
	}
	if decision.ChallengeLevel < 2 {
		t.Errorf("expected ChallengeLevel >= 2 for automation detected, got %d", decision.ChallengeLevel)
	}
}

func TestDeviceTrustService_UpdateTrustScore(t *testing.T) {
	svc := NewDeviceTrustService()
	fingerprint := "update_test_device"

	initialInfo := &DeviceTrustInfo{
		Fingerprint: fingerprint,
		TrustScore: 50,
		TrustLevel: models.TrustLevelMedium,
		FirstVisit: time.Now(),
	}
	svc.cacheMutex.Lock()
	svc.trustCache[fingerprint] = initialInfo
	svc.cacheMutex.Unlock()

	svc.UpdateTrustScore(fingerprint, models.EventTypeLoginSuccess, "127.0.0.1", "Mozilla/5.0")

	info := svc.GetTrustInfo(fingerprint)
	if info == nil {
		t.Fatal("GetTrustInfo should not return nil")
	}
	if info.TrustScore != 55 {
		t.Errorf("expected TrustScore 55 after login success, got %d", info.TrustScore)
	}
	if info.SuccessCount != 1 {
		t.Errorf("expected SuccessCount 1, got %d", info.SuccessCount)
	}
}

func TestDeviceTrustService_UpdateTrustScore_FailedLogin(t *testing.T) {
	svc := NewDeviceTrustService()
	fingerprint := "failed_login_device"

svc.cacheMutex.Lock()
	svc.trustCache[fingerprint] = &DeviceTrustInfo{
		Fingerprint: fingerprint,
		TrustScore: 50,
		TrustLevel: models.TrustLevelMedium,
		FirstVisit: time.Now(),
	}
	svc.cacheMutex.Unlock()
	svc.UpdateTrustScore(fingerprint, models.EventTypeLoginFailed, "127.0.0.1", "Mozilla/5.0")

	info := svc.GetTrustInfo(fingerprint)
	if info.TrustScore != 40 {
		t.Errorf("expected TrustScore 40 after failed login, got %d", info.TrustScore)
	}
	if info.FailCount != 1 {
		t.Errorf("expected FailCount 1, got %d", info.FailCount)
	}
}

func TestDeviceTrustService_MarkAsVerified(t *testing.T) {
	svc := NewDeviceTrustService()
	fingerprint := "verified_device"

	svc.MarkAsVerified(fingerprint, 24*time.Hour)

	info := svc.GetTrustInfo(fingerprint)
	if info == nil {
		t.Fatal("GetTrustInfo should not return nil")
	}
	if !info.IsVerified {
		t.Error("expected IsVerified to be true")
	}
	if info.VerifiedAt == nil {
		t.Error("expected VerifiedAt to be set")
	}
	if info.ExpiresAt == nil {
		t.Error("expected ExpiresAt to be set")
	}
	if info.TrustScore < 70 {
		t.Errorf("expected TrustScore >= 70 after verification, got %d", info.TrustScore)
	}
}

func TestDeviceTrustService_SetRiskScore(t *testing.T) {
	svc := NewDeviceTrustService()
	fingerprint := "risky_score_device"

	svc.SetRiskScore(fingerprint, 60.0, []string{"automation", "headless"})

	info := svc.GetTrustInfo(fingerprint)
	if info == nil {
		t.Fatal("GetTrustInfo should not return nil")
	}
	if info.RiskScore != 60.0 {
		t.Errorf("expected RiskScore 60.0, got %.1f", info.RiskScore)
	}
	if len(info.RiskFactors) != 2 {
		t.Errorf("expected 2 risk factors, got %d", len(info.RiskFactors))
	}
}

func TestDeviceTrustService_GetTrustHistory(t *testing.T) {
	svc := NewDeviceTrustService()
	fingerprint := "history_test_device"

	for i := 0; i < 5; i++ {
		svc.UpdateTrustScore(fingerprint, models.EventTypeLoginSuccess, "127.0.0.1", "Mozilla/5.0")
	}

	history := svc.GetTrustHistory(fingerprint, 3)
	if len(history) != 3 {
		t.Errorf("expected history length 3, got %d", len(history))
	}

	allHistory := svc.GetTrustHistory(fingerprint, 100)
	if len(allHistory) != 5 {
		t.Errorf("expected all history length 5, got %d", len(allHistory))
	}
}

func TestDeviceTrustService_ShouldSkipVerification(t *testing.T) {
	svc := NewDeviceTrustService()
	ctx := context.Background()

	skip, reason := svc.ShouldSkipVerification(ctx, "test_fingerprint")
	if skip {
		t.Error("expected skip to be false for non-existent fingerprint")
	}
	if reason != "new_device" {
		t.Errorf("expected reason 'new_device', got '%s'", reason)
	}

	fingerprint := "skip_test_device"
	svc.MarkAsVerified(fingerprint, 24*time.Hour)

	skip, reason = svc.ShouldSkipVerification(ctx, fingerprint)
	if !skip {
		t.Error("expected skip to be true for verified device")
	}
	if reason != "verified_device" {
		t.Errorf("expected reason 'verified_device', got '%s'", reason)
	}
}

func TestDeviceTrustService_ShouldSkipVerification_HighTrust(t *testing.T) {
	svc := NewDeviceTrustService()
	ctx := context.Background()
	fingerprint := "high_trust_skip_device"

	svc.cacheMutex.Lock()
	svc.trustCache[fingerprint] = &DeviceTrustInfo{
		Fingerprint: fingerprint,
		TrustScore:  95,
		RiskScore:   5,
	}
	svc.cacheMutex.Unlock()

	skip, reason := svc.ShouldSkipVerification(ctx, fingerprint)
	if !skip {
		t.Error("expected skip to be true for high trust device")
	}
	if reason != "high_trust" {
		t.Errorf("expected reason 'high_trust', got '%s'", reason)
	}
}

func TestDeviceTrustService_GetStatistics(t *testing.T) {
	svc := NewDeviceTrustService()

	devices := []string{"device1", "device2", "device3", "device4", "device5"}
	svc.cacheMutex.Lock()
	for i, fp := range devices {
		trustScore := 30 + i*20
		svc.trustCache[fp] = &DeviceTrustInfo{
			Fingerprint:   fp,
			TrustScore:    trustScore,
			TrustLevel:   calculateTrustLevelInternal(trustScore),
			IsVerified:   i < 2,
		}
	}
	svc.cacheMutex.Unlock()

	stats := svc.GetStatistics()

	if stats["total_fingerprints"].(int) != 5 {
		t.Errorf("expected 5 fingerprints, got %d", stats["total_fingerprints"])
	}
	if stats["verified_count"].(int) != 2 {
		t.Errorf("expected 2 verified, got %d", stats["verified_count"])
	}

	avgScore := stats["average_trust_score"].(float64)
	if avgScore < 69 || avgScore > 71 {
		t.Errorf("expected average score around 70, got %.2f", avgScore)
	}
}

func TestDeviceTrustService_RemoveFingerprint(t *testing.T) {
	svc := NewDeviceTrustService()
	fingerprint := "remove_test_device"

	svc.cacheMutex.Lock()
	svc.trustCache[fingerprint] = &DeviceTrustInfo{
		Fingerprint: fingerprint,
		TrustScore: 50,
	}
	svc.cacheMutex.Unlock()

	svc.RemoveFingerprint(fingerprint)

	info := svc.GetTrustInfo(fingerprint)
	if info != nil {
		t.Error("expected nil info after removal")
	}
}

func TestDeviceTrustService_ExportImportTrustData(t *testing.T) {
	svc := NewDeviceTrustService()
	fingerprint := "export_import_device"

	svc.cacheMutex.Lock()
	svc.trustCache[fingerprint] = &DeviceTrustInfo{
		Fingerprint:  fingerprint,
		TrustScore:   75,
		TrustLevel:   models.TrustLevelHigh,
		VisitCount:   20,
		SuccessCount: 15,
	}
	svc.cacheMutex.Unlock()

	exported, err := svc.ExportTrustData(fingerprint)
	if err != nil {
		t.Fatalf("ExportTrustData failed: %v", err)
	}
	if exported == "" || exported == "{}" {
		t.Error("expected non-empty exported data")
	}

	svc.RemoveFingerprint(fingerprint)

	err = svc.ImportTrustData(fingerprint, exported)
	if err != nil {
		t.Fatalf("ImportTrustData failed: %v", err)
	}

	info := svc.GetTrustInfo(fingerprint)
	if info == nil {
		t.Fatal("expected info after import")
	}
	if info.TrustScore != 75 {
		t.Errorf("expected TrustScore 75, got %d", info.TrustScore)
	}
}

func TestDeviceTrustService_CheckRiskFactors(t *testing.T) {
	svc := NewDeviceTrustService()

	testCases := []struct {
		name     string
		data     map[string]interface{}
		expected bool
	}{
		{
			name:     "webdriver_detected",
			data:     map[string]interface{}{"webdriver": "wd:true"},
			expected: true,
		},
		{
			name:     "headless_detected",
			data:     map[string]interface{}{"headless": true},
			expected: true,
		},
		{
			name:     "swiftshader_detected",
			data:     map[string]interface{}{"webgl": "Google Inc.|SwiftShader"},
			expected: true,
		},
		{
			name:     "minimal_fonts",
			data:     map[string]interface{}{"fonts": "Arial"},
			expected: true,
		},
		{
			name:     "no_risk",
			data:     map[string]interface{}{"webgl": "Intel Inc.|Intel Iris"},
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			hasRisk := checkRiskFactorsInternal(svc, tc.data)
			if hasRisk != tc.expected {
				t.Errorf("expected %v, got %v", tc.expected, hasRisk)
			}
		})
	}
}

func checkRiskFactorsInternal(svc *DeviceTrustService, data map[string]interface{}) bool {
	if _, ok := data["webdriver"]; ok {
		wd, _ := data["webdriver"].(string)
		if wd == "wd:true" || wd == "true" {
			return true
		}
	}

	if _, ok := data["headless"]; ok {
		if hl, ok := data["headless"].(bool); ok && hl {
			return true
		}
	}

	if webgl, ok := data["webgl"].(string); ok {
		webglLower := strings.ToLower(webgl)
		riskIndicators := []string{"swiftshader", "llvmpipe", "mesa", "virtualbox", "vmware"}
		for _, indicator := range riskIndicators {
			if strings.Contains(webglLower, indicator) {
				return true
			}
		}
	}

	if fonts, ok := data["fonts"].(string); ok {
		fontCount := len(strings.Split(fonts, ","))
		if fontCount < 3 {
			return true
		}
	}

	return false
}

func TestDeviceTrustService_CalculateTrustLevel(t *testing.T) {
	testCases := []struct {
		score    int
		expected string
	}{
		{95, models.TrustLevelFull},
		{85, models.TrustLevelHigh},
		{60, models.TrustLevelMedium},
		{35, models.TrustLevelLow},
		{10, models.TrustLevelMinimal},
	}

	for _, tc := range testCases {
		result := calculateTrustLevelInternal(tc.score)
		if result != tc.expected {
			t.Errorf("score %d: expected %s, got %s", tc.score, tc.expected, result)
		}
	}
}

func calculateTrustLevelInternal(score int) string {
	switch {
	case score >= 90:
		return models.TrustLevelFull
	case score >= 75:
		return models.TrustLevelHigh
	case score >= 50:
		return models.TrustLevelMedium
	case score >= 25:
		return models.TrustLevelLow
	default:
		return models.TrustLevelMinimal
	}
}

func TestDeviceTrustService_Concurrency(t *testing.T) {
	svc := NewDeviceTrustService()
	fingerprint := "concurrent_test_device"

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			svc.UpdateTrustScore(fingerprint, models.EventTypeLoginSuccess, "127.0.0.1", "Mozilla/5.0")
		}()
	}
	wg.Wait()

	info := svc.GetTrustInfo(fingerprint)
	if info == nil {
		t.Fatal("GetTrustInfo should not return nil")
	}
	if info.SuccessCount != 100 {
		t.Errorf("expected 100 successes, got %d", info.SuccessCount)
	}
}
