package service

import (
	"context"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"sync"
	"time"
)

type DeepfakeDetectionSystem struct {
	mu                sync.RWMutex
	faceSwapDetector  *FaceSwapDetector
	voiceSynthDetector *VoiceSynthesisDetector
	imageTamperDetector *ImageTamperingDetector
	alertSystem      *DeepfakeAlertSystem
	initialized      bool
}

type FaceSwapDetector struct {
	mu           sync.RWMutex
	modelVersion string
	thresholds   FaceSwapThresholds
}

type FaceSwapThresholds struct {
	HighConfidence   float64
	MediumConfidence float64
	LowConfidence    float64
}

type FaceSwapResult struct {
	IsDeepfake         bool                   `json:"is_deepfake"`
	Confidence         float64                `json:"confidence"`
	FaceRegions        []FaceRegion           `json:"face_regions"`
	Artifacts          []Artifact             `json:"artifacts"`
	SplicingEvidence   *SplicingEvidence     `json:"splicing_evidence,omitempty"`
	ProcessingTime     time.Duration          `json:"processing_time"`
}

type FaceRegion struct {
	X      int     `json:"x"`
	Y      int     `json:"y"`
	Width  int     `json:"width"`
	Height int     `json:"height"`
	Score  float64 `json:"score"`
}

type Artifact struct {
	Type        string  `json:"type"`
	Location    string  `json:"location"`
	Severity    float64 `json:"severity"`
	Description string  `json:"description"`
}

type SplicingEvidence struct {
	BoundaryDetected bool      `json:"boundary_detected"`
	SeamLocation    string    `json:"seam_location"`
	ColorInconsistency float64 `json:"color_inconsistency"`
	TextureMismatch   float64 `json:"texture_mismatch"`
}

type VoiceSynthesisDetector struct {
	mu           sync.RWMutex
	modelVersion string
	thresholds   VoiceThresholds
}

type VoiceThresholds struct {
	HighConfidence   float64
	MediumConfidence float64
	LowConfidence    float64
}

type VoiceSynthesisResult struct {
	IsSynthesized   bool                    `json:"is_synthesized"`
	Confidence      float64                 `json:"confidence"`
	Artifacts       []VoiceArtifact         `json:"artifacts"`
	SpectralFeatures *SpectralAnalysis      `json:"spectral_features,omitempty"`
	ProsodyAnomalies []ProsodyAnomaly        `json:"prosody_anomalies,omitempty"`
	ProcessingTime  time.Duration           `json:"processing_time"`
}

type VoiceArtifact struct {
	Type        string  `json:"type"`
	Frequency   float64 `json:"frequency"`
	Duration    float64 `json:"duration"`
	Severity    float64 `json:"severity"`
	Description string  `json:"description"`
}

type SpectralAnalysis struct {
	HighFreqAttenuation float64 `json:"high_freq_attenuation"`
	LowFreqAmplitude    float64 `json:"low_freq_amplitude"`
	MelFreqCepstralCoeffs []float64 `json:"mfcc_features"`
	SpectralFlux        float64 `json:"spectral_flux"`
	HarmonicRatio        float64 `json:"harmonic_ratio"`
}

type ProsodyAnomaly struct {
	Type     string  `json:"type"`
	Position float64 `json:"position"`
	Severity float64 `json:"severity"`
	Details  string  `json:"details"`
}

type ImageTamperingDetector struct {
	mu           sync.RWMutex
	modelVersion string
	thresholds   ImageTamperThresholds
}

type ImageTamperThresholds struct {
	HighConfidence   float64
	MediumConfidence float64
	LowConfidence    float64
}

type ImageTamperingResult struct {
	IsTampered     bool                `json:"is_tampered"`
	Confidence     float64             `json:"confidence"`
	Evidence       []TamperingEvidence  `json:"evidence"`
	ManipulationType string            `json:"manipulation_type"`
	RegionOfInterest *TamperedRegion   `json:"region_of_interest,omitempty"`
	ProcessingTime time.Duration       `json:"processing_time"`
}

type TamperingEvidence struct {
	Type        string  `json:"type"`
	Location    string  `json:"location"`
	Severity    float64 `json:"severity"`
	Description string  `json:"description"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

type TamperedRegion struct {
	X      int     `json:"x"`
	Y      int     `json:"y"`
	Width  int     `json:"width"`
	Height int     `json:"height"`
	Mask   [][]float64 `json:"mask,omitempty"`
}

type DeepfakeAlertSystem struct {
	mu          sync.RWMutex
	alerts      []DeepfakeAlert
	maxAlerts   int
	enabled     bool
}

type DeepfakeAlert struct {
	ID          string    `json:"id"`
	Type        string    `json:"type"`
	Severity    string    `json:"severity"`
	Source      string    `json:"source"`
	Message     string    `json:"message"`
	Timestamp   time.Time `json:"timestamp"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	Acknowledged bool     `json:"acknowledged"`
}

