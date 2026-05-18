package service

import (
	"context"
	"testing"
)

func TestDetectHeadlessChrome(t *testing.T) {
	detector := NewEnhancedEnvDetector()
	ctx := context.Background()

	t.Run("detect_headless_chrome_with_webdriver", func(t *testing.T) {
		data := map[string]interface{}{
			"navigator_webdriver":  true,
			"user_agent":          "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 Chrome/91.0.4472.124 HeadlessChrome",
			"webgl_renderer":      "SwiftShader for Chrome",
		}

		detected, err := detector.DetectHeadlessChrome(ctx, data)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if !detected {
			t.Error("Expected headless Chrome to be detected")
		}
	})

	t.Run("detect_headless_chrome_software_renderer", func(t *testing.T) {
		data := map[string]interface{}{
			"webgl_renderer": "llvmpipe software renderer",
			"missing_canvas_data": true,
		}

		detected, err := detector.DetectHeadlessChrome(ctx, data)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if !detected {
			t.Error("Expected headless Chrome with software renderer to be detected")
		}
	})

	t.Run("no_headless_chrome", func(t *testing.T) {
		data := map[string]interface{}{
			"user_agent":    "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 Chrome/91.0.4472.124",
			"webgl_renderer": "Intel Iris OpenGL Engine",
		}

		detected, err := detector.DetectHeadlessChrome(ctx, data)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if detected {
			t.Error("Expected no headless Chrome detection for normal browser")
		}
	})

	t.Run("detect_lighthouse", func(t *testing.T) {
		data := map[string]interface{}{
			"user_agent": "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 Chrome/91.0.4472.124 Chrome-Lighthouse",
		}

		detected, err := detector.DetectHeadlessChrome(ctx, data)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if !detected {
			t.Error("Expected Lighthouse automation to be detected")
		}
	})

	t.Run("nil_data_error", func(t *testing.T) {
		_, err := detector.DetectHeadlessChrome(ctx, nil)
		if err == nil {
			t.Error("Expected error for nil data")
		}
	})
}

func TestDetectPlaywright(t *testing.T) {
	detector := NewEnhancedEnvDetector()
	ctx := context.Background()

	t.Run("detect_playwright_window", func(t *testing.T) {
		data := map[string]interface{}{
			"window_playwright": true,
			"user_agent":        "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 Playwright",
		}

		detected, err := detector.DetectPlaywright(ctx, data)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if !detected {
			t.Error("Expected Playwright to be detected")
		}
	})

	t.Run("detect_playwright_eval", func(t *testing.T) {
		data := map[string]interface{}{
			"window_playwright_eval": true,
			"navigator_webdriver":    true,
		}

		detected, err := detector.DetectPlaywright(ctx, data)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if !detected {
			t.Error("Expected Playwright eval to be detected")
		}
	})

	t.Run("no_playwright", func(t *testing.T) {
		data := map[string]interface{}{
			"user_agent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 Chrome/91.0.4472.124 Safari/537.36",
		}

		detected, err := detector.DetectPlaywright(ctx, data)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if detected {
			t.Error("Expected no Playwright detection for normal browser")
		}
	})

	t.Run("detect_playwright_channel", func(t *testing.T) {
		data := map[string]interface{}{
			"playwright_channel": "chrome",
		}

		detected, err := detector.DetectPlaywright(ctx, data)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if !detected {
			t.Error("Expected Playwright channel to be detected")
		}
	})
}

func TestDetectSelenium(t *testing.T) {
	detector := NewEnhancedEnvDetector()
	ctx := context.Background()

	t.Run("detect_selenium_webdriver", func(t *testing.T) {
		data := map[string]interface{}{
			"webdriver_property": true,
			"user_agent":         "Mozilla/5.0 (Windows NT 10.0; Win64; x64) Gecko/20100101 Firefox/89.0 webdriver",
		}

		detected, err := detector.DetectSelenium(ctx, data)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if !detected {
			t.Error("Expected Selenium WebDriver to be detected")
		}
	})

	t.Run("detect_selenium_chrome_options", func(t *testing.T) {
		data := map[string]interface{}{
			"chrome_options":   true,
			"selenium_property": true,
		}

		detected, err := detector.DetectSelenium(ctx, data)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if !detected {
			t.Error("Expected Selenium with Chrome options to be detected")
		}
	})

	t.Run("no_selenium", func(t *testing.T) {
		data := map[string]interface{}{
			"user_agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 Chrome/91.0.4472.124 Safari/537.36",
		}

		detected, err := detector.DetectSelenium(ctx, data)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if detected {
			t.Error("Expected no Selenium detection for normal browser")
		}
	})

	t.Run("detect_selenium_driver_name", func(t *testing.T) {
		data := map[string]interface{}{
			"driver_name": "chromedriver",
		}

		detected, err := detector.DetectSelenium(ctx, data)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if !detected {
			t.Error("Expected Selenium driver name to be detected")
		}
	})
}

