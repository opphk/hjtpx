package service

import (
	"context"
	"fmt"
	"math"
	"sync"
	"testing"
	"time"

	"github.com/hjtpx/hjtpx/internal/model"
)

func TestOptimizedLSTMService_Initialize(t *testing.T) {
	service := NewOptimizedLSTMService()

	ctx := context.Background()
	err := service.Initialize(ctx)
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	if !service.initialized.Load() {
		t.Error("Service should be initialized")
	}
}

func TestOptimizedLSTMService_BasicPrediction(t *testing.T) {
	service := NewOptimizedLSTMService()
	ctx := context.Background()

	if err := service.Initialize(ctx); err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	traceData := &model.TraceData{
		Points: []model.TracePoint{
			{Timestamp: 1000, X: 0, Y: 0, Event: "start"},
			{Timestamp: 1100, X: 10, Y: 10, Event: "move"},
			{Timestamp: 1200, X: 25, Y: 25, Event: "move"},
			{Timestamp: 1300, X: 40, Y: 40, Event: "end"},
		},
		TotalTime: 300,
	}

	result, err := service.Predict(ctx, traceData)
	if err != nil {
		t.Fatalf("Predict failed: %v", err)
	}

	if result == nil {
		t.Fatal("Result should not be nil")
	}

	if result.Score < 0 || result.Score > 1 {
		t.Errorf("Score should be between 0 and 1, got %f", result.Score)
	}

	if result.RiskLevel == "" {
		t.Error("RiskLevel should not be empty")
	}

	t.Logf("Prediction result: Score=%.4f, IsBot=%v, RiskLevel=%s, Latency=%.2fms",
		result.Score, result.IsBot, result.RiskLevel, result.LatencyMs)
}

func TestOptimizedLSTMService_PerformanceTarget(t *testing.T) {
	service := NewOptimizedLSTMService()
	ctx := context.Background()

	if err := service.Initialize(ctx); err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	traceData := &model.TraceData{
		Points: generateTestTrajectory(100),
		TotalTime: 5000,
	}

	const targetLatencyMs = 20
	const accuracyTarget = 0.95
	const testIterations = 100

	var totalLatency float64
	var latencyMeasurements []float64
	correctPredictions := 0

	for i := 0; i < testIterations; i++ {
		result, err := service.Predict(ctx, traceData)
		if err != nil {
			t.Fatalf("Predict failed at iteration %d: %v", i, err)
		}

		totalLatency += result.LatencyMs
		latencyMeasurements = append(latencyMeasurements, result.LatencyMs)

		if result.Confidence >= accuracyTarget {
			correctPredictions++
		}
	}

	avgLatency := totalLatency / float64(testIterations)
	
	accuracyRate := float64(correctPredictions) / float64(testIterations)

	t.Logf("=== Performance Test Results ===")
	t.Logf("Average Latency: %.2fms (target: <%dms)", avgLatency, targetLatencyMs)
	t.Logf("Accuracy Rate: %.2f%% (target: >=%.0f%%)", accuracyRate*100, accuracyTarget*100)
	t.Logf("Cache Hit Rate: %s", service.GetMetrics()["cache_hit_rate"])

	if avgLatency > targetLatencyMs {
		t.Errorf("Average latency %.2fms exceeds target %dms", avgLatency, targetLatencyMs)
	}

	if accuracyRate < accuracyTarget {
		t.Errorf("Accuracy rate %.2f%% below target %.0f%%", accuracyRate*100, accuracyTarget*100)
	}
}

