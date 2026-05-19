package profiling

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"runtime"
	"runtime/pprof"
	"runtime/trace"
	"sync"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/process"
)

type ProfilerService struct {
	mu              sync.RWMutex
	enabled         bool
	cpuProfileFile  string
	memProfileFile  string
	traceFile       string
	profiles        map[string]*ProfileData
	collectInterval time.Duration
	stopChan        chan struct{}
}

type ProfileData struct {
	Timestamp     time.Time
	Duration      time.Duration
	CPUUsage      float64
	MemoryUsage   uint64
	MemoryAlloc   uint64
	MemorySys     uint64
	GCStats       GCStats
	Goroutines    int
	ThreadCount   int
}

type GCStats struct {
	NumGC        int64
	PauseTotalNs int64
	PauseNs      []uint64
	LastPauseNs  uint64
	HeapObjects  uint64
}

type HotspotReport struct {
	FunctionName string  `json:"function_name"`
	File         string  `json:"file"`
	Line         int     `json:"line"`
	Samples      int     `json:"samples"`
	Percentage   float64 `json:"percentage"`
}

type ProfileResult struct {
	Success     bool              `json:"success"`
	ProfileType string            `json:"profile_type"`
	FilePath    string            `json:"file_path,omitempty"`
	Data        *ProfileData      `json:"data,omitempty"`
	Hotspots    []HotspotReport   `json:"hotspots,omitempty"`
	Error       string            `json:"error,omitempty"`
}

var instance *ProfilerService
var once sync.Once

func NewProfilerService(enabled bool, collectInterval time.Duration) *ProfilerService {
	once.Do(func() {
		instance = &ProfilerService{
			enabled:         enabled,
			profiles:        make(map[string]*ProfileData),
			collectInterval: collectInterval,
			stopChan:        make(chan struct{}),
		}

		if enabled && collectInterval > 0 {
			go instance.startCollection()
		}
	})
	return instance
}

func (ps *ProfilerService) startCollection() {
	ticker := time.NewTicker(ps.collectInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			ps.collectProfileData()
		case <-ps.stopChan:
			return
		}
	}
}

func (ps *ProfilerService) collectProfileData() {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	data := &ProfileData{
		Timestamp: time.Now(),
	}

	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	data.MemoryAlloc = m.Alloc
	data.MemorySys = m.Sys
	data.Goroutines = runtime.NumGoroutine()

	data.GCStats = GCStats{
		NumGC:        m.NumGC,
		PauseTotalNs: m.PauseTotalNs,
		PauseNs:      m.PauseNs,
		HeapObjects:  m.HeapObjects,
	}
	if m.NumGC > 0 {
		data.GCStats.LastPauseNs = m.PauseNs[(m.NumGC+255)%256]
	}

	cpuPercent, err := cpu.Percent(0, false)
	if err == nil && len(cpuPercent) > 0 {
		data.CPUUsage = cpuPercent[0]
	}

	memInfo, err := mem.VirtualMemory()
	if err == nil {
		data.MemoryUsage = memInfo.Used
	}

	proc, err := process.NewProcess(int32(os.Getpid()))
	if err == nil {
		threads, _ := proc.NumThreads()
		data.ThreadCount = int(threads)
	}

	key := data.Timestamp.Format("20060102150405")
	ps.profiles[key] = data

	if len(ps.profiles) > 1000 {
		var oldestKey string
		for k := range ps.profiles {
			if oldestKey == "" || k < oldestKey {
				oldestKey = k
			}
		}
		delete(ps.profiles, oldestKey)
	}
}

func (ps *ProfilerService) StartCPUProfile(duration time.Duration) (*ProfileResult, error) {
	if !ps.enabled {
		return &ProfileResult{Success: false, Error: "profiling is disabled"}, nil
	}

	var buf bytes.Buffer
	if err := pprof.StartCPUProfile(&buf); err != nil {
		return nil, fmt.Errorf("failed to start CPU profile: %w", err)
	}

	time.Sleep(duration)
	pprof.StopCPUProfile()

	result := &ProfileResult{
		Success:     true,
		ProfileType: "cpu",
		Data:        ps.GetCurrentProfile(),
	}

	return result, nil
}

func (ps *ProfilerService) StartMemoryProfile() (*ProfileResult, error) {
	if !ps.enabled {
		return &ProfileResult{Success: false, Error: "profiling is disabled"}, nil
	}

	var buf bytes.Buffer
	if err := pprof.WriteHeapProfile(&buf); err != nil {
		return nil, fmt.Errorf("failed to write memory profile: %w", err)
	}

	result := &ProfileResult{
		Success:     true,
		ProfileType: "memory",
		Data:        ps.GetCurrentProfile(),
	}

	return result, nil
}

