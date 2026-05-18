package service

import (
	"testing"
	"time"
)

func TestCanvasSimilarityAnalyzer(t *testing.T) {
	db := NewFingerprintDatabase()
	analyzer := NewCanvasSimilarityAnalyzer(db)

	t.Run("CalculateCanvasSimilarity", func(t *testing.T) {
		hash1 := "abc123"
		hash2 := "abc123"
		similarity := analyzer.CalculateCanvasSimilarity(hash1, hash2)
		if similarity != 100.0 {
			t.Errorf("Expected similarity 100.0, got %.2f", similarity)
		}

		hash3 := "abc123"
		hash4 := "def456"
		similarity = analyzer.CalculateCanvasSimilarity(hash3, hash4)
		if similarity != 0 {
			t.Errorf("Expected similarity 0, got %.2f", similarity)
		}
	})

	t.Run("CalculateHistogramSimilarity", func(t *testing.T) {
		hash1 := "aabbcc"
		hash2 := "aabbcc"
		similarity := analyzer.CalculateHistogramSimilarity(hash1, hash2)
		if similarity != 100.0 {
			t.Errorf("Expected similarity 100.0, got %.2f", similarity)
		}
	})

	t.Run("CalculateEnhancedSimilarity", func(t *testing.T) {
		hash1 := "abc123"
		hash2 := "abc123"
		similarity := analyzer.CalculateEnhancedSimilarity(hash1, hash2)
		if similarity != 100.0 {
			t.Errorf("Expected similarity 100.0, got %.2f", similarity)
		}
	})

	t.Run("AnalyzeHashStability", func(t *testing.T) {
		samples := []string{"abc123", "abc123", "abc123"}
		result := analyzer.AnalyzeHashStability(samples)
		if !result.IsStable {
			t.Error("Expected stable result")
		}
		if result.StabilityScore < 95 {
			t.Errorf("Expected stability score >= 95, got %.2f", result.StabilityScore)
		}
	})

	t.Run("DetectHashTampering", func(t *testing.T) {
		result := analyzer.DetectHashTampering("", 0)
		if !result.IsTampered {
			t.Error("Expected tampered result for empty hash")
		}

		result = analyzer.DetectHashTampering("abc123", 6)
		if result.IsTampered {
			t.Error("Expected not tampered for valid hash")
		}
	})
}

func TestWebGLAnalyzer(t *testing.T) {
	db := NewFingerprintDatabase()
	analyzer := NewWebGLAnalyzer(db)

	t.Run("AnalyzeWebGLFingerprint", func(t *testing.T) {
		data := map[string]interface{}{
			"webgl_vendor":       "NVIDIA Corporation",
			"webgl_unmasked_vendor": "NVIDIA Corporation",
			"webgl_renderer":      "NVIDIA GeForce GTX 1080",
			"webgl_unmasked_renderer": "NVIDIA GeForce GTX 1080",
			"webgl_extensions": []interface{}{
				"GL_EXT_blend_minmax",
				"GL_EXT_color_buffer_float",
			},
			"webgl_max_texture_size": float64(8192),
			"webgl_max_vertex_attribs": float64(16),
		}

		result := analyzer.AnalyzeWebGLFingerprint(data)
		if result.IsTampered {
			t.Errorf("Expected not tampered, got tampering score %.2f", result.TamperingScore)
		}
	})

	t.Run("AnalyzeWebGLFingerprintWithSoftwareRenderer", func(t *testing.T) {
		data := map[string]interface{}{
			"webgl_vendor": "Google Inc.",
			"webgl_renderer": "SwiftShader",
		}

		result := analyzer.AnalyzeWebGLFingerprint(data)
		if !result.RendererAnalysis.IsSoftwareRenderer {
			t.Error("Expected software renderer detection")
		}
	})

	t.Run("CompareWebGLFingerprints", func(t *testing.T) {
		fp1 := &WebGLMetrics{
			Vendor:    "NVIDIA",
			Renderer:  "GeForce",
			MaxTextureSize: 8192,
		}
		fp2 := &WebGLMetrics{
			Vendor:    "NVIDIA",
			Renderer:  "GeForce",
			MaxTextureSize: 8192,
		}

		similarity := analyzer.CompareWebGLFingerprints(fp1, fp2)
		if similarity < 80 {
			t.Errorf("Expected similarity >= 80, got %.2f", similarity)
		}
	})
}

