package captcha

import (
	"context"
	"testing"
	"time"
)

func TestVideoGeneratorService_Generate(t *testing.T) {
	service := NewVideoGeneratorService(nil, nil)

	req := &VideoCaptchaRequest{
		Width:       640,
		Height:      360,
		Difficulty:  2,
		ClientIP:    "127.0.0.1",
		UserAgent:   "test-agent",
		Fingerprint: "test-fingerprint",
	}

	resp, err := service.Generate(context.Background(), req)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if resp.SessionID == "" {
		t.Error("SessionID should not be empty")
	}

	if resp.Question == "" {
		t.Error("Question should not be empty")
	}

	if len(resp.Options) == 0 {
		t.Error("Options should not be empty")
	}

	if resp.Difficulty != 2 {
		t.Errorf("Expected difficulty 2, got %d", resp.Difficulty)
	}

	if resp.Width != 640 {
		t.Errorf("Expected width 640, got %d", resp.Width)
	}

	if resp.Height != 360 {
		t.Errorf("Expected height 360, got %d", resp.Height)
	}

	if resp.ExpiresIn <= 0 {
		t.Error("ExpiresIn should be positive")
	}

	if resp.ExpiresAt <= time.Now().Unix() {
		t.Error("ExpiresAt should be in the future")
	}
}

func TestVideoGeneratorService_Generate_DefaultValues(t *testing.T) {
	service := NewVideoGeneratorService(nil, nil)

	req := &VideoCaptchaRequest{}

	resp, err := service.Generate(context.Background(), req)
	if err != nil {
		t.Fatalf("Generate with default values failed: %v", err)
	}

	if resp.Width != 640 {
		t.Errorf("Expected default width 640, got %d", resp.Width)
	}

	if resp.Height != 360 {
		t.Errorf("Expected default height 360, got %d", resp.Height)
	}

	if resp.Difficulty != 2 {
		t.Errorf("Expected default difficulty 2, got %d", resp.Difficulty)
	}
}

func TestVideoGeneratorService_Generate_MaxDifficulty(t *testing.T) {
	service := NewVideoGeneratorService(nil, nil)

	req := &VideoCaptchaRequest{
		Difficulty: 10,
	}

	resp, err := service.Generate(context.Background(), req)
	if err != nil {
		t.Fatalf("Generate with max difficulty failed: %v", err)
	}

	if resp.Difficulty != 5 {
		t.Errorf("Expected difficulty capped at 5, got %d", resp.Difficulty)
	}
}

func TestVideoGeneratorService_Verify_Success(t *testing.T) {
	service := NewVideoGeneratorService(nil, nil)

	req := &VideoCaptchaRequest{
		Difficulty: 2,
	}

	resp, err := service.Generate(context.Background(), req)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	verifyReq := &VideoVerifyRequest{
		SessionID:    resp.SessionID,
		Answer:       resp.CorrectAnswer,
		BehaviorData: map[string]interface{}{"move_count": 10.0, "time_spent": 5.0},
	}

	verifyResp, err := service.Verify(context.Background(), verifyReq)
	if err != nil {
		t.Fatalf("Verify failed: %v", err)
	}

	if !verifyResp.Success {
		t.Error("Verification should succeed with correct answer")
	}

	if verifyResp.Score <= 0 {
		t.Error("Score should be positive")
	}
}

func TestVideoGeneratorService_Verify_WrongAnswer(t *testing.T) {
	service := NewVideoGeneratorService(nil, nil)

	req := &VideoCaptchaRequest{
		Difficulty: 2,
	}

	resp, err := service.Generate(context.Background(), req)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	verifyReq := &VideoVerifyRequest{
		SessionID: resp.SessionID,
		Answer:    "wrong_answer",
	}

	verifyResp, err := service.Verify(context.Background(), verifyReq)
	if err != nil {
		t.Fatalf("Verify failed: %v", err)
	}

	if verifyResp.Success {
		t.Error("Verification should fail with wrong answer")
	}

	if verifyResp.Hint == "" {
		t.Error("Hint should be provided for wrong answer")
	}
}

