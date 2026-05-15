package captchax

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func BenchmarkExtendedCaptchaGeneration(b *testing.B) {
	benchmarks := []struct {
		name    string
		captcha func(*Client, context.Context) (interface{}, error)
	}{
		{"Slider", func(c *Client, ctx context.Context) (interface{}, error) { return c.GenerateSliderCaptcha(ctx, nil) }},
		{"Click", func(c *Client, ctx context.Context) (interface{}, error) { return c.GenerateClickCaptcha(ctx, nil) }},
		{"Puzzle", func(c *Client, ctx context.Context) (interface{}, error) { return c.GeneratePuzzleCaptcha(ctx, nil) }},
		{"Text", func(c *Client, ctx context.Context) (interface{}, error) { return c.GenerateTextCaptcha(ctx, nil) }},
		{"Icon", func(c *Client, ctx context.Context) (interface{}, error) { return c.GenerateIconCaptcha(ctx, nil) }},
		{"Rotate", func(c *Client, ctx context.Context) (interface{}, error) { return c.GenerateRotateCaptcha(ctx, nil) }},
	}

	for _, bm := range benchmarks {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(APIResponse{
				Code:    0,
				Message: "success",
				Data: map[string]interface{}{
					"id":        "test-captcha",
					"image":     "base64data",
					"target_x":  150,
					"target_y":  75,
				},
			})
		}))
		defer server.Close()

		client, _ := NewClient(NewConfig(server.URL).WithAppID("test-app"))
		ctx := context.Background()

		b.Run(bm.name, func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, _ = bm.captcha(client, ctx)
			}
		})
	}
}

func BenchmarkExtendedCaptchaVerification(b *testing.B) {
	benchmarks := []struct {
		name       string
		verifyFunc func(*Client, context.Context) (interface{}, error)
	}{
		{"Slider", func(c *Client, ctx context.Context) (interface{}, error) {
			targetY := 80
			return c.VerifySliderCaptcha(ctx, "captcha-id", 150, &targetY)
		}},
		{"Click", func(c *Client, ctx context.Context) (interface{}, error) {
			return c.VerifyClickCaptcha(ctx, "captcha-id", []CharPosition{{Char: "A", X: 100, Y: 100}})
		}},
		{"Puzzle", func(c *Client, ctx context.Context) (interface{}, error) {
			targetY := 80
			return c.VerifyPuzzleCaptcha(ctx, "captcha-id", 150, &targetY)
		}},
		{"Text", func(c *Client, ctx context.Context) (interface{}, error) {
			return c.VerifyTextCaptcha(ctx, "captcha-id", "ABC123")
		}},
		{"Icon", func(c *Client, ctx context.Context) (interface{}, error) {
			return c.VerifyIconCaptcha(ctx, "captcha-id", "icon-1")
		}},
		{"Rotate", func(c *Client, ctx context.Context) (interface{}, error) {
			return c.VerifyRotateCaptcha(ctx, "captcha-id", 45.0)
		}},
	}

	for _, bm := range benchmarks {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(APIResponse{
				Code:    0,
				Message: "success",
				Data: map[string]interface{}{
					"success": true,
					"score":   0.95,
				},
			})
		}))
		defer server.Close()

		client, _ := NewClient(NewConfig(server.URL).WithAppID("test-app"))
		ctx := context.Background()

		b.Run(bm.name, func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, _ = bm.verifyFunc(client, ctx)
			}
		})
	}
}

func BenchmarkConcurrentStress(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(5 * time.Millisecond)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(APIResponse{
			Code:    0,
			Message: "success",
			Data:    map[string]interface{}{"status": "ok"},
		})
	}))
	defer server.Close()

	client, _ := NewClient(NewConfig(server.URL).WithAppID("test-app"))
	ctx := context.Background()

	concurrencyLevels := []int{1, 5, 10, 50, 100, 500, 1000}

	for _, concurrency := range concurrencyLevels {
		b.Run(fmt.Sprintf("Concurrency%d", concurrency), func(b *testing.B) {
			var wg sync.WaitGroup
			var successCount int64
			var errorCount int64

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					_, err := client.GenerateSliderCaptcha(ctx, nil)
					if err != nil {
						atomic.AddInt64(&errorCount, 1)
					} else {
						atomic.AddInt64(&successCount, 1)
					}
				}()

				if i > 0 && i%concurrency == 0 {
					wg.Wait()
				}
			}
			wg.Wait()
			b.StopTimer()

			fmt.Printf("\nConcurrency %d: Success=%d, Errors=%d\n",
				concurrency, successCount, errorCount)
		})
	}
}

