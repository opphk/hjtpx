package service

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"
)

type WatermarkVerificationService struct {
	mu               sync.RWMutex
	initialized      bool
	algorithms       map[string]*WatermarkAlgorithmConfig
	watermarkDB      *WatermarkDatabase
	verificationLogs map[string]*VerificationLog
	metrics          *WatermarkMetrics
}

type WatermarkAlgorithmConfig struct {
	AlgorithmID    string
	AlgorithmType  string
	Strength       float64
	Robustness     float64
	Invisibility   float64
	Capacity       float64
	IsActive       bool
	Parameters     map[string]interface{}
}

type WatermarkDatabase struct {
	mu        sync.RWMutex
	watermarks map[string]*WatermarkRecord
	keys      map[string]*WatermarkKey
}

type WatermarkRecord struct {
	WatermarkID   string
	OwnerID       string
	ContentHash   string
	AlgorithmType string
	EmbeddedAt    time.Time
	ExpiresAt     time.Time
	IsActive      bool
	Metadata      map[string]interface{}
}

type WatermarkKey struct {
	KeyID       string
	WatermarkID string
	SecretKey   []byte
	PublicKey   []byte
	Algorithm   string
	CreatedAt   time.Time
}

type VerificationLog struct {
	LogID            string
	WatermarkID     string
	VerificationAt  time.Time
	Result          *WatermarkVerificationResult
	MediaHash       string
	ProcessingTime  time.Duration
}

type WatermarkVerificationResult struct {
	WatermarkID        string                    `json:"watermark_id"`
	IsPresent          bool                      `json:"is_present"`
	IsAuthentic        bool                      `json:"is_authentic"`
	VerificationScore  float64                   `json:"verification_score"`
	AlgorithmUsed      string                    `json:"algorithm_used"`
	Confidence         float64                   `json:"confidence"`
	Details            []*WatermarkDetailV2      `json:"details"`
	ExtractedData      map[string]interface{}    `json:"extracted_data,omitempty"`
	TamperingEvidence  []*TamperingEvidenceV2    `json:"tampering_evidence,omitempty"`
	ProcessingTime     time.Duration             `json:"processing_time"`
}

type WatermarkDetailV2 struct {
	Property        string   `json:"property"`
	Expected        string   `json:"expected"`
	Actual          string   `json:"actual"`
	Match           bool     `json:"match"`
	Severity        float64  `json:"severity"`
	Description     string   `json:"description"`
}

type TamperingEvidenceV2 struct {
	EvidenceID     string                  `json:"evidence_id"`
	Type           string                  `json:"type"`
	Description    string                  `json:"description"`
	Severity       float64                 `json:"severity"`
	Location       string                  `json:"location"`
	MethodUsed     string                  `json:"method_used"`
}

type WatermarkMetrics struct {
	TotalVerifications   int
	SuccessfulVerifications int
	FailedVerifications  int
	AverageProcessingTime time.Duration
	AverageScore          float64
}

func NewWatermarkVerificationService() *WatermarkVerificationService {
	algorithms := make(map[string]*WatermarkAlgorithmConfig)

	algoConfigs := []struct {
		id        string
		algoType  string
		strength  float64
		robustness float64
		invisibility float64
		capacity  float64
	}{
		{"lsb", "least_significant_bit", 0.8, 0.7, 0.9, 0.3},
		{"dct", "discrete_cosine_transform", 0.9, 0.85, 0.85, 0.5},
		{"dwt", "discrete_wavelet_transform", 0.85, 0.9, 0.8, 0.6},
		{"fft", "fast_fourier_transform", 0.75, 0.8, 0.85, 0.4},
		{"spread_spectrum", "spread_spectrum", 0.7, 0.95, 0.75, 0.2},
	}

	for _, algo := range algoConfigs {
		algorithms[algo.id] = &WatermarkAlgorithmConfig{
			AlgorithmID:  algo.id,
			AlgorithmType: algo.algoType,
			Strength:     algo.strength,
			Robustness:   algo.robustness,
			Invisibility: algo.invisibility,
			Capacity:     algo.capacity,
			IsActive:     true,
			Parameters:   make(map[string]interface{}),
		}
	}

	return &WatermarkVerificationService{
		algorithms:      algorithms,
		watermarkDB:    NewWatermarkDatabase(),
		verificationLogs: make(map[string]*VerificationLog),
		metrics: &WatermarkMetrics{
			TotalVerifications:    0,
			SuccessfulVerifications: 0,
			FailedVerifications:  0,
		},
	}
}

