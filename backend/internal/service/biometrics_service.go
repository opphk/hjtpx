package service

import (
	"encoding/json"
	"fmt"
	"math"
	"time"
)

// BiometricProfile 生物识别特征档案
type BiometricProfile struct {
	UserID            string             `json:"user_id"`
	CreatedAt         time.Time          `json:"created_at"`
	UpdatedAt         time.Time          `json:"updated_at"`
	KeyboardProfile   KeyboardBiometrics `json:"keyboard_profile"`
	MouseProfile      MouseBiometrics    `json:"mouse_profile"`
	VerificationCount int                `json:"verification_count"`
	ConfidenceScore   float64            `json:"confidence_score"`
}

// KeyboardBiometrics 键盘生物特征
type KeyboardBiometrics struct {
	AverageHoldTime   float64            `json:"average_hold_time"`
	HoldTimeStdDev    float64            `json:"hold_time_std_dev"`
	AverageFlightTime float64            `json:"average_flight_time"`
	FlightTimeStdDev  float64            `json:"flight_time_std_dev"`
	TypingSpeed       float64            `json:"typing_speed"`
	KeyPairTimings    map[string]float64 `json:"key_pair_timings"`
	CommonKeys        map[string]float64 `json:"common_keys"`
	ErrorRate         float64            `json:"error_rate"`
}

// MouseBiometrics 鼠标生物特征
type MouseBiometrics struct {
	AverageSpeed        float64 `json:"average_speed"`
	SpeedStdDev         float64 `json:"speed_std_dev"`
	AccelerationPattern float64 `json:"acceleration_pattern"`
	PathEfficiency      float64 `json:"path_efficiency"`
	CurvatureAverage    float64 `json:"curvature_average"`
	ClickTiming         float64 `json:"click_timing"`
	ClickPrecision      float64 `json:"click_precision"`
	MotionEntropy       float64 `json:"motion_entropy"`
}

// KeyboardSample 键盘输入样本
type KeyboardSample struct {
	KeyEvents []KeyEvent `json:"key_events"`
	Timestamp int64      `json:"timestamp"`
}

// KeyEvent 按键事件
type KeyEvent struct {
	Key       string `json:"key"`
	Type      string `json:"type"` // keydown, keyup
	Timestamp int64  `json:"timestamp"`
	KeyCode   int    `json:"key_code"`
}

// MouseSample 鼠标移动样本
type MouseSample struct {
	MouseEvents []MouseEvent `json:"mouse_events"`
	Timestamp   int64        `json:"timestamp"`
}

// MouseEvent 鼠标事件
type MouseEvent struct {
	Type      string `json:"type"` // mousemove, click, mousedown, mouseup
	X         int    `json:"x"`
	Y         int    `json:"y"`
	Timestamp int64  `json:"timestamp"`
	Button    int    `json:"button,omitempty"`
}

// BiometricVerificationResult 生物识别验证结果
type BiometricVerificationResult struct {
	IsVerified    bool    `json:"is_verified"`
	Confidence    float64 `json:"confidence"`
	KeyboardScore float64 `json:"keyboard_score"`
	MouseScore    float64 `json:"mouse_score"`
	Details       string  `json:"details"`
}

// BiometricsService 生物识别服务
type BiometricsService struct {
	profiles map[string]*BiometricProfile
}

// NewBiometricsService 创建新的生物识别服务
func NewBiometricsService() *BiometricsService {
	return &BiometricsService{
		profiles: make(map[string]*BiometricProfile),
	}
}

