package performance

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hjtpx/hjtpx/pkg/jwt"
	"github.com/hjtpx/hjtpx/pkg/response"
	"github.com/stretchr/testify/assert"
)

func BenchmarkHealthEndpoint(b *testing.B) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/health", func(c *gin.Context) {
		response.Success(c, gin.H{"status": "healthy"})
	})

	req, _ := http.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.ServeHTTP(w, req)
	}
}

func BenchmarkJWTGeneration(b *testing.B) {
	jwt.InitJWT("benchmark-secret-key")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = jwt.GenerateToken(uint(i), "testuser")
	}
}

func BenchmarkJWTValidation(b *testing.B) {
	jwt.InitJWT("benchmark-secret-key")
	token, _ := jwt.GenerateToken(1, "testuser")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = jwt.ParseToken(token)
	}
}

func BenchmarkConcurrentJWTGeneration(b *testing.B) {
	jwt.InitJWT("concurrent-secret-key")

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			_, _ = jwt.GenerateToken(uint(i), fmt.Sprintf("user%d", i))
			i++
		}
	})
}

func BenchmarkConcurrentJWTValidation(b *testing.B) {
	jwt.InitJWT("concurrent-secret-key")
	tokens := make([]string, 1000)
	for i := 0; i < 1000; i++ {
		tokens[i], _ = jwt.GenerateToken(uint(i), fmt.Sprintf("user%d", i))
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			_, _ = jwt.ParseToken(tokens[i%1000])
			i++
		}
	})
}

func BenchmarkJSONMarshalling(b *testing.B) {
	data := map[string]interface{}{
		"id":       1,
		"username": "testuser",
		"email":    "test@example.com",
		"status":   "active",
		"metadata": map[string]string{
			"key1": "value1",
			"key2": "value2",
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = json.Marshal(data)
	}
}

func BenchmarkJSONUnmarshalling(b *testing.B) {
	jsonData := `{"id":1,"username":"testuser","email":"test@example.com","status":"active"}`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var data map[string]interface{}
		_ = json.Unmarshal([]byte(jsonData), &data)
	}
}

func BenchmarkConcurrentHTTPRequests(b *testing.B) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/test", func(c *gin.Context) {
		response.Success(c, gin.H{"message": "ok"})
	})

	var wg sync.WaitGroup
	requests := 1000

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			req, _ := http.NewRequest("GET", "/test", nil)
			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, req)
		}
	})
	b.StopTimer()

	_ = wg
	_ = requests
}

func BenchmarkResponseCreation(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		response.Success(nil, gin.H{"key": "value"})
	}
}

func BenchmarkMutexLock(b *testing.B) {
	var mu sync.Mutex
	counter := 0

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mu.Lock()
		counter++
		mu.Unlock()
	}
}

func BenchmarkRWMutexRead(b *testing.B) {
	var mu sync.RWMutex
	data := make(map[string]string)
	for i := 0; i < 100; i++ {
		data[fmt.Sprintf("key%d", i)] = fmt.Sprintf("value%d", i)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mu.RLock()
		_ = data["key50"]
		mu.RUnlock()
	}
}

func BenchmarkRWMutexWrite(b *testing.B) {
	var mu sync.RWMutex
	data := make(map[string]string)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mu.Lock()
		data[fmt.Sprintf("key%d", i)] = fmt.Sprintf("value%d", i)
		mu.Unlock()
	}
}

func BenchmarkChannelSend(b *testing.B) {
	ch := make(chan int, 100)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		select {
		case ch <- i:
		default:
		}
	}
}

func BenchmarkChannelReceive(b *testing.B) {
	ch := make(chan int, 100)
	for i := 0; i < 100; i++ {
		ch <- i
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		select {
		case <-ch:
		default:
		}
	}
}

func BenchmarkMapAccess(b *testing.B) {
	data := make(map[string]string)
	for i := 0; i < 100; i++ {
		data[fmt.Sprintf("key%d", i)] = fmt.Sprintf("value%d", i)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = data["key50"]
	}
}

func BenchmarkSliceAppend(b *testing.B) {
	var slice []int

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		slice = append(slice, i)
	}
}

func BenchmarkStringConcatenation(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = "prefix" + fmt.Sprintf("%d", i) + "suffix"
	}
}

func BenchmarkBytesBuffer(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf := bytes.NewBuffer(nil)
		buf.WriteString("prefix")
		buf.WriteString(fmt.Sprintf("%d", i))
		buf.WriteString("suffix")
		_ = buf.String()
	}
}

func BenchmarkTimeNow(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = time.Now()
	}
}

func BenchmarkTimeUnix(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = time.Now().Unix()
	}
}

func BenchmarkSleep(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		time.Sleep(1 * time.Microsecond)
	}
}

func TestPerformanceMetrics(t *testing.T) {
	start := time.Now()
	
	for i := 0; i < 1000; i++ {
		jwt.InitJWT(fmt.Sprintf("secret-%d", i))
		_, _ = jwt.GenerateToken(1, "testuser")
	}
	
	elapsed := time.Since(start)
	avgLatency := elapsed / 1000
	
	assert.Less(t, avgLatency, 1*time.Millisecond, "Average JWT generation should be under 1ms")
}

func TestConcurrentSafety(t *testing.T) {
	jwt.InitJWT("concurrent-test-secret")
	
	var wg sync.WaitGroup
	errors := make(chan error, 100)
	
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				token, err := jwt.GenerateToken(uint(id), fmt.Sprintf("user%d", id))
				if err != nil {
					errors <- err
					return
				}
				_, err = jwt.ParseToken(token)
				if err != nil {
					errors <- err
					return
				}
			}
		}(i)
	}
	
	wg.Wait()
	close(errors)
	
	for err := range errors {
		t.Errorf("Concurrent operation failed: %v", err)
	}
}

func TestMemoryUsage(t *testing.T) {
	var memBefore, memAfter runtime.MemStats
	runtime.ReadMemStats(&memBefore)
	
	jwt.InitJWT("memory-test-secret")
	for i := 0; i < 10000; i++ {
		_, _ = jwt.GenerateToken(1, "testuser")
	}
	
	runtime.ReadMemStats(&memAfter)
	
	allocDiff := memAfter.Alloc - memBefore.Alloc
	assert.Less(t, allocDiff, uint64(100*1024*1024), "Memory allocation should be reasonable")
}
