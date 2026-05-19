package captcha

import (
	"context"
	"testing"
	"time"
)

func TestARGeneratorService_Generate(t *testing.T) {
	service := NewARGeneratorService(nil, nil)

	req := &ARCaptchaRequest{
		SceneType:   "object_placement",
		Width:       640,
		Height:      480,
		Difficulty:  3,
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

	if resp.SceneType != "object_placement" {
		t.Errorf("Expected scene type 'object_placement', got '%s'", resp.SceneType)
	}

	if resp.SceneConfig == nil {
		t.Error("SceneConfig should not be nil")
	}

	if resp.Instructions == "" {
		t.Error("Instructions should not be empty")
	}

	if resp.WebXRSupport != true {
		t.Error("WebXRSupport should be true")
	}

	if resp.ExpiresIn <= 0 {
		t.Error("ExpiresIn should be positive")
	}

	if resp.ExpiresAt <= time.Now().Unix() {
		t.Error("ExpiresAt should be in the future")
	}
}

func TestARGeneratorService_Generate_DefaultValues(t *testing.T) {
	service := NewARGeneratorService(nil, nil)

	req := &ARCaptchaRequest{}

	resp, err := service.Generate(context.Background(), req)
	if err != nil {
		t.Fatalf("Generate with default values failed: %v", err)
	}

	if resp.Width != 640 {
		t.Errorf("Expected default width 640, got %d", resp.Width)
	}

	if resp.Height != 480 {
		t.Errorf("Expected default height 480, got %d", resp.Height)
	}

	if resp.Difficulty != 2 {
		t.Errorf("Expected default difficulty 2, got %d", resp.Difficulty)
	}

	if resp.SceneType == "" {
		t.Error("SceneType should be set to a default value")
	}
}

func TestARGeneratorService_Generate_MaxDifficulty(t *testing.T) {
	service := NewARGeneratorService(nil, nil)

	req := &ARCaptchaRequest{
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

func TestARGeneratorService_Generate_GestureRecognition(t *testing.T) {
	service := NewARGeneratorService(nil, nil)

	req := &ARCaptchaRequest{
		SceneType:  "gesture_recognition",
		Difficulty: 2,
	}

	resp, err := service.Generate(context.Background(), req)
	if err != nil {
		t.Fatalf("Generate gesture recognition failed: %v", err)
	}

	if resp.SceneConfig == nil {
		t.Fatal("SceneConfig should not be nil")
	}

	if len(resp.SceneConfig.Gestures) == 0 {
		t.Error("Gestures should not be empty for gesture_recognition scene")
	}

	if resp.SceneConfig.TargetGesture == "" {
		t.Error("TargetGesture should be set for gesture_recognition scene")
	}
}

func TestARGeneratorService_Generate_ObjectRotation(t *testing.T) {
	service := NewARGeneratorService(nil, nil)

	req := &ARCaptchaRequest{
		SceneType:  "object_rotation",
		Difficulty: 3,
	}

	resp, err := service.Generate(context.Background(), req)
	if err != nil {
		t.Fatalf("Generate object rotation failed: %v", err)
	}

	if resp.SceneConfig == nil {
		t.Fatal("SceneConfig should not be nil")
	}

	if len(resp.SceneConfig.Objects) == 0 {
		t.Error("Objects should not be empty for object_rotation scene")
	}

	for _, obj := range resp.SceneConfig.Objects {
		if obj.TargetRotation == nil {
			t.Error("Each object should have TargetRotation for object_rotation scene")
		}
	}
}

func TestARGeneratorService_Verify_Success(t *testing.T) {
	service := NewARGeneratorService(nil, nil)

	req := &ARCaptchaRequest{
		SceneType:  "object_placement",
		Difficulty: 2,
	}

	genResp, err := service.Generate(context.Background(), req)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	targetPos := []float64{0, 0, 0}
	for _, obj := range genResp.SceneConfig.Objects {
		if obj.TargetPosition != nil {
			targetPos = obj.TargetPosition
			break
		}
	}

	verifyReq := &ARVerifyRequest{
		SessionID:   genResp.SessionID,
		ObjectPos:   targetPos,
		ObjectRot:   []float64{0, 0, 0},
		BehaviorData: map[string]interface{}{"move_count": 10.0, "time_spent": 5.0},
	}

	verifyResp, err := service.Verify(context.Background(), verifyReq)
	if err != nil {
		t.Fatalf("Verify failed: %v", err)
	}

	if !verifyResp.Success {
		t.Error("Verification should succeed with correct position")
	}

	if verifyResp.Score <= 0 {
		t.Error("Score should be positive")
	}
}

func TestARGeneratorService_Verify_WrongPosition(t *testing.T) {
	service := NewARGeneratorService(nil, nil)

	req := &ARCaptchaRequest{
		SceneType:  "object_placement",
		Difficulty: 2,
	}

	genResp, err := service.Generate(context.Background(), req)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	verifyReq := &ARVerifyRequest{
		SessionID: genResp.SessionID,
		ObjectPos: []float64{10, 10, 10},
	}

	verifyResp, err := service.Verify(context.Background(), verifyReq)
	if err != nil {
		t.Fatalf("Verify should not return error: %v", err)
	}

	if verifyResp.Success {
		t.Error("Verification should fail with wrong position")
	}

	if verifyResp.Feedback == "" {
		t.Error("Feedback should be provided for failed verification")
	}
}

func TestARGeneratorService_Verify_SessionNotFound(t *testing.T) {
	service := NewARGeneratorService(nil, nil)

	verifyReq := &ARVerifyRequest{
		SessionID: "nonexistent_session",
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

func TestARGeneratorService_SelectDefaultSceneType(t *testing.T) {
	service := NewARGeneratorService(nil, nil)

	sceneTypes := make(map[string]bool)
	validTypes := map[string]bool{
		"object_placement":    true,
		"gesture_recognition": true,
		"object_rotation":     true,
		"sequential_action":   true,
	}

	for i := 0; i < 100; i++ {
		sceneType := service.selectDefaultSceneType()
		sceneTypes[sceneType] = true

		if !validTypes[sceneType] {
			t.Errorf("Invalid scene type: %s", sceneType)
		}
	}

	if len(sceneTypes) == 0 {
		t.Error("Should generate at least one scene type")
	}
}

func TestARGeneratorService_GeneratePlacementObjects(t *testing.T) {
	service := NewARGeneratorService(nil, nil)

	for difficulty := 1; difficulty <= 5; difficulty++ {
		objects := service.generatePlacementObjects(difficulty)

		expectedCount := 1 + difficulty
		if len(objects) != expectedCount {
			t.Errorf("Difficulty %d: expected %d objects, got %d", difficulty, expectedCount, len(objects))
		}

		for i, obj := range objects {
			if obj.ID == "" {
				t.Errorf("Object %d should have ID", i)
			}
			if obj.Type == "" {
				t.Errorf("Object %d should have Type", i)
			}
			if len(obj.Position) != 3 {
				t.Errorf("Object %d should have 3D position", i)
			}
			if obj.TargetPosition == nil {
				t.Errorf("Object %d should have TargetPosition", i)
			}
			if !obj.Interactable {
				t.Errorf("Object %d should be interactable", i)
			}
		}
	}
}

func TestARGeneratorService_GenerateRotationObjects(t *testing.T) {
	service := NewARGeneratorService(nil, nil)

	for difficulty := 1; difficulty <= 5; difficulty++ {
		objects := service.generateRotationObjects(difficulty)

		expectedCount := 1 + difficulty/2
		if len(objects) != expectedCount {
			t.Errorf("Difficulty %d: expected %d objects, got %d", difficulty, expectedCount, len(objects))
		}

		for i, obj := range objects {
			if obj.TargetRotation == nil {
				t.Errorf("Rotation object %d should have TargetRotation", i)
			}
		}
	}
}

func TestARGeneratorService_GenerateTargetZone(t *testing.T) {
	service := NewARGeneratorService(nil, nil)

	for difficulty := 1; difficulty <= 5; difficulty++ {
		zone := service.generateTargetZone(difficulty)

		if len(zone.Position) != 3 {
			t.Errorf("Target zone should have 3D position, got %d dimensions", len(zone.Position))
		}

		if len(zone.Size) != 3 {
			t.Errorf("Target zone should have 3D size, got %d dimensions", len(zone.Size))
		}

		if zone.Shape == "" {
			t.Error("Target zone should have Shape")
		}
	}
}

func TestARGeneratorService_GenerateInstructions(t *testing.T) {
	service := NewARGeneratorService(nil, nil)

	sceneTypes := []string{
		"object_placement",
		"gesture_recognition",
		"object_rotation",
		"sequential_action",
	}

	for _, sceneType := range sceneTypes {
		for difficulty := 1; difficulty <= 5; difficulty++ {
			instructions := service.generateInstructions(sceneType, difficulty)
			if instructions == "" {
				t.Errorf("Instructions should not be empty for scene %s difficulty %d", sceneType, difficulty)
			}
		}
	}
}

func TestARGeneratorService_ExtractTargetActions(t *testing.T) {
	service := NewARGeneratorService(nil, nil)

	config := &ARSceneConfig{
		Type: "object_placement",
		Objects: []*ARObject{
			{
				ID:             "obj_0",
				TargetPosition: []float64{1, 0, 0},
			},
			{
				ID:             "obj_1",
				TargetPosition: []float64{2, 0, 0},
			},
		},
		TargetGesture: "wave",
	}

	actions := service.extractTargetActions(config)

	if len(actions) == 0 {
		t.Error("Should extract at least one action")
	}

	hasPlacement := false
	for _, action := range actions {
		if len(action) > 6 && action[:6] == "place:" {
			hasPlacement = true
			break
		}
	}

	if !hasPlacement {
		t.Error("Should have at least one placement action")
	}
}

func TestARGeneratorService_EvaluateInteraction(t *testing.T) {
	service := NewARGeneratorService(nil, nil)

	session := &ARSession{
		SceneConfig: &ARSceneConfig{
			Type: "object_placement",
			Objects: []*ARObject{
				{
					ID:             "obj_0",
					TargetPosition: []float64{1, 0, 0},
				},
			},
			Constraints: []ARConstraint{
				{
					Type:      "distance",
					TargetID:  "obj_0",
					Tolerance: 0.2,
					Weight:    0.5,
				},
			},
		},
	}

	testCases := []struct {
		name     string
		objectPos []float64
		minScore float64
	}{
		{"exact position", []float64{1, 0, 0}, 0.9},
		{"close position", []float64{1.1, 0, 0}, 0.5},
		{"far position", []float64{2, 0, 0}, 0.0},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := &ARVerifyRequest{
				ObjectPos: tc.objectPos,
			}
			score := service.evaluateInteraction(req, session)
			if score < tc.minScore {
				t.Errorf("Expected score >= %f, got %f", tc.minScore, score)
			}
		})
	}
}

func TestARGeneratorService_CalculateAccuracy_Placement(t *testing.T) {
	service := NewARGeneratorService(nil, nil)

	session := &ARSession{
		SceneConfig: &ARSceneConfig{
			Type: "object_placement",
			Objects: []*ARObject{
				{ID: "obj_0", TargetPosition: []float64{1, 0, 0}},
				{ID: "obj_1", TargetPosition: []float64{2, 0, 0}},
			},
		},
	}

	testCases := []struct {
		name       string
		objectPos  []float64
		minAccuracy float64
	}{
		{"all correct", []float64{1, 0, 0}, 0.5},
		{"one correct", []float64{2, 0, 0}, 0.5},
		{"all wrong", []float64{10, 10, 10}, 0.0},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := &ARVerifyRequest{
				ObjectPos: tc.objectPos,
			}
			accuracy := service.calculateAccuracy(req, session)
			if accuracy < tc.minAccuracy {
				t.Errorf("Expected accuracy >= %f, got %f", tc.minAccuracy, accuracy)
			}
		})
	}
}

func TestARGeneratorService_CalculateAccuracy_Gesture(t *testing.T) {
	service := NewARGeneratorService(nil, nil)

	session := &ARSession{
		SceneConfig: &ARSceneConfig{
			Type:          "gesture_recognition",
			TargetGesture: "wave",
		},
	}

	testCases := []struct {
		name        string
		gestureData *ARGestureData
		minAccuracy float64
	}{
		{
			"recognized with high confidence",
			&ARGestureData{Recognized: true, Confidence: 0.9},
			0.9,
		},
		{
			"recognized with low confidence",
			&ARGestureData{Recognized: true, Confidence: 0.5},
			0.5,
		},
		{
			"not recognized",
			&ARGestureData{Recognized: false, Confidence: 0.0},
			0.0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := &ARVerifyRequest{
				GestureData: tc.gestureData,
			}
			accuracy := service.calculateAccuracy(req, session)
			if accuracy < tc.minAccuracy {
				t.Errorf("Expected accuracy >= %f, got %f", tc.minAccuracy, accuracy)
			}
		})
	}
}

