package performance

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hjtpx/hjtpx/internal/api/handler"
	"github.com/hjtpx/hjtpx/pkg/database"
	"github.com/hjtpx/hjtpx/pkg/models"
	"github.com/hjtpx/hjtpx/pkg/redis"
	"github.com/stretchr/testify/assert"
)

func setupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/api/captcha/slider", handler.GetSliderCaptcha)
	r.POST("/api/captcha/click", handler.GetClickCaptcha)
	r.POST("/api/captcha/verify", handler.VerifyCaptcha)
	return r
}

func TestConcurrentVerifications(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	t.Run("ConcurrentSliderCaptchaGeneration", func(t *testing.T) {
		concurrency := 100
		duration := measureTime(func() {
			var wg sync.WaitGroup
			wg.Add(concurrency)

			for i := 0; i < concurrency; i++ {
				go func() {
					defer wg.Done()
					router := setupTestRouter()
					resp := httptest.NewRecorder()
					req, _ := http.NewRequest("POST", "/api/captcha/slider", nil)
					router.ServeHTTP(resp, req)
					assert.Equal(t, http.StatusOK, resp.Code)
				}()
			}

			wg.Wait()
		})

		t.Logf("100并发滑块验证码生成完成，耗时: %v", duration)
		assert.Less(t, duration, 10*time.Second, "100并发请求应在10秒内完成")
	})

	t.Run("ConcurrentClickCaptchaGeneration", func(t *testing.T) {
		concurrency := 50
		duration := measureTime(func() {
			var wg sync.WaitGroup
			wg.Add(concurrency)

			for i := 0; i < concurrency; i++ {
				go func() {
					defer wg.Done()
					router := setupTestRouter()
					resp := httptest.NewRecorder()
					req, _ := http.NewRequest("POST", "/api/captcha/click", nil)
					router.ServeHTTP(resp, req)
					assert.Equal(t, http.StatusOK, resp.Code)
				}()
			}

			wg.Wait()
		})

		t.Logf("50并发点选验证码生成完成，耗时: %v", duration)
		assert.Less(t, duration, 5*time.Second, "50并发请求应在5秒内完成")
	})

	t.Run("ConcurrentVerificationRequests", func(t *testing.T) {
		concurrency := 50
		var successCount int64
		var failCount int64

		router := setupTestRouter()

		sessionIDs := make([]string, concurrency)
		for i := 0; i < concurrency; i++ {
			resp := httptest.NewRecorder()
			req, _ := http.NewRequest("POST", "/api/captcha/slider", nil)
			router.ServeHTTP(resp, req)

			var result map[string]interface{}
			json.Unmarshal(resp.Body.Bytes(), &result)
			sessionIDs[i] = result["session_id"].(string)
		}

		duration := measureTime(func() {
			var wg sync.WaitGroup
			wg.Add(concurrency)

			for i := 0; i < concurrency; i++ {
				go func(idx int) {
					defer wg.Done()
					sessionID := sessionIDs[idx]

					verifyReq := map[string]interface{}{
						"session_id": sessionID,
						"type":       "slider",
						"x":          150,
						"y":          100,
					}
					verifyJSON, _ := json.Marshal(verifyReq)

					resp := httptest.NewRecorder()
					req, _ := http.NewRequest("POST", "/api/captcha/verify", bytes.NewBuffer(verifyJSON))
					req.Header.Set("Content-Type", "application/json")
					router.ServeHTTP(resp, req)

					if resp.Code == http.StatusOK {
						atomic.AddInt64(&successCount, 1)
					} else {
						atomic.AddInt64(&failCount, 1)
					}
				}(i)
			}

			wg.Wait()
		})

		totalRequests := int64(concurrency)
		successRate := float64(successCount) / float64(totalRequests) * 100
		avgResponseTime := float64(duration.Milliseconds()) / float64(totalRequests)

		t.Logf("并发验证测试结果:")
		t.Logf("  总请求数: %d", totalRequests)
		t.Logf("  成功: %d", successCount)
		t.Logf("  失败: %d", failCount)
		t.Logf("  成功率: %.2f%%", successRate)
		t.Logf("  总耗时: %v", duration)
		t.Logf("  平均响应时间: %.2fms", avgResponseTime)

		assert.Greater(t, successCount, int64(0), "至少应有部分请求成功")
	})

	t.Run("MixedConcurrentRequests", func(t *testing.T) {
		concurrency := 30
		var wg sync.WaitGroup
		wg.Add(concurrency * 2)

		for i := 0; i < concurrency; i++ {
			go func() {
				defer wg.Done()
				router := setupTestRouter()
				resp := httptest.NewRecorder()
				req, _ := http.NewRequest("POST", "/api/captcha/slider", nil)
				router.ServeHTTP(resp, req)
			}()
		}

		for i := 0; i < concurrency; i++ {
			go func() {
				defer wg.Done()
				router := setupTestRouter()
				resp := httptest.NewRecorder()
				req, _ := http.NewRequest("POST", "/api/captcha/click", nil)
				router.ServeHTTP(resp, req)
			}()
		}

		wg.Wait()
		t.Log("混合并发请求测试完成")
	})
}

func TestDatabasePerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping database performance test in short mode")
	}

	db := database.GetDB()
	if db == nil {
		t.Skip("Database not available")
	}

	t.Run("LargeDataVolumeQuery", func(t *testing.T) {
		t.Log("测试大数据量查询性能")

		before := time.Now()
		var logs []models.VerificationLog
		err := db.Limit(1000).Find(&logs).Error
		duration := time.Since(before)

		t.Logf("查询1000条日志记录耗时: %v", duration)
		assert.NoError(t, err)
		assert.Less(t, duration, 2*time.Second, "查询1000条记录应在2秒内完成")
	})

	t.Run("IndexPerformance", func(t *testing.T) {
		t.Log("测试索引性能")

		before := time.Now()
		var count int64
		err := db.Model(&models.VerificationLog{}).
			Where("created_at > ?", time.Now().AddDate(0, 0, -7)).
			Count(&count).Error
		duration := time.Since(before)

		t.Logf("带索引的查询耗时: %v", duration)
		assert.NoError(t, err)
		assert.Less(t, duration, 500*time.Millisecond, "索引查询应在500ms内完成")
	})

	t.Run("BatchInsertPerformance", func(t *testing.T) {
		t.Log("测试批量插入性能")

		testLogs := make([]models.VerificationLog, 100)
		for i := 0; i < 100; i++ {
			testLogs[i] = models.VerificationLog{
				SessionID:    fmt.Sprintf("perf_test_%d_%d", time.Now().UnixNano(), i),
				ApplicationID: 1,
				CaptchaType:  "slider",
				Status:       "success",
				IPAddress:   "127.0.0.1",
				RiskScore:    0.1,
				Duration:    100,
			}
		}

		before := time.Now()
		err := db.CreateInBatches(testLogs, 50).Error
		duration := time.Since(before)

		t.Logf("批量插入100条记录耗时: %v", duration)
		assert.NoError(t, err)
		assert.Less(t, duration, 3*time.Second, "批量插入应在3秒内完成")

		db.Where("session_id LIKE ?", "perf_test_%").Delete(&models.VerificationLog{})
	})

	t.Run("ComplexQueryPerformance", func(t *testing.T) {
		t.Log("测试复杂查询性能")

		before := time.Now()
		var result struct {
			TotalCount    int64
			SuccessCount  int64
			FailCount     int64
			AvgRiskScore  float64
		}

		err := db.Model(&models.VerificationLog{}).
			Select("COUNT(*) as total_count, SUM(CASE WHEN status = 'success' THEN 1 ELSE 0 END) as success_count, SUM(CASE WHEN status = 'failed' THEN 1 ELSE 0 END) as fail_count, AVG(risk_score) as avg_risk_score").
			Where("created_at > ?", time.Now().AddDate(0, 0, -30)).
			Scan(&result).Error
		duration := time.Since(before)

		t.Logf("复杂聚合查询耗时: %v", duration)
		t.Logf("查询结果: 总数=%d, 成功=%d, 失败=%d, 平均风险=%.2f",
			result.TotalCount, result.SuccessCount, result.FailCount, result.AvgRiskScore)
		assert.NoError(t, err)
		assert.Less(t, duration, 1*time.Second, "复杂查询应在1秒内完成")
	})
}

func TestCachePerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping cache performance test in short mode")
	}

	client := redis.GetClient()
	if client == nil {
		t.Skip("Redis not available")
	}

	ctx := redis.Context

	t.Run("RedisSetGetPerformance", func(t *testing.T) {
		t.Log("测试Redis Set/Get性能")

		key := "test:perf:setget"
		value := "performance_test_data"

		before := time.Now()
		for i := 0; i < 1000; i++ {
			client.Set(ctx, key, value, 5*time.Minute)
			client.Get(ctx, key)
		}
		duration := time.Since(before)

		t.Logf("1000次Set/Get操作耗时: %v", duration)
		t.Logf("平均每次操作: %.3fms", float64(duration.Milliseconds())/2000)
		assert.Less(t, duration, 5*time.Second, "1000次操作应在5秒内完成")

		client.Del(ctx, key)
	})

	t.Run("RedisBatchOperations", func(t *testing.T) {
		t.Log("测试Redis批量操作")

		pipe := client.Pipeline()
		for i := 0; i < 100; i++ {
			key := fmt.Sprintf("test:perf:batch:%d", i)
			pipe.Set(ctx, key, fmt.Sprintf("value_%d", i), 5*time.Minute)
		}

		before := time.Now()
		_, err := pipe.Exec(ctx)
		duration := time.Since(before)

		t.Logf("100次批量写入耗时: %v", duration)
		assert.NoError(t, err)
		assert.Less(t, duration, 1*time.Second, "批量写入应在1秒内完成")

		for i := 0; i < 100; i++ {
			key := fmt.Sprintf("test:perf:batch:%d", i)
			client.Del(ctx, key)
		}
	})

	t.Run("CacheHitRate", func(t *testing.T) {
		t.Log("测试缓存命中率")

		key := "test:perf:hitrate"
		client.Set(ctx, key, "test_value", 10*time.Minute)

		hits := 0
		misses := 0
		iterations := 100

		for i := 0; i < iterations; i++ {
			_, err := client.Get(ctx, key).Result()
			if err == nil {
				hits++
			} else {
				misses++
			}
		}

		hitRate := float64(hits) / float64(iterations) * 100
		t.Logf("缓存命中率: %.2f%% (命中: %d, 未命中: %d)", hitRate, hits, misses)
		assert.GreaterOrEqual(t, hitRate, 95.0, "缓存命中率应>=95%%")

		client.Del(ctx, key)
	})

	t.Run("CacheExpiration", func(t *testing.T) {
		t.Log("测试缓存过期机制")

		key := "test:perf:expiry"
		client.Set(ctx, key, "temp_value", 1*time.Second)

		time.Sleep(100 * time.Millisecond)
		val1, _ := client.Get(ctx, key).Result()
		assert.Equal(t, "temp_value", val1)

		time.Sleep(2 * time.Second)
		_, err := client.Get(ctx, key).Result()
		assert.Error(t, err, "缓存应该已过期")
	})
}

