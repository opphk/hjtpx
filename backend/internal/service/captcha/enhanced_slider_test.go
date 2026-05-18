package captcha

import (
	"context"
	"image"
	"image/color"
	"testing"
	"time"

	"github.com/hjtpx/hjtpx/pkg/models"
)

func TestEnhancedSliderGenerator(t *testing.T) {
	generator := NewEnhancedSliderGenerator(nil, nil)

	tests := []struct {
		name       string
		difficulty int
		mode       string
	}{
		{"简单难度标准模式", 1, "standard"},
		{"中等难度双轨模式", 2, "dual_track"},
		{"困难难度多障碍模式", 3, "multi_obstacle"},
		{"极难难度混沌模式", 4, "chaos"},
		{"地狱难度标准模式", 5, "standard"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := generator.Create(context.Background(), &EnhancedCreateRequest{
				Difficulty: tt.difficulty,
				Mode:       tt.mode,
			})

			if err != nil {
				t.Fatalf("Failed to create enhanced slider captcha: %v", err)
			}

			if result.SessionID == "" {
				t.Error("Session ID should not be empty")
			}

			if len(result.BackgroundURL) == 0 {
				t.Error("Background URL should not be empty")
			}

			if len(result.SliderURL) == 0 {
				t.Error("Slider URL should not be empty")
			}

			if result.GapX <= 0 || result.GapX > 320 {
				t.Errorf("Invalid GapX: %d", result.GapX)
			}

			if result.GapY <= 0 || result.GapY > 200 {
				t.Errorf("Invalid GapY: %d", result.GapY)
			}

			if result.Difficulty != tt.difficulty {
				t.Errorf("Expected difficulty %d, got %d", tt.difficulty, result.Difficulty)
			}

			if result.ResistanceLevel < 1 || result.ResistanceLevel > 5 {
				t.Errorf("Invalid resistance level: %d", result.ResistanceLevel)
			}

			if result.ExpiresIn <= 0 {
				t.Error("ExpiresIn should be positive")
			}

			if result.ExpiresAt <= time.Now().Unix() {
				t.Error("ExpiresAt should be in the future")
			}

			if result.TrackInfo.UpperTrackY <= 0 || result.TrackInfo.LowerTrackY <= 0 {
				t.Error("Track info should have valid Y positions")
			}
		})
	}
}

func TestEnhancedImageGenerator(t *testing.T) {
	generator := NewEnhancedImageGenerator()

	tests := []struct {
		name       string
		difficulty int
		mode       string
	}{
		{"标准模式", 1, "standard"},
		{"双轨模式", 2, "dual_track"},
		{"多障碍模式", 3, "multi_obstacle"},
		{"混沌模式", 4, "chaos"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := generator.GenerateEnhancedSliderCaptcha(tt.difficulty, tt.mode)

			if err != nil {
				t.Fatalf("Failed to generate enhanced slider captcha: %v", err)
			}

			if len(result.Background) == 0 {
				t.Error("Background should not be empty")
			}

			if len(result.Slider) == 0 {
				t.Error("Slider should not be empty")
			}

			if result.GapX <= 0 || result.GapX > 320 {
				t.Errorf("Invalid GapX: %d", result.GapX)
			}

			if result.GapY <= 0 || result.GapY > 200 {
				t.Errorf("Invalid GapY: %d", result.GapY)
			}

			if result.UpperTrackY <= 0 || result.UpperTrackY > 200 {
				t.Errorf("Invalid UpperTrackY: %d", result.UpperTrackY)
			}

			if result.LowerTrackY <= 0 || result.LowerTrackY > 200 {
				t.Errorf("Invalid LowerTrackY: %d", result.LowerTrackY)
			}

			if tt.mode == "multi_obstacle" || tt.mode == "chaos" {
				if len(result.Obstacles) == 0 {
					t.Error("Obstacles should be present in multi_obstacle or chaos mode")
				}
			}
		})
	}
}

