package xlog

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"gopkg.in/natefinch/lumberjack.v2"
)

// LogRotator 定义日志轮转接口
type LogRotator interface {
	GetWriter() (io.Writer, error)
	Rotate() error
}

// LumberjackRotator 使用 lumberjack 实现的日志轮转
type LumberjackRotator struct {
	logger *lumberjack.Logger
}

// NewLumberjackRotator 创建一个新的 LumberjackRotator 实例
func NewLumberjackRotator(filename string, maxSize, maxBackups, maxAge int, compress bool) *LumberjackRotator {
	return &LumberjackRotator{
		logger: &lumberjack.Logger{
			Filename:   filename,
			MaxSize:    maxSize,    // megabytes
			MaxBackups: maxBackups, // number of backups
			MaxAge:     maxAge,     // days
			Compress:   compress,   // compress backups
		},
	}
}

func (lr *LumberjackRotator) Rotate() error {
	return lr.logger.Rotate()
}

// GetWriter 返回日志轮转的 writer
func (lr *LumberjackRotator) GetWriter() (io.Writer, error) {
	return lr.logger, nil
}

// rotatorEntry entry结构
type rotatorEntry struct {
	message string
	container.list
	time time.Time
}

// CustomRotator 自定义日志轮转器
/*
 * 1. 满足按照时间(小时)或者大小即进行轮转， 日志大小或者时间满足一个就切割
 * 2. 最近的N个日志文件不压缩，其余的都压缩
 * 3. 保留日志文件满足时间或者数量中的任何一个， 最多保留X数目个日志文件与Y天日志文件
 * 4. 异步提交 批量写入
 *  即: 超出X个日志文件就删除较早的日志，超出Y天就删除较早的日志文件， 二者满足其一就进行删除
 */
type CustomRotator struct {
	logger        *lumberjack.Logger
	currentFile   *os.File
	mu            sync.Mutex
	notCompressed int           // 不需要压缩的日志文件数
	maxBackups    int           // 最多保存日志文件数
	maxAge        int           // 日志最多保存时间
	rotationFunc  func() string // 日志轮转策略函数

	buffer        []rotatorEntry    // 缓冲区
	bufferSize    int               // 缓冲区大小
	writeChannel  chan rotatorEntry // 通道
	flushInterval time.Duration     // 刷新间隔
}

// RotatorParams 创建自定义的轮转器的参数
type RotatorParams struct {
	Filename      string
	MaxSize       int
	NotCompressed int
	MaxBackups    int
	MaxAge        int
	Compress      bool
	RotationFunc  func() string
	BufferSize    int           // 缓冲区大小
	FlushInterval time.Duration // 刷新间隔
}

// NewCustomRotator 新建一个自定义的日志轮转器
func NewCustomRotator(params RotatorParams) *CustomRotator {
	cr := &CustomRotator{
		logger: &lumberjack.Logger{
			Filename:   params.Filename,
			MaxSize:    params.MaxSize,
			MaxBackups: params.MaxBackups,
			MaxAge:     params.MaxAge,
			Compress:   params.Compress,
		},
		notCompressed: params.NotCompressed,
		maxBackups:    params.MaxBackups,
		maxAge:        params.MaxAge,
		rotationFunc:  params.RotationFunc,
		buffer:        make([]rotatorEntry, 0, params.BufferSize),
		bufferSize:    params.BufferSize,
		writeChannel:  make(chan rotatorEntry, params.BufferSize),
		flushInterval: params.FlushInterval,
	}
	go cr.writeWorker()
	return cr
}

// writeWorker 专门批量写入
func (cr *CustomRotator) writeWorker() {
	ticker := time.NewTicker(cr.flushInterval)
	defer ticker.Stop()

	for {
		select {
		case entry := <-cr.writeChannel:
			cr.buffer = append(cr.buffer, entry)
			if len(cr.buffer) >= cr.bufferSize {
				cr.flushBuffer()
			}
		case <-ticker.C:
			cr.flushBuffer()
		}
	}
}