func BenchmarkMemoryAllocation(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(APIResponse{
			Code:    0,
			Message: "success",
			Data: map[string]interface{}{
				"id":             "test-captcha",
				"image":          strings.Repeat("a", 10000),
				"background_b64": strings.Repeat("b", 10000),
				"slider_b64":     strings.Repeat("c", 5000),
			},
		})
	}))
	defer server.Close()

	client, _ := NewClient(NewConfig(server.URL).WithAppID("test-app"))
	ctx := context.Background()

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = client.GenerateSliderCaptcha(ctx, nil)
	}
}

func BenchmarkConnectionReuse(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(APIResponse{
			Code:    0,
			Message: "success",
			Data:    map[string]interface{}{"status": "ok"},
		})
	}))
	defer server.Close()

	httpClient := newHTTPClient(server.URL, 10*time.Second, 0)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = httpClient.get(ctx, "/test")
	}
}

func BenchmarkClientCreation(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		client, _ := NewClient(NewConfig("https://example.com/api").WithAppID("test"))
		if client == nil {
			b.Fatal("client is nil")
		}
	}
}

func BenchmarkConfigBuilding(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		config := NewConfig("https://example.com").
			WithAppID("test").
			WithTimeout(30 * time.Second).
			WithRetryTimes(5).
			WithAPIVersion(APIVersionV2)

		if config == nil {
			b.Fatal("config is nil")
		}
	}
}

func BenchmarkJSONEncoding(b *testing.B) {
	data := map[string]interface{}{
		"app_id":     "test-app",
		"width":      300,
		"height":     150,
		"client_info": "test-info",
		"scenario_id": "scenario-1",
		"extra_data": map[string]interface{}{
			"key1": "value1",
			"key2": 123,
			"key3": true,
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := json.Marshal(data)
		if err != nil {
			b.Fatalf("marshal failed: %v", err)
		}
	}
}

func BenchmarkJSONDecoding(b *testing.B) {
	jsonStr := `{
		"code": 0,
		"message": "success",
		"data": {
			"id": "captcha-123",
			"image": "base64data",
			"target_x": 150,
			"target_y": 75,
			"metadata": {
				"created_at": "2024-01-01T00:00:00Z",
				"expires_at": "2024-01-01T00:05:00Z"
			}
		}
	}`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var resp APIResponse
		err := json.Unmarshal([]byte(jsonStr), &resp)
		if err != nil {
			b.Fatalf("unmarshal failed: %v", err)
		}
	}
}

func BenchmarkBatchOperations(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		var body struct {
			Items []BatchVerifyItem `json:"items"`
		}
		json.NewDecoder(r.Body).Decode(&body)

		results := make([]BatchVerifyResult, len(body.Items))
		for i := range body.Items {
			results[i] = BatchVerifyResult{
				CaptchaID: body.Items[i].CaptchaID,
				Success:   true,
				Message:   "success",
				Score:     0.95,
			}
		}

		json.NewEncoder(w).Encode(APIResponse{
			Code:    0,
			Message: "success",
			Data: BatchVerifyResponse{
				Results: results,
				Summary: BatchVerifySummary{
					Total:   len(body.Items),
					Success: len(body.Items),
					Failed:  0,
				},
			},
		})
	}))
	defer server.Close()

	client, _ := NewClient(NewConfig(server.URL).WithAppID("test-app"))
	ctx := context.Background()

	batchSizes := []int{1, 10, 50, 100, 500, 1000}

	for _, size := range batchSizes {
		b.Run(fmt.Sprintf("BatchSize%d", size), func(b *testing.B) {
			items := make([]BatchVerifyItem, size)
			for i := 0; i < size; i++ {
				items[i] = BatchVerifyItem{
					CaptchaID: fmt.Sprintf("captcha-%d", i),
					Type:      "slider",
					TargetX:   rand.Intn(300),
				}
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, _ = client.BatchVerify(ctx, items, fmt.Sprintf("dedup-%d", i))
			}
		})
	}
}

func BenchmarkRetryMechanism(b *testing.B) {
	successAfter := 3
	requestCount := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requestCount, 1)
		w.Header().Set("Content-Type", "application/json")

		if atomic.LoadInt32(&requestCount) < int32(successAfter) {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(APIResponse{
				Code:    500,
				Message: "internal error",
			})
			return
		}

		json.NewEncoder(w).Encode(APIResponse{
			Code:    0,
			Message: "success",
			Data:    map[string]string{"status": "ok"},
		})
	}))
	defer server.Close()

	httpClient := newHTTPClient(server.URL, 10*time.Second, 5)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		atomic.StoreInt32(&requestCount, 0)
		_, _ = httpClient.get(ctx, "/test")
	}
}

