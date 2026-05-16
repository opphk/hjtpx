package handler

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestGetDifficultyConfig(t *testing.T) {
	tests := []struct {
		difficulty    ShuffleDifficultyLevel
		wantTargets  int
		wantTolerance int
		wantMinInt   int
		wantMaxInt   int
	}{
		{ShuffleEasy, 3, 35, 200, 3000},
		{ShuffleMedium, 4, 30, 150, 2500},
		{ShuffleHard, 5, 25, 100, 2000},
		{ShuffleExpert, 6, 20, 80, 1500},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("Difficulty_%v", tt.difficulty), func(t *testing.T) {
			targets, tolerance, minInt, maxInt := getShuffleDifficultyConfig(tt.difficulty)
			assert.Equal(t, tt.wantTargets, targets)
			assert.Equal(t, tt.wantTolerance, tolerance)
			assert.Equal(t, tt.wantMinInt, minInt)
			assert.Equal(t, tt.wantMaxInt, maxInt)
		})
	}
}

func TestGetRandomChars(t *testing.T) {
	modes := []TargetType{
		TargetTypeChinese,
		TargetTypeLetter,
		TargetTypeNumber,
		TargetTypeMixed,
		TargetTypeIcon,
	}

	for _, mode := range modes {
		t.Run(string(mode), func(t *testing.T) {
			count := 5
			chars := getRandomChars(mode, count)
			assert.Len(t, chars, count)

			for _, char := range chars {
				assert.NotEmpty(t, char)
			}

			chars2 := getRandomChars(mode, count)
			assert.Len(t, chars2, count)
			assert.NotEqual(t, chars, chars2, "Random chars should be different between calls")
		})
	}
}

func TestGenerateChineseClickChallenge(t *testing.T) {
	t.Run("Default difficulty ShuffleMedium", func(t *testing.T) {
		challenge := GenerateChineseClickChallenge(ShuffleMedium, 0)
		assert.NotNil(t, challenge)
		assert.NotEmpty(t, challenge.SessionID)
		assert.Contains(t, challenge.SessionID, "shuffle_")
		assert.Len(t, challenge.Targets, 4)
		assert.Equal(t, TargetTypeChinese, challenge.Mode)
		assert.Equal(t, ShuffleMedium, challenge.Difficulty)
		assert.NotEmpty(t, challenge.ImageURL)
		assert.Contains(t, challenge.ImageURL, "data:image/png;base64,")
	})

	t.Run("ShuffleEasy difficulty with 3 targets", func(t *testing.T) {
		challenge := GenerateChineseClickChallenge(ShuffleEasy, 3)
		assert.NotNil(t, challenge)
		assert.Len(t, challenge.Targets, 3)
		assert.Equal(t, ShuffleEasy, challenge.Difficulty)
		assert.Equal(t, 3, challenge.MaxTargets)
	})

	t.Run("ShuffleExpert difficulty with 6 targets", func(t *testing.T) {
		challenge := GenerateChineseClickChallenge(ShuffleExpert, 6)
		assert.NotNil(t, challenge)
		assert.Len(t, challenge.Targets, 6)
		assert.Equal(t, ShuffleExpert, challenge.Difficulty)
		assert.Equal(t, 6, challenge.MaxTargets)
	})

	t.Run("Clamped target count", func(t *testing.T) {
		challenge := GenerateChineseClickChallenge(ShuffleMedium, 10)
		assert.NotNil(t, challenge)
		assert.LessOrEqual(t, len(challenge.Targets), 6)
	})

	t.Run("Correct order and display order differ", func(t *testing.T) {
		challenge := GenerateChineseClickChallenge(ShuffleMedium, 4)
		assert.NotNil(t, challenge)
		assert.Len(t, challenge.CorrectOrder, 4)
		assert.Len(t, challenge.DisplayOrder, 4)
	})

	t.Run("Hint text generation", func(t *testing.T) {
		challenge := GenerateChineseClickChallenge(ShuffleMedium, 3)
		assert.NotNil(t, challenge)
		assert.Contains(t, challenge.HintText, "依次点击:")
		assert.Contains(t, challenge.HintText, "→")
	})

	t.Run("Target positions within bounds", func(t *testing.T) {
		challenge := GenerateChineseClickChallenge(ShuffleMedium, 4)
		assert.NotNil(t, challenge)
		for _, target := range challenge.Targets {
			assert.Greater(t, target.X, 0)
			assert.Greater(t, target.Y, 0)
			assert.Less(t, target.X, challenge.ImageWidth)
			assert.Less(t, target.Y, challenge.ImageHeight)
		}
	})

	t.Run("Distractors generated based on difficulty", func(t *testing.T) {
		challengeEasy := GenerateChineseClickChallenge(ShuffleEasy, 3)
		challengeExpert := GenerateChineseClickChallenge(ShuffleExpert, 3)
		assert.NotNil(t, challengeEasy)
		assert.NotNil(t, challengeExpert)
		assert.GreaterOrEqual(t, len(challengeExpert.Distractors), len(challengeEasy.Distractors))
	})
}

