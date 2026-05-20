package service

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"
)

type DeepfakeDetectionV3 struct {
	mu                     sync.RWMutex
	initialized            bool
	enhancedFaceDetector   *V3FaceDetector
	aiContentRecognizer    *AIGeneratedContentRecognizer
	watermarkVerifier      *SyntheticMediaWatermarkVerifier
	tamperingDetector      *AdvancedTamperingDetector
	detectionHistory       map[string]*V3DetectionResult
}

type V3FaceDetector struct {
	mu          sync.RWMutex
	initialized bool
	modelVersion string
	detectionThreshold float64
	features    *FaceDetectionFeatures
}

type FaceDetectionFeatures struct {
	BlinkDetection     bool
	ExpressionAnalysis bool
	GazeTracking      bool
	LipSyncAnalysis    bool
}

type V3DetectionResult struct {
	ID            string                    `json:"id"`
	Timestamp     time.Time                `json:"timestamp"`
	ContentType   string                    `json:"content_type"`
	OverallScore  float64                   `json:"overall_score"`
	RiskLevel     string                    `json:"risk_level"`
	SubResults    []*V3SubDetectionResult   `json:"sub_results"`
	ProcessingTime time.Duration            `json:"processing_time"`
}

type V3SubDetectionResult struct {
	Component     string                 `json:"component"`
	Score         float64                `json:"score"`
	Confidence     float64                `json:"confidence"`
	Evidence      []V3DetectionEvidence  `json:"evidence"`
	Recommendations []string             `json:"recommendations"`
}

type V3DetectionEvidence struct {
	Type        string                  `json:"type"`
	Description string                  `json:"description"`
	Severity    float64                 `json:"severity"`
	Location    string                  `json:"location"`
	Metadata    map[string]interface{}  `json:"metadata,omitempty"`
}

type AIGeneratedContentRecognizer struct {
	mu          sync.RWMutex
	initialized bool
	modelType   string
	detectors   []*ContentDetector
}

type ContentDetector struct {
	DetectorID   string
	DetectorType string
	Sensitivity  float64
	Thresholds   DetectionThresholds
}

type DetectionThresholds struct {
	HighConfidence   float64
	MediumConfidence float64
	LowConfidence    float64
}

type AIAnalysisResult struct {
	IsAIGenerated     bool                    `json:"is_ai_generated"`
	Confidence         float64                 `json:"confidence"`
	GenerationType     string                  `json:"generation_type"`
	ModelSource        string                  `json:"model_source,omitempty"`
	Artifacts          []AIArtifact            `json:"artifacts"`
	PatternIndicators  []PatternIndicator      `json:"pattern_indicators"`
	ProcessingTime     time.Duration           `json:"processing_time"`
}

type AIArtifact struct {
	Type        string                 `json:"type"`
	Description string                 `json:"description"`
	Severity    float64                `json:"severity"`
	Location    string                 `json:"location"`
}

type PatternIndicator struct {
	PatternType string  `json:"pattern_type"`
	Score       float64 `json:"score"`
	Description string  `json:"description"`
}

type SyntheticMediaWatermarkVerifier struct {
	mu          sync.RWMutex
	initialized bool
	algorithms  []WatermarkAlgorithm
	verified    map[string]*WatermarkVerification
}

type WatermarkAlgorithm struct {
	AlgorithmID   string
	AlgorithmType  string
	Strength       float64
	Robustness     float64
}

type WatermarkVerification struct {
	WatermarkID    string                  `json:"watermark_id"`
	IsPresent       bool                    `json:"is_present"`
	IsAuthentic     bool                    `json:"is_authentic"`
	VerificationScore float64               `json:"verification_score"`
	AlgorithmUsed   string                  `json:"algorithm_used"`
	Details         []WatermarkDetail       `json:"details"`
	Timestamp       time.Time               `json:"timestamp"`
}

type WatermarkDetail struct {
	Property      string  `json:"property"`
	Expected      string  `json:"expected"`
	Actual        string  `json:"actual"`
	Match         bool    `json:"match"`
}

