package model

import (
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"
)

type AudioContextProperty struct {
	SampleRate           int    `json:"sample_rate"`
	State                string `json:"state"`
	NumberOfInputs       int    `json:"number_of_inputs"`
	NumberOfOutputs      int    `json:"number_of_outputs"`
	ChannelCount         int    `json:"channel_count"`
	ChannelCountMode     string `json:"channel_count_mode"`
	ChannelInterpretation string `json:"channel_interpretation"`
	LatencyHint          string `json:"latency_hint"`
	BaseLatency          float64 `json:"base_latency"`
	OutputTimestamp      float64 `json:"output_timestamp"`
	CurrentTime          float64 `json:"current_time"`
}

type AudioDestinationInfo struct {
	Type                 string `json:"type"`
	Context              string `json:"context"`
	NumberOfInputs       int    `json:"number_of_inputs"`
	NumberOfOutputs      int    `json:"number_of_outputs"`
	ChannelCount         int    `json:"channel_count"`
	ChannelCountMode     string `json:"channel_count_mode"`
}

type AudioNodeInfo struct {
	ContextTime          float64 `json:"context_time"`
	NumberOfInputs       int    `json:"number_of_inputs"`
	NumberOfOutputs      int    `json:"number_of_outputs"`
	ChannelCount         int    `json:"channel_count"`
	ChannelCountMode     string `json:"channel_count_mode"`
	ChannelInterpretation string `json:"channel_interpretation"`
}

type OscillatorConfig struct {
	Type                 string  `json:"type"`
	Frequency            float64 `json:"frequency"`
	Detune               float64 `json:"detune"`
}

type GainNodeConfig struct {
	Gain                float64 `json:"gain"`
}

type AnalyserNodeConfig struct {
	FFTSize             int     `json:"fft_size"`
	FrequencyBinCount   int     `json:"frequency_bin_count"`
	MinDecibels         float64 `json:"min_decibels"`
	MaxDecibels         float64 `json:"max_decibels"`
	SmoothingTimeConstant float64 `json:"smoothing_time_constant"`
}

type AudioProcessingData struct {
	FrequencyData        []float64 `json:"frequency_data"`
	TimeDomainData       []float64 `json:"time_domain_data"`
	PeakFrequency        float64   `json:"peak_frequency"`
	PeakAmplitude        float64   `json:"peak_amplitude"`
	RMSAmplitude         float64   `json:"rms_amplitude"`
	SpectralCentroid     float64   `json:"spectral_centroid"`
	SpectralFlatness     float64   `json:"spectral_flatness"`
	ZeroCrossingRate     float64   `json:"zero_crossing_rate"`
	TotalHarmonicDistortion float64 `json:"total_harmonic_distortion"`
}

type AudioFingerprint struct {
	FingerprintID        string                 `json:"fingerprint_id"`
	AudioHash            string                 `json:"audio_hash"`
	ContextProperties    AudioContextProperty   `json:"context_properties"`
	DestinationInfo      AudioDestinationInfo   `json:"destination_info"`
	ProcessingData       AudioProcessingData    `json:"processing_data"`
	OscillatorConfig     OscillatorConfig       `json:"oscillator_config"`
	GainNodeConfig       GainNodeConfig         `json:"gain_node_config"`
	AnalyserConfig       AnalyserNodeConfig     `json:"analyser_config"`
	SupportedFormats     []string               `json:"supported_formats"`
	DecodingCapabilities []string               `json:"decoding_capabilities"`
	IsAudioContextSupported bool                `json:"is_audio_context_supported"`
	IsOscillatorSupported  bool                `json:"is_oscillator_supported"`
	IsAnalyserSupported    bool                `json:"is_analyser_supported"`
	IsGainNodeSupported    bool                `json:"is_gain_node_supported"`
	IsStereoPannerSupported bool                `json:"is_stereo_panner_supported"`
	RenderingMode        string                 `json:"rendering_mode"`
	RenderingLatency     float64                `json:"rendering_latency"`
	IsOfflineContext     bool                   `json:"is_offline_context"`
	OfflineDuration      float64                `json:"offline_duration"`
	MaxChannelCount      int                    `json:"max_channel_count"`
	RenderingConsistency float64                `json:"rendering_consistency"`
	NoiseLevel           float64                `json:"noise_level"`
	IsSoftwareRenderer   bool                   `json:"is_software_renderer"`
	IsHardwareAccelerated bool                  `json:"is_hardware_accelerated"`
	BrowserAudioAPIVersion string               `json:"browser_audio_api_version"`
	Timestamp            time.Time              `json:"timestamp"`
	AnalysisDuration     float64                `json:"analysis_duration"`
}

