package utils

import (
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"
)

type LogLevel int

const (
	DEBUG LogLevel = iota
	INFO
	WARN
	ERROR
)

var (
	levelNames = map[LogLevel]string{
		DEBUG: "DEBUG",
		INFO:  "INFO",
		WARN:  "WARN",
		ERROR: "ERROR",
	}
)

type Logger struct {
	mu      sync.Mutex
	level   LogLevel
	output  io.Writer
	logger  *log.Logger
}

var defaultLogger *Logger
var once sync.Once

func InitLogger(levelStr, outputPath string) error {
	var lvl LogLevel
	switch levelStr {
	case "debug":
		lvl = DEBUG
	case "info":
		lvl = INFO
	case "warn":
		lvl = WARN
	case "error":
		lvl = ERROR
	default:
		lvl = INFO
	}

	var output io.Writer
	if outputPath == "stdout" || outputPath == "" {
		output = os.Stdout
	} else {
		dir := filepath.Dir(outputPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
		file, err := os.OpenFile(outputPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			return err
		}
		output = file
	}

	once.Do(func() {
		defaultLogger = &Logger{
			level:  lvl,
			output: output,
			logger: log.New(output, "", log.LstdFlags|log.Lshortfile),
		}
	})

	return nil
}

func GetLogger() *Logger {
	if defaultLogger == nil {
		defaultLogger = &Logger{
			level:  INFO,
			output: os.Stdout,
			logger: log.New(os.Stdout, "", log.LstdFlags),
		}
	}
	return defaultLogger
}

func (l *Logger) shouldLog(level LogLevel) bool {
	return level >= l.level
}

func (l *Logger) log(level LogLevel, format string, args ...interface{}) {
	if !l.shouldLog(level) {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()
	l.logger.Printf("["+levelNames[level]+"] "+format, args...)
}

func Debug(format string, args ...interface{}) {
	GetLogger().log(DEBUG, format, args...)
}

func Info(format string, args ...interface{}) {
	GetLogger().log(INFO, format, args...)
}

func Warn(format string, args ...interface{}) {
	GetLogger().log(WARN, format, args...)
}

func Error(format string, args ...interface{}) {
	GetLogger().log(ERROR, format, args...)
}

func SetLevel(levelStr string) {
	var lvl LogLevel
	switch levelStr {
	case "debug":
		lvl = DEBUG
	case "info":
		lvl = INFO
	case "warn":
		lvl = WARN
	case "error":
		lvl = ERROR
	default:
		lvl = INFO
	}
	GetLogger().level = lvl
}
