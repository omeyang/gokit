package roator

import (
	"io"
	"os"
	"path/filepath"
	"time"

	"go.uber.org/zap/zapcore"
)

// ZapRotator 实现了基于 zap 的日志轮转器
type ZapRotator struct {
	config RotatorConfig
	writer zapcore.WriteSyncer
	size   int64
}

// NewZapRotator 创建一个新的基于 zap 的日志轮转器
func NewZapRotator(config RotatorConfig) (LogRotator, error) {
	writer, err := os.OpenFile(config.Filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, err
	}
	info, err := writer.Stat()
	if err != nil {
		return nil, err
	}
	return &ZapRotator{
		config: config,
		writer: zapcore.AddSync(writer),
		size:   info.Size(),
	}, nil
}

// GetWriter 返回一个 io.Writer，可以用于写入日志
func (r *ZapRotator) GetWriter() (io.Writer, error) {
	return r.writer, nil
}

// Rotate 手动触发日志轮转
func (r *ZapRotator) Rotate() error {
	_ = r.writer.Sync()
	if closer, ok := r.writer.(io.Closer); ok {
		_ = closer.Close()
	}

	// 重命名当前日志文件
	newName := r.config.Filename + "." + time.Now().Format("2006-01-02_15-04-05")
	if err := os.Rename(r.config.Filename, newName); err != nil {
		return err
	}

	// 创建新的日志文件
	newFile, err := os.OpenFile(r.config.Filename, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	r.writer = zapcore.AddSync(newFile)
	r.size = 0

	// 清理旧日志文件
	r.cleanOldLogs()

	return nil
}

// 清理旧的日志
func (r *ZapRotator) cleanOldLogs() {
	dir := filepath.Dir(r.config.Filename)
	base := filepath.Base(r.config.Filename)
	pattern := filepath.Join(dir, base+".*")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return
	}

	for _, match := range matches {
		info, err := os.Stat(match)
		if err != nil {
			continue
		}
		if time.Since(info.ModTime()) > time.Duration(r.config.MaxAge)*24*time.Hour {
			os.Remove(match)
		}
	}

	if r.config.MaxBackups > 0 && len(matches) > r.config.MaxBackups {
		for _, match := range matches[r.config.MaxBackups:] {
			os.Remove(match)
		}
	}
}

// Write实现些操作
func (r *ZapRotator) Write(p []byte) (n int, err error) {
	n, err = r.writer.Write(p)
	r.size += int64(n)
	if r.size > int64(r.config.MaxSize*1024*1024) {
		_ = r.Rotate()
	}
	return n, err
}
