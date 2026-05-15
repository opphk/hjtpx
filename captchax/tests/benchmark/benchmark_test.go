package benchmark

import (
	"context"
	"image"
	"image/color"
	"strconv"
	"testing"
	"time"

	"captchax/internal/cache"
	"captchax/internal/database"
	"captchax/internal/optimization"
)

func createTestImage(width, height int) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for x := 0; x < width; x++ {
		for y := 0; y < height; y++ {
			img.Set(x, y, color.RGBA{
				R: uint8(x % 256),
				G: uint8(y % 256),
				B: uint8((x + y) % 256),
				A: 255,
			})
		}
	}
	return img
}

func BenchmarkImageGeneration(b *testing.B) {
	compressor := optimization.NewImageCompressor()
	img := createTestImage(300, 150)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = compressor.CompressJPEG(img, 80)
	}
}

func BenchmarkCacheGet(b *testing.B) {
	cache := optimization.NewImageCache(1000, 10*time.Minute)
	cache.Set("test-key", []byte("test-data"))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = cache.Get("test-key")
	}
}

func BenchmarkCacheSet(b *testing.B) {
	cache := optimization.NewImageCache(1000, 10*time.Minute)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.Set("test-key", []byte("test-data"))
	}
}

func BenchmarkCacheGetMiss(b *testing.B) {
	cache := optimization.NewImageCache(1000, 10*time.Minute)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = cache.Get("non-existent-key")
	}
}

// Additional benchmarks for key paths

func BenchmarkCacheSetMultiple(b *testing.B) {
	cache := optimization.NewImageCache(10000, 10*time.Minute)
	keys := make([]string, 1000)
	for i := 0; i < 1000; i++ {
		keys[i] = "key-" + strconv.Itoa(i)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.Set(keys[i%1000], []byte("test-data"))
	}
}

func BenchmarkCacheConcurrentGet(b *testing.B) {
	cache := optimization.NewImageCache(1000, 10*time.Minute)
	cache.Set("test-key", []byte("test-data"))

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = cache.Get("test-key")
		}
	})
}

func BenchmarkCacheConcurrentSet(b *testing.B) {
	cache := optimization.NewImageCache(1000, 10*time.Minute)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			key := "key-" + strconv.Itoa(i%100)
			cache.Set(key, []byte("test-data"))
			i++
		}
	})
}

func BenchmarkCacheMixedOperations(b *testing.B) {
	cache := optimization.NewImageCache(1000, 10*time.Minute)
	for i := 0; i < 100; i++ {
		cache.Set("key-"+strconv.Itoa(i), []byte("test-data"))
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			key := "key-" + strconv.Itoa(i%200)
			if i%2 == 0 {
				_, _ = cache.Get(key)
			} else {
				cache.Set(key, []byte("test-data"))
			}
			i++
		}
	})
}

func BenchmarkConnectionPoolAcquire(b *testing.B) {
	poolConfig := database.ConnectionPoolConfig{
		MaxOpenConns:    100,
		MaxIdleConns:    20,
		ConnMaxLifetime: 30 * time.Minute,
		ConnMaxIdleTime: 10 * time.Minute,
	}

	conn, err := database.NewConnectionPool("postgres://test:test@localhost/test?sslmode=disable", poolConfig)
	if err != nil {
		b.Skipf("Skipping benchmark: failed to create connection pool: %v", err)
		return
	}
	defer conn.Close()

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tc, err := conn.AcquireTrackedConn(ctx)
		if err == nil {
			tc.Release()
		}
	}
}

func BenchmarkConnectionPoolHealthCheck(b *testing.B) {
	poolConfig := database.ConnectionPoolConfig{
		MaxOpenConns:    100,
		MaxIdleConns:    20,
		ConnMaxLifetime: 30 * time.Minute,
		ConnMaxIdleTime: 10 * time.Minute,
	}

	conn, err := database.NewConnectionPool("postgres://test:test@localhost/test?sslmode=disable", poolConfig)
	if err != nil {
		b.Skipf("Skipping benchmark: failed to create connection pool: %v", err)
		return
	}
	defer conn.Close()

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		conn.HealthCheck(ctx)
	}
}

