package logging

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/hjtpx/hjtpx/internal/pkg/logger"
)

type LogLevel string

const (
	LogLevelDebug   LogLevel = "debug"
	LogLevelInfo    LogLevel = "info"
	LogLevelWarn    LogLevel = "warn"
	LogLevelError   LogLevel = "error"
	LogLevelFatal   LogLevel = "fatal"
)

type LogEntry struct {
	Timestamp   time.Time              `json:"timestamp"`
	Level       LogLevel               `json:"level"`
	Service     string                 `json:"service"`
	Component   string                 `json:"component"`
	Message     string                 `json:"message"`
	TraceID     string                 `json:"trace_id,omitempty"`
	SpanID      string                 `json:"span_id,omitempty"`
	Error       string                 `json:"error,omitempty"`
	Stacktrace  string                 `json:"stacktrace,omitempty"`
	Fields      map[string]interface{} `json:"fields,omitempty"`
	Source      string                 `json:"source"`
}

type LogQuery struct {
	Levels      []LogLevel
	Services    []string
	Components  []string
	SearchText  string
	TraceID     string
	StartTime   time.Time
	EndTime     time.Time
	Limit       int
	Offset      int
}

type LogAggregationResult struct {
	Total      int64      `json:"total"`
	Logs       []LogEntry `json:"logs"`
	Stats      LogStats   `json:"stats"`
}

type LogStats struct {
	TotalLogs    int64             `json:"total_logs"`
	ByLevel      map[LogLevel]int64 `json:"by_level"`
	ByService    map[string]int64   `json:"by_service"`
	ByComponent  map[string]int64   `json:"by_component"`
	TimeRange    TimeRange         `json:"time_range"`
}

type TimeRange struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

type LogAggregator struct {
	mu            sync.RWMutex
	logs          []LogEntry
	maxEntries    int
	logFiles      []string
	fileWatcher   *FileWatcher
	enabled       bool
}

func NewLogAggregator(maxEntries int, enabled bool) *LogAggregator {
	la := &LogAggregator{
		logs:       make([]LogEntry, 0),
		maxEntries: maxEntries,
		enabled:    enabled,
	}

	if enabled {
		la.fileWatcher = NewFileWatcher(la)
	}

	return la
}

func (la *LogAggregator) AddEntry(entry LogEntry) {
	if !la.enabled {
		return
	}

	la.mu.Lock()
	defer la.mu.Unlock()

	entry.Timestamp = time.Now()
	la.logs = append(la.logs, entry)

	if len(la.logs) > la.maxEntries {
		la.logs = la.logs[len(la.logs)-la.maxEntries:]
	}
}

func (la *LogAggregator) QueryLogs(query LogQuery) *LogAggregationResult {
	la.mu.RLock()
	defer la.mu.RUnlock()

	var filtered []LogEntry
	levelSet := make(map[LogLevel]bool)
	serviceSet := make(map[string]bool)
	componentSet := make(map[string]bool)

	for _, l := range query.Levels {
		levelSet[l] = true
	}
	for _, s := range query.Services {
		serviceSet[s] = true
	}
	for _, c := range query.Components {
		componentSet[c] = true
	}

	for _, entry := range la.logs {
		if len(levelSet) > 0 && !levelSet[entry.Level] {
			continue
		}
		if len(serviceSet) > 0 && !serviceSet[entry.Service] {
			continue
		}
		if len(componentSet) > 0 && !componentSet[entry.Component] {
			continue
		}
		if query.TraceID != "" && entry.TraceID != query.TraceID {
			continue
		}
		if !query.StartTime.IsZero() && entry.Timestamp.Before(query.StartTime) {
			continue
		}
		if !query.EndTime.IsZero() && entry.Timestamp.After(query.EndTime) {
			continue
		}
		if query.SearchText != "" {
			if !containsIgnoreCase(entry.Message, query.SearchText) &&
				!containsIgnoreCase(entry.Service, query.SearchText) &&
				!containsIgnoreCase(entry.Component, query.SearchText) {
				continue
			}
		}

		filtered = append(filtered, entry)
	}

	for i := 0; i < len(filtered)-1; i++ {
		for j := i + 1; j < len(filtered); j++ {
			if filtered[j].Timestamp.After(filtered[i].Timestamp) {
				filtered[i], filtered[j] = filtered[j], filtered[i]
			}
		}
	}

	start := query.Offset
	end := query.Offset + query.Limit
	if start > len(filtered) {
		start = len(filtered)
	}
	if end > len(filtered) {
		end = len(filtered)
	}

	result := &LogAggregationResult{
		Total: int64(len(filtered)),
		Logs:  filtered[start:end],
		Stats: la.calculateStats(filtered),
	}

	return result
}

