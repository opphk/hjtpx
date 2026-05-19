package main

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	captchago "github.com/hjtpx/hjtpx/sdk/go"
)

type CaptchaService struct {
	client *captchago.CaptchaClient
	pool   *sync.Pool
}

func NewCaptchaService(baseURL, appID, appSecret string) *CaptchaService {
	cfg := &captchago.Config{
		BaseURL:         baseURL,
		MaxIdleConns:    20,
		MaxOpenConns:    100,
		ConnMaxLifetime: 30 * time.Minute,
		HTTPTimeout:     10 * time.Second,
		MaxRetries:      3,
		RetryDelay:      100 * time.Millisecond,
	}

	client := captchago.NewCaptchaClient(appID, appSecret, cfg)

	pool := &sync.Pool{
		New: func() interface{} {
			return &CaptchaRequest{
				CreatedAt: time.Now(),
			}
		},
	}

	return &CaptchaService{
		client: client,
		pool:   pool,
	}
}

type CaptchaRequest struct {
	Type      string
	SessionID string
	Answer    interface{}
	CreatedAt time.Time
}

func (s *CaptchaService) GenerateSlider() (*CaptchaRequest, error) {
	slider, err := s.client.GenerateSliderCaptcha()
	if err != nil {
		return nil, err
	}

	req := s.pool.Get().(*CaptchaRequest)
	req.Type = "slider"
	req.SessionID = slider.ChallengeID
	req.CreatedAt = time.Now()

	return req, nil
}

func (s *CaptchaService) GenerateClick() (*CaptchaRequest, error) {
	click, err := s.client.GenerateClickCaptcha()
	if err != nil {
		return nil, err
	}

	req := s.pool.Get().(*CaptchaRequest)
	req.Type = "click"
	req.SessionID = click.ChallengeID
	req.CreatedAt = time.Now()

	return req, nil
}

func (s *CaptchaService) VerifySlider(sessionID, position string) (bool, error) {
	result, err := s.client.VerifySliderCaptcha(sessionID, position)
	if err != nil {
		return false, err
	}

	s.pool.Put(&CaptchaRequest{
		Type:      "slider",
		SessionID: sessionID,
		CreatedAt: time.Now(),
	})

	return result.Success, nil
}

func (s *CaptchaService) GetStats() captchago.PoolStats {
	return s.client.GetStats()
}

func (s *CaptchaService) Close() error {
	return s.client.Close()
}

func main() {
	fmt.Println("======================================")
	fmt.Println("  Microservice Integration Demo")
	fmt.Println("======================================")
	fmt.Println()

	service := NewCaptchaService(
		"http://localhost:8080",
		"app-id",
		"app-secret",
	)
	defer service.Close()

	ctx := context.Background()

	basicUsage(service)
	fmt.Println()

	concurrentLoad(service, 10)
	fmt.Println()

	monitoringExample(service)
	fmt.Println()

	contextTimeoutExample(service)
	fmt.Println()

	fmt.Println("======================================")
	fmt.Println("  Microservice demo completed!")
	fmt.Println("======================================")
}

func basicUsage(service *CaptchaService) {
	fmt.Println("[Basic Usage Example]")
	fmt.Println("-----------------------------------")

	slider, err := service.GenerateSlider()
	if err != nil {
		log.Printf("Failed to generate slider: %v", err)
		return
	}
	fmt.Printf("✓ Generated slider captcha: %s\n", slider.SessionID)
	fmt.Printf("  Type: %s, Created: %s\n", slider.Type, slider.CreatedAt.Format(time.RFC3339))

	click, err := service.GenerateClick()
	if err != nil {
		log.Printf("Failed to generate click: %v", err)
		return
	}
	fmt.Printf("✓ Generated click captcha: %s\n", click.SessionID)
	fmt.Printf("  Type: %s, Created: %s\n", click.Type, click.CreatedAt.Format(time.RFC3339))

	success, err := service.VerifySlider(slider.SessionID, "150")
	if err != nil {
		log.Printf("Failed to verify slider: %v", err)
		return
	}
	fmt.Printf("✓ Verification result: %v\n", success)
}

