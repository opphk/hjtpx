package service

import (
	"testing"
)

func TestNewAudioContextService(t *testing.T) {
	service := NewAudioContextService()
	if service == nil {
		t.Error("NewAudioContextService 返回了 nil")
	}
	if service.database == nil {
		t.Error("database 未初始化")
	}
	if service.config == nil {
		t.Error("config 未初始化")
	}
	if service.analyzer == nil {
		t.Error("analyzer 未初始化")
	}
	if service.similarityCalculator == nil {
		t.Error("similarityCalculator 未初始化")
	}
	if service.cache == nil {
		t.Error("cache 未初始化")
	}
}

func TestGenerateFingerprint(t *testing.T) {
	service := NewAudioContextService()

	data := map[string]interface{}{
		"sample_rate":                float64(44100),
		"state":                      "running",
		"channel_count":               float64(2),
		"number_of_inputs":            float64(0),
		"number_of_outputs":           float64(1),
		"channel_count_mode":          "max",
		"channel_interpretation":      "speakers",
		"latency_hint":                "interactive",
		"base_latency":                float64(0.02),
		"output_timestamp":            float64(0),
		"current_time":                float64(0),
		"oscillator_type":             "sine",
		"oscillator_frequency":        float64(440),
		"oscillator_detune":           float64(0),
		"gain_value":                  float64(1.0),
		"fft_size":                    float64(2048),
		"frequency_bin_count":         float64(1024),
		"min_decibels":                float64(-100),
		"max_decibels":                float64(-30),
		"smoothing_time_constant":    float64(0.8),
		"frequency_data":              []interface{}{float64(0.1), float64(0.2), float64(0.3)},
		"time_domain_data":            []interface{}{float64(0.5), float64(-0.3), float64(0.8)},
		"peak_frequency":              float64(440),
		"peak_amplitude":              float64(0.9),
		"rms_amplitude":               float64(0.6),
		"spectral_centroid":           float64(2000),
		"spectral_flatness":           float64(0.5),
		"zero_crossing_rate":          float64(0.1),
		"total_harmonic_distortion":   float64(0.01),
		"is_audio_context_supported":  true,
		"is_oscillator_supported":     true,
		"is_analyser_supported":       true,
		"is_gain_node_supported":       true,
		"is_stereo_panner_supported":   true,
		"max_channel_count":           float64(32),
		"rendering_consistency":       float64(0.95),
		"noise_level":                 float64(0.001),
		"is_software_renderer":         false,
		"is_hardware_accelerated":     true,
		"browser_audio_api_version":    "1.0",
		"analysis_duration":            float64(0.1),
	}

	fingerprint, err := service.GenerateFingerprint(data)
	if err != nil {
		t.Errorf("生成指纹失败: %v", err)
	}
	if fingerprint == nil {
		t.Error("指纹不应为 nil")
	}
	if fingerprint.FingerprintID == "" {
		t.Error("指纹ID不应为空")
	}
	if fingerprint.AudioHash == "" {
		t.Error("音频Hash不应为空")
	}
	if fingerprint.ContextProperties.SampleRate != 44100 {
		t.Errorf("采样率不匹配: 期望 %d, 实际 %d", 44100, fingerprint.ContextProperties.SampleRate)
	}
	if fingerprint.ContextProperties.ChannelCount != 2 {
		t.Errorf("通道数不匹配: 期望 %d, 实际 %d", 2, fingerprint.ContextProperties.ChannelCount)
	}
}

func TestGenerateFingerprintWithNilData(t *testing.T) {
	service := NewAudioContextService()

	fingerprint, err := service.GenerateFingerprint(nil)
	if err != nil {
		t.Errorf("nil数据不应返回错误: %v", err)
	}
	if fingerprint == nil {
		t.Error("指纹不应为 nil")
	}
}

func TestGenerateFingerprintWithEmptyData(t *testing.T) {
	service := NewAudioContextService()

	data := map[string]interface{}{}

	fingerprint, err := service.GenerateFingerprint(data)
	if err != nil {
		t.Errorf("空数据不应返回错误: %v", err)
	}
	if fingerprint == nil {
		t.Error("指纹不应为 nil")
	}
}