func containsIgnoreCase(a, b string) bool {
	if len(b) == 0 {
		return true
	}
	if len(a) < len(b) {
		return false
	}

	for i := 0; i <= len(a)-len(b); i++ {
		match := true
		for j := 0; j < len(b); j++ {
			ac := a[i+j]
			bc := b[j]
			if ac >= 'A' && ac <= 'Z' {
				ac += 32
			}
			if bc >= 'A' && bc <= 'Z' {
				bc += 32
			}
			if ac != bc {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}

func (la *LogAggregator) calculateStats(logs []LogEntry) LogStats {
	stats := LogStats{
		ByLevel:     make(map[LogLevel]int64),
		ByService:   make(map[string]int64),
		ByComponent: make(map[string]int64),
		TimeRange: TimeRange{
			Start: time.Now(),
			End:   time.Time{},
		},
	}

	if len(logs) == 0 {
		return stats
	}

	stats.TotalLogs = int64(len(logs))

	for _, entry := range logs {
		stats.ByLevel[entry.Level]++
		stats.ByService[entry.Service]++
		stats.ByComponent[entry.Component]++

		if entry.Timestamp.Before(stats.TimeRange.Start) {
			stats.TimeRange.Start = entry.Timestamp
		}
		if entry.Timestamp.After(stats.TimeRange.End) {
			stats.TimeRange.End = entry.Timestamp
		}
	}

	return stats
}

func (la *LogAggregator) GetRecentLogs(count int) []LogEntry {
	la.mu.RLock()
	defer la.mu.RUnlock()

	if count > len(la.logs) {
		count = len(la.logs)
	}

	return la.logs[len(la.logs)-count:]
}

func (la *LogAggregator) GetLogsByTraceID(traceID string) []LogEntry {
	la.mu.RLock()
	defer la.mu.RUnlock()

	var result []LogEntry
	for _, entry := range la.logs {
		if entry.TraceID == traceID {
			result = append(result, entry)
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

func (la *LogAggregator) ExportLogsJSON(query LogQuery) ([]byte, error) {
	result := la.QueryLogs(query)
	return json.MarshalIndent(result, "", "  ")
}

func (la *LogAggregator) TailLogs(ctx interface{}, callback func(entry LogEntry)) {
	lastIndex := len(la.logs)

	go func() {
		for {
			la.mu.RLock()
			currentLen := len(la.logs)
			la.mu.RUnlock()

			if currentLen > lastIndex {
				la.mu.RLock()
				newEntries := la.logs[lastIndex:currentLen]
				la.mu.RUnlock()

				for _, entry := range newEntries {
					callback(entry)
				}
				lastIndex = currentLen
			}

			time.Sleep(100 * time.Millisecond)
		}
	}()
}

func (la *LogAggregator) LoadLogFiles(paths []string) error {
	for _, path := range paths {
		err := la.loadLogFile(path)
		if err != nil {
			logger.Error(fmt.Sprintf("Failed to load log file %s: %v", path, err))
		}
	}
	return nil
}

func (la *LogAggregator) loadLogFile(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		entry := parseLogLine(line, filepath.Base(path))
		if entry != nil {
			la.AddEntry(*entry)
		}
	}

	return scanner.Err()
}

func parseLogLine(line string, source string) *LogEntry {
	var entry LogEntry
	if err := json.Unmarshal([]byte(line), &entry); err == nil {
		entry.Source = source
		return &entry
	}

	return &LogEntry{
		Timestamp: time.Now(),
		Level:     LogLevelInfo,
		Message:   line,
		Source:    source,
	}
}

func (la *LogAggregator) WatchLogFiles(paths []string) {
	if la.fileWatcher != nil {
		la.fileWatcher.Watch(paths)
	}
}

type FileWatcher struct {
	aggregator *LogAggregator
	watching   bool
}

func NewFileWatcher(aggregator *LogAggregator) *FileWatcher {
	return &FileWatcher{
		aggregator: aggregator,
	}
}

func (fw *FileWatcher) Watch(paths []string) {
	if fw.watching {
		return
	}
	fw.watching = true

	for _, path := range paths {
		go fw.watchFile(path)
	}
}

func (fw *FileWatcher) watchFile(path string) {
	for fw.watching {
		file, err := os.Open(path)
		if err != nil {
			time.Sleep(5 * time.Second)
			continue
		}

		stat, _ := file.Stat()
		initialSize := stat.Size()

		reader := bufio.NewReader(file)
		file.Seek(initialSize, io.SeekStart)

		for fw.watching {
			line, err := reader.ReadString('\n')
			if err != nil {
				if err == io.EOF {
					time.Sleep(100 * time.Millisecond)
					continue
				}
				break
			}

			entry := parseLogLine(line, filepath.Base(path))
			if entry != nil {
				fw.aggregator.AddEntry(*entry)
			}
		}

		file.Close()
	}
}

func (fw *FileWatcher) Stop() {
	fw.watching = false
}