func TestGenerateAlphaNumericClickChallenge(t *testing.T) {
	t.Run("Number mode", func(t *testing.T) {
		challenge := GenerateAlphaNumericClickChallenge(ShuffleMedium, 0, TargetTypeNumber)
		assert.NotNil(t, challenge)
		assert.Equal(t, TargetTypeNumber, challenge.Mode)
		for _, target := range challenge.Targets {
			_, err := fmt.Sscan(target.Char, new(int))
			assert.NoError(t, err, "Number mode should contain numeric characters")
		}
	})

	t.Run("Letter mode", func(t *testing.T) {
		challenge := GenerateAlphaNumericClickChallenge(ShuffleMedium, 0, TargetTypeLetter)
		assert.NotNil(t, challenge)
		assert.Equal(t, TargetTypeLetter, challenge.Mode)
		for _, target := range challenge.Targets {
			assert.Regexp(t, "^[A-Z]$", target.Char, "Letter mode should contain uppercase letters")
		}
	})

	t.Run("Mixed mode", func(t *testing.T) {
		challenge := GenerateAlphaNumericClickChallenge(ShuffleMedium, 4, TargetTypeMixed)
		assert.NotNil(t, challenge)
		assert.Equal(t, TargetTypeMixed, challenge.Mode)
	})

	t.Run("Chinese mode falls back to Mixed", func(t *testing.T) {
		challenge := GenerateAlphaNumericClickChallenge(ShuffleMedium, 0, TargetTypeChinese)
		assert.NotNil(t, challenge)
		assert.Equal(t, TargetTypeMixed, challenge.Mode)
	})

	t.Run("Icon mode falls back to Mixed", func(t *testing.T) {
		challenge := GenerateAlphaNumericClickChallenge(ShuffleMedium, 0, TargetTypeIcon)
		assert.NotNil(t, challenge)
		assert.Equal(t, TargetTypeMixed, challenge.Mode)
	})

	t.Run("Custom target count", func(t *testing.T) {
		challenge := GenerateAlphaNumericClickChallenge(ShuffleMedium, 5, TargetTypeLetter)
		assert.NotNil(t, challenge)
		assert.Len(t, challenge.Targets, 5)
	})
}

func TestShuffleTargets(t *testing.T) {
	t.Run("Shuffle produces different order", func(t *testing.T) {
		targets := []ClickTarget{
			{ID: 0, Char: "A"},
			{ID: 1, Char: "B"},
			{ID: 2, Char: "C"},
			{ID: 3, Char: "D"},
		}

		order := ShuffleTargets(targets)
		assert.Len(t, order, 4)

		isSame := true
		for i := 0; i < len(order); i++ {
			if order[i] != i {
				isSame = false
				break
			}
		}
		assert.False(t, isSame, "Shuffled order should be different from original")
	})

	t.Run("Shuffle contains all indices", func(t *testing.T) {
		targets := []ClickTarget{
			{ID: 0}, {ID: 1}, {ID: 2}, {ID: 3}, {ID: 4},
		}

		order := ShuffleTargets(targets)
		seen := make(map[int]bool)
		for _, idx := range order {
			seen[idx] = true
		}

		for i := 0; i < len(targets); i++ {
			assert.True(t, seen[i], "All indices should be present in shuffle")
		}
	})

	t.Run("Multiple shuffles produce different results", func(t *testing.T) {
		targets := []ClickTarget{
			{ID: 0}, {ID: 1}, {ID: 2}, {ID: 3}, {ID: 4},
		}

		order1 := ShuffleTargets(targets)
		order2 := ShuffleTargets(targets)
		order3 := ShuffleTargets(targets)

		differentOrders := 0
		if !equalIntSlices(order1, order2) {
			differentOrders++
		}
		if !equalIntSlices(order2, order3) {
			differentOrders++
		}
		if !equalIntSlices(order1, order3) {
			differentOrders++
		}

		assert.Greater(t, differentOrders, 0, "Multiple shuffles should produce different orders")
	})

	t.Run("Empty targets", func(t *testing.T) {
		targets := []ClickTarget{}
		order := ShuffleTargets(targets)
		assert.Empty(t, order)
	})

	t.Run("Single target", func(t *testing.T) {
		targets := []ClickTarget{{ID: 0}}
		order := ShuffleTargets(targets)
		assert.Len(t, order, 1)
		assert.Equal(t, 0, order[0])
	})
}

