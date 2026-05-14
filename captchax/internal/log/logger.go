package log

import (
	"encoding/json"
	"fmt"
	"os"
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

func (l Level) String() string {
	switch l {
	case DEBUG:
		return "debug"
	case INFO:
		return "info"
	case WARN:
		return "warn"
	case ERROR:
		return "error"
	case FATAL:
		return "fatal"
	default:
		return "unknown"
	}
}

func ParseLevel(s string) Level {
	switch s {
	case "debug":
		return DEBUG
	case "info":
		return INFO
	case "warn", "warning":
		return WARN
	case "error":
		return ERROR
	case "fatal":
		return FATAL
	default:
		return INFO
	}
}

type LogEntry struct {
	Time    string                 `json:"time"`
	Level   string                 `json:"level"`
	Message string                 `json:"message"`
	Fields  map[string]interface{} `json:"fields,omitempty"`
}

type Logger struct {
	mu     sync.Mutex
	level  Level
	format string
	output *os.File
}

var (
	defaultLogger *Logger
	once          sync.Once
)

func Init(level, format, output string) {
	once.Do(func() {
		defaultLogger = &Logger{
			level:  ParseLevel(level),
			format: format,
		}
		if output == "stderr" {
			defaultLogger.output = os.Stderr
		} else {
			defaultLogger.output = os.Stdout
		}
	})
}

func Default() *Logger {
	if defaultLogger == nil {
		Init("info", "json", "stdout")
	}
	return defaultLogger
}

func (l *Logger) log(level Level, msg string, fields map[string]interface{}) {
	if level < l.level {
		return
	}

	entry := LogEntry{
		Time:    time.Now().UTC().Format(time.RFC3339),
		Level:   level.String(),
		Message: msg,
		Fields:  fields,
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	if l.format == "json" {
		data, _ := json.Marshal(entry)
		l.output.Write(append(data, '\n'))
	} else {
		text := fmt.Sprintf("%s [%s] %s", entry.Time, entry.Level, entry.Message)
		if len(entry.Fields) > 0 {
			text += " | "
			for k, v := range entry.Fields {
				text += fmt.Sprintf("%s=%v ", k, v)
			}
		}
		l.output.WriteString(text + "\n")
	}
}

func (l *Logger) Debug(msg string, fields ...map[string]interface{}) {
	f := mergeFields(fields)
	l.log(DEBUG, msg, f)
}

func (l *Logger) Info(msg string, fields ...map[string]interface{}) {
	f := mergeFields(fields)
	l.log(INFO, msg, f)
}

func (l *Logger) Warn(msg string, fields ...map[string]interface{}) {
	f := mergeFields(fields)
	l.log(WARN, msg, f)
}

func (l *Logger) Error(msg string, fields ...map[string]interface{}) {
	f := mergeFields(fields)
	l.log(ERROR, msg, f)
}

func (l *Logger) Fatal(msg string, fields ...map[string]interface{}) {
	f := mergeFields(fields)
	l.log(FATAL, msg, f)
	os.Exit(1)
}

func mergeFields(fields []map[string]interface{}) map[string]interface{} {
	if len(fields) == 0 {
		return nil
	}
	result := make(map[string]interface{})
	for _, f := range fields {
		for k, v := range f {
			result[k] = v
		}
	}
	return result
}

func Debug(msg string, fields ...map[string]interface{}) {
	Default().Debug(msg, fields...)
}

func Info(msg string, fields ...map[string]interface{}) {
	Default().Info(msg, fields...)
}

func Warn(msg string, fields ...map[string]interface{}) {
	Default().Warn(msg, fields...)
}

func Error(msg string, fields ...map[string]interface{}) {
	Default().Error(msg, fields...)
}

func Fatal(msg string, fields ...map[string]interface{}) {
	Default().Fatal(msg, fields...)
}
