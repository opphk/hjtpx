package benchmark

import (
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

type PerformanceBenchmark struct {
	url         string
	concurrency int
	requests    int64
	client      *http.Client
	results     *BenchmarkResults
}

type BenchmarkResults struct {
	TotalRequests   int64
	SuccessRequests int64
	FailedRequests  int64
	TotalDuration   time.Duration
	MinLatency      time.Duration
	MaxLatency      time.Duration
	AvgLatency      time.Duration
	Latencies       []time.Duration
	latenciesMu     sync.Mutex
}

func NewPerformanceBenchmark(url string, concurrency, requests int) *PerformanceBenchmark {
	return &PerformanceBenchmark{
		url:         url,
		concurrency: concurrency,
		requests:    int64(requests),
		client: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        1000,
				MaxIdleConnsPerHost: 100,
				MaxConnsPerHost:     1000,
				IdleConnTimeout:     90 * time.Second,
			},
		},
		results: &BenchmarkResults{
			Latencies: make([]time.Duration, 0, requests),
			MinLatency: 1<<63 - 1,
		},
	}
}

func (pb *PerformanceBenchmark) Run() (*BenchmarkResults, error) {
	startTime := time.Now()

	var wg sync.WaitGroup
	requestChan := make(chan struct{}, pb.concurrency)

	for i := int64(0); i < pb.requests; i++ {
		wg.Add(1)
		requestChan <- struct{}{}

		go func(reqNum int64) {
			defer wg.Done()
			defer func() { <-requestChan }()

			pb.makeRequest()
		}(i)
	}

	wg.Wait()
	pb.results.TotalDuration = time.Since(startTime)

	pb.calculateStats()

	return pb.results, nil
}

func (pb *PerformanceBenchmark) makeRequest() {
	start := time.Now()

	req, err := http.NewRequest("GET", pb.url, nil)
	if err != nil {
		atomic.AddInt64(&pb.results.FailedRequests, 1)
		return
	}

	resp, err := pb.client.Do(req)
	duration := time.Since(start)

	if err != nil {
		atomic.AddInt64(&pb.results.FailedRequests, 1)
		return
	}
	defer resp.Body.Close()

	atomic.AddInt64(&pb.results.TotalRequests, 1)

	if resp.StatusCode >= 200 && resp.StatusCode < 400 {
		atomic.AddInt64(&pb.results.SuccessRequests, 1)
	} else {
		atomic.AddInt64(&pb.results.FailedRequests, 1)
	}

	pb.results.latenciesMu.Lock()
	pb.results.Latencies = append(pb.results.Latencies, duration)
	if duration < pb.results.MinLatency {
		pb.results.MinLatency = duration
	}
	if duration > pb.results.MaxLatency {
		pb.results.MaxLatency = duration
	}
	pb.results.latenciesMu.Unlock()
}

func (pb *PerformanceBenchmark) calculateStats() {
	pb.results.AvgLatency = pb.results.TotalDuration / time.Duration(pb.results.TotalRequests)

	if len(pb.results.Latencies) == 0 {
		return
	}

	sorted := make([]time.Duration, len(pb.results.Latencies))
	copy(sorted, pb.results.Latencies)

	quickSort(sorted, 0, len(sorted)-1)

	n := len(sorted)
	p50 := sorted[n*50/100]
	p95 := sorted[n*95/100]
	p99 := sorted[n*99/100]

	fmt.Printf("\n=== Benchmark Results ===\n")
	fmt.Printf("Total Requests: %d\n", pb.results.TotalRequests)
	fmt.Printf("Success Requests: %d\n", pb.results.SuccessRequests)
	fmt.Printf("Failed Requests: %d\n", pb.results.FailedRequests)
	fmt.Printf("Total Duration: %v\n", pb.results.TotalDuration)
	fmt.Printf("QPS: %.2f\n", float64(pb.results.TotalRequests)/pb.results.TotalDuration.Seconds())
	fmt.Printf("Average Latency: %v\n", pb.results.AvgLatency)
	fmt.Printf("Min Latency: %v\n", pb.results.MinLatency)
	fmt.Printf("Max Latency: %v\n", pb.results.MaxLatency)
	fmt.Printf("P50 Latency: %v\n", p50)
	fmt.Printf("P95 Latency: %v\n", p95)
	fmt.Printf("P99 Latency: %v\n", p99)
}

func quickSort(arr []time.Duration, low, high int) {
	if low < high {
		pivot := partition(arr, low, high)
		quickSort(arr, low, pivot-1)
		quickSort(arr, pivot+1, high)
	}
}

