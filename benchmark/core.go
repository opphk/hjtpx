package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/hjtpx/hjtpx/internal/service"
)

type BenchmarkResult struct {
	Name      string
	Ops      int64
	Duration  time.Duration
	OpsPerSec float64
	MemStats  runtime.MemStats
}

type CacheBenchmark struct {
	items      int
	keySize    int
	valueSize  int
}

func NewCacheBenchmark(items, keySize, valueSize int) *CacheBenchmark {
	return &CacheBenchmark{
		items:     items,
		keySize:   keySize,
		valueSize: valueSize,
	}
}

func (cb *CacheBenchmark) Run() *BenchmarkResult {
	log.Printf("Starting cache benchmark with %d items...", cb.items)

	var memBefore, memAfter runtime.MemStats
	runtime.ReadMemStats(&memBefore)

	start := time.Now()
	ctx := context.Background()

	cache := service.NewCacheService()

	var wg sync.WaitGroup
	ops := atomic.Int64{}

	for i := 0; i < runtime.NumCPU(); i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			for j := 0; j < cb.items/runtime.NumCPU(); j++ {
				key := fmt.Sprintf("benchmark:key:%d:%d", id, j)
				value := bytes.Repeat([]byte("x"), cb.valueSize)

				if err := cache.Set(ctx, key, value); err == nil {
					ops.Add(1)
				}

				if j%10 == 0 {
					cache.Get(ctx, key)
				}
			}
		}(i)
	}

	wg.Wait()
	duration := time.Since(start)

	runtime.ReadMemStats(&memAfter)

	opsSec := float64(ops.Load()) / duration.Seconds()

	return &BenchmarkResult{
		Name:       "Cache Benchmark",
		Ops:        ops.Load(),
		Duration:   duration,
		OpsPerSec:  opsSec,
		MemStats:   memAfter,
	}
}

type ConcurrencyBenchmark struct {
	workers    int
	tasks      int
	taskDelay  time.Duration
}

func NewConcurrencyBenchmark(workers, tasks int) *ConcurrencyBenchmark {
	return &ConcurrencyBenchmark{
		workers:   workers,
		tasks:     tasks,
		taskDelay: 1 * time.Millisecond,
	}
}

func (cb *ConcurrencyBenchmark) Run() *BenchmarkResult {
	log.Printf("Starting concurrency benchmark with %d workers and %d tasks...", cb.workers, cb.tasks)

	var memBefore, memAfter runtime.MemStats
	runtime.ReadMemStats(&memBefore)

	start := time.Now()

	pool := service.NewAdaptiveWorkerPool(cb.workers, cb.workers*2, cb.tasks)

	pool.Start()
	defer pool.Stop()

	var wg sync.WaitGroup
	ops := atomic.Int64{}

	for i := 0; i < cb.tasks; i++ {
		task := i
		wg.Add(1)

		pool.Submit(func() error {
			defer wg.Done()

			time.Sleep(cb.taskDelay)

			var sum int
			for j := 0; j < 100; j++ {
				sum += task * j
			}

			atomic.AddInt64(&ops, 1)
			return nil
		})
	}

	wg.Wait()
	duration := time.Since(start)

	runtime.ReadMemStats(&memAfter)

	opsSec := float64(ops.Load()) / duration.Seconds()

	return &BenchmarkResult{
		Name:       "Concurrency Benchmark",
		Ops:        ops.Load(),
		Duration:   duration,
		OpsPerSec:  opsSec,
		MemStats:   memAfter,
	}
}

type MemoryPoolBenchmark struct {
	iterations int
	bufSize    int
}

func NewMemoryPoolBenchmark(iterations, bufSize int) *MemoryPoolBenchmark {
	return &MemoryPoolBenchmark{
		iterations: iterations,
		bufSize:    bufSize,
	}
}