// RegisterProfile 注册或更新生物识别档案
func (s *BiometricsService) RegisterProfile(userID string, keyboardSample *KeyboardSample, mouseSample *MouseSample) (*BiometricProfile, error) {
	if userID == "" {
		return nil, fmt.Errorf("user ID is required")
	}

	profile, exists := s.profiles[userID]
	if !exists {
		profile = &BiometricProfile{
			UserID:    userID,
			CreatedAt: time.Now(),
			KeyboardProfile: KeyboardBiometrics{
				KeyPairTimings: make(map[string]float64),
				CommonKeys:     make(map[string]float64),
			},
		}
	}

	profile.UpdatedAt = time.Now()

	if keyboardSample != nil && len(keyboardSample.KeyEvents) > 0 {
		keyboardProfile := s.extractKeyboardFeatures(keyboardSample)
		profile.KeyboardProfile = keyboardProfile
	}

	if mouseSample != nil && len(mouseSample.MouseEvents) > 0 {
		mouseProfile := s.extractMouseFeatures(mouseSample)
		profile.MouseProfile = mouseProfile
	}

	profile.VerificationCount++
	profile.ConfidenceScore = math.Min(1.0, float64(profile.VerificationCount)/10.0)

	s.profiles[userID] = profile
	return profile, nil
}

// Verify 生物特征验证
func (s *BiometricsService) Verify(userID string, keyboardSample *KeyboardSample, mouseSample *MouseSample) (*BiometricVerificationResult, error) {
	profile, exists := s.profiles[userID]
	if !exists {
		return &BiometricVerificationResult{
			IsVerified: false,
			Confidence: 0,
			Details:    "No profile found for user",
		}, nil
	}

	var keyboardScore float64 = 0.5
	var mouseScore float64 = 0.5

	if keyboardSample != nil && len(keyboardSample.KeyEvents) > 0 {
		sampleFeatures := s.extractKeyboardFeatures(keyboardSample)
		keyboardScore = s.compareKeyboardBiometrics(profile.KeyboardProfile, sampleFeatures)
	}

	if mouseSample != nil && len(mouseSample.MouseEvents) > 0 {
		sampleFeatures := s.extractMouseFeatures(mouseSample)
		mouseScore = s.compareMouseBiometrics(profile.MouseProfile, sampleFeatures)
	}

	overallConfidence := (keyboardScore*0.6 + mouseScore*0.4)
	isVerified := overallConfidence >= 0.95

	result := &BiometricVerificationResult{
		IsVerified:    isVerified,
		Confidence:    overallConfidence,
		KeyboardScore: keyboardScore,
		MouseScore:    mouseScore,
		Details:       fmt.Sprintf("Verification with %.2f%% confidence", overallConfidence*100),
	}

	return result, nil
}

// extractKeyboardFeatures 提取键盘生物特征
func (s *BiometricsService) extractKeyboardFeatures(sample *KeyboardSample) KeyboardBiometrics {
	features := KeyboardBiometrics{
		KeyPairTimings: make(map[string]float64),
		CommonKeys:     make(map[string]float64),
	}

	if len(sample.KeyEvents) < 4 {
		return features
	}

	holdTimes := []float64{}
	flightTimes := []float64{}
	keyDownMap := make(map[string]int64)
	keyCount := make(map[string]int)

	for i := 0; i < len(sample.KeyEvents); i++ {
		event := sample.KeyEvents[i]
		key := fmt.Sprintf("%s:%d", event.Key, event.KeyCode)

		if event.Type == "keydown" {
			keyDownMap[key] = event.Timestamp
			keyCount[key]++
		} else if event.Type == "keyup" {
			if downTime, exists := keyDownMap[key]; exists {
				holdTime := float64(event.Timestamp - downTime)
				if holdTime > 0 {
					holdTimes = append(holdTimes, holdTime)
				}
				delete(keyDownMap, key)
			}
		}

		// 计算飞行时间（按键间隔）
		if i > 0 && event.Type == "keydown" && sample.KeyEvents[i-1].Type == "keydown" {
			flightTime := float64(event.Timestamp - sample.KeyEvents[i-1].Timestamp)
			if flightTime > 0 {
				flightTimes = append(flightTimes, flightTime)

				// 记录按键对时序
				prevKey := fmt.Sprintf("%s:%d", sample.KeyEvents[i-1].Key, sample.KeyEvents[i-1].KeyCode)
				pairKey := fmt.Sprintf("%s→%s", prevKey, key)
				features.KeyPairTimings[pairKey] = flightTime
			}
		}
	}

	// 计算统计值
	if len(holdTimes) > 0 {
		features.AverageHoldTime = mean(holdTimes)
		features.HoldTimeStdDev = stdDev(holdTimes)
	}

	if len(flightTimes) > 0 {
		features.AverageFlightTime = mean(flightTimes)
		features.FlightTimeStdDev = stdDev(flightTimes)
		features.TypingSpeed = float64(len(flightTimes)) / (float64(flightTimes[len(flightTimes)-1]-flightTimes[0]) / 1000)
	}

	// 计算常用键分布
	totalKeys := 0
	for _, count := range keyCount {
		totalKeys += count
	}
	if totalKeys > 0 {
		for key, count := range keyCount {
			features.CommonKeys[key] = float64(count) / float64(totalKeys)
		}
	}

	return features
}