func TestAnalyzeFingerprint(t *testing.T) {
	service := NewAudioContextService()

	data := map[string]interface{}{
		"sample_rate":               float64(44100),
		"channel_count":              float64(2),
		"channel_count_mode":         "max",
		"oscillator_type":            "sine",
		"fft_size":                   float64(2048),
		"frequency_data":            []interface{}{float64(0.1), float64(0.2), float64(0.3)},
		"time_domain_data":          []interface{}{float64(0.5), float64(-0.3), float64(0.8)},
		"peak_frequency":            float64(440),
		"peak_amplitude":            float64(0.9),
		"rms_amplitude":             float64(0.6),
		"spectral_centroid":         float64(2000),
		"spectral_flatness":         float64(0.5),
		"rendering_consistency":     float64(0.95),
		"is_audio_context_supported": true,
		"is_hardware_accelerated":   true,
	}

	fingerprint, _ := service.GenerateFingerprint(data)

	analysis, err := service.AnalyzeFingerprint(fingerprint.FingerprintID)
	if err != nil {
		t.Errorf("分析指纹失败: %v", err)
	}
	if analysis == nil {
		t.Error("分析结果不应为 nil")
	}
	if analysis.FingerprintID != fingerprint.FingerprintID {
		t.Errorf("指纹ID不匹配: 期望 %s, 实际 %s", fingerprint.FingerprintID, analysis.FingerprintID)
	}
}

func TestAnalyzeFingerprintNotFound(t *testing.T) {
	service := NewAudioContextService()

	_, err := service.AnalyzeFingerprint("nonexistent_id")
	if err == nil {
		t.Error("应该返回错误对于不存在的指纹")
	}
}

func TestCompareFingerprints(t *testing.T) {
	service := NewAudioContextService()

	data1 := map[string]interface{}{
		"sample_rate":               float64(44100),
		"channel_count":              float64(2),
		"channel_count_mode":         "max",
		"oscillator_type":            "sine",
		"fft_size":                   float64(2048),
		"frequency_data":            []interface{}{float64(0.1), float64(0.2), float64(0.3)},
		"peak_frequency":            float64(440),
		"rms_amplitude":             float64(0.6),
		"spectral_centroid":         float64(2000),
		"rendering_consistency":     float64(0.95),
		"is_audio_context_supported": true,
	}

	data2 := map[string]interface{}{
		"sample_rate":               float64(44100),
		"channel_count":              float64(2),
		"channel_count_mode":         "max",
		"oscillator_type":            "sine",
		"fft_size":                   float64(2048),
		"frequency_data":            []interface{}{float64(0.1), float64(0.2), float64(0.3)},
		"peak_frequency":            float64(440),
		"rms_amplitude":             float64(0.6),
		"spectral_centroid":         float64(2000),
		"rendering_consistency":     float64(0.95),
		"is_audio_context_supported": true,
	}

	fp1, _ := service.GenerateFingerprint(data1)
	fp2, _ := service.GenerateFingerprint(data2)

	comparison, err := service.CompareFingerprints(fp1.FingerprintID, fp2.FingerprintID)
	if err != nil {
		t.Errorf("比较指纹失败: %v", err)
	}
	if comparison == nil {
		t.Error("比较结果不应为 nil")
	}
	if comparison.ContextMatchScore == 0 && comparison.ProcessingMatchScore == 0 {
		t.Error("匹配分数不应都为0")
	}
}

func TestCompareFingerprintsNotFound(t *testing.T) {
	service := NewAudioContextService()

	data := map[string]interface{}{
		"sample_rate": float64(44100),
	}

	fp, _ := service.GenerateFingerprint(data)

	_, err := service.CompareFingerprints(fp.FingerprintID, "nonexistent_id")
	if err == nil {
		t.Error("应该返回错误对于不存在的指纹")
	}

	_, err = service.CompareFingerprints("nonexistent_id", fp.FingerprintID)
	if err == nil {
		t.Error("应该返回错误对于不存在的指纹")
	}
}

func TestDetectAnomalies(t *testing.T) {
	service := NewAudioContextService()

	data := map[string]interface{}{
		"sample_rate":                 float64(44100),
		"channel_count":                float64(2),
		"oscillator_type":             "sine",
		"fft_size":                     float64(2048),
		"frequency_data":              []interface{}{float64(0.1), float64(0.2), float64(0.3)},
		"time_domain_data":            []interface{}{float64(0.5), float64(-0.3), float64(0.8)},
		"peak_frequency":              float64(440),
		"peak_amplitude":              float64(0.9),
		"rms_amplitude":               float64(0.6),
		"spectral_centroid":           float64(2000),
		"spectral_flatness":           float64(0.5),
		"zero_crossing_rate":          float64(0.1),
		"rendering_consistency":       float64(0.95),
		"is_audio_context_supported":   true,
		"max_channel_count":           float64(32),
	}

	fingerprint, _ := service.GenerateFingerprint(data)

	detection, err := service.DetectAnomalies(fingerprint.FingerprintID)
	if err != nil {
		t.Errorf("检测异常失败: %v", err)
	}
	if detection == nil {
		t.Error("检测结果不应为 nil")
	}
	if detection.Fingerprint == nil {
		t.Error("指纹信息不应为 nil")
	}
}

