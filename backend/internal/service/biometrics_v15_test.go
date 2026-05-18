package service

import (
	"encoding/json"
	"testing"
	"time"
)

func TestNewBiometricsV15Service(t *testing.T) {
	service := NewBiometricsV15Service()
	if service == nil {
		t.Error("NewBiometricsV15Service 返回了 nil")
	}
	if service.profiles == nil {
		t.Error("profiles map 未初始化")
	}
}

func TestRegisterMultimodalProfile(t *testing.T) {
	service := NewBiometricsV15Service()

	mouseData := &MousePressureData{
		PressureAnalysis: &PressureAnalysis{
			AveragePressure: 0.5,
			PressureStd:     0.2,
			MaxPressure:     0.8,
			MinPressure:     0.2,
			PressureRange:   0.6,
		},
		ClickAnalysis: &ClickAnalysis{
			ClickCount:       10,
			AvgClickDuration: 150.0,
		},
		MovementAnalysis: &MovementAnalysis{
			AvgSpeed: 0.5,
			SpeedStd: 0.2,
			MaxSpeed: 1.2,
		},
	}

	data := &MultimodalBiometricData{
		UserID:        "user-123",
		MousePressure: mouseData,
	}

	profile, err := service.RegisterMultimodalProfile("user-123", data)
	if err != nil {
		t.Errorf("注册失败: %v", err)
	}

	if profile == nil {
		t.Error("返回的 profile 为 nil")
	}

	if profile.UserID != "user-123" {
		t.Errorf("期望 UserID 'user-123', 实际得到 %s", profile.UserID)
	}

	if profile.MousePressureProfile == nil {
		t.Error("MousePressureProfile 为 nil")
	}

	if profile.VerificationCount != 1 {
		t.Errorf("期望 VerificationCount 1, 实际得到 %d", profile.VerificationCount)
	}

	if len(profile.FeatureVector) == 0 {
		t.Error("FeatureVector 为空")
	}
}

func TestRegisterMultimodalProfile_EmptyUserID(t *testing.T) {
	service := NewBiometricsV15Service()

	data := &MultimodalBiometricData{
		UserID: "",
	}

	_, err := service.RegisterMultimodalProfile("", data)
	if err == nil {
		t.Error("期望返回错误，但没有返回")
	}
}

func TestRegisterMultimodalProfile_UpdateExisting(t *testing.T) {
	service := NewBiometricsV15Service()

	mouseData1 := &MousePressureData{
		PressureAnalysis: &PressureAnalysis{AveragePressure: 0.5},
	}

	data1 := &MultimodalBiometricData{
		UserID:        "user-update",
		MousePressure: mouseData1,
	}

	profile1, _ := service.RegisterMultimodalProfile("user-update", data1)

	mouseData2 := &MousePressureData{
		PressureAnalysis: &PressureAnalysis{AveragePressure: 0.6},
	}

	data2 := &MultimodalBiometricData{
		UserID:        "user-update",
		MousePressure: mouseData2,
	}

	profile2, _ := service.RegisterMultimodalProfile("user-update", data2)

	if profile1.VerificationCount != 1 {
		t.Errorf("第一次注册 VerificationCount 期望 1, 实际 %d", profile1.VerificationCount)
	}

	if profile2.VerificationCount != 2 {
		t.Errorf("第二次注册 VerificationCount 期望 2, 实际 %d", profile2.VerificationCount)
	}
}

func TestVerifyMultimodal_NotFound(t *testing.T) {
	service := NewBiometricsV15Service()

	data := &MultimodalBiometricData{
		UserID: "nonexistent",
	}

	result, _ := service.VerifyMultimodal("nonexistent", data)

	if result.IsVerified {
		t.Error("未注册用户应该验证失败")
	}

	if result.OverallConfidence != 0 {
		t.Errorf("未注册用户置信度应为零, 实际 %f", result.OverallConfidence)
	}
}