func BenchmarkConnectionPoolStats(b *testing.B) {
	poolConfig := database.ConnectionPoolConfig{
		MaxOpenConns:    100,
		MaxIdleConns:    20,
		ConnMaxLifetime: 30 * time.Minute,
		ConnMaxIdleTime: 10 * time.Minute,
	}

	conn, err := database.NewConnectionPool("postgres://test:test@localhost/test?sslmode=disable", poolConfig)
	if err != nil {
		b.Skipf("Skipping benchmark: failed to create connection pool: %v", err)
		return
	}
	defer conn.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		conn.GetStats()
	}
}

func BenchmarkWarmupProgressTracking(b *testing.B) {
	progress := cache.NewWarmupProgress()
	progress.Start("test_phase")
	progress.SetTotalItems(10000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		progress.IncrementCompleted()
		progress.GetProgress()
	}
}

func BenchmarkWarmupServiceRegistration(b *testing.B) {
	c := cache.NewAdvancedCache(nil, cache.NewLocalLRUCache(1000, 10*time.Minute))
	service := cache.NewWarmupService(c)

	strategy := cache.NewPopularCaptchaWarmupStrategy(c)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		service.RegisterStrategy(strategy)
		service.UnregisterStrategy(strategy.GetStrategyName())
	}
}

func BenchmarkLoadTestScenario(b *testing.B) {
	baseURL := "http://localhost:8080"
	scenario := &CaptchaGenerateScenario{
		BaseURL:  baseURL,
		AppID:    "benchmark-app",
		ClientIP: "127.0.0.1",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req, err := scenario.Request()
		if err != nil {
			b.Fatal(err)
		}
		_ = req
	}
}

func BenchmarkLoadTestScenarioConcurrent(b *testing.B) {
	baseURL := "http://localhost:8080"
	scenario := &CaptchaGenerateScenario{
		BaseURL:  baseURL,
		AppID:    "benchmark-app",
		ClientIP: "127.0.0.1",
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			req, _ := scenario.Request()
			_ = req
		}
	})
}

func BenchmarkLoadTesterCreation(b *testing.B) {
	config := LoadTestConfig{
		TargetQPS:  1000,
		Duration:  1 * time.Minute,
		NumWorkers: 10,
		BaseURL:   "http://localhost:8080",
		Timeout:   10 * time.Second,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		NewLoadTester(config)
	}
}

func BenchmarkMixedLoadScenario(b *testing.B) {
	baseURL := "http://localhost:8080"
	scenario := &MixedLoadScenario{
		BaseURL: baseURL,
		Scenarios: []LoadTestScenario{
			&HealthCheckScenario{BaseURL: baseURL},
			&CaptchaGenerateScenario{BaseURL: baseURL, AppID: "test-app", ClientIP: "127.0.0.1"},
		},
		Weights: []int{30, 70},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req, _ := scenario.Request()
		_ = req
	}
}

func BenchmarkConnectionPoolMetricsRecording(b *testing.B) {
	metrics := database.NewConnectionPoolMetrics()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		metrics.RecordAcquire(true, 5*time.Millisecond)
	}
}

func BenchmarkPoolManagerOperations(b *testing.B) {
	pm := database.GetPoolManager()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pm.GetAllStats()
	}
}

func BenchmarkWarmupDetailedProgress(b *testing.B) {
	c := cache.NewAdvancedCache(nil, cache.NewLocalLRUCache(1000, 10*time.Minute))
	service := cache.NewWarmupService(c)

	service.RegisterStrategy(cache.NewStartupWarmupStrategy(c))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		service.GetDetailedProgress()
	}
}

func BenchmarkLeakDetectorStats(b *testing.B) {
	leakDetector := database.NewLeakDetector(5 * time.Minute)

	for i := 0; i < 100; i++ {
		leakDetector.GetStats()
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		leakDetector.GetStats()
	}
}

func BenchmarkWarmupMetrics(b *testing.B) {
	metrics := cache.NewWarmupMetrics()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		metrics.RecordWarmup(true, 100, 5*time.Second)
	}
}

func BenchmarkLoadTestResultCalculation(b *testing.B) {
	result := &LoadTestResult{
		LatencyArray: make([]time.Duration, 10000),
	}
	for i := 0; i < 10000; i++ {
		result.LatencyArray[i] = time.Duration(i%1000) * time.Millisecond
	}
	sortDurations(result.LatencyArray)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result.AvgLatency = calculateAverage(result.LatencyArray)
		n := len(result.LatencyArray)
		result.P50Latency = result.LatencyArray[n*50/100]
		result.P90Latency = result.LatencyArray[n*90/100]
		result.P99Latency = result.LatencyArray[n*99/100]
	}
}