func BenchmarkErrorHandling(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(APIResponse{
			Code:    400,
			Message: "bad request",
		})
	}))
	defer server.Close()

	client, _ := NewClient(NewConfig(server.URL).WithAppID("test-app"))
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := client.GenerateSliderCaptcha(ctx, nil)
		if err == nil {
			b.Fatal("expected error")
		}
	}
}

func BenchmarkTimeoutHandling(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(APIResponse{
			Code:    0,
			Message: "success",
			Data:    map[string]string{"status": "ok"},
		})
	}))
	defer server.Close()

	client, _ := NewClient(NewConfig(server.URL).WithAppID("test-app").WithTimeout(50 * time.Millisecond))
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := client.HealthCheck(ctx)
		if err != nil && !strings.Contains(err.Error(), "timeout") {
			b.Fatalf("unexpected error: %v", err)
		}
	}
}

func BenchmarkCacheHitRate(b *testing.B) {
	cache := make(map[string]*SliderCaptchaResult)
	cacheHits := int64(0)
	cacheMisses := int64(0)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(APIResponse{
			Code:    0,
			Message: "success",
			Data: SliderCaptchaResult{
				ID:            "cached-id",
				BackgroundB64: "background",
				SliderB64:     "slider",
				TargetX:       150,
				TargetY:       75,
			},
		})
	}))
	defer server.Close()

	client, _ := NewClient(NewConfig(server.URL).WithAppID("test-app"))
	ctx := context.Background()

	keys := []string{"key1", "key2", "key3", "key4", "key5"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := keys[i%len(keys)]

		if result, ok := cache[key]; ok {
			atomic.AddInt64(&cacheHits, 1)
			_ = result
		} else {
			atomic.AddInt64(&cacheMisses, 1)
			result, _ := client.GenerateSliderCaptcha(ctx, nil)
			if result != nil {
				cache[key] = result
			}
		}
	}

	hitRate := float64(atomic.LoadInt64(&cacheHits)) / float64(atomic.LoadInt64(&cacheHits)+atomic.LoadInt64(&cacheMisses))
	b.Logf("Cache hit rate: %.2f%%", hitRate*100)
}

func BenchmarkPayloadSize(b *testing.B) {
	sizes := []int{100, 1000, 10000, 100000, 1000000}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("PayloadSize%d", size), func(b *testing.B) {
			payload := strings.Repeat("a", size)

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(APIResponse{
					Code:    0,
					Message: "success",
					Data: map[string]string{"data": payload},
				})
			}))
			defer server.Close()

			client, _ := NewClient(NewConfig(server.URL).WithAppID("test-app"))
			ctx := context.Background()

			type PayloadSize struct {
				Data string `json:"data"`
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, _ = client.httpClient.post(ctx, "/test", PayloadSize{Data: payload}, "")
			}
		})
	}
}

func BenchmarkMutexContention(b *testing.B) {
	var mu sync.RWMutex
	counter := 0

	operations := []int{100, 1000, 10000}

	for _, opCount := range operations {
		b.Run(fmt.Sprintf("Ops%d", opCount), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				var wg sync.WaitGroup
				for j := 0; j < opCount; j++ {
					wg.Add(1)
					go func() {
						defer wg.Done()
						mu.Lock()
						counter++
						mu.Unlock()
					}()
				}
				wg.Wait()
			}
		})
	}
}

func BenchmarkAtomicOperations(b *testing.B) {
	var counter int64

	operations := []int{100, 1000, 10000}

	for _, opCount := range operations {
		b.Run(fmt.Sprintf("Ops%d", opCount), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				var wg sync.WaitGroup
				for j := 0; j < opCount; j++ {
					wg.Add(1)
					go func() {
						defer wg.Done()
						atomic.AddInt64(&counter, 1)
					}()
				}
				wg.Wait()
			}
		})
	}
}

func BenchmarkChannelCommunication(b *testing.B) {
	sizes := []int{10, 100, 1000}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("ChannelSize%d", size), func(b *testing.B) {
			ch := make(chan int, size)

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				var wg sync.WaitGroup
				wg.Add(size)

				for j := 0; j < size; j++ {
					go func(val int) {
						defer wg.Done()
						ch <- val
					}(j)
				}

				go func() {
					for k := 0; k < size; k++ {
						<-ch
					}
				}()

				wg.Wait()
			}
		})
	}
}

func BenchmarkSliceOperations(b *testing.B) {
	sizes := []int{10, 100, 1000, 10000}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("SliceSize%d", size), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				slice := make([]int, 0, size)
				for j := 0; j < size; j++ {
					slice = append(slice, j)
				}
				_ = slice
			}
		})
	}
}

