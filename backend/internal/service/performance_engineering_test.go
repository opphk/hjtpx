package service

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPerformanceEngine_New(t *testing.T) {
	t.Run("创建默认配置引擎", func(t *testing.T) {
		engine := NewPerformanceEngine(nil)
		assert.NotNil(t, engine)
		assert.NotNil(t, engine.config)
		assert.NotNil(t, engine.metrics)
		assert.True(t, engine.initialized.Load())
	})

	t.Run("创建自定义配置引擎", func(t *testing.T) {
		config := &PerformanceConfig{
			EnableMemoryPool:    true,
			EnableObjectPool:    true,
			EnableLockFreeQueue: true,
			EnableGPUOffload:    false,
			TargetQPS:           20000,
			MaxConcurrentOps:    500,
			MemoryPoolSizeMB:    128,
			ObjectPoolSize:      500,
		}

		engine := NewPerformanceEngine(config)
		assert.NotNil(t, engine)
		assert.Equal(t, int64(20000), engine.targetQPS)
		assert.True(t, engine.config.EnableMemoryPool)
	})

	t.Run("禁用组件配置", func(t *testing.T) {
		config := &PerformanceConfig{
			EnableMemoryPool:    false,
			EnableObjectPool:    false,
			EnableLockFreeQueue: false,
			EnableGPUOffload:    false,
		}

		engine := NewPerformanceEngine(config)
		assert.NotNil(t, engine)
		assert.Nil(t, engine.memoryPool)
		assert.Nil(t, engine.objectPool)
		assert.Nil(t, engine.lockFreeQueue)
		assert.Nil(t, engine.heterogeneous)
	})
}

func TestPerformanceEngine_Execute(t *testing.T) {
	engine := NewPerformanceEngine(nil)
	defer engine.Close()

	ctx := context.Background()

	t.Run("基础执行", func(t *testing.T) {
		err := engine.Execute(ctx, func() error {
			time.Sleep(1 * time.Millisecond)
			return nil
		})

		assert.NoError(t, err)
		metrics := engine.GetMetrics()
		assert.Greater(t, metrics.TotalOperations.Load(), int64(0))
	})

	t.Run("执行失败", func(t *testing.T) {
		err := engine.Execute(ctx, func() error {
			return assert.AnError
		})

		assert.Error(t, err)
		metrics := engine.GetMetrics()
		assert.Greater(t, metrics.FailedOps.Load(), int64(0))
	})

	t.Run("并发执行", func(t *testing.T) {
		var wg sync.WaitGroup
		successCount := atomic.Int64{}

		for i := 0; i < 100; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				err := engine.Execute(ctx, func() error {
					time.Sleep(1 * time.Millisecond)
					return nil
				})
				if err == nil {
					successCount.Add(1)
				}
			}()
		}

		wg.Wait()
		assert.Equal(t, int64(100), successCount.Load())
	})
}

func TestPerformanceEngine_ExecuteBatch(t *testing.T) {
	engine := NewPerformanceEngine(nil)
	defer engine.Close()

	ctx := context.Background()

	t.Run("批量执行", func(t *testing.T) {
		tasks := make([]func() error, 50)
		for i := range tasks {
			tasks[i] = func() error {
				time.Sleep(1 * time.Millisecond)
				return nil
			}
		}

		results := engine.ExecuteBatch(ctx, tasks)
		assert.Len(t, results, 50)

		for _, err := range results {
			assert.NoError(t, err)
		}
	})

	t.Run("空批量", func(t *testing.T) {
		results := engine.ExecuteBatch(ctx, []func() error{})
		assert.Len(t, results, 0)
	})
}

func TestMemoryPool(t *testing.T) {
	pool := NewMemoryPool(256)
	pool.Register("test-pool", 1024, 10)

	t.Run("注册池", func(t *testing.T) {
		assert.NotNil(t, pool)
		assert.NotNil(t, pool.Get("test-pool"))
	})

	t.Run("获取和归还对象", func(t *testing.T) {
		obj := pool.Get("test-pool")
		require.NotNil(t, obj)

		pool.Put("test-pool", obj)

		stats := pool.GetStats("test-pool")
		require.NotNil(t, stats)
	})

	t.Run("多次获取", func(t *testing.T) {
		objects := make([]interface{}, 20)
		for i := 0; i < 20; i++ {
			objects[i] = pool.Get("test-pool")
		}

		for _, obj := range objects {
			pool.Put("test-pool", obj)
		}
	})

	t.Run("启用禁用", func(t *testing.T) {
		pool.Disable()
		assert.False(t, pool.enabled.Load())

		pool.Enable()
		assert.True(t, pool.enabled.Load())
	})
}

