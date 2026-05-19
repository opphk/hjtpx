package service

import (
	"context"
	"sync"
	"sync/atomic"
	"time"
)

type DataStream struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Type        string            `json:"type"`
	Status      string            `json:"status"`
	RecordsIn   atomic.Int64      `json:"records_in"`
	RecordsOut  atomic.Int64      `json:"records_out"`
	Errors      atomic.Int64      `json:"errors"`
	LatencyMs   atomic.Int64      `json:"latency_ms"`
	Throughput  atomic.Int64      `json:"throughput_per_sec"`
	BufferSize  atomic.Int32      `json:"buffer_size"`
	Metadata    map[string]string `json:"metadata"`
}

type StreamProcessor struct {
	ID            string          `json:"id"`
	Name          string          `json:"name"`
	Type          string          `json:"type"`
	Stream        *DataStream     `json:"stream"`
	Handler       func(interface{}) interface{}
	Filter        func(interface{}) bool
	Transform     func(interface{}) interface{}
	Aggregator    func([]interface{}) interface{}
	IsRunning     atomic.Bool      `json:"is_running"`
	mu            sync.RWMutex
	records       []interface{}
	buffer        chan interface{}
	errorCount    atomic.Int64
	processTime   atomic.Int64
}

type BatchConfig struct {
	BatchSize        int `json:"batch_size"`
	BatchTimeoutSec  int `json:"batch_timeout_seconds"`
	MaxRetries       int `json:"max_retries"`
	RetryDelaySec    int `json:"retry_delay_seconds"`
	ParallelBatches  int `json:"parallel_batches"`
}

type BatchJob struct {
	ID           string          `json:"id"`
	Name         string          `json:"name"`
	Type         string          `json:"type"`
	Status       string          `json:"status"`
	RecordsTotal int64           `json:"records_total"`
	RecordsProcessed int64        `json:"records_processed"`
	RecordsFailed int64          `json:"records_failed"`
	StartedAt    time.Time       `json:"started_at"`
	CompletedAt  time.Time       `json:"completed_at"`
	Duration     time.Duration   `json:"duration"`
	ErrorMessage string          `json:"error_message"`
}

type DataQualityRule struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Field       string  `json:"field"`
	Type        string  `json:"type"`
	Condition   string  `json:"condition"`
	Threshold   float64 `json:"threshold"`
	Severity    string  `json:"severity"`
	Enabled     bool    `json:"enabled"`
}

type DataQualityMetrics struct {
	TotalRecords    atomic.Int64   `json:"total_records"`
	ValidRecords    atomic.Int64   `json:"valid_records"`
	InvalidRecords  atomic.Int64   `json:"invalid_records"`
	DuplicateRecords atomic.Int64  `json:"duplicate_records"`
	NullRecords     atomic.Int64   `json:"null_records"`
	QualityScore    float64        `json:"quality_score_percent"`
	LastCheckTime   atomic.Int64   `json:"last_check_time"`
	Violations      []QualityViolation `json:"violations"`
	mu              sync.RWMutex
}

type QualityViolation struct {
	ID        string    `json:"id"`
	RuleID    string    `json:"rule_id"`
	RuleName  string    `json:"rule_name"`
	Severity  string    `json:"severity"`
	Field     string    `json:"field"`
	Message   string    `json:"message"`
	Count     int       `json:"count"`
	FirstSeen time.Time `json:"first_seen"`
	LastSeen  time.Time `json:"last_seen"`
}

type AnalyticsDashboard struct {
	TotalEvents      atomic.Int64   `json:"total_events"`
	EventsPerSecond  float64        `json:"events_per_second"`
	AvgLatencyMs     atomic.Int64   `json:"avg_latency_ms"`
	P95LatencyMs     atomic.Int64   `json:"p95_latency_ms"`
	P99LatencyMs     atomic.Int64   `json:"p99_latency_ms"`
	ErrorRate        float64        `json:"error_rate_percent"`
	SuccessRate      float64        `json:"success_rate_percent"`
	ActiveStreams    atomic.Int32   `json:"active_streams"`
	ActiveBatches    atomic.Int32   `json:"active_batches"`
	LastUpdated      atomic.Int64   `json:"last_updated"`
}