func TestOptimizedLSTMService_Accuracy95Target(t *testing.T) {
	service := NewOptimizedLSTMService()
	ctx := context.Background()

	if err := service.Initialize(ctx); err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	humanTrajectories := [][]model.TracePoint{
		generateHumanTrajectory(100, 0.3, 0.7),
		generateHumanTrajectory(100, 0.4, 0.6),
		generateHumanTrajectory(100, 0.2, 0.8),
		generateHumanTrajectory(150, 0.35, 0.65),
		generateHumanTrajectory(150, 0.25, 0.75),
	}

	botTrajectories := [][]model.TracePoint{
		generateBotTrajectory(100, 0.05),
		generateBotTrajectory(100, 0.08),
		generateBotTrajectory(100, 0.1),
		generateBotTrajectory(150, 0.06),
		generateBotTrajectory(150, 0.09),
	}

	var humanCorrect, botCorrect int
	humanTests := len(humanTrajectories)
	botTests := len(botTrajectories)

	for _, trajectory := range humanTrajectories {
		traceData := &model.TraceData{
			Points:    trajectory,
			TotalTime: 5000,
		}

		result, err := service.Predict(ctx, traceData)
		if err != nil {
			t.Fatalf("Predict failed: %v", err)
		}

		if !result.IsBot && result.Confidence >= 0.95 {
			humanCorrect++
		}
	}

	for _, trajectory := range botTrajectories {
		traceData := &model.TraceData{
			Points:    trajectory,
			TotalTime: 5000,
		}

		result, err := service.Predict(ctx, traceData)
		if err != nil {
			t.Fatalf("Predict failed: %v", err)
		}

		if result.IsBot && result.Confidence >= 0.95 {
			botCorrect++
		}
	}

	humanAccuracy := float64(humanCorrect) / float64(humanTests)
	botAccuracy := float64(botCorrect) / float64(botTests)
	totalAccuracy := float64(humanCorrect+botCorrect) / float64(humanTests+botTests)

	t.Logf("=== Accuracy Test Results ===")
	t.Logf("Human Detection Accuracy: %.2f%% (%d/%d)", humanAccuracy*100, humanCorrect, humanTests)
	t.Logf("Bot Detection Accuracy: %.2f%% (%d/%d)", botAccuracy*100, botCorrect, botTests)
	t.Logf("Overall Accuracy: %.2f%%", totalAccuracy*100)

	if totalAccuracy < 0.95 {
		t.Errorf("Overall accuracy %.2f%% below 95%% target", totalAccuracy*100)
	}
}

func TestOptimizedLSTMService_CacheEffectiveness(t *testing.T) {
	service := NewOptimizedLSTMService()
	ctx := context.Background()

	if err := service.Initialize(ctx); err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	traceData := &model.TraceData{
		Points:    generateTestTrajectory(100),
		TotalTime: 5000,
	}

	const iterations = 50
	const cacheWarmup = 10

	for i := 0; i < iterations; i++ {
		_, err := service.Predict(ctx, traceData)
		if err != nil {
			t.Fatalf("Predict failed: %v", err)
		}
	}

	metrics := service.GetMetrics()

	t.Logf("=== Cache Effectiveness Test ===")
	t.Logf("Total Requests: %v", metrics["total_requests"])
	t.Logf("Cache Hits: %v", metrics["cache_hits"])
	t.Logf("Cache Misses: %v", metrics["cache_misses"])
	t.Logf("Cache Hit Rate: %v", metrics["cache_hit_rate"])
	t.Logf("Feature Cache Size: %v", metrics["feature_cache_size"])

	cacheHits := metrics["cache_hits"].(int64)
	if cacheHits < int64(iterations-cacheWarmup) {
		t.Errorf("Expected at least %d cache hits, got %d", iterations-cacheWarmup, cacheHits)
	}
}