type AdvancedTamperingDetector struct {
	mu          sync.RWMutex
	initialized bool
	methods     []TamperingDetectionMethod
}

type TamperingDetectionMethod struct {
	MethodID     string
	MethodName   string
	Accuracy     float64
	IsApplicable bool
}

type TamperingAnalysisResult struct {
	IsTampered         bool                     `json:"is_tampered"`
	Confidence         float64                  `json:"confidence"`
	ManipulationTypes  []string                 `json:"manipulation_types"`
	RegionsOfInterest  []TamperedRegionV3       `json:"regions_of_interest"`
	Evidence           []TamperingEvidenceV3    `json:"evidence"`
	IntegrityScore     float64                  `json:"integrity_score"`
	ProcessingTime     time.Duration            `json:"processing_time"`
}

type TamperedRegionV3 struct {
	X           int        `json:"x"`
	Y           int        `json:"y"`
	Width       int        `json:"width"`
	Height      int        `json:"height"`
	Mask        [][]float64 `json:"mask,omitempty"`
	Confidence  float64    `json:"confidence"`
}

type TamperingEvidenceV3 struct {
	EvidenceID    string                  `json:"evidence_id"`
	Type          string                  `json:"type"`
	Description   string                  `json:"description"`
	Severity      float64                 `json:"severity"`
	MethodUsed    string                  `json:"method_used"`
}

func NewDeepfakeDetectionV3() *DeepfakeDetectionV3 {
	return &DeepfakeDetectionV3{
		enhancedFaceDetector: NewV3FaceDetector(),
		aiContentRecognizer:  NewAIGeneratedContentRecognizer(),
		watermarkVerifier:    NewSyntheticMediaWatermarkVerifier(),
		tamperingDetector:    NewAdvancedTamperingDetector(),
		detectionHistory:     make(map[string]*V3DetectionResult),
	}
}

func (dd *DeepfakeDetectionV3) Initialize(ctx context.Context) error {
	dd.mu.Lock()
	defer dd.mu.Unlock()

	if dd.initialized {
		return nil
	}

	if err := dd.enhancedFaceDetector.Initialize(ctx); err != nil {
		return err
	}

	if err := dd.aiContentRecognizer.Initialize(ctx); err != nil {
		return err
	}

	if err := dd.watermarkVerifier.Initialize(ctx); err != nil {
		return err
	}

	if err := dd.tamperingDetector.Initialize(ctx); err != nil {
		return err
	}

	dd.initialized = true
	return nil
}

func NewV3FaceDetector() *V3FaceDetector {
	return &V3FaceDetector{
		modelVersion: "v3.0",
		detectionThreshold: 0.6,
		features: &FaceDetectionFeatures{
			BlinkDetection:      true,
			ExpressionAnalysis:  true,
			GazeTracking:        true,
			LipSyncAnalysis:     true,
		},
	}
}

func (fd *V3FaceDetector) Initialize(ctx context.Context) error {
	fd.mu.Lock()
	defer fd.mu.Unlock()
	fd.initialized = true
	return nil
}

func (fd *V3FaceDetector) AnalyzeFace(ctx context.Context, data []byte) (*FaceAnalysisResult, error) {
	fd.mu.RLock()
	defer fd.mu.RUnlock()

	if !fd.initialized {
		return nil, fmt.Errorf("V3 face detector not initialized")
	}

	result := &FaceAnalysisResult{
		IsReal:         true,
		Confidence:     0.0,
		BlinkScore:     0.85,
		ExpressionScore: 0.90,
		GazeScore:      0.88,
		LipSyncScore:   0.82,
		Artifacts:      make([]FaceArtifact, 0),
	}

	blinkResult := fd.analyzeBlinkPattern(data)
	result.BlinkScore = blinkResult.Score
	if blinkResult.HasAnomaly {
		result.Artifacts = append(result.Artifacts, FaceArtifact{
			Type:        "blink_anomaly",
			Description: "Blink pattern is irregular",
			Severity:    blinkResult.Severity,
		})
	}

	expressionResult := fd.analyzeExpression(data)
	result.ExpressionScore = expressionResult.Score

	gazeResult := fd.analyzeGaze(data)
	result.GazeScore = gazeResult.Score

	lipSyncResult := fd.analyzeLipSync(data)
	result.LipSyncScore = lipSyncResult.Score

	result.Confidence = (result.BlinkScore + result.ExpressionScore + result.GazeScore + result.LipSyncScore) / 4.0
	result.IsReal = result.Confidence >= fd.detectionThreshold

	return result, nil
}

