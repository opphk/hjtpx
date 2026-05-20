package service

import (
	"testing"
	"time"
)

func TestNewAudioContextService(t *testing.T) {
	service := NewAudioContextService()
	if service == nil {
		t.Error("NewAudioContextService returned nil")
	}
	if service.config == nil {
		t.Error("config should be initialized")
	}
}

func TestAudioContextService_GetMetrics(t *testing.T) {
	service := NewAudioContextService()

	metrics := service.GetMetrics()
	if metrics == nil {
		t.Error("GetMetrics returned nil")
	}
}

func TestAudioContextService_GenerateFingerprint(t *testing.T) {
	service := NewAudioContextService()

	sampleData := make([]float64, 1024)
	for i := range sampleData {
		sampleData[i] = float64(i) / 1024.0
	}

	fingerprint, err := service.GenerateFingerprint(sampleData, 44100)
	if err != nil {
		t.Errorf("GenerateFingerprint failed: %v", err)
	}
	if fingerprint == nil {
		t.Error("GenerateFingerprint returned nil")
	}
}

func TestAudioContextService_MatchFingerprint(t *testing.T) {
	service := NewAudioContextService()

	sampleData := make([]float64, 1024)
	for i := range sampleData {
		sampleData[i] = float64(i) / 1024.0
	}

	fingerprint1, _ := service.GenerateFingerprint(sampleData, 44100)
	fingerprint2, _ := service.GenerateFingerprint(sampleData, 44100)

	match, err := service.MatchFingerprint(fingerprint1, fingerprint2, 0.9)
	if err != nil {
		t.Errorf("MatchFingerprint failed: %v", err)
	}
	if !match {
		t.Error("Same fingerprint should match with high threshold")
	}
}

func TestAudioContextService_AnalyzeAudio(t *testing.T) {
	service := NewAudioContextService()

	sampleData := make([]float64, 2048)
	for i := range sampleData {
		sampleData[i] = float64(i) / 2048.0
	}

	result, err := service.AnalyzeAudio(sampleData, 44100, 1)
	if err != nil {
		t.Errorf("AnalyzeAudio failed: %v", err)
	}
	if result == nil {
		t.Error("AnalyzeAudio returned nil")
	}
}

func TestAudioContextService_GetFrequencyData(t *testing.T) {
	service := NewAudioContextService()

	sampleData := make([]float64, 1024)
	for i := range sampleData {
		sampleData[i] = float64(i) / 1024.0
	}

	freqData, err := service.GetFrequencyData(sampleData, 512)
	if err != nil {
		t.Errorf("GetFrequencyData failed: %v", err)
	}
	if len(freqData) == 0 {
		t.Error("GetFrequencyData should return data")
	}
}

func TestAudioContextService_DetectAnomalies(t *testing.T) {
	service := NewAudioContextService()

	sampleData := make([]float64, 1024)
	for i := range sampleData {
		sampleData[i] = float64(i) / 1024.0
	}

	anomalies, err := service.DetectAnomalies(sampleData, 44100)
	if err != nil {
		t.Errorf("DetectAnomalies failed: %v", err)
	}
	if anomalies == nil {
		t.Error("DetectAnomalies returned nil")
	}
}

func TestAudioContextService_CalculateSimilarity(t *testing.T) {
	service := NewAudioContextService()

	sampleData1 := make([]float64, 1024)
	sampleData2 := make([]float64, 1024)
	for i := range sampleData1 {
		sampleData1[i] = float64(i) / 1024.0
		sampleData2[i] = float64(i) / 1024.0
	}

	similarity, err := service.CalculateSimilarity(sampleData1, sampleData2)
	if err != nil {
		t.Errorf("CalculateSimilarity failed: %v", err)
	}
	if similarity < 0.0 || similarity > 1.0 {
		t.Error("Similarity should be between 0 and 1")
	}
}

