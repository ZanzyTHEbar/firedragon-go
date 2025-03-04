package internal

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

var (
	globalLogger *Logger
	once         sync.Once
)

const (
	LogLevelDebug LogLevel = iota
	LogLevelInfo
	LogLevelWarn
	LogLevelError
	LogLevelFatal
)

type Component string
type LogLevel int

const (
	ComponentNATS        Component = "NATS"
	ComponentStorage     Component = "Storage"
	ComponentConfig      Component = "Config"
	ComponentTransaction Component = "Trans"
	ComponentService     Component = "Service"
	ComponentGeneral     Component = "General"
)

type Logger struct {
	mu                sync.RWMutex
	logger            *log.Logger
	file              *os.File
	level             LogLevel
	enabledComponents map[Component]bool
}

func InitGlobalLogger(logDir string, level LogLevel, components []Component) error {
	var err error
	once.Do(func() {
		globalLogger, err = NewLogger(logDir, level, components)
	})
	return err
}

func GetLogger() *Logger {
	if globalLogger == nil {
		globalLogger = &Logger{
			logger: log.New(os.Stderr, "", log.LstdFlags),
			level:  LogLevelInfo,
			enabledComponents: map[Component]bool{
				ComponentGeneral:     true,
				ComponentService:     true,
				ComponentTransaction: true,
				ComponentConfig:      true,
				ComponentStorage:     true,
				ComponentNATS:        true,
			},
		}
	}
	return globalLogger
}

func NewLogger(logDir string, level LogLevel, components []Component) (*Logger, error) {
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	timestamp := time.Now().Format("2006-01-02_15-04-05")
	logPath := filepath.Join(logDir, fmt.Sprintf("app_%s.log", timestamp))
	file, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}

	multiWriter := io.MultiWriter(file, os.Stderr)
	logger := log.New(multiWriter, "", log.LstdFlags|log.Lmicroseconds)

	enabledComponents := make(map[Component]bool)
	for _, component := range components {
		enabledComponents[component] = true
	}

	return &Logger{
		logger:            logger,
		file:              file,
		level:             level,
		enabledComponents: enabledComponents,
	}, nil
}

func (l *Logger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.file != nil {
		return l.file.Close()
	}
	return nil
}

func (l *Logger) SetLevel(level LogLevel) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.level = level
}

func (l *Logger) EnableComponent(component Component) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.enabledComponents[component] = true
}

func (l *Logger) DisableComponent(component Component) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.enabledComponents[component] = false
}

func (l *Logger) IsComponentEnabled(component Component) bool {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.enabledComponents[component]
}

func (l *Logger) log(level LogLevel, component Component, format string, args ...interface{}) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	// Check if we should log this message
	if level < l.level || !l.enabledComponents[component] {
		return
	}

	// Get level name
	levelName := "UNKNOWN"
	switch level {
	case LogLevelDebug:
		levelName = "DEBUG"
	case LogLevelInfo:
		levelName = "INFO"
	case LogLevelWarn:
		levelName = "WARN"
	case LogLevelError:
		levelName = "ERROR"
	case LogLevelFatal:
		levelName = "FATAL"
	}

	// Format message
	message := fmt.Sprintf(format, args...)
	l.logger.Printf("[%s][%s] %s", levelName, component, message)

	// Exit on fatal
	if level == LogLevelFatal {
		os.Exit(1)
	}
}

func (l *Logger) Debug(component Component, format string, args ...interface{}) {
	l.log(LogLevelDebug, component, format, args...)
}

func (l *Logger) Info(component Component, format string, args ...interface{}) {
	l.log(LogLevelInfo, component, format, args...)
}

func (l *Logger) Warn(component Component, format string, args ...interface{}) {
	l.log(LogLevelWarn, component, format, args...)
}

func (l *Logger) Error(component Component, format string, args ...interface{}) {
	l.log(LogLevelError, component, format, args...)
}

func (l *Logger) Fatal(component Component, format string, args ...interface{}) {
	l.log(LogLevelFatal, component, format, args...)
}
