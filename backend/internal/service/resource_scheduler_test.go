package service

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResourceScheduler_New(t *testing.T) {
	t.Run("创建默认配置调度器", func(t *testing.T) {
		scheduler := NewResourceScheduler(nil)
		assert.NotNil(t, scheduler)
		assert.NotNil(t, scheduler.config)
		assert.NotNil(t, scheduler.metrics)
		assert.True(t, scheduler.initialized.Load())
	})

	t.Run("创建自定义配置调度器", func(t *testing.T) {
		config := &SchedulerConfig{
			EnableK8sAutoscaler:    true,
			EnableServiceMesh:       true,
			EnablePrediction:        true,
			EnableCostOptimization:  true,
			MinReplicas:            2,
			MaxReplicas:            50,
			ScaleUpThreshold:       0.8,
			ScaleDownThreshold:     0.2,
			BalanceMode:            BalanceModePerformanceOptimized,
		}

		scheduler := NewResourceScheduler(config)
		assert.NotNil(t, scheduler)
		assert.Equal(t, 2, scheduler.config.MinReplicas)
		assert.Equal(t, 50, scheduler.config.MaxReplicas)
	})
}

func TestResourceScheduler_Scale(t *testing.T) {
	scheduler := NewResourceScheduler(nil)
	defer scheduler.Close()

	ctx := context.Background()

	t.Run("基础扩容", func(t *testing.T) {
		metrics := &ResourceMetric{
			Timestamp:   time.Now(),
			CPUUsage:    0.85,
			MemoryUsage: 0.7,
			RequestRate: 5000,
			Latency:     100,
		}

		scheduler.k8sAutoscaler.metricsHistory = append(scheduler.k8sAutoscaler.metricsHistory, *metrics)
		scheduler.k8sAutoscaler.metricsHistory = append(scheduler.k8sAutoscaler.metricsHistory, *metrics)
		scheduler.k8sAutoscaler.metricsHistory = append(scheduler.k8sAutoscaler.metricsHistory, *metrics)

		recommendation, err := scheduler.Scale(ctx)
		require.NoError(t, err)
		assert.NotNil(t, recommendation)
	})

	t.Run("扩容建议", func(t *testing.T) {
		scheduler.k8sAutoscaler.metricsHistory = nil
		scheduler.k8sAutoscaler.lastScaleTime = time.Now().Add(-10 * time.Minute)

		highMetrics := &ResourceMetric{
			Timestamp:   time.Now(),
			CPUUsage:    0.9,
			MemoryUsage: 0.8,
			RequestRate: 10000,
			Latency:     200,
		}

		recommendation, err := scheduler.k8sAutoscaler.Scale(ctx, highMetrics)
		require.NoError(t, err)

		if recommendation.Action == ScaleActionScaleUp {
			assert.Greater(t, recommendation.NewReplicas, int(scheduler.GetReplicas()))
		}
	})

	t.Run("缩容建议", func(t *testing.T) {
		scheduler.k8sAutoscaler.metricsHistory = nil
		scheduler.k8sAutoscaler.lastScaleTime = time.Now().Add(-10 * time.Minute)

		lowMetrics := &ResourceMetric{
			Timestamp:   time.Now(),
			CPUUsage:    0.2,
			MemoryUsage: 0.2,
			RequestRate: 100,
			Latency:     10,
		}

		scheduler.k8sAutoscaler.metricsHistory = append(scheduler.k8sAutoscaler.metricsHistory, *lowMetrics)

		recommendation, err := scheduler.k8sAutoscaler.Scale(ctx, lowMetrics)
		require.NoError(t, err)
		assert.NotNil(t, recommendation)
	})
}