// flushBuffer 将缓冲区的日志写入文件
func (cr *CustomRotator) flushBuffer() {
	if len(cr.buffer) == 0 {
		return
	}
	cr.mu.Lock()
	defer cr.mu.Unlock()
	var writer io.Writer
	var err error
	if writer, err = cr.getWriterInternal(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get writer: %v\n", err)
		return
	}
	for _, entry := range cr.buffer {
		if _, err := writer.Write([]byte(
			fmt.Sprintf("%s %s\n", entry.time.Format(time.RFC3339), entry.message))); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to write log entry: %v\n", err)
		}
	}
	cr.buffer = cr.buffer[:0]
}

// getWriterInternal 生成唯一文件名并返回io.Writer
func (cr *CustomRotator) getWriterInternal() (io.Writer, error) {
	baseFilename := filepath.Join(filepath.Dir(cr.logger.Filename), cr.rotationFunc())
	filename := baseFilename
	seq := 1
	// 检查下是否已经存在同名文件了
	for {
		if _, err := os.Stat(filename); os.IsNotExist(err) {
			break
		}
		info, err := os.Stat(filename)
		if err != nil {
			return nil, err
		}
		if info.Size() < int64(cr.logger.MaxSize*1024*1024) {
			break
		}
		filename = fmt.Sprintf("%s-%d.log", baseFilename, seq)
		seq++
	}
	if cr.currentFile == nil || cr.currentFile.Name() != filename {
		if cr.currentFile != nil {
			cr.currentFile.Close()
		}
		var err error
		cr.currentFile, err = os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return nil, err
		}
		cr.logger.Filename = filename
	}

	return io.MultiWriter(cr.currentFile, cr.logger), nil
}

// GetWriter 自定义轮转器返回日志轮转的 writer
func (cr *CustomRotator) GetWriter() (io.Writer, error) {
	return &logWriter{cr: cr}, nil
}

// logWriter 写的结构体
type logWriter struct {
	cr *CustomRotator
}

// Write 实现写方法， 实际送到channel里面
func (lw *logWriter) Write(p []byte) (n int, err error) {
	entry := rotatorEntry{
		message: string(p),
		time:    time.Now(),
	}
	lw.cr.writeChannel <- entry
	return len(p), nil
}

// Rotate 手动触发日志轮转
func (cr *CustomRotator) Rotate() error {
	cr.mu.Lock()
	defer cr.mu.Unlock()
	if cr.currentFile != nil {
		cr.currentFile.Close()
		cr.currentFile = nil
	}
	return cr.logger.Rotate()
}

// Cleanup 清理旧日志文件
func (cr *CustomRotator) Cleanup() error {
	// 删除超过 maxBackups 的日志文件，并压缩旧文件
	files, err := filepath.Glob(filepath.Join(filepath.Dir(cr.logger.Filename), "*.log"))
	if err != nil {
		return err
	}
	// 按照操作时间降序排列
	sort.Slice(files, func(i, j int) bool {
		info1, _ := os.Stat(files[i])
		info2, _ := os.Stat(files[j])
		return info1.ModTime().After(info2.ModTime())
	})

	// 压缩旧的日志文件
	for i, file := range files {
		if i >= cr.notCompressed && strings.HasSuffix(file, ".log") {
			if err := compressFile(file); err != nil {
				return err
			}
		}
	}

	// 删除超出的日志文件
	now := time.Now()
	for i, file := range files {
		if i >= cr.maxBackups || now.Sub(getModTime(file)).Hours() > float64(cr.maxAge*24) {
			if err := os.Remove(file); err != nil {
				return err
			}
		}
	}
	return nil
}

// CleanupOldFiles 定期清理旧的日志文件
func (cr *CustomRotator) CleanupOldFiles() {
	ticker := time.NewTicker(24 * time.Hour)
	for range ticker.C {
		cr.Cleanup()
	}
}