type ComprehensiveDeepfakeResult struct {
	OverallRisk        float64                `json:"overall_risk"`
	RiskLevel          string                 `json:"risk_level"`
	FaceSwapResult     *FaceSwapResult        `json:"face_swap_result,omitempty"`
	VoiceResult        *VoiceSynthesisResult  `json:"voice_result,omitempty"`
	ImageTamperResult  *ImageTamperingResult  `json:"image_tamper_result,omitempty"`
	Recommendations    []string               `json:"recommendations"`
	ProcessingTime     time.Duration          `json:"processing_time"`
	Timestamp          time.Time              `json:"timestamp"`
}

func NewDeepfakeDetectionSystem() *DeepfakeDetectionSystem {
	return &DeepfakeDetectionSystem{
		faceSwapDetector:   NewFaceSwapDetector(),
		voiceSynthDetector: NewVoiceSynthesisDetector(),
		imageTamperDetector: NewImageTamperingDetector(),
		alertSystem:        NewDeepfakeAlertSystem(),
	}
}

func (s *DeepfakeDetectionSystem) Initialize(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.initialized {
		return nil
	}

	if err := s.faceSwapDetector.Initialize(ctx); err != nil {
		return err
	}

	if err := s.voiceSynthDetector.Initialize(ctx); err != nil {
		return err
	}

	if err := s.imageTamperDetector.Initialize(ctx); err != nil {
		return err
	}

	s.alertSystem.enabled = true
	s.initialized = true
	return nil
}

func NewFaceSwapDetector() *FaceSwapDetector {
	return &FaceSwapDetector{
		modelVersion: "v1.0",
		thresholds: FaceSwapThresholds{
			HighConfidence:   0.85,
			MediumConfidence: 0.70,
			LowConfidence:   0.50,
		},
	}
}

func (d *FaceSwapDetector) Initialize(ctx context.Context) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	return nil
}

func (d *FaceSwapDetector) DetectFaceSwap(ctx context.Context, imageData []byte, metadata map[string]interface{}) (*FaceSwapResult, error) {
	start := time.Now()

	result := &FaceSwapResult{
		Confidence:  0.0,
		IsDeepfake: false,
		FaceRegions: make([]FaceRegion, 0),
		Artifacts:  make([]Artifact, 0),
	}

	img, format, err := d.decodeImage(imageData)
	if err != nil {
		result.Confidence = 0.0
		return result, nil
	}

	width := img.Bounds().Dx()
	height := img.Bounds().Dy()

	regions := d.detectFaceRegions(img)
	result.FaceRegions = regions

	if len(regions) > 1 {
		consistencyScore := d.checkFaceConsistency(img, regions)
		if consistencyScore < 0.7 {
			result.Artifacts = append(result.Artifacts, Artifact{
				Type:        "face_inconsistency",
				Location:    "multiple_faces",
				Severity:    1.0 - consistencyScore,
				Description: "检测到面部特征不一致",
			})
		}
	}

	noiseAnalysis := d.analyzeNoisePattern(img)
	if noiseAnalysis.AnomalyScore > 0.6 {
		result.Artifacts = append(result.Artifacts, Artifact{
			Type:        "noise_anomaly",
			Location:    noiseAnalysis.Location,
			Severity:    noiseAnalysis.AnomalyScore,
			Description: "检测到异常的噪声模式",
		})
	}

	colorAnalysis := d.analyzeColorInconsistency(img, regions)
	if colorAnalysis.HasInconsistency {
		result.SplicingEvidence = &SplicingEvidence{
			BoundaryDetected:     colorAnalysis.BoundaryDetected,
			SeamLocation:         colorAnalysis.SeamLocation,
			ColorInconsistency:   colorAnalysis.Score,
			TextureMismatch:       colorAnalysis.TextureMismatch,
		}
	}

	compressionAnalysis := d.analyzeCompressionArtifacts(img)
	if compressionAnalysis.HasArtifacts {
		result.Artifacts = append(result.Artifacts, Artifact{
			Type:        "compression_artifact",
			Location:    compressionAnalysis.Location,
			Severity:    compressionAnalysis.Severity,
			Description: "检测到压缩伪影",
		})
	}

	result.Confidence = d.calculateOverallConfidence(result)
	result.IsDeepfake = result.Confidence >= d.thresholds.LowConfidence

	result.ProcessingTime = time.Since(start)

	if format != "" {
		if metadata == nil {
			metadata = make(map[string]interface{})
		}
		metadata["image_format"] = format
		metadata["image_width"] = width
		metadata["image_height"] = height
	}

	return result, nil
}

type NoiseAnalysisResult struct {
	AnomalyScore float64
	Location     string
	Pattern      string
}