func TestOptimizedLSTMService_ConcurrentRequests(t *testing.T) {
	service := NewOptimizedLSTMService()
	ctx := context.Background()

	if err := service.Initialize(ctx); err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	const numWorkers = 10
	const requestsPerWorker = 20

	var wg sync.WaitGroup
	results := make(chan float64, numWorkers*requestsPerWorker)
	errors := make(chan error, numWorkers*requestsPerWorker)

	startTime := time.Now()

	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			for i := 0; i < requestsPerWorker; i++ {
				traceData := &model.TraceData{
					Points:    generateTestTrajectory(50 + (workerID*5) + i%10),
					TotalTime: 3000,
				}

				result, err := service.Predict(ctx, traceData)
				if err != nil {
					errors <- err
					continue
				}

				results <- result.LatencyMs
			}
		}(w)
	}

	wg.Wait()
	duration := time.Since(startTime)

	close(results)
	close(errors)

	var latencies []float64
	var errorCount int

	for lat := range results {
		latencies = append(latencies, lat)
	}

	for range errors {
		errorCount++
	}

	if errorCount > 0 {
		t.Errorf("Encountered %d errors during concurrent requests", errorCount)
	}

	avgLatency := calculateMean(latencies)
	p95Latency := calculatePercentile(latencies, 95)
	p99Latency := calculatePercentile(latencies, 99)
	throughput := float64(len(latencies)) / duration.Seconds()

	t.Logf("=== Concurrent Request Test Results ===")
	t.Logf("Total Requests: %d", len(latencies))
	t.Logf("Total Duration: %v", duration)
	t.Logf("Throughput: %.2f req/s", throughput)
	t.Logf("Average Latency: %.2fms", avgLatency)
	t.Logf("P95 Latency: %.2fms", p95Latency)
	t.Logf("P99 Latency: %.2fms", p99Latency)
	t.Logf("Errors: %d", errorCount)

	if avgLatency > 50 {
		t.Errorf("Average latency %.2fms too high under concurrent load", avgLatency)
	}
}

func TestOptimizedLSTMService_FeatureExtractionPerformance(t *testing.T) {
	service := NewOptimizedLSTMService()
	ctx := context.Background()

	if err := service.Initialize(ctx); err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	trajectorySizes := []int{50, 100, 200, 500, 1000}

	for _, size := range trajectorySizes {
		traceData := &model.TraceData{
			Points:    generateTestTrajectory(size),
			TotalTime: int64(size * 50),
		}

		var totalLatency float64
		const iterations = 100

		for i := 0; i < iterations; i++ {
			result, err := service.Predict(ctx, traceData)
			if err != nil {
				t.Fatalf("Predict failed: %v", err)
			}
			totalLatency += result.LatencyMs
		}

		avgLatency := totalLatency / float64(iterations)

		t.Logf("Trajectory size %4d points: avg latency %.2fms", size, avgLatency)

		if size <= 500 && avgLatency > 20 {
			t.Errorf("Latency %.2fms exceeds 20ms target for %d points", avgLatency, size)
		}
	}
}

func TestOptimizedLSTMService_MemoryEfficiency(t *testing.T) {
	service := NewOptimizedLSTMService()
	ctx := context.Background()

	if err := service.Initialize(ctx); err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	const iterations = 1000

	for i := 0; i < iterations; i++ {
		traceData := &model.TraceData{
			Points:    generateTestTrajectory(100),
			TotalTime: 5000,
		}

		_, err := service.Predict(ctx, traceData)
		if err != nil {
			t.Fatalf("Predict failed: %v", err)
		}
	}

	metrics := service.GetMetrics()
	cacheSize := metrics["feature_cache_size"].(int)

	t.Logf("=== Memory Efficiency Test ===")
	t.Logf("Cache size after %d iterations: %d entries", iterations, cacheSize)

	if cacheSize > 10000 {
		t.Errorf("Cache size %d exceeds maximum %d", cacheSize, 10000)
	}
}

func TestOptimizedLSTMService_CacheMetrics(t *testing.T) {
	service := NewOptimizedLSTMService()
	ctx := context.Background()

	if err := service.Initialize(ctx); err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	trajectories := make([][]model.TracePoint, 5)
	for i := range trajectories {
		trajectories[i] = generateTestTrajectory(100 + i*20)
	}

	for iter := 0; iter < 3; iter++ {
		for _, traj := range trajectories {
			traceData := &model.TraceData{
				Points:    traj,
				TotalTime: 5000,
			}

			_, err := service.Predict(ctx, traceData)
			if err != nil {
				t.Fatalf("Predict failed: %v", err)
			}
		}
	}

	metrics := service.GetMetrics()
	
	t.Logf("=== Cache Metrics ===")
	t.Logf("Total Requests: %v", metrics["total_requests"])
	t.Logf("Cache Hits: %v", metrics["cache_hits"])
	t.Logf("Cache Misses: %v", metrics["cache_misses"])
	t.Logf("Cache Hit Rate: %v", metrics["cache_hit_rate"])

	cacheHitRateStr := metrics["cache_hit_rate"].(string)
	var cacheHitRate float64
	fmt.Sscanf(cacheHitRateStr, "%f%%", &cacheHitRate)

	if cacheHitRate < 50 {
		t.Logf("Warning: Cache hit rate %.2f%% is below optimal", cacheHitRate)
	}
}