func TestObstacleGenerator(t *testing.T) {
	generator := NewObstacleGenerator()

	tests := []struct {
		name   string
		count  int
		width  int
		height int
	}{
		{"少量障碍物", 2, 320, 200},
		{"中等障碍物", 5, 320, 200},
		{"大量障碍物", 10, 320, 200},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			obstacles := generator.GenerateObstacles(tt.count, tt.width, tt.height)

			if len(obstacles) != tt.count {
				t.Errorf("Expected %d obstacles, got %d", tt.count, len(obstacles))
			}

			for _, obs := range obstacles {
				if obs.Width <= 0 || obs.Height <= 0 {
					t.Error("Obstacle dimensions should be positive")
				}

				if obs.X < 0 || obs.Y < 0 {
					t.Error("Obstacle position should be non-negative")
				}

				if obs.X+obs.Width > tt.width {
					t.Error("Obstacle X + Width exceeds width")
				}

				if obs.Y+obs.Height > tt.height {
					t.Error("Obstacle Y + Height exceeds height")
				}

				if obs.Type == "" {
					t.Error("Obstacle type should not be empty")
				}
			}
		})
	}
}

func TestTrajectoryGenerator(t *testing.T) {
	generator := NewTrajectoryGenerator()

	t.Run("生成轨迹提示", func(t *testing.T) {
		hint := generator.GenerateHint(100, 50, 3)

		if hint.SuggestedSpeed <= 0 {
			t.Error("SuggestedSpeed should be positive")
		}

		if hint.PathComplexity != 3 {
			t.Errorf("Expected PathComplexity 3, got %d", hint.PathComplexity)
		}

		if len(hint.Hints) == 0 {
			t.Error("Hints should not be empty")
		}

		for _, point := range hint.Hints {
			if point.X < 0 || point.Y < 0 {
				t.Error("Hint point coordinates should be non-negative")
			}
		}
	})

	t.Run("生成随机轨迹", func(t *testing.T) {
		trajectory := generator.GenerateRandomTrajectory(0, 25, 200, 25, 3)

		if len(trajectory) == 0 {
			t.Error("Trajectory should not be empty")
		}

		if trajectory[0].X != 0 {
			t.Error("Trajectory should start at X=0")
		}

		if trajectory[len(trajectory)-1].X != 200 {
			t.Errorf("Trajectory should end at X=200, got %f", trajectory[len(trajectory)-1].X)
		}
	})
}

func TestSmartGapDetector(t *testing.T) {
	detector := NewSmartGapDetector()

	t.Run("检测最优缺口", func(t *testing.T) {
		img := image.NewRGBA(image.Rect(0, 0, 320, 200))
		
		for y := 0; y < 200; y++ {
			for x := 0; x < 320; x++ {
				img.Set(x, y, color.RGBA{R: 200, G: 200, B: 200, A: 255})
			}
		}

		gapX, gapY := detector.DetectOptimalGap(img, 3)

		if gapX < 0 || gapX > 320 {
			t.Errorf("Invalid gapX: %d", gapX)
		}

		if gapY < 0 || gapY > 200 {
			t.Errorf("Invalid gapY: %d", gapY)
		}
	})

	t.Run("边缘检测", func(t *testing.T) {
		img := image.NewRGBA(image.Rect(0, 0, 320, 200))
		
		for y := 0; y < 200; y++ {
			for x := 0; x < 320; x++ {
				if x < 160 {
					img.Set(x, y, color.RGBA{R: 100, G: 100, B: 100, A: 255})
				} else {
					img.Set(x, y, color.RGBA{R: 200, G: 200, B: 200, A: 255})
				}
			}
		}

		edges := detector.detectEdges(img)

		if len(edges) != 200 {
			t.Errorf("Expected 200 rows, got %d", len(edges))
		}

		if len(edges[0]) != 320 {
			t.Errorf("Expected 320 columns, got %d", len(edges[0]))
		}
	})

	t.Run("投影平滑", func(t *testing.T) {
		projection := make([]float64, 100)
		for i := range projection {
			projection[i] = float64(i % 20)
		}

		smoothed := detector.smoothProjection(projection, 5)

		if len(smoothed) != len(projection) {
			t.Error("Smoothed projection should have same length")
		}
	})
}