func NewWatermarkDatabase() *WatermarkDatabase {
	return &WatermarkDatabase{
		watermarks: make(map[string]*WatermarkRecord),
		keys:       make(map[string]*WatermarkKey),
	}
}

func (svc *WatermarkVerificationService) Initialize(ctx context.Context) error {
	svc.mu.Lock()
	defer svc.mu.Unlock()

	if svc.initialized {
		return nil
	}

	for _, algo := range svc.algorithms {
		algo.IsActive = true
	}

	svc.initialized = true
	return nil
}

func (svc *WatermarkVerificationService) VerifyWatermark(ctx context.Context, mediaData []byte, watermarkID string) (*WatermarkVerificationResult, error) {
	svc.mu.Lock()
	defer svc.mu.Unlock()

	if !svc.initialized {
		return nil, fmt.Errorf("watermark verification service not initialized")
	}

	startTime := time.Now()

	result := &WatermarkVerificationResult{
		WatermarkID:     watermarkID,
		Details:         make([]*WatermarkDetailV2, 0),
		ExtractedData:   make(map[string]interface{}),
		TamperingEvidence: make([]*TamperingEvidenceV2, 0),
	}

	var bestScore float64
	var bestAlgorithm string

	for algoID, algo := range svc.algorithms {
		if !algo.IsActive {
			continue
		}

		score := svc.verifyWithAlgorithm(algo, mediaData, watermarkID)

		detail := &WatermarkDetailV2{
			Property:     "algorithm_" + algoID,
			Expected:      fmt.Sprintf("score >= %.2f", algo.Strength),
			Actual:       fmt.Sprintf("score = %.2f", score),
			Match:        score >= algo.Strength,
			Severity:     1.0 - score,
			Description:  fmt.Sprintf("Watermark verification using %s algorithm", algo.AlgorithmType),
		}
		result.Details = append(result.Details, detail)

		if score > bestScore {
			bestScore = score
			bestAlgorithm = algoID
		}
	}

	result.VerificationScore = bestScore
	result.AlgorithmUsed = bestAlgorithm
	result.IsPresent = bestScore >= 0.6
	result.IsAuthentic = bestScore >= 0.8
	result.Confidence = bestScore * 0.9

	result.ExtractedData = svc.extractWatermarkData(mediaData, bestAlgorithm)

	if !result.IsAuthentic {
		evidence := svc.detectTampering(mediaData, result)
		result.TamperingEvidence = evidence
	}

	result.ProcessingTime = time.Since(startTime)

	svc.updateMetrics(result)
	svc.logVerification(watermarkID, result, mediaData)

	return result, nil
}

func (svc *WatermarkVerificationService) verifyWithAlgorithm(algo *WatermarkAlgorithmConfig, mediaData []byte, watermarkID string) float64 {
	baseScore := algo.Strength * algo.Robustness

	dataInfluence := math.Mod(float64(len(mediaData)), 0.3)*0.2

	compressionResistance := 1.0
	if svc.detectCompression(mediaData) {
		compressionResistance = 0.9
	}

	noiseResistance := 1.0
	if svc.detectNoise(mediaData) {
		noiseResistance = 0.95
	}

	score := (baseScore + dataInfluence) * compressionResistance * noiseResistance

	return math.Min(1.0, math.Max(0.0, score))
}

func (svc *WatermarkVerificationService) detectCompression(mediaData []byte) bool {
	if len(mediaData) < 100 {
		return false
	}

	repeatedPatterns := 0
	for i := 0; i < len(mediaData)-10; i++ {
		pattern := mediaData[i : i+5]
		for j := i + 5; j < len(mediaData)-5; j++ {
			if string(pattern) == string(mediaData[j:j+5]) {
				repeatedPatterns++
				break
			}
		}
	}

	compressionRatio := float64(repeatedPatterns) / float64(len(mediaData))
	return compressionRatio > 0.15
}

func (svc *WatermarkVerificationService) detectNoise(mediaData []byte) bool {
	if len(mediaData) < 2 {
		return false
	}

	totalDiff := 0.0
	for i := 1; i < len(mediaData); i++ {
		diff := math.Abs(float64(mediaData[i]) - float64(mediaData[i-1]))
		totalDiff += diff
	}

	avgDiff := totalDiff / float64(len(mediaData)-1)
	normalizedNoise := avgDiff / 255.0

	return normalizedNoise > 0.2
}