type BlinkAnalysisResult struct {
	Score      float64
	HasAnomaly bool
	Severity   float64
}

func (fd *V3FaceDetector) analyzeBlinkPattern(data []byte) BlinkAnalysisResult {
	return BlinkAnalysisResult{
		Score:      0.7 + math.Mod(float64(len(data)), 0.3)*0.2,
		HasAnomaly: math.Mod(float64(len(data)), 10) < 3,
		Severity:   0.2 + math.Mod(float64(len(data)), 0.2)*0.3,
	}
}

type ExpressionAnalysisResult struct {
	Score float64
}

func (fd *V3FaceDetector) analyzeExpression(data []byte) ExpressionAnalysisResult {
	return ExpressionAnalysisResult{
		Score: 0.75 + math.Mod(float64(len(data)), 0.25)*0.15,
	}
}

type GazeAnalysisResult struct {
	Score float64
}

func (fd *V3FaceDetector) analyzeGaze(data []byte) GazeAnalysisResult {
	return GazeAnalysisResult{
		Score: 0.8 + math.Mod(float64(len(data)), 0.2)*0.1,
	}
}

type LipSyncAnalysisResult struct {
	Score float64
}

func (fd *V3FaceDetector) analyzeLipSync(data []byte) LipSyncAnalysisResult {
	return LipSyncAnalysisResult{
		Score: 0.78 + math.Mod(float64(len(data)), 0.22)*0.12,
	}
}

type FaceAnalysisResult struct {
	IsReal          bool
	Confidence      float64
	BlinkScore      float64
	ExpressionScore float64
	GazeScore       float64
	LipSyncScore    float64
	Artifacts       []FaceArtifact
}

type FaceArtifact struct {
	Type        string
	Description string
	Severity    float64
}

func NewAIGeneratedContentRecognizer() *AIGeneratedContentRecognizer {
	detectors := []*ContentDetector{
		{
			DetectorID:    "texture_detector",
			DetectorType:  "texture_analysis",
			Sensitivity:   0.8,
			Thresholds: DetectionThresholds{
				HighConfidence:   0.85,
				MediumConfidence: 0.70,
				LowConfidence:    0.50,
			},
		},
		{
			DetectorID:    "statistical_detector",
			DetectorType:  "statistical_analysis",
			Sensitivity:   0.75,
			Thresholds: DetectionThresholds{
				HighConfidence:   0.85,
				MediumConfidence: 0.70,
				LowConfidence:    0.50,
			},
		},
		{
			DetectorID:    "semantic_detector",
			DetectorType:  "semantic_consistency",
			Sensitivity:   0.85,
			Thresholds: DetectionThresholds{
				HighConfidence:   0.85,
				MediumConfidence: 0.70,
				LowConfidence:    0.50,
			},
		},
	}

	return &AIGeneratedContentRecognizer{
		modelType: "multi_detector_v3",
		detectors: detectors,
	}
}

func (r *AIGeneratedContentRecognizer) Initialize(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.initialized = true
	return nil
}

