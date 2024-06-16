package xlog

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
)

// SlogLogger 使用 slog 实现实例; 异步批量写入; 单例加载实例
type SlogLogger struct {
	logger  *slog.Logger
	level   LogLevel
	rotator LogRotator // 日志轮转工具
	ch      chan logEntry
	wg      sync.WaitGroup
	mu      sync.RWMutex // 并发情况下保证设置日志等级与获取日志等级协程安全
}

var (
	once     sync.Once
	instance *SlogLogger
)

// NewSlogLogger 创建一个新的 SlogLogger 实例，并接受一个可选的缓冲区大小参数。
func NewSlogLogger(rotator LogRotator, handler slog.Handler, level LogLevel, bufferSize int) Logger {
	once.Do(func() {
		size := 100 // 默认缓冲区大小
		if bufferSize > 0 {
			size = bufferSize
		}
		instance = &SlogLogger{
			logger:  slog.New(handler),
			level:   level,                     // 默认日志级别
			ch:      make(chan logEntry, size), // 缓冲区大小
			rotator: rotator,
		}
		instance.wg.Add(1)
		go instance.processLogEntries()
	})
	return instance
}

// NewDefaultSlogLogger 创建一个默认配置的 SlogLogger 实例。
func NewDefaultSlogLogger(filename string) Logger {
	rotator := NewLumberjackRotator(filename, 100, 10, 30, true)
	handler := slog.NewJSONHandler(rotator.GetWriter(), nil)
	return NewSlogLogger(rotator, handler, Info, 100)
}

// processLogEntries 处理日志条目
func (s *SlogLogger) processLogEntries() {
	defer s.wg.Done()
	for entry := range s.ch {
		switch entry.level {
		case Debug:
			if entry.ctx != nil {
				s.logger.DebugContext(entry.ctx, entry.msg, entry.keysAndValues...)
			} else {
				s.logger.Debug(entry.msg, entry.keysAndValues...)
			}
		case Info:
			if entry.ctx != nil {
				s.logger.InfoContext(entry.ctx, entry.msg, entry.keysAndValues...)
			} else {
				s.logger.Info(entry.msg, entry.keysAndValues...)
			}
		case Warn:
			if entry.ctx != nil {
				s.logger.WarnContext(entry.ctx, entry.msg, entry.keysAndValues...)
			} else {
				s.logger.Warn(entry.msg, entry.keysAndValues...)
			}
		case Error:
			if entry.ctx != nil {
				s.logger.ErrorContext(entry.ctx, entry.msg, entry.keysAndValues...)
			} else {
				s.logger.Error(entry.msg, entry.keysAndValues...)
			}
		}
	}
}

// SetLevel 设置日志等级
func (s *SlogLogger) SetLevel(level LogLevel) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	switch level {
	case Debug, Info, Warn, Error:
		s.level = level
		return nil
	default:
		return fmt.Errorf("unknown log level: %s", level)
	}
}

// GetLevel 获取日志等级
func (s *SlogLogger) GetLevel() LogLevel {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.level
}

// log 记录日志发送到缓冲区
func (s *SlogLogger) log(level LogLevel, msg string, keysAndValues ...any) {
	if s.compareLevels(level) {
		s.ch <- logEntry{level: level, msg: msg, keysAndValues: keysAndValues}
	}
}

// logWithContext 带有上下文的日志记录器
func (s *SlogLogger) logWithContext(ctx context.Context, level LogLevel, msg string, keysAndValues ...any) {
	if s.compareLevels(level) {
		s.ch <- logEntry{level: level, msg: msg, keysAndValues: keysAndValues, ctx: ctx}
	}
}

// Debug 调试级别日志
func (s *SlogLogger) Debug(msg string, keysAndValues ...any) {
	s.log(Debug, msg, keysAndValues...)
}

// DebugContext 带有上下文的调试级别日志
func (s *SlogLogger) DebugContext(ctx context.Context, msg string, keysAndValues ...any) {
	s.logWithContext(ctx, Debug, msg, keysAndValues...)
}

// Info 信息级别的日志
func (s *SlogLogger) Info(msg string, keysAndValues ...any) {
	s.log(Info, msg, keysAndValues...)
}

// InfoContext 带有上下文的信息级别日志
func (s *SlogLogger) InfoContext(ctx context.Context, msg string, keysAndValues ...any) {
	s.logWithContext(ctx, Info, msg, keysAndValues...)
}

// Warn 警告级别的日志
func (s *SlogLogger) Warn(msg string, keysAndValues ...any) {
	s.log(Warn, msg, keysAndValues...)
}

// WarnContext 带有上下文的警告级别日志
func (s *SlogLogger) WarnContext(ctx context.Context, msg string, keysAndValues ...any) {
	s.logWithContext(ctx, Warn, msg, keysAndValues...)
}

// Error 错误级别的日志
func (s *SlogLogger) Error(msg string, keysAndValues ...any) {
	s.log(Error, msg, keysAndValues...)
}

// ErrorContext 带有上下文的错误级别日志
func (s *SlogLogger) ErrorContext(ctx context.Context, msg string, keysAndValues ...any) {
	s.logWithContext(ctx, Error, msg, keysAndValues...)
}

// With 带有更多信息的日志
func (s *SlogLogger) With(fields map[string]any) Logger {
	newLogger := s.logger.With(fields)
	return &SlogLogger{logger: newLogger, level: s.level, ch: s.ch}
}

// compareLevels 比较日志级别
func (s *SlogLogger) compareLevels(level LogLevel) bool {
	switch s.level {
	case Debug:
		return true
	case Info:
		return level != Debug
	case Warn:
		return level == Warn || level == Error
	case Error:
		return level == Error
	default:
		return false
	}
}

// Close 关闭日志记录器，等待所有日志写入完成
func (s *SlogLogger) Close() {
	close(s.ch)
	s.wg.Wait()
}

// SetBufferSize 动态设置缓冲区大小
func (s *SlogLogger) SetBufferSize(size int) {
	newCh := make(chan logEntry, size)
	s.mu.Lock()
	oldCh := s.ch
	s.ch = newCh
	s.mu.Unlock()

	// 将旧缓冲区中的日志条目转移到新缓冲区
	go func() {
		for entry := range oldCh {
			newCh <- entry
		}
		close(newCh)
	}()
}

// SetFormat 设置日志格式
func (s *SlogLogger) SetFormat(format string) error {
	var handler slog.Handler
	switch format {
	case "json":
		handler = slog.NewJSONHandler(s.rotator.GetWriter(), nil)
	case "text":
		handler = slog.NewTextHandler(s.rotator.GetWriter(), nil)
	default:
		return fmt.Errorf("unknown format: %s", format)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.logger = slog.New(handler)
	return nil
}
