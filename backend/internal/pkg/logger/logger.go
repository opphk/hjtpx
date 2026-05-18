package logger

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
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

type Logger struct {
	mu         sync.Mutex
	level      Level
	output     *os.File
	outputPath string
	isJSON     bool
	isTerminal bool
	hooks      []Hook
	timeFormat string
}

type Hook interface {
	Fire(*LogEntry) error
}

type FileHook struct {
	file    *os.File
	mu      sync.Mutex
	maxSize int64
	maxAge  time.Duration
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
		level:      INFO,
		isJSON:     false,
		isTerminal: true,
		output:     os.Stdout,
		timeFormat: "2006-01-02 15:04:05",
		hooks:      make([]Hook, 0),
	}
}

func Default() *Logger {
	return defaultLogger
}

func SetLevel(level Level) {
	defaultLogger.SetLevel(level)
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

func (l *Logger) AddHook(hook Hook) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.hooks = append(l.hooks, hook)
}

func (l *Logger) Log(level Level, message string, fields ...Fields) {
	if level < l.level {
		return
	}

	entry := l.createEntry(level, message, fields...)
	l.outputLog(entry)

	for _, hook := range l.hooks {
		if err := hook.Fire(entry); err != nil {
			fmt.Fprintf(os.Stderr, "钩子执行失败: %v\n", err)
		}
	}
}

func (l *Logger) Logf(level Level, format string, args ...interface{}) {
	l.Log(level, fmt.Sprintf(format, args...))
}

func (l *Logger) createEntry(level Level, message string, fields ...Fields) *LogEntry {
	entry := &LogEntry{
		Time:      time.Now().Format(l.timeFormat),
		Level:     level.String(),
		Message:   message,
		Timestamp: time.Now(),
	}

	if len(fields) > 0 {
		entry.Fields = fields[0]
	}

	if pc, file, line, ok := runtime.Caller(2); ok {
		entry.File = filepath.Base(file)
		entry.Line = line
		if fn := runtime.FuncForPC(pc); fn != nil {
			entry.FuncName = filepath.Base(fn.Name())
		}
	}

	return entry
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

	if l.isTerminal {
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
		for k, v := range entry.Fields {
			base += fmt.Sprintf("%s=%v ", k, v)
		}
	}

	return base
}

func (l *Logger) colorize(level string, message string) string {
	if !l.isTerminal {
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
	hook    *FileHook
	dir     string
	prefix  string
	maxSize int64
	maxAge  time.Duration
	rotator *time.Ticker
	done    chan struct{}
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
	if h.maxAge <= 0 {
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

		if info.ModTime().Before(cutoff) {
			os.Remove(match)
		}
	}
}

func (h *RotatingFileHook) Close() error {
	close(h.done)
	return h.hook.Close()
}