func TestARGeneratorService_CalculateBehaviorScore(t *testing.T) {
	service := NewARGeneratorService(nil, nil)

	testCases := []struct {
		name        string
		behaviorData map[string]interface{}
		expectMin   float64
		expectMax   float64
	}{
		{
			"empty behavior data",
			nil,
			0.5,
			0.5,
		},
		{
			"good behavior",
			map[string]interface{}{"move_count": 10.0, "time_spent": 5.0, "accuracy": 0.9},
			0.8,
			1.0,
		},
		{
			"low move count",
			map[string]interface{}{"move_count": 3.0, "time_spent": 5.0},
			0.6,
			0.8,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			score := service.calculateBehaviorScore(tc.behaviorData)
			if score < tc.expectMin || score > tc.expectMax {
				t.Errorf("Expected score between %f and %f, got %f", tc.expectMin, tc.expectMax, score)
			}
		})
	}
}

func TestARGeneratorService_GenerateFeedback(t *testing.T) {
	service := NewARGeneratorService(nil, nil)

	sceneTypes := []string{
		"object_placement",
		"gesture_recognition",
		"object_rotation",
		"sequential_action",
	}

	for _, sceneType := range sceneTypes {
		session := &ARSession{
			SceneConfig: &ARSceneConfig{Type: sceneType},
		}
		req := &ARVerifyRequest{}

		feedback := service.generateFeedback(req, session)
		if feedback == "" {
			t.Errorf("Feedback should not be empty for scene type %s", sceneType)
		}
	}
}

