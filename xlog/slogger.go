package xlog

import (
	"context"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/omeyang/gokit/metrics/sample"

	"go.opentelemetry.io/otel/trace"
)

// internalErrorLogger 用于记录内部错误
var internalErrorLogger = log.New(os.Stderr, "INTERNAL_ERROR: ", log.LstdFlags)

// SlogLogger 实现了 HighPerformanceLogger 接口，基于 slog
type SlogLogger struct {
	handler          atomic.Value     // 存储 slog.Handler
	level            atomic.Value     // 存储 LogLevel
	config           LogConfig        // 日志配置
	buffer           chan slog.Record // 存储日志记录的缓冲通道 实现异步处理日志
	done             chan struct{}    // 优雅地关闭日志处理goroutine
	sampler          sample.Sampler   // 采样器
	wg               sync.WaitGroup
	contextExtractor ContextExtractor // context中的提取字段
}

// validateConfig 验证日志配置
func validateConfig(config *LogConfig) error {
	if config.Writer == nil {
		return errors.New("log writer is not set")
	}
	if config.AsyncBufferSize <= 0 {
		return errors.New("async buffer size must be greater than 0")
	}
	if config.FlushInterval <= 0 {
		return errors.New("flush interval must be greater than 0")
	}
	if config.Sampling.Rate < 0 || config.Sampling.Rate > 1 {
		return errors.New("sampling rate must be between 0 and 1")
	}
	return nil
}

// NewSlogLogger 创建一个新的 SlogLogger 实例
func NewSlogLogger(config LogConfig, additionalContextKeys ...string) (*SlogLogger, error) {
	if err := validateConfig(&config); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}
	// 创建 slog 处理器
	handler := createHandler(config)

	var sampler sample.Sampler
	switch config.Sampling.Type {
	case sample.RateSamplerType:
		sampler = sample.NewRateSampler(config.Sampling.Rate)
	case sample.JitterSamplerType:
		sampler = sample.NewJitterSampler(config.Sampling.Rate, config.Sampling.Jitter)
	default:
		sampler = sample.NewRateSampler(1) // 默认不采样
	}

	logger := &SlogLogger{
		config:           config,
		buffer:           make(chan slog.Record, config.AsyncBufferSize),
		done:             make(chan struct{}),
		sampler:          sampler,
		contextExtractor: NewDefaultContextExtractor(additionalContextKeys...),
	}
	logger.handler.Store(handler)
	logger.level.Store(config.Level)

	// 启动异步处理 goroutine
	logger.wg.Add(1)
	go logger.processLogs()

	return logger, nil
}

// createHandler 根据配置创建 slog.Handler
func createHandler(config LogConfig) slog.Handler {
	opts := &slog.HandlerOptions{
		Level:     slog.Level(levelOrder[config.Level]),
		AddSource: config.EnableCaller,
	}

	if config.Encoder == TextEncoder {
		return slog.NewTextHandler(config.Writer, opts)
	} else if config.Encoder == JSONEncoder {
		return slog.NewJSONHandler(config.Writer, opts)
	} else if config.Encoder == ProtoEncoder {
		return nil
	}
	return nil
}

// processLogs 异步处理日志记录
func (l *SlogLogger) processLogs() {
	defer l.wg.Done()
	ticker := time.NewTicker(l.config.FlushInterval)
	defer ticker.Stop()

	var records []slog.Record
	for {
		select {
		case record := <-l.buffer:
			if record.Message == "FLUSH_SIGNAL" {
				// 刷新信号
				l.writeBatch(records)
				records = records[:0]
				l.buffer <- slog.Record{} // 发送完成信号
				continue
			}
			records = append(records, record)
			if len(records) >= l.config.AsyncBufferSize {
				l.writeBatch(records)
				records = records[:0]
			}
		case <-ticker.C:
			if len(records) > 0 {
				l.writeBatch(records)
				records = records[:0]
			}
		case <-l.done:
			if len(records) > 0 {
				l.writeBatch(records)
			}
			return
		}
	}
}

// writeBatch 批量写入日志记录
func (l *SlogLogger) writeBatch(records []slog.Record) {
	handler := l.handler.Load().(slog.Handler)
	for _, record := range records {
		if err := handler.Handle(context.Background(), record); err != nil {
			l.logInternalError(fmt.Sprintf("Failed to handle log record: %v", err))
		}
	}
}

