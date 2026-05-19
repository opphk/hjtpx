package main

import (
	"context"
	"fmt"
	"log"
	"time"

	captchago "github.com/hjtpx/hjtpx/sdk/go"
)

type CaptchaMiddleware struct {
	client     *captchago.CaptchaClient
	maxRetries int
	timeout    time.Duration
}

func NewCaptchaMiddleware(baseURL, appID, appSecret string) *CaptchaMiddleware {
	cfg := &captchago.Config{
		BaseURL:     baseURL,
		MaxRetries:  3,
		HTTPTimeout: 10 * time.Second,
		DebugMode:   false,
	}

	client := captchago.NewCaptchaClient(appID, appSecret, cfg)

	return &CaptchaMiddleware{
		client:     client,
		maxRetries: 3,
		timeout:    10 * time.Second,
	}
}

type MiddlewareContext struct {
	StartTime    time.Time
	RequestID    string
	CaptchaType  string
	SessionID    string
	Verification bool
	Latency      time.Duration
	Error        error
}

func (m *CaptchaMiddleware) GenerateAndVerify() (*MiddlewareContext, error) {
	ctx := &MiddlewareContext{
		StartTime: time.Now(),
		RequestID: fmt.Sprintf("req-%d", time.Now().UnixNano()),
	}

	slider, err := m.client.GenerateSliderCaptcha()
	if err != nil {
		ctx.Error = err
		ctx.Latency = time.Since(ctx.StartTime)
		return ctx, err
	}

	ctx.SessionID = slider.ChallengeID
	ctx.CaptchaType = "slider"

	time.Sleep(100 * time.Millisecond)

	result, err := m.client.VerifySliderCaptcha(slider.ChallengeID, fmt.Sprintf("%d", slider.SecretX))
	if err != nil {
		ctx.Error = err
		ctx.Verification = false
	} else {
		ctx.Verification = result.Success
	}

	ctx.Latency = time.Since(ctx.StartTime)
	return ctx, nil
}

func (m *CaptchaMiddleware) BatchGenerate(count int) ([]*MiddlewareContext, error) {
	results := make([]*MiddlewareContext, count)
	errors := make([]error, count)
	successCount := 0

	for i := 0; i < count; i++ {
		ctx := &MiddlewareContext{
			StartTime: time.Now(),
			RequestID: fmt.Sprintf("batch-%d-%d", time.Now().UnixNano(), i),
		}

		slider, err := m.client.GenerateSliderCaptcha()
		if err != nil {
			ctx.Error = err
			errors[i] = err
			results[i] = ctx
			continue
		}

		ctx.SessionID = slider.ChallengeID
		ctx.CaptchaType = "slider"
		ctx.Latency = time.Since(ctx.StartTime)
		results[i] = ctx
		successCount++
	}

	if successCount == 0 {
		return results, fmt.Errorf("all requests failed")
	}

	return results, nil
}

func (m *CaptchaMiddleware) PerformanceTest(duration time.Duration) {
	fmt.Printf("[Performance Test - Duration: %v]\n", duration)
	fmt.Println("-----------------------------------")

	start := time.Now()
	endTime := start.Add(duration)

	var totalRequests int64
	var successfulRequests int64
	var failedRequests int64
	var totalLatency time.Duration
	var mutex int64

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if time.Now().After(endTime) {
				goto testComplete
			}

			current := atomicLoadInt64(&totalRequests)
			success := atomicLoadInt64(&successfulRequests)
			fail := atomicLoadInt64(&failedRequests)
			avgLatency := time.Duration(0)
			if current > 0 {
				avgLatency = totalLatency / time.Duration(current)
			}

			fmt.Printf("  [%3ds] Total: %d, Success: %d, Failed: %d, Avg Latency: %v\n",
				int(time.Since(start).Seconds()),
				current,
				success,
				fail,
				avgLatency,
			)

		default:
			go func() {
				reqStart := time.Now()

				_, err := m.client.GenerateSliderCaptcha()
				latency := time.Since(reqStart)

				atomicAddInt64(&totalRequests, 1)
				atomicAddInt64(&totalLatency, int64(latency))

				if err != nil {
					atomicAddInt64(&failedRequests, 1)
				} else {
					atomicAddInt64(&successfulRequests, 1)
				}
			}()

			time.Sleep(10 * time.Millisecond)
		}
	}

testComplete:
	finalTotal := atomicLoadInt64(&totalRequests)
	finalSuccess := atomicLoadInt64(&successfulRequests)
	finalFailed := atomicLoadInt64(&failedRequests)
	finalLatency := totalLatency

	fmt.Printf("\n✓ Test completed!\n")
	fmt.Printf("  Total requests: %d\n", finalTotal)
	fmt.Printf("  Successful: %d\n", finalSuccess)
	fmt.Printf("  Failed: %d\n", finalFailed)
	fmt.Printf("  Success rate: %.2f%%\n", float64(finalSuccess)/float64(finalTotal)*100)
	fmt.Printf("  Total latency: %v\n", finalLatency)
	fmt.Printf("  Avg latency: %v\n", finalLatency/time.Duration(finalTotal))
	fmt.Printf("  Requests/sec: %.2f\n", float64(finalTotal)/duration.Seconds())
}

func atomicLoadInt64(addr *int64) int64 {
	return *addr
}