func TestCalculateDistance3D(t *testing.T) {
	testCases := []struct {
		name     string
		pos1     []float64
		pos2     []float64
		expected float64
		delta    float64
	}{
		{"same point", []float64{0, 0, 0}, []float64{0, 0, 0}, 0, 0.001},
		{"unit distance", []float64{0, 0, 0}, []float64{1, 0, 0}, 1, 0.001},
		{"diagonal", []float64{0, 0, 0}, []float64{1, 1, 1}, 1.732, 0.001},
		{"2D only", []float64{0, 0}, []float64{3, 4}, 999.0, 0.001},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := calculateDistance3D(tc.pos1, tc.pos2)
			diff := result - tc.expected
			if diff < 0 {
				diff = -diff
			}
			if diff > tc.delta {
				t.Errorf("Expected %f (±%f), got %f", tc.expected, tc.delta, result)
			}
		})
	}
}

func TestCalculateAngleDifference(t *testing.T) {
	testCases := []struct {
		name     string
		rot1     []float64
		rot2     []float64
		expected float64
	}{
		{"same angle", []float64{0, 45, 0}, []float64{0, 45, 0}, 0},
		{"90 degree diff", []float64{0, 0, 0}, []float64{0, 90, 0}, 90},
		{"wrap around 360", []float64{0, 350, 0}, []float64{0, 10, 0}, 20},
		{"half circle", []float64{0, 0, 0}, []float64{0, 180, 0}, 180},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := calculateAngleDifference(tc.rot1, tc.rot2)
			if result != tc.expected {
				t.Errorf("Expected %f, got %f", tc.expected, result)
			}
		})
	}
}

