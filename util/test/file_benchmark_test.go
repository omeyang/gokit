package test

import (
	"os"
	"testing"

	"github.com/omeyang/gokit/util"
)

// BenchmarkDefaultRenameFunc 基准测试 DefaultRenameFunc
func BenchmarkDefaultRenameFunc(b *testing.B) {
	base := "testfile"
	ext := ".txt"
	for i := 0; i < b.N; i++ {
		_ = util.DefaultRenameFunc(base, ext)
	}
}

// BenchmarkDefaultSeqRename 基准测试 DefaultSeqRename
func BenchmarkDefaultSeqRename(b *testing.B) {
	base := "testfile"
	ext := ".txt"
	for i := 0; i < b.N; i++ {
		_ = util.DefaultSeqRename(base, ext)
	}
}

// BenchmarkDefaultSeqRenameWithConflict 测试有文件名冲突的情况
func BenchmarkDefaultSeqRenameWithConflict(b *testing.B) {
	base := "testfile"
	ext := ".txt"
	// 创建一个临时文件以测试重命名
	tmpFile, err := os.Create("testfile.txt")
	if err != nil {
		b.Fatalf("Failed to create temporary file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	for i := 0; i < b.N; i++ {
		_ = util.DefaultSeqRename(base, ext)
	}
}

// BenchmarkSafeFileOperation 基准测试 SafeFileOperation
func BenchmarkSafeFileOperation(b *testing.B) {
	filePath := "testfile.txt"
	flag := os.O_CREATE | os.O_WRONLY
	perm := os.FileMode(0644)

	operation := func(file *os.File) error {
		_, err := file.WriteString("hello world")
		return err
	}

	for i := 0; i < b.N; i++ {
		file, err := util.SafeFileOperation(filePath, flag, perm, operation)
		if err != nil {
			b.Fatalf("Failed to perform file operation: %v", err)
		}
		file.Close()
		os.Remove(filePath)
	}
}

// BenchmarkSafeFileOperationWithError 基准测试 SafeFileOperation 处理错误的情况
func BenchmarkSafeFileOperationWithError(b *testing.B) {
	filePath := string([]byte{0}) // 无效文件路径
	flag := os.O_CREATE | os.O_WRONLY
	perm := os.FileMode(0644)

	operation := func(file *os.File) error {
		_, err := file.WriteString("hello world")
		return err
	}

	for i := 0; i < b.N; i++ {
		_, err := util.SafeFileOperation(filePath, flag, perm, operation)
		if err == nil {
			b.Fatalf("Expected error for invalid file path, got nil")
		}
	}
}

// BenchmarkEnsureDirExists 基准测试 EnsureDirExists
func BenchmarkEnsureDirExists(b *testing.B) {
	dirPath := "testdir"
	for i := 0; i < b.N; i++ {
		err := util.EnsureDirExists(dirPath)
		if err != nil {
			b.Fatalf("Failed to ensure directory exists: %v", err)
		}
		os.RemoveAll(dirPath)
	}
}

// BenchmarkEnsureDirExistsWithError 基准测试 EnsureDirExists 处理错误的情况
func BenchmarkEnsureDirExistsWithError(b *testing.B) {
	invalidDirPath := string([]byte{0}) // 无效目录路径
	for i := 0; i < b.N; i++ {
		err := util.EnsureDirExists(invalidDirPath)
		if err == nil {
			b.Fatalf("Expected error for invalid directory path, got nil")
		}
	}
}