func TestDetectAutomationTools(t *testing.T) {
	detector := NewEnhancedEnvDetector()
	ctx := context.Background()

	t.Run("detect_all_tools", func(t *testing.T) {
		data := map[string]interface{}{
			"user_agent":           "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 Chrome/91.0.4472.124 HeadlessChrome",
			"navigator_webdriver":  true,
			"window_playwright":    true,
			"webdriver_property":   true,
		}

		results, err := detector.DetectAutomationTools(ctx, data)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if results == nil {
			t.Fatal("Expected results to be non-nil")
		}

		detectedCount := 0
		for _, detected := range results {
			if detected {
				detectedCount++
			}
		}

		if detectedCount == 0 {
			t.Error("Expected at least one automation tool to be detected")
		}

		t.Logf("Detected %d automation tools", detectedCount)
	})

	t.Run("no_tools_detected", func(t *testing.T) {
		data := map[string]interface{}{
			"user_agent":    "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 Chrome/91.0.4472.124 Safari/537.36",
			"webgl_renderer": "Apple M1",
		}

		results, err := detector.DetectAutomationTools(ctx, data)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if results == nil {
			t.Fatal("Expected results to be non-nil")
		}

		for tool, detected := range results {
			if detected {
				t.Errorf("Unexpected detection: %s", tool)
			}
		}
	})

	t.Run("detect_puppeteer", func(t *testing.T) {
		data := map[string]interface{}{
			"user_agent":          "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 Chrome/91.0.4472.124 Headless puppeteer",
			"puppeteer_protocol":  true,
		}

		results, err := detector.DetectAutomationTools(ctx, data)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if !results["puppeteer"] {
			t.Error("Expected Puppeteer to be detected")
		}
	})

	t.Run("detect_phantomjs", func(t *testing.T) {
		data := map[string]interface{}{
			"user_agent":            "Mozilla/5.0 (Windows NT 6.1; WOW64) AppleWebKit/538.1 PhantomJS/2.1.1",
			"phantomjs_page_callbacks": true,
		}

		results, err := detector.DetectAutomationTools(ctx, data)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if !results["phantomjs"] {
			t.Error("Expected PhantomJS to be detected")
		}
	})
}

func TestGetAutomationDetectionResult(t *testing.T) {
	detector := NewEnhancedEnvDetector()
	ctx := context.Background()

	t.Run("multiple_tools_detected", func(t *testing.T) {
		data := map[string]interface{}{
			"user_agent":          "Mozilla/5.0 (X11; Linux x86_64) Chrome/91.0.4472.124 HeadlessChrome",
			"navigator_webdriver": true,
			"window_playwright":   true,
		}

		result := detector.GetAutomationDetectionResult(ctx, data)

		if result == nil {
			t.Fatal("Expected result to be non-nil")
		}

		if !result.Detected {
			t.Error("Expected automation to be detected")
		}

		if result.Confidence < 50 {
			t.Errorf("Expected high confidence for multiple tools, got %.2f", result.Confidence)
		}

		if len(result.Evidence) == 0 {
			t.Error("Expected evidence to be present")
		}

		t.Logf("Detected tool: %s, Confidence: %.2f, Severity: %s", result.ToolName, result.Confidence, result.Severity)
	})

	t.Run("single_tool_detected", func(t *testing.T) {
		data := map[string]interface{}{
			"user_agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) Chrome/91.0.4472.124 webdriver",
		}

		result := detector.GetAutomationDetectionResult(ctx, data)

		if result == nil {
			t.Fatal("Expected result to be non-nil")
		}

		if result.Severity != "medium" {
			t.Errorf("Expected severity to be 'medium' for single tool, got %s", result.Severity)
		}
	})

	t.Run("no_automation", func(t *testing.T) {
		data := map[string]interface{}{
			"user_agent":    "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 Safari/537.36",
			"webgl_renderer": "Apple GPU",
		}

		result := detector.GetAutomationDetectionResult(ctx, data)

		if result == nil {
			t.Fatal("Expected result to be non-nil")
		}

		if result.Detected {
			t.Error("Expected no automation detection")
		}
	})
}

