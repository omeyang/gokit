package xlog

import (
	"context"

	"go.opentelemetry.io/otel/trace"
)

// LogLevel 定义日志级别的类型
type LogLevel string

// 定义日志级别常量
const (
	Debug LogLevel = "DEBUG" // 调试级别
	Info  LogLevel = "INFO"  // 信息级别
	Warn  LogLevel = "WARN"  // 告警级别
	Error LogLevel = "ERROR" // 错误级别
	Fatal LogLevel = "FATAL" // 致命级别，记录后会导致程序退出
)

// levelOrder 定义日志级别的顺序
var levelOrder = map[LogLevel]int{
	Debug: 0,
	Info:  1,
	Warn:  2,
	Error: 3,
	Fatal: 4,
}

// IsLowerOrEqualThan 比较日志级别
func (l LogLevel) IsLowerOrEqualThan(other LogLevel) bool {
	return levelOrder[l] <= levelOrder[other]
}

// IsHighThan 日志级别高于指定级别
func (l LogLevel) IsHighThan(other LogLevel) bool {
	return levelOrder[l] > levelOrder[other]
}

// LoggerType 定义日志类型
type LoggerType string

// 定义日志类型常量
const (
	SlogLoggerType LoggerType = "slog"
	ZapLoggerType  LoggerType = "zap"
)

// Field 定义日志字段
type Field struct {
	Key   string
	Value any
}

// Logger 是一个支持结构化日志和上下文的通用日志接口
type Logger interface {
	// SetLevel 设置日志等级
	SetLevel(level LogLevel) error
	// GetLevel 获取日志等级
	GetLevel() LogLevel

	// Debug 调试级别
	Debug(msg string, fields ...Field)
	// Info 信息级别
	Info(msg string, fields ...Field)
	// Warn 告警级别
	Warn(msg string, fields ...Field)
	// Error 错误级别
	Error(msg string, fields ...Field)
	// Fatal 致命级别
	Fatal(msg string, fields ...Field)

	// DebugContext 带有上下文的调试级别
	DebugContext(ctx context.Context, msg string, fields ...Field)
	// InfoContext 带有上下文的信息级别
	InfoContext(ctx context.Context, msg string, fields ...Field)
	// WarnContext 带有上下文的告警级别
	WarnContext(ctx context.Context, msg string, fields ...Field)
	// ErrorContext 带有上下文的错误级别日志
	ErrorContext(ctx context.Context, msg string, fields ...Field)
	// FatalContext 带有上下文的致命错误级别日志
	FatalContext(ctx context.Context, msg string, fields ...Field)

	// Close 关闭日志记录器，执行任何必要的清理操作
	Close()
}

// HighPerformanceLogger 定义高性能日志接口，扩展基本 Logger
type HighPerformanceLogger interface {
	Logger
	WithTrace(ctx context.Context) HighPerformanceLogger        // trace追踪
	WithMetadata(metadata map[string]any) HighPerformanceLogger // 元数据, eg:k8s pod信息
	Flush() error                                               // 释放资源
}

// LoggerFactory 定义日志工厂接口
type LoggerFactory interface {
	CreateLogger(config LogConfig) (HighPerformanceLogger, error)
}

// ContextExtractor 定义了从上下文中提取信息的接口
type ContextExtractor interface {
	Extract(ctx context.Context) map[string]string
}

// DefaultContextExtractor 提供了默认的上下文提取实现
type DefaultContextExtractor struct {
	// 定义一个需要从上下文中提取的键的列表
	keys []string
}

// NewDefaultContextExtractor 创建一个新的 DefaultContextExtractor 实例
func NewDefaultContextExtractor(additionalKeys ...string) *DefaultContextExtractor {
	// 预定义一些常见的键
	defaultKeys := []string{"request_id", "user_id", "session_id"}
	return &DefaultContextExtractor{
		keys: append(defaultKeys, additionalKeys...),
	}
}

func (e *DefaultContextExtractor) Extract(ctx context.Context) map[string]string {
	info := make(map[string]string)
	// 提取 trace 信息
	span := trace.SpanFromContext(ctx)
	if span.SpanContext().IsValid() {
		info["trace_id"] = span.SpanContext().TraceID().String()
		info["span_id"] = span.SpanContext().SpanID().String()
	}

	// 提取预定义的键值
	for _, key := range e.keys {
		if value := ctx.Value(key); value != nil {
			if strValue, ok := value.(string); ok {
				info[key] = strValue
			}
		}
	}
	return info
}