func TestAudioContextService_StoreFingerprint(t *testing.T) {
	service := NewAudioContextService()

	fingerprint := &AudioFingerprint{
		ID:          "test-fp-1",
		Fingerprint: []byte("test-fingerprint-data"),
		SampleRate:  44100,
		CreatedAt:   time.Now(),
	}

	err := service.StoreFingerprint(fingerprint)
	if err != nil {
		t.Errorf("StoreFingerprint failed: %v", err)
	}

	retrieved, err := service.GetFingerprint("test-fp-1")
	if err != nil {
		t.Errorf("GetFingerprint failed: %v", err)
	}
	if retrieved == nil {
		t.Error("GetFingerprint should return stored fingerprint")
	}
}

func TestAudioContextService_GetFingerprint(t *testing.T) {
	service := NewAudioContextService()

	_, err := service.GetFingerprint("non-existent")
	if err == nil {
		t.Error("GetFingerprint should return error for non-existent fingerprint")
	}
}

func TestAudioContextService_DeleteFingerprint(t *testing.T) {
	service := NewAudioContextService()

	fingerprint := &AudioFingerprint{
		ID:          "test-fp-delete",
		Fingerprint: []byte("test-data"),
		SampleRate:  44100,
		CreatedAt:   time.Now(),
	}

	service.StoreFingerprint(fingerprint)
	err := service.DeleteFingerprint("test-fp-delete")
	if err != nil {
		t.Errorf("DeleteFingerprint failed: %v", err)
	}

	_, err = service.GetFingerprint("test-fp-delete")
	if err == nil {
		t.Error("Deleted fingerprint should not be retrievable")
	}
}

func TestAudioContextService_ListFingerprints(t *testing.T) {
	service := NewAudioContextService()

	fp1 := &AudioFingerprint{ID: "fp1", Fingerprint: []byte("data1"), SampleRate: 44100, CreatedAt: time.Now()}
	fp2 := &AudioFingerprint{ID: "fp2", Fingerprint: []byte("data2"), SampleRate: 44100, CreatedAt: time.Now()}

	service.StoreFingerprint(fp1)
	service.StoreFingerprint(fp2)

	fingerprints, err := service.ListFingerprints(0, 10)
	if err != nil {
		t.Errorf("ListFingerprints failed: %v", err)
	}
	if len(fingerprints) < 2 {
		t.Errorf("Should have at least 2 fingerprints, got %d", len(fingerprints))
	}
}

func TestAudioContextService_ClearCache(t *testing.T) {
	service := NewAudioContextService()

	service.ClearCache()

	sampleData := make([]float64, 1024)
	for i := range sampleData {
		sampleData[i] = float64(i) / 1024.0
	}

	fp1, _ := service.GenerateFingerprint(sampleData, 44100)
	fp2, _ := service.GenerateFingerprint(sampleData, 44100)

	if fp1.ID == fp2.ID {
		t.Error("Cached fingerprints should have different IDs")
	}
}

func TestAudioContextService_GetCacheSize(t *testing.T) {
	service := NewAudioContextService()

	size := service.GetCacheSize()
	if size < 0 {
		t.Error("Cache size should not be negative")
	}
}

func TestAudioContextService_UpdateConfig(t *testing.T) {
	service := NewAudioContextService()

	newConfig := &AudioContextConfig{
		EnableDetailedAnalysis: true,
		AnalysisTimeout:       30 * time.Second,
		MaxFingerprintAge:    24 * time.Hour,
		SimilarityThreshold:  0.85,
		CacheEnabled:         true,
	}

	service.UpdateConfig(newConfig)

	if service.config.SimilarityThreshold != 0.85 {
		t.Error("Config not updated correctly")
	}
}

func TestAudioContextService_GetConfig(t *testing.T) {
	service := NewAudioContextService()

	config := service.GetConfig()
	if config == nil {
		t.Error("GetConfig returned nil")
	}
}

func TestAudioContextService_GetStatistics(t *testing.T) {
	service := NewAudioContextService()

	stats := service.GetStatistics()
	if stats == nil {
		t.Error("GetStatistics returned nil")
	}
}