func TestDetectAnomaliesNotFound(t *testing.T) {
	service := NewAudioContextService()

	_, err := service.DetectAnomalies("nonexistent_id")
	if err == nil {
		t.Error("应该返回错误对于不存在的指纹")
	}
}

func TestGetSimilarFingerprints(t *testing.T) {
	service := NewAudioContextService()

	data := map[string]interface{}{
		"sample_rate":               float64(44100),
		"channel_count":              float64(2),
		"oscillator_type":            "sine",
		"fft_size":                   float64(2048),
		"frequency_data":            []interface{}{float64(0.1), float64(0.2), float64(0.3)},
		"rendering_consistency":     float64(0.95),
		"is_audio_context_supported": true,
	}

	fp1, _ := service.GenerateFingerprint(data)
	fp2, _ := service.GenerateFingerprint(data)

	similar := service.GetSimilarFingerprints(fp1.FingerprintID, 50.0)
	if similar == nil {
		t.Error("相似指纹列表不应为 nil")
	}
	if len(similar) == 0 {
		t.Log("警告: 未找到相似指纹（可能正常）")
	}

	_ = fp2
}

func TestValidateFingerprint(t *testing.T) {
	service := NewAudioContextService()

	data := map[string]interface{}{
		"sample_rate":               float64(44100),
		"channel_count":              float64(2),
		"frequency_data":            []interface{}{float64(0.1), float64(0.2), float64(0.3)},
		"peak_frequency":            float64(440),
		"is_audio_context_supported": true,
	}

	fingerprint, _ := service.GenerateFingerprint(data)

	valid, issues := service.ValidateFingerprint(fingerprint.FingerprintID)
	if !valid {
		t.Errorf("有效指纹应该通过验证: %v", issues)
	}
}

func TestValidateFingerprintNotFound(t *testing.T) {
	service := NewAudioContextService()

	valid, issues := service.ValidateFingerprint("nonexistent_id")
	if valid {
		t.Error("不存在的指纹应该验证失败")
	}
	if len(issues) == 0 {
		t.Error("应该返回错误信息")
	}
}

func TestGetFingerprint(t *testing.T) {
	service := NewAudioContextService()

	data := map[string]interface{}{
		"sample_rate": float64(44100),
	}

	fingerprint, _ := service.GenerateFingerprint(data)

	retrieved, exists := service.GetFingerprint(fingerprint.FingerprintID)
	if !exists {
		t.Error("应该能获取已存在的指纹")
	}
	if retrieved.FingerprintID != fingerprint.FingerprintID {
		t.Errorf("指纹ID不匹配: 期望 %s, 实际 %s", fingerprint.FingerprintID, retrieved.FingerprintID)
	}
}

func TestGetFingerprintNotFound(t *testing.T) {
	service := NewAudioContextService()

	_, exists := service.GetFingerprint("nonexistent_id")
	if exists {
		t.Error("不应该存在不存在的指纹")
	}
}

func TestGetAllFingerprints(t *testing.T) {
	service := NewAudioContextService()

	initialCount := len(service.GetAllFingerprints())

	data := map[string]interface{}{
		"sample_rate": float64(44100),
	}

	service.GenerateFingerprint(data)
	service.GenerateFingerprint(data)

	count := len(service.GetAllFingerprints())
	if count != initialCount+2 {
		t.Errorf("指纹数量不匹配: 期望 %d, 实际 %d", initialCount+2, count)
	}
}

func TestRemoveFingerprint(t *testing.T) {
	service := NewAudioContextService()

	data := map[string]interface{}{
		"sample_rate": float64(44100),
	}

	fingerprint, _ := service.GenerateFingerprint(data)
	fpID := fingerprint.FingerprintID

	service.RemoveFingerprint(fpID)

	_, exists := service.GetFingerprint(fpID)
	if exists {
		t.Error("指纹应该已被删除")
	}
}

func TestGetStatistics(t *testing.T) {
	service := NewAudioContextService()

	data := map[string]interface{}{
		"sample_rate":               float64(44100),
		"channel_count":              float64(2),
		"frequency_data":            []interface{}{float64(0.1), float64(0.2), float64(0.3)},
		"rendering_consistency":     float64(0.95),
		"is_audio_context_supported": true,
		"is_hardware_accelerated":   true,
	}

	service.GenerateFingerprint(data)
	service.GenerateFingerprint(data)

	stats := service.GetStatistics()
	if stats == nil {
		t.Error("统计数据不应为 nil")
	}
	if stats.TotalFingerprints < 2 {
		t.Errorf("指纹总数应该 >= 2, 实际 %d", stats.TotalFingerprints)
	}
}

