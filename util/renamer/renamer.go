package renamer

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"time"

	"github.com/omeyang/gokit/util/xfile"
)

// FileRenamer 定义文件重命名接口
type FileRenamer interface {
	// Rename 重命名文件
	// oldPath: 原文件路径
	// 返回新的文件路径和可能的错误
	Rename(oldPath string) (string, error)
}

// FileRenamerFactory 定义创建重命名器的工厂函数类型
type FileRenamerFactory func(separator ...string) FileRenamer

// DefaultSeparator 定义默认的分隔符
const DefaultSeparator = "-"

// SequentialRenamer 实现按顺序重命名
type SequentialRenamer struct {
	separator string
}

// NewSequentialRenamer 创建一个新的 SequentialRenamer 实例
// 参数 separator 是可选的，如果不提供，将使用默认分隔符
func NewSequentialRenamer(separator ...string) xfile.FileRenamer {
	sep := DefaultSeparator
	if len(separator) > 0 && separator[0] != "" {
		sep = separator[0]
	}
	return &SequentialRenamer{separator: sep}
}

// Rename 实现按顺序重命名文件
// 如果文件不存在，则创建一个新的空文件
// 如果文件存在，则在文件名后添加递增的数字
func (r *SequentialRenamer) Rename(oldPath string) (string, error) {
	dir, file := filepath.Split(oldPath)
	ext := filepath.Ext(file)
	base := file[:len(file)-len(ext)]

	// 如果文件不存在，直接创建一个新的空文件
	if _, err := os.Stat(oldPath); os.IsNotExist(err) {
		return oldPath, os.WriteFile(oldPath, []byte{}, 0o644)
	}

	seq := -1 // 从 -1 开始，这样第一次尝试时 seq 会是 0
	// 创建一个正则表达式来匹配文件名末尾的数字
	pattern := fmt.Sprintf(`%s(\d+)$`, regexp.QuoteMeta(r.separator))
	re := regexp.MustCompile(pattern)

	// 如果文件名末尾已经有数字，则提取出来作为起始序号
	if match := re.FindStringSubmatch(base); len(match) > 1 {
		if num, err := strconv.Atoi(match[1]); err == nil {
			seq = num - 1 // 减 1 是为了让循环中的 seq++ 后从正确的数字开始
			base = base[:len(base)-len(match[0])]
		}
	}

	// 循环尝试新的文件名，直到找到一个不存在的文件名
	for {
		seq++
		var newName string
		if seq == 0 {
			newName = base + ext
		} else {
			newName = fmt.Sprintf("%s%s%d%s", base, r.separator, seq, ext)
		}
		newPath := filepath.Join(dir, newName)
		if _, err := os.Stat(newPath); os.IsNotExist(err) {
			return newPath, os.Rename(oldPath, newPath)
		}
	}
}

// TimestampRenamer 实现使用时间戳重命名
type TimestampRenamer struct {
	separator string
}

// NewTimestampRenamer 创建一个新的 TimestampRenamer 实例
// 参数 separator 是可选的，如果不提供，将使用默认分隔符
func NewTimestampRenamer(separator ...string) xfile.FileRenamer {
	sep := DefaultSeparator
	if len(separator) > 0 && separator[0] != "" {
		sep = separator[0]
	}
	return &TimestampRenamer{separator: sep}
}

// Rename 实现使用时间戳重命名文件
// 如果文件不存在，则创建一个新的空文件
// 如果文件存在，则在文件名后添加当前时间戳
func (r *TimestampRenamer) Rename(oldPath string) (string, error) {
	dir, file := filepath.Split(oldPath)
	ext := filepath.Ext(file)
	base := file[:len(file)-len(ext)]

	// 如果文件不存在，直接创建一个新的空文件
	if _, err := os.Stat(oldPath); os.IsNotExist(err) {
		return oldPath, os.WriteFile(oldPath, []byte{}, 0o644)
	}

	// 移除已存在的时间戳（如果有）
	pattern := fmt.Sprintf(`%s\d{13}$`, regexp.QuoteMeta(r.separator))
	re := regexp.MustCompile(pattern)
	base = re.ReplaceAllString(base, "")

	// 生成新的时间戳并添加到文件名中
	timestamp := strconv.FormatInt(time.Now().UnixNano()/int64(time.Millisecond), 10)
	newName := fmt.Sprintf("%s%s%s%s", base, r.separator, timestamp, ext)
	newPath := filepath.Join(dir, newName)

	return newPath, os.Rename(oldPath, newPath)
}

// DateTimeRenamer 实现使用日期时间重命名
type DateTimeRenamer struct {
	separator string
}

// NewDateTimeRenamer 创建一个新的 DateTimeRenamer 实例
// 参数 separator 是可选的，如果不提供，将使用默认分隔符
func NewDateTimeRenamer(separator ...string) xfile.FileRenamer {
	sep := DefaultSeparator
	if len(separator) > 0 && separator[0] != "" {
		sep = separator[0]
	}
	return &DateTimeRenamer{separator: sep}
}

// Rename 实现使用日期时间重命名文件
// 如果文件不存在，则创建一个新的空文件
// 如果文件存在，则在文件名后添加当前日期和时间
func (r *DateTimeRenamer) Rename(oldPath string) (string, error) {
	dir, file := filepath.Split(oldPath)
	ext := filepath.Ext(file)
	base := file[:len(file)-len(ext)]

	// 如果文件不存在，直接创建一个新的空文件
	if _, err := os.Stat(oldPath); os.IsNotExist(err) {
		return oldPath, os.WriteFile(oldPath, []byte{}, 0o644)
	}

	// 移除已存在的日期时间（如果有）
	pattern := fmt.Sprintf(`%s\d{8}-\d{6}$`, regexp.QuoteMeta(r.separator))
	re := regexp.MustCompile(pattern)
	base = re.ReplaceAllString(base, "")

	// 生成新的日期时间并添加到文件名中
	datetime := time.Now().Format("20060102-150405")
	newName := fmt.Sprintf("%s%s%s%s", base, r.separator, datetime, ext)
	newPath := filepath.Join(dir, newName)

	return newPath, os.Rename(oldPath, newPath)
}