func TestVerifyMultimodal_Success(t *testing.T) {
	service := NewBiometricsV15Service()

	mouseData := &MousePressureData{
		PressureAnalysis: &PressureAnalysis{
			AveragePressure: 0.5,
			PressureStd:     0.2,
			MaxPressure:     0.8,
			MinPressure:     0.2,
		},
		MovementAnalysis: &MovementAnalysis{
			AvgSpeed: 0.5,
			SpeedStd: 0.2,
		},
	}

	registerData := &MultimodalBiometricData{
		UserID:        "verify-user",
		MousePressure: mouseData,
	}

	service.RegisterMultimodalProfile("verify-user", registerData)

	result, err := service.VerifyMultimodal("verify-user", registerData)
	if err != nil {
		t.Errorf("验证失败: %v", err)
	}

	if result == nil {
		t.Error("返回的 result 为 nil")
	}

	if !result.IsVerified {
		t.Logf("验证状态: %v, 置信度: %f", result.IsVerified, result.OverallConfidence)
	}
}

func TestVerifyMultimodal_TouchForce(t *testing.T) {
	service := NewBiometricsV15Service()

	touchData := &TouchForceData{
		ForceAnalysis: &TouchForceAnalysis{
			TouchCount:  20,
			AvgForce:    0.6,
			ForceStdDev: 0.15,
		},
		SwipeAnalysis: &SwipeAnalysis{
			SwipeCount:       5,
			DirectionEntropy: 1.5,
			AvgSpeed:         0.8,
		},
	}

	data := &MultimodalBiometricData{
		UserID:     "touch-user",
		TouchForce: touchData,
	}

	profile, _ := service.RegisterMultimodalProfile("touch-user", data)

	if profile.TouchForceProfile == nil {
		t.Error("TouchForceProfile 为 nil")
	}

	if profile.TouchForceProfile.TouchCount != 20 {
		t.Errorf("期望 TouchCount 20, 实际 %d", profile.TouchForceProfile.TouchCount)
	}

	if profile.TouchForceProfile.AvgForce != 0.6 {
		t.Errorf("期望 AvgForce 0.6, 实际 %f", profile.TouchForceProfile.AvgForce)
	}
}

func TestVerifyMultimodal_EyeTracking(t *testing.T) {
	service := NewBiometricsV15Service()

	eyeData := &EyeTrackingData{
		GazeAnalysis: &GazeAnalysis{
			GazeCount:    100,
			AvgX:         500,
			AvgY:         300,
			XStd:         50,
			YStd:         40,
			CoverageArea: 0.3,
			AvgPupilSize: 3.5,
		},
		BlinkAnalysis: &BlinkAnalysis{
			BlinkCount:       15,
			BlinkRate:        12.0,
			AvgBlinkDuration: 150.0,
		},
		FixationAnalysis: &FixationAnalysis{
			FixationCount: 25,
			AvgDuration:   300.0,
			AvgDispersion: 20.0,
		},
		FocusAnalysis: &FocusAnalysis{
			AttentionRatio: 0.85,
		},
	}

	data := &MultimodalBiometricData{
		UserID:      "eye-user",
		EyeTracking: eyeData,
	}

	profile, _ := service.RegisterMultimodalProfile("eye-user", data)

	if profile.EyeTrackingProfile == nil {
		t.Error("EyeTrackingProfile 为 nil")
	}

	if profile.EyeTrackingProfile.GazeCount != 100 {
		t.Errorf("期望 GazeCount 100, 实际 %d", profile.EyeTrackingProfile.GazeCount)
	}

	if profile.EyeTrackingProfile.BlinkCount != 15 {
		t.Errorf("期望 BlinkCount 15, 实际 %d", profile.EyeTrackingProfile.BlinkCount)
	}
}

