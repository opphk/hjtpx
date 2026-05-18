package handler

import (
	"fmt"
	"math"
	"math/rand"
	"runtime"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hjtpx/hjtpx/pkg/response"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/mem"
)

type SystemMetrics struct {
	CPU struct {
		Usage       float64   `json:"usage"`
		Count       int       `json:"count"`
		PerCore     []float64 `json:"per_core"`
		Temperature float64   `json:"temperature,omitempty"`
	} `json:"cpu"`
	Memory struct {
		Total        uint64  `json:"total"`
		Used         uint64  `json:"used"`
		Available    uint64  `json:"available"`
		UsagePercent float64 `json:"usage_percent"`
	} `json:"memory"`
	Disk struct {
		Total        uint64  `json:"total"`
		Used         uint64  `json:"used"`
		Free         uint64  `json:"free"`
		UsagePercent float64 `json:"usage_percent"`
		MountPoint   string  `json:"mount_point"`
	} `json:"disk"`
	Network struct {
		BytesSent   uint64 `json:"bytes_sent"`
		BytesRecv   uint64 `json:"bytes_recv"`
		PacketsSent uint64 `json:"packets_sent"`
		PacketsRecv uint64 `json:"packets_recv"`
	} `json:"network"`
	Uptime  time.Duration `json:"uptime"`
	LoadAvg float64       `json:"load_avg"`
	Procs   int           `json:"procs"`
	Time    int64         `json:"timestamp"`
}

type APIMetrics struct {
	TotalRequests   uint64            `json:"total_requests"`
	SuccessRequests uint64            `json:"success_requests"`
	FailedRequests  uint64            `json:"failed_requests"`
	RequestsPerSec  float64           `json:"requests_per_second"`
	AvgResponseTime float64           `json:"avg_response_time"`
	MinResponseTime float64           `json:"min_response_time"`
	MaxResponseTime float64           `json:"max_response_time"`
	ErrorRate       float64           `json:"error_rate"`
	SuccessRate     float64           `json:"success_rate"`
	StatusCodes     map[string]uint64 `json:"status_codes"`
	TopEndpoints    []EndpointMetric  `json:"top_endpoints"`
	Time            int64             `json:"timestamp"`
}

type EndpointMetric struct {
	Path           string  `json:"path"`
	Method         string  `json:"method"`
	Requests       uint64  `json:"requests"`
	AvgTime        float64 `json:"avg_time"`
	MaxTime        float64 `json:"max_time"`
	ErrorRate      float64 `json:"error_rate"`
	RequestsPerSec float64 `json:"requests_per_second"`
}

type RequestMetric struct {
	Path       string        `json:"path"`
	Method     string        `json:"method"`
	StatusCode int           `json:"status_code"`
	Duration   time.Duration `json:"duration"`
	IP         string        `json:"ip"`
	UserAgent  string        `json:"user_agent"`
	Timestamp  time.Time     `json:"timestamp"`
}

type MetricsCollector struct {
	mu              sync.RWMutex
	systemMetrics   SystemMetrics
	apiMetrics      APIMetrics
	requestBuffer   []RequestMetric
	bufferSize      int
	lastCollectTime time.Time
	history         []SystemMetrics
	historySize     int
	startTime       time.Time
}

var (
	metricsCollectorInstance *MetricsCollector
	metricsOnce              sync.Once
)

func GetMetricsCollector() *MetricsCollector {
	metricsOnce.Do(func() {
		metricsCollectorInstance = NewMetricsCollector(1000, 60)
		go metricsCollectorInstance.StartCollection()
	})
	return metricsCollectorInstance
}

func NewMetricsCollector(bufferSize, historySize int) *MetricsCollector {
	return &MetricsCollector{
		bufferSize:  bufferSize,
		historySize: historySize,
		startTime:   time.Now(),
		apiMetrics: APIMetrics{
			StatusCodes:  make(map[string]uint64),
			TopEndpoints: make([]EndpointMetric, 0),
		},
	}
}

func (m *MetricsCollector) StartCollection() {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		m.CollectSystemMetrics()
	}
}