func TestAdaptiveResistanceSystem(t *testing.T) {
	system := NewAdaptiveResistanceSystem()

	t.Run("计算阻力等级", func(t *testing.T) {
		tests := []struct {
			fingerprint string
			difficulty  int
			minLevel   int
			maxLevel   int
		}{
			{"", 1, 1, 3},
			{"abc123", 2, 1, 4},
			{"xyz789", 5, 1, 5},
		}

		for _, tt := range tests {
			level := system.CalculateResistanceLevel(tt.fingerprint, tt.difficulty)
			if level < tt.minLevel || level > tt.maxLevel {
				t.Errorf("Level %d out of range [%d, %d]", level, tt.minLevel, tt.maxLevel)
			}
		}
	})

	t.Run("获取阻力曲线", func(t *testing.T) {
		tests := []int{1, 2, 3, 4, 5}

		for _, level := range tests {
			curve := system.GetResistanceCurve(level)

			if len(curve) != 100 {
				t.Errorf("Expected curve length 100, got %d", len(curve))
			}

			for i, value := range curve {
				if value < 0 || value > 1 {
					t.Errorf("Curve value at %d out of range [0, 1]: %f", i, value)
				}
			}
		}
	})

	t.Run("计算拖拽阻力", func(t *testing.T) {
		tests := []struct {
			level    int
			position float64
		}{
			{1, 0.0},
			{2, 0.5},
			{3, 1.0},
			{5, 0.3},
		}

		for _, tt := range tests {
			resistance := system.CalculateDragResistance(tt.position, tt.level)
			if resistance < 0 || resistance > 1 {
				t.Errorf("Resistance %f out of range [0, 1]", resistance)
			}
		}
	})
}

func TestEnhancedSliderVerifier(t *testing.T) {
	verifier := NewEnhancedSliderVerifier(nil, nil)

	t.Run("推荐难度", func(t *testing.T) {
		tests := []struct {
			fingerprint string
			minDiff     int
			maxDiff     int
		}{
			{"", 1, 3},
			{"abc123", 1, 5},
			{"xyz", 1, 5},
		}

		for _, tt := range tests {
			diff := verifier.GetRecommendedDifficulty(tt.fingerprint)
			if diff < tt.minDiff || diff > tt.maxDiff {
				t.Errorf("Difficulty %d out of range [%d, %d]", diff, tt.minDiff, tt.maxDiff)
			}
		}
	})

	t.Run("检查会话有效性", func(t *testing.T) {
		valid, message := verifier.CheckSessionValid(context.Background(), "nonexistent")
		if valid {
			t.Error("Should return false for nonexistent session")
		}
		if message == "" {
			t.Error("Should return error message for nonexistent session")
		}
	})
}

func TestTrajectoryPredictor(t *testing.T) {
	predictor := NewTrajectoryPredictor()

	t.Run("预测风险", func(t *testing.T) {
		tests := []struct {
			name       string
			trajectory []EnhancedTrajectoryPoint
			minRisk    float64
		}{
			{
				"空轨迹",
				[]EnhancedTrajectoryPoint{},
				0.5,
			},
			{
				"人类轨迹",
				generateHumanLikeTrajectory(),
				0.0,
			},
			{
				"机器人轨迹",
				generateBotTrajectory(),
				0.5,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				risk := predictor.PredictRisk(tt.trajectory)
				if risk < tt.minRisk {
					t.Errorf("Risk %f should be at least %f", risk, tt.minRisk)
				}
			})
		}
	})

	t.Run("提取特征", func(t *testing.T) {
		trajectory := generateHumanLikeTrajectory()
		features := predictor.extractFeatures(trajectory)

		if features["avg_speed"] < 0 {
			t.Error("Average speed should be non-negative")
		}

		if features["total_distance"] < 0 {
			t.Error("Total distance should be non-negative")
		}
	})
}