// extractMouseFeatures 提取鼠标生物特征
func (s *BiometricsService) extractMouseFeatures(sample *MouseSample) MouseBiometrics {
	features := MouseBiometrics{}

	moveEvents := []MouseEvent{}
	clickEvents := []MouseEvent{}

	for _, event := range sample.MouseEvents {
		if event.Type == "mousemove" {
			moveEvents = append(moveEvents, event)
		} else if event.Type == "click" || event.Type == "mousedown" {
			clickEvents = append(clickEvents, event)
		}
	}

	if len(moveEvents) < 3 {
		return features
	}

	// 计算速度相关
	speeds := []float64{}
	accelerations := []float64{}
	totalDistance := 0.0
	straightDistance := 0.0
	curvatures := []float64{}

	for i := 1; i < len(moveEvents); i++ {
		dx := float64(moveEvents[i].X - moveEvents[i-1].X)
		dy := float64(moveEvents[i].Y - moveEvents[i-1].Y)
		distance := math.Sqrt(dx*dx + dy*dy)
		dt := float64(moveEvents[i].Timestamp - moveEvents[i-1].Timestamp)

		if dt > 0 {
			speed := distance / dt
			speeds = append(speeds, speed)
			totalDistance += distance
		}

		// 计算曲率
		if i > 1 {
			curvature := s.calculateCurvature(
				moveEvents[i-2],
				moveEvents[i-1],
				moveEvents[i],
			)
			curvatures = append(curvatures, curvature)
		}
	}

	// 计算加速度
	for i := 1; i < len(speeds); i++ {
		dt := float64(moveEvents[i+1].Timestamp - moveEvents[i-1].Timestamp)
		if dt > 0 {
			accel := (speeds[i] - speeds[i-1]) / dt
			accelerations = append(accelerations, accel)
		}
	}

	// 计算直线路径距离
	if len(moveEvents) > 0 {
		first := moveEvents[0]
		last := moveEvents[len(moveEvents)-1]
		dx := float64(last.X - first.X)
		dy := float64(last.Y - first.Y)
		straightDistance = math.Sqrt(dx*dx + dy*dy)
	}

	if len(speeds) > 0 {
		features.AverageSpeed = mean(speeds)
		features.SpeedStdDev = stdDev(speeds)
	}

	if len(accelerations) > 0 {
		features.AccelerationPattern = mean(accelerations)
	}

	if totalDistance > 0 {
		features.PathEfficiency = straightDistance / totalDistance
	}

	if len(curvatures) > 0 {
		features.CurvatureAverage = mean(curvatures)
	}

	// 计算点击特征
	if len(clickEvents) > 1 {
		clickIntervals := []float64{}
		for i := 1; i < len(clickEvents); i++ {
			interval := float64(clickEvents[i].Timestamp - clickEvents[i-1].Timestamp)
			clickIntervals = append(clickIntervals, interval)
		}
		features.ClickTiming = mean(clickIntervals)
	}

	// 计算运动熵
	features.MotionEntropy = s.calculateMotionEntropy(moveEvents)

	return features
}

