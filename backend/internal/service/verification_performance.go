package service

import (
	"sync"
	"sync/atomic"
	"time"
)

type VerificationMetrics struct {
	TotalVerifications     int64           `json:"total_verifications"`
	SuccessfulVerifications int64           `json:"successful_verifications"`
	FailedVerifications    int64           `json:"failed_verifications"`
	AverageLatencyMs       atomic.Int64     `json:"average_latency_ms"`
	P50LatencyMs           atomic.Int64     `json:"p50_latency_ms"`
	P95LatencyMs           atomic.Int64     `json:"p95_latency_ms"`
	P99LatencyMs           atomic.Int64     `json:"p99_latency_ms"`
	LastVerificationTime   atomic.Int64     `json:"last_verification_time"`
}

type ImageOptimization struct {
	Enabled              bool               `json:"enabled"`
	CompressionLevel     int                `json:"compression_level"`
	MaxWidth             int                `json:"max_width"`
	MaxHeight            int                `json:"max_height"`
	ProgressiveLoading   bool               `json:"progressive_loading"`
	LazyLoading          bool               `json:"lazy_loading"`
	WebPLossless         bool               `json:"webp_lossless"`
	Quality              int                `json:"quality"`
}

type MemoryOptimization struct {
	MaxMemoryMB          int64              `json:"max_memory_mb"`
	CurrentMemoryMB      atomic.Int64        `json:"current_memory_mb"`
	ObjectPoolSize      int                `json:"object_pool_size"`
	GCFrequencySeconds  int                `json:"gc_frequency_seconds"`
	EnableMemoryPooling  bool               `json:"enable_memory_pooling"`
	CacheSizeMB          int                `json:"cache_size_mb"`
}

type BatteryOptimization struct {
	Enabled               bool               `json:"enabled"`
	LowPowerModeThreshold int                `json:"low_power_mode_threshold"`
	ReduceAnimations      bool               `json:"reduce_animations"`
	ReduceNetworkRequests bool               `json:"reduce_network_requests"`
	BatchProcessing       bool               `json:"batch_processing"`
	AdaptiveRefreshRate   bool               `json:"adaptive_refresh_rate"`
	CurrentBatteryLevel   atomic.Int64       `json:"current_battery_level"`
	IsCharging           atomic.Bool        `json:"is_charging"`
}

type VerificationPerformanceOptimizer struct {
	mu              sync.RWMutex
	metrics         VerificationMetrics
	imageOptim      ImageOptimization
	memOptim        MemoryOptimization
	batteryOptim    BatteryOptimization
	latencySamples  []int64
	muLatency       sync.RWMutex
	pool            *sync.Pool
}

type PerformanceProfile string

const (
	ProfileHighPerformance PerformanceProfile = "high_performance"
	ProfileBalanced        PerformanceProfile = "balanced"
	ProfilePowerSaving     PerformanceProfile = "power_saving"
	ProfileUltraLight      PerformanceProfile = "ultra_light"
)

type OptimizationConfigV2 struct {
	Profile          PerformanceProfile   `json:"profile"`
	ImageOptim       ImageOptimization    `json:"image_optimization"`
	MemOptim         MemoryOptimization   `json:"memory_optimization"`
	BatteryOptim     BatteryOptimization  `json:"battery_optimization"`
	EnableMetrics    bool                `json:"enable_metrics"`
	MetricsInterval  int                 `json:"metrics_interval_seconds"`
}

func NewVerificationPerformanceOptimizer() *VerificationPerformanceOptimizer {
	vpo := &VerificationPerformanceOptimizer{
		latencySamples: make([]int64, 0, 10000),
		imageOptim: ImageOptimization{
			Enabled:            true,
			CompressionLevel:   6,
			MaxWidth:           1920,
			MaxHeight:          1080,
			ProgressiveLoading: true,
			LazyLoading:        true,
			WebPLossless:       false,
			Quality:            85,
		},
		memOptim: MemoryOptimization{
			MaxMemoryMB:         512,
			ObjectPoolSize:      100,
			GCFrequencySeconds: 60,
			EnableMemoryPooling: true,
			CacheSizeMB:        64,
		},
		batteryOptim: BatteryOptimization{
			Enabled:               true,
			LowPowerModeThreshold: 20,
			ReduceAnimations:      false,
			ReduceNetworkRequests: false,
			BatchProcessing:       true,
			AdaptiveRefreshRate:   true,
		},
	}

	vpo.pool = &sync.Pool{
		New: func() interface{} {
			return &VerificationContextV2{}
		},
	}

	return vpo
}

