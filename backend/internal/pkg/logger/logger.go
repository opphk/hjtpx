package logger

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type Level int

const (
	DEBUG Level = iota
	INFO
	WARN
	ERROR
	FATAL
)

var levelNames = map[Level]string{
	DEBUG: "DEBUG",
	INFO:  "INFO",
	WARN:  "WARN",
	ERROR: "ERROR",
	FATAL: "FATAL",
}

func (l Level) String() string {
	if name, ok := levelNames[l]; ok {
		return name
	}
	return "UNKNOWN"
}

func ParseLevel(s string) Level {
	switch s {
	case "debug", "DEBUG":
		return DEBUG
	case "info", "INFO":
		return INFO
	case "warn", "WARN", "warning", "WARNING":
		return WARN
	case "error", "ERROR":
		return ERROR
	case "fatal", "FATAL":
		return FATAL
	default:
		return INFO
	}
}

type Fields map[string]interface{}

type LogEntry struct {
	Time      string    `json:"time"`
	Level     string    `json:"level"`
	Message   string    `json:"message"`
	Fields    Fields    `json:"fields,omitempty"`
	File      string    `json:"file,omitempty"`
	Line      int       `json:"line,omitempty"`
	FuncName  string    `json:"func_name,omitempty"`
	Timestamp time.Time `json:"-"`
}

type ContextKey string

const (
	ContextKeyRequestID    ContextKey = "request_id"
	ContextKeyUserID       ContextKey = "user_id"
	ContextKeySessionID    ContextKey = "session_id"
	ContextKeyTraceID      ContextKey = "trace_id"
	ContextKeySpanID       ContextKey = "span_id"
	ContextKeyClientIP     ContextKey = "client_ip"
	ContextKeyUserAgent    ContextKey = "user_agent"
	ContextKeyEndpoint     ContextKey = "endpoint"
	ContextKeyMethod       ContextKey = "method"
	ContextKeyStatusCode   ContextKey = "status_code"
	ContextKeyDuration     ContextKey = "duration_ms"
	ContextKeyServiceName  ContextKey = "service_name"
	ContextKeyEnvironment  ContextKey = "environment"
	ContextKeyVersion      ContextKey = "version"
)

type LogContext struct {
	mu       sync.RWMutex
	values   map[ContextKey]interface{}
	parent   *LogContext
}

func NewLogContext() *LogContext {
	return &LogContext{
		values: make(map[ContextKey]interface{}),
	}
}

func (c *LogContext) Set(key ContextKey, value interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.values[key] = value
}

func (c *LogContext) Get(key ContextKey) (interface{}, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if v, ok := c.values[key]; ok {
		return v, true
	}
	if c.parent != nil {
		return c.parent.Get(key)
	}
	return nil, false
}

func (c *LogContext) WithValues(fields ...interface{}) *LogContext {
	child := &LogContext{
		values: make(map[ContextKey]interface{}),
		parent: c,
	}
	for i := 0; i < len(fields)-1; i += 2 {
		if key, ok := fields[i].(ContextKey); ok {
			child.values[key] = fields[i+1]
		}
	}
	return child
}

func (c *LogContext) ToFields() Fields {
	c.mu.RLock()
	defer c.mu.RUnlock()
	result := make(Fields)
	for k, v := range c.values {
		result[string(k)] = v
	}
	if c.parent != nil {
		parentFields := c.parent.ToFields()
		for k, v := range parentFields {
			if _, exists := result[k]; !exists {
				result[k] = v
			}
		}
	}
	return result
}

var (
	globalContext atomic.Value
)

func init() {
	globalContext.Store(NewLogContext())
}

func SetGlobalContext(ctx *LogContext) {
	globalContext.Store(ctx)
}

func GetGlobalContext() *LogContext {
	if ctx, ok := globalContext.Load().(*LogContext); ok {
		return ctx
	}
	return NewLogContext()
}

type Logger struct {
	mu           sync.RWMutex
	level        Level
	output       *os.File
	outputPath   string
	isJSON       bool
	isTerminal   bool
	hooks        []Hook
	timeFormat   string
	serviceName  string
	environment  string
	version      string
	enableColors bool
	enableCaller bool
}

type Hook interface {
	Fire(*LogEntry) error
}

type FileHook struct {
	file     *os.File
	mu       sync.Mutex
	maxSize  int64
	maxAge   time.Duration
	rotator  *time.Ticker
	done     chan struct{}
	dir      string
	prefix   string
}

func NewFileHook(path string, maxSize int64, maxAge time.Duration) (*FileHook, error) {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("创建日志目录失败: %w", err)
	}

	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("打开日志文件失败: %w", err)
	}

	return &FileHook{
		file:    file,
		maxSize: maxSize,
		maxAge:  maxAge,
	}, nil
}