func equalIntSlices(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestVerifyShuffleClickCaptcha(t *testing.T) {
	gin.SetMode(gin.TestMode)
	shuffleClickSessions = make(map[string]*ClickChallenge)

	t.Run("Valid sequential clicks", func(t *testing.T) {
		challenge := GenerateChineseClickChallenge(ShuffleEasy, 3)
		shuffleClickSessions[challenge.SessionID] = challenge

		r := gin.New()
		r.POST("/api/v1/captcha/shuffle/verify", VerifyShuffleClickCaptcha)

		clicks := make([]ClickData, 3)
		baseTime := time.Now().UnixMilli()
		for i, idx := range challenge.CorrectOrder {
			target := challenge.Targets[idx]
			clicks[i] = ClickData{
				X:         target.X,
				Y:         target.Y,
				Timestamp: baseTime + int64(i*500),
				TargetID:  idx,
			}
		}

		payload := ClickVerifyRequest{
			SessionID: challenge.SessionID,
			Clicks:    clicks,
		}

		w := httptest.NewRecorder()
		body, _ := json.Marshal(payload)
		req, _ := http.NewRequest("POST", "/api/v1/captcha/shuffle/verify", strings.NewReader(string(body)))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp ClickVerifyResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		assert.NoError(t, err)
		assert.True(t, resp.Success)
		assert.True(t, resp.CaptchaPass)
	})

	t.Run("Invalid click order", func(t *testing.T) {
		challenge := GenerateChineseClickChallenge(ShuffleEasy, 3)
		shuffleClickSessions[challenge.SessionID] = challenge

		r := gin.New()
		r.POST("/api/v1/captcha/shuffle/verify", VerifyShuffleClickCaptcha)

		clicks := make([]ClickData, 3)
		baseTime := time.Now().UnixMilli()
		for i, idx := range challenge.CorrectOrder {
			target := challenge.Targets[idx]
			clicks[i] = ClickData{
				X:         target.X,
				Y:         target.Y,
				Timestamp: baseTime + int64(i*500),
				TargetID:  idx,
			}
		}
		for i, j := 0, len(clicks)-1; i < j; i, j = i+1, j-1 {
			clicks[i], clicks[j] = clicks[j], clicks[i]
		}

		payload := ClickVerifyRequest{
			SessionID: challenge.SessionID,
			Clicks:    clicks,
		}

		w := httptest.NewRecorder()
		body, _ := json.Marshal(payload)
		req, _ := http.NewRequest("POST", "/api/v1/captcha/shuffle/verify", strings.NewReader(string(body)))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp ClickVerifyResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		assert.NoError(t, err)
		assert.False(t, resp.Success)
		assert.False(t, resp.CaptchaPass)
		assert.Contains(t, resp.FailReason, "点击顺序错误")
	})

	t.Run("Click interval too short", func(t *testing.T) {
		challenge := GenerateChineseClickChallenge(ShuffleEasy, 3)
		shuffleClickSessions[challenge.SessionID] = challenge

		r := gin.New()
		r.POST("/api/v1/captcha/shuffle/verify", VerifyShuffleClickCaptcha)

		clicks := make([]ClickData, 3)
		baseTime := time.Now().UnixMilli()
		for i, idx := range challenge.CorrectOrder {
			target := challenge.Targets[idx]
			clicks[i] = ClickData{
				X:         target.X,
				Y:         target.Y,
				Timestamp: baseTime + int64(i*50),
				TargetID:  idx,
			}
		}

		payload := ClickVerifyRequest{
			SessionID: challenge.SessionID,
			Clicks:    clicks,
		}

		w := httptest.NewRecorder()
		body, _ := json.Marshal(payload)
		req, _ := http.NewRequest("POST", "/api/v1/captcha/shuffle/verify", strings.NewReader(string(body)))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp ClickVerifyResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		assert.NoError(t, err)
		assert.False(t, resp.Success)
		assert.Contains(t, resp.FailReason, "点击间隔过短")
	})

	t.Run("Session not found", func(t *testing.T) {
		r := gin.New()
		r.POST("/api/v1/captcha/shuffle/verify", VerifyShuffleClickCaptcha)

		payload := ClickVerifyRequest{
			SessionID: "nonexistent_session",
			Clicks:    []ClickData{{X: 100, Y: 100, Timestamp: time.Now().UnixMilli()}},
		}

		w := httptest.NewRecorder()
		body, _ := json.Marshal(payload)
		req, _ := http.NewRequest("POST", "/api/v1/captcha/shuffle/verify", strings.NewReader(string(body)))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("Empty clicks", func(t *testing.T) {
		challenge := GenerateChineseClickChallenge(ShuffleEasy, 3)
		shuffleClickSessions[challenge.SessionID] = challenge

		r := gin.New()
		r.POST("/api/v1/captcha/shuffle/verify", VerifyShuffleClickCaptcha)

		payload := ClickVerifyRequest{
			SessionID: challenge.SessionID,
			Clicks:    []ClickData{},
		}

		w := httptest.NewRecorder()
		body, _ := json.Marshal(payload)
		req, _ := http.NewRequest("POST", "/api/v1/captcha/shuffle/verify", strings.NewReader(string(body)))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp ClickVerifyResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		assert.NoError(t, err)
		assert.False(t, resp.Success)
		assert.Contains(t, resp.FailReason, "未提供点击数据")
	})

	t.Run("Click count mismatch", func(t *testing.T) {
		challenge := GenerateChineseClickChallenge(ShuffleEasy, 3)
		shuffleClickSessions[challenge.SessionID] = challenge

		r := gin.New()
		r.POST("/api/v1/captcha/shuffle/verify", VerifyShuffleClickCaptcha)

		clicks := make([]ClickData, 2)
		baseTime := time.Now().UnixMilli()
		for i := 0; i < 2; i++ {
			clicks[i] = ClickData{
				X:         100,
				Y:         100,
				Timestamp: baseTime + int64(i*500),
			}
		}

		payload := ClickVerifyRequest{
			SessionID: challenge.SessionID,
			Clicks:    clicks,
		}

		w := httptest.NewRecorder()
		body, _ := json.Marshal(payload)
		req, _ := http.NewRequest("POST", "/api/v1/captcha/shuffle/verify", strings.NewReader(string(body)))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp ClickVerifyResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		assert.NoError(t, err)
		assert.False(t, resp.Success)
		assert.Contains(t, resp.FailReason, "点击数量不匹配")
	})
}

func TestGetShuffleClickCaptcha(t *testing.T) {
	gin.SetMode(gin.TestMode)
	shuffleClickSessions = make(map[string]*ClickChallenge)

	tests := []struct {
		name           string
		query          string
		wantStatus     int
		wantDifficulty ShuffleDifficultyLevel
		wantMode       TargetType
	}{
		{
			name:           "Default Chinese mode",
			query:          "",
			wantStatus:     http.StatusOK,
			wantDifficulty: ShuffleMedium,
			wantMode:       TargetTypeChinese,
		},
		{
			name:           "ShuffleEasy difficulty",
			query:          "?difficulty=easy",
			wantStatus:     http.StatusOK,
			wantDifficulty: ShuffleEasy,
			wantMode:       TargetTypeChinese,
		},
		{
			name:           "ShuffleHard difficulty",
			query:          "?difficulty=hard",
			wantStatus:     http.StatusOK,
			wantDifficulty: ShuffleHard,
			wantMode:       TargetTypeChinese,
		},
		{
			name:           "ShuffleExpert difficulty",
			query:          "?difficulty=expert",
			wantStatus:     http.StatusOK,
			wantDifficulty: ShuffleExpert,
			wantMode:       TargetTypeChinese,
		},
		{
			name:           "Letter mode",
			query:          "?mode=letter",
			wantStatus:     http.StatusOK,
			wantDifficulty: ShuffleMedium,
			wantMode:       TargetTypeLetter,
		},
		{
			name:           "Number mode",
			query:          "?mode=number",
			wantStatus:     http.StatusOK,
			wantDifficulty: ShuffleMedium,
			wantMode:       TargetTypeNumber,
		},
		{
			name:           "Mixed mode",
			query:          "?mode=mixed",
			wantStatus:     http.StatusOK,
			wantDifficulty: ShuffleMedium,
			wantMode:       TargetTypeMixed,
		},
		{
			name:           "Custom points",
			query:          "?points=5",
			wantStatus:     http.StatusOK,
			wantDifficulty: ShuffleMedium,
			wantMode:       TargetTypeChinese,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := gin.New()
			r.GET("/api/v1/captcha/shuffle/click", GetShuffleClickCaptcha)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/api/v1/captcha/shuffle/click"+tt.query, nil)
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.wantStatus, w.Code)

			var resp map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &resp)
			assert.NoError(t, err)

			assert.NotEmpty(t, resp["session_id"])
			assert.NotEmpty(t, resp["image_url"])
			assert.Contains(t, resp["image_url"], "data:image/png;base64,")
			assert.NotNil(t, resp["targets"])
			assert.NotNil(t, resp["correct_order"])
			assert.NotNil(t, resp["display_order"])
			assert.NotEmpty(t, resp["hint_text"])
			assert.Equal(t, float64(tt.wantDifficulty), resp["difficulty"])
			assert.Equal(t, string(tt.wantMode), resp["mode"])
		})
	}
}