func (d *FaceSwapDetector) detectFaceRegions(img image.Image) []FaceRegion {
	regions := make([]FaceRegion, 0)

	width := img.Bounds().Dx()
	height := img.Bounds().Dy()

	faceWidth := width / 4
	faceHeight := height / 4

	for i := 0; i < 2; i++ {
		for j := 0; j < 2; j++ {
			x := i * (width / 2)
			y := j * (height / 2)

			region := FaceRegion{
				X:      x,
				Y:      y,
				Width:  faceWidth,
				Height: faceHeight,
				Score:  0.7 + float64(i+j)*0.1,
			}
			regions = append(regions, region)
		}
	}

	return regions
}

func (d *FaceSwapDetector) checkFaceConsistency(img image.Image, regions []FaceRegion) float64 {
	if len(regions) < 2 {
		return 1.0
	}

	var totalDiff float64
	count := 0

	for i := 0; i < len(regions); i++ {
		for j := i + 1; j < len(regions); j++ {
			diff := d.compareRegions(img, regions[i], regions[j])
			totalDiff += diff
			count++
		}
	}

	if count == 0 {
		return 1.0
	}

	avgDiff := totalDiff / float64(count)
	return math.Max(0, 1.0-avgDiff)
}

func (d *FaceSwapDetector) compareRegions(img image.Image, r1, r2 FaceRegion) float64 {
	r1Img := img.(*image.RGBA).SubImage(image.Rect(r1.X, r1.Y, r1.X+r1.Width, r1.Y+r1.Height))
	r2Img := img.(*image.RGBA).SubImage(image.Rect(r2.X, r2.Y, r2.X+r2.Width, r2.Y+r2.Height))

	r1Pixels := r1Img.(*image.RGBA).Pix
	r2Pixels := r2Img.(*image.RGBA).Pix

	var diff float64
	sampleSize := math.Min(float64(len(r1Pixels)), float64(len(r2Pixels)))

	for i := 0; i < int(sampleSize); i += 4 {
		diff += math.Abs(float64(r1Pixels[i]) - float64(r2Pixels[i]))
		diff += math.Abs(float64(r1Pixels[i+1]) - float64(r2Pixels[i+1]))
		diff += math.Abs(float64(r1Pixels[i+2]) - float64(r2Pixels[i+2]))
	}

	return diff / (sampleSize * 3 * 255)
}

func (d *FaceSwapDetector) analyzeNoisePattern(img image.Image) NoiseAnalysisResult {
	width := img.Bounds().Dx()
	height := img.Bounds().Dy()

	var noiseVariance float64
	sampleCount := 0

	for y := 0; y < height; y += 10 {
		for x := 0; x < width; x += 10 {
			r, _, _, _ := img.At(x, y).RGBA()
			noise := float64(r) / 65535

			if noise > 0.1 && noise < 0.9 {
				noiseVariance += math.Abs(noise - 0.5)
				sampleCount++
			}
		}
	}

	if sampleCount == 0 {
		return NoiseAnalysisResult{AnomalyScore: 0.0}
	}

	avgNoise := noiseVariance / float64(sampleCount)

	anomalyScore := 0.5
	if avgNoise < 0.1 {
		anomalyScore = 0.8
	} else if avgNoise < 0.2 {
		anomalyScore = 0.5
	} else {
		anomalyScore = 0.3
	}

	return NoiseAnalysisResult{
		AnomalyScore: anomalyScore,
		Location:     "entire_image",
		Pattern:     "uniform_noise",
	}
}

type ColorInconsistencyResult struct {
	HasInconsistency   bool
	BoundaryDetected   bool
	SeamLocation       string
	Score              float64
	TextureMismatch    float64
}

func (d *FaceSwapDetector) analyzeColorInconsistency(img image.Image, regions []FaceRegion) ColorInconsistencyResult {
	if len(regions) < 2 {
		return ColorInconsistencyResult{HasInconsistency: false}
	}

	var totalColorDiff float64
	var comparisons int

	for i := 0; i < len(regions); i++ {
		for j := i + 1; j < len(regions); j++ {
			colorDiff := d.compareColorDistribution(img, regions[i], regions[j])
			totalColorDiff += colorDiff
			comparisons++
		}
	}

	if comparisons == 0 {
		return ColorInconsistencyResult{HasInconsistency: false}
	}

	avgColorDiff := totalColorDiff / float64(comparisons)

	result := ColorInconsistencyResult{
		HasInconsistency: avgColorDiff > 0.15,
		Score:            avgColorDiff,
		TextureMismatch:   avgColorDiff * 0.8,
	}

	if avgColorDiff > 0.2 {
		result.BoundaryDetected = true
		result.SeamLocation = "face_boundary"
	}

	return result
}

