package service

import (
	"encoding/json"
	"testing"

	github.com/hjtpx/hjtpx/internal/testing/fuzz"
	"github.com/stretchr/testify/assert"
)

func TestCaptchaFuzzing(t *testing.T) {
	fuzzer := fuzz.NewFuzzer(&fuzz.FuzzingConfig{
		Iterations: 1000,
		Verbose:    false,
	})

	t.Run("FuzzVerifyCaptcha", func(t *testing.T) {
		result := fuzzer.Run(t, "VerifyCaptcha", func(input []byte) error {
			var req VerifyCaptchaRequest
			if err := json.Unmarshal(input, &req); err != nil {
				return nil
			}
			_, _ = VerifyCaptcha(&req)
			return nil
		})
		assert.Zero(t, result.Panics, "Captcha verification should not panic")
		assert.Greater(t, result.TotalIterations, 0)
	})

	t.Run("FuzzGenerateSlider", func(t *testing.T) {
		result := fuzzer.Run(t, "GenerateSlider", func(input []byte) error {
			var req GenerateSliderRequest
			if err := json.Unmarshal(input, &req); err != nil {
				return nil
			}
			_, _ = GenerateSliderCaptcha(&req)
			return nil
		})
		assert.Zero(t, result.Panics, "Slider generation should not panic")
	})

	t.Run("EdgeCases", func(t *testing.T) {
		for _, input := range fuzz.EdgeCases() {
			var req VerifyCaptchaRequest
			_ = json.Unmarshal(input, &req)
			_, _ = VerifyCaptcha(&req)
		}
	})
}

type VerifyCaptchaRequest struct {
	CaptchaID string      `json:"captcha_id"`
	Answer    interface{} `json:"answer"`
	Type      string      `json:"type"`
	SessionID string      `json:"session_id"`
}

type GenerateSliderRequest struct {
	Width  int `json:"width"`
	Height int `json:"height"`
	Difficulty string `json:"difficulty"`
}

func VerifyCaptcha(req *VerifyCaptchaRequest) (bool, error) {
	if req == nil {
		return false, nil
	}
	return len(req.CaptchaID) > 0, nil
}

func GenerateSliderCaptcha(req *GenerateSliderRequest) (interface{}, error) {
	if req == nil {
		return nil, nil
	}
	return map[string]interface{}{"id": "test"}, nil
}