func TestFontAnalyzer(t *testing.T) {
	db := NewFingerprintDatabase()
	analyzer := NewFontAnalyzer(db)

	t.Run("AnalyzeFontFingerprint", func(t *testing.T) {
		data := map[string]interface{}{
			"detected_fonts": []interface{}{
				"Arial",
				"Helvetica",
				"Times New Roman",
				"Verdana",
				"Georgia",
			},
			"platform": "windows",
		}

		result := analyzer.AnalyzeFontFingerprint(data)
		if result.IsTampered {
			t.Errorf("Expected not tampered, got confidence %.2f", result.Confidence)
		}
		if result.FontAnalysis.CommonFontCount < 3 {
			t.Errorf("Expected at least 3 common fonts, got %d", result.FontAnalysis.CommonFontCount)
		}
	})

	t.Run("AnalyzeFontFingerprintWithFewFonts", func(t *testing.T) {
		data := map[string]interface{}{
			"detected_fonts": []interface{}{"Arial"},
			"platform":       "windows",
		}

		result := analyzer.AnalyzeFontFingerprint(data)
		if !result.FontAnalysis.IsLimitedFontSet {
			t.Error("Expected limited font set")
		}
	})

	t.Run("CompareFontFingerprints", func(t *testing.T) {
		fp1 := &FontMetrics{
			Hash:          "hash1",
			DetectedFonts: []string{"Arial", "Helvetica"},
			FontCount:     2,
		}
		fp2 := &FontMetrics{
			Hash:          "hash1",
			DetectedFonts: []string{"Arial", "Helvetica"},
			FontCount:     2,
		}

		similarity := analyzer.CompareFontFingerprints(fp1, fp2)
		if similarity != 100.0 {
			t.Errorf("Expected similarity 100.0, got %.2f", similarity)
		}
	})
}

func TestFingerprintStabilityAnalyzer(t *testing.T) {
	db := NewFingerprintDatabase()
	analyzer := NewFingerprintStabilityAnalyzer(db)

	t.Run("AnalyzeStability", func(t *testing.T) {
		fp1 := &FingerprintAnalysis{
			FingerprintID:   "fp1",
			CanvasHash:      "canvas1",
			WebGLHash:       "webgl1",
			UserAgent:       "Mozilla/5.0",
			ScreenResolution: "1920x1080",
			FirstSeen:       time.Now(),
			LastSeen:        time.Now(),
		}

		fp2 := &FingerprintAnalysis{
			FingerprintID:   "fp1",
			CanvasHash:      "canvas1",
			WebGLHash:       "webgl1",
			UserAgent:       "Mozilla/5.0",
			ScreenResolution: "1920x1080",
			FirstSeen:       time.Now(),
			LastSeen:        time.Now(),
		}

		analyzer.TrackFingerprint(fp1)
		analyzer.TrackFingerprint(fp2)

		result := analyzer.AnalyzeStability("fp1")
		if !result.IsStable {
			t.Errorf("Expected stable, got stability score %.2f", result.StabilityScore)
		}
	})

	t.Run("AnalyzeStabilityWithInsufficientSamples", func(t *testing.T) {
		result := analyzer.AnalyzeStability("nonexistent")
		if !result.InsufficientSamples {
			t.Error("Expected insufficient samples")
		}
	})

	t.Run("AnalyzeTemporalStability", func(t *testing.T) {
		for i := 0; i < 6; i++ {
			fp := &FingerprintAnalysis{
				FingerprintID: "fp2",
				CanvasHash:    "canvas2",
				WebGLHash:     "webgl2",
				UserAgent:     "Mozilla/5.0",
				FirstSeen:     time.Now(),
				LastSeen:      time.Now(),
			}
			analyzer.TrackFingerprint(fp)
			time.Sleep(time.Millisecond)
		}

		result := analyzer.AnalyzeTemporalStability("fp2")
		if !result.IsStable {
			t.Errorf("Expected stable, got overall score %.2f", result.OverallScore)
		}
	})

	t.Run("DetectStabilityAnomalies", func(t *testing.T) {
		fp1 := &FingerprintAnalysis{
			FingerprintID: "fp3",
			CanvasHash:    "canvasA",
			WebGLHash:     "webglA",
			FirstSeen:     time.Now(),
			LastSeen:      time.Now(),
		}
		analyzer.TrackFingerprint(fp1)

		fp2 := &FingerprintAnalysis{
			FingerprintID: "fp3",
			CanvasHash:    "canvasB",
			WebGLHash:     "webglB",
			FirstSeen:     time.Now(),
			LastSeen:      time.Now(),
		}
		analyzer.TrackFingerprint(fp2)

		anomalies := analyzer.DetectStabilityAnomalies("fp3")
		if len(anomalies) > 0 {
			t.Errorf("Expected no anomalies with <5 samples")
		}
	})
}