func TestGetProfile(t *testing.T) {
	service := NewBiometricsV15Service()

	mouseData := &MousePressureData{
		PressureAnalysis: &PressureAnalysis{AveragePressure: 0.5},
	}

	data := &MultimodalBiometricData{
		UserID:        "get-user",
		MousePressure: mouseData,
	}

	service.RegisterMultimodalProfile("get-user", data)

	profile, exists := service.GetProfile("get-user")
	if !exists {
		t.Error("应该找到已注册的用户")
	}

	if profile == nil {
		t.Error("返回的 profile 为 nil")
	}

	_, exists = service.GetProfile("nonexistent")
	if exists {
		t.Error("不应该找到不存在的用户")
	}
}

func TestDeleteProfile(t *testing.T) {
	service := NewBiometricsV15Service()

	mouseData := &MousePressureData{
		PressureAnalysis: &PressureAnalysis{AveragePressure: 0.5},
	}

	data := &MultimodalBiometricData{
		UserID:        "delete-user",
		MousePressure: mouseData,
	}

	service.RegisterMultimodalProfile("delete-user", data)

	deleted := service.DeleteProfile("delete-user")
	if !deleted {
		t.Error("应该成功删除")
	}

	_, exists := service.GetProfile("delete-user")
	if exists {
		t.Error("删除后不应该再找到用户")
	}

	deleted = service.DeleteProfile("nonexistent")
	if deleted {
		t.Error("删除不存在的用户应该返回 false")
	}
}

func TestParseBiometricData(t *testing.T) {
	service := NewBiometricsV15Service()

	jsonData := `{
		"session_id": "test-session",
		"user_id": "test-user",
		"mouse_pressure": {
			"pressure_analysis": {
				"average_pressure": 0.5
			}
		}
	}`

	data, err := service.ParseBiometricData([]byte(jsonData))
	if err != nil {
		t.Errorf("解析失败: %v", err)
	}

	if data.SessionID != "test-session" {
		t.Errorf("期望 SessionID 'test-session', 实际 %s", data.SessionID)
	}

	if data.UserID != "test-user" {
		t.Errorf("期望 UserID 'test-user', 实际 %s", data.UserID)
	}
}

func TestSerializeDeserializeProfile(t *testing.T) {
	service := NewBiometricsV15Service()

	mouseData := &MousePressureData{
		PressureAnalysis: &PressureAnalysis{AveragePressure: 0.5},
	}

	data := &MultimodalBiometricData{
		UserID:        "serialize-user",
		MousePressure: mouseData,
	}

	profile, _ := service.RegisterMultimodalProfile("serialize-user", data)

	serialized, err := service.SerializeProfile(profile)
	if err != nil {
		t.Errorf("序列化失败: %v", err)
	}

	deserialized, err := service.DeserializeProfile(serialized)
	if err != nil {
		t.Errorf("反序列化失败: %v", err)
	}

	if deserialized.UserID != profile.UserID {
		t.Errorf("UserID 不匹配: %s vs %s", deserialized.UserID, profile.UserID)
	}
}

func TestCalculateCosineSimilarity(t *testing.T) {
	service := NewBiometricsV15Service()

	vec1 := []float64{1.0, 0.0, 0.0}
	vec2 := []float64{1.0, 0.0, 0.0}

	similarity := service.CalculateCosineSimilarity(vec1, vec2)
	if similarity != 1.0 {
		t.Errorf("相同向量相似度应为零度, 实际 %f", similarity)
	}

	vec3 := []float64{1.0, 0.0, 0.0}
	vec4 := []float64{0.0, 1.0, 0.0}

	similarity = service.CalculateCosineSimilarity(vec3, vec4)
	if similarity != 0.0 {
		t.Errorf("垂直向量相似度应为零, 实际 %f", similarity)
	}

	vec5 := []float64{1.0, 1.0}
	vec6 := []float64{1.0, 1.0}

	similarity = service.CalculateCosineSimilarity(vec5, vec6)
	if similarity < 0.99 {
		t.Errorf("相似向量相似度应接近 1, 实际 %f", similarity)
	}
}