type VerificationContext struct {
	UserID       string
	StartTime    time.Time
	LatencyMs    int64
	Success      bool
	Steps        []string
}

func (vpo *VerificationPerformanceOptimizer) StartVerification(userID string) *VerificationContext {
	ctx := vpo.pool.Get().(*VerificationContext)
	ctx.UserID = userID
	ctx.StartTime = time.Now()
	ctx.Success = false
	ctx.Steps = make([]string, 0)

	return ctx
}

func (vpo *VerificationPerformanceOptimizer) EndVerification(vctx *VerificationContext, success bool) int64 {
	vctx.Success = success
	latency := time.Since(vctx.StartTime).Milliseconds()

	atomic.AddInt64(&vpo.metrics.TotalVerifications, 1)
	if success {
		atomic.AddInt64(&vpo.metrics.SuccessfulVerifications, 1)
	} else {
		atomic.AddInt64(&vpo.metrics.FailedVerifications, 1)
	}

	vpo.muLatency.Lock()
	vpo.latencySamples = append(vpo.latencySamples, latency)
	if len(vpo.latencySamples) > 10000 {
		vpo.latencySamples = vpo.latencySamples[1:]
	}
	vpo.muLatency.Unlock()

	vpo.updateLatencyStats()
	vpo.metrics.LastVerificationTime.Store(time.Now().Unix())

	vctx.Reset()
	vpo.pool.Put(vctx)

	return latency
}

func (vctx *VerificationContext) Reset() {
	vctx.UserID = ""
	vctx.StartTime = time.Time{}
	vctx.LatencyMs = 0
	vctx.Success = false
	vctx.Steps = vctx.Steps[:0]
}

func (vpo *VerificationPerformanceOptimizer) updateLatencyStats() {
	vpo.muLatency.RLock()
	samples := make([]int64, len(vpo.latencySamples))
	copy(samples, vpo.latencySamples)
	vpo.muLatency.RUnlock()

	if len(samples) == 0 {
		return
	}

	var total int64
	for _, s := range samples {
		total += s
	}
	avg := total / int64(len(samples))
	vpo.metrics.AverageLatencyMs.Store(avg)

	sortInt64(samples)
	p50 := samples[len(samples)*50/100]
	p95 := samples[len(samples)*95/100]
	p99 := samples[len(samples)*99/100]

	vpo.metrics.P50LatencyMs.Store(p50)
	vpo.metrics.P95LatencyMs.Store(p95)
	vpo.metrics.P99LatencyMs.Store(p99)
}

func sortInt64(arr []int64) {
	for i := 1; i < len(arr); i++ {
		for j := i; j > 0 && arr[j] < arr[j-1]; j-- {
			arr[j], arr[j-1] = arr[j-1], arr[j]
		}
	}
}

func (vpo *VerificationPerformanceOptimizer) OptimizeImage(params *ImageOptimizationParams) (*ImageOptimizationResult, error) {
	result := &ImageOptimizationResult{
		OriginalSize:   params.OriginalSize,
		OptimizedSize:  params.OriginalSize,
		Format:         params.Format,
		Width:          params.Width,
		Height:         params.Height,
	}

	if vpo.imageOptim.CompressionLevel > 0 {
		compressionRatio := 1.0 - (float64(vpo.imageOptim.CompressionLevel) / 10.0)
		result.OptimizedSize = int64(float64(params.OriginalSize) * compressionRatio)
	}

	if params.Width > vpo.imageOptim.MaxWidth || params.Height > vpo.imageOptim.MaxHeight {
		scale := float64(vpo.imageOptim.MaxWidth) / float64(params.Width)
		if scale < 1.0 {
			result.Width = int(float64(params.Width) * scale)
			result.Height = int(float64(params.Height) * scale)
		}
	}

	if vpo.imageOptim.WebPLossless && params.Format == "jpeg" {
		result.Format = "webp"
		result.OptimizedSize = int64(float64(result.OptimizedSize) * 0.75)
	}

	result.CompressionRatio = float64(result.OptimizedSize) / float64(params.OriginalSize)
	result.LoadTimeMs = result.OptimizedSize / 1000

	return result, nil
}

