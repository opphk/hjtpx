package performance

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHeterogeneousComputing(t *testing.T) {
	t.Run("Initialize and Start", func(t *testing.T) {
		hc := NewHeterogeneousComputing()
		require.NotNil(t, hc)

		err := hc.Start()
		require.NoError(t, err)

		assert.True(t, hc.isRunning)

		hc.Stop()
		assert.False(t, hc.isRunning)
	})

	t.Run("Process Request", func(t *testing.T) {
		hc := NewHeterogeneousComputing()
		hc.Start()
		defer hc.Stop()

		ctx := context.Background()

		req := &ComputeRequest{
			ID:   "hetero-req-1",
			Data: []byte(`{"test":"data"}`),
			Priority: 90,
		}

		resp, err := hc.ProcessRequest(ctx, req)
		require.NoError(t, err)
		require.NotNil(t, resp)

		assert.Equal(t, "hetero-req-1", resp.RequestID)
	})
}

func TestServerlessArchitecture(t *testing.T) {
	t.Run("Initialize and Start", func(t *testing.T) {
		sa := NewServerlessArchitecture()
		require.NotNil(t, sa)

		err := sa.Start()
		require.NoError(t, err)

		assert.True(t, sa.isRunning)

		sa.Stop()
		assert.False(t, sa.isRunning)
	})

	t.Run("Deploy and Invoke Function", func(t *testing.T) {
		sa := NewServerlessArchitecture()
		sa.Start()
		defer sa.Stop()

		ctx := context.Background()

		fn := &ServerlessFunction{
			Name:       "test-func",
			MemoryMB:   128,
			TimeoutSec: 30,
			Code:       []byte("test code"),
		}

		err := sa.DeployFunction(ctx, fn)
		require.NoError(t, err)

		resp, err := sa.InvokeFunction(ctx, fn.ID, []byte(`{"test":"data"}`))
		require.NoError(t, err)
		require.NotNil(t, resp)
	})
}

func TestColdStartOptimizer(t *testing.T) {
	t.Run("Prewarm and IsWarmed", func(t *testing.T) {
		cso := NewColdStartOptimizer()
		require.NotNil(t, cso)

		err := cso.Prewarm("test-func", 128)
		require.NoError(t, err)

		assert.True(t, cso.IsWarmed("test-func"))

		cso.Cleanup()
	})
}