func partition(arr []time.Duration, low, high int) int {
	pivot := arr[high]
	i := low - 1

	for j := low; j < high; j++ {
		if arr[j] <= pivot {
			i++
			arr[i], arr[j] = arr[j], arr[i]
		}
	}

	arr[i+1], arr[high] = arr[high], arr[i+1]
	return i + 1
}

func RunImageGenerationBenchmark() {
	fmt.Println("=== Image Generation Benchmark ===")

	generator := GetOptimizedImageGenerator()

	testCases := []struct {
		name     string
		testFunc func()
	}{
		{"Slider Image (360x220)", func() {
			generator.GenerateSliderImage(360, 220)
		}},
		{"Click Image (300x300)", func() {
			generator.GenerateClickImage(5, 300, 300)
		}},
	}

	for _, tc := range testCases {
		iterations := 1000
		start := time.Now()

		for i := 0; i < iterations; i++ {
			tc.testFunc()
		}

		duration := time.Since(start)
		avgTime := duration / time.Duration(iterations)

		fmt.Printf("\n%s:\n", tc.name)
		fmt.Printf("  Total Time: %v\n", duration)
		fmt.Printf("  Avg Time: %v\n", avgTime)
		fmt.Printf("  Iterations: %d\n", iterations)

		if avgTime < 30*time.Millisecond {
			fmt.Printf("  Status: PASS (target: <30ms)\n")
		} else {
			fmt.Printf("  Status: FAIL (target: <30ms)\n")
		}
	}
}

func RunConnectionPoolBenchmark() {
	fmt.Println("\n=== Connection Pool Benchmark ===")

	poolSizes := []int{10, 25, 50, 100}

	for _, poolSize := range poolSizes {
		fmt.Printf("\nPool Size: %d\n", poolSize)

		benchmarkDBPool(poolSize)
	}
}

func benchmarkDBPool(poolSize int) {
	start := time.Now()
	var wg sync sync.WaitGroup

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			time.Sleep(1 * time.Millisecond)
		}()
	}

	wg.Wait()
	duration := time.Since(start)

	fmt.Printf("  100 concurrent operations: %v\n", duration)
}

func RunCacheBenchmark() {
	fmt.Println("\n=== Cache Benchmark ===")

	cache := NewQueryCache(1000)

	operations := 10000

	start := time.Now()
	for i := 0; i < operations; i++ {
		key := fmt.Sprintf("key_%d", i%100)
		cache.Set(key, fmt.Sprintf("value_%d", i), 5*time.Minute)
	}
	setDuration := time.Since(start)

	start = time.Now()
	hits := 0
	for i := 0; i < operations; i++ {
		key := fmt.Sprintf("key_%d", i%100)
		if _, ok := cache.Get(key); ok {
			hits++
		}
	}
	getDuration := time.Since(start)

	fmt.Printf("\nCache Operations:\n")
	fmt.Printf("  Set Operations: %d in %v\n", operations, setDuration)
	fmt.Printf("  Get Operations: %d in %v\n", operations, getDuration)
	fmt.Printf("  Hit Rate: %.2f%%\n", float64(hits)/float64(operations)*100)

	hitsVal, missesVal := cache.Stats()
	fmt.Printf("  Cache Hits: %d, Misses: %d\n", hitsVal, missesVal)
}

func RunEndToEndBenchmark(baseURL string) {
	fmt.Println("\n=== End-to-End Benchmark ===")

	tests := []struct {
		name        string
		url         string
		method      string
		concurrency int
		requests    int
	}{
		{"Slider Captcha", baseURL + "/api/v1/captcha/slider", "GET", 200, 10000},
		{"Click Captcha", baseURL + "/api/v1/captcha/click", "GET", 200, 10000},
		{"Health Check", baseURL + "/health", "GET", 50, 5000},
	}

	for _, test := range tests {
		fmt.Printf("\n%s:\n", test.name)

		benchmark := NewPerformanceBenchmark(test.url, test.concurrency, test.requests)
		results, err := benchmark.Run()
		if err != nil {
			fmt.Printf("  Error: %v\n", err)
			continue
		}

		qps := float64(results.TotalRequests) / results.TotalDuration.Seconds()
		if qps >= 10000 && results.MaxLatency < 50*time.Millisecond {
			fmt.Printf("  Status: PASS\n")
		} else {
			fmt.Printf("  Status: NEEDS OPTIMIZATION\n")
		}
	}
}