func TestAnalyzeClickBehavior(t *testing.T) {
	t.Run("Normal human-like clicks", func(t *testing.T) {
		behavior := []ClickBehavior{
			{X: 100, Y: 100, Timestamp: 1000, Event: "click"},
			{X: 150, Y: 150, Timestamp: 1600, Event: "click"},
			{X: 200, Y: 200, Timestamp: 2300, Event: "click"},
		}
		score := analyzeClickBehavior(behavior)
		assert.Less(t, score, 20.0, "Normal clicks should have low risk score")
	})

	t.Run("Very fast clicks (bot-like)", func(t *testing.T) {
		behavior := []ClickBehavior{
			{X: 100, Y: 100, Timestamp: 1000, Event: "click"},
			{X: 150, Y: 150, Timestamp: 1050, Event: "click"},
			{X: 200, Y: 200, Timestamp: 1100, Event: "click"},
		}
		score := analyzeClickBehavior(behavior)
		assert.Greater(t, score, 0.0, "Fast clicks should have elevated risk score")
	})

	t.Run("Empty behavior data", func(t *testing.T) {
		behavior := []ClickBehavior{}
		score := analyzeClickBehavior(behavior)
		assert.Equal(t, 0.0, score)
	})

	t.Run("Single click", func(t *testing.T) {
		behavior := []ClickBehavior{
			{X: 100, Y: 100, Timestamp: 1000, Event: "click"},
		}
		score := analyzeClickBehavior(behavior)
		assert.Equal(t, 0.0, score)
	})

	t.Run("Mixed events", func(t *testing.T) {
		behavior := []ClickBehavior{
			{X: 100, Y: 100, Timestamp: 1000, Event: "move"},
			{X: 110, Y: 110, Timestamp: 1200, Event: "move"},
			{X: 150, Y: 150, Timestamp: 1500, Event: "click"},
		}
		score := analyzeClickBehavior(behavior)
		assert.GreaterOrEqual(t, score, 0.0)
		assert.LessOrEqual(t, score, 100.0)
	})
}