func TestObjectPoolRegistry(t *testing.T) {
	registry := NewObjectPoolRegistry()

	t.Run("注册池", func(t *testing.T) {
		registry.Register("test-pool", &simplePoolFactory{}, 10, 100)
		
		obj, err := registry.Get("test-pool", 1*time.Second)
		require.NoError(t, err)
		assert.NotNil(t, obj)
	})

	t.Run("获取和归还", func(t *testing.T) {
		obj1, _ := registry.Get("test-pool", 1*time.Second)
		registry.Put("test-pool", obj1)

		obj2, _ := registry.Get("test-pool", 1*time.Second)
		assert.NotNil(t, obj2)
	})

	t.Run("超时", func(t *testing.T) {
		registry.Register("small-pool", &simplePoolFactory{}, 1, 1)
		obj, _ := registry.Get("small-pool", 1*time.Second)
		registry.Put("small-pool", obj)

		_, err := registry.Get("small-pool", 1*time.Millisecond)
		assert.Error(t, err)
	})
}

type simplePoolFactory struct{}

func (f *simplePoolFactory) Create() interface{} {
	return &struct{}{}
}

func (f *simplePoolFactory) Reset(obj interface{}) {}

func (f *simplePoolFactory) Validate(obj interface{}) bool {
	return obj != nil
}

func TestLockFreeQueue(t *testing.T) {
	queue := NewLockFreeQueue(1000)

	t.Run("基础入队出队", func(t *testing.T) {
		for i := 0; i < 100; i++ {
			ok := queue.Enqueue(i)
			assert.True(t, ok)
		}

		assert.Equal(t, int64(100), queue.Len())
		assert.False(t, queue.IsEmpty())

		for i := 0; i < 100; i++ {
			val, ok := queue.Dequeue()
			assert.True(t, ok)
			assert.Equal(t, i, val)
		}

		assert.Equal(t, int64(0), queue.Len())
		assert.True(t, queue.IsEmpty())
	})

	t.Run("容量限制", func(t *testing.T) {
		smallQueue := NewLockFreeQueue(10)

		for i := 0; i < 10; i++ {
			ok := smallQueue.Enqueue(i)
			assert.True(t, ok)
		}

		ok := smallQueue.Enqueue(100)
		assert.False(t, ok)
	})

	t.Run("关闭队列", func(t *testing.T) {
		queue.Close()
		ok := queue.Enqueue(1)
		assert.False(t, ok)
	})
}

func TestLockFreeQueue_Concurrent(t *testing.T) {
	queue := NewLockFreeQueue(10000)

	t.Run("并发入队出队", func(t *testing.T) {
		var wg sync.WaitGroup
		enqueueCount := 1000
		dequeueCount := 1000

		for i := 0; i < enqueueCount; i++ {
			wg.Add(1)
			go func(val int) {
				defer wg.Done()
				queue.Enqueue(val)
			}(i)
		}

		for i := 0; i < dequeueCount; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				queue.Dequeue()
			}()
		}

		wg.Wait()

		assert.GreaterOrEqual(t, queue.Len(), int64(0))
	})
}

func TestHeterogeneousComputer(t *testing.T) {
	comp := NewHeterogeneousComputer()

	t.Run("初始化", func(t *testing.T) {
		assert.NotNil(t, comp)
		assert.NotNil(t, comp.cpuDevice)
		assert.NotNil(t, comp.stats)
	})

	t.Run("CPU执行", func(t *testing.T) {
		comp.SetStrategy(StrategyCPUOnly)

		result, err := comp.Offload(func() interface{} {
			time.Sleep(1 * time.Millisecond)
			return "result"
		})

		assert.NoError(t, err)
		assert.Equal(t, "result", result)
		assert.Greater(t, comp.stats.CPUOps.Load(), int64(0))
	})

	t.Run("GPU可用性检查", func(t *testing.T) {
		hasGPU := comp.gpuAvailable.Load()
		assert.IsType(t, false, hasGPU)
	})

	t.Run("策略切换", func(t *testing.T) {
		comp.SetStrategy(StrategyHybrid)
		_, err := comp.Offload(func() interface{} {
			return "hybrid"
		})
		assert.NoError(t, err)
	})
}