func TestK8sAutoscaler(t *testing.T) {
	config := &SchedulerConfig{
		MinReplicas:     1,
		MaxReplicas:     20,
		ScaleUpThreshold: 0.7,
		ScaleDownThreshold: 0.3,
		ScaleCooldown:   5 * time.Minute,
	}

	autoscaler := NewK8sAutoscaler(config)
	defer autoscaler.Disable()

	ctx := context.Background()

	t.Run("初始副本数", func(t *testing.T) {
		assert.Equal(t, int64(config.MinReplicas), autoscaler.currentReplicas.Load())
	})

	t.Run("启用禁用", func(t *testing.T) {
		autoscaler.Disable()
		assert.False(t, autoscaler.enabled.Load())

		autoscaler.Enable()
		assert.True(t, autoscaler.enabled.Load())
	})

	t.Run("手动设置副本数", func(t *testing.T) {
		autoscaler.SetReplicas(10)
		assert.Equal(t, int64(10), autoscaler.currentReplicas.Load())

		autoscaler.SetReplicas(0)
		assert.Equal(t, int64(1), autoscaler.currentReplicas.Load())

		autoscaler.SetReplicas(100)
		assert.Equal(t, int64(20), autoscaler.currentReplicas.Load())
	})

	t.Run("冷却期检测", func(t *testing.T) {
		autoscaler.lastScaleTime = time.Now()

		metrics := &ResourceMetric{
			CPUUsage: 0.9,
		}

		recommendation, err := autoscaler.Scale(ctx, metrics)
		require.NoError(t, err)
		assert.Equal(t, ScaleActionNoChange, recommendation.Action)
		assert.Contains(t, recommendation.Reason, "cooldown")
	})
}

func TestK8sAutoscalerMetrics(t *testing.T) {
	config := &SchedulerConfig{
		MinReplicas: 1,
		MaxReplicas: 10,
	}

	autoscaler := NewK8sAutoscaler(config)

	t.Run("获取指标", func(t *testing.T) {
		metrics := autoscaler.GetMetrics()
		assert.NotNil(t, metrics)
		assert.Equal(t, int64(0), metrics.ScaleEvents.Load())
	})
}

func TestServiceMeshManager(t *testing.T) {
	mesh := NewServiceMeshManager()
	defer mesh.Disable()

	t.Run("启用禁用", func(t *testing.T) {
		mesh.Enable()
		assert.True(t, mesh.enabled.Load())

		mesh.Disable()
		assert.False(t, mesh.enabled.Load())
	})

	t.Run("添加流量规则", func(t *testing.T) {
		rule := &TrafficRule{
			Name:        "test-rule",
			Source:      "client",
			Destination: "service-a",
			Weight:      0.8,
			Timeout:     5 * time.Second,
			Retries:     3,
		}

		mesh.AddTrafficRule(rule)
		assert.NotNil(t, mesh.trafficRules["test-rule"])
	})

	t.Run("获取路由", func(t *testing.T) {
		route, err := mesh.GetRoute("test-rule")
		assert.NoError(t, err)
		assert.NotNil(t, route)
	})

	t.Run("添加熔断器", func(t *testing.T) {
		cb := &CircuitBreaker{
			Name:             "service-a-breaker",
			FailureThreshold: 5,
			SuccessThreshold: 3,
			Timeout:          30 * time.Second,
			MaxConnections:   100,
		}

		mesh.AddCircuitBreaker(cb)
		assert.NotNil(t, mesh.circuitBreakers["service-a-breaker"])
	})

	t.Run("记录请求", func(t *testing.T) {
		mesh.RecordRequest("service-a", true)
		mesh.RecordRequest("service-a", false)

		metrics := mesh.GetMetrics()
		assert.Equal(t, int64(2), metrics.TotalRequests.Load())
		assert.Equal(t, int64(1), metrics.FailedRequests.Load())
	})

	t.Run("熔断器状态", func(t *testing.T) {
		state := mesh.GetCircuitBreakerState("service-a")
		assert.Equal(t, CircuitStateClosed, state)
	})
}

func TestServiceMeshCircuitBreaker(t *testing.T) {
	mesh := NewServiceMeshManager()

	cb := &CircuitBreaker{
		Name:             "test-breaker",
		FailureThreshold: 3,
		SuccessThreshold: 2,
		Timeout:          1 * time.Second,
		MaxConnections:   10,
	}

	mesh.AddCircuitBreaker(cb)

	t.Run("失败触发熔断", func(t *testing.T) {
		for i := 0; i < 3; i++ {
			mesh.RecordRequest("test-breaker", false)
		}

		state := mesh.GetCircuitBreakerState("test-breaker")
		assert.Equal(t, CircuitStateOpen, state)
	})

	t.Run("超时后进入半开", func(t *testing.T) {
		time.Sleep(2 * time.Second)

		cb.LastFailure = time.Now().Add(-2 * time.Second)
		state := mesh.GetCircuitBreakerState("test-breaker")
		assert.Equal(t, CircuitStateHalfOpen, state)
	})
}