type DataPipeline struct {
	mu          sync.RWMutex
	streams     map[string]*StreamProcessor
	batchJobs   map[string]*BatchJob
	qualityRules []DataQualityRule
	qualityMetrics DataQualityMetrics
	dashboard   AnalyticsDashboard
	config      PipelineConfig
	batchConfig BatchConfig
}

type PipelineConfig struct {
	BufferSize          int     `json:"buffer_size"`
	WorkerCount         int     `json:"worker_count"`
	MaxLatencyMs        int64   `json:"max_latency_ms"`
	EnableMonitoring    bool    `json:"enable_monitoring"`
	MonitoringIntervalSec int   `json:"monitoring_interval_seconds"`
	EnableQualityCheck  bool    `json:"enable_quality_check"`
	EnableAggregation   bool    `json:"enable_aggregation"`
}

func NewDataPipeline() *DataPipeline {
	dp := &DataPipeline{
		streams:      make(map[string]*StreamProcessor),
		batchJobs:    make(map[string]*BatchJob),
		qualityRules: make([]DataQualityRule, 0),
		qualityMetrics: DataQualityMetrics{
			Violations: make([]QualityViolation, 0),
		},
		config: PipelineConfig{
			BufferSize:          10000,
			WorkerCount:         4,
			MaxLatencyMs:        1000,
			EnableMonitoring:    true,
			MonitoringIntervalSec: 60,
			EnableQualityCheck:  true,
			EnableAggregation:   true,
		},
		batchConfig: BatchConfig{
			BatchSize:       1000,
			BatchTimeoutSec: 30,
			MaxRetries:      3,
			RetryDelaySec:   5,
			ParallelBatches: 2,
		},
	}

	dp.qualityRules = append(dp.qualityRules,
		DataQualityRule{ID: "rule-1", Name: "Not Null", Field: "user_id", Type: "not_null", Enabled: true, Severity: "error"},
		DataQualityRule{ID: "rule-2", Name: "Valid Email", Field: "email", Type: "regex", Condition: "^[a-zA-Z0-9+_.-]+@[a-zA-Z0-9.-]+$", Enabled: true, Severity: "warning"},
		DataQualityRule{ID: "rule-3", Name: "Positive Amount", Field: "amount", Type: "range", Condition: ">0", Enabled: true, Severity: "error"},
	)

	return dp
}

func (dp *DataPipeline) CreateStream(name string, streamType string) (*StreamProcessor, error) {
	dp.mu.Lock()
	defer dp.mu.Unlock()

	stream := &DataStream{
		ID:   generateID(),
		Name: name,
		Type: streamType,
		Status: "idle",
		Metadata:   make(map[string]string),
	}
	stream.BufferSize.Store(int32(dp.config.BufferSize))

	processor := &StreamProcessor{
		ID:      generateID(),
		Name:    name,
		Type:    streamType,
		Stream:  stream,
		buffer:  make(chan interface{}, dp.config.BufferSize),
		records: make([]interface{}, 0),
	}

	dp.streams[processor.ID] = processor

	return processor, nil
}

func (sp *StreamProcessor) Start(ctx context.Context) {
	sp.IsRunning.Store(true)
	sp.Stream.Status = "running"

	go sp.processLoop(ctx)
}