func (h *FileHook) Fire(entry *LogEntry) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if err := h.rotateIfNeeded(); err != nil {
		return err
	}

	data, err := json.Marshal(entry)
	if err != nil {
		return err
	}

	_, err = h.file.Write(append(data, '\n'))
	return err
}

func (h *FileHook) rotateIfNeeded() error {
	info, err := h.file.Stat()
	if err != nil {
		return err
	}

	if h.maxSize > 0 && info.Size() >= h.maxSize {
		backupPath := h.file.Name() + ".bak"
		h.file.Close()

		if err := os.Rename(h.file.Name(), backupPath); err != nil {
			return err
		}

		h.file, err = os.OpenFile(h.file.Name(), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return err
		}
	}

	return nil
}

func (h *FileHook) Close() error {
	if h.file != nil {
		return h.file.Close()
	}
	return nil
}

var defaultLogger *Logger

func init() {
	defaultLogger = New()
}

func New() *Logger {
	return &Logger{
		level:         INFO,
		isJSON:        false,
		isTerminal:    true,
		output:        os.Stdout,
		timeFormat:    "2006-01-02T15:04:05.000Z07:00",
		hooks:         make([]Hook, 0),
		serviceName:   "hjtpx",
		environment:   "development",
		version:       "1.0.0",
		enableColors:  true,
		enableCaller:  true,
	}
}

func NewWithConfig(config *LoggerConfig) *Logger {
	logger := New()
	if config != nil {
		if config.Level != "" {
			logger.level = ParseLevel(config.Level)
		}
		if config.Format == "json" {
			logger.isJSON = true
		}
		if config.TimeFormat != "" {
			logger.timeFormat = config.TimeFormat
		}
		if config.ServiceName != "" {
			logger.serviceName = config.ServiceName
		}
		if config.Environment != "" {
			logger.environment = config.Environment
		}
		if config.Version != "" {
			logger.version = config.Version
		}
		logger.enableColors = config.EnableColors
		logger.enableCaller = config.EnableCaller
	}
	return logger
}

type LoggerConfig struct {
	Level        string `json:"level"`
	Format       string `json:"format"`
	TimeFormat   string `json:"time_format"`
	OutputPath   string `json:"output_path"`
	MaxSizeMB    int    `json:"max_size_mb"`
	MaxBackups   int    `json:"max_backups"`
	MaxAgeDays   int    `json:"max_age_days"`
	Compress     bool   `json:"compress"`
	ServiceName  string `json:"service_name"`
	Environment  string `json:"environment"`
	Version      string `json:"version"`
	EnableColors bool   `json:"enable_colors"`
	EnableCaller bool   `json:"enable_caller"`
}

func Default() *Logger {
	return defaultLogger
}

func SetLevel(level Level) {
	defaultLogger.SetLevel(level)
}

func SetLevelFromString(levelStr string) {
	defaultLogger.SetLevel(ParseLevel(levelStr))
}

func SetOutput(path string) error {
	return defaultLogger.SetOutput(path)
}

func SetJSONFormat(isJSON bool) {
	defaultLogger.SetJSONFormat(isJSON)
}

func SetTimeFormat(format string) {
	defaultLogger.SetTimeFormat(format)
}

func SetServiceName(name string) {
	defaultLogger.SetServiceName(name)
}

func SetEnvironment(env string) {
	defaultLogger.SetEnvironment(env)
}

func SetVersion(ver string) {
	defaultLogger.SetVersion(ver)
}

func AddHook(hook Hook) {
	defaultLogger.AddHook(hook)
}

func Debug(message string, fields ...Fields) {
	defaultLogger.Log(DEBUG, message, fields...)
}

func Info(message string, fields ...Fields) {
	defaultLogger.Log(INFO, message, fields...)
}

func Warn(message string, fields ...Fields) {
	defaultLogger.Log(WARN, message, fields...)
}

func Error(message string, fields ...Fields) {
	defaultLogger.Log(ERROR, message, fields...)
}

func Fatal(message string, fields ...Fields) {
	defaultLogger.Log(FATAL, message, fields...)
	os.Exit(1)
}

func Debugf(format string, args ...interface{}) {
	defaultLogger.Logf(DEBUG, format, args...)
}

func Infof(format string, args ...interface{}) {
	defaultLogger.Logf(INFO, format, args...)
}

func Warnf(format string, args ...interface{}) {
	defaultLogger.Logf(WARN, format, args...)
}

func Errorf(format string, args ...interface{}) {
	defaultLogger.Logf(ERROR, format, args...)
}

func Fatalf(format string, args ...interface{}) {
	defaultLogger.Logf(FATAL, format, args...)
	os.Exit(1)
}