// compareKeyboardBiometrics 比较键盘生物特征相似度
func (s *BiometricsService) compareKeyboardBiometrics(profile, sample KeyboardBiometrics) float64 {
	score := 0.0
	weights := 0.0

	// 平均保持时间 (25%)
	if profile.AverageHoldTime > 0 && sample.AverageHoldTime > 0 {
		holdScore := s.calculateSimilarityScore(profile.AverageHoldTime, sample.AverageHoldTime, 0.3)
		score += holdScore * 0.25
		weights += 0.25
	}

	// 保持时间标准差 (15%)
	if profile.HoldTimeStdDev > 0 && sample.HoldTimeStdDev > 0 {
		holdDevScore := s.calculateSimilarityScore(profile.HoldTimeStdDev, sample.HoldTimeStdDev, 0.5)
		score += holdDevScore * 0.15
		weights += 0.15
	}

	// 平均飞行时间 (25%)
	if profile.AverageFlightTime > 0 && sample.AverageFlightTime > 0 {
		flightScore := s.calculateSimilarityScore(profile.AverageFlightTime, sample.AverageFlightTime, 0.3)
		score += flightScore * 0.25
		weights += 0.25
	}

	// 飞行时间标准差 (15%)
	if profile.FlightTimeStdDev > 0 && sample.FlightTimeStdDev > 0 {
		flightDevScore := s.calculateSimilarityScore(profile.FlightTimeStdDev, sample.FlightTimeStdDev, 0.5)
		score += flightDevScore * 0.15
		weights += 0.15
	}

	// 打字速度 (10%)
	if profile.TypingSpeed > 0 && sample.TypingSpeed > 0 {
		speedScore := s.calculateSimilarityScore(profile.TypingSpeed, sample.TypingSpeed, 0.4)
		score += speedScore * 0.10
		weights += 0.10
	}

	// 按键对时序比较 (10%)
	if len(profile.KeyPairTimings) > 0 && len(sample.KeyPairTimings) > 0 {
		pairScore := s.compareKeyPairTimings(profile.KeyPairTimings, sample.KeyPairTimings)
		score += pairScore * 0.10
		weights += 0.10
	}

	if weights > 0 {
		return score / weights
	}

	return 0.5
}

// compareMouseBiometrics 比较鼠标生物特征相似度
func (s *BiometricsService) compareMouseBiometrics(profile, sample MouseBiometrics) float64 {
	score := 0.0
	weights := 0.0

	// 平均速度 (20%)
	if profile.AverageSpeed > 0 && sample.AverageSpeed > 0 {
		speedScore := s.calculateSimilarityScore(profile.AverageSpeed, sample.AverageSpeed, 0.4)
		score += speedScore * 0.20
		weights += 0.20
	}

	// 速度标准差 (15%)
	if profile.SpeedStdDev > 0 && sample.SpeedStdDev > 0 {
		speedDevScore := s.calculateSimilarityScore(profile.SpeedStdDev, sample.SpeedStdDev, 0.5)
		score += speedDevScore * 0.15
		weights += 0.15
	}

	// 路径效率 (25%)
	if profile.PathEfficiency > 0 && sample.PathEfficiency > 0 {
		efficiencyScore := 1.0 - math.Abs(profile.PathEfficiency-sample.PathEfficiency)
		score += math.Max(0, efficiencyScore) * 0.25
		weights += 0.25
	}

	// 平均曲率 (15%)
	if profile.CurvatureAverage > 0 && sample.CurvatureAverage > 0 {
		curvatureScore := s.calculateSimilarityScore(profile.CurvatureAverage, sample.CurvatureAverage, 0.5)
		score += curvatureScore * 0.15
		weights += 0.15
	}

	// 点击时序 (10%)
	if profile.ClickTiming > 0 && sample.ClickTiming > 0 {
		clickScore := s.calculateSimilarityScore(profile.ClickTiming, sample.ClickTiming, 0.4)
		score += clickScore * 0.10
		weights += 0.10
	}

	// 运动熵 (15%)
	if profile.MotionEntropy > 0 && sample.MotionEntropy > 0 {
		entropyScore := 1.0 - math.Abs(profile.MotionEntropy-sample.MotionEntropy)
		score += math.Max(0, entropyScore) * 0.15
		weights += 0.15
	}

	if weights > 0 {
		return score / weights
	}

	return 0.5
}