func (sp *StreamProcessor) processLoop(ctx context.Context) {
	for sp.IsRunning.Load() {
		select {
		case <-ctx.Done():
			sp.Stop()
			return
		case data := <-sp.buffer:
			start := time.Now()

			if sp.Filter != nil && !sp.Filter(data) {
				continue
			}

			if sp.Transform != nil {
				data = sp.Transform(data)
			}

			if sp.Aggregator != nil {
				sp.mu.Lock()
				sp.records = append(sp.records, data)
				if len(sp.records) >= 100 {
					result := sp.Aggregator(sp.records)
					sp.records = sp.records[:0]
					_ = result
				}
				sp.mu.Unlock()
			}

			sp.Stream.RecordsOut.Add(1)
			latency := time.Since(start).Milliseconds()
			sp.Stream.LatencyMs.Store(latency)
			sp.processTime.Add(latency)

		default:
			time.Sleep(10 * time.Millisecond)
		}
	}
}

func (sp *StreamProcessor) Stop() {
	sp.IsRunning.Store(false)
	sp.Stream.Status = "stopped"
	close(sp.buffer)
}

func (sp *StreamProcessor) Push(data interface{}) bool {
	if !sp.IsRunning.Load() || sp.Stream.BufferSize.Load() <= 0 {
		sp.Stream.Errors.Add(1)
		return false
	}

	select {
	case sp.buffer <- data:
		sp.Stream.RecordsIn.Add(1)
		return true
	default:
		sp.Stream.Errors.Add(1)
		return false
	}
}

func (dp *DataPipeline) GetStream(id string) (*StreamProcessor, bool) {
	dp.mu.RLock()
	defer dp.mu.RUnlock()

	sp, exists := dp.streams[id]
	return sp, exists
}

func (dp *DataPipeline) GetAllStreams() []*StreamProcessor {
	dp.mu.RLock()
	defer dp.mu.RUnlock()

	streams := make([]*StreamProcessor, 0, len(dp.streams))
	for _, sp := range dp.streams {
		streams = append(streams, sp)
	}

	return streams
}

func (dp *DataPipeline) CreateBatchJob(name, jobType string, records []interface{}) *BatchJob {
	dp.mu.Lock()
	defer dp.mu.Unlock()

	job := &BatchJob{
		ID:            generateID(),
		Name:          name,
		Type:          jobType,
		Status:        "pending",
		RecordsTotal:  int64(len(records)),
		RecordsProcessed: 0,
		RecordsFailed: 0,
		StartedAt:     time.Now(),
	}

	dp.batchJobs[job.ID] = job

	go dp.processBatch(job, records)

	return job
}

func (dp *DataPipeline) processBatch(job *BatchJob, records []interface{}) {
	job.Status = "running"

	batchSize := dp.batchConfig.BatchSize
	var processed, failed int64

	for i := 0; i < len(records); i += batchSize {
		end := i + batchSize
		if end > len(records) {
			end = len(records)
		}

		batch := records[i:end]

		for _, record := range batch {
			if dp.config.EnableQualityCheck {
				violations := dp.checkDataQuality(record)
				if len(violations) > 0 {
					failed++
					continue
				}
			}

			processed++
		}

		job.RecordsProcessed = processed
		job.RecordsFailed = failed

		time.Sleep(10 * time.Millisecond)
	}

	job.Status = "completed"
	job.CompletedAt = time.Now()
	job.Duration = job.CompletedAt.Sub(job.StartedAt)
}

func (dp *DataPipeline) checkDataQuality(record interface{}) []QualityViolation {
	dp.mu.RLock()
	rules := dp.qualityRules
	dp.mu.RUnlock()

	violations := make([]QualityViolation, 0)

	data, ok := record.(map[string]interface{})
	if !ok {
		return violations
	}

	for _, rule := range rules {
		if !rule.Enabled {
			continue
		}

		value, exists := data[rule.Field]
		if !exists || value == nil {
			if rule.Type == "not_null" {
				violations = append(violations, QualityViolation{
					ID:       generateID(),
					RuleID:   rule.ID,
					RuleName: rule.Name,
					Severity: rule.Severity,
					Field:    rule.Field,
					Message:  "Field is null",
					Count:    1,
					FirstSeen: time.Now(),
					LastSeen:  time.Now(),
				})
				dp.qualityMetrics.NullRecords.Add(1)
			}
			continue
		}

		dp.qualityMetrics.ValidRecords.Add(1)
	}

	dp.qualityMetrics.TotalRecords.Add(1)
	dp.updateQualityScore()

	return violations
}