func TestAnalyzeAllAutomationIndicators(t *testing.T) {
	detector := NewEnhancedEnvDetector()
	ctx := context.Background()

	t.Run("comprehensive_analysis", func(t *testing.T) {
		data := map[string]interface{}{
			"user_agent":          "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 Chrome/91.0.4472.124 HeadlessChrome",
			"navigator_webdriver": true,
			"webgl_renderer":      "SwiftShader",
		}

		analysis := detector.AnalyzeAllAutomationIndicators(ctx, data)

		if analysis == nil {
			t.Fatal("Expected analysis to be non-nil")
		}

		if _, ok := analysis["headless_chrome_detected"]; !ok {
			t.Error("Expected headless_chrome_detected in analysis")
		}

		if _, ok := analysis["all_tools"]; !ok {
			t.Error("Expected all_tools in analysis")
		}

		if _, ok := analysis["overall_automation_score"]; !ok {
			t.Error("Expected overall_automation_score in analysis")
		}

		score := analysis["overall_automation_score"].(float64)
		if score < 0 {
			t.Error("Expected score to be non-negative")
		}

		t.Logf("Overall automation score: %.2f", score)
	})

	t.Run("normal_browser_analysis", func(t *testing.T) {
		data := map[string]interface{}{
			"user_agent":    "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 Chrome/91.0.4472.124 Safari/537.36",
			"webgl_renderer": "NVIDIA GeForce GTX 1080",
			"webgl_vendor":  "NVIDIA Corporation",
		}

		analysis := detector.AnalyzeAllAutomationIndicators(ctx, data)

		if analysis == nil {
			t.Fatal("Expected analysis to be non-nil")
		}

		score := analysis["overall_automation_score"].(float64)
		if score > 10 {
			t.Errorf("Expected low score for normal browser, got %.2f", score)
		}
	})
}

func TestEnhancedEnvironmentDetection(t *testing.T) {
	detection := NewEnhancedEnvironmentDetection()
	ctx := context.Background()

	t.Run("detect_headless_chrome_via_service", func(t *testing.T) {
		data := map[string]interface{}{
			"navigator_webdriver": true,
			"user_agent":         "Mozilla/5.0 Chrome/91.0 HeadlessChrome",
		}

		detected, err := detection.DetectHeadlessChrome(ctx, data)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if !detected {
			t.Error("Expected headless Chrome detection via service")
		}
	})

	t.Run("detect_playwright_via_service", func(t *testing.T) {
		data := map[string]interface{}{
			"window_playwright": true,
		}

		detected, err := detection.DetectPlaywright(ctx, data)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if !detected {
			t.Error("Expected Playwright detection via service")
		}
	})

	t.Run("detect_selennium_via_service", func(t *testing.T) {
		data := map[string]interface{}{
			"selenium_property": true,
		}

		detected, err := detection.DetectSelenium(ctx, data)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if !detected {
			t.Error("Expected Selenium detection via service")
		}
	})

	t.Run("detect_all_tools_via_service", func(t *testing.T) {
		data := map[string]interface{}{
			"user_agent":  "Mozilla/5.0 Playwright Headless Chrome",
			"navigator_webdriver": true,
		}

		results, err := detection.DetectAutomationTools(ctx, data)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if results == nil {
			t.Fatal("Expected results to be non-nil")
		}

		if len(results) == 0 {
			t.Error("Expected at least one tool in results")
		}
	})

	t.Run("get_detection_result", func(t *testing.T) {
		data := map[string]interface{}{
			"user_agent":          "Mozilla/5.0 Chrome/91.0 Headless",
			"navigator_webdriver": true,
		}

		result := detection.GetDetectionResult(ctx, data)
		if result == nil {
			t.Fatal("Expected result to be non-nil")
		}
	})

	t.Run("analyze_all", func(t *testing.T) {
		data := map[string]interface{}{
			"user_agent":    "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 Chrome/91.0",
			"webgl_renderer": "Intel Iris",
		}

		analysis := detection.AnalyzeAll(ctx, data)
		if analysis == nil {
			t.Fatal("Expected analysis to be non-nil")
		}
	})
}

func BenchmarkDetectHeadlessChrome(b *testing.B) {
	detector := NewEnhancedEnvDetector()
	ctx := context.Background()

	data := map[string]interface{}{
		"navigator_webdriver":     true,
		"webgl_software_renderer": true,
		"user_agent":              "Mozilla/5.0 Chrome/91.0 HeadlessChrome",
		"webgl_renderer":          "SwiftShader",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = detector.DetectHeadlessChrome(ctx, data)
	}
}

func BenchmarkDetectPlaywright(b *testing.B) {
	detector := NewEnhancedEnvDetector()
	ctx := context.Background()

	data := map[string]interface{}{
		"window_playwright":     true,
		"navigator_webdriver":  true,
		"user_agent":            "Mozilla/5.0 Playwright Chrome",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = detector.DetectPlaywright(ctx, data)
	}
}

func BenchmarkDetectSelenium(b *testing.B) {
	detector := NewEnhancedEnvDetector()
	ctx := context.Background()

	data := map[string]interface{}{
		"webdriver_property": true,
		"user_agent":         "Mozilla/5.0 webdriver Chrome",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = detector.DetectSelenium(ctx, data)
	}
}

func BenchmarkDetectAutomationTools(b *testing.B) {
	detector := NewEnhancedEnvDetector()
	ctx := context.Background()

	data := map[string]interface{}{
		"user_agent":           "Mozilla/5.0 Chrome/91.0 Headless",
		"navigator_webdriver":  true,
		"window_playwright":    true,
		"webdriver_property":   true,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = detector.DetectAutomationTools(ctx, data)
	}
}