func TestExportFingerprints(t *testing.T) {
	service := NewAudioContextService()

	data := map[string]interface{}{
		"sample_rate": float64(44100),
	}

	service.GenerateFingerprint(data)

	exportData, err := service.ExportFingerprints()
	if err != nil {
		t.Errorf("导出指纹失败: %v", err)
	}
	if len(exportData) == 0 {
		t.Error("导出的数据不应为空")
	}
}

func TestImportFingerprints(t *testing.T) {
	service := NewAudioContextService()

	exportData, _ := service.ExportFingerprints()

	newService := NewAudioContextService()
	err := newService.ImportFingerprints(exportData)
	if err != nil {
		t.Errorf("导入指纹失败: %v", err)
	}
}

func TestExtractAudioFeatures(t *testing.T) {
	service := NewAudioContextService()

	data := map[string]interface{}{
		"frequency_data":    []interface{}{float64(0.1), float64(0.5), float64(0.3), float64(0.8), float64(0.2)},
		"time_domain_data":  []interface{}{float64(0.5), float64(-0.3), float64(0.8), float64(-0.2), float64(0.6)},
		"processing_time":   float64(0.05),
	}

	features := service.ExtractAudioFeatures(data)
	if features == nil {
		t.Error("音频特征不应为 nil")
	}
	if len(features.FrequencyData) != 5 {
		t.Errorf("频率数据长度不匹配: 期望 %d, 实际 %d", 5, len(features.FrequencyData))
	}
	if len(features.TimeDomainData) != 5 {
		t.Errorf("时域数据长度不匹配: 期望 %d, 实际 %d", 5, len(features.TimeDomainData))
	}
	if features.ProcessingTime != 0.05 {
		t.Errorf("处理时间不匹配: 期望 %f, 实际 %f", 0.05, features.ProcessingTime)
	}
}

func TestDetectAudioSpoofing(t *testing.T) {
	service := NewAudioContextService()

	data := map[string]interface{}{
		"sample_rate":               float64(44100),
		"channel_count":              float64(2),
		"frequency_data":            []interface{}{float64(0.1), float64(0.2), float64(0.3)},
		"spectral_flatness":          float64(0.001),
		"rendering_consistency":     float64(0.9999),
		"is_audio_context_supported": true,
	}

	detection := service.DetectAudioSpoofing(data)
	if detection == nil {
		t.Error("欺骗检测结果不应为 nil")
	}
	if detection.Fingerprint == nil {
		t.Error("指纹信息不应为 nil")
	}
}

func TestGetAudioContextMetrics(t *testing.T) {
	service := NewAudioContextService()

	data := map[string]interface{}{
		"sample_rate":          float64(44100),
		"state":                "running",
		"channel_count":        float64(2),
		"latency":              float64(0.02),
		"is_supported":         true,
		"max_channel_count":    float64(32),
		"output_timestamp":     float64(1.5),
		"current_time":         float64(0.5),
	}

	metrics := service.GetAudioContextMetrics(data)
	if metrics == nil {
		t.Error("音频上下文指标不应为 nil")
	}
	if metrics.SampleRate != 44100 {
		t.Errorf("采样率不匹配: 期望 %d, 实际 %d", 44100, metrics.SampleRate)
	}
	if metrics.State != "running" {
		t.Errorf("状态不匹配: 期望 %s, 实际 %s", "running", metrics.State)
	}
	if metrics.ChannelCount != 2 {
		t.Errorf("通道数不匹配: 期望 %d, 实际 %d", 2, metrics.ChannelCount)
	}
}

func TestAnalyzeAudioRendering(t *testing.T) {
	service := NewAudioContextService()

	data := map[string]interface{}{
		"rendering_mode":      "interactive",
		"latency":             float64(0.02),
		"consistency":         float64(0.95),
		"hardware_accelerated": true,
	}

	analysis := service.AnalyzeAudioRendering(data)
	if analysis == nil {
		t.Error("音频渲染分析不应为 nil")
	}
	if analysis.RenderingMode != "interactive" {
		t.Errorf("渲染模式不匹配: 期望 %s, 实际 %s", "interactive", analysis.RenderingMode)
	}
	if analysis.Latency != 0.02 {
		t.Errorf("延迟不匹配: 期望 %f, 实际 %f", 0.02, analysis.Latency)
	}
}