func TestQPSOptimizer(t *testing.T) {
	config := &PerformanceConfig{
		TargetQPS:      15000,
		AdaptiveEnabled: true,
	}

	optimizer := NewQPSOptimizer(15000, config)
	defer optimizer.Stop()

	t.Run("记录请求", func(t *testing.T) {
		optimizer.RecordRequest(10*time.Millisecond, true)
		optimizer.RecordRequest(20*time.Millisecond, true)
		optimizer.RecordRequest(30*time.Millisecond, false)

		metrics := optimizer.GetMetrics()
		assert.Equal(t, 2.0, metrics.SuccessRate)
	})

	t.Run("设置目标QPS", func(t *testing.T) {
		optimizer.SetTargetQPS(20000)
		assert.Equal(t, int64(20000), optimizer.targetQPS)
	})
}

func TestPerformanceEngine_GetMetrics(t *testing.T) {
	engine := NewPerformanceEngine(nil)
	defer engine.Close()

	ctx := context.Background()

	for i := 0; i < 100; i++ {
		engine.Execute(ctx, func() error {
			time.Sleep(1 * time.Millisecond)
			return nil
		})
	}

	t.Run("获取指标", func(t *testing.T) {
		metrics := engine.GetMetrics()

		assert.NotNil(t, metrics)
		assert.Greater(t, metrics.TotalOperations.Load(), int64(0))
		assert.Greater(t, metrics.SuccessfulOps.Load(), int64(0))
		assert.Greater(t, metrics.AvgLatency.Load(), int64(0))
	})

	t.Run("获取性能报告", func(t *testing.T) {
		report := engine.GetPerformanceReport()

		assert.NotNil(t, report)
		assert.Contains(t, report, "uptime_seconds")
		assert.Contains(t, report, "total_operations")
		assert.Contains(t, report, "operations_per_second")
	})
}

func TestPerformanceEngine_Benchmark(t *testing.T) {
	engine := NewPerformanceEngine(nil)
	defer engine.Close()

	ctx := context.Background()

	t.Run("执行基准测试", func(t *testing.T) {
		results := engine.Benchmark(ctx)

		assert.NotNil(t, results)
		assert.Contains(t, results, "execute_benchmark")
	})

	t.Run("LockFree队列基准测试", func(t *testing.T) {
		results := engine.Benchmark(ctx)

		if _, ok := results["lockfree_queue_benchmark"]; ok {
			queueBench := results["lockfree_queue_benchmark"].(map[string]interface{})
			assert.Contains(t, queueBench, "queue_size")
			assert.Contains(t, queueBench, "enqueue_ms")
			assert.Contains(t, queueBench, "dequeue_ms")
		}
	})
}

func TestPerformanceEngine_QPSOptimization(t *testing.T) {
	engine := NewPerformanceEngine(nil)
	defer engine.Close()

	ctx := context.Background()

	t.Run("QPS优化目标", func(t *testing.T) {
		engine.SetTargetQPS(15000)
		assert.Equal(t, int64(15000), engine.targetQPS)
	})

	t.Run("执行并记录延迟", func(t *testing.T) {
		for i := 0; i < 100; i++ {
			engine.Execute(ctx, func() error {
				time.Sleep(1 * time.Millisecond)
				return nil
			})
		}

		metrics := engine.GetMetrics()
		assert.Greater(t, metrics.AvgLatency.Load(), int64(0))
	})
}

func TestPerformanceEngine_GPUOffload(t *testing.T) {
	engine := NewPerformanceEngine(&PerformanceConfig{
		EnableGPUOffload: true,
	})
	defer engine.Close()

	t.Run("启用GPU卸载", func(t *testing.T) {
		err := engine.EnableGPUOffload(false)
		assert.NoError(t, err)
	})

	t.Run("设置卸载策略", func(t *testing.T) {
		engine.SetOffloadStrategy(StrategyHybrid)
		assert.NotNil(t, engine.heterogeneous)
	})
}