func DebugContext(ctx *LogContext, message string, fields ...Fields) {
	defaultLogger.LogContext(ctx, DEBUG, message, fields...)
}

func InfoContext(ctx *LogContext, message string, fields ...Fields) {
	defaultLogger.LogContext(ctx, INFO, message, fields...)
}

func WarnContext(ctx *LogContext, message string, fields ...Fields) {
	defaultLogger.LogContext(ctx, WARN, message, fields...)
}

func ErrorContext(ctx *LogContext, message string, fields ...Fields) {
	defaultLogger.LogContext(ctx, ERROR, message, fields...)
}

func (l *Logger) SetLevel(level Level) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.level = level
}

func (l *Logger) SetOutput(path string) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("创建日志目录失败: %w", err)
	}

	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("打开日志文件失败: %w", err)
	}

	l.output = file
	l.outputPath = path
	l.isTerminal = false

	return nil
}

func (l *Logger) SetJSONFormat(isJSON bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.isJSON = isJSON
}

func (l *Logger) SetTimeFormat(format string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.timeFormat = format
}

func (l *Logger) SetServiceName(name string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.serviceName = name
}

func (l *Logger) SetEnvironment(env string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.environment = env
}

func (l *Logger) SetVersion(ver string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.version = ver
}

func (l *Logger) AddHook(hook Hook) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.hooks = append(l.hooks, hook)
}

func (l *Logger) Log(level Level, message string, fields ...Fields) {
	l.LogContext(nil, level, message, fields...)
}

func (l *Logger) LogContext(ctx *LogContext, level Level, message string, fields ...Fields) {
	if level < l.level {
		return
	}

	entry := l.createEntry(level, message, fields...)
	if ctx != nil {
		entry.Fields = l.mergeFields(entry.Fields, ctx.ToFields())
	}
	entry.Fields = l.mergeFields(entry.Fields, GetGlobalContext().ToFields())
	l.outputLog(entry)

	l.mu.RLock()
	hooks := l.hooks
	l.mu.RUnlock()
	for _, hook := range hooks {
		if err := hook.Fire(entry); err != nil {
			fmt.Fprintf(os.Stderr, "钩子执行失败: %v\n", err)
		}
	}
}

func (l *Logger) Logf(level Level, format string, args ...interface{}) {
	l.Log(level, fmt.Sprintf(format, args...))
}

func (l *Logger) LogWithRequest(level Level, requestID, userID, sessionID, clientIP, userAgent, method, path string, statusCode int, durationMs int64, message string, fields ...Fields) {
	logFields := make(Fields)
	logFields[string(ContextKeyRequestID)] = requestID
	logFields[string(ContextKeyUserID)] = userID
	logFields[string(ContextKeySessionID)] = sessionID
	logFields[string(ContextKeyClientIP)] = clientIP
	logFields[string(ContextKeyUserAgent)] = userAgent
	logFields[string(ContextKeyMethod)] = method
	logFields[string(ContextKeyEndpoint)] = path
	logFields[string(ContextKeyStatusCode)] = statusCode
	logFields[string(ContextKeyDuration)] = durationMs

	for k, v := range fields {
		logFields[string(k)] = v
	}

	l.Log(level, message, logFields)
}

func (l *Logger) createEntry(level Level, message string, fields ...Fields) *LogEntry {
	entry := &LogEntry{
		Time:      time.Now().Format(l.timeFormat),
		Level:     level.String(),
		Message:   message,
		Timestamp: time.Now(),
	}

	if len(fields) > 0 && fields[0] != nil {
		entry.Fields = fields[0]
	} else {
		entry.Fields = make(Fields)
	}

	l.mu.RLock()
	enableCaller := l.enableCaller
	l.mu.RUnlock()

	if enableCaller {
		if pc, file, line, ok := runtime.Caller(2); ok {
			entry.File = filepath.Base(file)
			entry.Line = line
			if fn := runtime.FuncForPC(pc); fn != nil {
				entry.FuncName = filepath.Base(fn.Name())
			}
		}
	}

	return entry
}

func (l *Logger) mergeFields(base, override Fields) Fields {
	result := make(Fields)
	for k, v := range base {
		result[k] = v
	}
	for k, v := range override {
		result[k] = v
	}
	return result
}

func (l *Logger) outputLog(entry *LogEntry) {
	l.mu.Lock()
	defer l.mu.Unlock()

	var output string
	if l.isJSON {
		data, err := json.Marshal(entry)
		if err != nil {
			fmt.Fprintf(os.Stderr, "序列化日志失败: %v\n", err)
			return
		}
		output = string(data)
	} else {
		output = l.formatText(entry)
	}

	if l.isTerminal && l.enableColors {
		coloredOutput := l.colorize(entry.Level, output)
		fmt.Fprintln(l.output, coloredOutput)
	} else {
		fmt.Fprintln(l.output, output)
	}
}