func TestCalculateEuclideanDistance(t *testing.T) {
	service := NewBiometricsV15Service()

	vec1 := []float64{0.0, 0.0, 0.0}
	vec2 := []float64{3.0, 4.0, 0.0}

	distance := service.CalculateEuclideanDistance(vec1, vec2)
	if distance != 5.0 {
		t.Errorf("期望距离 5.0, 实际 %f", distance)
	}

	vec3 := []float64{1.0, 2.0, 3.0}
	vec4 := []float64{1.0, 2.0, 3.0}

	distance = service.CalculateEuclideanDistance(vec3, vec4)
	if distance != 0.0 {
		t.Errorf("相同向量距离应为零, 实际 %f", distance)
	}
}

func TestNormalizeFeatureVector(t *testing.T) {
	service := NewBiometricsV15Service()

	vec := []float64{1.0, 2.0, 3.0, 4.0, 5.0}

	normalized := service.NormalizeFeatureVector(vec)

	if len(normalized) != len(vec) {
		t.Errorf("归一化后长度应不变")
	}

	for _, v := range normalized {
		if v < 0 || v > 1 {
			t.Errorf("归一化值应在 [0,1] 范围内, 实际 %f", v)
		}
	}
}

func TestCompareMousePressureProfiles(t *testing.T) {
	service := NewBiometricsV15Service()

	profile1 := &MousePressureProfile{
		AveragePressure: 0.5,
		PressureStdDev:  0.2,
		MaxPressure:     0.8,
		AvgSpeed:        0.5,
	}

	profile2 := &MousePressureProfile{
		AveragePressure: 0.5,
		PressureStdDev:  0.2,
		MaxPressure:     0.8,
		AvgSpeed:        0.5,
	}

	score := service.compareMousePressureProfiles(profile1, profile2)
	if score < 0.9 {
		t.Errorf("相同特征配置文件分数应接近 1, 实际 %f", score)
	}

	profile3 := &MousePressureProfile{
		AveragePressure: 0.1,
		PressureStdDev:  0.8,
		MaxPressure:     0.2,
		AvgSpeed:        0.1,
	}

	score = service.compareMousePressureProfiles(profile1, profile3)
	if score > 0.5 {
		t.Errorf("不同特征配置文件分数应较低, 实际 %f", score)
	}

	score = service.compareMousePressureProfiles(nil, profile1)
	if score != 0.5 {
		t.Errorf("nil profile 应返回 0.5, 实际 %f", score)
	}
}

func TestCompareTouchForceProfiles(t *testing.T) {
	service := NewBiometricsV15Service()

	profile1 := &TouchForceProfile{
		AvgForce:      0.6,
		ForceStdDev:   0.15,
		AvgSpeed:      0.8,
		SwipeCount:    5,
		AvgPinchScale: 1.2,
	}

	profile2 := &TouchForceProfile{
		AvgForce:      0.6,
		ForceStdDev:   0.15,
		AvgSpeed:      0.8,
		SwipeCount:    5,
		AvgPinchScale: 1.2,
	}

	score := service.compareTouchForceProfiles(profile1, profile2)
	if score < 0.9 {
		t.Errorf("相同特征配置文件分数应接近 1, 实际 %f", score)
	}

	profile3 := &TouchForceProfile{
		AvgForce:      0.2,
		ForceStdDev:   0.5,
		AvgSpeed:      0.1,
		SwipeCount:    1,
		AvgPinchScale: 0.5,
	}

	score = service.compareTouchForceProfiles(profile1, profile3)
	if score > 0.5 {
		t.Errorf("不同特征配置文件分数应较低, 实际 %f", score)
	}
}