type AudioFingerprintAnalysis struct {
	FingerprintID        string                `json:"fingerprint_id"`
	Similarity           float64               `json:"similarity"`
	CommonFeatures       []string              `json:"common_features"`
	DiffFeatures         []string              `json:"diff_features"`
	AnomalyScore         float64               `json:"anomaly_score"`
	AnomalyIndicators    []string              `json:"anomaly_indicators"`
	RenderingAnalysis    *RenderingAnalysis    `json:"rendering_analysis"`
	ProcessingAnalysis   *ProcessingAnalysis   `json:"processing_analysis"`
	ConsistencyScore     float64               `json:"consistency_score"`
	UniquenessScore      float64               `json:"uniqueness_score"`
	Confidence           float64               `json:"confidence"`
	IsSuspicious         bool                  `json:"is_suspicious"`
	SuspiciousReasons    []string              `json:"suspicious_reasons"`
}

type RenderingAnalysis struct {
	RenderingMode        string   `json:"rendering_mode"`
	RenderingLatency     float64  `json:"rendering_latency"`
	IsConsistent         bool     `json:"is_consistent"`
	ConsistencyScore     float64  `json:"consistency_score"`
	LatencyVariance      float64  `json:"latency_variance"`
	IsHardwareAccelerated bool    `json:"is_hardware_accelerated"`
	IsSoftwareRenderer   bool     `json:"is_software_renderer"`
	RenderingArtifacts   []string `json:"rendering_artifacts"`
}

type ProcessingAnalysis struct {
	ProcessingTime       float64  `json:"processing_time"`
	ProcessingEfficiency float64  `json:"processing_efficiency"`
	DataQuality         float64  `json:"data_quality"`
	SignalToNoiseRatio   float64  `json:"signal_to_noise_ratio"`
	FrequencyStability   float64  `json:"frequency_stability"`
	IsStable             bool     `json:"is_stable"`
	ProcessingWarnings   []string `json:"processing_warnings"`
}

type AudioComparisonResult struct {
	SimilarityScore      float64                `json:"similarity_score"`
	ContextMatchScore    float64                `json:"context_match_score"`
	ProcessingMatchScore float64                `json:"processing_match_score"`
	HashMatchScore       float64                `json:"hash_match_score"`
	OverallMatch         bool                   `json:"overall_match"`
	Confidence           float64                `json:"confidence"`
	MatchDetails         *MatchDetails          `json:"match_details"`
	MismatchDetails      *MismatchDetails       `json:"mismatch_details"`
	Recommendation       string                 `json:"recommendation"`
}

type MatchDetails struct {
	MatchingProperties   []string `json:"matching_properties"`
	SimilarValues       map[string]float64 `json:"similar_values"`
	CloseMatches        []string `json:"close_matches"`
}

type MismatchDetails struct {
	MismatchedProperties []string `json:"mismatched_properties"`
	SignificantDiffs    map[string]float64 `json:"significant_diffs"`
	MinorDiffs          []string `json:"minor_diffs"`
}

func (af *AudioFingerprint) GenerateHash() string {
	hasher := &AudioHashBuilder{}

	hasher.AddInt(af.ContextProperties.SampleRate)
	hasher.AddString(af.ContextProperties.ChannelCountMode)
	hasher.AddInt(af.ContextProperties.ChannelCount)
	hasher.AddInt(af.MaxChannelCount)
	hasher.AddFloat(af.ProcessingData.PeakFrequency)
	hasher.AddFloat(af.ProcessingData.RMSAmplitude)
	hasher.AddFloat(af.ProcessingData.SpectralCentroid)
	hasher.AddString(af.OscillatorConfig.Type)
	hasher.AddInt(af.AnalyserConfig.FFTSize)
	hasher.AddBool(af.IsAudioContextSupported)
	hasher.AddBool(af.IsHardwareAccelerated)

	return hasher.Finalize()
}

type AudioHashBuilder struct {
	components []string
}

func (b *AudioHashBuilder) AddString(s string) {
	if s != "" {
		b.components = append(b.components, fmt.Sprintf("s:%s", s))
	}
}

func (b *AudioHashBuilder) AddInt(i int) {
	b.components = append(b.components, fmt.Sprintf("i:%d", i))
}

func (b *AudioHashBuilder) AddFloat(f float64) {
	if !math.IsNaN(f) && !math.IsInf(f, 0) {
		b.components = append(b.components, fmt.Sprintf("f:%.2f", f))
	}
}

func (b *AudioHashBuilder) AddBool(b_ bool) {
	if b_ {
		b.components = append(b.components, "b:1")
	}
}