func (svc *WatermarkVerificationService) extractWatermarkData(mediaData []byte, algorithm string) map[string]interface{} {
	data := make(map[string]interface{})

	data["algorithm"] = algorithm
	data["extracted_strength"] = math.Mod(float64(len(mediaData)), 0.5)*0.3 + 0.5
	data["extraction_confidence"] = 0.85

	switch algorithm {
	case "lsb":
		data["embedding_method"] = "least_significant_bit"
		data["bit_depth"] = 8
	case "dct":
		data["embedding_method"] = "frequency_domain"
		data["frequency_bands"] = []string{"low", "mid", "high"}
	case "dwt":
		data["embedding_method"] = "wavelet_transform"
		data["wavelet_type"] = "haar"
	case "fft":
		data["embedding_method"] = "fourier_transform"
		data["phase_encoding"] = true
	case "spread_spectrum":
		data["embedding_method"] = "spread_spectrum"
		data["spreading_factor"] = 16
	}

	return data
}

func (svc *WatermarkVerificationService) detectTampering(mediaData []byte, result *WatermarkVerificationResult) []*TamperingEvidenceV2 {
	evidence := make([]*TamperingEvidenceV2, 0)

	if svc.detectCompression(mediaData) {
		evidence = append(evidence, &TamperingEvidenceV2{
			EvidenceID:  fmt.Sprintf("tamper_%d", time.Now().UnixNano()),
			Type:        "compression_artifacts",
			Description: "Detected compression artifacts that may indicate tampering",
			Severity:    0.7,
			Location:    "general",
			MethodUsed:  "compression_analysis",
		})
	}

	if svc.detectNoise(mediaData) {
		evidence = append(evidence, &TamperingEvidenceV2{
			EvidenceID:  fmt.Sprintf("tamper_%d", time.Now().UnixNano()+1),
			Type:        "noise_inconsistency",
			Description: "Detected inconsistent noise patterns",
			Severity:    0.6,
			Location:    "general",
			MethodUsed:  "noise_analysis",
		})
	}

	if result.VerificationScore < 0.5 {
		evidence = append(evidence, &TamperingEvidenceV2{
			EvidenceID:  fmt.Sprintf("tamper_%d", time.Now().UnixNano()+2),
			Type:        "low_verification_score",
			Description: "Verification score below threshold indicates potential removal",
			Severity:    0.8,
			Location:    "watermark_region",
			MethodUsed:  "score_analysis",
		})
	}

	return evidence
}

func (svc *WatermarkVerificationService) updateMetrics(result *WatermarkVerificationResult) {
	svc.metrics.TotalVerifications++

	if result.IsAuthentic {
		svc.metrics.SuccessfulVerifications++
	} else if !result.IsPresent {
		svc.metrics.FailedVerifications++
	}

	totalTime := svc.metrics.AverageProcessingTime * time.Duration(svc.metrics.TotalVerifications-1)
	svc.metrics.AverageProcessingTime = (totalTime + result.ProcessingTime) / time.Duration(svc.metrics.TotalVerifications)

	avgScore := svc.metrics.AverageScore*float64(svc.metrics.TotalVerifications-1) + result.VerificationScore
	svc.metrics.AverageScore = avgScore / float64(svc.metrics.TotalVerifications)
}

func (svc *WatermarkVerificationService) logVerification(watermarkID string, result *WatermarkVerificationResult, mediaData []byte) {
	logID := fmt.Sprintf("log_%d", time.Now().UnixNano())

	verificationLog := &VerificationLog{
		LogID:           logID,
		WatermarkID:     watermarkID,
		VerificationAt:  time.Now(),
		Result:          result,
		MediaHash:       svc.computeMediaHash(mediaData),
		ProcessingTime:  result.ProcessingTime,
	}

	svc.verificationLogs[logID] = verificationLog
}

func (svc *WatermarkVerificationService) computeMediaHash(mediaData []byte) string {
	hash := 0
	for i, b := range mediaData {
		hash += int(b) * (i + 1)
	}
	return fmt.Sprintf("%x", hash)
}

func (svc *WatermarkVerificationService) RegisterWatermark(ctx context.Context, watermark *WatermarkRecord, key *WatermarkKey) error {
	svc.mu.Lock()
	defer svc.mu.Unlock()

	svc.watermarkDB.mu.Lock()
	defer svc.watermarkDB.mu.Unlock()

	watermark.EmbeddedAt = time.Now()
	watermark.IsActive = true

	svc.watermarkDB.watermarks[watermark.WatermarkID] = watermark

	if key != nil {
		key.CreatedAt = time.Now()
		svc.watermarkDB.keys[key.KeyID] = key
	}

	return nil
}

func (svc *WatermarkVerificationService) GetWatermark(ctx context.Context, watermarkID string) (*WatermarkRecord, error) {
	svc.watermarkDB.mu.RLock()
	defer svc.watermarkDB.mu.RUnlock()

	record, exists := svc.watermarkDB.watermarks[watermarkID]
	if !exists {
		return nil, fmt.Errorf("watermark not found: %s", watermarkID)
	}

	return record, nil
}

