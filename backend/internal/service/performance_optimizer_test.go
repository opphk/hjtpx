package service

import (
	"testing"
)

func TestNewPerformanceOptimizerService(t *testing.T) {
	optimizer := NewPerformanceOptimizerService()
	if optimizer == nil {
		t.Error("NewPerformanceOptimizerService 返回了 nil")
	}
}

func TestOptimizePerformance(t *testing.T) {
	optimizer := NewPerformanceOptimizerService()
	
	err := optimizer.OptimizePerformance()
	if err != nil {
		t.Errorf("优化性能失败: %v", err)
	}
}

func TestGetPerformanceMetrics(t *testing.T) {
	optimizer := NewPerformanceOptimizerService()
	
	metrics, err := optimizer.GetPerformanceMetrics()
	if err != nil {
		t.Errorf("获取性能指标失败: %v", err)
	}
	if metrics == nil {
		t.Error("性能指标不应为 nil")
	}
}

func TestAnalyzePerformance(t *testing.T) {
	optimizer := NewPerformanceOptimizerService()
	
	analysis, err := optimizer.AnalyzePerformance()
	if err != nil {
		t.Errorf("分析性能失败: %v", err)
	}
	if analysis == nil {
		t.Error("性能分析结果不应为 nil")
	}
}

func TestApplyOptimizationStrategy(t *testing.T) {
	optimizer := NewPerformanceOptimizerService()
	
	strategy := map[string]interface{}{
		"cache_enabled":    true,
		"compression_level": 6,
	}
	
	err := optimizer.ApplyOptimizationStrategy(strategy)
	if err != nil {
		t.Errorf("应用优化策略失败: %v", err)
	}
}

func TestResetPerformance(t *testing.T) {
	optimizer := NewPerformanceOptimizerService()
	
	err := optimizer.ResetPerformance()
	if err != nil {
		t.Errorf("重置性能设置失败: %v", err)
	}
}

func TestGetOptimizationRecommendations(t *testing.T) {
	optimizer := NewPerformanceOptimizerService()
	
	recommendations, err := optimizer.GetOptimizationRecommendations()
	if err != nil {
		t.Errorf("获取优化建议失败: %v", err)
	}
	if recommendations == nil {
		t.Error("优化建议列表不应为 nil")
	}
}

func TestMonitorPerformance(t *testing.T) {
	optimizer := NewPerformanceOptimizerService()
	
	monitoringData, err := optimizer.MonitorPerformance()
	if err != nil {
		t.Errorf("监控性能失败: %v", err)
	}
	if monitoringData == nil {
		t.Error("监控数据不应为 nil")
	}
}

func TestExportPerformanceReport(t *testing.T) {
	optimizer := NewPerformanceOptimizerService()
	
	report, err := optimizer.ExportPerformanceReport()
	if err != nil {
		t.Errorf("导出性能报告失败: %v", err)
	}
	if report == "" {
		t.Error("性能报告不应为空")
	}
}

func TestSetPerformanceTarget(t *testing.T) {
	optimizer := NewPerformanceOptimizerService()
	
	targets := map[string]float64{
		"response_time_ms": 100,
		"throughput_rps":    1000,
	}
	
	err := optimizer.SetPerformanceTarget(targets)
	if err != nil {
		t.Errorf("设置性能目标失败: %v", err)
	}
}

func TestGetPerformanceTargets(t *testing.T) {
	optimizer := NewPerformanceOptimizerService()
	
	targets, err := optimizer.GetPerformanceTargets()
	if err != nil {
		t.Errorf("获取性能目标失败: %v", err)
	}
	if targets == nil {
		t.Error("性能目标不应为 nil")
	}
}
