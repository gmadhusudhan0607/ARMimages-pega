/*
 * Copyright (c) 2024 Pegasystems Inc.
 * All rights reserved.
 */

package log

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/helpers"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/helpers/contexthelper"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	DebugLevel = zapcore.DebugLevel
	InfoLevel  = zapcore.InfoLevel
	WarnLevel  = zapcore.WarnLevel
	ErrorLevel = zapcore.ErrorLevel
	PanicLevel = zapcore.PanicLevel
	FatalLevel = zapcore.FatalLevel
)

var (
	once   sync.Once
	logger *zap.Logger

	// Configuration constants for tuning logger performance
	loggerBufferSize    = 256 * 1024 // 256KB buffer
	loggerFlushInterval = 1 * time.Second

	// LoggerPool caches named loggers to avoid recreation
	loggerPool = sync.Map{}
)

// parseLogLevel converts a string log level to zapcore.Level
func parseLogLevel(levelStr string) zapcore.Level {
	switch strings.ToUpper(levelStr) {
	case "DEBUG":
		return DebugLevel
	case "INFO":
		return InfoLevel
	case "WARN", "WARNING":
		return WarnLevel
	case "ERROR":
		return ErrorLevel
	case "PANIC":
		return PanicLevel
	case "FATAL":
		return FatalLevel
	default:
		return InfoLevel
	}
}

// getHighPerformanceConfig creates an optimized zap logger configuration
func getHighPerformanceConfig(level zapcore.Level) zap.Config {
	// Start with production defaults
	cfg := zap.NewProductionConfig()

	// Set the log level
	cfg.Level = zap.NewAtomicLevelAt(level)

	// Configure sampling to reduce volume in high-throughput situations
	// First message in a 5-second window always gets through
	// After that, only log 1 message per 100 similar messages
	cfg.Sampling = &zap.SamplingConfig{
		Initial:    100,
		Thereafter: 100,
		Hook: func(entry zapcore.Entry, decision zapcore.SamplingDecision) {
			// Custom logic can be added here if needed, but dropped count is not available in this zap version
			if entry.Level >= ErrorLevel {
				if _, err := fmt.Fprintf(os.Stderr, "Sampled error log: %s\n", entry.Message); err != nil {
					fmt.Fprintf(os.Stderr, "Failed to write sampled error log: %v\n", err)
				}
			}
		},
	}

	// Optimize encoding performance
	cfg.EncoderConfig.TimeKey = "ts"
	cfg.EncoderConfig.EncodeTime = zapcore.EpochTimeEncoder

	return cfg
}

// Initialize the global logger with high-performance configuration
func getLogger() *zap.Logger {
	once.Do(func() {
		logLevelStr := helpers.GetEnvOrDefault("LOG_LEVEL", "INFO")
		logLevel := parseLogLevel(logLevelStr)

		// Get optimized configuration
		cfg := getHighPerformanceConfig(logLevel)

		// Create a core with buffered writer
		core, err := createBufferedCore(cfg)
		if err != nil {
			// Fallback to standard configuration if buffered setup fails
			zapLogger, _ := cfg.Build()
			logger = zapLogger
			return
		}

		// Create the buffered logger
		logger = zap.New(core, zap.AddCaller())
	})

	return logger
}

// createBufferedCore sets up a buffered logging core that flushes periodically
func createBufferedCore(cfg zap.Config) (zapcore.Core, error) {
	// Create the standard encoder based on config
	encoder := zapcore.NewJSONEncoder(cfg.EncoderConfig)

	// Create the writer with buffer
	writer, err := getBufferedWriter()
	if err != nil {
		return nil, err
	}

	// Create the core
	core := zapcore.NewCore(
		encoder,
		writer,
		cfg.Level,
	)

	// Setup periodic flushing
	go func() {
		for {
			time.Sleep(loggerFlushInterval)
			if err := writer.Sync(); err != nil {
				// Only log error if not os.Stdout
				if err.Error() != "sync /dev/stdout: invalid argument" {
					fmt.Fprintf(os.Stderr, "Failed to sync log writer: %v\n", err)
				}
			}
		}
	}()

	return core, nil
}