func TestCompareAudioRendering(t *testing.T) {
	service := NewAudioContextService()

	data1 := map[string]interface{}{
		"sample_rate":       float64(44100),
		"rendering_mode":    "interactive",
		"latency":           float64(0.02),
		"consistency":       float64(0.95),
		"hardware_accelerated": true,
	}

	data2 := map[string]interface{}{
		"sample_rate":       float64(44100),
		"rendering_mode":    "interactive",
		"latency":           float64(0.025),
		"consistency":       float64(0.94),
		"hardware_accelerated": true,
	}

	fp1, _ := service.GenerateFingerprint(data1)
	fp2, _ := service.GenerateFingerprint(data2)

	comparison := service.CompareAudioRendering(fp1.FingerprintID, fp2.FingerprintID)
	if comparison == nil {
		t.Error("渲染比较结果不应为 nil")
	}
	if !comparison.RenderingModeMatch {
		t.Error("渲染模式应该匹配")
	}
	if !comparison.HardwareMatch {
		t.Error("硬件加速应该匹配")
	}
}

func TestGenerateAudioReport(t *testing.T) {
	service := NewAudioContextService()

	data := map[string]interface{}{
		"sample_rate":               float64(44100),
		"channel_count":              float64(2),
		"oscillator_type":            "sine",
		"fft_size":                   float64(2048),
		"frequency_data":            []interface{}{float64(0.1), float64(0.2), float64(0.3)},
		"rendering_consistency":     float64(0.95),
		"is_audio_context_supported": true,
		"is_oscillator_supported":    true,
		"is_analyser_supported":       true,
		"is_gain_node_supported":     true,
	}

	fingerprint, _ := service.GenerateFingerprint(data)

	report, err := service.GenerateAudioReport(fingerprint.FingerprintID)
	if err != nil {
		t.Errorf("生成报告失败: %v", err)
	}
	if report == "" {
		t.Error("报告不应为空")
	}
}

func TestGenerateAudioReportNotFound(t *testing.T) {
	service := NewAudioContextService()

	_, err := service.GenerateAudioReport("nonexistent_id")
	if err == nil {
		t.Error("应该返回错误对于不存在的指纹")
	}
}

func TestValidateAudioContextSupport(t *testing.T) {
	service := NewAudioContextService()

	data := map[string]interface{}{
		"is_audio_context_supported":   true,
		"is_oscillator_supported":      true,
		"is_analyser_supported":         true,
		"is_gain_node_supported":        true,
		"is_stereo_panner_supported":    true,
	}

	validation := service.ValidateAudioContextSupport(data)
	if validation == nil {
		t.Error("验证结果不应为 nil")
	}
	if !validation.Supported {
		t.Error("应该支持音频")
	}
	if len(validation.Checks) != 5 {
		t.Errorf("检查数量不匹配: 期望 %d, 实际 %d", 5, len(validation.Checks))
	}
	if validation.SupportScore != 100 {
		t.Errorf("支持分数应该为 100, 实际 %f", validation.SupportScore)
	}
}

func TestDetectAudioFingerprintingPatterns(t *testing.T) {
	service := NewAudioContextService()

	data := map[string]interface{}{
		"patterns": []interface{}{"pattern1", "pattern2", "pattern3", "pattern4", "pattern5", "pattern6"},
	}

	pattern := service.DetectAudioFingerprintingPatterns(data)
	if pattern == nil {
		t.Error("指纹模式检测结果不应为 nil")
	}
	if !pattern.IsSuspicious {
		t.Error("应该被标记为可疑")
	}
	if pattern.RiskLevel != "high" {
		t.Errorf("风险等级应该为 high, 实际 %s", pattern.RiskLevel)
	}
}

func TestAnalyzeAudioFrequencySpectrum(t *testing.T) {
	service := NewAudioContextService()

	freqData := make([]interface{}, 100)
	for i := 0; i < 100; i++ {
		freqData[i] = float64(i % 10) * 0.1
	}

	data := map[string]interface{}{
		"frequency_data": freqData,
	}

	spectrum := service.AnalyzeAudioFrequencySpectrum(data)
	if spectrum == nil {
		t.Error("频谱分析结果不应为 nil")
	}
	if len(spectrum.Bands) == 0 {
		t.Error("频带数量不应为0")
	}
}

func TestMatchAudioFingerprints(t *testing.T) {
	service := NewAudioContextService()

	data := map[string]interface{}{
		"sample_rate":               float64(44100),
		"channel_count":              float64(2),
		"oscillator_type":            "sine",
		"fft_size":                   float64(2048),
		"frequency_data":            []interface{}{float64(0.1), float64(0.2), float64(0.3)},
		"rendering_consistency":     float64(0.95),
		"is_audio_context_supported": true,
	}

	fp1, _ := service.GenerateFingerprint(data)
	fp2, _ := service.GenerateFingerprint(data)

	similarity := service.MatchAudioFingerprints(fp1.FingerprintID, fp2.FingerprintID)
	if similarity < 0 {
		t.Errorf("相似度不应为负数: %f", similarity)
	}
}

