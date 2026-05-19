package logging

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestLogAggregator(t *testing.T) {
	tests := []struct {
		name string
		test func(t *testing.T, la *LogAggregator)
	}{
		{
			name: "Create log aggregator disabled",
			test: func(t *testing.T, la *LogAggregator) {
				assert.False(t, la.enabled)
			},
		},
		{
			name: "Add entry to disabled aggregator",
			test: func(t *testing.T, la *LogAggregator) {
				entry := LogEntry{
					Level:   LogLevelInfo,
					Service: "test-service",
					Message: "test message",
				}
				la.AddEntry(entry)
			},
		},
		{
			name: "Get recent logs from disabled aggregator",
			test: func(t *testing.T, la *LogAggregator) {
				logs := la.GetRecentLogs(10)
				assert.NotNil(t, logs)
				assert.Empty(t, logs)
			},
		},
		{
			name: "Query logs from disabled aggregator",
			test: func(t *testing.T, la *LogAggregator) {
				query := LogQuery{
					Limit: 10,
				}
				result := la.QueryLogs(query)
				assert.NotNil(t, result)
				assert.Equal(t, int64(0), result.Total)
			},
		},
	}

	la := NewLogAggregator(100, false)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.test(t, la)
		})
	}
}

func TestLogAggregatorEnabled(t *testing.T) {
	la := NewLogAggregator(100, true)

	entry := LogEntry{
		Level:     LogLevelInfo,
		Service:   "test-service",
		Component: "test-component",
		Message:   "test message",
	}
	la.AddEntry(entry)

	logs := la.GetRecentLogs(10)
	assert.Len(t, logs, 1)
	assert.Equal(t, LogLevelInfo, logs[0].Level)
	assert.Equal(t, "test-service", logs[0].Service)
	assert.Equal(t, "test message", logs[0].Message)

	query := LogQuery{
		Levels:   []LogLevel{LogLevelInfo},
		Services: []string{"test-service"},
		Limit:    10,
	}
	result := la.QueryLogs(query)
	assert.Equal(t, int64(1), result.Total)
	assert.Len(t, result.Logs, 1)

	entry2 := LogEntry{
		Level:   LogLevelError,
		Service: "test-service",
		Message: "error message",
	}
	la.AddEntry(entry2)

	stats := la.QueryLogs(LogQuery{Limit: 100}).Stats
	assert.Equal(t, int64(2), stats.TotalLogs)
	assert.Equal(t, int64(1), stats.ByLevel[LogLevelInfo])
	assert.Equal(t, int64(1), stats.ByLevel[LogLevelError])
}

func TestLogEntry(t *testing.T) {
	entry := LogEntry{
		Timestamp: time.Now(),
		Level:     LogLevelError,
		Service:   "auth-service",
		Component: "login",
		Message:   "login failed",
		TraceID:   "test-trace-id",
		Error:     "invalid credentials",
	}

	assert.NotNil(t, entry.Timestamp)
	assert.Equal(t, LogLevelError, entry.Level)
	assert.Equal(t, "auth-service", entry.Service)
	assert.Equal(t, "login", entry.Component)
	assert.Equal(t, "login failed", entry.Message)
	assert.Equal(t, "test-trace-id", entry.TraceID)
	assert.Equal(t, "invalid credentials", entry.Error)
}

func TestLogQuery(t *testing.T) {
	query := LogQuery{
		Levels:      []LogLevel{LogLevelError, LogLevelWarning},
		Services:    []string{"api", "auth"},
		SearchText:  "error",
		StartTime:   time.Now().Add(-1 * time.Hour),
		EndTime:     time.Now(),
		Limit:       10,
		Offset:      0,
	}

	assert.Len(t, query.Levels, 2)
	assert.Len(t, query.Services, 2)
	assert.Equal(t, "error", query.SearchText)
	assert.Equal(t, 10, query.Limit)
	assert.False(t, query.StartTime.IsZero())
	assert.False(t, query.EndTime.IsZero())
}

func TestContainsIgnoreCase(t *testing.T) {
	assert.True(t, containsIgnoreCase("Hello World", "world"))
	assert.True(t, containsIgnoreCase("Hello World", "HELLO"))
	assert.True(t, containsIgnoreCase("Hello World", "lo W"))
	assert.False(t, containsIgnoreCase("Hello World", "test"))
	assert.False(t, containsIgnoreCase("Hello World", "hello world!"))
	assert.True(t, containsIgnoreCase("Hello World", ""))
}