func TestClickChallenge_GeneratePositions(t *testing.T) {
	t.Run("Positions are non-overlapping", func(t *testing.T) {
		challenge := GenerateChineseClickChallenge(ShuffleMedium, 4)
		assert.NotNil(t, challenge)

		for i, target1 := range challenge.Targets {
			for j, target2 := range challenge.Targets {
				if i == j {
					continue
				}
				dx := target1.X - target2.X
				dy := target1.Y - target2.Y
				distance := math.Sqrt(float64(dx*dx + dy*dy))
				assert.Greater(t, distance, float64(target1.Size/2+target2.Size/2),
					"Targets should not overlap")
			}
		}
	})

	t.Run("All targets have valid positions", func(t *testing.T) {
		challenge := GenerateChineseClickChallenge(ShuffleMedium, 4)
		assert.NotNil(t, challenge)

		for _, target := range challenge.Targets {
			assert.Greater(t, target.X, 0)
			assert.Greater(t, target.Y, 0)
			assert.Less(t, target.X, challenge.ImageWidth)
			assert.Less(t, target.Y, challenge.ImageHeight)
		}
	})
}

func TestClickChallenge_GenerateDistractors(t *testing.T) {
	t.Run("Distractors count varies by difficulty", func(t *testing.T) {
		challengeEasy := GenerateChineseClickChallenge(ShuffleEasy, 3)
		challengeHard := GenerateChineseClickChallenge(ShuffleHard, 3)

		assert.NotNil(t, challengeEasy)
		assert.NotNil(t, challengeHard)
		assert.Greater(t, len(challengeHard.Distractors), len(challengeEasy.Distractors))
	})

	t.Run("Distractors have different characters", func(t *testing.T) {
		challenge := GenerateChineseClickChallenge(ShuffleMedium, 3)
		assert.NotNil(t, challenge)

		targetChars := make(map[string]bool)
		for _, target := range challenge.Targets {
			targetChars[target.Char] = true
		}

		for _, distractor := range challenge.Distractors {
			assert.False(t, targetChars[distractor.Char],
				"Distractor should not have same character as target")
		}
	})
}