func TestVideoGeneratorService_Verify_SessionNotFound(t *testing.T) {
	service := NewVideoGeneratorService(nil, nil)

	verifyReq := &VideoVerifyRequest{
		SessionID: "nonexistent_session",
		Answer:    "test",
	}

	verifyResp, err := service.Verify(context.Background(), verifyReq)
	if err != nil {
		t.Fatalf("Verify should not return error: %v", err)
	}

	if verifyResp.Success {
		t.Error("Verification should fail for nonexistent session")
	}

	if verifyResp.Message == "" {
		t.Error("Error message should be provided")
	}
}

func TestVideoGeneratorService_SelectSceneType(t *testing.T) {
	service := NewVideoGeneratorService(nil, nil)

	sceneTypes := make(map[string]bool)

	for i := 0; i < 100; i++ {
		sceneType := service.selectSceneType(2)
		sceneTypes[sceneType] = true
	}

	if len(sceneTypes) == 0 {
		t.Error("Should generate at least one scene type")
	}
}

func TestVideoGeneratorService_GenerateSceneConfig(t *testing.T) {
	service := NewVideoGeneratorService(nil, nil)

	sceneTypes := []string{
		"object_count",
		"color_recognition",
		"action_recognition",
		"pattern_matching",
		"sequence_memory",
	}

	for _, sceneType := range sceneTypes {
		config := service.generateSceneConfig(sceneType, 3)

		if config["scene_type"] != sceneType {
			t.Errorf("Scene type mismatch: expected %s, got %v", sceneType, config["scene_type"])
		}

		if config["duration"] == nil {
			t.Error("Duration should be set")
		}

		if config["fps"] == nil {
			t.Error("FPS should be set")
		}
	}
}

func TestVideoGeneratorService_GenerateQuestion_ObjectCount(t *testing.T) {
	service := NewVideoGeneratorService(nil, nil)

	config := map[string]interface{}{
		"scene_type":   "object_count",
		"count_target": 5,
	}

	question, options, correct := service.generateQuestion("object_count", config, 2)

	if question == "" {
		t.Error("Question should not be empty")
	}

	if len(options) != 4 {
		t.Errorf("Expected 4 options, got %d", len(options))
	}

	if correct != "5" {
		t.Errorf("Expected correct answer '5', got %s", correct)
	}
}

func TestVideoGeneratorService_GenerateQuestion_ColorRecognition(t *testing.T) {
	service := NewVideoGeneratorService(nil, nil)

	config := map[string]interface{}{
		"scene_type":    "color_recognition",
		"target_color":  "red",
		"flash_count":   3,
	}

	question, options, correct := service.generateQuestion("color_recognition", config, 2)

	if question == "" {
		t.Error("Question should not be empty")
	}

	if len(options) != 4 {
		t.Errorf("Expected 4 options, got %d", len(options))
	}

	if correct != "red" {
		t.Errorf("Expected correct answer 'red', got %s", correct)
	}
}

func TestVideoGeneratorService_GenerateQuestion_ActionRecognition(t *testing.T) {
	service := NewVideoGeneratorService(nil, nil)

	config := map[string]interface{}{
		"scene_type":    "action_recognition",
		"target_action": "wave",
	}

	question, options, correct := service.generateQuestion("action_recognition", config, 2)

	if question == "" {
		t.Error("Question should not be empty")
	}

	if len(options) != 4 {
		t.Errorf("Expected 4 options, got %d", len(options))
	}

	if correct != "挥手(wave)" {
		t.Errorf("Expected correct answer '挥手(wave)', got %s", correct)
	}
}

func TestVideoGeneratorService_CheckAnswer(t *testing.T) {
	service := NewVideoGeneratorService(nil, nil)

	testCases := []struct {
		correct string
		answer  string
		expect  bool
	}{
		{"red", "red", true},
		{"blue", "blue", true},
		{"red", "blue", false},
		{"3", "3", true},
		{"3", "4", false},
		{"[a b c]", "[a b c]", true},
		{"[a b c]", "[a c b]", false},
	}

	for _, tc := range testCases {
		result := service.checkAnswer(tc.correct, tc.answer)
		if result != tc.expect {
			t.Errorf("checkAnswer(%s, %s): expected %v, got %v", tc.correct, tc.answer, tc.expect, result)
		}
	}
}