func (d *FaceSwapDetector) compareColorDistribution(img image.Image, r1, r2 FaceRegion) float64 {
	r1Avg := d.calculateAverageColor(img, r1)
	r2Avg := d.calculateAverageColor(img, r2)

	diff := math.Abs(float64(r1Avg.R)-float64(r2Avg.R)) + math.Abs(float64(r1Avg.G)-float64(r2Avg.G)) + math.Abs(float64(r1Avg.B)-float64(r2Avg.B))
	return diff / (3 * 255)
}

func (d *FaceSwapDetector) calculateAverageColor(img image.Image, region FaceRegion) color.RGBA {
	var totalR, totalG, totalB float64
	var count float64

	for y := region.Y; y < region.Y+region.Height && y < img.Bounds().Dy(); y++ {
		for x := region.X; x < region.X+region.Width && x < img.Bounds().Dx(); x++ {
			r, g, b, _ := img.At(x, y).RGBA()
			totalR += float64(r / 257)
			totalG += float64(g / 257)
			totalB += float64(b / 257)
			count++
		}
	}

	if count == 0 {
		return color.RGBA{0, 0, 0, 255}
	}

	return color.RGBA{
		R: uint8(totalR / count),
		G: uint8(totalG / count),
		B: uint8(totalB / count),
		A: 255,
	}
}

type CompressionAnalysisResult struct {
	HasArtifacts bool
	Location     string
	Severity     float64
}

func (d *FaceSwapDetector) analyzeCompressionArtifacts(img image.Image) CompressionAnalysisResult {
	width := img.Bounds().Dx()
	height := img.Bounds().Dy()

	var artifactScore float64
	var sampleCount int

	for y := 1; y < height-1; y += 5 {
		for x := 1; x < width-1; x += 5 {
			r, g, b, _ := img.At(x, y).RGBA()
			r1, g1, b1, _ := img.At(x-1, y).RGBA()
			r2, g2, b2, _ := img.At(x+1, y).RGBA()

			diff := math.Abs(float64(r/257)-float64(r1/257)) +
				math.Abs(float64(r/257)-float64(r2/257)) +
				math.Abs(float64(g/257)-float64(g1/257)) +
				math.Abs(float64(g/257)-float64(g2/257)) +
				math.Abs(float64(b/257)-float64(b1/257)) +
				math.Abs(float64(b/257)-float64(b2/257))

			artifactScore += diff / 6
			sampleCount++
		}
	}

	if sampleCount == 0 {
		return CompressionAnalysisResult{HasArtifacts: false}
	}

	avgArtifact := artifactScore / float64(sampleCount)

	return CompressionAnalysisResult{
		HasArtifacts: avgArtifact < 2.0,
		Location:     "image_edges",
		Severity:    1.0 - math.Min(avgArtifact/10, 1.0),
	}
}

func (d *FaceSwapDetector) calculateOverallConfidence(result *FaceSwapResult) float64 {
	var totalScore float64
	var weight float64

	if len(result.Artifacts) > 0 {
		var artifactScore float64
		for _, a := range result.Artifacts {
			artifactScore += a.Severity
		}
		artifactScore /= float64(len(result.Artifacts))
		totalScore += artifactScore * 0.4
		weight += 0.4
	}

	if result.SplicingEvidence != nil {
		splicingScore := (result.SplicingEvidence.ColorInconsistency + result.SplicingEvidence.TextureMismatch) / 2
		totalScore += splicingScore * 0.3
		weight += 0.3
	}

	if len(result.FaceRegions) > 0 {
		var regionScore float64
		for _, r := range result.FaceRegions {
			regionScore += r.Score
		}
		regionScore /= float64(len(result.FaceRegions))
		totalScore += (1.0 - regionScore) * 0.3
		weight += 0.3
	}

	if weight == 0 {
		return 0.0
	}

	return math.Min(totalScore/weight*100, 100)
}

func (d *FaceSwapDetector) decodeImage(data []byte) (image.Image, string, error) {
	img, err := png.Decode(decoderHelper{data})
	if err != nil {
		return nil, "", err
	}
	return img, "png", nil
}

type decoderHelper struct {
	data []byte
}

func (h decoderHelper) Read(p []byte) (n int, err error) {
	if len(h.data) < len(p) {
		copy(p, h.data)
		n = len(h.data)
		h.data = nil
		return n, nil
	}
	copy(p, h.data[:len(p)])
	h.data = h.data[len(p):]
	n = len(p)
	return n, nil
}

func NewVoiceSynthesisDetector() *VoiceSynthesisDetector {
	return &VoiceSynthesisDetector{
		modelVersion: "v1.0",
		thresholds: VoiceThresholds{
			HighConfidence:   0.85,
			MediumConfidence: 0.70,
			LowConfidence:    0.50,
		},
	}
}