func (dp *DataPipeline) updateQualityScore() {
	total := dp.qualityMetrics.TotalRecords.Load()
	valid := dp.qualityMetrics.ValidRecords.Load()
	invalid := dp.qualityMetrics.InvalidRecords.Load()
	duplicates := dp.qualityMetrics.DuplicateRecords.Load()

	if total > 0 {
		score := float64(valid) / float64(total) * 100
		score -= float64(invalid) * 0.5
		score -= float64(duplicates) * 0.3
		if score < 0 {
			score = 0
		}
		dp.qualityMetrics.QualityScore = score
	}

	dp.qualityMetrics.LastCheckTime.Store(time.Now().Unix())
}

func (dp *DataPipeline) GetBatchJob(id string) (*BatchJob, bool) {
	dp.mu.RLock()
	defer dp.mu.RUnlock()

	job, exists := dp.batchJobs[id]
	return job, exists
}

func (dp *DataPipeline) GetAllBatchJobs() []*BatchJob {
	dp.mu.RLock()
	defer dp.mu.RUnlock()

	jobs := make([]*BatchJob, 0, len(dp.batchJobs))
	for _, job := range dp.batchJobs {
		jobs = append(jobs, job)
	}

	return jobs
}

func (dp *DataPipeline) AddQualityRule(rule DataQualityRule) {
	dp.mu.Lock()
	defer dp.mu.Unlock()

	dp.qualityRules = append(dp.qualityRules, rule)
}

func (dp *DataPipeline) GetQualityRules() []DataQualityRule {
	dp.mu.RLock()
	defer dp.mu.RUnlock()

	rules := make([]DataQualityRule, len(dp.qualityRules))
	copy(rules, dp.qualityRules)

	return rules
}

func (dp *DataPipeline) GetQualityMetrics() DataQualityMetrics {
	dp.mu.RLock()
	defer dp.mu.RUnlock()

	metrics := dp.qualityMetrics
	violations := make([]QualityViolation, len(metrics.Violations))
	copy(violations, metrics.Violations)
	metrics.Violations = violations

	return metrics
}

func (dp *DataPipeline) GetDashboardMetrics() AnalyticsDashboard {
	dp.mu.RLock()
	defer dp.mu.RUnlock()

	dp.dashboard.TotalEvents.Add(1)
	dp.dashboard.LastUpdated.Store(time.Now().Unix())

	streams := dp.GetAllStreams()
	var totalThroughput int64
	var totalErrors int64
	var totalLatency int64
	var runningStreams int32

	for _, sp := range streams {
		if sp.IsRunning.Load() {
			runningStreams++
			totalThroughput += sp.Stream.Throughput.Load()
			totalErrors += sp.Stream.Errors.Load()
			totalLatency += sp.Stream.LatencyMs.Load()
		}
	}

	if runningStreams > 0 {
		dp.dashboard.EventsPerSecond = float64(totalThroughput) / float64(runningStreams)
		dp.dashboard.AvgLatencyMs.Store(totalLatency / int64(runningStreams))
	}

	dp.dashboard.ActiveStreams.Store(runningStreams)
	dp.dashboard.ActiveBatches.Store(int32(len(dp.batchJobs)))

	batchJobs := dp.GetAllBatchJobs()
	var totalRecords, totalFailed int64
	for _, job := range batchJobs {
		if job.Status == "running" {
			totalRecords += job.RecordsProcessed
			totalFailed += job.RecordsFailed
		}
	}

	if totalRecords > 0 {
		dp.dashboard.ErrorRate = float64(totalFailed) / float64(totalRecords) * 100
		dp.dashboard.SuccessRate = float64(totalRecords-totalFailed) / float64(totalRecords) * 100
	}

	return dp.dashboard
}

