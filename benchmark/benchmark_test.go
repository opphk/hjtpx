package benchmark

import (
	"testing"
)

func BenchmarkCaptchaGeneration(b *testing.B) {
	for i := 0; i < b.N; i++ {
		GenerateTestCaptcha()
	}
}

func BenchmarkCaptchaVerification(b *testing.B) {
	captcha := GenerateTestCaptcha()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		VerifyTestCaptcha(captcha)
	}
}

func BenchmarkSessionCreation(b *testing.B) {
	for i := 0; i < b.N; i++ {
		CreateTestSession()
	}
}

func BenchmarkCacheOperations(b *testing.B) {
	cache := NewTestCache()
	key := "test-key"
	value := "test-value"
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.Set(key, value)
		cache.Get(key)
		cache.Delete(key)
	}
}

func BenchmarkDatabaseQueries(b *testing.B) {
	db := NewTestDatabase()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		db.Query("SELECT * FROM users WHERE id = ?", i%100)
	}
}

func BenchmarkFingerprintGeneration(b *testing.B) {
	for i := 0; i < b.N; i++ {
		GenerateTestFingerprint()
	}
}

func BenchmarkRateLimitCheck(b *testing.B) {
	limiter := NewTestRateLimiter()
	clientID := "test-client"
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		limiter.Check(clientID)
	}
}

func BenchmarkProxyDetection(b *testing.B) {
	detector := NewTestProxyDetector()
	ip := "192.168.1.1"
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		detector.Detect(ip)
	}
}

func BenchmarkBehaviorAnalysis(b *testing.B) {
	analyzer := NewTestBehaviorAnalyzer()
	behavior := GenerateTestBehavior()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		analyzer.Analyze(behavior)
	}
}

func BenchmarkRiskCalculation(b *testing.B) {
	riskCalc := NewTestRiskCalculator()
	features := GenerateTestRiskFeatures()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		riskCalc.Calculate(features)
	}
}

func BenchmarkEncryption(b *testing.B) {
	data := []byte("test data for encryption")
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		EncryptTestData(data)
	}
}

func BenchmarkDecryption(b *testing.B) {
	data := []byte("test data for decryption")
	encrypted := EncryptTestData(data)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		DecryptTestData(encrypted)
	}
}

func BenchmarkTokenGeneration(b *testing.B) {
	for i := 0; i < b.N; i++ {
		GenerateTestToken()
	}
}

func BenchmarkTokenValidation(b *testing.B) {
	token := GenerateTestToken()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ValidateTestToken(token)
	}
}

func BenchmarkJSONSerialization(b *testing.B) {
	data := GenerateTestData()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		SerializeToJSON(data)
	}
}

func BenchmarkJSONDeserialization(b *testing.B) {
	jsonData := []byte(`{"test":"data","number":123,"array":[1,2,3]}`)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		DeserializeFromJSON(jsonData)
	}
}

func BenchmarkConcurrentRequests(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
	 counter := 0
	 for pb.Next() {
		 counter++
		 ProcessTestRequest(counter)
	 }
	})
}

func BenchmarkMemoryAllocation(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		AllocateTestMemory()
	}
}

func BenchmarkGoroutineCreation(b *testing.B) {
	for i := 0; i < b.N; i++ {
		go func() {
			PerformTestTask()
		}()
	}
}