func (r *AIGeneratedContentRecognizer) RecognizeAI(ctx context.Context, content []byte, contentType string) (*AIAnalysisResult, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if !r.initialized {
		return nil, fmt.Errorf("AI content recognizer not initialized")
	}

	startTime := time.Now()

	result := &AIAnalysisResult{
		GenerationType:    "unknown",
		Artifacts:        make([]AIArtifact, 0),
		PatternIndicators: make([]PatternIndicator, 0),
	}

	for _, detector := range r.detectors {
		analysis := r.runDetector(detector, content, contentType)

		if analysis.IsAIGenerated {
			result.Artifacts = append(result.Artifacts, AIArtifact{
				Type:        detector.DetectorType,
				Description: analysis.Description,
				Severity:    analysis.Severity,
				Location:    "general",
			})
		}

		result.PatternIndicators = append(result.PatternIndicators, PatternIndicator{
			PatternType: detector.DetectorType,
			Score:       analysis.Score,
			Description: analysis.Description,
		})
	}

	result.Confidence = r.calculateOverallConfidence(result)
	result.IsAIGenerated = result.Confidence >= 0.5

	if result.IsAIGenerated {
		result.GenerationType = r.identifyGenerationType(result.PatternIndicators)
	}

	result.ProcessingTime = time.Since(startTime)

	return result, nil
}

type DetectorAnalysis struct {
	IsAIGenerated bool
	Score         float64
	Severity      float64
	Description   string
}

func (r *AIGeneratedContentRecognizer) runDetector(detector *ContentDetector, content []byte, contentType string) DetectorAnalysis {
	analysis := DetectorAnalysis{
		IsAIGenerated: false,
		Score:         0.0,
		Severity:      0.0,
		Description:   "No anomalies detected",
	}

	baseScore := 0.5 + math.Mod(float64(len(content)), 0.3)*0.2
	analysis.Score = baseScore

	if analysis.Score > 0.7 {
		analysis.IsAIGenerated = true
		analysis.Severity = (analysis.Score - 0.7) * 3.0
		analysis.Description = fmt.Sprintf("Potential AI-generated content detected by %s", detector.DetectorType)
	}

	return analysis
}

func (r *AIGeneratedContentRecognizer) calculateOverallConfidence(result *AIAnalysisResult) float64 {
	if len(result.PatternIndicators) == 0 {
		return 0.0
	}

	totalScore := 0.0
	for _, indicator := range result.PatternIndicators {
		totalScore += indicator.Score
	}

	return totalScore / float64(len(result.PatternIndicators))
}

func (r *AIGeneratedContentRecognizer) identifyGenerationType(indicators []PatternIndicator) string {
	if len(indicators) == 0 {
		return "unknown"
	}

	maxScore := 0.0
	dominantType := "general"

	for _, indicator := range indicators {
		if indicator.Score > maxScore {
			maxScore = indicator.Score
			dominantType = indicator.PatternType
		}
	}

	return dominantType
}

func NewSyntheticMediaWatermarkVerifier() *SyntheticMediaWatermarkVerifier {
	algorithms := []WatermarkAlgorithm{
		{
			AlgorithmID:   "lsb_watermark",
			AlgorithmType:  "least_significant_bit",
			Strength:       0.8,
			Robustness:     0.7,
		},
		{
			AlgorithmID:   "dct_watermark",
			AlgorithmType:  "discrete_cosine_transform",
			Strength:       0.9,
			Robustness:     0.85,
		},
		{
			AlgorithmID:   "dwt_watermark",
			AlgorithmType:  "discrete_wavelet_transform",
			Strength:       0.85,
			Robustness:     0.9,
		},
	}

	return &SyntheticMediaWatermarkVerifier{
		algorithms: algorithms,
		verified:  make(map[string]*WatermarkVerification),
	}
}

func (wv *SyntheticMediaWatermarkVerifier) Initialize(ctx context.Context) error {
	wv.mu.Lock()
	defer wv.mu.Unlock()
	wv.initialized = true
	return nil
}

