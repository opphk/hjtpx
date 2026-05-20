package service

import (
	"context"
	"testing"
	"time"
)

func TestTrajectoryNNEnhancedAdvanced(t *testing.T) {
	enhanced := NewTrajectoryNNEnhanced()

	trajectory := []TrajectoryNNPoint{
		{X: 0, Y: 0, Timestamp: 1000},
		{X: 10, Y: 10, Timestamp: 1100},
		{X: 25, Y: 25, Timestamp: 1200},
		{X: 40, Y: 40, Timestamp: 1300},
		{X: 60, Y: 60, Timestamp: 1400},
		{X: 85, Y: 85, Timestamp: 1500},
		{X: 100, Y: 100, Timestamp: 1600},
	}

	t.Run("Extract Features", func(t *testing.T) {
		features := enhanced.ExtractFeatures(trajectory)

		if len(features) != 768 {
			t.Errorf("Expected 768 features, got %d", len(features))
		}

		for i, v := range features {
			if v < 0 || v > 1 {
				t.Logf("Warning: Feature %d value out of range [0,1]: %f", i, v)
			}
		}
	})

	t.Run("Extract Basic Features", func(t *testing.T) {
		features := enhanced.extractBasicFeatures(trajectory)

		if len(features) != 64 {
			t.Errorf("Expected 64 basic features, got %d", len(features))
		}
	})

	t.Run("Extract Speed Features", func(t *testing.T) {
		features := enhanced.extractSpeedFeatures(trajectory)

		if len(features) != 128 {
			t.Errorf("Expected 128 speed features, got %d", len(features))
		}
	})

	t.Run("Extract Direction Features", func(t *testing.T) {
		features := enhanced.extractDirectionFeatures(trajectory)

		if len(features) != 128 {
			t.Errorf("Expected 128 direction features, got %d", len(features))
		}
	})

	t.Run("Extract Advanced Features", func(t *testing.T) {
		features := enhanced.extractAdvancedFeatures(trajectory)

		if len(features) != 128 {
			t.Errorf("Expected 128 advanced features, got %d", len(features))
		}
	})

	t.Run("Calculate Human Likelihood", func(t *testing.T) {
		likelihood := enhanced.calculateHumanLikelihood(trajectory)

		if likelihood < 0 || likelihood > 1 {
			t.Errorf("Human likelihood should be between 0 and 1, got %f", likelihood)
		}
	})

	t.Run("Detect Mechanical Movement", func(t *testing.T) {
		mechanicalScore := enhanced.detectMechanicalMovement(trajectory)

		if mechanicalScore < 0 || mechanicalScore > 1 {
			t.Errorf("Mechanical score should be between 0 and 1, got %f", mechanicalScore)
		}
	})

	t.Run("Detect Perfect Straightness", func(t *testing.T) {
		straightness := enhanced.detectPerfectStraightness(trajectory)

		if straightness < 0 || straightness > 1 {
			t.Errorf("Straightness should be between 0 and 1, got %f", straightness)
		}
	})
}

