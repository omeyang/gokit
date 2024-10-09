package roator

import (
	"io"
)

// LogRotator 定义日志轮转接口
type LogRotator interface {
	GetWriter() (io.Writer, error)
	Rotate() error
}

// RotatorConfig 定义日志轮转器的通用配置
type RotatorConfig struct {
	// Filename 是要写入日志的文件名
	// 如果路径中的目录不存在，会自动创建
	Filename string

	// MaxSize 是日志文件在轮转之前的最大大小（以MB为单位）
	// 默认值是 100MB
	MaxSize int

	// MaxBackups 是要保留的旧日志文件的最大数量
	// 默认是保留所有旧日志文件（虽然 MaxAge 可能仍会导致它们被删除）
	MaxBackups int

	// MaxAge 是保留旧日志文件的最大天数
	// 默认是不根据时间删除旧日志文件
	MaxAge int

	// Compress 确定是否应该使用 gzip 压缩轮转的日志文件
	// 默认是不压缩
	Compress bool
}

// RotatorFactory 定义创建轮转器的工厂函数类型
type RotatorFactory func(config RotatorConfig) (LogRotator, error)
