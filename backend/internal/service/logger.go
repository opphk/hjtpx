package service

import (
	"log"
	"os"
)

type Logger struct {
	*log.Logger
}

var defaultLogger = log.New(os.Stdout, "[SERVICE] ", log.LstdFlags|log.Lshortfile)

func GetLogger() *Logger {
	return &Logger{defaultLogger}
}

func (l *Logger) Info(msg string, keyvals ...interface{}) {
	l.Printf("INFO: %s %v", msg, keyvals)
}

func (l *Logger) Error(msg string, keyvals ...interface{}) {
	l.Printf("ERROR: %s %v", msg, keyvals)
}

func (l *Logger) Debug(msg string, keyvals ...interface{}) {
	l.Printf("DEBUG: %s %v", msg, keyvals)
}

func (l *Logger) Warn(msg string, keyvals ...interface{}) {
	l.Printf("WARN: %s %v", msg, keyvals)
}
