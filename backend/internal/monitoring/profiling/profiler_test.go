package profiling

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestProfilerService(t *testing.T) {
	tests := []struct {
		name string
		test func(t *testing.T, ps *ProfilerService)
	}{
		{
			name: "Create profiler service disabled",
			test: func(t *testing.T, ps *ProfilerService) {
				assert.False(t, ps.IsEnabled())
			},
		},
		{
			name: "Get current profile with disabled profiler",
			test: func(t *testing.T, ps *ProfilerService) {
				data := ps.GetCurrentProfile()
				assert.NotNil(t, data)
				assert.NotNil(t, data.Timestamp)
			},
		},
		{
			name: "Get profile history with disabled profiler",
			test: func(t *testing.T, ps *ProfilerService) {
				start := time.Now().Add(-1 * time.Hour)
				end := time.Now()
				history := ps.GetProfileHistory(start, end)
				assert.NotNil(t, history)
				assert.Empty(t, history)
			},
		},
		{
			name: "Get system metrics",
			test: func(t *testing.T, ps *ProfilerService) {
				metrics, err := ps.GetSystemMetrics(context.Background())
				assert.Nil(t, err)
				assert.NotNil(t, metrics)
				assert.Contains(t, metrics, "go_num_goroutines")
				assert.Contains(t, metrics, "cpu_cores")
			},
		},
	}

	ps := NewProfilerService(false, 0)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.test(t, ps)
		})
	}

	ps.Stop()
}

func TestProfilerServiceEnabled(t *testing.T) {
	ps := NewProfilerService(true, 100*time.Millisecond)
	defer ps.Stop()

	assert.True(t, ps.IsEnabled())

	time.Sleep(150 * time.Millisecond)

	data := ps.GetCurrentProfile()
	assert.NotNil(t, data)
	assert.GreaterOrEqual(t, data.Goroutines, 1)

	metrics, err := ps.GetSystemMetrics(context.Background())
	assert.Nil(t, err)
	assert.NotNil(t, metrics)
}

func TestProfileData(t *testing.T) {
	data := &ProfileData{
		Timestamp:   time.Now(),
		Duration:    100 * time.Millisecond,
		CPUUsage:    50.5,
		MemoryUsage: 1024 * 1024,
		Goroutines:  10,
		ThreadCount: 2,
	}

	assert.NotNil(t, data.Timestamp)
	assert.Equal(t, 100*time.Millisecond, data.Duration)
	assert.Equal(t, 50.5, data.CPUUsage)
	assert.Equal(t, uint64(1024*1024), data.MemoryUsage)
	assert.Equal(t, 10, data.Goroutines)
	assert.Equal(t, 2, data.ThreadCount)
}

func TestGCStats(t *testing.T) {
	stats := GCStats{
		NumGC:        10,
		PauseTotalNs: 1000000,
		LastPauseNs:  100000,
		HeapObjects:  1000,
	}

	assert.Equal(t, int64(10), stats.NumGC)
	assert.Equal(t, int64(1000000), stats.PauseTotalNs)
	assert.Equal(t, uint64(100000), stats.LastPauseNs)
	assert.Equal(t, uint64(1000), stats.HeapObjects)
}

func TestHotspotReport(t *testing.T) {
	report := HotspotReport{
		FunctionName: "TestFunction",
		File:         "test.go",
		Line:         42,
		Samples:      100,
		Percentage:   25.5,
	}

	assert.Equal(t, "TestFunction", report.FunctionName)
	assert.Equal(t, "test.go", report.File)
	assert.Equal(t, 42, report.Line)
	assert.Equal(t, 100, report.Samples)
	assert.Equal(t, 25.5, report.Percentage)
}
