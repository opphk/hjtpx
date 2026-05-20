package service

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestHeterogeneousComputeService_Initialize(t *testing.T) {
	service := NewHeterogeneousComputeService()
	ctx := context.Background()

	err := service.Initialize(ctx)
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	if !service.isRunning {
		t.Error("Service should be running after Initialize")
	}

	service.Shutdown()
}

func TestHeterogeneousComputeService_RegisterGPUDevice(t *testing.T) {
	service := NewHeterogeneousComputeService()
	ctx := context.Background()
	service.Initialize(ctx)
	defer service.Shutdown()

	device := &GPUDevice{
		ID:           "gpu-0",
		Name:         "NVIDIA Tesla V100",
		Model:        "V100",
		ComputeUnits: 80,
		MemoryMB:     32768,
		Healthy:      true,
	}

	err := service.RegisterGPUDevice(ctx, device)
	if err != nil {
		t.Fatalf("RegisterGPUDevice failed: %v", err)
	}

	devices := service.GetDeviceStatus(ctx)
	if len(devices) == 0 {
		t.Fatal("Should have at least one device registered")
	}

	found := false
	for _, d := range devices {
		if d.DeviceID == "gpu-0" && d.DeviceType == DeviceTypeGPU {
			found = true
			break
		}
	}

	if !found {
		t.Error("GPU device should be registered")
	}
}

func TestHeterogeneousComputeService_RegisterTPUDevice(t *testing.T) {
	service := NewHeterogeneousComputeService()
	ctx := context.Background()
	service.Initialize(ctx)
	defer service.Shutdown()

	device := &TPUDevice{
		ID:              "tpu-0",
		Name:            "Google TPU v3",
		Version:         "v3",
		Cores:           8,
		MemoryMB:        65536,
		Healthy:         true,
		PerformanceTFLOPS: 420,
	}

	err := service.RegisterTPUDevice(ctx, device)
	if err != nil {
		t.Fatalf("RegisterTPUDevice failed: %v", err)
	}
}

func TestHeterogeneousComputeService_RegisterFPGADevice(t *testing.T) {
	service := NewHeterogeneousComputeService()
	ctx := context.Background()
	service.Initialize(ctx)
	defer service.Shutdown()

	device := &FPGADevice{
		ID:           "fpga-0",
		Name:         "Xilinx Alveo U250",
		Model:        "U250",
		LogicCells:   1228800,
		MemoryMB:     16384,
		Healthy:      true,
	}

	err := service.RegisterFPGADevice(ctx, device)
	if err != nil {
		t.Fatalf("RegisterFPGADevice failed: %v", err)
	}
}

func TestHeterogeneousComputeService_RegisterASICDevice(t *testing.T) {
	service := NewHeterogeneousComputeService()
	ctx := context.Background()
	service.Initialize(ctx)
	defer service.Shutdown()

	device := &ASICDevice{
		ID:           "asic-0",
		Name:         "SHA-256 ASIC",
		Type:         "crypto",
		Throughput:   100000,
		LatencyNanos: 100,
		PowerUsageW:  50,
		Healthy:      true,
	}

	err := service.RegisterASICDevice(ctx, device)
	if err != nil {
		t.Fatalf("RegisterASICDevice failed: %v", err)
	}
}

func TestHeterogeneousComputeService_ProcessOnGPU(t *testing.T) {
	service := NewHeterogeneousComputeService()
	ctx := context.Background()
	service.Initialize(ctx)
	defer service.Shutdown()

	device := &GPUDevice{
		ID:           "gpu-test",
		Name:         "Test GPU",
		ComputeUnits: 80,
		MemoryMB:     32768,
		Healthy:      true,
	}
	service.RegisterGPUDevice(ctx, device)

	req := &ComputeRequest{
		RequestID:  "gpu-test-req",
		Data:       []byte("test data"),
		Algorithm:  "neural_network",
		DeviceType: DeviceTypeGPU,
		Timeout:    5 * time.Second,
	}

	result, err := service.ProcessCompute(ctx, req)
	if err != nil {
		t.Fatalf("ProcessCompute failed: %v", err)
	}

	if !result.Success {
		t.Error("GPU compute should succeed")
	}

	if result.DeviceType != DeviceTypeGPU {
		t.Errorf("Expected device type %s, got %s", DeviceTypeGPU, result.DeviceType)
	}

	t.Logf("GPU processing latency: %v", result.Latency)
}

func TestHeterogeneousComputeService_ProcessOnTPU(t *testing.T) {
	service := NewHeterogeneousComputeService()
	ctx := context.Background()
	service.Initialize(ctx)
	defer service.Shutdown()

	device := &TPUDevice{
		ID:      "tpu-test",
		Name:    "Test TPU",
		Cores:   8,
		MemoryMB: 65536,
		Healthy: true,
	}
	service.RegisterTPUDevice(ctx, device)

	req := &ComputeRequest{
		RequestID:  "tpu-test-req",
		Data:      []byte("test data"),
		Algorithm: "matrix_multiplication",
		DeviceType: DeviceTypeTPU,
		Timeout:   5 * time.Second,
	}

	result, err := service.ProcessCompute(ctx, req)
	if err != nil {
		t.Fatalf("ProcessCompute failed: %v", err)
	}

	if !result.Success {
		t.Error("TPU compute should succeed")
	}

	t.Logf("TPU processing latency: %v", result.Latency)
}