func TestWebGLAnalyzerEdgeCases(t *testing.T) {
	db := NewFingerprintDatabase()
	analyzer := NewWebGLAnalyzer(db)

	t.Run("MissingVendor", func(t *testing.T) {
		data := map[string]interface{}{
			"webgl_renderer": "NVIDIA",
		}
		result := analyzer.AnalyzeWebGLFingerprint(data)
		if len(result.Errors) == 0 {
			t.Error("Expected error for missing vendor")
		}
	})

	t.Run("BlacklistedVendor", func(t *testing.T) {
		data := map[string]interface{}{
			"webgl_vendor": "fake vendor",
			"webgl_renderer": "test renderer",
		}
		result := analyzer.AnalyzeWebGLFingerprint(data)
		if !result.IsTampered {
			t.Error("Expected tampered for blacklisted vendor pattern")
		}
	})
}

func TestFontAnalyzerEdgeCases(t *testing.T) {
	db := NewFingerprintDatabase()
	analyzer := NewFontAnalyzer(db)

	t.Run("NoCommonFonts", func(t *testing.T) {
		data := map[string]interface{}{
			"detected_fonts": []interface{}{"SomeRareFont"},
		}
		result := analyzer.AnalyzeFontFingerprint(data)
		if len(result.Errors) == 0 {
			t.Error("Expected error for no common fonts")
		}
	})

	t.Run("LowPlatformMatch", func(t *testing.T) {
		data := map[string]interface{}{
			"detected_fonts": []interface{}{"Arial"},
			"platform":       "windows",
		}
		result := analyzer.AnalyzeFontFingerprint(data)
		if result.PlatformMatch.MatchRatio >= 0.3 {
			t.Errorf("Expected low platform match ratio, got %.2f", result.PlatformMatch.MatchRatio)
		}
	})
}

func TestCanvasSimilarityAnalyzerEdgeCases(t *testing.T) {
	db := NewFingerprintDatabase()
	analyzer := NewCanvasSimilarityAnalyzer(db)

	t.Run("EmptyHash", func(t *testing.T) {
		similarity := analyzer.CalculateCanvasSimilarity("", "abc123")
		if similarity != 0 {
			t.Errorf("Expected 0 similarity for empty hash, got %.2f", similarity)
		}
	})

	t.Run("NonHexHash", func(t *testing.T) {
		result := analyzer.DetectHashTampering("gibberish!", 0)
		if !result.IsTampered {
			t.Error("Expected tampered for non-hex hash")
		}
	})

	t.Run("LowEntropyHash", func(t *testing.T) {
		result := analyzer.DetectHashTampering("aaaaaaaaaaaaaaaa", 16)
		if !result.IsTampered {
			t.Error("Expected tampered for low entropy hash")
		}
	})
}

func TestStabilityAnalyzerEdgeCases(t *testing.T) {
	db := NewFingerprintDatabase()
	analyzer := NewFingerprintStabilityAnalyzer(db)

	t.Run("FingerprintDrift", func(t *testing.T) {
		for i := 0; i < 5; i++ {
			fp := &FingerprintAnalysis{
				FingerprintID: "fp_drift",
				CanvasHash:    string(rune('a'+i)) + "bc123",
				WebGLHash:     "webgl1",
				FirstSeen:     time.Now(),
				LastSeen:      time.Now(),
			}
			analyzer.TrackFingerprint(fp)
			time.Sleep(time.Millisecond)
		}

		anomalies := analyzer.DetectStabilityAnomalies("fp_drift")
		if len(anomalies) == 0 {
			t.Error("Expected anomalies for fingerprint drift")
		}
	})

	t.Run("SuspiciouslyConsistent", func(t *testing.T) {
		for i := 0; i < 11; i++ {
			fp := &FingerprintAnalysis{
				FingerprintID: "fp_consistent",
				CanvasHash:    "samehash",
				WebGLHash:     "samewebgl",
				UserAgent:     "sameua",
				FirstSeen:     time.Now(),
				LastSeen:      time.Now(),
			}
			analyzer.TrackFingerprint(fp)
			time.Sleep(time.Millisecond)
		}

		result := analyzer.AnalyzeStability("fp_consistent")
		if len(result.Warnings) == 0 {
			t.Error("Expected warning for suspiciously consistent fingerprints")
		}
	})
}