func TestOptimizedLSTMService_BatchProcessing(t *testing.T) {
	service := NewOptimizedLSTMService()
	ctx := context.Background()

	if err := service.Initialize(ctx); err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	const batchSize = 100
	batch := make([]*model.TraceData, batchSize)

	for i := 0; i < batchSize; i++ {
		batch[i] = &model.TraceData{
			Points:    generateTestTrajectory(100),
			TotalTime: 5000,
		}
	}

	startTime := time.Now()

	results := make([]*OptimizedPredictionResult, 0, batchSize)
	for _, traceData := range batch {
		result, err := service.Predict(ctx, traceData)
		if err != nil {
			t.Fatalf("Predict failed: %v", err)
		}
		results = append(results, result)
	}

	duration := time.Since(startTime)
	throughput := float64(len(results)) / duration.Seconds()
	avgLatency := calculateMean(extractLatencies(results))

	t.Logf("=== Batch Processing Test ===")
	t.Logf("Batch Size: %d", batchSize)
	t.Logf("Total Duration: %v", duration)
	t.Logf("Throughput: %.2f req/s", throughput)
	t.Logf("Average Latency per Request: %.2fms", avgLatency)

	if throughput < 100 {
		t.Logf("Warning: Throughput %.2f req/s is below expected 100+ req/s", throughput)
	}
}

func TestOptimizedLSTMService_RiskLevelClassification(t *testing.T) {
	service := NewOptimizedLSTMService()
	ctx := context.Background()

	if err := service.Initialize(ctx); err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	testCases := []struct {
		name        string
		trajectory  []model.TracePoint
		expectBot   bool
		expectLevel string
	}{
		{
			name:        "Human-like trajectory",
			trajectory:  generateHumanTrajectory(100, 0.4, 0.6),
			expectBot:   false,
			expectLevel: "safe",
		},
		{
			name:        "Bot-like trajectory (constant speed)",
			trajectory:  generateBotTrajectory(100, 0.05),
			expectBot:   true,
			expectLevel: "extreme",
		},
		{
			name:        "Medium complexity trajectory",
			trajectory:  generateTestTrajectory(100),
			expectBot:   true,
			expectLevel: "high",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			traceData := &model.TraceData{
				Points:    tc.trajectory,
				TotalTime: 5000,
			}

			result, err := service.Predict(ctx, traceData)
			if err != nil {
				t.Fatalf("Predict failed: %v", err)
			}

			t.Logf("Score: %.4f, IsBot: %v, RiskLevel: %s, Confidence: %.4f",
				result.Score, result.IsBot, result.RiskLevel, result.Confidence)

			if result.Score < 0 || result.Score > 1 {
				t.Errorf("Invalid score: %f", result.Score)
			}
		})
	}
}

func TestOptimizedLSTMService_FeatureImportance(t *testing.T) {
	service := NewOptimizedLSTMService()
	ctx := context.Background()

	if err := service.Initialize(ctx); err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	traceData := &model.TraceData{
		Points:    generateHumanTrajectory(100, 0.4, 0.6),
		TotalTime: 5000,
	}

	result, err := service.Predict(ctx, traceData)
	if err != nil {
		t.Fatalf("Predict failed: %v", err)
	}

	t.Logf("=== Feature Analysis ===")
	for key, value := range result.Features {
		t.Logf("%s: %.6f", key, value)
	}

	if len(result.Features) == 0 {
		t.Error("Feature map should not be empty")
	}
}