func (b *AudioHashBuilder) Finalize() string {
	sort.Strings(b.components)
	combined := strings.Join(b.components, "|")
	return fmt.Sprintf("%x", len(combined))
}

func (af *AudioFingerprint) ValidateFingerprint() (bool, []string) {
	issues := make([]string, 0)

	if af.ContextProperties.SampleRate == 0 {
		issues = append(issues, "missing_sample_rate")
	}

	if af.ContextProperties.ChannelCount == 0 {
		issues = append(issues, "missing_channel_count")
	}

	if len(af.ProcessingData.FrequencyData) == 0 {
		issues = append(issues, "missing_frequency_data")
	}

	if af.ProcessingData.PeakFrequency == 0 {
		issues = append(issues, "missing_peak_frequency")
	}

	if !af.IsAudioContextSupported && !af.IsOscillatorSupported {
		issues = append(issues, "no_audio_support")
	}

	return len(issues) == 0, issues
}

func (af *AudioFingerprint) CalculateComplexityScore() float64 {
	score := 0.0

	if af.ContextProperties.SampleRate > 0 {
		score += 10
		if af.ContextProperties.SampleRate >= 44100 {
			score += 5
		}
	}

	if af.MaxChannelCount >= 2 {
		score += 10
		if af.MaxChannelCount >= 6 {
			score += 5
		}
	}

	if len(af.ProcessingData.FrequencyData) > 0 {
		score += 15
	}

	if len(af.SupportedFormats) > 0 {
		score += 10
	}

	if af.IsHardwareAccelerated {
		score += 15
	}

	if af.RenderingConsistency > 0.9 {
		score += 10
	}

	return math.Min(score, 100)
}

func (af *AudioFingerprint) GetRenderingCharacteristics() map[string]interface{} {
	chars := make(map[string]interface{})

	chars["sample_rate"] = af.ContextProperties.SampleRate
	chars["channel_count"] = af.ContextProperties.ChannelCount
	chars["rendering_mode"] = af.RenderingMode
	chars["latency"] = af.RenderingLatency
	chars["is_hardware"] = af.IsHardwareAccelerated
	chars["is_software"] = af.IsSoftwareRenderer

	if len(af.ProcessingData.FrequencyData) > 0 {
		chars["has_frequency_data"] = true
		chars["peak_freq"] = af.ProcessingData.PeakFrequency
		chars["spectral_centroid"] = af.ProcessingData.SpectralCentroid
	}

	return chars
}

func (afa *AudioFingerprintAnalysis) AddAnomalyIndicator(indicator string, weight float64) {
	for _, existing := range afa.AnomalyIndicators {
		if existing == indicator {
			return
		}
	}
	afa.AnomalyIndicators = append(afa.AnomalyIndicators, indicator)
	afa.AnomalyScore += weight
}

func (afa *AudioFingerprintAnalysis) CalculateFinalScore() float64 {
	if afa.AnomalyScore > 100 {
		afa.AnomalyScore = 100
	}

	baseScore := 100 - afa.AnomalyScore

	if afa.ConsistencyScore > 0 {
		baseScore = baseScore * 0.6 + afa.ConsistencyScore * 0.4
	}

	if afa.Confidence > 0 {
		baseScore = baseScore * (0.7 + afa.Confidence * 0.3)
	}

	return math.Max(0, math.Min(100, baseScore))
}

func (afa *AudioFingerprintAnalysis) DetermineSuspicious() bool {
	if afa.AnomalyScore > 50 {
		afa.IsSuspicious = true
		afa.SuspiciousReasons = append(afa.SuspiciousReasons, "high_anomaly_score")
	}

	if afa.ConsistencyScore < 0.7 {
		afa.IsSuspicious = true
		afa.SuspiciousReasons = append(afa.SuspiciousReasons, "low_consistency")
	}

	if afa.UniquenessScore > 95 {
		afa.IsSuspicious = true
		afa.SuspiciousReasons = append(afa.SuspiciousReasons, "unusually_unique")
	}

	if len(afa.AnomalyIndicators) > 5 {
		afa.IsSuspicious = true
		afa.SuspiciousReasons = append(afa.SuspiciousReasons, "multiple_anomalies")
	}

	return afa.IsSuspicious
}

func (acr *AudioComparisonResult) CalculateOverallSimilarity() float64 {
	total := 0.0
	count := 0.0

	if acr.ContextMatchScore > 0 {
		total += acr.ContextMatchScore * 0.3
		count += 0.3
	}

	if acr.ProcessingMatchScore > 0 {
		total += acr.ProcessingMatchScore * 0.4
		count += 0.4
	}

	if acr.HashMatchScore > 0 {
		total += acr.HashMatchScore * 0.3
		count += 0.3
	}

	if count == 0 {
		return 0
	}

	return (total / count) * 100
}