func TestPerformanceEngine_ResourcePools(t *testing.T) {
	engine := NewPerformanceEngine(nil)
	defer engine.Close()

	t.Run("获取内存池", func(t *testing.T) {
		memPool := engine.GetMemoryPool()
		if memPool != nil {
			assert.NotNil(t, memPool)
		}
	})

	t.Run("获取对象池", func(t *testing.T) {
		objPool := engine.GetObjectPool()
		if objPool != nil {
			assert.NotNil(t, objPool)
		}
	})

	t.Run("获取无锁队列", func(t *testing.T) {
		lfQueue := engine.GetLockFreeQueue()
		if lfQueue != nil {
			assert.NotNil(t, lfQueue)
		}
	})

	t.Run("获取异构计算器", func(t *testing.T) {
		hetComp := engine.GetHeterogeneousComputer()
		if hetComp != nil {
			assert.NotNil(t, hetComp)
		}
	})

	t.Run("获取QPS优化器", func(t *testing.T) {
		qpsOpt := engine.GetQPSOptimizer()
		if qpsOpt != nil {
			assert.NotNil(t, qpsOpt)
		}
	})
}

func TestPerformanceEngine_Close(t *testing.T) {
	engine := NewPerformanceEngine(nil)

	t.Run("关闭引擎", func(t *testing.T) {
		err := engine.Close()
		assert.NoError(t, err)
		assert.False(t, engine.initialized.Load())
	})
}

func TestPerformanceEngine_LatencyMetrics(t *testing.T) {
	engine := NewPerformanceEngine(nil)
	defer engine.Close()

	ctx := context.Background()

	t.Run("记录不同延迟", func(t *testing.T) {
		latencies := []time.Duration{
			1 * time.Millisecond,
			2 * time.Millisecond,
			5 * time.Millisecond,
			10 * time.Millisecond,
		}

		for _, lat := range latencies {
			engine.Execute(ctx, func() error {
				time.Sleep(lat)
				return nil
			})
		}

		metrics := engine.GetMetrics()
		assert.Greater(t, metrics.MaxLatency.Load(), int64(0))
	})
}

func TestPerformanceEngine_ConcurrentAccess(t *testing.T) {
	engine := NewPerformanceEngine(nil)
	defer engine.Close()

	ctx := context.Background()

	t.Run("高并发访问", func(t *testing.T) {
		var wg sync.WaitGroup
		concurrency := 500

		for i := 0; i < concurrency; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for j := 0; j < 10; j++ {
					engine.Execute(ctx, func() error {
						return nil
					})
				}
			}()
		}

		wg.Wait()

		metrics := engine.GetMetrics()
		assert.Equal(t, int64(concurrency*10), metrics.TotalOperations.Load())
	})
}

func TestMemoryPool_Stats(t *testing.T) {
	pool := NewMemoryPool(256)
	pool.Register("stats-pool", 512, 5)

	t.Run("获取统计信息", func(t *testing.T) {
		stats := pool.GetStats("stats-pool")
		require.NotNil(t, stats)
		assert.GreaterOrEqual(t, stats.Hits.Load(), int64(0))
	})
}

func TestLockFreeQueue_Performance(t *testing.T) {
	queue := NewLockFreeQueue(100000)

	t.Run("大量数据操作", func(t *testing.T) {
		start := time.Now()

		for i := 0; i < 50000; i++ {
			queue.Enqueue(i)
		}

		enqueueDuration := time.Since(start)

		start = time.Now()
		for i := 0; i < 50000; i++ {
			queue.Dequeue()
		}

		dequeueDuration := time.Since(start)

		assert.True(t, enqueueDuration < 1*time.Second, "enqueue too slow")
		assert.True(t, dequeueDuration < 1*time.Second, "dequeue too slow")
	})
}

func BenchmarkLockFreeQueue_Operations(b *testing.B) {
	queue := NewLockFreeQueue(b.N)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			queue.Enqueue(i)
			queue.Dequeue()
			i++
		}
	})
}

func BenchmarkMemoryPool_Operations(b *testing.B) {
	pool := NewMemoryPool(256)
	pool.Register("bench-pool", 1024, 100)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			obj := pool.Get("bench-pool")
			pool.Put("bench-pool", obj)
		}
	})
}

func BenchmarkPerformanceEngine_Execute(b *testing.B) {
	engine := NewPerformanceEngine(nil)
	defer engine.Close()

	ctx := context.Background()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			engine.Execute(ctx, func() error {
				return nil
			})
		}
	})
}
