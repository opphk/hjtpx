package service

import (
	"testing"
)

func TestNewMemoryOptimizerService(t *testing.T) {
	optimizer := NewMemoryOptimizerService()
	if optimizer == nil {
		t.Error("NewMemoryOptimizerService 返回了 nil")
	}
}

func TestOptimizeMemory(t *testing.T) {
	optimizer := NewMemoryOptimizerService()
	
	err := optimizer.OptimizeMemory()
	if err != nil {
		t.Errorf("优化内存失败: %v", err)
	}
}

func TestGetMemoryUsage(t *testing.T) {
	optimizer := NewMemoryOptimizerService()
	
	usage, err := optimizer.GetMemoryUsage()
	if err != nil {
		t.Errorf("获取内存使用情况失败: %v", err)
	}
	if usage == nil {
		t.Error("内存使用情况不应为 nil")
	}
}

func TestAnalyzeMemoryLeaks(t *testing.T) {
	optimizer := NewMemoryOptimizerService()
	
	analysis, err := optimizer.AnalyzeMemoryLeaks()
	if err != nil {
		t.Errorf("分析内存泄漏失败: %v", err)
	}
	if analysis == nil {
		t.Error("内存泄漏分析结果不应为 nil")
	}
}

func TestClearMemoryCache(t *testing.T) {
	optimizer := NewMemoryOptimizerService()
	
	err := optimizer.ClearMemoryCache()
	if err != nil {
		t.Errorf("清空内存缓存失败: %v", err)
	}
}

func TestSetMemoryLimit(t *testing.T) {
	optimizer := NewMemoryOptimizerService()
	
	err := optimizer.SetMemoryLimit(1024 * 1024 * 1024)
	if err != nil {
		t.Errorf("设置内存限制失败: %v", err)
	}
}

func TestGetMemoryLimit(t *testing.T) {
	optimizer := NewMemoryOptimizerService()
	
	limit, err := optimizer.GetMemoryLimit()
	if err != nil {
		t.Errorf("获取内存限制失败: %v", err)
	}
	if limit <= 0 {
		t.Error("内存限制应该大于 0")
	}
}

func TestMonitorMemory(t *testing.T) {
	optimizer := NewMemoryOptimizerService()
	
	monitoringData, err := optimizer.MonitorMemory()
	if err != nil {
		t.Errorf("监控内存失败: %v", err)
	}
	if monitoringData == nil {
		t.Error("内存监控数据不应为 nil")
	}
}

func TestGetMemoryStats(t *testing.T) {
	optimizer := NewMemoryOptimizerService()
	
	stats, err := optimizer.GetMemoryStats()
	if err != nil {
		t.Errorf("获取内存统计失败: %v", err)
	}
	if stats == nil {
		t.Error("内存统计不应为 nil")
	}
}

func TestTriggerGC(t *testing.T) {
	optimizer := NewMemoryOptimizerService()
	
	err := optimizer.TriggerGC()
	if err != nil {
		t.Errorf("触发垃圾回收失败: %v", err)
	}
}

func TestSetMemoryPressureLevel(t *testing.T) {
	optimizer := NewMemoryOptimizerService()
	
	err := optimizer.SetMemoryPressureLevel("low")
	if err != nil {
		t.Errorf("设置内存压力级别失败: %v", err)
	}
}

func TestGetMemoryPressureLevel(t *testing.T) {
	optimizer := NewMemoryOptimizerService()
	
	level, err := optimizer.GetMemoryPressureLevel()
	if err != nil {
		t.Errorf("获取内存压力级别失败: %v", err)
	}
	if level == "" {
		t.Error("内存压力级别不应为空")
	}
}
