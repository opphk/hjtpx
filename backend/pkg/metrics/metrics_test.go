package metrics

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestIncrementRequestCount(t *testing.T) {
	ResetMetrics()
	initial := GetRequestCount()
	IncrementRequestCount()
	current := GetRequestCount()
	assert.Equal(t, initial+1, current)
}

func TestIncrementSuccessCount(t *testing.T) {
	ResetMetrics()
	initial := GetSuccessCount()
	IncrementSuccessCount()
	current := GetSuccessCount()
	assert.Equal(t, initial+1, current)
}

func TestIncrementFailureCount(t *testing.T) {
	ResetMetrics()
	initial := GetFailureCount()
	IncrementFailureCount()
	current := GetFailureCount()
	assert.Equal(t, initial+1, current)
}

func TestGetSuccessRate(t *testing.T) {
	ResetMetrics()

	// Test with no requests
	assert.Equal(t, 100.0, GetSuccessRate())

	// Test with all successes
	IncrementRequestCount()
	IncrementSuccessCount()
	IncrementRequestCount()
	IncrementSuccessCount()
	assert.Equal(t, 100.0, GetSuccessRate())

	// Test with mixed results
	ResetMetrics()
	IncrementRequestCount()
	IncrementSuccessCount()
	IncrementRequestCount()
	IncrementFailureCount()
	assert.Equal(t, 50.0, GetSuccessRate())
}

func TestGetUptime(t *testing.T) {
	uptime := GetUptime()
	assert.True(t, uptime >= 0)

	// Wait a little to ensure it increases
	time.Sleep(1 * time.Millisecond)
	newUptime := GetUptime()
	assert.True(t, newUptime > uptime)
}

func TestResetMetrics(t *testing.T) {
	ResetMetrics()
	IncrementRequestCount()
	IncrementSuccessCount()
	IncrementFailureCount()

	assert.Greater(t, GetRequestCount(), uint64(0))
	assert.Greater(t, GetSuccessCount(), uint64(0))
	assert.Greater(t, GetFailureCount(), uint64(0))

	ResetMetrics()

	assert.Equal(t, uint64(0), GetRequestCount())
	assert.Equal(t, uint64(0), GetSuccessCount())
	assert.Equal(t, uint64(0), GetFailureCount())
}