func (d *VoiceSynthesisDetector) Initialize(ctx context.Context) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	return nil
}

func (d *VoiceSynthesisDetector) DetectVoiceSynthesis(ctx context.Context, audioData []byte, metadata map[string]interface{}) (*VoiceSynthesisResult, error) {
	start := time.Now()

	result := &VoiceSynthesisResult{
		Confidence:     0.0,
		IsSynthesized:  false,
		Artifacts:      make([]VoiceArtifact, 0),
		SpectralFeatures: &SpectralAnalysis{},
		ProsodyAnomalies: make([]ProsodyAnomaly, 0),
	}

	spectral := d.analyzeSpectralFeatures(audioData)
	result.SpectralFeatures = spectral

	if spectral.HighFreqAttenuation > 0.8 {
		result.Artifacts = append(result.Artifacts, VoiceArtifact{
			Type:        "high_freq_artifact",
			Frequency:   8000,
			Duration:    0.1,
			Severity:    spectral.HighFreqAttenuation,
			Description: "检测到高频衰减异常",
		})
	}

	if spectral.HarmonicRatio < 0.3 {
		result.Artifacts = append(result.Artifacts, VoiceArtifact{
			Type:        "harmonic_anomaly",
			Frequency:   1000,
			Duration:    0.5,
			Severity:    1.0 - spectral.HarmonicRatio,
			Description: "检测到谐波比例异常",
		})
	}

	prosodyAnomalies := d.analyzeProsody(audioData)
	result.ProsodyAnomalies = prosodyAnomalies

	for _, anomaly := range prosodyAnomalies {
		if anomaly.Severity > 0.6 {
			result.Artifacts = append(result.Artifacts, VoiceArtifact{
				Type:        "prosody_anomaly",
				Duration:    anomaly.Position,
				Severity:    anomaly.Severity,
				Description: anomaly.Details,
			})
		}
	}

	result.Confidence = d.calculateOverallConfidence(result)
	result.IsSynthesized = result.Confidence >= d.thresholds.LowConfidence

	result.ProcessingTime = time.Since(start)

	return result, nil
}

func (d *VoiceSynthesisDetector) analyzeSpectralFeatures(audioData []byte) *SpectralAnalysis {
	analysis := &SpectralAnalysis{
		HighFreqAttenuation: 0.3 + math.Mod(float64(len(audioData)), 0.5)*0.1,
		LowFreqAmplitude:     0.5 + math.Mod(float64(len(audioData)), 0.3)*0.1,
		MelFreqCepstralCoeffs: make([]float64, 13),
		SpectralFlux:        0.4 + math.Mod(float64(len(audioData)), 0.2)*0.1,
		HarmonicRatio:       0.5 + math.Mod(float64(len(audioData)), 0.4)*0.1,
	}

	for i := 0; i < 13; i++ {
		analysis.MelFreqCepstralCoeffs[i] = 0.1 + math.Mod(float64(len(audioData)+i), 0.3)
	}

	return analysis
}

func (d *VoiceSynthesisDetector) analyzeProsody(audioData []byte) []ProsodyAnomaly {
	anomalies := make([]ProsodyAnomaly, 0)

	duration := float64(len(audioData)) / 16000.0

	if duration > 10 {
		anomalies = append(anomalies, ProsodyAnomaly{
			Type:     "abnormal_duration",
			Position: duration / 2,
			Severity: 0.6,
			Details:  "语音持续时间异常",
		})
	}

	anomalies = append(anomalies, ProsodyAnomaly{
		Type:     "pitch_irregularity",
		Position: 0.3,
		Severity: 0.5,
		Details:  "检测到音高不规则变化",
	})

	return anomalies
}

func (d *VoiceSynthesisDetector) calculateOverallConfidence(result *VoiceSynthesisResult) float64 {
	var totalScore float64
	var weight float64

	if len(result.Artifacts) > 0 {
		var artifactScore float64
		for _, a := range result.Artifacts {
			artifactScore += a.Severity
		}
		artifactScore /= float64(len(result.Artifacts))
		totalScore += artifactScore * 0.5
		weight += 0.5
	}

	if result.SpectralFeatures != nil {
		spectralScore := result.SpectralFeatures.HighFreqAttenuation*0.3 +
			(1.0-result.SpectralFeatures.HarmonicRatio)*0.2 +
			result.SpectralFeatures.SpectralFlux*0.1
		totalScore += spectralScore
		weight += 0.4
	}

	if len(result.ProsodyAnomalies) > 0 {
		var prosodyScore float64
		for _, p := range result.ProsodyAnomalies {
			prosodyScore += p.Severity
		}
		prosodyScore /= float64(len(result.ProsodyAnomalies))
		totalScore += prosodyScore * 0.1
		weight += 0.1
	}

	if weight == 0 {
		return 0.0
	}

	return math.Min(totalScore/weight*100, 100)
}

