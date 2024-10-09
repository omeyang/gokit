package xfile

import (
	"os"
)

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