func concurrentLoad(service *CaptchaService, workers int) {
	fmt.Println(fmt.Sprintf("[Concurrent Load Test - %d workers]", workers))
	fmt.Println("-----------------------------------")

	start := time.Now()

	var wg sync.WaitGroup
	results := make(chan string, workers*3)
	errors := make(chan error, workers*3)

	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			for i := 0; i < 3; i++ {
				req, err := service.GenerateSlider()
				if err != nil {
					errors <- fmt.Errorf("worker %d: %v", id, err)
					continue
				}
				results <- fmt.Sprintf("worker-%d-request-%d: %s", id, i, req.SessionID[:16])

				time.Sleep(10 * time.Millisecond)
			}
		}(w)
	}

	wg.Wait()
	close(results)
	close(errors)

	duration := time.Since(start)

	fmt.Printf("✓ Completed in: %v\n", duration)
	fmt.Printf("✓ Successful requests: %d\n", len(results))
	fmt.Printf("✓ Failed requests: %d\n", len(errors))

	stats := service.GetStats()
	fmt.Printf("  Total requests: %d\n", stats.TotalRequests)
	fmt.Printf("  Success rate: %.2f%%\n", stats.SuccessRate)
}

func monitoringExample(service *CaptchaService) {
	fmt.Println("[Monitoring Example]")
	fmt.Println("-----------------------------------")

	stats := service.GetStats()

	fmt.Printf("Connection Pool Stats:\n")
	fmt.Printf("  Active connections: %d\n", stats.ActiveConnections)
	fmt.Printf("  Idle connections: %d\n", stats.IdleConnections)
	fmt.Printf("  Total requests: %d\n", stats.TotalRequests)
	fmt.Printf("  Successful requests: %d\n", stats.SuccessfulRequests)
	fmt.Printf("  Failed requests: %d\n", stats.FailedRequests)
	fmt.Printf("  Retried requests: %d\n", stats.RetriedRequests)
	fmt.Printf("  Success rate: %.2f%%\n", stats.SuccessRate)

	if stats.LastError != nil {
		fmt.Printf("  Last error: %v\n", stats.LastError)
		fmt.Printf("  Last error time: %s\n", stats.LastErrorTime.Format(time.RFC3339))
	}
}

func contextTimeoutExample(service *CaptchaService) {
	fmt.Println("[Context Timeout Example]")
	fmt.Println("-----------------------------------")

	testCases := []struct {
		name    string
		timeout time.Duration
	}{
		{"Quick timeout", 100 * time.Millisecond},
		{"Normal timeout", 5 * time.Second},
		{"Long timeout", 30 * time.Second},
	}

	for _, tc := range testCases {
		fmt.Printf("\nTesting %s...\n", tc.name)

		ctx, cancel := context.WithTimeout(context.Background(), tc.timeout)

		done := make(chan struct{})
		go func() {
			slider, err := service.client.GenerateSliderCaptchaWithContext(ctx)
			if err != nil {
				if ctx.Err() == context.DeadlineExceeded {
					fmt.Printf("  ✗ Request timed out after %v\n", tc.timeout)
				} else {
					fmt.Printf("  ✗ Error: %v\n", err)
				}
			} else {
				fmt.Printf("  ✓ Success: %s\n", slider.ChallengeID)
			}
			close(done)
		}()

		select {
		case <-done:
		case <-time.After(tc.timeout + 1*time.Second):
			fmt.Printf("  ✗ Test timed out\n")
		}

		cancel()
	}
}

func gracefulShutdownExample() {
	fmt.Println("\n[Graceful Shutdown Example]")
	fmt.Println("-----------------------------------")

	service := NewCaptchaService(
		"http://localhost:8080",
		"app-id",
		"app-secret",
	)

	go func() {
		time.Sleep(10 * time.Second)

		fmt.Println("Shutting down service...")
		if err := service.Close(); err != nil {
			log.Printf("Error during shutdown: %v", err)
		}
		fmt.Println("Service stopped")
	}()

	for i := 0; i < 5; i++ {
		slider, err := service.client.GenerateSliderCaptcha()
		if err != nil {
			log.Printf("Error: %v", err)
			continue
		}
		log.Printf("Generated: %s", slider.ChallengeID)
		time.Sleep(1 * time.Second)
	}
}