func TestEnhancedTrajectoryAnalyzer(t *testing.T) {
	analyzer := NewEnhancedTrajectoryAnalyzer()

	t.Run("分析轨迹", func(t *testing.T) {
		tests := []struct {
			name       string
			trajectory []EnhancedTrajectoryPoint
		}{
			{"空轨迹", []EnhancedTrajectoryPoint{}},
			{"人类轨迹", generateHumanLikeTrajectory()},
			{"机器人轨迹", generateBotTrajectory()},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result := analyzer.AnalyzeTrajectory(tt.trajectory, 100, 100)

				if len(tt.trajectory) < 3 {
					if result.Confidence != 0 {
						t.Error("Confidence should be 0 for short trajectories")
					}
				} else {
					if result.Confidence < 0 || result.Confidence > 1 {
						t.Error("Confidence should be in [0, 1]")
					}

					if result.AnomalyScore < 0 || result.AnomalyScore > 1 {
						t.Error("AnomalyScore should be in [0, 1]")
					}
				}
			})
		}
	})

	t.Run("提取详细特征", func(t *testing.T) {
		trajectory := generateHumanLikeTrajectory()
		features := analyzer.extractDetailedFeatures(trajectory)

		if features.TotalDistance < 0 {
			t.Error("TotalDistance should be non-negative")
		}

		if features.DirectDistance < 0 {
			t.Error("DirectDistance should be non-negative")
		}

		if features.Efficiency < 0 || features.Efficiency > 1 {
			t.Error("Efficiency should be in [0, 1]")
		}
	})

	t.Run("分析速度配置", func(t *testing.T) {
		trajectory := generateHumanLikeTrajectory()
		profile := analyzer.analyzeSpeedProfile(trajectory)

		if profile.AverageSpeed < 0 {
			t.Error("AverageSpeed should be non-negative")
		}

		if profile.MaxSpeed < profile.MinSpeed {
			t.Error("MaxSpeed should be >= MinSpeed")
		}

		if profile.SpeedVariance < 0 {
			t.Error("SpeedVariance should be non-negative")
		}
	})

	t.Run("识别模式", func(t *testing.T) {
		features := TrajectoryFeatures{
			Efficiency: 0.99,
			Curvature: 0.01,
			Sinuosity: 1.0,
		}

		pattern := analyzer.identifyPattern([]EnhancedTrajectoryPoint{}, features)
		if pattern != "linear" {
			t.Errorf("Expected 'linear' pattern, got '%s'", pattern)
		}
	})
}

func generateHumanLikeTrajectory() []EnhancedTrajectoryPoint {
	var trajectory []EnhancedTrajectoryPoint
	startTime := time.Now().UnixMilli()

	for i := 0; i < 30; i++ {
		trajectory = append(trajectory, EnhancedTrajectoryPoint{
			X:         float64(i * 8),
			Y:         25 + float64(i%5-2),
			Timestamp: startTime + int64(i*30),
			Pressure:  0.5 + float64(i%3)*0.1,
		})
	}

	return trajectory
}

func generateBotTrajectory() []EnhancedTrajectoryPoint {
	var trajectory []EnhancedTrajectoryPoint
	startTime := time.Now().UnixMilli()

	for i := 0; i < 30; i++ {
		trajectory = append(trajectory, EnhancedTrajectoryPoint{
			X:         float64(i * 8),
			Y:         25.0,
			Timestamp: startTime + int64(i*10),
			Pressure:  0.5,
		})
	}

	return trajectory
}

func TestModelConversion(t *testing.T) {
	t.Run("转换为模型会话", func(t *testing.T) {
		session := &models.CaptchaSession{
			SessionID:   "test_123",
			Status:      "pending",
			VerifyCount: 0,
			MaxAttempts: 3,
			RiskScore:   0,
			TraceScore:  0,
			EnvScore:    0,
			CreatedAt:   time.Now(),
			ExpiredAt:   time.Now().Add(5 * time.Minute),
			ClientIP:    "127.0.0.1",
			UserAgent:   "test",
			Fingerprint: "",
			GapX:        100,
			GapY:        50,
		}

		if session.SessionID == "" {
			t.Error("SessionID should not be empty")
		}

		if session.MaxAttempts != 3 {
			t.Errorf("Expected MaxAttempts 3, got %d", session.MaxAttempts)
		}

		if session.ExpiredAt.Before(time.Now()) {
			t.Error("ExpiredAt should be in the future")
		}
	})
}

func TestEdgeCases(t *testing.T) {
	generator := NewEnhancedImageGenerator()

	t.Run("极小难度", func(t *testing.T) {
		result, err := generator.GenerateEnhancedSliderCaptcha(0, "standard")
		if err != nil {
			t.Fatalf("Should not error with difficulty 0: %v", err)
		}
		if result.GapX == 0 {
			t.Error("GapX should not be 0")
		}
	})

	t.Run("极大难度", func(t *testing.T) {
		result, err := generator.GenerateEnhancedSliderCaptcha(100, "chaos")
		if err != nil {
			t.Fatalf("Should not error with high difficulty: %v", err)
		}
		if len(result.Obstacles) == 0 {
			t.Error("Should have obstacles in chaos mode")
		}
	})
}