func atomicAddInt64(addr *int64, delta int64) {
	*addr += delta
}

func (m *CaptchaMiddleware) StressTest(concurrency int, duration time.Duration) {
	fmt.Printf("[Stress Test - Concurrency: %d, Duration: %v]\n", concurrency, duration)
	fmt.Println("-----------------------------------")

	start := time.Now()
	endTime := start.Add(duration)

	var wg int64
	var totalRequests int64
	var successfulRequests int64
	var failedRequests int64
	var activeGoroutines int64

	for i := 0; i < concurrency; i++ {
		go func(workerID int) {
			atomicAddInt64(&wg, 1)
			atomicAddInt64(&activeGoroutines, 1)
			defer atomicAddInt64(&activeGoroutines, -1)

			for time.Now().Before(endTime) {
				reqStart := time.Now()

				_, err := m.client.GenerateSliderCaptcha()
				latency := time.Since(reqStart)

				atomicAddInt64(&totalRequests, 1)

				if err != nil {
					atomicAddInt64(&failedRequests, 1)
					log.Printf("Worker %d error: %v", workerID, err)
				} else {
					atomicAddInt64(&successfulRequests, 1)
				}

				if latency < 50*time.Millisecond {
					time.Sleep(50 * time.Millisecond)
				}
			}

			atomicAddInt64(&wg, -1)
		}(i)
	}

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			elapsed := time.Since(start)
			if elapsed >= duration {
				goto stressComplete
			}

			current := atomicLoadInt64(&totalRequests)
			success := atomicLoadInt64(&successfulRequests)
			fail := atomicLoadInt64(&failedRequests)
			active := atomicLoadInt64(&activeGoroutines)

			fmt.Printf("  [%3ds] Workers: %d, Total: %d, Success: %d, Failed: %d\n",
				int(elapsed.Seconds()),
				active,
				current,
				success,
				fail,
			)

		default:
			time.Sleep(100 * time.Millisecond)
		}
	}

stressComplete:
	for atomicLoadInt64(&wg) > 0 {
		time.Sleep(100 * time.Millisecond)
	}

	finalTotal := atomicLoadInt64(&totalRequests)
	finalSuccess := atomicLoadInt64(&successfulRequests)
	finalFailed := atomicLoadInt64(&failedRequests)

	fmt.Printf("\n✓ Stress test completed!\n")
	fmt.Printf("  Total requests: %d\n", finalTotal)
	fmt.Printf("  Successful: %d\n", finalSuccess)
	fmt.Printf("  Failed: %d\n", finalFailed)
	fmt.Printf("  Success rate: %.2f%%\n", float64(finalSuccess)/float64(finalTotal)*100)
	fmt.Printf("  Requests/sec: %.2f\n", float64(finalTotal)/duration.Seconds())
}

func (m *CaptchaMiddleware) Close() error {
	return m.client.Close()
}

func main() {
	fmt.Println("======================================")
	fmt.Println("  Middleware Integration Demo")
	fmt.Println("======================================")
	fmt.Println()

	middleware := NewCaptchaMiddleware(
		"http://localhost:8080",
		"app-id",
		"app-secret",
	)
	defer middleware.Close()

	middlewareContextExample(middleware)
	fmt.Println()

	batchGenerationExample(middleware)
	fmt.Println()

	performanceTestExample(middleware)
	fmt.Println()

	fmt.Println("======================================")
	fmt.Println("  Middleware demo completed!")
	fmt.Println("======================================")
}

func middlewareContextExample(m *CaptchaMiddleware) {
	fmt.Println("[Middleware Context Example]")
	fmt.Println("-----------------------------------")

	ctx, err := m.GenerateAndVerify()
	if err != nil {
		fmt.Printf("✗ Error: %v\n", err)
		return
	}

	fmt.Printf("✓ Request ID: %s\n", ctx.RequestID)
	fmt.Printf("✓ Captcha Type: %s\n", ctx.CaptchaType)
	fmt.Printf("✓ Session ID: %s\n", ctx.SessionID)
	fmt.Printf("✓ Verification: %v\n", ctx.Verification)
	fmt.Printf("✓ Latency: %v\n", ctx.Latency)

	if ctx.Error != nil {
		fmt.Printf("✗ Error: %v\n", ctx.Error)
	}
}

func batchGenerationExample(m *CaptchaMiddleware) {
	fmt.Println("[Batch Generation Example]")
	fmt.Println("-----------------------------------")

	count := 20
	start := time.Now()

	results, err := m.BatchGenerate(count)
	duration := time.Since(start)

	if err != nil {
		fmt.Printf("✗ Batch generation failed: %v\n", err)
		return
	}

	successCount := 0
	for _, ctx := range results {
		if ctx.Error == nil {
			successCount++
		}
	}

	fmt.Printf("✓ Generated %d/%d captchas in %v\n", successCount, count, duration)
	fmt.Printf("  Average latency: %v\n", duration/time.Duration(count))

	stats := m.client.GetStats()
	fmt.Printf("  Client stats - Total: %d, Success: %d, Failed: %d\n",
		stats.TotalRequests,
		stats.SuccessfulRequests,
		stats.FailedRequests,
	)
}

func performanceTestExample(m *CaptchaMiddleware) {
	fmt.Println("[Performance Test Example]")
	fmt.Println("-----------------------------------")

	m.PerformanceTest(10 * time.Second)
}
