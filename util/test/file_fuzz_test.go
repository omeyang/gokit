package test

import (
	"os"
	"testing"

	"github.com/omeyang/gokit/util"
)

// FuzzDefaultRenameFunc fuzzing测试 DefaultRenameFunc
func FuzzDefaultRenameFunc(f *testing.F) {
	f.Add("base", ".ext")
	f.Fuzz(func(t *testing.T, base, ext string) {
		_ = util.DefaultRenameFunc(base, ext)
	})
}

// FuzzDefaultSeqRename fuzzing测试 DefaultSeqRename
func FuzzDefaultSeqRename(f *testing.F) {
	f.Add("base", ".ext")
	f.Fuzz(func(t *testing.T, base, ext string) {
		_ = util.DefaultSeqRename(base, ext)
	})
}

// FuzzSafeFileOperation fuzzing测试 SafeFileOperation
func FuzzSafeFileOperation(f *testing.F) {
	f.Add("testfile")
	f.Fuzz(func(t *testing.T, filePath string) {
		// 仅测试文件路径和简单文件操作
		flag := os.O_CREATE | os.O_RDWR
		perm := os.FileMode(0644)
		operation := func(file *os.File) error {
			_, err := file.WriteString("test")
			return err
		}
		_, _ = util.SafeFileOperation(filePath, flag, perm, operation)
	})
}

// FuzzEnsureDirExists fuzzing测试 EnsureDirExists
func FuzzEnsureDirExists(f *testing.F) {
	f.Add("testdir")
	f.Fuzz(func(t *testing.T, dirPath string) {
		_ = util.EnsureDirExists(dirPath)
	})
}