func (acr *AudioComparisonResult) GenerateRecommendation() string {
	similarity := acr.CalculateOverallSimilarity()

	if similarity >= 90 {
		return "very_likely_same_source"
	} else if similarity >= 70 {
		return "likely_same_browser_family"
	} else if similarity >= 50 {
		return "possible_same_device"
	} else if similarity >= 30 {
		return "unlikely_related"
	}

	return "different_sources"
}

func (afa *AudioFingerprintAnalysis) ToJSON() (string, error) {
	data, err := json.MarshalIndent(afa, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func ParseAudioFingerprintAnalysis(data string) (*AudioFingerprintAnalysis, error) {
	var analysis AudioFingerprintAnalysis
	err := json.Unmarshal([]byte(data), &analysis)
	return &analysis, err
}

type AudioFingerprintDatabase struct {
	Fingerprints    map[string]*AudioFingerprint
	SimilarityIndex map[string][]string
	mu              sync.RWMutex
}

type sync struct{}

func NewAudioFingerprintDatabase() *AudioFingerprintDatabase {
	return &AudioFingerprintDatabase{
		Fingerprints:    make(map[string]*AudioFingerprint),
		SimilarityIndex: make(map[string][]string),
	}
}

func (db *AudioFingerprintDatabase) AddFingerprint(fp *AudioFingerprint) {
	db.mu.Lock()
	defer db.mu.Unlock()

	if fp.FingerprintID == "" {
		fp.FingerprintID = fmt.Sprintf("afp_%d_%d", time.Now().UnixNano(), len(db.Fingerprints))
	}

	if fp.AudioHash == "" {
		fp.AudioHash = fp.GenerateHash()
	}

	fp.Timestamp = time.Now()
	db.Fingerprints[fp.FingerprintID] = fp

	db.updateSimilarityIndex(fp)
}

func (db *AudioFingerprintDatabase) GetFingerprint(fpID string) (*AudioFingerprint, bool) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	fp, exists := db.Fingerprints[fpID]
	return fp, exists
}

func (db *AudioFingerprintDatabase) CalculateSimilarity(fp1, fp2 *AudioFingerprint) float64 {
	if fp1 == nil || fp2 == nil {
		return 0
	}

	fields := []struct {
		name   string
		val1   interface{}
		val2   interface{}
		weight float64
	}{
		{"sample_rate", fp1.ContextProperties.SampleRate, fp2.ContextProperties.SampleRate, 15},
		{"channel_count", fp1.ContextProperties.ChannelCount, fp2.ContextProperties.ChannelCount, 10},
		{"channel_mode", fp1.ContextProperties.ChannelCountMode, fp2.ContextProperties.ChannelCountMode, 10},
		{"peak_frequency", fp1.ProcessingData.PeakFrequency, fp2.ProcessingData.PeakFrequency, 15},
		{"rms_amplitude", fp1.ProcessingData.RMSAmplitude, fp2.ProcessingData.RMSAmplitude, 10},
		{"spectral_centroid", fp1.ProcessingData.SpectralCentroid, fp2.ProcessingData.SpectralCentroid, 10},
		{"oscillator_type", fp1.OscillatorConfig.Type, fp2.OscillatorConfig.Type, 10},
		{"fft_size", fp1.AnalyserConfig.FFTSize, fp2.AnalyserConfig.FFTSize, 10},
		{"rendering_mode", fp1.RenderingMode, fp2.RenderingMode, 5},
		{"hardware_accelerated", fp1.IsHardwareAccelerated, fp2.IsHardwareAccelerated, 5},
	}

	totalWeight := 0.0
	matchWeight := 0.0

	for _, field := range fields {
		totalWeight += field.weight
		if fmt.Sprintf("%v", field.val1) == fmt.Sprintf("%v", field.val2) {
			matchWeight += field.weight
		}
	}

	if totalWeight == 0 {
		return 0
	}

	return (matchWeight / totalWeight) * 100
}

func (db *AudioFingerprintDatabase) FindSimilarFingerprints(fpID string, threshold float64) []*AudioFingerprint {
	db.mu.RLock()
	defer db.mu.RUnlock()

	target, exists := db.Fingerprints[fpID]
	if !exists {
		return nil
	}

	similar := make([]*AudioFingerprint, 0)

	for id, fp := range db.Fingerprints {
		if id == fpID {
			continue
		}

		similarity := db.CalculateSimilarity(target, fp)
		if similarity >= threshold {
			similar = append(similar, fp)
		}
	}

	sort.Slice(similar, func(i, j int) bool {
		return db.CalculateSimilarity(target, similar[i]) > db.CalculateSimilarity(target, similar[j])
	})

	return similar
}

func (db *AudioFingerprintDatabase) updateSimilarityIndex(fp *AudioFingerprint) {
	for otherID, otherFP := range db.Fingerprints {
		if otherID == fp.FingerprintID {
			continue
		}

		similarity := db.CalculateSimilarity(fp, otherFP)
		if similarity > 70 {
			db.SimilarityIndex[fp.FingerprintID] = append(
				db.SimilarityIndex[fp.FingerprintID], otherID,
			)
			db.SimilarityIndex[otherID] = append(
				db.SimilarityIndex[otherID], fp.FingerprintID,
			)
		}
	}
}

func (db *AudioFingerprintDatabase) GetAllFingerprints() []*AudioFingerprint {
	db.mu.RLock()
	defer db.mu.RUnlock()

	result := make([]*AudioFingerprint, 0, len(db.Fingerprints))
	for _, fp := range db.Fingerprints {
		result = append(result, fp)
	}

	return result
}

func (db *AudioFingerprintDatabase) RemoveFingerprint(fpID string) {
	db.mu.Lock()
	defer db.mu.Unlock()

	delete(db.Fingerprints, fpID)

	delete(db.SimilarityIndex, fpID)
	for otherID, similar := range db.SimilarityIndex {
		newSimilar := make([]string, 0)
		for _, simID := range similar {
			if simID != fpID {
				newSimilar = append(newSimilar, simID)
			}
		}
		db.SimilarityIndex[otherID] = newSimilar
	}
}

func (db *AudioFingerprintDatabase) ExportData() ([]byte, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	data := map[string]interface{}{
		"fingerprints": db.Fingerprints,
		"exported_at":  time.Now(),
	}

	return json.MarshalIndent(data, "", "  ")
}

func (db *AudioFingerprintDatabase) ImportData(data []byte) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	type ImportData struct {
		Fingerprints map[string]*AudioFingerprint `json:"fingerprints"`
	}

	var importData ImportData
	if err := json.Unmarshal(data, &importData); err != nil {
		return err
	}

	for id, fp := range importData.Fingerprints {
		fp.FingerprintID = id
		db.Fingerprints[id] = fp
	}

	return nil
}

