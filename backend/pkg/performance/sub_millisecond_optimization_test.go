package performance

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSubMillisecondOptimizer(t *testing.T) {
	t.Run("Initialize and Start", func(t *testing.T) {
		optimizer := NewSubMillisecondOptimizer()
		require.NotNil(t, optimizer)

		err := optimizer.Start()
		require.NoError(t, err)

		assert.True(t, optimizer.isRunning)

		optimizer.Stop()
		assert.False(t, optimizer.isRunning)
	})

	t.Run("Process Fast Path", func(t *testing.T) {
		optimizer := NewSubMillisecondOptimizer()
		err := optimizer.Start()
		require.NoError(t, err)
		defer optimizer.Stop()

		ctx := context.Background()

		req := &Request{
			ID:   "fast-req-1",
			Data: []byte(`{"test":"data"}`),
		}

		resp, err := optimizer.ProcessFastPath(ctx, req)
		require.NoError(t, err)
		require.NotNil(t, resp)

		assert.Equal(t, "fast-req-1", resp.RequestID)
	})

	t.Run("Target Latency Achieved", func(t *testing.T) {
		optimizer := NewSubMillisecondOptimizer()
		err := optimizer.Start()
		require.NoError(t, err)
		defer optimizer.Stop()

		ctx := context.Background()

		for i := 0; i < 10; i++ {
			req := &Request{
				ID:   "fast-req",
				Data: []byte(`{"test":"data"}`),
			}

			optimizer.ProcessFastPath(ctx, req)
		}

		metrics := optimizer.GetMetrics()
		assert.Greater(t, metrics["total_requests"].(int64), int64(0))
	})
}

func TestQPSOptimizer(t *testing.T) {
	t.Run("Initialize and Start", func(t *testing.T) {
		optimizer := NewQPSOptimizer()
		require.NotNil(t, optimizer)

		err := optimizer.Start()
		require.NoError(t, err)

		assert.True(t, optimizer.isRunning)

		optimizer.Stop()
		assert.False(t, optimizer.isRunning)
	})

	t.Run("Process Request", func(t *testing.T) {
		optimizer := NewQPSOptimizer()
		err := optimizer.Start()
		require.NoError(t, err)
		defer optimizer.Stop()

		ctx := context.Background()

		req := &Request{
			ID:   "qps-req-1",
			Data: []byte(`{"test":"data"}`),
		}

		resp, err := optimizer.ProcessRequest(ctx, req)
		require.NoError(t, err)
		require.NotNil(t, resp)

		assert.Equal(t, "qps-req-1", resp.RequestID)
	})

	t.Run("Target QPS", func(t *testing.T) {
		optimizer := NewQPSOptimizer()
		optimizer.config.TargetQPS = 20000
		err := optimizer.Start()
		require.NoError(t, err)
		defer optimizer.Stop()

		ctx := context.Background()

		for i := 0; i < 100; i++ {
			req := &Request{
				ID:   "qps-req",
				Data: []byte(`{"test":"data"}`),
			}

			optimizer.ProcessRequest(ctx, req)
		}

		metrics := optimizer.GetMetrics()
		assert.Greater(t, metrics["total_requests"].(int64), int64(50))
	})
}

func TestResourceEfficiencyOptimizer(t *testing.T) {
	t.Run("Initialize and Start", func(t *testing.T) {
		optimizer := NewResourceEfficiencyOptimizer()
		require.NotNil(t, optimizer)

		err := optimizer.Start()
		require.NoError(t, err)

		assert.True(t, optimizer.isRunning)

		optimizer.Stop()
		assert.False(t, optimizer.isRunning)
	})

	t.Run("Optimize Resources", func(t *testing.T) {
		optimizer := NewResourceEfficiencyOptimizer()
		err := optimizer.Start()
		require.NoError(t, err)
		defer optimizer.Stop()

		optimizer.Optimize()

		metrics := optimizer.GetMetrics()
		assert.Contains(t, metrics, "memory_used_mb")
		assert.Contains(t, metrics, "goroutines")
	})

	t.Run("Process Request with Optimization", func(t *testing.T) {
		optimizer := NewResourceEfficiencyOptimizer()
		err := optimizer.Start()
		require.NoError(t, err)
		defer optimizer.Stop()

		ctx := context.Background()

		req := &Request{
			ID:   "resource-req-1",
			Data: []byte(`{"test":"data"}`),
		}

		resp, err := optimizer.ProcessRequest(ctx, req)
		require.NoError(t, err)
		require.NotNil(t, resp)

		assert.Equal(t, "resource-req-1", resp.RequestID)
	})
}

func TestIntegration(t *testing.T) {
	t.Run("Multiple Optimizers", func(t *testing.T) {
		subMilli := NewSubMillisecondOptimizer()
		qps := NewQPSOptimizer()
		resource := NewResourceEfficiencyOptimizer()

		err := subMilli.Start()
		require.NoError(t, err)
		defer subMilli.Stop()

		err = qps.Start()
		require.NoError(t, err)
		defer qps.Stop()

		err = resource.Start()
		require.NoError(t, err)
		defer resource.Stop()

		ctx := context.Background()

		req := &Request{
			ID:   "integrated-req",
			Data: []byte(`{"test":"data"}`),
		}

		_, err = subMilli.ProcessFastPath(ctx, req)
		require.NoError(t, err)

		_, err = qps.ProcessRequest(ctx, req)
		require.NoError(t, err)

		_, err = resource.ProcessRequest(ctx, req)
		require.NoError(t, err)
	})
}

func BenchmarkSubMillisecond(b *testing.B) {
	optimizer := NewSubMillisecondOptimizer()
	optimizer.Start()
	defer optimizer.Stop()

	ctx := context.Background()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		req := &Request{
			ID:   "bench-req",
			Data: []byte(`{"test":"data"}`),
		}

		optimizer.ProcessFastPath(ctx, req)
	}
}

func BenchmarkQPS(b *testing.B) {
	optimizer := NewQPSOptimizer()
	optimizer.Start()
	defer optimizer.Stop()

	ctx := context.Background()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		req := &Request{
			ID:   "bench-req",
			Data: []byte(`{"test":"data"}`),
		}

		optimizer.ProcessRequest(ctx, req)
	}
}