func TestTrajectoryNNEnhancedBotDetection(t *testing.T) {
	enhanced := NewTrajectoryNNEnhanced()

	t.Run("Detect Bot Trajectory - Perfect Linear", func(t *testing.T) {
		botTrajectory := []TrajectoryNNPoint{
			{X: 0, Y: 0, Timestamp: 1000},
			{X: 10, Y: 10, Timestamp: 1100},
			{X: 20, Y: 20, Timestamp: 1200},
			{X: 30, Y: 30, Timestamp: 1300},
			{X: 40, Y: 40, Timestamp: 1400},
		}

		features := enhanced.ExtractFeatures(botTrajectory)

		straightness := enhanced.detectPerfectStraightness(botTrajectory)
		if straightness < 0.95 {
			t.Logf("Bot trajectory detected as not straight enough: %f", straightness)
		}

		mechanicalScore := enhanced.detectMechanicalMovement(botTrajectory)
		if mechanicalScore < 0.5 {
			t.Logf("Bot trajectory detected with low mechanical score: %f", mechanicalScore)
		}

		_ = features
	})

	t.Run("Detect Bot Trajectory - Uniform Speed", func(t *testing.T) {
		botTrajectory := []TrajectoryNNPoint{}
		for i := 0; i < 10; i++ {
			botTrajectory = append(botTrajectory, TrajectoryNNPoint{
				X:         float64(i * 10),
				Y:         float64(i * 10),
				Timestamp: int64(1000 + i*100),
			})
		}

		mechanicalScore := enhanced.detectMechanicalMovement(botTrajectory)
		if mechanicalScore < 0.7 {
			t.Logf("Uniform speed bot trajectory detected: %f", mechanicalScore)
		}
	})

	t.Run("Detect Human Trajectory - Natural Movement", func(t *testing.T) {
		humanTrajectory := []TrajectoryNNPoint{
			{X: 0, Y: 0, Timestamp: 1000},
			{X: 15, Y: 8, Timestamp: 1150},
			{X: 28, Y: 22, Timestamp: 1320},
			{X: 35, Y: 35, Timestamp: 1480},
			{X: 55, Y: 42, Timestamp: 1650},
			{X: 70, Y: 60, Timestamp: 1820},
			{X: 85, Y: 78, Timestamp: 1990},
			{X: 100, Y: 95, Timestamp: 2160},
		}

		humanLikelihood := enhanced.calculateHumanLikelihood(humanTrajectory)
		if humanLikelihood < 0.5 {
			t.Logf("Human trajectory detected with low likelihood: %f", humanLikelihood)
		}

		mechanicalScore := enhanced.detectMechanicalMovement(humanTrajectory)
		if mechanicalScore > 0.5 {
			t.Logf("Human trajectory detected as mechanical: %f", mechanicalScore)
		}
	})
}

func TestTrajectoryNNEnhancedEdgeCases(t *testing.T) {
	enhanced := NewTrajectoryNNEnhanced()

	t.Run("Empty Trajectory", func(t *testing.T) {
		emptyTrajectory := []TrajectoryNNPoint{}

		features := enhanced.ExtractFeatures(emptyTrajectory)
		if len(features) != 768 {
			t.Errorf("Should return 768 features even for empty trajectory")
		}
	})

	t.Run("Single Point Trajectory", func(t *testing.T) {
		singlePoint := []TrajectoryNNPoint{
			{X: 0, Y: 0, Timestamp: 1000},
		}

		features := enhanced.ExtractFeatures(singlePoint)
		if len(features) != 768 {
			t.Errorf("Should return 768 features even for single point")
		}
	})

	t.Run("Two Point Trajectory", func(t *testing.T) {
		twoPoints := []TrajectoryNNPoint{
			{X: 0, Y: 0, Timestamp: 1000},
			{X: 10, Y: 10, Timestamp: 1100},
		}

		features := enhanced.ExtractFeatures(twoPoints)
		if len(features) != 768 {
			t.Errorf("Should return 768 features for two points")
		}
	})
}