type AudioFingerprintStats struct {
	TotalFingerprints      int64                  `json:"total_fingerprints"`
	HardwareAccelerated    int64                  `json:"hardware_accelerated"`
	SoftwareRendered       int64                  `json:"software_rendered"`
	AvgRenderingLatency    float64                `json:"avg_rendering_latency"`
	AvgConsistencyScore    float64                `json:"avg_consistency_score"`
	AvgUniquenessScore     float64                `json:"avg_uniqueness_score"`
	SuspiciousCount        int64                  `json:"suspicious_count"`
	BrowserDistribution    map[string]int64       `json:"browser_distribution"`
	PlatformDistribution   map[string]int64       `json:"platform_distribution"`
	HighRiskCount          int64                  `json:"high_risk_count"`
	MediumRiskCount        int64                  `json:"medium_risk_count"`
	LowRiskCount           int64                  `json:"low_risk_count"`
}

func (db *AudioFingerprintDatabase) CalculateStats() *AudioFingerprintStats {
	db.mu.RLock()
	defer db.mu.RUnlock()

	stats := &AudioFingerprintStats{
		BrowserDistribution:  make(map[string]int64),
		PlatformDistribution: make(map[string]int64),
	}

	var totalLatency float64
	var totalConsistency float64
	var totalUniqueness float64
	var suspiciousCount int64

	for _, fp := range db.Fingerprints {
		stats.TotalFingerprints++

		if fp.IsHardwareAccelerated {
			stats.HardwareAccelerated++
		} else {
			stats.SoftwareRendered++
		}

		totalLatency += fp.RenderingLatency
		totalConsistency += fp.RenderingConsistency

		if fp.RenderingConsistency > 0.95 && fp.ProcessingData.SpectralFlatness < 0.1 {
			suspiciousCount++
		}

		if fp.BrowserAudioAPIVersion != "" {
			stats.BrowserDistribution[fp.BrowserAudioAPIVersion]++
		}
	}

	if stats.TotalFingerprints > 0 {
		stats.AvgRenderingLatency = totalLatency / float64(stats.TotalFingerprints)
		stats.AvgConsistencyScore = totalConsistency / float64(stats.TotalFingerprints)
		stats.SuspiciousCount = suspiciousCount
	}

	return stats
}