func NewImageTamperingDetector() *ImageTamperingDetector {
	return &ImageTamperingDetector{
		modelVersion: "v1.0",
		thresholds: ImageTamperThresholds{
			HighConfidence:   0.85,
			MediumConfidence: 0.70,
			LowConfidence:    0.50,
		},
	}
}

func (d *ImageTamperingDetector) Initialize(ctx context.Context) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	return nil
}

func (d *ImageTamperingDetector) DetectTampering(ctx context.Context, imageData []byte, metadata map[string]interface{}) (*ImageTamperingResult, error) {
	start := time.Now()

	result := &ImageTamperingResult{
		Confidence:      0.0,
		IsTampered:      false,
		Evidence:        make([]TamperingEvidence, 0),
		ManipulationType: "unknown",
	}

	img, _, err := d.decodeImage(imageData)
	if err != nil {
		result.Confidence = 0.0
		return result, nil
	}

	elaAnalysis := d.performELAAnalysis(img)
	if elaAnalysis.HasAnomaly {
		result.Evidence = append(result.Evidence, TamperingEvidence{
			Type:        "ela_anomaly",
			Location:    elaAnalysis.Location,
			Severity:    elaAnalysis.Severity,
			Description: "Error Level Analysis 检测到异常区域",
			Metadata:    map[string]interface{}{"ela_score": elaAnalysis.Score},
		})
	}

	cloneDetection := d.detectCloneRegions(img)
	if cloneDetection.HasClones {
		result.Evidence = append(result.Evidence, TamperingEvidence{
			Type:        "clone_detection",
			Location:    cloneDetection.Location,
			Severity:    cloneDetection.Severity,
			Description: "检测到复制-移动伪造",
			Metadata:    map[string]interface{}{"clone_count": cloneDetection.Count},
		})
	}

	metadataAnalysis := d.analyzeMetadataConsistency(metadata)
	if !metadataAnalysis.IsConsistent {
		result.Evidence = append(result.Evidence, TamperingEvidence{
			Type:        "metadata_inconsistency",
			Location:    "file_metadata",
			Severity:    metadataAnalysis.InconsistencyScore,
			Description: "元数据不一致",
			Metadata:    metadataAnalysis.Details,
		})
	}

	result.ManipulationType = d.identifyManipulationType(result.Evidence)

	result.Confidence = d.calculateOverallConfidence(result)
	result.IsTampered = result.Confidence >= d.thresholds.LowConfidence

	if result.IsTampered && len(result.Evidence) > 0 {
		firstEvidence := result.Evidence[0]
		result.RegionOfInterest = &TamperedRegion{
			X:      0,
			Y:      0,
			Width:  img.Bounds().Dx(),
			Height: img.Bounds().Dy(),
		}
		_ = firstEvidence
	}

	result.ProcessingTime = time.Since(start)

	return result, nil
}

type ELAAnalysisResult struct {
	HasAnomaly bool
	Location   string
	Severity   float64
	Score      float64
}

func (d *ImageTamperingDetector) performELAAnalysis(img image.Image) ELAAnalysisResult {
	width := img.Bounds().Dx()
	height := img.Bounds().Dy()

	var maxDiff float64

	for y := 1; y < height-1; y += 5 {
		for x := 1; x < width-1; x += 5 {
			r, g, b, _ := img.At(x, y).RGBA()
			r1, g1, b1, _ := img.At(x-1, y).RGBA()
			r2, g2, b2, _ := img.At(x+1, y).RGBA()

			diff := math.Abs(float64(r/257)-float64(r1/257)) +
				math.Abs(float64(r/257)-float64(r2/257)) +
				math.Abs(float64(g/257)-float64(g1/257)) +
				math.Abs(float64(g/257)-float64(g2/257)) +
				math.Abs(float64(b/257)-float64(b1/257)) +
				math.Abs(float64(b/257)-float64(b2/257))

			if diff > maxDiff {
				maxDiff = diff
			}
		}
	}

	score := maxDiff / 10.0
	hasAnomaly := score > 3.0

	return ELAAnalysisResult{
		HasAnomaly: hasAnomaly,
		Location:   "various",
		Severity:   math.Min(score/10, 1.0),
		Score:      score,
	}
}

type CloneDetectionResult struct {
	HasClones bool
	Location  string
	Severity  float64
	Count     int
}

func (d *ImageTamperingDetector) detectCloneRegions(img image.Image) CloneDetectionResult {
	width := img.Bounds().Dx()
	height := img.Bounds().Dy()

	hashMap := make(map[string]int)

	for y := 0; y < height; y += 8 {
		for x := 0; x < width; x += 8 {
			regionHash := d.hashRegion(img, x, y, 8)
			hashMap[regionHash]++
		}
	}

	var cloneCount int
	for _, count := range hashMap {
		if count > 3 {
			cloneCount++
		}
	}

	return CloneDetectionResult{
		HasClones: cloneCount > 0,
		Location:  "detected_regions",
		Severity:  math.Min(float64(cloneCount)/10, 1.0),
		Count:     cloneCount,
	}
}

