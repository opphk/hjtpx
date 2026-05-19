package captcha

import (
	"image"
	"testing"
)

func BenchmarkSliderCaptchaGeneration(b *testing.B) {
	generator := NewImageGenerator()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := generator.GenerateSliderCaptcha()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkVideoCaptchaGeneration(b *testing.B) {
	generator := NewVideoGeneratorServiceSimple()

	req := &VideoCaptchaRequest{
		Width:      640,
		Height:     360,
		Difficulty: 2,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := generator.Create(nil, req)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkARCaptchaGeneration(b *testing.B) {
	generator := NewARGeneratorService(nil, nil)

	req := &CreateARRequest{
		SceneType:  "gesture_recognition",
		Difficulty: "medium",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := generator.Create(nil, req)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkTrajectoryAnalysis(b *testing.B) {
	enhancer := NewSliderSecurityEnhancer()

	points := make([]SliderPoint, 50)
	for i := 0; i < 50; i++ {
		points[i] = SliderPoint{
			X:         i * 10,
			Y:         i * 5,
			Timestamp: int64(i * 100),
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		enhancer.EnhancedTrajectoryAnalysis(points, 500)
	}
}

func BenchmarkClickAnalysis(b *testing.B) {
	enhancer := NewClickCaptchaSecurityEnhancer()

	clicks := make([]ClickPoint, 10)
	for i := 0; i < 10; i++ {
		clicks[i] = ClickPoint{
			X:         50 + i*10,
			Y:         50 + i*5,
			Timestamp: int64(i * 200),
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		enhancer.AnalyzeClickPattern(clicks)
	}
}

func BenchmarkSessionIDGeneration(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = generateSessionID()
	}
}

func BenchmarkVideoSessionIDGeneration(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = generateVideoSessionID()
	}
}

func BenchmarkImageEncoding(b *testing.B) {
	generator := NewImageGenerator()
	img := image.NewRGBA(image.Rect(0, 0, 320, 160))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		generator.EncodeToBase64(img)
	}
}

func BenchmarkVideoAnswerChecking(b *testing.B) {
	verifier := NewVideoVerifierServiceSimple()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		verifier.checkAnswer("举手", "举手过头顶")
	}
}

func BenchmarkVoiceAudioGeneration(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = generateVoiceAudio("1234", "zh-CN")
	}
}