func TestGenerateARSessionID(t *testing.T) {
	ids := make(map[string]bool)

	for i := 0; i < 100; i++ {
		id := generateARSessionID()
		if ids[id] {
			t.Errorf("Duplicate session ID generated: %s", id)
		}
		ids[id] = true

		if len(id) < 10 {
			t.Errorf("Session ID too short: %s", id)
		}
	}
}

func TestARSceneConfig_Structure(t *testing.T) {
	config := &ARSceneConfig{
		Type:        "object_placement",
		Objects:     []*ARObject{},
		Environment: "room",
		Lighting:   "natural",
		Constraints: []ARConstraint{},
	}

	if config.Type != "object_placement" {
		t.Errorf("Expected Type 'object_placement', got '%s'", config.Type)
	}

	if config.Environment != "room" {
		t.Errorf("Expected Environment 'room', got '%s'", config.Environment)
	}
}

func TestARObject_Structure(t *testing.T) {
	obj := &ARObject{
		ID:             "test_obj",
		Type:           "cube",
		Position:       []float64{1, 2, 3},
		Rotation:       []float64{0, 90, 0},
		Scale:          []float64{0.5, 0.5, 0.5},
		Color:          "red",
		TargetPosition: []float64{2, 2, 3},
		TargetRotation: []float64{0, 180, 0},
		Interactable:   true,
		Physics:        true,
	}

	if obj.ID != "test_obj" {
		t.Errorf("Expected ID 'test_obj', got '%s'", obj.ID)
	}

	if obj.Type != "cube" {
		t.Errorf("Expected Type 'cube', got '%s'", obj.Type)
	}

	if len(obj.Position) != 3 {
		t.Errorf("Position should have 3 dimensions, got %d", len(obj.Position))
	}

	if !obj.Interactable {
		t.Error("Object should be interactable")
	}
}

