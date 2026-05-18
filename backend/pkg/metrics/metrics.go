package metrics

import (
	"sync/atomic"
	"time"
)

var (
	startTime    = time.Now()
	requestCount uint64
	successCount uint64
	failureCount uint64
)

func init() {
	go collectBasicMetrics()
}

func collectBasicMetrics() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		if atomic.LoadUint64(&requestCount) > 0 {
			atomic.StoreUint64(&requestCount, atomic.LoadUint64(&requestCount)/2)
		}
	}
}

func IncrementRequestCount() {
	atomic.AddUint64(&requestCount, 1)
}

func IncrementSuccessCount() {
	atomic.AddUint64(&successCount, 1)
}

func IncrementFailureCount() {
	atomic.AddUint64(&failureCount, 1)
}

func GetRequestCount() uint64 {
	return atomic.LoadUint64(&requestCount)
}

func GetSuccessCount() uint64 {
	return atomic.LoadUint64(&successCount)
}

func GetFailureCount() uint64 {
	return atomic.LoadUint64(&failureCount)
}

func GetSuccessRate() float64 {
	total := atomic.LoadUint64(&requestCount)
	if total == 0 {
		return 100.0
	}
	success := atomic.LoadUint64(&successCount)
	return float64(success) / float64(total) * 100
}

func GetUptime() time.Duration {
	return time.Since(startTime)
}

func ResetMetrics() {
	atomic.StoreUint64(&requestCount, 0)
	atomic.StoreUint64(&successCount, 0)
	atomic.StoreUint64(&failureCount, 0)
}