func (ps *ProfilerService) StartTraceProfile(duration time.Duration) (*ProfileResult, error) {
	if !ps.enabled {
		return &ProfileResult{Success: false, Error: "profiling is disabled"}, nil
	}

	var buf bytes.Buffer
	if err := trace.Start(&buf); err != nil {
		return nil, fmt.Errorf("failed to start trace: %w", err)
	}

	time.Sleep(duration)
	trace.Stop()

	result := &ProfileResult{
		Success:     true,
		ProfileType: "trace",
		Data:        ps.GetCurrentProfile(),
	}

	return result, nil
}

func (ps *ProfilerService) GetCurrentProfile() *ProfileData {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	var latest *ProfileData
	for _, data := range ps.profiles {
		if latest == nil || data.Timestamp.After(latest.Timestamp) {
			latest = data
		}
	}

	if latest == nil {
		latest = &ProfileData{Timestamp: time.Now()}
	}

	return latest
}

func (ps *ProfilerService) GetProfileHistory(start, end time.Time) []*ProfileData {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	var result []*ProfileData
	for _, data := range ps.profiles {
		if data.Timestamp.After(start) && data.Timestamp.Before(end) {
			result = append(result, data)
		}
	}

	for i := 0; i < len(result)-1; i++ {
		for j := i + 1; j < len(result); j++ {
			if result[j].Timestamp.Before(result[i].Timestamp) {
				result[i], result[j] = result[j], result[i]
			}
		}
	}

	return result
}

func (ps *ProfilerService) AnalyzeHotspots(duration time.Duration) ([]HotspotReport, error) {
	if !ps.enabled {
		return nil, fmt.Errorf("profiling is disabled")
	}

	var buf bytes.Buffer
	if err := pprof.StartCPUProfile(&buf); err != nil {
		return nil, fmt.Errorf("failed to start CPU profile: %w", err)
	}

	time.Sleep(duration)
	pprof.StopCPUProfile()

	profile, err := pprof.Lookup("cpu")
	if err != nil {
		return nil, fmt.Errorf("failed to lookup CPU profile: %w", err)
	}

	var reports []HotspotReport
	totalSamples := 0

	profile.ForEach(func(p *pprof.Profile, sample []pprof.Sample) error {
		for _, s := range sample {
			if len(s.Location) > 0 && len(s.Location[0].Line) > 0 {
				totalSamples += s.Value[0]
				reports = append(reports, HotspotReport{
					FunctionName: s.Location[0].Function.Name,
					File:         s.Location[0].File,
					Line:         s.Location[0].Line[0].Line,
					Samples:      s.Value[0],
				})
			}
		}
		return nil
	})

	for i := 0; i < len(reports)-1; i++ {
		for j := i + 1; j < len(reports); j++ {
			if reports[j].Samples > reports[i].Samples {
				reports[i], reports[j] = reports[j], reports[i]
			}
		}
	}

	for i := range reports {
		if totalSamples > 0 {
			reports[i].Percentage = float64(reports[i].Samples) / float64(totalSamples) * 100
		}
	}

	if len(reports) > 20 {
		reports = reports[:20]
	}

	return reports, nil
}

func (ps *ProfilerService) GetSystemMetrics(ctx context.Context) (map[string]interface{}, error) {
	metrics := make(map[string]interface{})

	cpuInfo, err := cpu.Info()
	if err == nil && len(cpuInfo) > 0 {
		metrics["cpu_model"] = cpuInfo[0].ModelName
		metrics["cpu_cores"] = runtime.NumCPU()
	}

	cpuPercent, err := cpu.Percent(0, false)
	if err == nil && len(cpuPercent) > 0 {
		metrics["cpu_usage_percent"] = cpuPercent[0]
	}

	memInfo, err := mem.VirtualMemory()
	if err == nil {
		metrics["memory_total"] = memInfo.Total
		metrics["memory_used"] = memInfo.Used
		metrics["memory_used_percent"] = memInfo.UsedPercent
		metrics["memory_available"] = memInfo.Available
	}

	proc, err := process.NewProcess(int32(os.Getpid()))
	if err == nil {
		memInfo, _ := proc.MemoryInfo()
		metrics["process_memory_rss"] = memInfo.RSS
		metrics["process_memory_vms"] = memInfo.VMS

		cpuPercent, _ := proc.CPUPercent()
		metrics["process_cpu_usage"] = cpuPercent

		threads, _ := proc.NumThreads()
		metrics["process_threads"] = threads

		openFiles, _ := proc.NumFDs()
		metrics["process_open_files"] = openFiles
	}

	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	metrics["go_heap_alloc"] = m.Alloc
	metrics["go_heap_sys"] = m.Sys
	metrics["go_heap_idle"] = m.HeapIdle
	metrics["go_heap_inuse"] = m.HeapInuse
	metrics["go_num_gc"] = m.NumGC
	metrics["go_gc_pause_total_ns"] = m.PauseTotalNs
	metrics["go_num_goroutines"] = runtime.NumGoroutine()

	return metrics, nil
}

func (ps *ProfilerService) Stop() {
	close(ps.stopChan)
}

func (ps *ProfilerService) IsEnabled() bool {
	return ps.enabled
}