func TestARTargetZone_Structure(t *testing.T) {
	zone := &ARTargetZone{
		Position: []float64{1, 0, 0},
		Size:     []float64{0.5, 0.5, 0.5},
		Shape:    "cube",
		Visible:  true,
	}

	if zone.Shape != "cube" {
		t.Errorf("Expected Shape 'cube', got '%s'", zone.Shape)
	}

	if !zone.Visible {
		t.Error("Zone should be visible")
	}
}

func TestARGestureData_Structure(t *testing.T) {
	data := &ARGestureData{
		Type:       "wave",
		Points:     [][]float64{{0, 0}, {1, 1}, {2, 2}},
		Duration:   1000,
		Recognized: true,
		Confidence: 0.95,
	}

	if data.Type != "wave" {
		t.Errorf("Expected Type 'wave', got '%s'", data.Type)
	}

	if len(data.Points) != 3 {
		t.Errorf("Expected 3 points, got %d", len(data.Points))
	}

	if data.Recognized != true {
		t.Error("Gesture should be recognized")
	}
}

func BenchmarkARGeneratorService_Generate(b *testing.B) {
	service := NewARGeneratorService(nil, nil)

	req := &ARCaptchaRequest{
		SceneType:  "object_placement",
		Difficulty: 3,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = service.Generate(context.Background(), req)
	}
}

func BenchmarkARGeneratorService_GenerateSceneConfig(b *testing.B) {
	service := NewARGeneratorService(nil, nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		service.generateSceneConfig("object_placement", 3)
	}
}

func BenchmarkARGeneratorService_EvaluateInteraction(b *testing.B) {
	service := NewARGeneratorService(nil, nil)

	session := &ARSession{
		SceneConfig: &ARSceneConfig{
			Type: "object_placement",
			Objects: []*ARObject{
				{
					ID:             "obj_0",
					TargetPosition: []float64{1, 0, 0},
				},
			},
			Constraints: []ARConstraint{
				{
					Type:      "distance",
					TargetID:  "obj_0",
					Tolerance: 0.2,
					Weight:    0.5,
				},
			},
		},
	}

	req := &ARVerifyRequest{
		ObjectPos: []float64{1, 0, 0},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		service.evaluateInteraction(req, session)
	}
}

func BenchmarkCalculateDistance3D(b *testing.B) {
	pos1 := []float64{1, 2, 3}
	pos2 := []float64{4, 5, 6}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		calculateDistance3D(pos1, pos2)
	}
}