func TestDetectAudioAnomalies(t *testing.T) {
	service := NewAudioContextService()

	data := map[string]interface{}{
		"sample_rate":                 float64(44100),
		"channel_count":                float64(2),
		"frequency_data":              []interface{}{float64(0.1), float64(0.2), float64(0.3)},
		"spectral_flatness":           float64(0.001),
		"rendering_consistency":       float64(0.9999),
		"is_audio_context_supported":   true,
		"max_channel_count":           float64(32),
	}

	fingerprint, _ := service.GenerateFingerprint(data)

	result := service.DetectAudioAnomalies(fingerprint.FingerprintID)
	if result == nil {
		t.Error("异常检测结果不应为 nil")
	}
	if result.FingerprintID != fingerprint.FingerprintID {
		t.Errorf("指纹ID不匹配: 期望 %s, 实际 %s", fingerprint.FingerprintID, result.FingerprintID)
	}
}

func TestCleanupOldFingerprints(t *testing.T) {
	service := NewAudioContextService()

	data := map[string]interface{}{
		"sample_rate": float64(44100),
	}

	fp, _ := service.GenerateFingerprint(data)
	_ = fp

	removed := service.CleanupOldFingerprints(0)
	if removed < 1 {
		t.Errorf("应该清理至少1个指纹, 实际 %d", removed)
	}
}

func TestGetFingerprintCount(t *testing.T) {
	service := NewAudioContextService()

	initialCount := service.GetFingerprintCount()

	service.GenerateFingerprint(map[string]interface{}{"sample_rate": float64(44100)})
	service.GenerateFingerprint(map[string]interface{}{"sample_rate": float64(48000)})

	count := service.GetFingerprintCount()
	if count != initialCount+2 {
		t.Errorf("指纹数量不匹配: 期望 %d, 实际 %d", initialCount+2, count)
	}
}

func TestClearCache(t *testing.T) {
	service := NewAudioContextService()

	data := map[string]interface{}{
		"sample_rate": float64(44100),
	}

	fp, _ := service.GenerateFingerprint(data)
	_ = service.cache.Get(fp.FingerprintID)

	service.ClearCache()

	_, exists := service.cache.Get(fp.FingerprintID)
	if exists {
		t.Error("缓存应该被清空")
	}
}

func TestSetAndGetConfig(t *testing.T) {
	service := NewAudioContextService()

	newConfig := &AudioContextConfig{
		EnableDetailedAnalysis: false,
		AnalysisTimeout:        10 * time.Second,
		MaxFingerprintAge:     48 * time.Hour,
		SimilarityThreshold:    80.0,
		CacheEnabled:           false,
	}

	service.SetConfig(newConfig)

	retrievedConfig := service.GetConfig()
	if retrievedConfig.EnableDetailedAnalysis != false {
		t.Error("EnableDetailedAnalysis 配置不匹配")
	}
	if retrievedConfig.AnalysisTimeout != 10*time.Second {
		t.Error("AnalysisTimeout 配置不匹配")
	}
	if retrievedConfig.SimilarityThreshold != 80.0 {
		t.Error("SimilarityThreshold 配置不匹配")
	}
	if retrievedConfig.CacheEnabled != false {
		t.Error("CacheEnabled 配置不匹配")
	}
}

func TestAudioFingerprintDB(t *testing.T) {
	db := NewAudioFingerprintDB()
	if db == nil {
		t.Error("AudioFingerprintDB 初始化失败")
	}

	fp := &model.AudioFingerprint{
		FingerprintID: "test_id",
		AudioHash:     "test_hash",
	}

	db.Add(fp)

	retrieved, exists := db.Get("test_id")
	if !exists {
		t.Error("应该能获取已添加的指纹")
	}
	if retrieved.FingerprintID != "test_id" {
		t.Error("指纹ID不匹配")
	}

	allFps := db.GetAll()
	if len(allFps) != 1 {
		t.Errorf("指纹数量不匹配: 期望 %d, 实际 %d", 1, len(allFps))
	}

	db.Remove("test_id")
	_, exists = db.Get("test_id")
	if exists {
		t.Error("指纹应该已被删除")
	}
}

func TestAudioFingerprintAnalyzer(t *testing.T) {
	analyzer := NewAudioFingerprintAnalyzer()
	if analyzer == nil {
		t.Error("AudioFingerprintAnalyzer 初始化失败")
	}

	fp := &model.AudioFingerprint{
		FingerprintID: "test_id",
		AudioHash:     "test_hash",
		ContextProperties: model.AudioContextProperty{
			SampleRate:   44100,
			ChannelCount: 2,
		},
		ProcessingData: model.AudioProcessingData{
			PeakFrequency: 440,
			RMSAmplitude:  0.6,
		},
		RenderingConsistency: 0.95,
	}

	analyzer.database.AddFingerprint(fp)

	analysis := analyzer.Analyze(fp)
	if analysis == nil {
		t.Error("分析结果不应为 nil")
	}
	if analysis.FingerprintID != "test_id" {
		t.Error("指纹ID不匹配")
	}
}