// SetLevel 设置日志级别
func (l *SlogLogger) SetLevel(level LogLevel) error {
	l.level.Store(level)
	return nil
}

// GetLevel 获取当前日志级别
func (l *SlogLogger) GetLevel() LogLevel {
	return l.level.Load().(LogLevel)
}

// log 通用日志记录方法
func (l *SlogLogger) log(ctx context.Context, level LogLevel, msg string, fields ...Field) {
	currentLevel := l.GetLevel()
	if currentLevel.IsHighThan(level) {
		return
	}

	// 对于 Error 和 Fatal 级别的日志，不进行采样，始终记录
	// 对于 Warn 及以下级别的日志，进行采样
	if level.IsLowerOrEqualThan(Warn) {
		if !l.sampler.Sample() {
			return // 不记录这条日志
		}
	}

	attrs := make([]slog.Attr, 0, len(fields))
	for _, f := range fields {
		attrs = append(attrs, slog.Any(f.Key, f.Value))
	}

	record := slog.NewRecord(time.Now(), slog.Level(levelOrder[level]), msg, 0)
	record.AddAttrs(attrs...)

	select {
	case l.buffer <- record:
	case <-l.done:
		// 日志记录器已关闭，不再接受新的日志
	default:
		// 缓冲区已满，直接写入
		handler := l.handler.Load().(slog.Handler)
		_ = handler.Handle(ctx, record)
	}
}

// Debug 记录调试级别的日志
func (l *SlogLogger) Debug(msg string, fields ...Field) {
	l.log(context.Background(), Debug, msg, fields...)
}

// Info 记录信息级别的日志
func (l *SlogLogger) Info(msg string, fields ...Field) {
	l.log(context.Background(), Info, msg, fields...)
}

// Warn 记录警告级别的日志
func (l *SlogLogger) Warn(msg string, fields ...Field) {
	l.log(context.Background(), Warn, msg, fields...)
}

// Error 记录错误级别的日志
func (l *SlogLogger) Error(msg string, fields ...Field) {
	l.log(context.Background(), Error, msg, fields...)
}

// Fatal 记录致命错误级别的日志
func (l *SlogLogger) Fatal(msg string, fields ...Field) {
	l.log(context.Background(), Fatal, msg, fields...)
	os.Exit(1)
}

// DebugContext 记录带有上下文的调试级别日志
func (l *SlogLogger) DebugContext(ctx context.Context, msg string, fields ...Field) {
	l.log(ctx, Debug, msg, fields...)
}

// InfoContext 记录带有上下文的信息级别日志
func (l *SlogLogger) InfoContext(ctx context.Context, msg string, fields ...Field) {
	l.log(ctx, Info, msg, fields...)
}

// WarnContext 记录带有上下文的警告级别日志
func (l *SlogLogger) WarnContext(ctx context.Context, msg string, fields ...Field) {
	l.log(ctx, Warn, msg, fields...)
}

// ErrorContext 记录带有上下文的错误级别日志
func (l *SlogLogger) ErrorContext(ctx context.Context, msg string, fields ...Field) {
	l.log(ctx, Error, msg, fields...)
}

// FatalContext 记录带有上下文的致命错误级别日志
func (l *SlogLogger) FatalContext(ctx context.Context, msg string, fields ...Field) {
	l.log(ctx, Fatal, msg, fields...)
	os.Exit(1)
}

// WithTrace 添加追踪信息到日志
func (l *SlogLogger) WithTrace(ctx context.Context) HighPerformanceLogger {
	traceID, spanID := extractTraceInfo(ctx)
	oldHandler := l.handler.Load().(slog.Handler)

	// 创建新的属性，包含追踪信息
	newAttrs := []slog.Attr{
		slog.String("trace_id", traceID),
		slog.String("span_id", spanID),
	}
	newHandler := oldHandler.WithAttrs(newAttrs)
	newLogger := &SlogLogger{
		config: l.config,
		buffer: l.buffer,
		done:   l.done,
	}
	newLogger.handler.Store(newHandler)
	newLogger.level.Store(l.level.Load())

	return newLogger
}