func TestVideoGeneratorService_CalculateBehaviorScore(t *testing.T) {
	service := NewVideoGeneratorService(nil, nil)

	testCases := []struct {
		name        string
		behaviorData map[string]interface{}
		expectMin   float64
		expectMax   float64
	}{
		{
			name:        "empty behavior data",
			behaviorData: nil,
			expectMin:   0.5,
			expectMax:   0.5,
		},
		{
			name:        "good behavior",
			behaviorData: map[string]interface{}{"move_count": 10.0, "time_spent": 5.0},
			expectMin:   0.7,
			expectMax:   0.9,
		},
		{
			name:        "low move count",
			behaviorData: map[string]interface{}{"move_count": 3.0, "time_spent": 5.0},
			expectMin:   0.6,
			expectMax:   0.8,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			score := service.calculateBehaviorScore(tc.behaviorData, nil)
			if score < tc.expectMin || score > tc.expectMax {
				t.Errorf("Expected score between %f and %f, got %f", tc.expectMin, tc.expectMax, score)
			}
		})
	}
}

func TestVideoGeneratorService_GenerateHint(t *testing.T) {
	service := NewVideoGeneratorService(nil, nil)

	sceneTypes := []string{
		"object_count",
		"color_recognition",
		"action_recognition",
		"pattern_matching",
		"sequence_memory",
	}

	for _, sceneType := range sceneTypes {
		for difficulty := 1; difficulty <= 5; difficulty++ {
			hint := service.generateHint(sceneType, difficulty)
			if hint == "" {
				t.Errorf("Hint should not be empty for scene %s difficulty %d", sceneType, difficulty)
			}
		}
	}
}

func TestRotatePattern(t *testing.T) {
	testCases := []struct {
		input    []string
		expected []string
	}{
		{[]string{"a", "b", "c"}, []string{"c", "a", "b"}},
		{[]string{"red", "blue", "green", "yellow"}, []string{"yellow", "red", "blue", "green"}},
		{[]string{"x"}, []string{"x"}},
		{[]string{}, []string{}},
	}

	for _, tc := range testCases {
		result := rotatePattern(tc.input)
		if len(result) != len(tc.expected) {
			t.Errorf("rotatePattern length mismatch: expected %d, got %d", len(tc.expected), len(result))
			continue
		}
		for i := range result {
			if result[i] != tc.expected[i] {
				t.Errorf("rotatePattern[%d]: expected %s, got %s", i, tc.expected[i], result[i])
			}
		}
	}
}

func TestShufflePattern(t *testing.T) {
	original := []string{"a", "b", "c", "d", "e"}
	shuffled := shufflePattern(original)

	if len(shuffled) != len(original) {
		t.Error("Shuffled length should match original length")
	}

	originalSet := make(map[string]int)
	for _, s := range original {
		originalSet[s]++
	}

	shuffledSet := make(map[string]int)
	for _, s := range shuffled {
		shuffledSet[s]++
	}

	for k, v := range originalSet {
		if shuffledSet[k] != v {
			t.Errorf("Element %s count mismatch: expected %d, got %d", k, v, shuffledSet[k])
		}
	}
}

func TestReverseSequence(t *testing.T) {
	testCases := []struct {
		input    []string
		expected []string
	}{
		{[]string{"a", "b", "c"}, []string{"c", "b", "a"}},
		{[]string{"★", "●", "■", "▲"}, []string{"▲", "■", "●", "★"}},
		{[]string{"x"}, []string{"x"}},
		{[]string{}, []string{}},
	}

	for _, tc := range testCases {
		result := reverseSequence(tc.input)
		if len(result) != len(tc.expected) {
			t.Errorf("reverseSequence length mismatch: expected %d, got %d", len(tc.expected), len(result))
			continue
		}
		for i := range result {
			if result[i] != tc.expected[i] {
				t.Errorf("reverseSequence[%d]: expected %s, got %s", i, tc.expected[i], result[i])
			}
		}
	}
}