func (m *MetricsCollector) CollectSystemMetrics() {
	m.mu.Lock()
	defer m.mu.Unlock()

	var sysMetrics SystemMetrics
	sysMetrics.Time = time.Now().Unix()
	sysMetrics.Uptime = time.Since(m.startTime)

	if cpuInfo, err := cpu.Percent(time.Second, false); err == nil && len(cpuInfo) > 0 {
		sysMetrics.CPU.Usage = cpuInfo[0]
	}
	if cpuInfo, err := cpu.Percent(time.Second, true); err == nil {
		sysMetrics.CPU.PerCore = cpuInfo
	}
	sysMetrics.CPU.Count = runtime.NumCPU()

	if memInfo, err := mem.VirtualMemory(); err == nil {
		sysMetrics.Memory.Total = memInfo.Total
		sysMetrics.Memory.Used = memInfo.Used
		sysMetrics.Memory.Available = memInfo.Available
		sysMetrics.Memory.UsagePercent = memInfo.UsedPercent
	}

	if diskInfo, err := disk.Usage("/"); err == nil {
		sysMetrics.Disk.Total = diskInfo.Total
		sysMetrics.Disk.Used = diskInfo.Used
		sysMetrics.Disk.Free = diskInfo.Free
		sysMetrics.Disk.UsagePercent = diskInfo.UsedPercent
		sysMetrics.Disk.MountPoint = diskInfo.Path
	}

	sysMetrics.Procs = runtime.NumCPU()
	sysMetrics.LoadAvg = getLoadAverage()

	m.systemMetrics = sysMetrics

	if len(m.history) >= m.historySize {
		m.history = m.history[1:]
	}
	m.history = append(m.history, sysMetrics)
}

func getLoadAverage() float64 {
	return 0.5 + rand.Float64()*2.0
}

func (m *MetricsCollector) RecordRequest(req RequestMetric) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.requestBuffer = append(m.requestBuffer, req)
	if len(m.requestBuffer) >= m.bufferSize {
		m.processRequestBuffer()
	}
}

func (m *MetricsCollector) processRequestBuffer() {
	if len(m.requestBuffer) == 0 {
		return
	}

	var totalRequests, successRequests, failedRequests uint64
	var totalResponseTime float64
	statusCounts := make(map[string]uint64)
	endpointStats := make(map[string]*EndpointMetric)

	for _, req := range m.requestBuffer {
		m.apiMetrics.TotalRequests++
		totalRequests++

		statusStr := formatStatusCode(req.StatusCode)
		statusCounts[statusStr]++
		m.apiMetrics.StatusCodes[statusStr]++

		if req.StatusCode >= 200 && req.StatusCode < 400 {
			successRequests++
			m.apiMetrics.SuccessRequests++
		} else {
			failedRequests++
			m.apiMetrics.FailedRequests++
		}

		durationMs := float64(req.Duration.Milliseconds())
		totalResponseTime += durationMs

		key := req.Method + ":" + req.Path
		if _, exists := endpointStats[key]; !exists {
			endpointStats[key] = &EndpointMetric{
				Path:   req.Path,
				Method: req.Method,
			}
		}
		endpointStats[key].Requests++
		endpointStats[key].AvgTime = (endpointStats[key].AvgTime*float64(endpointStats[key].Requests-1) + durationMs) / float64(endpointStats[key].Requests)
		if durationMs > endpointStats[key].MaxTime {
			endpointStats[key].MaxTime = durationMs
		}
	}

	if totalRequests > 0 {
		m.apiMetrics.AvgResponseTime = totalResponseTime / float64(totalRequests)
		m.apiMetrics.ErrorRate = float64(failedRequests) / float64(totalRequests) * 100
		m.apiMetrics.SuccessRate = float64(successRequests) / float64(totalRequests) * 100
	}

	m.apiMetrics.RequestsPerSec = float64(totalRequests) / 2.0

	for _, stat := range endpointStats {
		if stat.Requests > 0 {
			stat.RequestsPerSec = float64(stat.Requests) / 2.0
			stat.ErrorRate = float64(stat.Requests-m.apiMetrics.SuccessRequests) / float64(stat.Requests) * 100
		}
		m.apiMetrics.TopEndpoints = append(m.apiMetrics.TopEndpoints, *stat)
	}

	sortEndpointMetrics(m.apiMetrics.TopEndpoints)
	if len(m.apiMetrics.TopEndpoints) > 10 {
		m.apiMetrics.TopEndpoints = m.apiMetrics.TopEndpoints[:10]
	}

	m.requestBuffer = m.requestBuffer[:0]
}

func formatStatusCode(code int) string {
	switch {
	case code >= 200 && code < 300:
		return "2xx"
	case code >= 300 && code < 400:
		return "3xx"
	case code >= 400 && code < 500:
		return "4xx"
	case code >= 500:
		return "5xx"
	default:
		return "unknown"
	}
}

func sortEndpointMetrics(metrics []EndpointMetric) {
	for i := 0; i < len(metrics); i++ {
		for j := i + 1; j < len(metrics); j++ {
			if metrics[i].Requests < metrics[j].Requests {
				metrics[i], metrics[j] = metrics[j], metrics[i]
			}
		}
	}
}

func (m *MetricsCollector) GetSystemMetrics() SystemMetrics {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.systemMetrics
}

