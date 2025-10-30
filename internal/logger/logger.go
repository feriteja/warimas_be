package logger

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"sync"
	"time"
)

type LogLevel string

const (
	LevelDebug LogLevel = "DEBUG"
	LevelInfo  LogLevel = "INFO"
	LevelWarn  LogLevel = "WARN"
	LevelError LogLevel = "ERROR"
)

type Logger struct {
	mu     sync.Mutex
	level  LogLevel
	writer io.Writer
}

type logEntry struct {
	Time    string      `json:"time"`
	Level   LogLevel    `json:"level"`
	Message string      `json:"message"`
	Fields  interface{} `json:"fields,omitempty"`
}

// global singleton
var std = New(LevelInfo, os.Stdout)

// New creates a new logger
func New(level LogLevel, writer io.Writer) *Logger {
	return &Logger{level: level, writer: writer}
}

// SetLevel allows changing log level dynamically
func SetLevel(level LogLevel) {
	std.level = level
}

// internal helper
func (l *Logger) log(level LogLevel, msg string, fields interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Only log if level is >= configured level
	if !shouldLog(l.level, level) {
		return
	}

	entry := logEntry{
		Time:    time.Now().Format(time.RFC3339),
		Level:   level,
		Message: msg,
		Fields:  fields,
	}

	data, _ := json.Marshal(entry)
	fmt.Fprintln(l.writer, string(data))
}

// level filtering
func shouldLog(current, incoming LogLevel) bool {
	order := map[LogLevel]int{
		LevelDebug: 1,
		LevelInfo:  2,
		LevelWarn:  3,
		LevelError: 4,
	}
	return order[incoming] >= order[current]
}

// Public helpers
func Debug(msg string, fields ...interface{}) { std.log(LevelDebug, msg, join(fields...)) }
func Info(msg string, fields ...interface{})  { std.log(LevelInfo, msg, join(fields...)) }
func Warn(msg string, fields ...interface{})  { std.log(LevelWarn, msg, join(fields...)) }
func Error(msg string, fields ...interface{}) { std.log(LevelError, msg, join(fields...)) }

// join merges fields (if you want structured context)
func join(fields ...interface{}) interface{} {
	if len(fields) == 1 {
		return fields[0]
	}
	return fields
}

// Optional: redirect default `log` package output
func RedirectStdLog() {
	log.SetOutput(std.writer)
	log.SetFlags(0)
	log.SetPrefix("")
}