func TestClampInt(t *testing.T) {
	tests := []struct {
		val      int
		min      int
		max      int
		expected int
	}{
		{5, 1, 10, 5},
		{0, 1, 10, 1},
		{15, 1, 10, 10},
		{-5, 1, 10, 1},
		{100, 1, 10, 10},
	}

	for _, tt := range tests {
		result := clampInt(tt.val, tt.min, tt.max)
		assert.Equal(t, tt.expected, result)
	}
}

func TestGetRandomDistractorChar(t *testing.T) {
	existingChars := make(map[string]bool)

	t.Run("Chinese mode", func(t *testing.T) {
		char := getRandomDistractorChar(TargetTypeChinese, existingChars)
		assert.NotEmpty(t, char)
	})

	t.Run("Letter mode", func(t *testing.T) {
		char := getRandomDistractorChar(TargetTypeLetter, existingChars)
		assert.NotEmpty(t, char)
		assert.Regexp(t, "^[A-Z]$", char)
	})

	t.Run("Number mode", func(t *testing.T) {
		char := getRandomDistractorChar(TargetTypeNumber, existingChars)
		assert.NotEmpty(t, char)
	})

	t.Run("Icon mode", func(t *testing.T) {
		char := getRandomDistractorChar(TargetTypeIcon, existingChars)
		assert.NotEmpty(t, char)
	})

	t.Run("Mixed mode", func(t *testing.T) {
		char := getRandomDistractorChar(TargetTypeMixed, existingChars)
		assert.NotEmpty(t, char)
	})
}

func TestClickChallenge_ImageGeneration(t *testing.T) {
	t.Run("Image URL is valid base64 PNG", func(t *testing.T) {
		challenge := GenerateChineseClickChallenge(ShuffleMedium, 4)
		assert.NotNil(t, challenge)
		assert.Contains(t, challenge.ImageURL, "data:image/png;base64,")

		prefix := "data:image/png;base64,"
		base64Data := strings.TrimPrefix(challenge.ImageURL, prefix)
		assert.NotEmpty(t, base64Data)
	})

	t.Run("Image dimensions match configuration", func(t *testing.T) {
		challenge := GenerateChineseClickChallenge(ShuffleMedium, 4)
		assert.NotNil(t, challenge)
		assert.Equal(t, 400, challenge.ImageWidth)
		assert.Equal(t, 300, challenge.ImageHeight)
	})
}

func TestDifficultyLevels_Values(t *testing.T) {
	assert.Equal(t, 1, int(ShuffleEasy))
	assert.Equal(t, 2, int(ShuffleMedium))
	assert.Equal(t, 3, int(ShuffleHard))
	assert.Equal(t, 4, int(ShuffleExpert))
}

func TestClickVerifyResponse_JSON(t *testing.T) {
	t.Run("Success response", func(t *testing.T) {
		resp := ClickVerifyResponse{
			Success:       true,
			Message:       "验证成功",
			RiskScore:     10.5,
			CaptchaPass:   true,
		}
		data, err := json.Marshal(resp)
		assert.NoError(t, err)
		assert.Contains(t, string(data), `"success":true`)
		assert.Contains(t, string(data), `"captcha_pass":true`)
	})

	t.Run("Failure response with reason", func(t *testing.T) {
		resp := ClickVerifyResponse{
			Success:     false,
			Message:     "验证失败",
			RiskScore:   75.0,
			CaptchaPass: false,
			FailReason:  "点击顺序错误",
		}
		data, err := json.Marshal(resp)
		assert.NoError(t, err)
		assert.Contains(t, string(data), `"success":false`)
		assert.Contains(t, string(data), `"fail_reason":"点击顺序错误"`)
	})
}