func TestAudioSimilarityCalculator(t *testing.T) {
	calculator := NewAudioSimilarityCalculator()
	if calculator == nil {
		t.Error("AudioSimilarityCalculator 初始化失败")
	}

	fp1 := &model.AudioFingerprint{
		FingerprintID: "test_id_1",
		ContextProperties: model.AudioContextProperty{
			SampleRate:   44100,
			ChannelCount: 2,
		},
		ProcessingData: model.AudioProcessingData{
			PeakFrequency:   440,
			RMSAmplitude:    0.6,
			SpectralCentroid: 2000,
		},
	}

	fp2 := &model.AudioFingerprint{
		FingerprintID: "test_id_2",
		ContextProperties: model.AudioContextProperty{
			SampleRate:   44100,
			ChannelCount: 2,
		},
		ProcessingData: model.AudioProcessingData{
			PeakFrequency:   440,
			RMSAmplitude:    0.6,
			SpectralCentroid: 2000,
		},
	}

	result := calculator.Calculate(fp1, fp2)
	if result == nil {
		t.Error("计算结果不应为 nil")
	}
	if result.SimilarityScore < 0 || result.SimilarityScore > 100 {
		t.Errorf("相似度分数应该在 0-100 范围内: %f", result.SimilarityScore)
	}
}

func TestAudioFingerprintCache(t *testing.T) {
	cache := NewAudioFingerprintCache(5 * time.Minute)
	if cache == nil {
		t.Error("AudioFingerprintCache 初始化失败")
	}

	fp := &model.AudioFingerprint{
		FingerprintID: "test_id",
		AudioHash:     "test_hash",
	}

	cache.Store("test_id", fp)

	retrieved, exists := cache.Get("test_id")
	if !exists {
		t.Error("应该能获取已缓存的指纹")
	}
	if retrieved.FingerprintID != "test_id" {
		t.Error("指纹ID不匹配")
	}

	cache.Remove("test_id")
	_, exists = cache.Get("test_id")
	if exists {
		t.Error("指纹应该已被删除")
	}

	cache.Clear()
	if len(cache.cache) != 0 {
		t.Error("缓存应该被清空")
	}
}

func TestCalculatePeakFrequency(t *testing.T) {
	freqData := []float64{0.1, 0.5, 0.3, 0.8, 0.2}
	peakFreq := calculatePeakFrequency(freqData)
	if peakFreq != 3 {
		t.Errorf("峰值频率索引应该为 3, 实际 %f", peakFreq)
	}
}

func TestCalculatePeakAmplitude(t *testing.T) {
	timeData := []float64{0.5, -0.8, 0.3, -0.9, 0.6}
	peakAmp := calculatePeakAmplitude(timeData)
	if peakAmp != 0.9 {
		t.Errorf("峰值幅度应该为 0.9, 实际 %f", peakAmp)
	}
}

func TestCalculateRMSAmplitude(t *testing.T) {
	timeData := []float64{0.5, -0.5, 0.5, -0.5}
	rmsAmp := calculateRMSAmplitude(timeData)
	if rmsAmp != 0.5 {
		t.Errorf("RMS幅度应该为 0.5, 实际 %f", rmsAmp)
	}
}

func TestCalculateSpectralCentroid(t *testing.T) {
	freqData := []float64{1.0, 2.0, 3.0, 4.0, 5.0}
	centroid := calculateSpectralCentroid(freqData)
	expectedCentroid := 3.0
	if centroid != expectedCentroid {
		t.Errorf("频谱质心应该为 %f, 实际 %f", expectedCentroid, centroid)
	}
}

func TestCalculateSpectralFlatness(t *testing.T) {
	freqData := []float64{1.0, 1.0, 1.0, 1.0}
	flatness := calculateSpectralFlatness(freqData)
	if flatness != 1.0 {
		t.Errorf("频谱平整度应该为 1.0 (完全平坦), 实际 %f", flatness)
	}
}

func TestCalculateZeroCrossingRate(t *testing.T) {
	timeData := []float64{1.0, -1.0, 1.0, -1.0, 1.0}
	zcr := calculateZeroCrossingRate(timeData)
	expectedZCR := 1.0
	if zcr != expectedZCR {
		t.Errorf("过零率应该为 %f, 实际 %f", expectedZCR, zcr)
	}
}