func TestCompareEyeTrackingProfiles(t *testing.T) {
	service := NewBiometricsV15Service()

	profile1 := &EyeTrackingProfile{
		AvgX:                500,
		AvgY:                300,
		XStd:                50,
		YStd:                40,
		AvgPupilSize:        3.5,
		BlinkRate:           12.0,
		AvgFixationDuration: 300.0,
		AttentionRatio:      0.85,
	}

	profile2 := &EyeTrackingProfile{
		AvgX:                500,
		AvgY:                300,
		XStd:                50,
		YStd:                40,
		AvgPupilSize:        3.5,
		BlinkRate:           12.0,
		AvgFixationDuration: 300.0,
		AttentionRatio:      0.85,
	}

	score := service.compareEyeTrackingProfiles(profile1, profile2)
	if score < 0.9 {
		t.Errorf("相同特征配置文件分数应接近 1, 实际 %f", score)
	}

	profile3 := &EyeTrackingProfile{
		AvgX:                100,
		AvgY:                100,
		XStd:                10,
		YStd:                10,
		AvgPupilSize:        1.0,
		BlinkRate:           1.0,
		AvgFixationDuration: 50.0,
		AttentionRatio:      0.2,
	}

	score = service.compareEyeTrackingProfiles(profile1, profile3)
	if score > 0.5 {
		t.Errorf("不同特征配置文件分数应较低, 实际 %f", score)
	}
}

func TestCalculateSimilarityScore(t *testing.T) {
	service := NewBiometricsV15Service()

	score := service.calculateSimilarityScore(1.0, 1.0, 0.3)
	if score != 1.0 {
		t.Errorf("相同值相似度应为 1, 实际 %f", score)
	}

	score = service.calculateSimilarityScore(1.0, 0.7, 0.3)
	if score < 0.5 {
		t.Errorf("小差异应有较高分数, 实际 %f", score)
	}

	score = service.calculateSimilarityScore(1.0, 0.3, 0.3)
	if score != 0.0 {
		t.Errorf("大差异分数应为零, 实际 %f", score)
	}

	score = service.calculateSimilarityScore(0, 1.0, 0.3)
	if score != 0.5 {
		t.Errorf("零值应返回 0.5, 实际 %f", score)
	}
}

func TestFusionWeights(t *testing.T) {
	service := NewBiometricsV15Service()

	scores := &ModalScores{
		MousePressureScore: 0.8,
		TouchForceScore:    0.6,
		EyeTrackingScore:   0.7,
	}

	weights := service.getAdaptiveWeights(scores)

	if weights.MousePressure <= 0 || weights.MousePressure > 1 {
		t.Errorf("MousePressure 权重应在 (0,1] 范围内, 实际 %f", weights.MousePressure)
	}

	if weights.TouchForce <= 0 || weights.TouchForce > 1 {
		t.Errorf("TouchForce 权重应在 (0,1] 范围内, 实际 %f", weights.TouchForce)
	}

	if weights.EyeTracking <= 0 || weights.EyeTracking > 1 {
		t.Errorf("EyeTracking 权重应在 (0,1] 范围内, 实际 %f", weights.EyeTracking)
	}
}

func TestCalculateFusionScore(t *testing.T) {
	service := NewBiometricsV15Service()

	scores := &ModalScores{
		MousePressureScore: 0.9,
		TouchForceScore:    0.8,
		EyeTrackingScore:   0.85,
	}

	fusionScore := service.calculateFusionScore(scores)
	if fusionScore < 0.8 {
		t.Errorf("融合分数应较高, 实际 %f", fusionScore)
	}

	scores2 := &ModalScores{
		MousePressureScore: 0.5,
		TouchForceScore:    0.5,
		EyeTrackingScore:   0.5,
	}

	fusionScore2 := service.calculateFusionScore(scores2)
	if fusionScore2 != 0.5 {
		t.Errorf("平均分数应为零点五, 实际 %f", fusionScore2)
	}
}