func TestMemoryUsage(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping memory usage test in short mode")
	}

	t.Run("MemoryLeakDetection", func(t *testing.T) {
		t.Log("检测内存泄漏")

		runtime.GC()
		var m1 runtime.MemStats
		runtime.ReadMemStats(&m1)

		router := setupTestRouter()
		for i := 0; i < 1000; i++ {
			resp := httptest.NewRecorder()
			req, _ := http.NewRequest("POST", "/api/captcha/slider", nil)
			router.ServeHTTP(resp, req)
		}

		runtime.GC()
		var m2 runtime.MemStats
		runtime.ReadMemStats(&m2)

		allocDiff := m2.Alloc - m1.Alloc
		t.Logf("初始内存: %d KB", m1.Alloc/1024)
		t.Logf("操作后内存: %d KB", m2.Alloc/1024)
		t.Logf("内存增长: %d KB", allocDiff/1024)

		assert.Less(t, allocDiff, int64(100*1024*1024), "1000次操作内存增长应<100MB")
	})

	t.Run("MemoryUsageUnderLoad", func(t *testing.T) {
		t.Log("负载下的内存使用")

		var initialMem runtime.MemStats
		runtime.ReadMemStats(&initialMem)

		var wg sync.WaitGroup
		concurrency := 50
		wg.Add(concurrency)

		for i := 0; i < concurrency; i++ {
			go func() {
				defer wg.Done()
				router := setupTestRouter()
				for j := 0; j < 20; j++ {
					resp := httptest.NewRecorder()
					req, _ := http.NewRequest("POST", "/api/captcha/slider", nil)
					router.ServeHTTP(resp, req)
				}
			}()
		}

		wg.Wait()

		runtime.GC()
		var finalMem runtime.MemStats
		runtime.ReadMemStats(&finalMem)

		totalRequests := concurrency * 20
		memPerRequest := (finalMem.Alloc - initialMem.Alloc) / uint64(totalRequests)

		t.Logf("总请求数: %d", totalRequests)
		t.Logf("每次请求平均内存: %d bytes", memPerRequest)
		t.Logf("峰值内存: %d KB", finalMem.Alloc/1024)

		assert.Less(t, finalMem.Alloc-initialMem.Alloc, int64(500*1024*1024), "负载下内存增长应<500MB")
	})

	t.Run("GoroutineLeakDetection", func(t *testing.T) {
		t.Log("检测Goroutine泄漏")

		initialGoroutines := runtime.NumGoroutine()

		router := setupTestRouter()
		for i := 0; i < 100; i++ {
			resp := httptest.NewRecorder()
			req, _ := http.NewRequest("POST", "/api/captcha/slider", nil)
			router.ServeHTTP(resp, req)
		}

		time.Sleep(100 * time.Millisecond)

		finalGoroutines := runtime.NumGoroutine()
		goroutineDiff := finalGoroutines - initialGoroutines

		t.Logf("初始Goroutines: %d", initialGoroutines)
		t.Logf("操作后Goroutines: %d", finalGoroutines)
		t.Logf("Goroutine增长: %d", goroutineDiff)

		assert.Less(t, goroutineDiff, 10, "Goroutine增长应<10")
	})
}

func TestResponseTime(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping response time test in short mode")
	}

	t.Run("CaptchaGenerationResponseTime", func(t *testing.T) {
		router := setupTestRouter()

		iterations := 100
		var totalDuration time.Duration

		for i := 0; i < iterations; i++ {
			before := time.Now()
			resp := httptest.NewRecorder()
			req, _ := http.NewRequest("POST", "/api/captcha/slider", nil)
			router.ServeHTTP(resp, req)
			totalDuration += time.Since(before)

			assert.Equal(t, http.StatusOK, resp.Code)
		}

		avgDuration := totalDuration / time.Duration(iterations)
		t.Logf("验证码生成平均响应时间: %v", avgDuration)

		p99Duration := avgDuration * 150 / 100
		t.Logf("预期P99响应时间: %v", p99Duration)
	})

	t.Run("VerificationResponseTime", func(t *testing.T) {
		router := setupTestRouter()

		resp := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/captcha/slider", nil)
		router.ServeHTTP(resp, req)

		var result map[string]interface{}
		json.Unmarshal(resp.Body.Bytes(), &result)
		sessionID := result["session_id"].(string)

		iterations := 50
		var totalDuration time.Duration

		for i := 0; i < iterations; i++ {
			verifyReq := map[string]interface{}{
				"session_id": sessionID,
				"type":       "slider",
				"x":          150,
				"y":          100,
			}
			verifyJSON, _ := json.Marshal(verifyReq)

			before := time.Now()
			verifyResp := httptest.NewRecorder()
			req, _ := http.NewRequest("POST", "/api/captcha/verify", bytes.NewBuffer(verifyJSON))
			req.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(verifyResp, req)
			totalDuration += time.Since(before)
		}

		avgDuration := totalDuration / time.Duration(iterations)
		t.Logf("验证平均响应时间: %v", avgDuration)
	})
}

func measureTime(f func()) time.Duration {
	before := time.Now()
	f()
	return time.Since(before)
}