func TestGenerateAudioFingerprintID(t *testing.T) {
	id1 := generateAudioFingerprintID()
	id2 := generateAudioFingerprintID()

	if id1 == "" {
		t.Error("生成的ID不应为空")
	}
	if id1 == id2 {
		t.Error("两次生成的ID应该不同")
	}
	if len(id1) < 10 {
		t.Error("ID长度应该至少为10")
	}
}

func TestValidateAudioFingerprintData(t *testing.T) {
	validData := map[string]interface{}{
		"sample_rate": float64(44100),
	}

	err := validateAudioFingerprintData(validData)
	if err != nil {
		t.Error("有效数据不应该返回错误")
	}

	err = validateAudioFingerprintData(nil)
	if err == nil {
		t.Error("nil数据应该返回错误")
	}

	err = validateAudioFingerprintData(map[string]interface{}{})
	if err == nil {
		t.Error("缺少必需字段的数据应该返回错误")
	}
}

func TestIsValidSampleRate(t *testing.T) {
	if !isValidSampleRate(44100) {
		t.Error("44100 应该是有效的采样率")
	}
	if !isValidSampleRate(48000) {
		t.Error("48000 应该是有效的采样率")
	}
	if isValidSampleRate(12345) {
		t.Error("12345 不应该是有效的采样率")
	}
}

func TestIsValidChannelCount(t *testing.T) {
	if !isValidChannelCount(2) {
		t.Error("2 应该是有效的通道数")
	}
	if !isValidChannelCount(32) {
		t.Error("32 应该是有效的通道数")
	}
	if isValidChannelCount(0) {
		t.Error("0 不应该是有效的通道数")
	}
	if isValidChannelCount(33) {
		t.Error("33 不应该是有效的通道数")
	}
}

func TestIsValidFFTSize(t *testing.T) {
	if !isValidFFTSize(2048) {
		t.Error("2048 应该是有效的FFT大小")
	}
	if !isValidFFTSize(4096) {
		t.Error("4096 应该是有效的FFT大小")
	}
	if isValidFFTSize(1000) {
		t.Error("1000 不应该是有效的FFT大小")
	}
}

func TestDetectSuspiciousPatterns(t *testing.T) {
	fp := &model.AudioFingerprint{
		ProcessingData: model.AudioProcessingData{
			SpectralFlatness:    0.001,
			FrequencyData:       []float64{0.1, 0.2, 0.3},
		},
		RenderingConsistency: 0.9999,
	}

	patterns := detectSuspiciousPatterns(fp)
	if len(patterns) < 2 {
		t.Errorf("应该检测到至少2个可疑模式, 实际 %d", len(patterns))
	}
}

func TestCalculateAudioEntropy(t *testing.T) {
	data := []float64{1.0, 2.0, 3.0, 4.0, 5.0, 6.0, 7.0, 8.0}
	entropy := calculateAudioEntropy(data)
	if entropy < 0 || entropy > 1 {
		t.Errorf("熵值应该在 0-1 范围内: %f", entropy)
	}
}

func TestAnalyzeAudioQuality(t *testing.T) {
	fp := &model.AudioFingerprint{
		ContextProperties: model.AudioContextProperty{
			SampleRate:   44100,
			ChannelCount: 2,
		},
		ProcessingData: model.AudioProcessingData{
			FrequencyData: []float64{0.1, 0.2, 0.3},
			RMSAmplitude:  0.6,
		},
		AnalyserConfig: model.AnalyserNodeConfig{
			FFTSize: 2048,
		},
	}

	quality := analyzeAudioQuality(fp)
	if quality < 0 || quality > 100 {
		t.Errorf("质量分数应该在 0-100 范围内: %f", quality)
	}
}

func TestAnalyzeAudioQualityService(t *testing.T) {
	service := NewAudioContextService()

	data := map[string]interface{}{
		"sample_rate":               float64(44100),
		"channel_count":              float64(2),
		"frequency_data":            []interface{}{float64(0.1), float64(0.2), float64(0.3)},
		"time_domain_data":          []interface{}{float64(0.5), float64(-0.3), float64(0.8)},
		"rms_amplitude":             float64(0.6),
		"fft_size":                   float64(2048),
		"spectral_flatness":          float64(0.5),
		"spectral_centroid":         float64(2000),
		"is_audio_context_supported": true,
	}

	fingerprint, _ := service.GenerateFingerprint(data)

	analysis := service.AnalyzeAudioQuality(fingerprint.FingerprintID)
	if analysis == nil {
		t.Error("质量分析结果不应为 nil")
	}
	if analysis.OverallQuality < 0 || analysis.OverallQuality > 100 {
		t.Errorf("总体质量分数应该在 0-100 范围内: %f", analysis.OverallQuality)
	}
}