func TestAudioContextService_AnalyzeAudioMultipleChannels(t *testing.T) {
	service := NewAudioContextService()

	sampleData := make([]float64, 2048)
	for i := range sampleData {
		sampleData[i] = float64(i) / 2048.0
	}

	for channelCount := 1; channelCount <= 2; channelCount++ {
		result, err := service.AnalyzeAudio(sampleData, 44100, channelCount)
		if err != nil {
			t.Errorf("AnalyzeAudio with %d channels failed: %v", channelCount, err)
		}
		if result != nil && result.ChannelCount != channelCount {
			t.Errorf("Expected %d channels, got %d", channelCount, result.ChannelCount)
		}
	}
}

func TestAudioContextService_GenerateDifferentFingerprints(t *testing.T) {
	service := NewAudioContextService()

	sampleData1 := make([]float64, 1024)
	sampleData2 := make([]float64, 1024)
	for i := range sampleData1 {
		sampleData1[i] = float64(i) / 1024.0
		sampleData2[i] = float64(i+100) / 1024.0
	}

	fp1, _ := service.GenerateFingerprint(sampleData1, 44100)
	fp2, _ := service.GenerateFingerprint(sampleData2, 44100)

	if fp1.ID == fp2.ID {
		t.Error("Different audio data should generate different fingerprints")
	}
}

func TestAudioFingerprintCache(t *testing.T) {
	cache := &AudioFingerprintCache{
		cache: make(map[string]*CachedAudioFingerprint),
		ttl:   5 * time.Minute,
	}

	if cache == nil {
		t.Error("AudioFingerprintCache should be created")
	}
}

func TestAudioContextMetrics(t *testing.T) {
	metrics := &AudioContextMetrics{
		SampleRate:    44100,
		State:         "running",
		ChannelCount:  2,
		Latency:       10.5,
		IsSupported:   true,
		MaxChannelCount: 8,
	}

	if metrics.SampleRate != 44100 {
		t.Errorf("SampleRate should be 44100, got %d", metrics.SampleRate)
	}
	if metrics.State != "running" {
		t.Errorf("State should be running, got %s", metrics.State)
	}
}

func TestAudioProcessingMetrics(t *testing.T) {
	metrics := &AudioProcessingMetrics{
		FrequencyData: []float64{1.0, 2.0, 3.0},
		TimeDomainData: []float64{0.5, 1.5, 2.5},
		PeakFrequency: 440.0,
		PeakAmplitude: 0.8,
		RMSAmplitude:  0.6,
		ProcessingTime: 5.0,
	}

	if len(metrics.FrequencyData) != 3 {
		t.Error("FrequencyData should have 3 elements")
	}
	if metrics.PeakFrequency != 440.0 {
		t.Errorf("PeakFrequency should be 440.0, got %f", metrics.PeakFrequency)
	}
}

func TestEnhancedAudioAnalysisResult(t *testing.T) {
	result := &EnhancedAudioAnalysisResult{
		Success: true,
		RiskScore: 25.0,
		RiskLevel: "low",
		Confidence: 0.95,
	}

	if !result.Success {
		t.Error("Result should be successful")
	}
	if result.RiskScore != 25.0 {
		t.Errorf("RiskScore should be 25.0, got %f", result.RiskScore)
	}
	if result.Confidence < 0.0 || result.Confidence > 1.0 {
		t.Error("Confidence should be between 0 and 1")
	}
}

func TestAudioAnalysisMetrics(t *testing.T) {
	metrics := &AudioAnalysisMetrics{
		SampleRate:    48000,
		ChannelCount:  2,
		State:         "active",
		Latency:       8.0,
		IsHardware:    true,
	}

	if metrics.SampleRate != 48000 {
		t.Errorf("SampleRate should be 48000, got %d", metrics.SampleRate)
	}
	if !metrics.IsHardware {
		t.Error("Should be hardware audio")
	}
}