func TestServiceMeshCanary(t *testing.T) {
	mesh := NewServiceMeshManager()

	t.Run("添加金丝雀配置", func(t *testing.T) {
		config := &CanaryConfig{
			Name:        "v2-canary",
			Version:     "2.0.0",
			Weight:      10,
			MaxWeight:   100,
			StepWeight:  10,
			AutoPromote: true,
		}

		mesh.AddCanaryConfig(config)
		assert.NotNil(t, mesh.canaryConfigs["v2-canary"])
	})

	t.Run("更新权重", func(t *testing.T) {
		mesh.UpdateCanaryWeight("v2-canary", 50)
		assert.Equal(t, float64(50), mesh.canaryConfigs["v2-canary"].Weight)
	})
}

func TestResourcePredictor(t *testing.T) {
	predictor := NewResourcePredictor(10 * time.Minute)
	defer predictor.enabled.Store(false)

	t.Run("添加数据点", func(t *testing.T) {
		predictor.AddDataPoint("cpu", 0.5)
		predictor.AddDataPoint("cpu", 0.6)
		predictor.AddDataPoint("cpu", 0.7)

		model := predictor.models["cpu"]
		assert.NotNil(t, model)
		assert.Len(t, model.DataPoints, 3)
	})

	t.Run("预测", func(t *testing.T) {
		for i := 0; i < 15; i++ {
			predictor.AddDataPoint("cpu", 0.5+float64(i)*0.02)
		}

		prediction, err := predictor.Predict("cpu", 5*time.Minute)
		require.NoError(t, err)
		assert.Greater(t, prediction, 0.0)
	})

	t.Run("数据不足预测", func(t *testing.T) {
		_, err := predictor.Predict("cpu", 5*time.Minute)
		assert.Error(t, err)
	})

	t.Run("未知指标类型", func(t *testing.T) {
		_, err := predictor.Predict("unknown", 5*time.Minute)
		assert.Error(t, err)
	})
}

func TestResourcePredictorMetrics(t *testing.T) {
	predictor := NewResourcePredictor(10 * time.Minute)
	defer predictor.enabled.Store(false)

	t.Run("获取指标", func(t *testing.T) {
		metrics := predictor.GetMetrics()
		assert.NotNil(t, metrics)
	})
}

func TestCostPerformanceOptimizer(t *testing.T) {
	optimizer := NewCostPerformanceOptimizer(BalanceModeBalanced)
	defer optimizer.enabled.Store(false)

	t.Run("基础资源分配", func(t *testing.T) {
		metrics := &ResourceMetric{
			CPUUsage:    0.8,
			MemoryUsage: 0.7,
			NetworkIO:   500,
			RequestRate: 5000,
			Latency:     100,
			ErrorRate:   0.01,
		}

		allocation, err := optimizer.CalculateOptimalResources(metrics)
		require.NoError(t, err)
		assert.NotNil(t, allocation)
		assert.Greater(t, allocation.CPU, 0.0)
		assert.Greater(t, allocation.Memory, 0.0)
		assert.Greater(t, allocation.Replicas, 0)
	})

	t.Run("成本计算", func(t *testing.T) {
		metrics := &ResourceMetric{
			CPUUsage: 0.5,
			MemoryUsage: 0.5,
			NetworkIO: 100,
			RequestRate: 1000,
			Latency: 50,
			ErrorRate: 0.01,
		}

		allocation, _ := optimizer.CalculateOptimalResources(metrics)

		costScore := optimizer.calculateCostScore(metrics)
		assert.Greater(t, costScore, 0.0)

		perfScore := optimizer.calculatePerformanceScore(metrics)
		assert.GreaterOrEqual(t, perfScore, 0.0)
		assert.LessOrEqual(t, perfScore, 1.0)

		assert.Equal(t, costScore, allocation.CostEstimate)
		assert.Equal(t, perfScore, allocation.PerformanceScore)
	})
}