// calculateSimilarityScore 计算相似度分数
func (s *BiometricsService) calculateSimilarityScore(value1, value2, maxDiffRatio float64) float64 {
	if value1 <= 0 || value2 <= 0 {
		return 0.5
	}

	diffRatio := math.Abs(value1-value2) / math.Max(value1, value2)

	if diffRatio <= maxDiffRatio {
		return 1.0 - (diffRatio / maxDiffRatio)
	}

	return 0.0
}

// compareKeyPairTimings 比较按键对时序
func (s *BiometricsService) compareKeyPairTimings(pairs1, pairs2 map[string]float64) float64 {
	if len(pairs1) == 0 || len(pairs2) == 0 {
		return 0.5
	}

	matchingPairs := 0
	totalScore := 0.0

	for key, time1 := range pairs1 {
		if time2, exists := pairs2[key]; exists {
			score := s.calculateSimilarityScore(time1, time2, 0.4)
			totalScore += score
			matchingPairs++
		}
	}

	if matchingPairs > 0 {
		return totalScore / float64(matchingPairs)
	}

	return 0.3
}

// calculateCurvature 计算曲率
func (s *BiometricsService) calculateCurvature(p1, p2, p3 MouseEvent) float64 {
	v1x := float64(p2.X - p1.X)
	v1y := float64(p2.Y - p1.Y)
	v2x := float64(p3.X - p2.X)
	v2y := float64(p3.Y - p2.Y)

	dot := v1x*v2x + v1y*v2y
	mag1 := math.Sqrt(v1x*v1x + v1y*v1y)
	mag2 := math.Sqrt(v2x*v2x + v2y*v2y)

	if mag1 == 0 || mag2 == 0 {
		return 0
	}

	cosAngle := dot / (mag1 * mag2)
	if cosAngle > 1 {
		cosAngle = 1
	} else if cosAngle < -1 {
		cosAngle = -1
	}

	return math.Acos(cosAngle)
}

// calculateMotionEntropy 计算运动熵
func (s *BiometricsService) calculateMotionEntropy(events []MouseEvent) float64 {
	if len(events) < 2 {
		return 0
	}

	// 分桶统计
	bucketSize := 50
	buckets := make(map[string]int)

	for _, event := range events {
		bucketX := event.X / bucketSize
		bucketY := event.Y / bucketSize
		key := fmt.Sprintf("%d:%d", bucketX, bucketY)
		buckets[key]++
	}

	total := len(events)
	entropy := 0.0

	for _, count := range buckets {
		if count > 0 {
			prob := float64(count) / float64(total)
			entropy -= prob * math.Log2(prob)
		}
	}

	return entropy
}

// mean 计算平均值
func mean(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

// stdDev 计算标准差
func stdDev(values []float64) float64 {
	if len(values) < 2 {
		return 0
	}
	avg := mean(values)
	variance := 0.0
	for _, v := range values {
		variance += math.Pow(v-avg, 2)
	}
	return math.Sqrt(variance / float64(len(values)))
}

// SerializeProfile 序列化档案
func (p *BiometricProfile) SerializeProfile() ([]byte, error) {
	return json.Marshal(p)
}

// DeserializeProfile 反序列化档案
func (s *BiometricsService) DeserializeProfile(data []byte) (*BiometricProfile, error) {
	var profile BiometricProfile
	err := json.Unmarshal(data, &profile)
	if err != nil {
		return nil, err
	}
	return &profile, nil
}