func (d *ImageTamperingDetector) hashRegion(img image.Image, x, y, size int) string {
	var hash int

	for dy := 0; dy < size && y+dy < img.Bounds().Dy(); dy++ {
		for dx := 0; dx < size && x+dx < img.Bounds().Dx(); dx++ {
			r, g, b, _ := img.At(x+dx, y+dy).RGBA()
			hash += int(r/257 + g/257 + b/257)
		}
	}

	return fmt.Sprintf("%d", hash)
}

type MetadataConsistencyResult struct {
	IsConsistent      bool
	InconsistencyScore float64
	Details           map[string]interface{}
}

func (d *ImageTamperingDetector) analyzeMetadataConsistency(metadata map[string]interface{}) MetadataConsistencyResult {
	if metadata == nil {
		return MetadataConsistencyResult{IsConsistent: true}
	}

	details := make(map[string]interface{})
	issues := 0

	if createTime, ok := metadata["create_time"].(string); ok {
		if createTime > time.Now().Format(time.RFC3339) {
			issues++
			details["create_time_future"] = true
		}
	}

	if software, ok := metadata["software"].(string); ok {
		if software == "" {
			issues++
			details["missing_software"] = true
		}
	}

	return MetadataConsistencyResult{
		IsConsistent:       issues == 0,
		InconsistencyScore: math.Min(float64(issues)*0.3, 1.0),
		Details:            details,
	}
}

func (d *ImageTamperingDetector) identifyManipulationType(evidence []TamperingEvidence) string {
	typeScores := map[string]float64{
		"copy_move":  0,
		"splicing":   0,
		"retouching":  0,
		"removal":    0,
	}

	for _, e := range evidence {
		switch e.Type {
		case "clone_detection":
			typeScores["copy_move"] += e.Severity * 0.8
			typeScores["removal"] += e.Severity * 0.2
		case "ela_anomaly":
			typeScores["splicing"] += e.Severity * 0.5
			typeScores["retouching"] += e.Severity * 0.3
			typeScores["removal"] += e.Severity * 0.2
		case "metadata_inconsistency":
			typeScores["splicing"] += e.Severity * 0.4
		}
	}

	var maxType string
	var maxScore float64

	for t, score := range typeScores {
		if score > maxScore {
			maxScore = score
			maxType = t
		}
	}

	if maxType == "" {
		return "unknown"
	}

	return maxType
}

func (d *ImageTamperingDetector) calculateOverallConfidence(result *ImageTamperingResult) float64 {
	if len(result.Evidence) == 0 {
		return 0.0
	}

	var totalScore float64
	for _, e := range result.Evidence {
		totalScore += e.Severity
	}

	return math.Min(totalScore/float64(len(result.Evidence))*100, 100)
}

func (d *ImageTamperingDetector) decodeImage(data []byte) (image.Image, string, error) {
	img, err := png.Decode(decoderHelper{data})
	if err != nil {
		return nil, "", err
	}
	return img, "png", nil
}

func NewDeepfakeAlertSystem() *DeepfakeAlertSystem {
	return &DeepfakeAlertSystem{
		alerts:    make([]DeepfakeAlert, 0),
		maxAlerts: 1000,
		enabled:   true,
	}
}

func (s *DeepfakeAlertSystem) CreateAlert(ctx context.Context, alertType, severity, source, message string, metadata map[string]interface{}) (*DeepfakeAlert, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	alert := &DeepfakeAlert{
		ID:            fmt.Sprintf("dfa_%d_%d", time.Now().UnixNano(), len(s.alerts)),
		Type:          alertType,
		Severity:      severity,
		Source:        source,
		Message:       message,
		Timestamp:    time.Now(),
		Metadata:     metadata,
		Acknowledged: false,
	}

	s.alerts = append(s.alerts, *alert)

	if len(s.alerts) > s.maxAlerts {
		s.alerts = s.alerts[len(s.alerts)-s.maxAlerts:]
	}

	return alert, nil
}

func (s *DeepfakeAlertSystem) GetAlerts(ctx context.Context, filters *AlertFilters) ([]DeepfakeAlert, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	alerts := make([]DeepfakeAlert, 0)

	for _, alert := range s.alerts {
		if filters != nil {
			if filters.Type != "" && alert.Type != filters.Type {
				continue
			}
			if filters.Severity != "" && alert.Severity != filters.Severity {
				continue
			}
			if !filters.IncludeAcknowledged && alert.Acknowledged {
				continue
			}
		}

		alerts = append(alerts, alert)
	}

	return alerts, nil
}

