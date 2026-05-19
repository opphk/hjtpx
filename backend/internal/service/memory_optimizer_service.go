package service

import (
	"math/rand"
	"runtime"
	"sync"
	"time"
)

type MemoryOptimizerService struct {
	memoryLimit int64
	pressureLevel string
	mu           sync.RWMutex
}

type MemoryUsage struct {
	Used      int64  `json:"used"`
	Total     int64  `json:"total"`
	Percent   float64 `json:"percent"`
	Timestamp time.Time `json:"timestamp"`
}

type MemoryAnalysis struct {
	LeaksDetected bool     `json:"leaks_detected"`
	Suggestions   []string `json:"suggestions"`
	Details       map[string]interface{} `json:"details"`
}

type MemoryStats struct {
	Alloc         int64 `json:"alloc"`
	TotalAlloc    int64 `json:"total_alloc"`
	Sys           int64 `json:"sys"`
	NumGC         int   `json:"num_gc"`
	HeapAlloc     int64 `json:"heap_alloc"`
	HeapSys       int64 `json:"heap_sys"`
	HeapIdle      int64 `json:"heap_idle"`
	HeapInuse     int64 `json:"heap_inuse"`
}

type MemoryMonitoringData struct {
	CurrentUsage MemoryUsage `json:"current_usage"`
	Trend        string      `json:"trend"`
	Alerts       []string    `json:"alerts"`
}

func NewMemoryOptimizerService() *MemoryOptimizerService {
	return &MemoryOptimizerService{
		memoryLimit:   1024 * 1024 * 1024,
		pressureLevel: "normal",
	}
}

func (o *MemoryOptimizerService) OptimizeMemory() error {
	runtime.GC()
	return nil
}

func (o *MemoryOptimizerService) GetMemoryUsage() (*MemoryUsage, error) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	
	return &MemoryUsage{
		Used:      int64(m.Alloc),
		Total:     int64(m.Sys),
		Percent:   float64(m.Alloc) / float64(m.Sys) * 100,
		Timestamp: time.Now(),
	}, nil
}

func (o *MemoryOptimizerService) AnalyzeMemoryLeaks() (*MemoryAnalysis, error) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	
	analysis := &MemoryAnalysis{
		LeaksDetected: false,
		Suggestions:   []string{},
		Details:       map[string]interface{}{},
	}
	
	if m.NumGC > 100 && m.Alloc > 100*1024*1024 {
		analysis.LeaksDetected = true
		analysis.Suggestions = append(analysis.Suggestions, "Possible memory leak detected")
	}
	
	analysis.Details["num_gc"] = m.NumGC
	analysis.Details["alloc"] = m.Alloc
	analysis.Details["heap_objects"] = m.HeapObjects
	
	return analysis, nil
}

func (o *MemoryOptimizerService) ClearMemoryCache() error {
	runtime.GC()
	return nil
}

func (o *MemoryOptimizerService) SetMemoryLimit(limit int64) error {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.memoryLimit = limit
	return nil
}

func (o *MemoryOptimizerService) GetMemoryLimit() (int64, error) {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return o.memoryLimit, nil
}

func (o *MemoryOptimizerService) MonitorMemory() (*MemoryMonitoringData, error) {
	usage, _ := o.GetMemoryUsage()
	
	trend := "stable"
	if usage.Percent > 80 {
		trend = "increasing"
	} else if usage.Percent < 30 {
		trend = "decreasing"
	}
	
	alerts := []string{}
	if usage.Percent > 90 {
		alerts = append(alerts, "Memory usage exceeds 90%")
	}
	
	return &MemoryMonitoringData{
		CurrentUsage: *usage,
		Trend:        trend,
		Alerts:       alerts,
	}, nil
}

func (o *MemoryOptimizerService) GetMemoryStats() (*MemoryStats, error) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	
	return &MemoryStats{
		Alloc:     int64(m.Alloc),
		TotalAlloc: int64(m.TotalAlloc),
		Sys:       int64(m.Sys),
		NumGC:     int(m.NumGC),
		HeapAlloc: int64(m.HeapAlloc),
		HeapSys:   int64(m.HeapSys),
		HeapIdle:  int64(m.HeapIdle),
		HeapInuse: int64(m.HeapInuse),
	}, nil
}

func (o *MemoryOptimizerService) TriggerGC() error {
	runtime.GC()
	return nil
}

func (o *MemoryOptimizerService) SetMemoryPressureLevel(level string) error {
	validLevels := map[string]bool{"low": true, "normal": true, "high": true, "critical": true}
	if !validLevels[level] {
		return nil
	}
	o.mu.Lock()
	defer o.mu.Unlock()
	o.pressureLevel = level
	return nil
}

func (o *MemoryOptimizerService) GetMemoryPressureLevel() (string, error) {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return o.pressureLevel, nil
}