func BenchmarkMapOperations(b *testing.B) {
	sizes := []int{10, 100, 1000}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("MapSize%d", size), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				m := make(map[string]int, size)
				for j := 0; j < size; j++ {
					m[fmt.Sprintf("key-%d", j)] = j
				}
				_ = m
			}
		})
	}
}

func BenchmarkStringManipulation(b *testing.B) {
	operations := []struct {
		name string
		fn   func() string
	}{
		{"Concat", func() string { return "a" + "b" + "c" + "d" }},
		{"Sprintf", func() string { return fmt.Sprintf("%s-%s-%s", "a", "b", "c") }},
		{"Builder", func() string {
			var sb strings.Builder
			sb.WriteString("a")
			sb.WriteString("-")
			sb.WriteString("b")
			sb.WriteString("-")
			sb.WriteString("c")
			return sb.String()
		}},
		{"Join", func() string { return strings.Join([]string{"a", "b", "c"}, "-") }},
	}

	for _, op := range operations {
		b.Run(op.name, func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				result := op.fn()
				if result == "" {
					b.Fatal("empty result")
				}
			}
		})
	}
}

func BenchmarkHTTPMethods(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(APIResponse{
			Code:    0,
			Message: "success",
			Data:    map[string]string{"method": r.Method},
		})
	}))
	defer server.Close()

	httpClient := newHTTPClient(server.URL, 10*time.Second, 0)
	ctx := context.Background()

	b.Run("GET", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = httpClient.get(ctx, "/test")
		}
	})

	b.Run("POST", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = httpClient.post(ctx, "/test", map[string]interface{}{"key": "value"}, "")
		}
	})

	b.Run("PUT", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = httpClient.put(ctx, "/test", map[string]interface{}{"key": "value"})
		}
	})

	b.Run("DELETE", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = httpClient.delete(ctx, "/test")
		}
	})
}

func BenchmarkValidation(b *testing.B) {
	validEmail := "test@example.com"
	invalidEmail := "invalid-email"

	validators := []struct {
		name string
		fn   func(string) bool
	}{
		{"SimpleContains", func(s string) bool { return strings.Contains(s, "@") }},
		{"SimpleRegex", func(s string) bool {
			atIndex := strings.Index(s, "@")
			dotIndex := strings.LastIndex(s, ".")
			return atIndex > 0 && dotIndex > atIndex+1 && dotIndex < len(s)-1
		}},
		{"LengthCheck", func(s string) bool {
			return len(s) >= 5 && len(s) <= 100
		}},
	}

	for _, v := range validators {
		b.Run(v.name, func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = v.fn(validEmail)
				_ = v.fn(invalidEmail)
			}
		})
	}
}

func BenchmarkMathOperations(b *testing.B) {
	operations := []struct {
		name string
		fn   func(float64) float64
	}{
		{"Sqrt", math.Sqrt},
		{"Sin", math.Sin},
		{"Cos", math.Cos},
		{"Tan", math.Tan},
		{"Log", math.Log},
		{"Exp", math.Exp},
		{"Pow", func(x float64) float64 { return math.Pow(x, 2) }},
		{"Abs", math.Abs},
	}

	for _, op := range operations {
		b.Run(op.name, func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				result := op.fn(float64(i))
				if math.IsNaN(result) {
					b.Fatal("NaN result")
				}
			}
		})
	}
}

func BenchmarkTimeOperations(b *testing.B) {
	timestamps := []int64{1640000000, 1650000000, 1660000000}

	b.Run("Now", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = time.Now()
		}
	})

	b.Run("Unix", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = time.Unix(timestamps[i%len(timestamps)], 0)
		}
	})

	b.Run("Parse", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = time.Parse(time.RFC3339, "2024-01-01T00:00:00Z")
		}
	})

	b.Run("Format", func(b *testing.B) {
		t := time.Now()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = t.Format(time.RFC3339)
		}
	})
}

func BenchmarkRandomGeneration(b *testing.B) {
	b.Run("MathRand", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = rand.Intn(1000)
		}
	})

	b.Run("CryptoRand", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			buf := make([]byte, 8)
			rand.Read(buf)
		}
	})
}

func BenchmarkInterfaceToInterface(b *testing.B) {
	var iface interface{}

	b.Run("SetNil", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			iface = nil
		}
	})

	b.Run("SetInt", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			iface = i
		}
	})

	b.Run("SetString", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			iface = fmt.Sprintf("value-%d", i)
		}
	})

	b.Run("SetStruct", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			iface = struct{ Value int }{Value: i}
		}
	})
}
