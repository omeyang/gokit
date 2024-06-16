package util

import (
	"fmt"
	"os"
)

// DefaultRenameFunc 默认的重命名函数，与原文件保持一致
func DefaultRenameFunc(base, ext string) string {
	return base + ext
}

// DefaultSeqRename 有重复则在后面添加seq的重命名压缩包的方式
func DefaultSeqRename(base, ext string) string {
	seq := 1
	newName := base + ext
	for {
		if _, err := os.Stat(newName); os.IsNotExist(err) {
			break
		}
		newName = fmt.Sprintf("%s-%d%s", base, seq, ext)
		seq++
	}
	return newName
}

// SafeFileOperation 安全的文件操作
func SafeFileOperation(filePath string, flag int, perm os.FileMode, operation func(*os.File) error) (*os.File, error) {
	outFile, err := os.OpenFile(filePath, flag, perm)
	if err != nil {
		return nil, err
	}
	if operation != nil {
		if err = operation(outFile); err != nil {
			outFile.Close()
			return nil, err
		}
	}
	return outFile, nil
}

// EnsureDirExists 确定目录存在
func EnsureDirExists(dirPath string) error {
	if err := os.MkdirAll(dirPath, os.ModePerm); err != nil {
		return err
	}
	return nil
}