type ImageOptimizationParams struct {
	OriginalSize int64  `json:"original_size"`
	Format       string `json:"format"`
	Width        int    `json:"width"`
	Height       int    `json:"height"`
}

type ImageOptimizationResult struct {
	OriginalSize     int64   `json:"original_size"`
	OptimizedSize    int64   `json:"optimized_size"`
	Format           string  `json:"format"`
	Width            int     `json:"width"`
	Height           int     `json:"height"`
	CompressionRatio float64 `json:"compression_ratio"`
	LoadTimeMs       int64   `json:"load_time_ms"`
}

func (vpo *VerificationPerformanceOptimizer) OptimizeMemory(operation string) MemoryOptimizationResult {
	result := MemoryOptimizationResult{
		Operation:      operation,
		BeforeMemoryMB: vpo.memOptim.CurrentMemoryMB.Load(),
	}

	switch operation {
	case "gc":
		result.AfterMemoryMB = result.BeforeMemoryMB - int64(vpo.memOptim.CacheSizeMB/2)
	case "pool":
		result.AfterMemoryMB = result.BeforeMemoryMB - int64(vpo.memOptim.CacheSizeMB/4)
	case "cache_clear":
		result.AfterMemoryMB = result.BeforeMemoryMB - int64(vpo.memOptim.CacheSizeMB)
	default:
		result.AfterMemoryMB = result.BeforeMemoryMB
	}

	if result.AfterMemoryMB < 0 {
		result.AfterMemoryMB = 0
	}

	vpo.memOptim.CurrentMemoryMB.Store(result.AfterMemoryMB)
	result.ReleasedMB = result.BeforeMemoryMB - result.AfterMemoryMB

	return result
}

type MemoryOptimizationResult struct {
	Operation      string `json:"operation"`
	BeforeMemoryMB int64 `json:"before_memory_mb"`
	AfterMemoryMB  int64 `json:"after_memory_mb"`
	ReleasedMB     int64 `json:"released_mb"`
}

func (vpo *VerificationPerformanceOptimizer) OptimizeForBattery(batteryLevel int, isCharging bool) BatteryOptimizationResult {
	result := BatteryOptimizationResult{
		Enabled:           vpo.batteryOptim.Enabled,
		BatteryLevel:      batteryLevel,
		IsCharging:        isCharging,
		OptimizationsApplied: make([]string, 0),
	}

	vpo.batteryOptim.CurrentBatteryLevel.Store(int64(batteryLevel))
	vpo.batteryOptim.IsCharging.Store(isCharging)

	if batteryLevel < int(vpo.batteryOptim.LowPowerModeThreshold) && !isCharging {
		result.LowPowerMode = true
		result.OptimizationsApplied = append(result.OptimizationsApplied,
			"reduce_animations",
			"reduce_network_requests",
			"batch_processing",
			"adaptive_refresh_rate",
		)

		result.EstimatedPowerSaving = 40
	} else if batteryLevel < 50 && !isCharging {
		result.LowPowerMode = false
		result.OptimizationsApplied = append(result.OptimizationsApplied,
			"adaptive_refresh_rate",
			"batch_processing",
		)

		result.EstimatedPowerSaving = 20
	} else {
		result.LowPowerMode = false
		result.EstimatedPowerSaving = 0
	}

	return result
}

type BatteryOptimizationResult struct {
	Enabled               bool     `json:"enabled"`
	BatteryLevel          int      `json:"battery_level"`
	IsCharging            bool     `json:"is_charging"`
	LowPowerMode          bool     `json:"low_power_mode"`
	OptimizationsApplied  []string `json:"optimizations_applied"`
	EstimatedPowerSaving  int      `json:"estimated_power_saving_percent"`
}

