package service

import (
	"testing"
)

func TestVerificationPerformanceOptimizer(t *testing.T) {
	t.Log("测试验证性能优化器...")

	vpo := NewVerificationPerformanceOptimizer()

	vctx := vpo.StartVerification("user-123")
	t.Logf("开始验证 - 用户: %s, 开始时间: %s", vctx.UserID, vctx.StartTime)

	latency := vpo.EndVerification(vctx, true)
	t.Logf("验证完成 - 延迟: %dms", latency)

	vpo.EndVerification(vpo.StartVerification("user-456"), false)
	vpo.EndVerification(vpo.StartVerification("user-789"), true)

	metrics := vpo.GetMetrics()
	t.Logf("性能指标 - 总验证: %d, 成功: %d, 失败: %d",
		metrics.TotalVerifications, metrics.SuccessfulVerifications, metrics.FailedVerifications)

	t.Log("验证性能优化器测试通过")
}

func TestImageOptimization(t *testing.T) {
	t.Log("测试图片优化...")

	vpo := NewVerificationPerformanceOptimizer()

	params := &ImageOptimizationParams{
		OriginalSize: 2048000,
		Format:       "jpeg",
		Width:        1920,
		Height:       1080,
	}

	result, err := vpo.OptimizeImage(params)
	if err != nil {
		t.Errorf("图片优化失败: %v", err)
		return
	}

	t.Logf("图片优化结果 - 原始大小: %d, 优化后: %d, 压缩比: %.2f%%, 加载时间: %dms",
		result.OriginalSize, result.OptimizedSize, result.CompressionRatio*100, result.LoadTimeMs)

	params2 := &ImageOptimizationParams{
		OriginalSize: 5000000,
		Format:       "png",
		Width:        3840,
		Height:       2160,
	}

	result2, _ := vpo.OptimizeImage(params2)
	t.Logf("4K图片优化 - 原始大小: %d, 优化后: %d, 新尺寸: %dx%d",
		result2.OriginalSize, result2.OptimizedSize, result2.Width, result2.Height)

	t.Log("图片优化测试通过")
}

func TestMemoryOptimization(t *testing.T) {
	t.Log("测试内存优化...")

	vpo := NewVerificationPerformanceOptimizer()

	vpo.memOptim.CurrentMemoryMB.Store(256)

	result := vpo.OptimizeMemory("gc")
	t.Logf("GC优化 - 之前: %dMB, 之后: %dMB, 释放: %dMB",
		result.BeforeMemoryMB, result.AfterMemoryMB, result.ReleasedMB)

	result2 := vpo.OptimizeMemory("pool")
	t.Logf("对象池优化 - 释放: %dMB", result2.ReleasedMB)

	result3 := vpo.OptimizeMemory("cache_clear")
	t.Logf("缓存清理 - 释放: %dMB", result3.ReleasedMB)

	t.Log("内存优化测试通过")
}

func TestBatteryOptimization(t *testing.T) {
	t.Log("测试电池优化...")

	vpo := NewVerificationPerformanceOptimizer()

	result1 := vpo.OptimizeForBattery(80, true)
	t.Logf("充电中80%% - 低功耗模式: %v, 节能预估: %d%%, 优化项: %v",
		result1.LowPowerMode, result1.EstimatedPowerSaving, result1.OptimizationsApplied)

	result2 := vpo.OptimizeForBattery(15, false)
	t.Logf("电量15%%未充电 - 低功耗模式: %v, 节能预估: %d%%, 优化项: %v",
		result2.LowPowerMode, result2.EstimatedPowerSaving, result2.OptimizationsApplied)

	result3 := vpo.OptimizeForBattery(40, false)
	t.Logf("电量40%%未充电 - 低功耗模式: %v, 节能预估: %d%%, 优化项: %v",
		result3.LowPowerMode, result3.EstimatedPowerSaving, result3.OptimizationsApplied)

	t.Log("电池优化测试通过")
}

func TestPerformanceProfiles(t *testing.T) {
	t.Log("测试性能配置文件...")

	vpo := NewVerificationPerformanceOptimizer()

	err := vpo.ApplyProfile(ProfileHighPerformance)
	if err != nil {
		t.Errorf("应用高性能配置失败: %v", err)
		return
	}
	t.Log("应用高性能配置成功")

	imgOpt := vpo.GetImageOptimization()
	t.Logf("高性能配置 - 图片质量: %d%%, 压缩级别: %d",
		imgOpt.Quality, imgOpt.CompressionLevel)

	err = vpo.ApplyProfile(ProfilePowerSaving)
	if err != nil {
		t.Errorf("应用节能配置失败: %v", err)
		return
	}

	imgOpt2 := vpo.GetImageOptimization()
	t.Logf("节能配置 - 图片质量: %d%%, 压缩级别: %d",
		imgOpt2.Quality, imgOpt2.CompressionLevel)

	err = vpo.ApplyProfile(ProfileUltraLight)
	if err != nil {
		t.Errorf("应用超轻量配置失败: %v", err)
		return
	}

	imgOpt3 := vpo.GetImageOptimization()
	memOpt := vpo.GetMemoryOptimization()
	t.Logf("超轻量配置 - 图片质量: %d%%, 缓存大小: %dMB",
		imgOpt3.Quality, memOpt.CacheSizeMB)

	t.Log("性能配置文件测试通过")
}

func TestLatencyPrediction(t *testing.T) {
	t.Log("测试延迟预测...")

	vpo := NewVerificationPerformanceOptimizer()

	for i := 0; i < 20; i++ {
		vctx := vpo.StartVerification("user-test")
		vpo.EndVerification(vctx, true)
	}

	predicted := vpo.PredictLatency()
	t.Logf("预测延迟: %dms", predicted)

	metrics := vpo.GetMetrics()
	t.Logf("实际平均延迟: %dms, P50: %dms, P95: %dms, P99: %dms",
		metrics.AverageLatencyMs.Load(),
		metrics.P50LatencyMs.Load(),
		metrics.P95LatencyMs.Load(),
		metrics.P99LatencyMs.Load())

	t.Log("延迟预测测试通过")
}

func TestPerformanceReport(t *testing.T) {
	t.Log("测试性能报告生成...")

	vpo := NewVerificationPerformanceOptimizer()

	for i := 0; i < 5; i++ {
		vctx := vpo.StartVerification("user-report")
		vpo.EndVerification(vctx, i%2 == 0)
	}

	report := vpo.GetPerformanceReport()
	t.Logf("性能报告生成成功")
	t.Logf("验证指标: %v", report["verification_metrics"])
	t.Logf("图片优化: %v", report["image_optimization"])
	t.Logf("内存优化: %v", report["memory_optimization"])
	t.Logf("电池优化: %v", report["battery_optimization"])

	t.Log("性能报告生成测试通过")
}