func (mb *MemoryPoolBenchmark) Run() *BenchmarkResult {
	log.Printf("Starting memory pool benchmark with %d iterations...", mb.iterations)

	var memBefore, memAfter runtime.MemStats
	runtime.ReadMemStats(&memBefore)

	start := time.Now()

	pool := service.NewMemoryPool()

	var wg sync.WaitGroup
	ops := atomic.Int64{}

	for i := 0; i < runtime.NumCPU(); i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			for j := 0; j < mb.iterations/runtime.NumCPU(); j++ {
				buf, _ := pool.GetBytes("bytes.Buffer")
				if buf != nil {
					buf.Write(make([]byte, mb.bufSize))
					pool.PutBytes("bytes.Buffer", buf)
					atomic.AddInt64(&ops, 1)
				}
			}
		}()
	}

	wg.Wait()
	duration := time.Since(start)

	runtime.ReadMemStats(&memAfter)

	opsSec := float64(ops.Load()) / duration.Seconds()

	memSaved := int64(memBefore.Alloc) - int64(memAfter.Alloc)
	if memSaved < 0 {
		memSaved = 0
	}

	log.Printf("Memory saved: %d bytes", memSaved)

	return &BenchmarkResult{
		Name:       "Memory Pool Benchmark",
		Ops:        ops.Load(),
		Duration:   duration,
		OpsPerSec:  opsSec,
		MemStats:   memAfter,
	}
}

type DatabaseQueryBenchmark struct {
	queries    int
	batchSize  int
}

func NewDatabaseQueryBenchmark(queries, batchSize int) *DatabaseQueryBenchmark {
	return &DatabaseQueryBenchmark{
		queries:   queries,
		batchSize: batchSize,
	}
}

func (dbb *DatabaseQueryBenchmark) Run() *BenchmarkResult {
	log.Printf("Starting database query benchmark with %d queries...", dbb.queries)

	var memBefore, memAfter runtime.MemStats
	runtime.ReadMemStats(&memBefore)

	start := time.Now()

	processor := service.NewAdaptiveBatchProcessor[int](dbb.batchSize, runtime.NumCPU())

	var ops atomic.Int64

	ctx := context.Background()
	items := make([]int, dbb.queries)
	for i := range items {
		items[i] = i
	}

	processor.Process(ctx, items, func(ctx context.Context, item int) error {
		time.Sleep(10 * time.Microsecond)
		atomic.AddInt64(&ops, 1)
		return nil
	})

	duration := time.Since(start)

	runtime.ReadMemStats(&memAfter)

	opsSec := float64(ops.Load()) / duration.Seconds()

	return &BenchmarkResult{
		Name:       "Database Query Benchmark",
		Ops:        ops.Load(),
		Duration:   duration,
		OpsPerSec:  opsSec,
		MemStats:   memAfter,
	}
}

type RedisCacheBenchmark struct {
	items   int
	keySize int
}

func NewRedisCacheBenchmark(items, keySize int) *RedisCacheBenchmark {
	return &RedisCacheBenchmark{
		items:   items,
		keySize: keySize,
	}
}

func (rcb *RedisCacheBenchmark) Run() *BenchmarkResult {
	log.Printf("Starting Redis cache benchmark with %d items...", rcb.items)

	var memBefore, memAfter runtime.MemStats
	runtime.ReadMemStats(&memBefore)

	start := time.Now()

	enhancedCache := &EnhancedCacheMock{}

	var wg sync.WaitGroup
	ops := atomic.Int64{}

	for i := 0; i < runtime.NumCPU(); i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			for j := 0; j < rcb.items/runtime.NumCPU(); j++ {
				key := fmt.Sprintf("redis:benchmark:%d:%d", id, j)
				value := bytes.Repeat([]byte("x"), 256)

				enhancedCache.Set(key, value)
				ops.Add(1)

				if j%5 == 0 {
					enhancedCache.Get(key)
				}
			}
		}(i)
	}

	wg.Wait()
	duration := time.Since(start)

	runtime.ReadMemStats(&memAfter)

	opsSec := float64(ops.Load()) / duration.Seconds()

	return &BenchmarkResult{
		Name:       "Redis Cache Benchmark",
		Ops:        ops.Load(),
		Duration:   duration,
		OpsPerSec:  opsSec,
		MemStats:   memAfter,
	}
}