func (wv *SyntheticMediaWatermarkVerifier) VerifyWatermark(ctx context.Context, mediaData []byte, watermarkID string) (*WatermarkVerification, error) {
	wv.mu.RLock()
	defer wv.mu.RUnlock()

	if !wv.initialized {
		return nil, fmt.Errorf("watermark verifier not initialized")
	}

	verification := &WatermarkVerification{
		WatermarkID:      watermarkID,
		Details:          make([]WatermarkDetail, 0),
		Timestamp:        time.Now(),
	}

	var bestScore float64
	var bestAlgorithm string

	for _, algorithm := range wv.algorithms {
		score := wv.verifyWithAlgorithm(algorithm, mediaData, watermarkID)

		if score > bestScore {
			bestScore = score
			bestAlgorithm = algorithm.AlgorithmID
		}

		verification.Details = append(verification.Details, WatermarkDetail{
			Property: "algorithm_" + algorithm.AlgorithmID,
			Expected: fmt.Sprintf("score > %.2f", algorithm.Strength),
			Actual:   fmt.Sprintf("score = %.2f", score),
			Match:    score >= algorithm.Strength,
		})
	}

	verification.VerificationScore = bestScore
	verification.AlgorithmUsed = bestAlgorithm
	verification.IsPresent = bestScore >= 0.6
	verification.IsAuthentic = bestScore >= 0.8

	wv.verified[watermarkID] = verification

	return verification, nil
}

func (wv *SyntheticMediaWatermarkVerifier) verifyWithAlgorithm(algorithm WatermarkAlgorithm, mediaData []byte, watermarkID string) float64 {
	baseScore := algorithm.Strength * algorithm.Robustness

	dataInfluence := math.Mod(float64(len(mediaData)), 0.3)*0.2
	score := (baseScore + dataInfluence) / 2.0

	return math.Min(1.0, math.Max(0.0, score))
}

func (wv *SyntheticMediaWatermarkVerifier) GetVerification(watermarkID string) (*WatermarkVerification, bool) {
	wv.mu.RLock()
	defer wv.mu.RUnlock()

	verification, exists := wv.verified[watermarkID]
	return verification, exists
}

func NewAdvancedTamperingDetector() *AdvancedTamperingDetector {
	methods := []TamperingDetectionMethod{
		{
			MethodID:     "ela",
			MethodName:   "Error Level Analysis",
			Accuracy:     0.85,
			IsApplicable: true,
		},
		{
			MethodID:     "cfa",
			MethodName:   "Color Filter Array Analysis",
			Accuracy:     0.80,
			IsApplicable: true,
		},
		{
			MethodID:     "clone",
			MethodName:   "Copy-Move Detection",
			Accuracy:     0.90,
			IsApplicable: true,
		},
		{
			MethodID:     "metadata",
			MethodName:   "Metadata Consistency",
			Accuracy:     0.75,
			IsApplicable: true,
		},
	}

	return &AdvancedTamperingDetector{
		methods: methods,
	}
}

func (td *AdvancedTamperingDetector) Initialize(ctx context.Context) error {
	td.mu.Lock()
	defer td.mu.Unlock()
	td.initialized = true
	return nil
}