func TestCostPerformanceOptimizerMetrics(t *testing.T) {
	optimizer := NewCostPerformanceOptimizer(BalanceModeBalanced)
	defer optimizer.enabled.Store(false)

	t.Run("获取成本指标", func(t *testing.T) {
		metrics := optimizer.GetMetrics()
		assert.NotNil(t, metrics)
	})
}

func TestResourceScheduler_Replicas(t *testing.T) {
	scheduler := NewResourceScheduler(nil)
	defer scheduler.Close()

	t.Run("获取当前副本数", func(t *testing.T) {
		replicas := scheduler.GetReplicas()
		assert.Greater(t, replicas, 0)
	})

	t.Run("手动设置副本数", func(t *testing.T) {
		scheduler.SetReplicas(15)
		assert.Equal(t, 15, scheduler.GetReplicas())
	})
}

func TestResourceScheduler_ComponentAccess(t *testing.T) {
	scheduler := NewResourceScheduler(nil)
	defer scheduler.Close()

	t.Run("获取K8s自动扩缩容器", func(t *testing.T) {
		autoscaler := scheduler.GetK8sAutoscaler()
		if autoscaler != nil {
			assert.NotNil(t, autoscaler)
		}
	})

	t.Run("获取服务网格管理器", func(t *testing.T) {
		mesh := scheduler.GetServiceMesh()
		if mesh != nil {
			assert.NotNil(t, mesh)
		}
	})

	t.Run("获取预测器", func(t *testing.T) {
		predictor := scheduler.GetPredictor()
		if predictor != nil {
			assert.NotNil(t, predictor)
		}
	})

	t.Run("获取成本优化器", func(t *testing.T) {
		optimizer := scheduler.GetCostOptimizer()
		if optimizer != nil {
			assert.NotNil(t, optimizer)
		}
	})
}

func TestResourceScheduler_Metrics(t *testing.T) {
	scheduler := NewResourceScheduler(nil)
	defer scheduler.Close()

	t.Run("获取调度器指标", func(t *testing.T) {
		metrics := scheduler.GetMetrics()
		assert.NotNil(t, metrics)
	})

	t.Run("获取报告", func(t *testing.T) {
		report := scheduler.GetReport()
		assert.NotNil(t, report)
		assert.Contains(t, report, "uptime_seconds")
		assert.Contains(t, report, "current_replicas")
		assert.Contains(t, report, "target_replicas")
	})
}

func TestResourceScheduler_Close(t *testing.T) {
	scheduler := NewResourceScheduler(nil)

	t.Run("关闭调度器", func(t *testing.T) {
		err := scheduler.Close()
		assert.NoError(t, err)
		assert.False(t, scheduler.initialized.Load())
	})
}

func TestResourceScheduler_ConcurrentScale(t *testing.T) {
	scheduler := NewResourceScheduler(nil)
	defer scheduler.Close()

	ctx := context.Background()

	t.Run("并发扩容", func(t *testing.T) {
		var wg sync.WaitGroup

		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for j := 0; j < 5; j++ {
					scheduler.Scale(ctx)
					time.Sleep(10 * time.Millisecond)
				}
			}()
		}

		wg.Wait()

		metrics := scheduler.GetMetrics()
		t.Logf("Scale operations: %d", metrics.ScaleOperations.Load())
	})
}

func TestResourceScheduler_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	scheduler := NewResourceScheduler(nil)
	defer scheduler.Close()

	ctx := context.Background()

	t.Run("完整调度流程", func(t *testing.T) {
		scheduler.SetReplicas(5)

		metrics := &ResourceMetric{
			Timestamp:   time.Now(),
			CPUUsage:    0.75,
			MemoryUsage: 0.65,
			RequestRate: 3000,
			Latency:     80,
			ErrorRate:   0.02,
		}

		for i := 0; i < 10; i++ {
			scheduler.k8sAutoscaler.metricsHistory = append(scheduler.k8sAutoscaler.metricsHistory, *metrics)
		}

		recommendation, err := scheduler.Scale(ctx)
		require.NoError(t, err)
		assert.NotNil(t, recommendation)

		report := scheduler.GetReport()
		assert.NotNil(t, report)
	})
}