func TestVideoMathSin(t *testing.T) {
	testCases := []struct {
		input    float64
		expected float64
		delta    float64
	}{
		{0, 0, 0.0001},
		{3.14159 / 2, 1, 0.0001},
		{3.14159, 0, 0.0001},
		{2 * 3.14159, 0, 0.0001},
		{3.14159 / 4, 0.7071, 0.001},
	}

	for _, tc := range testCases {
		result := videoMathSin(tc.input)
		diff := result - tc.expected
		if diff < 0 {
			diff = -diff
		}
		if diff > tc.delta {
			t.Errorf("videoMathSin(%f): expected %f (±%f), got %f", tc.input, tc.expected, tc.delta, result)
		}
	}
}

func TestGenerateVideoSessionID(t *testing.T) {
	ids := make(map[string]bool)

	for i := 0; i < 100; i++ {
		id := generateVideoSessionID()
		if ids[id] {
			t.Errorf("Duplicate session ID generated: %s", id)
		}
		ids[id] = true

		if len(id) < 10 {
			t.Errorf("Session ID too short: %s", id)
		}
	}
}

func TestVideoVerifyResponse_Structure(t *testing.T) {
	resp := &VideoVerifyResponse{
		Success: true,
		Score:   0.95,
		Message: "验证成功",
		Hint:    "测试提示",
	}

	if !resp.Success {
		t.Error("Success should be true")
	}

	if resp.Score != 0.95 {
		t.Errorf("Expected score 0.95, got %f", resp.Score)
	}

	if resp.Message != "验证成功" {
		t.Errorf("Expected message '验证成功', got '%s'", resp.Message)
	}

	if resp.Hint != "测试提示" {
		t.Errorf("Expected hint '测试提示', got '%s'", resp.Hint)
	}
}

func TestVideoSession_Structure(t *testing.T) {
	session := &VideoSession{
		SessionID:     "test_session_123",
		VideoData:     "base64_video_data",
		Question:      "测试问题",
		CorrectAnswer: "test_answer",
		Options:       []string{"A", "B", "C", "D"},
		SceneType:     "object_count",
		Difficulty:    3,
		Status:        "pending",
		VerifyCount:   0,
		MaxAttempts:   3,
		CreatedAt:     time.Now(),
		ExpiredAt:     time.Now().Add(5 * time.Minute),
	}

	if session.SessionID != "test_session_123" {
		t.Errorf("Expected SessionID 'test_session_123', got '%s'", session.SessionID)
	}

	if session.Status != "pending" {
		t.Errorf("Expected Status 'pending', got '%s'", session.Status)
	}

	if session.MaxAttempts != 3 {
		t.Errorf("Expected MaxAttempts 3, got %d", session.MaxAttempts)
	}

	if session.VerifyCount != 0 {
		t.Errorf("Expected VerifyCount 0, got %d", session.VerifyCount)
	}
}

func BenchmarkVideoGeneratorService_Generate(b *testing.B) {
	service := NewVideoGeneratorService(nil, nil)

	req := &VideoCaptchaRequest{
		Width:      640,
		Height:     360,
		Difficulty: 3,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = service.Generate(context.Background(), req)
	}
}

func BenchmarkVideoGeneratorService_GenerateQuestion(b *testing.B) {
	service := NewVideoGeneratorService(nil, nil)

	config := map[string]interface{}{
		"scene_type":   "object_count",
		"count_target": 5,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = service.generateQuestion("object_count", config, 3)
	}
}

func BenchmarkVideoGeneratorService_CheckAnswer(b *testing.B) {
	service := NewVideoGeneratorService(nil, nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		service.checkAnswer("correct", "correct")
	}
}

func BenchmarkVideoGeneratorService_CalculateBehaviorScore(b *testing.B) {
	service := NewVideoGeneratorService(nil, nil)

	behaviorData := map[string]interface{}{
		"move_count": 10.0,
		"time_spent": 5.0,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		service.calculateBehaviorScore(behaviorData, nil)
	}
}