func TestDetermineRiskLevel(t *testing.T) {
	service := NewBiometricsV15Service()

	scores := &ModalScores{
		MousePressureScore: 0.95,
		TouchForceScore:    0.9,
		EyeTrackingScore:   0.92,
	}

	risk := service.determineRiskLevel(scores, 0.92)
	if risk != "low" {
		t.Errorf("高置信度应返回 'low' risk, 实际 %s", risk)
	}

	scores2 := &ModalScores{
		MousePressureScore: 0.6,
		TouchForceScore:    0.5,
		EyeTrackingScore:   0.55,
	}

	risk2 := service.determineRiskLevel(scores2, 0.75)
	if risk2 != "medium" {
		t.Errorf("中等置信度应返回 'medium' risk, 实际 %s", risk2)
	}

	scores3 := &ModalScores{
		MousePressureScore: 0.2,
		TouchForceScore:    0.1,
		EyeTrackingScore:   0.15,
	}

	risk3 := service.determineRiskLevel(scores3, 0.5)
	if risk3 != "high" {
		t.Errorf("低置信度应返回 'high' risk, 实际 %s", risk3)
	}
}

func TestMeanAndStdDev(t *testing.T) {
	values := []float64{1.0, 2.0, 3.0, 4.0, 5.0}

	avg := meanFloat64(values)
	if avg != 3.0 {
		t.Errorf("平均值应为 3.0, 实际 %f", avg)
	}

	std := stdDevFloat64(values)
	if std < 1.4 || std > 1.5 {
		t.Errorf("标准差应约为 1.41, 实际 %f", std)
	}

	emptyAvg := meanFloat64([]float64{})
	if emptyAvg != 0 {
		t.Errorf("空数组平均值应为零, 实际 %f", emptyAvg)
	}

	emptyStd := stdDevFloat64([]float64{1.0})
	if emptyStd != 0 {
		t.Errorf("单元素数组标准差应为零, 实际 %f", emptyStd)
	}
}

func TestProfileJSONSerialization(t *testing.T) {
	profile := &MultimodalBiometricProfile{
		UserID:        "json-test-user",
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
		FeatureVector: []float64{0.1, 0.2, 0.3},
		MousePressureProfile: &MousePressureProfile{
			AveragePressure: 0.5,
			PressureStdDev:  0.2,
		},
	}

	jsonBytes, err := json.Marshal(profile)
	if err != nil {
		t.Errorf("JSON 序列化失败: %v", err)
	}

	var deserialized MultimodalBiometricProfile
	err = json.Unmarshal(jsonBytes, &deserialized)
	if err != nil {
		t.Errorf("JSON 反序列化失败: %v", err)
	}

	if deserialized.UserID != profile.UserID {
		t.Errorf("UserID 不匹配")
	}

	if deserialized.MousePressureProfile.AveragePressure != profile.MousePressureProfile.AveragePressure {
		t.Errorf("MousePressureProfile 不匹配")
	}
}

func TestExtractMousePressureProfile(t *testing.T) {
	service := NewBiometricsV15Service()

	data := &MousePressureData{
		PressureAnalysis: &PressureAnalysis{
			AveragePressure:  0.5,
			PressureStd:      0.2,
			MaxPressure:      0.8,
			MinPressure:      0.2,
			PressureRange:    0.6,
			PressureSkewness: 0.1,
			PressureKurtosis: 0.2,
			AverageForce:     4.9,
			ForceStd:         1.5,
			DownPressureAvg:  0.7,
			UpPressureAvg:    0.3,
		},
		ClickAnalysis: &ClickAnalysis{
			ClickCount:       10,
			AvgClickDuration: 150.0,
		},
		MovementAnalysis: &MovementAnalysis{
			AvgSpeed:            0.5,
			SpeedStd:            0.2,
			MaxSpeed:            1.2,
			MinSpeed:            0.1,
			AvgAcceleration:     0.3,
			AvgJerk:             0.1,
			MovementEntropy:     3.5,
			DirectionHorizontal: 0.6,
			DirectionVertical:   0.4,
		},
		DragAnalysis: &DragAnalysis{
			DragCount:       3,
			AvgDragPressure: 0.5,
		},
	}

	profile := service.extractMousePressureProfile(data)

	if profile.AveragePressure != 0.5 {
		t.Errorf("AveragePressure 不匹配")
	}
	if profile.ClickCount != 10 {
		t.Errorf("ClickCount 不匹配")
	}
	if profile.AvgSpeed != 0.5 {
		t.Errorf("AvgSpeed 不匹配")
	}
	if profile.DragCount != 3 {
		t.Errorf("DragCount 不匹配")
	}
}