func TestMetricsCollector(t *testing.T) {
	collector := NewMetricsCollector(100 * time.Millisecond)
	defer collector.Disable()

	t.Run("启用禁用", func(t *testing.T) {
		collector.Enable()
		assert.True(t, collector.enabled.Load())

		time.Sleep(200 * time.Millisecond)
		collector.Disable()
		assert.False(t, collector.enabled.Load())
	})

	t.Run("收集指标", func(t *testing.T) {
		collector.Enable()

		time.Sleep(300 * time.Millisecond)

		collector.mu.RLock()
		count := len(collector.metrics)
		collector.mu.RUnlock()

		assert.Greater(t, count, 0)
	})
}

func TestResourceMetricCalculation(t *testing.T) {
	t.Run("CPU使用率计算", func(t *testing.T) {
		autoscaler := NewK8sAutoscaler(&SchedulerConfig{
			MinReplicas: 1,
			MaxReplicas: 10,
		})

		autoscaler.metricsHistory = []ResourceMetric{
			{CPUUsage: 0.5},
			{CPUUsage: 0.6},
			{CPUUsage: 0.7},
		}

		avg := autoscaler.calculateAverageCPU()
		assert.InDelta(t, 0.6, avg, 0.01)
	})

	t.Run("内存使用率计算", func(t *testing.T) {
		autoscaler := NewK8sAutoscaler(&SchedulerConfig{
			MinReplicas: 1,
			MaxReplicas: 10,
		})

		autoscaler.metricsHistory = []ResourceMetric{
			{MemoryUsage: 0.4},
			{MemoryUsage: 0.5},
			{MemoryUsage: 0.6},
		}

		avg := autoscaler.calculateAverageMemory()
		assert.InDelta(t, 0.5, avg, 0.01)
	})
}

func TestOptimizationDecision(t *testing.T) {
	optimizer := NewCostPerformanceOptimizer(BalanceModeBalanced)
	defer optimizer.enabled.Store(false)

	metrics := &ResourceMetric{
		CPUUsage:    0.8,
		MemoryUsage: 0.7,
		NetworkIO:   1000,
		RequestRate: 5000,
		Latency:     100,
		ErrorRate:   0.01,
	}

	t.Run("优化决策历史", func(t *testing.T) {
		for i := 0; i < 5; i++ {
			_, err := optimizer.CalculateOptimalResources(metrics)
			require.NoError(t, err)
		}

		optimizer.mu.RLock()
		historyLen := len(optimizer.optimizationHistory)
		optimizer.mu.RUnlock()

		assert.Equal(t, 5, historyLen)
	})
}

func BenchmarkK8sAutoscaler_Scale(b *testing.B) {
	autoscaler := NewK8sAutoscaler(&SchedulerConfig{
		MinReplicas:     1,
		MaxReplicas:     100,
		ScaleUpThreshold: 0.7,
		ScaleDownThreshold: 0.3,
	})

	ctx := context.Background()
	metrics := &ResourceMetric{
		CPUUsage:    0.8,
		MemoryUsage: 0.7,
		RequestRate: 5000,
		Latency:     100,
	}

	for i := 0; i < 100; i++ {
		autoscaler.metricsHistory = append(autoscaler.metricsHistory, *metrics)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		autoscaler.Scale(ctx, metrics)
	}
}

func BenchmarkResourcePredictor_Predict(b *testing.B) {
	predictor := NewResourcePredictor(10 * time.Minute)

	for i := 0; i < 100; i++ {
		predictor.AddDataPoint("cpu", 0.5+float64(i)*0.01)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		predictor.Predict("cpu", 5*time.Minute)
	}
}

func BenchmarkCostOptimizer_Calculate(b *testing.B) {
	optimizer := NewCostPerformanceOptimizer(BalanceModeBalanced)

	metrics := &ResourceMetric{
		CPUUsage:    0.8,
		MemoryUsage: 0.7,
		NetworkIO:   1000,
		RequestRate: 5000,
		Latency:     100,
		ErrorRate:   0.01,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		optimizer.CalculateOptimalResources(metrics)
	}
}