func TestHeterogeneousComputeService_ProcessOnASIC(t *testing.T) {
	service := NewHeterogeneousComputeService()
	ctx := context.Background()
	service.Initialize(ctx)
	defer service.Shutdown()

	device := &ASICDevice{
		ID:           "asic-test",
		Name:         "Test ASIC",
		Type:         "verification",
		Throughput:   100000,
		LatencyNanos: 100,
		Healthy:      true,
	}
	service.RegisterASICDevice(ctx, device)

	req := &ComputeRequest{
		RequestID:  "asic-test-req",
		Data:      []byte("test data"),
		Algorithm: "hashing",
		DeviceType: DeviceTypeASIC,
		Timeout:   5 * time.Second,
	}

	result, err := service.ProcessCompute(ctx, req)
	if err != nil {
		t.Fatalf("ProcessCompute failed: %v", err)
	}

	if !result.Success {
		t.Error("ASIC compute should succeed")
	}

	if result.Latency > 1*time.Millisecond {
		t.Logf("ASIC latency %v seems high", result.Latency)
	}
}

func TestHeterogeneousComputeService_FallbackToCPU(t *testing.T) {
	service := NewHeterogeneousComputeService()
	ctx := context.Background()
	service.Initialize(ctx)
	defer service.Shutdown()

	req := &ComputeRequest{
		RequestID:  "cpu-fallback-req",
		Data:      []byte("test data"),
		Algorithm: "unknown",
		DeviceType: DeviceTypeGPU,
		Timeout:   5 * time.Second,
	}

	result, err := service.ProcessCompute(ctx, req)
	if err != nil {
		t.Fatalf("ProcessCompute failed: %v", err)
	}

	if result.DeviceType != DeviceTypeCPU {
		t.Logf("Expected CPU fallback, got %s", result.DeviceType)
	}
}

func TestHeterogeneousComputeService_ConcurrentCompute(t *testing.T) {
	service := NewHeterogeneousComputeService()
	ctx := context.Background()
	service.Initialize(ctx)
	defer service.Shutdown()

	device := &GPUDevice{
		ID:           "gpu-concurrent",
		Name:         "Concurrent GPU",
		ComputeUnits: 80,
		MemoryMB:     32768,
		Healthy:      true,
	}
	service.RegisterGPUDevice(ctx, device)

	var wg sync.WaitGroup
	requestCount := 500

	start := time.Now()
	for i := 0; i < requestCount; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			req := &ComputeRequest{
				RequestID:  "concurrent-req",
				Data:      []byte("concurrent test data"),
				Algorithm: "neural_network",
				DeviceType: DeviceTypeGPU,
				Timeout:   10 * time.Second,
			}
			service.ProcessCompute(ctx, req)
		}(i)
	}

	wg.Wait()
	elapsed := time.Since(start)

	t.Logf("Processed %d concurrent compute requests in %v", requestCount, elapsed)
	t.Logf("Throughput: %.2f req/s", float64(requestCount)/elapsed.Seconds())

	stats := service.GetStats()
	totalOps := stats["total_compute_ops"].(int64)
	if totalOps != int64(requestCount) {
		t.Errorf("Expected %d total compute ops, got %d", requestCount, totalOps)
	}
}

func TestHeterogeneousComputeService_LoadFPGABitstream(t *testing.T) {
	service := NewHeterogeneousComputeService()
	ctx := context.Background()
	service.Initialize(ctx)
	defer service.Shutdown()

	device := &FPGADevice{
		ID:           "fpga-load",
		Name:         "Load Test FPGA",
		LogicCells:   1228800,
		MemoryMB:     16384,
		Healthy:      true,
	}
	service.RegisterFPGADevice(ctx, device)

	bitstreamID := "crypto-accelerator"
	bitstream := []byte("mock-bitstream-data")

	err := service.LoadFPGABitstream(ctx, "fpga-load", bitstreamID, bitstream)
	if err != nil {
		t.Fatalf("LoadFPGABitstream failed: %v", err)
	}
}

func TestHeterogeneousComputeService_GetDeviceStatus(t *testing.T) {
	service := NewHeterogeneousComputeService()
	ctx := context.Background()
	service.Initialize(ctx)
	defer service.Shutdown()

	service.RegisterGPUDevice(ctx, &GPUDevice{
		ID:           "gpu-status",
		Name:         "Status GPU",
		ComputeUnits: 80,
		MemoryMB:     32768,
		Healthy:      true,
	})

	service.RegisterTPUDevice(ctx, &TPUDevice{
		ID:      "tpu-status",
		Name:    "Status TPU",
		Cores:   8,
		MemoryMB: 65536,
		Healthy: true,
	})

	statuses := service.GetDeviceStatus(ctx)

	if len(statuses) != 2 {
		t.Errorf("Expected 2 device statuses, got %d", len(statuses))
	}
}

