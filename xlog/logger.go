package xlog

import (
	"context"
)

// LogLevel 定义日志级别的类型
type LogLevel string

// 定义日志级别常量
const (
	Debug LogLevel = "DEBUG"
	Info  LogLevel = "INFO"
	Warn  LogLevel = "WARN"
	Error LogLevel = "ERROR"
)

// logEntry 异步批量写入日志
type logEntry struct {
	level         LogLevel
	msg           string
	keysAndValues []any
	ctx           context.Context
}

// Logger 是一个支持结构化日志和上下文的通用日志接口
type Logger interface {
	// SetLevel 设置日志等级
	SetLevel(level LogLevel) error
	// GetLevel 获取日志等级
	GetLevel() LogLevel
	// Debug 调试级别
	Debug(msg string, keysAndValues ...any)
	// DebugContext 带有上下文的调试级别
	DebugContext(ctx context.Context, msg string, keysAndValues ...any)
	// Info 信息级别
	Info(msg string, keysAndValues ...any)
	// InfoContext 带有上下文的信息级别
	InfoContext(ctx context.Context, msg string, keysAndValues ...any)
	// Warn 告警级别
	Warn(msg string, keysAndValues ...any)
	// WarnContext 带有上下文的告警级别
	WarnContext(ctx context.Context, msg string, keysAndValues ...any)
	// Error 错误级别
	Error(msg string, keysAndValues ...any)
	// ErrorContext 带有上下文的错误级别日志
	ErrorContext(ctx context.Context, msg string, keysAndValues ...any)
	// With 其他信息
	With(fields map[string]any) Logger
	// Close 关闭日志记录器
	Close()
}