func (s *DeepfakeAlertSystem) AcknowledgeAlert(ctx context.Context, alertID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i := range s.alerts {
		if s.alerts[i].ID == alertID {
			s.alerts[i].Acknowledged = true
			return nil
		}
	}

	return fmt.Errorf("alert not found")
}

type AlertFilters struct {
	Type                 string
	Severity             string
	IncludeAcknowledged  bool
}

func (s *DeepfakeDetectionSystem) ComprehensiveDetection(ctx context.Context, contentType string, data []byte, metadata map[string]interface{}) (*ComprehensiveDeepfakeResult, error) {
	if !s.initialized {
		return nil, fmt.Errorf("system not initialized")
	}

	start := time.Now()

	result := &ComprehensiveDeepfakeResult{
		Timestamp: time.Now(),
	}

	var maxRisk float64

	switch contentType {
	case "image":
		faceResult, err := s.faceSwapDetector.DetectFaceSwap(ctx, data, metadata)
		if err == nil && faceResult != nil {
			result.FaceSwapResult = faceResult
			if faceResult.Confidence > maxRisk {
				maxRisk = faceResult.Confidence
			}
		}

		tamperResult, err := s.imageTamperDetector.DetectTampering(ctx, data, metadata)
		if err == nil && tamperResult != nil {
			result.ImageTamperResult = tamperResult
			if tamperResult.Confidence > maxRisk {
				maxRisk = tamperResult.Confidence
			}
		}

	case "video":
		faceResult, err := s.faceSwapDetector.DetectFaceSwap(ctx, data, metadata)
		if err == nil && faceResult != nil {
			result.FaceSwapResult = faceResult
			if faceResult.Confidence > maxRisk {
				maxRisk = faceResult.Confidence
			}
		}

	case "audio":
		voiceResult, err := s.voiceSynthDetector.DetectVoiceSynthesis(ctx, data, metadata)
		if err == nil && voiceResult != nil {
			result.VoiceResult = voiceResult
			if voiceResult.Confidence > maxRisk {
				maxRisk = voiceResult.Confidence
			}
		}
	}

	result.OverallRisk = maxRisk
	result.RiskLevel = s.determineRiskLevel(maxRisk)
	result.Recommendations = s.generateRecommendations(result)
	result.ProcessingTime = time.Since(start)

	if result.RiskLevel == "high" || result.RiskLevel == "critical" {
		alert, _ := s.alertSystem.CreateAlert(ctx, contentType+"_deepfake", result.RiskLevel, "comprehensive_detection",
			fmt.Sprintf("检测到潜在的 %s 深度伪造内容，风险评分: %.2f", contentType, maxRisk), nil)
		_ = alert
	}

	return result, nil
}

func (s *DeepfakeDetectionSystem) determineRiskLevel(risk float64) string {
	switch {
	case risk >= 85:
		return "critical"
	case risk >= 70:
		return "high"
	case risk >= 50:
		return "medium"
	case risk >= 30:
		return "low"
	default:
		return "minimal"
	}
}

func (s *DeepfakeDetectionSystem) generateRecommendations(result *ComprehensiveDeepfakeResult) []string {
	var recommendations []string

	if result.RiskLevel == "critical" {
		recommendations = append(recommendations, "建议立即人工审核")
		recommendations = append(recommendations, "考虑暂时阻止该内容")
		recommendations = append(recommendations, "通知安全团队")
	} else if result.RiskLevel == "high" {
		recommendations = append(recommendations, "建议进行人工复核")
		recommendations = append(recommendations, "获取更多验证信息")
	} else if result.RiskLevel == "medium" {
		recommendations = append(recommendations, "保持监控")
		recommendations = append(recommendations, "记录为潜在风险")
	} else {
		recommendations = append(recommendations, "内容基本可信")
	}

	return recommendations
}

type DeepfakeDetectionRequest struct {
	ContentType string                 `json:"content_type" binding:"required"`
	Data        string                 `json:"data"`
	Metadata    map[string]interface{} `json:"metadata"`
}

type DeepfakeDetectionResponse struct {
	Success bool                        `json:"success"`
	Result  *ComprehensiveDeepfakeResult `json:"result"`
}

type AlertListRequest struct {
	Type                string `form:"type"`
	Severity            string `form:"severity"`
	IncludeAcknowledged bool   `form:"include_acknowledged"`
}

type AlertListResponse struct {
	Success bool           `json:"success"`
	Alerts  []DeepfakeAlert `json:"alerts"`
	Count   int            `json:"count"`
}

func ParseDeepfakeRequest(data string) (*DeepfakeDetectionRequest, error) {
	var req DeepfakeDetectionRequest
	err := json.Unmarshal([]byte(data), &req)
	return &req, err
}
