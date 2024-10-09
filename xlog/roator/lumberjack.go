package roator

import (
	"io"

	"gopkg.in/natefinch/lumberjack.v2"
)

// LumberjackRotator 实现了基于 lumberjack 的日志轮转器
type LumberjackRotator struct {
	logger *lumberjack.Logger
}

// NewLumberjackRotator 创建一个新的基于 lumberjack 的日志轮转器
func NewLumberjackRotator(config RotatorConfig) (LogRotator, error) {
	logger := &lumberjack.Logger{
		Filename:   config.Filename,
		MaxSize:    config.MaxSize,
		MaxBackups: config.MaxBackups,
		MaxAge:     config.MaxAge,
		Compress:   config.Compress,
	}

	return &LumberjackRotator{
		logger: logger,
	}, nil
}

// GetWriter 返回一个 io.Writer，可以用于写入日志
func (r *LumberjackRotator) GetWriter() (io.Writer, error) {
	return r.logger, nil
}

// Rotate 手动触发日志轮转
func (r *LumberjackRotator) Rotate() error {
	return r.logger.Rotate()
}