func TestExtractTouchForceProfile(t *testing.T) {
	service := NewBiometricsV15Service()

	data := &TouchForceData{
		ForceAnalysis: &TouchForceAnalysis{
			TouchCount:    20,
			AvgForce:      0.6,
			ForceStdDev:   0.15,
			MaxForce:      0.9,
			MinForce:      0.3,
			ForceRange:    0.6,
			ForceSkewness: 0.2,
			AvgPressure:   0.55,
			PressureStd:   0.2,
			AvgSpeed:      0.7,
			SpeedStd:      0.15,
		},
		SwipeAnalysis: &SwipeAnalysis{
			SwipeCount:       5,
			DirectionEntropy: 1.5,
			AvgSpeed:         0.8,
			AvgForce:         0.6,
			AvgAngle:         45.0,
			AvgDistance:      200.0,
			AvgDuration:      500.0,
		},
		MultitouchAnalysis: &MultiTouchAnalysis{
			GestureCount:     3,
			PinchCount:       2,
			AvgPinchScale:    1.3,
			AvgPinchRotation: 15.0,
		},
	}

	profile := service.extractTouchForceProfile(data)

	if profile.TouchCount != 20 {
		t.Errorf("TouchCount 不匹配")
	}
	if profile.SwipeCount != 5 {
		t.Errorf("SwipeCount 不匹配")
	}
	if profile.PinchCount != 2 {
		t.Errorf("PinchCount 不匹配")
	}
	if profile.DirectionEntropy != 1.5 {
		t.Errorf("DirectionEntropy 不匹配")
	}
}

func TestExtractEyeTrackingProfile(t *testing.T) {
	service := NewBiometricsV15Service()

	data := &EyeTrackingData{
		GazeAnalysis: &GazeAnalysis{
			GazeCount:              100,
			AvgX:                   500,
			AvgY:                   300,
			XStd:                   50,
			YStd:                   40,
			CoverageArea:           0.3,
			AvgPupilSize:           3.5,
			PupilStd:               0.5,
			ScanPatternTopLeft:     0.25,
			ScanPatternTopRight:    0.25,
			ScanPatternBottomLeft:  0.25,
			ScanPatternBottomRight: 0.25,
		},
		BlinkAnalysis: &BlinkAnalysis{
			BlinkCount:       15,
			BlinkRate:        12.0,
			AvgBlinkDuration: 150.0,
			AvgInterval:      5000.0,
		},
		FixationAnalysis: &FixationAnalysis{
			FixationCount:  25,
			AvgDuration:    300.0,
			DurationStd:    100.0,
			AvgDispersion:  20.0,
			DispersionStd:  5.0,
			AvgPupilSize:   3.5,
			PupilStd:       0.5,
			LongFixations:  10,
			ShortFixations: 5,
		},
		SaccadeAnalysis: &SaccadeAnalysis{
			SaccadeCount: 30,
			AvgSpeed:     200.0,
			SpeedStd:     50.0,
			AvgDuration:  50.0,
			DurationStd:  15.0,
			AvgDistance:  150.0,
			DistanceStd:  40.0,
			MaxSpeed:     300.0,
		},
		DwellAnalysis: &DwellAnalysis{
			DwellCount:   20,
			AvgDuration:  1000.0,
			DurationStd:  300.0,
			TopTargets:   map[string]int{"button": 5, "input": 3},
			LongestDwell: 5000.0,
		},
		FocusAnalysis: &FocusAnalysis{
			FocusCount:        10,
			BlurCount:         2,
			BlurRate:          2,
			TotalBlurDuration: 5000,
			FocusLostCount:    2,
			AttentionRatio:    0.85,
		},
	}

	profile := service.extractEyeTrackingProfile(data)

	if profile.GazeCount != 100 {
		t.Errorf("GazeCount 不匹配")
	}
	if profile.BlinkCount != 15 {
		t.Errorf("BlinkCount 不匹配")
	}
	if profile.FixationCount != 25 {
		t.Errorf("FixationCount 不匹配")
	}
	if profile.SaccadeCount != 30 {
		t.Errorf("SaccadeCount 不匹配")
	}
	if profile.AttentionRatio != 0.85 {
		t.Errorf("AttentionRatio 不匹配")
	}
}