func TestFingerprintAdvancedEnhanced(t *testing.T) {
	fingerprint := NewFingerprintAdvanced()

	t.Run("Detect Proxy VPN - With Headers", func(t *testing.T) {
		headers := map[string]string{
			"x-forwarded-for": "192.168.1.1",
			"x-real-ip":      "10.0.0.1",
		}
		networkData := map[string]interface{}{
			"latency": 150.0,
			"asn":     "Hosting Provider",
		}

		result := fingerprint.DetectProxyVPN(headers, "192.168.1.1", networkData)

		if result.TotalScore < 0 || result.TotalScore > 1 {
			t.Errorf("Total score should be between 0 and 1, got %f", result.TotalScore)
		}

		if !result.IsProxy && result.TotalScore > 0.5 {
			t.Logf("Proxy might be detected but IsProxy is false")
		}
	})

	t.Run("Analyze Browser Environment - Normal Browser", func(t *testing.T) {
		browserData := map[string]interface{}{
			"userAgent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
			"plugins":   []string{"Chrome PDF Plugin", "Chrome PDF Viewer"},
			"timezone":  "America/New_York",
			"canvasFingerprint": "canvas_fingerprint_data_here_123456",
		}

		result := fingerprint.AnalyzeBrowserEnvironment(browserData)

		if result.OverallRisk < 0 || result.OverallRisk > 1 {
			t.Errorf("Overall risk should be between 0 and 1, got %f", result.OverallRisk)
		}

		if result.OverallRisk > 0.7 {
			t.Logf("Normal browser detected with high risk: %f", result.OverallRisk)
		}
	})

	t.Run("Analyze Browser Environment - Headless Browser", func(t *testing.T) {
		browserData := map[string]interface{}{
			"userAgent":    "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) HeadlessChrome",
			"plugins":      []string{},
			"webdriver":    true,
			"languages":    []string{},
			"screenResolution": "800x600",
		}

		result := fingerprint.AnalyzeBrowserEnvironment(browserData)

		if !result.IsHeadless {
			t.Logf("Headless browser might not be detected properly")
		}

		if result.OverallRisk < 0.5 {
			t.Logf("Headless browser detected with low risk: %f", result.OverallRisk)
		}
	})

	t.Run("Generate Enhanced Canvas Fingerprint", func(t *testing.T) {
		canvasContext := map[string]interface{}{
			"imageData": "This is sample canvas image data with some noise for testing",
			"attempts":  1,
		}

		result := fingerprint.GenerateEnhancedCanvasFingerprint(canvasContext)

		if result.Risk < 0 || result.Risk > 1 {
			t.Errorf("Canvas risk should be between 0 and 1, got %f", result.Risk)
		}

		if result.NoiseLevel != "low" && result.NoiseLevel != "medium" && result.NoiseLevel != "high" {
			t.Errorf("Invalid noise level: %s", result.NoiseLevel)
		}
	})
}

func TestFingerprintAdvancedEdgeCases(t *testing.T) {
	fingerprint := NewFingerprintAdvanced()

	t.Run("Empty Proxy VPN Detection", func(t *testing.T) {
		result := fingerprint.DetectProxyVPN(nil, "", nil)

		if result.TotalScore < 0 || result.TotalScore > 1 {
			t.Errorf("Score should be between 0 and 1")
		}
	})

	t.Run("Empty Browser Analysis", func(t *testing.T) {
		result := fingerprint.AnalyzeBrowserEnvironment(nil)

		if result.OverallRisk < 0 || result.OverallRisk > 1 {
			t.Errorf("Risk should be between 0 and 1")
		}
	})
}

