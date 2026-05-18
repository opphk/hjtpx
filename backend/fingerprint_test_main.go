package main

import (
	"fmt"
	"time"

	"github.com/hjtpx/hjtpx/internal/service"
)

func main() {
	fmt.Println("Testing Canvas Similarity Analyzer...")
	db := service.NewFingerprintDatabase()
	canvasAnalyzer := service.NewCanvasSimilarityAnalyzer(db)

	// Test canvas similarity
	similarity := canvasAnalyzer.CalculateCanvasSimilarity("abc123", "abc123")
	fmt.Printf("Canvas similarity (same): %.2f\n", similarity)

	similarity = canvasAnalyzer.CalculateCanvasSimilarity("abc123", "def456")
	fmt.Printf("Canvas similarity (different): %.2f\n", similarity)

	// Test histogram similarity
	similarity = canvasAnalyzer.CalculateHistogramSimilarity("aabbcc", "aabbcc")
	fmt.Printf("Histogram similarity (same): %.2f\n", similarity)

	// Test hash stability
	samples := []string{"abc123", "abc123", "abc123"}
	stability := canvasAnalyzer.AnalyzeHashStability(samples)
	fmt.Printf("Hash stability: stable=%v, score=%.2f\n", stability.IsStable, stability.StabilityScore)

	// Test tampering detection
	tampering := canvasAnalyzer.DetectHashTampering("", 0)
	fmt.Printf("Empty hash tampering: %v\n", tampering.IsTampered)

	tampering = canvasAnalyzer.DetectHashTampering("abc123", 6)
	fmt.Printf("Valid hash tampering: %v\n", tampering.IsTampered)

	fmt.Println("\nTesting WebGL Analyzer...")
	webglAnalyzer := service.NewWebGLAnalyzer(db)
	data := map[string]interface{}{
		"webgl_vendor":        "NVIDIA Corporation",
		"webgl_unmasked_vendor": "NVIDIA Corporation",
		"webgl_renderer":       "NVIDIA GeForce GTX 1080",
		"webgl_unmasked_renderer": "NVIDIA GeForce GTX 1080",
		"webgl_extensions": []interface{}{
			"GL_EXT_blend_minmax",
			"GL_EXT_color_buffer_float",
		},
		"webgl_max_texture_size":    float64(8192),
		"webgl_max_vertex_attribs": float64(16),
	}
	webglResult := webglAnalyzer.AnalyzeWebGLFingerprint(data)
	fmt.Printf("WebGL tampered: %v, score=%.2f\n", webglResult.IsTampered, webglResult.TamperingScore)

	fmt.Println("\nTesting Font Analyzer...")
	fontAnalyzer := service.NewFontAnalyzer(db)
	fontData := map[string]interface{}{
		"detected_fonts": []interface{}{"Arial", "Helvetica", "Times New Roman"},
		"platform":       "windows",
	}
	fontResult := fontAnalyzer.AnalyzeFontFingerprint(fontData)
	fmt.Printf("Font tampered: %v, confidence=%.2f\n", fontResult.IsTampered, fontResult.Confidence)

	fmt.Println("\nTesting Stability Analyzer...")
	stabilityAnalyzer := service.NewFingerprintStabilityAnalyzer(db)
	
	fp1 := &service.FingerprintAnalysis{
		FingerprintID:    "test_fp",
		CanvasHash:       "canvas1",
		WebGLHash:        "webgl1",
		UserAgent:        "Mozilla/5.0",
		ScreenResolution: "1920x1080",
		FirstSeen:        time.Now(),
		LastSeen:         time.Now(),
	}
	stabilityAnalyzer.TrackFingerprint(fp1)
	
	fp2 := &service.FingerprintAnalysis{
		FingerprintID:    "test_fp",
		CanvasHash:       "canvas1",
		WebGLHash:        "webgl1",
		UserAgent:        "Mozilla/5.0",
		ScreenResolution: "1920x1080",
		FirstSeen:        time.Now(),
		LastSeen:         time.Now(),
	}
	stabilityAnalyzer.TrackFingerprint(fp2)
	
	stabResult := stabilityAnalyzer.AnalyzeStability("test_fp")
	fmt.Printf("Stability: stable=%v, score=%.2f, avg_similarity=%.2f\n", 
		stabResult.IsStable, stabResult.StabilityScore, stabResult.AverageSimilarity)

	fmt.Println("\nAll tests passed!")
}