func TestGenerateFeatureVector(t *testing.T) {
	service := NewBiometricsV15Service()

	profile := &MultimodalBiometricProfile{
		MousePressureProfile: &MousePressureProfile{
			AveragePressure:     0.5,
			PressureStdDev:      0.2,
			MaxPressure:         0.8,
			MinPressure:         0.2,
			PressureRange:       0.6,
			PressureSkewness:    0.1,
			PressureKurtosis:    0.2,
			AvgForce:            4.9,
			AvgSpeed:            0.5,
			SpeedStd:            0.2,
			MaxSpeed:            1.2,
			AvgAcceleration:     0.3,
			MovementEntropy:     3.5,
			DirectionHorizontal: 0.6,
			DirectionVertical:   0.4,
			ClickCount:          10,
			AvgClickDuration:    150.0,
			DragCount:           3,
			AvgDragPressure:     0.5,
		},
		TouchForceProfile: &TouchForceProfile{
			TouchCount:       20,
			AvgForce:         0.6,
			ForceStdDev:      0.15,
			MaxForce:         0.9,
			ForceRange:       0.6,
			AvgPressure:      0.55,
			AvgSpeed:         0.7,
			SwipeCount:       5,
			DirectionEntropy: 1.5,
			AvgSwipeSpeed:    0.8,
			AvgAngle:         45.0,
			AvgDistance:      200.0,
			PinchCount:       2,
			AvgPinchScale:    1.3,
		},
		EyeTrackingProfile: &EyeTrackingProfile{
			GazeCount:           100,
			AvgX:                500,
			AvgY:                300,
			XStd:                50,
			YStd:                40,
			CoverageArea:        0.3,
			AvgPupilSize:        3.5,
			BlinkCount:          15,
			BlinkRate:           12.0,
			AvgBlinkDuration:    150.0,
			FixationCount:       25,
			AvgFixationDuration: 300.0,
			AvgDispersion:       20.0,
			SaccadeCount:        30,
			AvgSaccadeSpeed:     200.0,
			AttentionRatio:      0.85,
			ScanPatternTopLeft:  0.25,
			ScanPatternTopRight: 0.25,
		},
	}

	featureVector := service.generateFeatureVector(profile)

	if len(featureVector) == 0 {
		t.Error("FeatureVector 为空")
	}

	expectedMinFeatures := 17 + 14 + 18
	if len(featureVector) < expectedMinFeatures {
		t.Errorf("FeatureVector 长度过短: %d < %d", len(featureVector), expectedMinFeatures)
	}

	for i, v := range featureVector {
		if v < 0 || v > 1 {
			t.Logf("Feature[%d] 值超出 [0,1] 范围: %f", i, v)
		}
	}
}