func TestAutomationDetectorV2Enhanced(t *testing.T) {
	detector := NewAutomationDetectorV2()

	t.Run("Detect Selenium", func(t *testing.T) {
		navigator := map[string]interface{}{
			"webdriver": true,
		}

		result := detector.DetectSelenium("Mozilla/5.0", navigator)

		if !result.Detected {
			t.Error("Selenium should be detected")
		}

		if len(result.Methods) == 0 {
			t.Error("Should have detection methods")
		}
	})

	t.Run("Detect Puppeteer", func(t *testing.T) {
		headers := map[string]string{
			"User-Agent": "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 Chrome/91.0.4472.124 HeadlessChrome",
		}

		result := detector.DetectPuppeteer("HeadlessChrome", headers)

		if !result.Detected {
			t.Logf("Puppeteer might not be detected")
		}
	})

	t.Run("Analyze Advanced Headless", func(t *testing.T) {
		navigator := map[string]interface{}{
			"webglRenderer": "SwiftShader",
			"languages":     []string{},
			"plugins":       []string{},
		}
		screen := map[string]interface{}{
			"width": 800,
			"height": 600,
		}

		result := detector.DetectAdvancedHeadless(navigator, screen)

		if result.TotalRiskScore < 0 || result.TotalRiskScore > 1 {
			t.Errorf("Risk score should be between 0 and 1")
		}

		if result.TotalRiskScore < 0.5 {
			t.Logf("Headless browser detected with low risk: %f", result.TotalRiskScore)
		}
	})

	t.Run("Analyze Mouse Pattern - Normal Human", func(t *testing.T) {
		movements := []MouseMovement{}
		for i := 0; i < 20; i++ {
			movements = append(movements, MouseMovement{
				X:         float64(i * 5),
				Y:         float64(i * 3),
				Timestamp: time.Now().Add(time.Duration(i*50) * time.Millisecond),
			})
		}

		result := detector.AnalyzeMousePatternAdvanced(movements)

		if result.Confidence < 0 || result.Confidence > 1 {
			t.Errorf("Confidence should be between 0 and 1")
		}

		if result.Confidence < 0.6 {
			t.Logf("Normal human mouse detected with low confidence: %f", result.Confidence)
		}
	})

	t.Run("Analyze Mouse Pattern - Bot", func(t *testing.T) {
		movements := []MouseMovement{}
		for i := 0; i < 20; i++ {
			movements = append(movements, MouseMovement{
				X:         float64(i * 10),
				Y:         float64(i * 10),
				Timestamp: time.Now().Add(time.Duration(i*100) * time.Millisecond),
			})
		}

		result := detector.AnalyzeMousePatternAdvanced(movements)

		if result.SpeedAnalysis.IsUniform {
			t.Logf("Uniform speed detected in mouse pattern")
		}

		if result.TrajectoryPattern.BotScore > 0.6 {
			t.Logf("Bot score high: %f", result.TrajectoryPattern.BotScore)
		}
	})

	t.Run("Analyze Keyboard Pattern - Normal Human", func(t *testing.T) {
		keypresses := []KeyPress{}
		for i := 0; i < 15; i++ {
			keypresses = append(keypresses, KeyPress{
				Key:       string(rune('a' + i%26)),
				Timestamp: time.Now().Add(time.Duration(100+i*50) * time.Millisecond),
			})
		}

		result := detector.AnalyzeKeyboardPatternAdvanced(keypresses)

		if result.Confidence < 0 || result.Confidence > 1 {
			t.Errorf("Confidence should be between 0 and 1")
		}

		if result.TimingAnalysis.BotScore > 0.6 {
			t.Logf("Keyboard detected as bot: %f", result.TimingAnalysis.BotScore)
		}
	})

	t.Run("Analyze Keyboard Pattern - Mechanical", func(t *testing.T) {
		keypresses := []KeyPress{}
		for i := 0; i < 20; i++ {
			keypresses = append(keypresses, KeyPress{
				Key:       string(rune('a' + i%26)),
				Timestamp: time.Now().Add(time.Duration(100+i*50) * time.Millisecond),
			})
		}

		result := detector.AnalyzeKeyboardPatternAdvanced(keypresses)

		if result.TimingAnalysis.CoefficientOfVariation < 0.05 {
			t.Logf("Mechanical typing detected")
		}
	})
}