// getBufferedWriter creates a buffered zapcore.WriteSyncer
func getBufferedWriter() (zapcore.WriteSyncer, error) {
	// Default to stdout for log output
	output := zapcore.Lock(os.Stdout)

	// Use bufio for buffered writing if zapcore.NewBufferedWriteSyncer is unavailable
	// This is a fallback for zap versions without NewBufferedWriteSyncer
	// Buffer size is configurable
	bufWriter := bufio.NewWriterSize(output, loggerBufferSize)
	syncer := zapcore.AddSync(&bufferedWriteSyncer{
		buf:      bufWriter,
		parent:   output,
		mu:       sync.Mutex{},
		isStdout: true, // always true for os.Stdout
	})
	return syncer, nil
}

// bufferedWriteSyncer wraps bufio.Writer and underlying zapcore.WriteSyncer
// to provide buffered writes and sync
type bufferedWriteSyncer struct {
	buf      *bufio.Writer
	parent   zapcore.WriteSyncer
	mu       sync.Mutex // protects buf
	isStdout bool       // true if parent is os.Stdout
}

func (b *bufferedWriteSyncer) Write(p []byte) (n int, err error) {
	b.mu.Lock()
	n, err = b.buf.Write(p)
	b.mu.Unlock()
	return
}

func (b *bufferedWriteSyncer) Sync() error {
	b.mu.Lock()
	err := b.buf.Flush()
	b.mu.Unlock()
	if err != nil {
		return err
	}
	if b.isStdout {
		return nil // skip parent.Sync() for os.Stdout
	}
	return b.parent.Sync()
}

// GetNamedLogger returns a named logger, reusing instances from the pool when possible
func GetNamedLogger(name string) *zap.Logger {
	// Check if we already have this named logger
	if cachedLogger, found := loggerPool.Load(name); found {
		return cachedLogger.(*zap.Logger)
	}

	// Create a new named logger
	namedLogger := getLogger().Named(name)

	// Store in the pool for future reuse
	loggerPool.Store(name, namedLogger)

	return namedLogger
}

func GetLoggerFromContext(ctx context.Context) *zap.Logger {
	namedLogger := getLogger()

	if isoID, ok := ctx.Value(contexthelper.IsolationIDKey).(string); ok {
		namedLogger = namedLogger.With(zap.String("isolation_id", isoID))
	}
	if colID, ok := ctx.Value(contexthelper.CollectionIDKey).(string); ok {
		namedLogger = namedLogger.With(zap.String("collection_id", colID))
	}
	if docID, ok := ctx.Value(contexthelper.DocumentIDKey).(string); ok {
		namedLogger = namedLogger.With(zap.String("document_id", docID))
	}
	if traceID, ok := ctx.Value(contexthelper.TraceIDKey).(string); ok {
		namedLogger = namedLogger.With(zap.String("trace_id", traceID))
	}
	if spanID, ok := ctx.Value(contexthelper.SpanIDKey).(string); ok {
		namedLogger = namedLogger.With(zap.String("span_id", spanID))
	}
	if reqID, ok := ctx.Value(contexthelper.RequestIDKey).(string); ok {
		namedLogger = namedLogger.With(zap.String("request_id", reqID))
	}

	return namedLogger
}

// GetNamedLoggerWithParams returns a logger with additional context fields
func GetNamedLoggerWithParams(name string, params ...zap.Field) *zap.Logger {
	baseLogger := GetNamedLogger(name)
	return baseLogger.With(params...)
}

// FlushLogs forces any buffered logs to be written
func FlushLogs() {
	if logger != nil {
		_ = logger.Sync()
	}
}