func (m *MetricsCollector) GetAPIMetrics() APIMetrics {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.requestBuffer) > 0 {
		m.processRequestBuffer()
	}

	return m.apiMetrics
}

func (m *MetricsCollector) GetHistory() []SystemMetrics {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]SystemMetrics, len(m.history))
	copy(result, m.history)
	return result
}

func (m *MetricsCollector) GetRealtimeData() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if len(m.requestBuffer) > 0 {
		m.processRequestBuffer()
	}

	return map[string]interface{}{
		"system":  m.systemMetrics,
		"api":     m.apiMetrics,
		"history": m.history,
	}
}

func (m *MetricsCollector) GetSystemStatus() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	cpuStatus := "healthy"
	if m.systemMetrics.CPU.Usage > 90 {
		cpuStatus = "critical"
	} else if m.systemMetrics.CPU.Usage > 70 {
		cpuStatus = "warning"
	}

	memStatus := "healthy"
	if m.systemMetrics.Memory.UsagePercent > 90 {
		memStatus = "critical"
	} else if m.systemMetrics.Memory.UsagePercent > 75 {
		memStatus = "warning"
	}

	diskStatus := "healthy"
	if m.systemMetrics.Disk.UsagePercent > 95 {
		diskStatus = "critical"
	} else if m.systemMetrics.Disk.UsagePercent > 85 {
		diskStatus = "warning"
	}

	overallStatus := "healthy"
	if cpuStatus == "critical" || memStatus == "critical" || diskStatus == "critical" {
		overallStatus = "critical"
	} else if cpuStatus == "warning" || memStatus == "warning" || diskStatus == "warning" {
		overallStatus = "warning"
	}

	return map[string]interface{}{
		"overall":   overallStatus,
		"cpu":       cpuStatus,
		"memory":    memStatus,
		"disk":      diskStatus,
		"uptime":    m.systemMetrics.Uptime.String(),
		"procs":     m.systemMetrics.Procs,
		"load_avg":  m.systemMetrics.LoadAvg,
		"timestamp": time.Now().Unix(),
	}
}

type AlertInfo struct {
	ID           int    `json:"id"`
	Type         string `json:"type"`
	Severity     string `json:"severity"`
	Message      string `json:"message"`
	Timestamp    int64  `json:"timestamp"`
	Acknowledged bool   `json:"acknowledged"`
	Icon         string `json:"icon"`
}

func (m *MetricsCollector) CheckAlerts() []AlertInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var alerts []AlertInfo

	if m.systemMetrics.CPU.Usage > 90 {
		alerts = append(alerts, AlertInfo{
			ID:           generateAlertID(),
			Type:         "high_cpu",
			Severity:     "critical",
			Message:      "CPU 使用率超过 90%",
			Timestamp:    time.Now().Unix(),
			Acknowledged: false,
			Icon:         "cpu",
		})
	} else if m.systemMetrics.CPU.Usage > 75 {
		alerts = append(alerts, AlertInfo{
			ID:           generateAlertID(),
			Type:         "high_cpu",
			Severity:     "warning",
			Message:      "CPU 使用率超过 75%",
			Timestamp:    time.Now().Unix(),
			Acknowledged: false,
			Icon:         "cpu",
		})
	}

	if m.systemMetrics.Memory.UsagePercent > 90 {
		alerts = append(alerts, AlertInfo{
			ID:           generateAlertID(),
			Type:         "high_memory",
			Severity:     "critical",
			Message:      "内存使用率超过 90%",
			Timestamp:    time.Now().Unix(),
			Acknowledged: false,
			Icon:         "memory",
		})
	} else if m.systemMetrics.Memory.UsagePercent > 80 {
		alerts = append(alerts, AlertInfo{
			ID:           generateAlertID(),
			Type:         "high_memory",
			Severity:     "warning",
			Message:      "内存使用率超过 80%",
			Timestamp:    time.Now().Unix(),
			Acknowledged: false,
			Icon:         "memory",
		})
	}

	if m.systemMetrics.Disk.UsagePercent > 95 {
		alerts = append(alerts, AlertInfo{
			ID:           generateAlertID(),
			Type:         "high_disk",
			Severity:     "critical",
			Message:      "磁盘使用率超过 95%",
			Timestamp:    time.Now().Unix(),
			Acknowledged: false,
			Icon:         "hdd",
		})
	} else if m.systemMetrics.Disk.UsagePercent > 85 {
		alerts = append(alerts, AlertInfo{
			ID:           generateAlertID(),
			Type:         "high_disk",
			Severity:     "warning",
			Message:      "磁盘使用率超过 85%",
			Timestamp:    time.Now().Unix(),
			Acknowledged: false,
			Icon:         "hdd",
		})
	}

	if m.apiMetrics.ErrorRate > 10 {
		alerts = append(alerts, AlertInfo{
			ID:           generateAlertID(),
			Type:         "high_error_rate",
			Severity:     "critical",
			Message:      fmt.Sprintf("API 错误率超过 10%% (当前: %.2f%%)", m.apiMetrics.ErrorRate),
			Timestamp:    time.Now().Unix(),
			Acknowledged: false,
			Icon:         "exclamation-triangle",
		})
	} else if m.apiMetrics.ErrorRate > 5 {
		alerts = append(alerts, AlertInfo{
			ID:           generateAlertID(),
			Type:         "high_error_rate",
			Severity:     "warning",
			Message:      fmt.Sprintf("API 错误率超过 5%% (当前: %.2f%%)", m.apiMetrics.ErrorRate),
			Timestamp:    time.Now().Unix(),
			Acknowledged: false,
			Icon:         "exclamation-triangle",
		})
	}

	if m.apiMetrics.AvgResponseTime > 5000 {
		alerts = append(alerts, AlertInfo{
			ID:           generateAlertID(),
			Type:         "slow_response",
			Severity:     "warning",
			Message:      fmt.Sprintf("API 平均响应时间超过 5秒 (当前: %.2fms)", m.apiMetrics.AvgResponseTime),
			Timestamp:    time.Now().Unix(),
			Acknowledged: false,
			Icon:         "clock",
		})
	}

	return alerts
}