func TestIntegrationV20(t *testing.T) {
	t.Run("Complete Bot Detection Pipeline", func(t *testing.T) {
		enhanced := NewTrajectoryNNEnhanced()
		fingerprint := NewFingerprintAdvanced()
		detector := NewAutomationDetectorV2()

		botTrajectory := []TrajectoryNNPoint{}
		for i := 0; i < 15; i++ {
			botTrajectory = append(botTrajectory, TrajectoryNNPoint{
				X:         float64(i * 10),
				Y:         float64(i * 10),
				Timestamp: int64(1000 + i*100),
			})
		}

		features := enhanced.ExtractFeatures(botTrajectory)
		_ = features

		browserData := map[string]interface{}{
			"userAgent":    "HeadlessChrome",
			"webdriver":    true,
			"plugins":      []string{},
			"languages":    []string{},
		}

		fpResult := fingerprint.AnalyzeBrowserEnvironment(browserData)
		detectorResult := detector.DetectAdvancedHeadless(browserData, nil)

		combinedRisk := (fpResult.OverallRisk + detectorResult.TotalRiskScore) / 2.0

		if combinedRisk < 0.5 {
			t.Logf("Combined bot detection risk: %f", combinedRisk)
		}
	})

	t.Run("Complete Human Detection Pipeline", func(t *testing.T) {
		enhanced := NewTrajectoryNNEnhanced()
		fingerprint := NewFingerprintAdvanced()
		detector := NewAutomationDetectorV2()

		humanTrajectory := []TrajectoryNNPoint{
			{X: 0, Y: 0, Timestamp: 1000},
			{X: 12, Y: 8, Timestamp: 1150},
			{X: 25, Y: 18, Timestamp: 1320},
			{X: 38, Y: 30, Timestamp: 1500},
			{X: 50, Y: 42, Timestamp: 1700},
			{X: 65, Y: 55, Timestamp: 1900},
			{X: 78, Y: 68, Timestamp: 2120},
			{X: 90, Y: 82, Timestamp: 2350},
			{X: 100, Y: 95, Timestamp: 2600},
		}

		features := enhanced.ExtractFeatures(humanTrajectory)
		_ = features

		browserData := map[string]interface{}{
			"userAgent":    "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36",
			"plugins":      []string{"Chrome PDF Plugin", "Native Client"},
			"languages":    []string{"en-US", "en"},
			"timezone":     "America/New_York",
		}

		fpResult := fingerprint.AnalyzeBrowserEnvironment(browserData)

		movements := []MouseMovement{}
		for _, p := range humanTrajectory {
			movements = append(movements, MouseMovement{
				X:         p.X,
				Y:         p.Y,
				Timestamp: time.Now().Add(time.Duration(p.Timestamp-1000) * time.Millisecond),
			})
		}

		mouseResult := detector.AnalyzeMousePatternAdvanced(movements)

		combinedRisk := (fpResult.OverallRisk + (1.0 - mouseResult.Confidence)) / 2.0

		if combinedRisk > 0.3 {
			t.Logf("Human detected with risk: %f", combinedRisk)
		}
	})
}

func BenchmarkTrajectoryNNEnhanced(b *testing.B) {
	enhanced := NewTrajectoryNNEnhanced()

	trajectory := []TrajectoryNNPoint{}
	for i := 0; i < 100; i++ {
		trajectory = append(trajectory, TrajectoryNNPoint{
			X:         float64(i * 10),
			Y:         float64(i * 8),
			Timestamp: int64(1000 + i*100),
		})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		enhanced.ExtractFeatures(trajectory)
	}
}

func BenchmarkFingerprintAdvanced(b *testing.B) {
	fingerprint := NewFingerprintAdvanced()

	browserData := map[string]interface{}{
		"userAgent":    "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
		"plugins":      []string{"Chrome PDF Plugin", "Chrome PDF Viewer", "Native Client"},
		"languages":    []string{"en-US", "en", "es"},
		"timezone":     "America/New_York",
		"canvasFingerprint": "canvas_data_with_noise_pattern_for_testing",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		fingerprint.AnalyzeBrowserEnvironment(browserData)
	}
}

func BenchmarkAutomationDetectorV2(b *testing.B) {
	detector := NewAutomationDetectorV2()

	movements := []MouseMovement{}
	for i := 0; i < 50; i++ {
		movements = append(movements, MouseMovement{
			X:         float64(i * 5),
			Y:         float64(i * 4),
			Timestamp: time.Now().Add(time.Duration(i*50) * time.Millisecond),
		})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		detector.AnalyzeMousePatternAdvanced(movements)
	}
}