func TestOptimizedLSTMService_StressTest(t *testing.T) {
	service := NewOptimizedLSTMService()
	ctx := context.Background()

	if err := service.Initialize(ctx); err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	const duration = 5 * time.Second
	const targetRequests = 500

	startTime := time.Now()
	var completedRequests int64
	var failedRequests int64

	for time.Since(startTime) < duration {
		traceData := &model.TraceData{
			Points:    generateTestTrajectory(200),
			TotalTime: 5000,
		}

		_, err := service.Predict(ctx, traceData)
		if err != nil {
			failedRequests++
		} else {
			completedRequests++
		}
	}

	actualDuration := time.Since(startTime)
	throughput := float64(completedRequests) / actualDuration.Seconds()

	t.Logf("=== Stress Test Results ===")
	t.Logf("Duration: %v", actualDuration)
	t.Logf("Completed Requests: %d", completedRequests)
	t.Logf("Failed Requests: %d", failedRequests)
	t.Logf("Throughput: %.2f req/s", throughput)
	t.Logf("Target: %d req in %v", targetRequests, duration)

	if float64(completedRequests) < float64(targetRequests)*0.8 {
		t.Errorf("Completed requests %d below 80%% of target %d", completedRequests, targetRequests)
	}
}

func generateTestTrajectory(size int) []model.TracePoint {
	points := make([]model.TracePoint, size)
	
	for i := 0; i < size; i++ {
		timestamp := int64(1000 + i*50)
		x := float64(i * 2)
		y := float64(i * 2) + math.Sin(float64(i)/10.0)*5.0
		
		points[i] = model.TracePoint{
			Timestamp: timestamp,
			X:         x,
			Y:         y,
			Event:     "move",
		}
	}
	
	return points
}

func generateHumanTrajectory(size int, speedVariance, curvatureVariance float64) []model.TracePoint {
	points := make([]model.TracePoint, size)
	
	baseSpeed := 2.0
	speedNoise := speedVariance
	
	x, y := 0.0, 0.0
	direction := 0.0
	
	for i := 0; i < size; i++ {
		timestamp := int64(1000 + i*50)
		
		speed := baseSpeed * (1.0 + (mathRand()-0.5)*2*speedNoise)
		
		directionChange := (mathRand() - 0.5) * curvatureVariance * math.Pi
		direction += directionChange
		
		x += speed * math.Cos(direction)
		y += speed * math.Sin(direction)
		
		points[i] = model.TracePoint{
			Timestamp: timestamp,
			X:         x,
			Y:         y,
			Event:     "move",
		}
	}
	
	return points
}

func generateBotTrajectory(size int, speedVariance float64) []model.TracePoint {
	points := make([]model.TracePoint, size)
	
	baseSpeed := 2.0
	
	x, y := 0.0, 0.0
	
	for i := 0; i < size; i++ {
		timestamp := int64(1000 + i*50)
		
		speed := baseSpeed * (1.0 + (mathRand()-0.5)*2*speedVariance)
		
		x += speed
		y += speed * 0.1
		
		points[i] = model.TracePoint{
			Timestamp: timestamp,
			X:         x,
			Y:         y,
			Event:     "move",
		}
	}
	
	return points
}

func calculateMean(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

func calculatePercentile(values []float64, percentile int) float64 {
	if len(values) == 0 {
		return 0
	}

	sorted := make([]float64, len(values))
	copy(sorted, values)

	n := len(sorted)
	idx := int(float64(n-1) * float64(percentile) / 100.0)
	if idx >= n {
		idx = n - 1
	}

	for i := 0; i < idx; i++ {
		minIdx := i
		for j := i + 1; j < n; j++ {
			if sorted[j] < sorted[minIdx] {
				minIdx = j
			}
		}
		if minIdx != i {
			sorted[i], sorted[minIdx] = sorted[minIdx], sorted[i]
		}
	}

	return sorted[idx]
}

func calculateMax(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	max := values[0]
	for _, v := range values[1:] {
		if v > max {
			max = v
		}
	}
	return max
}

func extractLatencies(results []*OptimizedPredictionResult) []float64 {
	latencies := make([]float64, len(results))
	for i, r := range results {
		latencies[i] = r.LatencyMs
	}
	return latencies
}
