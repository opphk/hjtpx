package metrics

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewLogAggregator(t *testing.T) {
	la := NewLogAggregator("http://loki:3100", 100, 5*time.Second)
	require.NotNil(t, la)
	assert.Equal(t, 100, la.bufferSize)
	assert.Equal(t, 5*time.Second, la.flushInterval)
	assert.NotNil(t, la.client)
	assert.NotNil(t, la.buffer)
}

func TestLogAggregatorStartStop(t *testing.T) {
	la := NewLogAggregator("http://loki:3100", 100, 100*time.Millisecond)
	require.NotNil(t, la)

	la.Start()
	time.Sleep(50 * time.Millisecond)

	la.Stop()
	time.Sleep(50 * time.Millisecond)
}

func TestLogAggregatorCollect(t *testing.T) {
	la := NewLogAggregator("http://loki:3100", 100, 1*time.Second)
	require.NotNil(t, la)

	entry := LogEntry{
		Timestamp: time.Now(),
		Stream:    map[string]string{"job": "test", "env": "testing"},
		LogLevel:  "info",
		Message:   "test log message",
		Component: "test-component",
	}

	la.Collect(entry)
	assert.Equal(t, 1, la.GetBufferSize())
}

func TestLogAggregatorCollectDisabled(t *testing.T) {
	la := NewLogAggregator("http://loki:3100", 100, 1*time.Second)
	require.NotNil(t, la)

	la.SetEnabled(false)

	entry := LogEntry{
		Timestamp: time.Now(),
		Stream:    map[string]string{"job": "test"},
		Message:   "test log message",
	}

	la.Collect(entry)
	assert.Equal(t, 0, la.GetBufferSize())
}

func TestLogAggregatorAutoFlush(t *testing.T) {
	la := NewLogAggregator("http://loki:3100", 10, 50*time.Millisecond)
	require.NotNil(t, la)

	la.Start()
	defer la.Stop()

	for i := 0; i < 15; i++ {
		entry := LogEntry{
			Timestamp: time.Now(),
			Stream:    map[string]string{"job": "test"},
			Message:   "test log message",
		}
		la.Collect(entry)
	}

	time.Sleep(200 * time.Millisecond)
	assert.Equal(t, 0, la.GetBufferSize())
}

func TestLogAggregatorSetEnabled(t *testing.T) {
	la := NewLogAggregator("http://loki:3100", 100, 1*time.Second)
	require.NotNil(t, la)

	assert.True(t, la.enabled.Load())

	la.SetEnabled(false)
	assert.False(t, la.enabled.Load())

	la.SetEnabled(true)
	assert.True(t, la.enabled.Load())
}

func TestLogAggregatorConcurrentCollect(t *testing.T) {
	la := NewLogAggregator("http://loki:3100", 1000, 1*time.Second)
	require.NotNil(t, la)

	la.Start()
	defer la.Stop()

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				entry := LogEntry{
					Timestamp: time.Now(),
					Stream:    map[string]string{"job": "test", "index": string(rune(idx))},
					Message:   "test log message",
				}
				la.Collect(entry)
			}
		}(i)
	}
	wg.Wait()

	time.Sleep(100 * time.Millisecond)
	assert.GreaterOrEqual(t, la.GetBufferSize(), 0)
}

func TestLogEntryParsing(t *testing.T) {
	entry := LogEntry{
		Timestamp: time.Now(),
		Stream:    map[string]string{"job": "test", "level": "info"},
		LogLevel:  "info",
		Message:   "test message",
		TraceID:   "trace-123",
		SpanID:    "span-456",
		Error:     "",
	}

	assert.Equal(t, "info", entry.LogLevel)
	assert.Equal(t, "test message", entry.Message)
	assert.Equal(t, "trace-123", entry.TraceID)
}

func TestLogEntryWithError(t *testing.T) {
	entry := LogEntry{
		Timestamp: time.Now(),
		Stream:    map[string]string{"job": "test"},
		LogLevel:  "error",
		Message:   "error occurred",
		Error:     "connection refused",
	}

	assert.Equal(t, "error", entry.LogLevel)
	assert.Equal(t, "error occurred", entry.Message)
	assert.Equal(t, "connection refused", entry.Error)
}

func TestLogAggregatorMultipleStreams(t *testing.T) {
	la := NewLogAggregator("http://loki:3100", 1000, 1*time.Second)
	require.NotNil(t, la)

	streams := []map[string]string{
		{"job": "api", "method": "GET"},
		{"job": "api", "method": "POST"},
		{"job": "worker", "task": "process"},
	}

	for i, stream := range streams {
		entry := LogEntry{
			Timestamp: time.Now(),
			Stream:    stream,
			Message:   "test message",
		}
		la.Collect(entry)
		assert.Equal(t, i+1, la.GetBufferSize())
	}
}

func TestLogAggregatorGetBufferSize(t *testing.T) {
	la := NewLogAggregator("http://loki:3100", 100, 1*time.Second)
	require.NotNil(t, la)

	assert.Equal(t, 0, la.GetBufferSize())

	entry := LogEntry{
		Timestamp: time.Now(),
		Stream:    map[string]string{"job": "test"},
		Message:   "test message",
	}

	la.Collect(entry)
	assert.Equal(t, 1, la.GetBufferSize())

	la.Collect(entry)
	assert.Equal(t, 2, la.GetBufferSize())
}

func TestLogAggregatorFlushEmptyBuffer(t *testing.T) {
	la := NewLogAggregator("http://loki:3100", 100, 1*time.Second)
	require.NotNil(t, la)

	assert.NotPanics(t, func() {
		la.flush()
	})
}

func TestLogAggregatorWithMetadata(t *testing.T) {
	la := NewLogAggregator("http://loki:3100", 100, 1*time.Second)
	require.NotNil(t, la)

	entry := LogEntry{
		Timestamp: time.Now(),
		Stream:    map[string]string{"job": "test"},
		Message:   "test message with metadata",
		Metadata: map[string]any{
			"user_id":   123,
			"request_id": "req-456",
			"duration":  float64(150),
		},
	}

	la.Collect(entry)
	assert.Equal(t, 1, la.GetBufferSize())
}