type EnhancedCacheMock struct {
	data map[string][]byte
	mu   sync.RWMutex
}

func (c *EnhancedCacheMock) Set(key string, value []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.data[key] = value
}

func (c *EnhancedCacheMock) Get(key string) ([]byte, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	val, exists := c.data[key]
	return val, exists
}

func RunAllBenchmarks() []*BenchmarkResult {
	results := make([]*BenchmarkResult, 0)

	benchmarks := []struct {
		name string
		run  func() *BenchmarkResult
	}{
		{
			name: "Cache",
			run: func() *BenchmarkResult {
				return NewCacheBenchmark(10000, 64, 256).Run()
			},
		},
		{
			name: "Concurrency",
			run: func() *BenchmarkResult {
				return NewConcurrencyBenchmark(runtime.NumCPU()*2, 5000).Run()
			},
		},
		{
			name: "Memory Pool",
			run: func() *BenchmarkResult {
				return NewMemoryPoolBenchmark(100000, 1024).Run()
			},
		},
		{
			name: "Database Query",
			run: func() *BenchmarkResult {
				return NewDatabaseQueryBenchmark(1000, 50).Run()
			},
		},
		{
			name: "Redis Cache",
			run: func() *BenchmarkResult {
				return NewRedisCacheBenchmark(10000, 64).Run()
			},
		},
	}

	for _, bm := range benchmarks {
		log.Printf("\n=== Running %s Benchmark ===", bm.name)
		result := bm.run()
		result.Name = bm.name
		results = append(results, result)

		log.Printf("Results for %s:", bm.name)
		log.Printf("  Ops: %d", result.Ops)
		log.Printf("  Duration: %v", result.Duration)
		log.Printf("  Ops/sec: %.2f", result.OpsPerSec)
		log.Printf("  Alloc: %d bytes", result.MemStats.Alloc)
		log.Printf("  TotalAlloc: %d bytes", result.MemStats.TotalAlloc)
		log.Printf("  NumGC: %d", result.MemStats.NumGC)
	}

	return results
}

func PrintSummary(results []*BenchmarkResult) {
	fmt.Println("\n=== Benchmark Summary ===")
	fmt.Println("Name\t\tOps\t\tDuration\tOps/sec\t\tAlloc(bytes)")
	fmt.Println("------\t\t---\t\t--------\t-------\t\t-----------")

	for _, r := range results {
		fmt.Printf("%s\t\t%d\t\t%v\t\t%.2f\t\t%d\n",
			r.Name, r.Ops, r.Duration, r.OpsPerSec, r.MemStats.Alloc)
	}
}

func ExportResults(results []*BenchmarkResult, filename string) error {
	data, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		return err
	}

	log.Printf("Exporting results to %s...", filename)
	return nil
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Println("Starting Performance Benchmark Suite")
	log.Printf("Go Version: %s", runtime.Version())
	log.Printf("NumCPU: %d", runtime.NumCPU())
	log.Printf("NumGoroutine: %d", runtime.NumGoroutine())

	runtime.GC()
	time.Sleep(100 * time.Millisecond)

	results := RunAllBenchmarks()

	PrintSummary(results)

	runtime.GC()
	time.Sleep(100 * time.Millisecond)

	var finalStats runtime.MemStats
	runtime.ReadMemStats(&finalStats)

	log.Println("\n=== Final Memory Stats ===")
	log.Printf("Alloc: %d bytes", finalStats.Alloc)
	log.Printf("TotalAlloc: %d bytes", finalStats.TotalAlloc)
	log.Printf("Sys: %d bytes", finalStats.Sys)
	log.Printf("NumGC: %d", finalStats.NumGC)
	log.Printf("NumGoroutine: %d", runtime.NumGoroutine())

	ExportResults(results, "benchmark_results.json")
}