// WithMetadata 添加元数据到日志
func (l *SlogLogger) WithMetadata(metadata map[string]any) HighPerformanceLogger {
	oldHandler := l.handler.Load().(slog.Handler)
	attrs := make([]slog.Attr, 0, len(metadata))
	for k, v := range metadata {
		attrs = append(attrs, slog.Any(k, v))
	}
	// 使用现有的处理器，添加新的属性
	newHandler := oldHandler.WithAttrs(attrs)
	newLogger := &SlogLogger{
		config: l.config,
		buffer: l.buffer,
		done:   l.done,
	}
	newLogger.handler.Store(newHandler)
	newLogger.level.Store(l.level.Load())
	return newLogger
}

// Flush 刷新所有缓冲的日志
func (l *SlogLogger) Flush() error {
	// 创建一个特殊的刷新信号
	flushDone := make(chan struct{})
	// 发送刷新信号到处理goroutine
	select {
	case l.buffer <- slog.Record{Time: time.Now(), Message: "FLUSH_SIGNAL"}: // 发送一个特殊的Record作为刷新信号
		// 等待刷新完成的信号
		<-flushDone
	default:
		// 如果缓冲区已满，直接返回，因为所有日志都会被处理
		return nil
	}
	return nil
}

// Close 关闭日志记录器
func (l *SlogLogger) Close() {
	close(l.done)
	l.wg.Wait()
}

// extractTraceInfo 从 context 中提取追踪信息
func extractTraceInfo(ctx context.Context) (string, string) {
	// 获取当前的 span
	span := trace.SpanFromContext(ctx)
	if !span.SpanContext().IsValid() {
		return "", ""
	}
	// 获取 trace ID 和 span ID
	traceID := span.SpanContext().TraceID().String()
	spanID := span.SpanContext().SpanID().String()
	return traceID, spanID
}

// SlogFactory 实现 LoggerFactory 接口
type SlogFactory struct{}

// CreateLogger 创建一个新的 SlogLogger 实例
func (f *SlogFactory) CreateLogger(config LogConfig) (HighPerformanceLogger, error) {
	return NewSlogLogger(config)
}

// UpdateSamplingRate 更新采样率
func (l *SlogLogger) UpdateSamplingRate(rate float64) {
	if l.sampler != nil {
		l.sampler.SetRate(rate)
	}
}

// GetSamplingRate 获取当前采样率
func (l *SlogLogger) GetSamplingRate() float64 {
	if l.sampler != nil {
		return l.sampler.GetRate()
	}
	return 1.0 // 默认不采样
}

// UpdateConfig 动态更新日志配置
func (l *SlogLogger) UpdateConfig(newConfig LogConfig) error {
	if err := validateConfig(&newConfig); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}
	// 更新日志级别
	if newConfig.Level != l.config.Level {
		err := l.SetLevel(newConfig.Level)
		if err != nil {
			return err
		}
	}

	// 更新采样配置
	if newConfig.Sampling != l.config.Sampling {
		var newSampler sample.Sampler
		switch newConfig.Sampling.Type {
		case sample.RateSamplerType:
			if newConfig.Sampling.Rate > 0 && newConfig.Sampling.Rate <= 1 {
				newSampler = sample.NewRateSampler(newConfig.Sampling.Rate)
			} else {
				newSampler = sample.NewRateSampler(1) // 默认不采样
			}
		case sample.JitterSamplerType:
			if newConfig.Sampling.Rate > 0 && newConfig.Sampling.Rate <= 1 {
				newSampler = sample.NewJitterSampler(newConfig.Sampling.Rate, newConfig.Sampling.Jitter)
			} else {
				newSampler = sample.NewJitterSampler(1, newConfig.Sampling.Jitter) // 默认不采样
			}
		default:
			newSampler = sample.NewRateSampler(1) // 默认使用 RateSampler 且不采样
		}
		l.sampler = newSampler
	}

	// 更新处理器
	if newConfig.Encoder != l.config.Encoder || newConfig.Writer != l.config.Writer {
		newHandler := createHandler(newConfig)
		l.handler.Store(newHandler)
	}
	// 更新其他配置
	l.config = newConfig
	return nil
}

// logInternalError 记录内部错误
func (l *SlogLogger) logInternalError(msg string) {
	internalErrorLogger.Println(msg)
}