func TestHeterogeneousComputeService_GetStats(t *testing.T) {
	service := NewHeterogeneousComputeService()
	ctx := context.Background()
	service.Initialize(ctx)
	defer service.Shutdown()

	device := &GPUDevice{
		ID:           "gpu-stats",
		Name:         "Stats GPU",
		ComputeUnits: 80,
		MemoryMB:     32768,
		Healthy:      true,
	}
	service.RegisterGPUDevice(ctx, device)

	for i := 0; i < 100; i++ {
		req := &ComputeRequest{
			RequestID:   "stats-test",
			Data:       []byte("stats test data"),
			Algorithm:  "neural_network",
			DeviceType: DeviceTypeGPU,
		}
		service.ProcessCompute(ctx, req)
	}

	stats := service.GetStats()

	if stats["total_compute_ops"].(int64) != 100 {
		t.Errorf("Expected 100 total compute ops, got %d", stats["total_compute_ops"])
	}

	gpuOps := stats["gpu_ops"].(int64)
	if gpuOps != 100 {
		t.Errorf("Expected 100 GPU ops, got %d", gpuOps)
	}

	if stats["avg_latency_ns"].(int64) == 0 {
		t.Error("Average latency should not be zero")
	}
}

func TestHeterogeneousComputeService_OptimalDeviceSelection(t *testing.T) {
	service := NewHeterogeneousComputeService()
	ctx := context.Background()
	service.Initialize(ctx)
	defer service.Shutdown()

	service.RegisterGPUDevice(ctx, &GPUDevice{
		ID:           "gpu-select",
		Name:         "Selection GPU",
		ComputeUnits: 80,
		MemoryMB:     32768,
		Healthy:      true,
	})

	testCases := []struct {
		algorithm string
		expected  string
	}{
		{"neural_network", DeviceTypeGPU},
		{"deep_learning", DeviceTypeGPU},
		{"image_processing", DeviceTypeGPU},
		{"matrix_multiplication", DeviceTypeTPU},
		{"transformer", DeviceTypeTPU},
		{"hashing", DeviceTypeASIC},
		{"verification", DeviceTypeASIC},
		{"custom_crypto", DeviceTypeFPGA},
		{"pattern_matching", DeviceTypeFPGA},
	}

	for _, tc := range testCases {
		selectedType := service.selectOptimalDeviceType(tc.algorithm)
		if selectedType != tc.expected {
			t.Errorf("For algorithm %s: expected %s, got %s", tc.algorithm, tc.expected, selectedType)
		}
	}
}

func BenchmarkHeterogeneousComputeService_GPUCompute(b *testing.B) {
	service := NewHeterogeneousComputeService()
	ctx := context.Background()
	service.Initialize(ctx)
	defer service.Shutdown()

	service.RegisterGPUDevice(ctx, &GPUDevice{
		ID:           "bench-gpu",
		Name:         "Benchmark GPU",
		ComputeUnits: 80,
		MemoryMB:     32768,
		Healthy:      true,
	})

	req := &ComputeRequest{
		RequestID:  "benchmark",
		Data:      []byte("benchmark data"),
		Algorithm: "neural_network",
		DeviceType: DeviceTypeGPU,
		Timeout:   10 * time.Second,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		service.ProcessCompute(ctx, req)
	}
}

func BenchmarkHeterogeneousComputeService_ConcurrentGPU(b *testing.B) {
	service := NewHeterogeneousComputeService()
	ctx := context.Background()
	service.Initialize(ctx)
	defer service.Shutdown()

	service.RegisterGPUDevice(ctx, &GPUDevice{
		ID:           "concurrent-bench-gpu",
		Name:         "Concurrent Benchmark GPU",
		ComputeUnits: 80,
		MemoryMB:     32768,
		Healthy:      true,
	})

	var wg sync.WaitGroup

	b.ResetTimer()
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < b.N/50; j++ {
				req := &ComputeRequest{
					RequestID:  "concurrent-bench",
					Data:      []byte("data"),
					Algorithm: "neural_network",
					DeviceType: DeviceTypeGPU,
				}
				service.ProcessCompute(ctx, req)
			}
		}()
	}

	wg.Wait()
}

func BenchmarkHeterogeneousComputeService_DeviceSelection(b *testing.B) {
	service := NewHeterogeneousComputeService()

	algorithms := []string{
		"neural_network",
		"deep_learning",
		"image_processing",
		"matrix_multiplication",
		"transformer",
		"hashing",
		"verification",
		"custom_crypto",
		"pattern_matching",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		service.selectOptimalDeviceType(algorithms[i%len(algorithms)])
	}
}