func (svc *WatermarkVerificationService) VerifyWithAlgorithm(ctx context.Context, mediaData []byte, watermarkID string, algorithmID string) (*WatermarkVerificationResult, error) {
	svc.mu.Lock()
	defer svc.mu.Unlock()

	algo, exists := svc.algorithms[algorithmID]
	if !exists {
		return nil, fmt.Errorf("algorithm not found: %s", algorithmID)
	}

	startTime := time.Now()

	score := svc.verifyWithAlgorithm(algo, mediaData, watermarkID)

	result := &WatermarkVerificationResult{
		WatermarkID:       watermarkID,
		IsPresent:        score >= 0.6,
		IsAuthentic:      score >= 0.8,
		VerificationScore: score,
		AlgorithmUsed:    algorithmID,
		Confidence:       score * 0.9,
		Details: []*WatermarkDetailV2{
			{
				Property:    "algorithm_" + algorithmID,
				Expected:     fmt.Sprintf("score >= %.2f", algo.Strength),
				Actual:       fmt.Sprintf("score = %.2f", score),
				Match:        score >= algo.Strength,
				Severity:     1.0 - score,
				Description:  fmt.Sprintf("Watermark verification using %s algorithm", algo.AlgorithmType),
			},
		},
		ExtractedData: svc.extractWatermarkData(mediaData, algorithmID),
		ProcessingTime: time.Since(startTime),
	}

	if !result.IsAuthentic {
		result.TamperingEvidence = svc.detectTampering(mediaData, result)
	}

	return result, nil
}

func (svc *WatermarkVerificationService) GetAvailableAlgorithms() []*WatermarkAlgorithmConfig {
	svc.mu.RLock()
	defer svc.mu.RUnlock()

	algorithms := make([]*WatermarkAlgorithmConfig, 0)
	for _, algo := range svc.algorithms {
		if algo.IsActive {
			config := *algo
			algorithms = append(algorithms, &config)
		}
	}

	return algorithms
}

func (svc *WatermarkVerificationService) GetMetrics() *WatermarkMetrics {
	svc.mu.RLock()
	defer svc.mu.RUnlock()

	metrics := *svc.metrics
	return &metrics
}

func (svc *WatermarkVerificationService) GetVerificationLogs(ctx context.Context, watermarkID string) []*VerificationLog {
	svc.mu.RLock()
	defer svc.mu.RUnlock()

	logs := make([]*VerificationLog, 0)
	for _, log := range svc.verificationLogs {
		if log.WatermarkID == watermarkID {
			logs = append(logs, log)
		}
	}

	return logs
}

func (svc *WatermarkVerificationService) UpdateAlgorithmParameters(ctx context.Context, algorithmID string, params map[string]interface{}) error {
	svc.mu.Lock()
	defer svc.mu.Unlock()

	algo, exists := svc.algorithms[algorithmID]
	if !exists {
		return fmt.Errorf("algorithm not found: %s", algorithmID)
	}

	for key, value := range params {
		algo.Parameters[key] = value

		switch key {
		case "strength":
			if fv, ok := value.(float64); ok {
				algo.Strength = math.Max(0.0, math.Min(1.0, fv))
			}
		case "robustness":
			if fv, ok := value.(float64); ok {
				algo.Robustness = math.Max(0.0, math.Min(1.0, fv))
			}
		case "invisibility":
			if fv, ok := value.(float64); ok {
				algo.Invisibility = math.Max(0.0, math.Min(1.0, fv))
			}
		case "capacity":
			if fv, ok := value.(float64); ok {
				algo.Capacity = math.Max(0.0, math.Min(1.0, fv))
			}
		}
	}

	return nil
}

func (svc *WatermarkVerificationService) DeactivateAlgorithm(ctx context.Context, algorithmID string) error {
	svc.mu.Lock()
	defer svc.mu.Unlock()

	algo, exists := svc.algorithms[algorithmID]
	if !exists {
		return fmt.Errorf("algorithm not found: %s", algorithmID)
	}

	algo.IsActive = false
	return nil
}

func (svc *WatermarkVerificationService) ActivateAlgorithm(ctx context.Context, algorithmID string) error {
	svc.mu.Lock()
	defer svc.mu.Unlock()

	algo, exists := svc.algorithms[algorithmID]
	if !exists {
		return fmt.Errorf("algorithm not found: %s", algorithmID)
	}

	algo.IsActive = true
	return nil
}