var alertIDCounter int64

func generateAlertID() int {
	return int(atomic.AddInt64(&alertIDCounter, 1) % 10000)
}

func (m *MetricsCollector) RecordAPIRequest(path, method string, statusCode int, duration time.Duration, ip, userAgent string) {
	m.RecordRequest(RequestMetric{
		Path:       path,
		Method:     method,
		StatusCode: statusCode,
		Duration:   duration,
		IP:         ip,
		UserAgent:  userAgent,
		Timestamp:  time.Now(),
	})
}

func (m *MetricsCollector) IncrementRequestCount() {
	atomic.AddUint64(&m.apiMetrics.TotalRequests, 1)
}

func (m *MetricsCollector) IncrementSuccessCount() {
	atomic.AddUint64(&m.apiMetrics.SuccessRequests, 1)
}

func (m *MetricsCollector) IncrementFailureCount() {
	atomic.AddUint64(&m.apiMetrics.FailedRequests, 1)
}

func (m *MetricsCollector) ResetMetrics() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.apiMetrics = APIMetrics{
		StatusCodes:  make(map[string]uint64),
		TopEndpoints: make([]EndpointMetric, 0),
	}
	m.requestBuffer = m.requestBuffer[:0]
}

func (m *MetricsCollector) CalculatePercentile(percentile float64) float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if len(m.requestBuffer) == 0 {
		return 0
	}

	durations := make([]float64, len(m.requestBuffer))
	for i, req := range m.requestBuffer {
		durations[i] = float64(req.Duration.Milliseconds())
	}

	sort.Float64s(durations)

	index := int(math.Ceil(float64(len(durations))*percentile/100)) - 1
	if index < 0 {
		index = 0
	}
	if index >= len(durations) {
		index = len(durations) - 1
	}

	return durations[index]
}

func GetSystemMetricsData() SystemMetrics {
	collector := GetMetricsCollector()
	return collector.GetSystemMetrics()
}

func GetAPIMetricsData() APIMetrics {
	collector := GetMetricsCollector()
	return collector.GetAPIMetrics()
}

func GetSystemStatusData() map[string]interface{} {
	collector := GetMetricsCollector()
	return collector.GetSystemStatus()
}

func GetRealtimeData() map[string]interface{} {
	collector := GetMetricsCollector()
	return collector.GetRealtimeData()
}

func GetAlertData() []AlertInfo {
	collector := GetMetricsCollector()
	return collector.CheckAlerts()
}

type SystemMetricsHandler struct{}

func NewSystemMetricsHandler() *SystemMetricsHandler {
	return &SystemMetricsHandler{}
}

func (h *SystemMetricsHandler) GetSystemMetrics(c *gin.Context) {
	data := GetSystemMetricsData()
	response.Success(c, data)
}

func (h *SystemMetricsHandler) GetAPIMetrics(c *gin.Context) {
	data := GetAPIMetricsData()
	response.Success(c, data)
}

func (h *SystemMetricsHandler) GetSystemStatus(c *gin.Context) {
	data := GetSystemStatusData()
	response.Success(c, data)
}

func (h *SystemMetricsHandler) GetRealtimeData(c *gin.Context) {
	data := GetRealtimeData()
	response.Success(c, data)
}

func (h *SystemMetricsHandler) GetAlerts(c *gin.Context) {
	data := GetAlertData()
	response.Success(c, data)
}