func (td *AdvancedTamperingDetector) DetectTampering(ctx context.Context, imageData []byte) (*TamperingAnalysisResult, error) {
	td.mu.RLock()
	defer td.mu.RUnlock()

	if !td.initialized {
		return nil, fmt.Errorf("tampering detector not initialized")
	}

	startTime := time.Now()

	result := &TamperingAnalysisResult{
		ManipulationTypes: make([]string, 0),
		Evidence:          make([]TamperingEvidenceV3, 0),
	}

	evidenceScores := make(map[string]float64)

	for _, method := range td.methods {
		if !method.IsApplicable {
			continue
		}

		detection := td.applyDetectionMethod(method, imageData)

		if detection.HasEvidence {
			evidenceScores[method.MethodID] = detection.Confidence

			result.Evidence = append(result.Evidence, TamperingEvidenceV3{
				EvidenceID:  fmt.Sprintf("evidence_%s_%d", method.MethodID, time.Now().UnixNano()),
				Type:        method.MethodName,
				Description: detection.Description,
				Severity:    detection.Confidence,
				MethodUsed:  method.MethodID,
			})

			if detection.Confidence >= 0.7 {
				result.ManipulationTypes = append(result.ManipulationTypes, method.MethodID)
			}
		}
	}

	result.Confidence = td.calculateConfidence(evidenceScores)
	result.IsTampered = result.Confidence >= 0.5

	if result.IsTampered {
		result.IntegrityScore = 1.0 - result.Confidence
	} else {
		result.IntegrityScore = result.Confidence
	}

	if len(evidenceScores) > 0 {
		avgConfidence := 0.0
		for _, score := range evidenceScores {
			avgConfidence += score
		}
		avgConfidence /= float64(len(evidenceScores))

		result.RegionsOfInterest = []TamperedRegionV3{
			{
				X:          0,
				Y:          0,
				Width:      100,
				Height:     100,
				Confidence: avgConfidence,
			},
		}
	}

	result.ProcessingTime = time.Since(startTime)

	return result, nil
}

type MethodDetection struct {
	HasEvidence  bool
	Confidence   float64
	Description  string
}

func (td *AdvancedTamperingDetector) applyDetectionMethod(method TamperingDetectionMethod, imageData []byte) MethodDetection {
	detection := MethodDetection{
		HasEvidence:  false,
		Confidence:   0.0,
		Description:  "No tampering detected",
	}

	baseProbability := 0.3 + math.Mod(float64(len(imageData)), 0.4)*0.2

	if baseProbability > method.Accuracy*0.8 {
		detection.HasEvidence = true
		detection.Confidence = baseProbability * method.Accuracy
		detection.Description = fmt.Sprintf("Potential %s detected", method.MethodName)
	}

	return detection
}

func (td *AdvancedTamperingDetector) calculateConfidence(scores map[string]float64) float64 {
	if len(scores) == 0 {
		return 0.0
	}

	totalScore := 0.0
	for _, score := range scores {
		totalScore += score
	}

	return totalScore / float64(len(scores))
}

func (dd *DeepfakeDetectionV3) ComprehensiveAnalysis(ctx context.Context, contentType string, data []byte) (*V3DetectionResult, error) {
	dd.mu.Lock()
	defer dd.mu.Unlock()

	if !dd.initialized {
		return nil, fmt.Errorf("DeepfakeDetectionV3 not initialized")
	}

	startTime := time.Now()

	result := &V3DetectionResult{
		ID:          fmt.Sprintf("v3_detect_%d", time.Now().UnixNano()),
		Timestamp:   time.Now(),
		ContentType: contentType,
		SubResults:  make([]*V3SubDetectionResult, 0),
	}

	switch contentType {
	case "image":
		faceResult, _ := dd.enhancedFaceDetector.AnalyzeFace(ctx, data)
		if faceResult != nil {
			result.SubResults = append(result.SubResults, &V3SubDetectionResult{
				Component:     "face_detection",
				Score:         faceResult.Confidence,
				Confidence:    faceResult.Confidence,
				Recommendations: dd.generateFaceRecommendations(faceResult),
			})
		}

		aiResult, _ := dd.aiContentRecognizer.RecognizeAI(ctx, data, contentType)
		if aiResult != nil {
			result.SubResults = append(result.SubResults, &V3SubDetectionResult{
				Component:     "ai_recognition",
				Score:         aiResult.Confidence,
				Confidence:    aiResult.Confidence,
				Evidence:      dd.convertAIArtifacts(aiResult.Artifacts),
			})
		}

		tamperResult, _ := dd.tamperingDetector.DetectTampering(ctx, data)
		if tamperResult != nil {
			result.SubResults = append(result.SubResults, &V3SubDetectionResult{
				Component:     "tampering_detection",
				Score:         tamperResult.Confidence,
				Confidence:    tamperResult.Confidence,
				Evidence:      dd.convertTamperingEvidence(tamperResult.Evidence),
			})
		}

	case "video":
		faceResult, _ := dd.enhancedFaceDetector.AnalyzeFace(ctx, data)
		if faceResult != nil {
			result.SubResults = append(result.SubResults, &V3SubDetectionResult{
				Component:     "face_detection",
				Score:         faceResult.Confidence,
				Confidence:    faceResult.Confidence,
			})
		}

		aiResult, _ := dd.aiContentRecognizer.RecognizeAI(ctx, data, contentType)
		if aiResult != nil {
			result.SubResults = append(result.SubResults, &V3SubDetectionResult{
				Component:     "ai_recognition",
				Score:         aiResult.Confidence,
				Confidence:    aiResult.Confidence,
			})
		}

	case "audio":
		aiResult, _ := dd.aiContentRecognizer.RecognizeAI(ctx, data, contentType)
		if aiResult != nil {
			result.SubResults = append(result.SubResults, &V3SubDetectionResult{
				Component:     "ai_recognition",
				Score:         aiResult.Confidence,
				Confidence:    aiResult.Confidence,
			})
		}
	}

	result.OverallScore = dd.calculateOverallScore(result.SubResults)
	result.RiskLevel = dd.determineRiskLevel(result.OverallScore)
	result.ProcessingTime = time.Since(startTime)

	dd.detectionHistory[result.ID] = result

	return result, nil
}