func (vpo *VerificationPerformanceOptimizer) ApplyProfile(profile PerformanceProfile) error {
	vpo.mu.Lock()
	defer vpo.mu.Unlock()

	switch profile {
	case ProfileHighPerformance:
		vpo.imageOptim.Quality = 95
		vpo.imageOptim.CompressionLevel = 3
		vpo.memOptim.EnableMemoryPooling = true
		vpo.memOptim.CacheSizeMB = 128
		vpo.batteryOptim.ReduceAnimations = false

	case ProfileBalanced:
		vpo.imageOptim.Quality = 85
		vpo.imageOptim.CompressionLevel = 6
		vpo.memOptim.EnableMemoryPooling = true
		vpo.memOptim.CacheSizeMB = 64
		vpo.batteryOptim.ReduceAnimations = false

	case ProfilePowerSaving:
		vpo.imageOptim.Quality = 70
		vpo.imageOptim.CompressionLevel = 8
		vpo.memOptim.EnableMemoryPooling = false
		vpo.memOptim.CacheSizeMB = 32
		vpo.batteryOptim.ReduceAnimations = true

	case ProfileUltraLight:
		vpo.imageOptim.Quality = 60
		vpo.imageOptim.CompressionLevel = 9
		vpo.memOptim.EnableMemoryPooling = false
		vpo.memOptim.CacheSizeMB = 16
		vpo.batteryOptim.ReduceAnimations = true
		vpo.batteryOptim.ReduceNetworkRequests = true

	default:
		return ErrInvalidParameter
	}

	return nil
}

func (vpo *VerificationPerformanceOptimizer) GetMetrics() VerificationMetrics {
	return vpo.metrics
}

func (vpo *VerificationPerformanceOptimizer) GetImageOptimization() ImageOptimization {
	return vpo.imageOptim
}

func (vpo *VerificationPerformanceOptimizer) GetMemoryOptimization() MemoryOptimization {
	return vpo.memOptim
}

func (vpo *VerificationPerformanceOptimizer) GetBatteryOptimization() BatteryOptimization {
	return vpo.batteryOptim
}

func (vpo *VerificationPerformanceOptimizer) GetPerformanceReport() map[string]interface{} {
	vpo.mu.RLock()
	defer vpo.mu.RUnlock()

	metrics := vpo.GetMetrics()
	total := metrics.TotalVerifications
	successRate := float64(0)
	if total > 0 {
		successRate = float64(metrics.SuccessfulVerifications) / float64(total) * 100
	}

	return map[string]interface{}{
		"verification_metrics": map[string]interface{}{
			"total":            total,
			"successful":       metrics.SuccessfulVerifications,
			"failed":           metrics.FailedVerifications,
			"success_rate":     successRate,
			"avg_latency_ms":   metrics.AverageLatencyMs.Load(),
			"p50_latency_ms":   metrics.P50LatencyMs.Load(),
			"p95_latency_ms":   metrics.P95LatencyMs.Load(),
			"p99_latency_ms":   metrics.P99LatencyMs.Load(),
			"last_verify_time": metrics.LastVerificationTime.Load(),
		},
		"image_optimization": vpo.imageOptim,
		"memory_optimization": map[string]interface{}{
			"max_memory_mb":       vpo.memOptim.MaxMemoryMB,
			"current_memory_mb":   vpo.memOptim.CurrentMemoryMB.Load(),
			"cache_size_mb":       vpo.memOptim.CacheSizeMB,
			"object_pool_size":   vpo.memOptim.ObjectPoolSize,
			"memory_pool_enabled": vpo.memOptim.EnableMemoryPooling,
		},
		"battery_optimization": map[string]interface{}{
			"enabled":               vpo.batteryOptim.Enabled,
			"current_level":         vpo.batteryOptim.CurrentBatteryLevel.Load(),
			"is_charging":           vpo.batteryOptim.IsCharging.Load(),
			"low_power_threshold":   vpo.batteryOptim.LowPowerModeThreshold,
		},
	}
}

func (vpo *VerificationPerformanceOptimizer) PredictLatency() int64 {
	vpo.muLatency.RLock()
	defer vpo.muLatency.RUnlock()

	if len(vpo.latencySamples) < 10 {
		return 0
	}

	var sum, count int64
	for i := len(vpo.latencySamples) - 10; i < len(vpo.latencySamples); i++ {
		sum += vpo.latencySamples[i]
		count++
	}

	return sum / count
}