func (dp *DataPipeline) GetPipelineStats() map[string]interface{} {
	streams := dp.GetAllStreams()
	batchJobs := dp.GetAllBatchJobs()

	var totalStreamRecordsIn, totalStreamRecordsOut, totalStreamErrors int64
	var runningStreams, stoppedStreams int

	for _, sp := range streams {
		totalStreamRecordsIn += sp.Stream.RecordsIn.Load()
		totalStreamRecordsOut += sp.Stream.RecordsOut.Load()
		totalStreamErrors += sp.Stream.Errors.Load()

		if sp.IsRunning.Load() {
			runningStreams++
		} else {
			stoppedStreams++
		}
	}

	var completedJobs, runningJobs, failedJobs int
	var totalBatchRecords, totalBatchProcessed int64

	for _, job := range batchJobs {
		totalBatchRecords += job.RecordsTotal
		totalBatchProcessed += job.RecordsProcessed

		switch job.Status {
		case "completed":
			completedJobs++
		case "running":
			runningJobs++
		case "failed":
			failedJobs++
		}
	}

	qualityMetrics := dp.GetQualityMetrics()

	return map[string]interface{}{
		"streams": map[string]interface{}{
			"total":            len(streams),
			"running":          runningStreams,
			"stopped":          stoppedStreams,
			"records_in":       totalStreamRecordsIn,
			"records_out":      totalStreamRecordsOut,
			"errors":           totalStreamErrors,
		},
		"batch_jobs": map[string]interface{}{
			"total":            len(batchJobs),
			"completed":        completedJobs,
			"running":          runningJobs,
			"failed":           failedJobs,
			"total_records":    totalBatchRecords,
			"processed":       totalBatchProcessed,
		},
		"quality": map[string]interface{}{
			"total_records":    qualityMetrics.TotalRecords.Load(),
			"valid_records":    qualityMetrics.ValidRecords.Load(),
			"invalid_records":  qualityMetrics.InvalidRecords.Load(),
			"quality_score":    qualityMetrics.QualityScore,
			"violations_count": len(qualityMetrics.Violations),
		},
		"dashboard": map[string]interface{}{
			"events_per_second": dp.dashboard.EventsPerSecond,
			"avg_latency_ms":    dp.dashboard.AvgLatencyMs.Load(),
			"error_rate":        dp.dashboard.ErrorRate,
			"success_rate":      dp.dashboard.SuccessRate,
		},
	}
}

func (dp *DataPipeline) ProcessStreamData(streamID string, data interface{}) error {
	sp, exists := dp.GetStream(streamID)
	if !exists {
		return ErrNotFound
	}

	if !sp.Push(data) {
		return ErrBufferFull
	}

	dp.dashboard.TotalEvents.Add(1)

	return nil
}

func (dp *DataPipeline) StartStream(ctx context.Context, streamID string) error {
	sp, exists := dp.GetStream(streamID)
	if !exists {
		return ErrNotFound
	}

	sp.Start(ctx)
	return nil
}

func (dp *DataPipeline) StopStream(streamID string) error {
	sp, exists := dp.GetStream(streamID)
	if !exists {
		return ErrNotFound
	}

	sp.Stop()
	return nil
}

func (dp *DataPipeline) ResetMetrics() {
	dp.qualityMetrics = DataQualityMetrics{
		Violations: make([]QualityViolation, 0),
	}

	dp.dashboard = AnalyticsDashboard{}

	for _, sp := range dp.streams {
		sp.Stream.RecordsIn.Store(0)
		sp.Stream.RecordsOut.Store(0)
		sp.Stream.Errors.Store(0)
	}
}