func (dd *DeepfakeDetectionV3) calculateOverallScore(subResults []*V3SubDetectionResult) float64 {
	if len(subResults) == 0 {
		return 0.0
	}

	totalScore := 0.0
	for _, sr := range subResults {
		totalScore += sr.Score
	}

	return totalScore / float64(len(subResults))
}

func (dd *DeepfakeDetectionV3) determineRiskLevel(score float64) string {
	switch {
	case score >= 85:
		return "critical"
	case score >= 70:
		return "high"
	case score >= 50:
		return "medium"
	case score >= 30:
		return "low"
	default:
		return "minimal"
	}
}

func (dd *DeepfakeDetectionV3) generateFaceRecommendations(result *FaceAnalysisResult) []string {
	recommendations := make([]string, 0)

	if !result.IsReal {
		recommendations = append(recommendations, "建议进行人工复核")
		recommendations = append(recommendations, "检查是否为深度伪造内容")
	}

	if result.BlinkScore < 0.7 {
		recommendations = append(recommendations, "检测到异常眨眼模式")
	}

	return recommendations
}

func (dd *DeepfakeDetectionV3) convertAIArtifacts(artifacts []AIArtifact) []V3DetectionEvidence {
	evidence := make([]V3DetectionEvidence, 0, len(artifacts))
	for _, a := range artifacts {
		evidence = append(evidence, V3DetectionEvidence{
			Type:        a.Type,
			Description: a.Description,
			Severity:    a.Severity,
		})
	}
	return evidence
}

func (dd *DeepfakeDetectionV3) convertTamperingEvidence(evidence []TamperingEvidenceV3) []V3DetectionEvidence {
	result := make([]V3DetectionEvidence, 0, len(evidence))
	for _, e := range evidence {
		result = append(result, V3DetectionEvidence{
			Type:        e.Type,
			Description: e.Description,
			Severity:    e.Severity,
		})
	}
	return result
}

func (dd *DeepfakeDetectionV3) GetDetectionHistory(ctx context.Context, resultID string) (*V3DetectionResult, error) {
	dd.mu.RLock()
	defer dd.mu.RUnlock()

	result, exists := dd.detectionHistory[resultID]
	if !exists {
		return nil, fmt.Errorf("detection result not found")
	}

	return result, nil
}

func (dd *DeepfakeDetectionV3) VerifyWatermark(ctx context.Context, watermarkID string, mediaData []byte) (*WatermarkVerification, error) {
	return dd.watermarkVerifier.VerifyWatermark(ctx, mediaData, watermarkID)
}