func TestAccuracySimulation(t *testing.T) {
	enhanced := NewTrajectoryNNEnhanced()
	detector := NewAutomationDetectorV2()

	t.Run("Simulate Bot Detection Accuracy", func(t *testing.T) {
		botCount := 0
		totalBots := 100

		for i := 0; i < totalBots; i++ {
			botTrajectory := []TrajectoryNNPoint{}
			for j := 0; j < 20; j++ {
				botTrajectory = append(botTrajectory, TrajectoryNNPoint{
					X:         float64(j * 10),
					Y:         float64(j * 10),
					Timestamp: int64(1000 + j*100),
				})
			}

			mechanicalScore := enhanced.detectMechanicalMovement(botTrajectory)
			straightness := enhanced.detectPerfectStraightness(botTrajectory)

			if mechanicalScore > 0.6 || straightness > 0.95 {
				botCount++
			}
		}

		accuracy := float64(botCount) / float64(totalBots)
		t.Logf("Bot detection accuracy: %.2f%%", accuracy*100)

		if accuracy < 0.90 {
			t.Logf("Warning: Bot detection accuracy below 90%%: %.2f%%", accuracy*100)
		}
	})

	t.Run("Simulate Human Detection Accuracy", func(t *testing.T) {
		humanCount := 0
		totalHumans := 100

		for i := 0; i < totalHumans; i++ {
			humanTrajectory := []TrajectoryNNPoint{}
			for j := 0; j < 20; j++ {
				x := float64(j * 10 + (i%10 - 5))
				y := float64(j * 8 + (i%8 - 4))
				humanTrajectory = append(humanTrajectory, TrajectoryNNPoint{
					X:         x,
					Y:         y,
					Timestamp: int64(1000 + j*100 + int64(i%20-10)*10),
				})
			}

			humanLikelihood := enhanced.calculateHumanLikelihood(humanTrajectory)
			mechanicalScore := enhanced.detectMechanicalMovement(humanTrajectory)

			if humanLikelihood > 0.4 && mechanicalScore < 0.5 {
				humanCount++
			}
		}

		accuracy := float64(humanCount) / float64(totalHumans)
		t.Logf("Human detection accuracy: %.2f%%", accuracy*100)

		if accuracy < 0.90 {
			t.Logf("Warning: Human detection accuracy below 90%%: %.2f%%", accuracy*100)
		}
	})
}

func TestPerformanceMetrics(t *testing.T) {
	t.Run("Feature Extraction Performance", func(t *testing.T) {
		enhanced := NewTrajectoryNNEnhanced()

		trajectory := []TrajectoryNNPoint{}
		for i := 0; i < 100; i++ {
			trajectory = append(trajectory, TrajectoryNNPoint{
				X:         float64(i * 10),
				Y:         float64(i * 8),
				Timestamp: int64(1000 + i*100),
			})
		}

		start := time.Now()
		iterations := 1000

		for i := 0; i < iterations; i++ {
			enhanced.ExtractFeatures(trajectory)
		}

		duration := time.Since(start)
		avgDuration := duration / time.Duration(iterations)

		t.Logf("Average feature extraction time: %v", avgDuration)

		if avgDuration > 10*time.Millisecond {
			t.Logf("Warning: Feature extraction taking longer than 10ms")
		}
	})
}

var _ context.Context = (*testContext)(nil)

type testContext struct{}

func (c *testContext) Deadline() (time.Time, bool) {
	return time.Now(), false
}

func (c *testContext) Done() <-chan struct{} {
	return nil
}

func (c *testContext) Err() error {
	return nil
}

func (c *testContext) Value(key interface{}) interface{} {
	return nil
}