func (l *Logger) formatText(entry *LogEntry) string {
	base := fmt.Sprintf("[%s] %s: %s", entry.Time, entry.Level, entry.Message)

	if entry.File != "" {
		base += fmt.Sprintf(" (%s:%d)", entry.File, entry.Line)
	}

	if len(entry.Fields) > 0 {
		base += " | "
		first := true
		for k, v := range entry.Fields {
			if !first {
				base += " "
			}
			base += fmt.Sprintf("%s=%v", k, v)
			first = false
		}
	}

	return base
}

func (l *Logger) colorize(level string, message string) string {
	l.mu.RLock()
	isTerminal := l.isTerminal
	l.mu.RUnlock()

	if !isTerminal {
		return message
	}

	colorCode := ""
	switch level {
	case "DEBUG":
		colorCode = "\033[36m"
	case "INFO":
		colorCode = "\033[32m"
	case "WARN":
		colorCode = "\033[33m"
	case "ERROR":
		colorCode = "\033[31m"
	case "FATAL":
		colorCode = "\033[35m"
	default:
		colorCode = "\033[0m"
	}

	return colorCode + message + "\033[0m"
}

func (l *Logger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.output != os.Stdout && l.output != os.Stderr {
		return l.output.Close()
	}
	return nil
}

type ConsoleHook struct{}

func NewConsoleHook() *ConsoleHook {
	return &ConsoleHook{}
}

func (h *ConsoleHook) Fire(entry *LogEntry) error {
	return nil
}

type RotatingFileHook struct {
	hook       *FileHook
	dir        string
	prefix     string
	maxSize    int64
	maxAge     time.Duration
	rotator    *time.Ticker
	done       chan struct{}
	maxBackups int
	compress   bool
}

func NewRotatingFileHook(dir, prefix string, maxSize int64, maxAge time.Duration) (*RotatingFileHook, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("创建日志目录失败: %w", err)
	}

	filename := fmt.Sprintf("%s%s.log", prefix, time.Now().Format("20060102150405"))
	path := filepath.Join(dir, filename)

	hook, err := NewFileHook(path, maxSize, maxAge)
	if err != nil {
		return nil, err
	}

	r := &RotatingFileHook{
		hook:    hook,
		dir:     dir,
		prefix:  prefix,
		maxSize: maxSize,
		maxAge:  maxAge,
		rotator: time.NewTicker(1 * time.Hour),
		done:    make(chan struct{}),
	}

	go r.rotatePeriodically()

	return r, nil
}

func (h *RotatingFileHook) Fire(entry *LogEntry) error {
	return h.hook.Fire(entry)
}

func (h *RotatingFileHook) rotatePeriodically() {
	for {
		select {
		case <-h.rotator.C:
			h.rotate()
		case <-h.done:
			h.rotator.Stop()
			return
		}
	}
}

func (h *RotatingFileHook) rotate() {
	filename := fmt.Sprintf("%s%s.log", h.prefix, time.Now().Format("20060102150405"))
	path := filepath.Join(h.dir, filename)

	newHook, err := NewFileHook(path, h.maxSize, h.maxAge)
	if err != nil {
		fmt.Fprintf(os.Stderr, "轮转日志文件失败: %v\n", err)
		return
	}

	oldHook := h.hook
	h.hook = newHook

	if oldHook != nil {
		oldHook.Close()
	}

	h.cleanOldLogs()
}

func (h *RotatingFileHook) cleanOldLogs() {
	if h.maxAge <= 0 && h.maxBackups <= 0 {
		return
	}

	pattern := filepath.Join(h.dir, h.prefix+"*.log")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return
	}

	cutoff := time.Now().Add(-h.maxAge)
	for _, match := range matches {
		info, err := os.Stat(match)
		if err != nil {
			continue
		}

		if h.maxAge > 0 && info.ModTime().Before(cutoff) {
			os.Remove(match)
		}
	}

	if h.maxBackups > 0 {
		if len(matches) > h.maxBackups {
			sortByTime := make([]string, 0)
			for _, m := range matches {
				if !strings.HasSuffix(m, ".gz") {
					sortByTime = append(sortByTime, m)
				}
			}
			sort.Strings(sortByTime)
			if len(sortByTime) > h.maxBackups {
				for i := 0; i < len(sortByTime)-h.maxBackups; i++ {
					os.Remove(sortByTime[i])
				}
			}
		}
	}
}

func (h *RotatingFileHook) Close() error {
	close(h.done)
	return h.hook.Close()
}
