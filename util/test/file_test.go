package test

import (
	"fmt"
	"os"
	"testing"

	"github.com/omeyang/gokit/util/xfile"

	"github.com/omeyang/gokit/util"
)

// TestDefaultRenameFunc 测试 DefaultRenameFunc
func TestDefaultRenameFunc(t *testing.T) {
	base := "testfile"
	ext := ".txt"
	expected := "testfile.txt"
	result := util.DefaultRenameFunc(base, ext)
	if result != expected {
		t.Errorf("Expected %s, got %s", expected, result)
	}
}

// 清理所有以 testfile-ome-26149 开头的文件
func cleanupTestFiles(identifier string) {
	files, err := os.ReadDir(".")
	if err != nil {
		fmt.Printf("Failed to read directory: %v\n", err)
		return
	}

	for _, file := range files {
		if !file.IsDir() && len(file.Name()) >= len(identifier) && file.Name()[:len(identifier)] == identifier {
			os.Remove(file.Name())
		}
	}
}

func TestDefaultSeqRename(t *testing.T) {
	base := "testfile-ome-26149"
	ext := ".txt"

	// 清理可能存在的测试文件
	cleanupTestFiles(base)
	defer cleanupTestFiles(base)

	expected := "testfile-ome-26149.txt"
	result := util.DefaultSeqRename(base, ext)
	if result != expected {
		t.Errorf("Expected %s, got %s", expected, result)
	}

	// 创建一个临时文件以测试重命名
	tmpFile, err := os.Create("testfile-ome-26149.txt")
	if err != nil {
		t.Fatalf("Failed to create temporary file: %v", err)
	}
	defer tmpFile.Close()

	expected = "testfile-ome-26149-1.txt"
	result = util.DefaultSeqRename(base, ext)
	if result != expected {
		t.Errorf("Expected %s, got %s", expected, result)
	}

	// 创建第二个临时文件以测试进一步重命名
	tmpFile2, err := os.Create("testfile-ome-26149-1.txt")
	if err != nil {
		t.Fatalf("Failed to create second temporary file: %v", err)
	}
	defer tmpFile2.Close()

	expected = "testfile-ome-26149-2.txt"
	result = util.DefaultSeqRename(base, ext)
	if result != expected {
		t.Errorf("Expected %s, got %s", expected, result)
	}
}

// TestSafeFileOperation 测试 SafeFileOperation
func TestSafeFileOperation(t *testing.T) {
	filePath := "testfile.txt"
	flag := os.O_CREATE | os.O_WRONLY
	perm := os.FileMode(0644)

	// 测试文件创建
	operation := func(file *os.File) error {
		_, err := file.WriteString("hello world")
		return err
	}

	file, err := xfile.SafeFileOperation(filePath, flag, perm, operation)
	if err != nil {
		t.Fatalf("Failed to perform file operation: %v", err)
	}
	defer os.Remove(filePath)
	defer file.Close()

	// 检查文件内容
	content := make([]byte, 11)
	file, err = os.Open(filePath)
	if err != nil {
		t.Fatalf("Failed to open file: %v", err)
	}
	_, err = file.Read(content)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}
	if string(content) != "hello world" {
		t.Errorf("Expected 'hello world', got %s", string(content))
	}

	// 测试 os.OpenFile 返回错误的情况
	invalidFilePath := string([]byte{0})
	_, err = xfile.SafeFileOperation(invalidFilePath, flag, perm, operation)
	if err == nil {
		t.Fatalf("Expected error for invalid file path, got nil")
	}

	// 测试 operation 返回错误的情况
	errorOperation := func(file *os.File) error {
		return os.ErrInvalid
	}
	_, err = xfile.SafeFileOperation(filePath, flag, perm, errorOperation)
	if err == nil {
		t.Fatalf("Expected error from operation, got nil")
	}
}

// TestEnsureDirExists 测试 EnsureDirExists
func TestEnsureDirExists(t *testing.T) {
	dirPath := "testdir"

	err := xfile.EnsureDirExists(dirPath)
	if err != nil {
		t.Fatalf("Failed to ensure directory exists: %v", err)
	}
	defer os.RemoveAll(dirPath)

	// 检查目录是否存在
	info, err := os.Stat(dirPath)
	if os.IsNotExist(err) {
		t.Fatalf("Directory does not exist")
	}
	if !info.IsDir() {
		t.Fatalf("Expected a directory, but found something else")
	}

	// 测试无效路径
	invalidDirPath := string([]byte{0})
	err = xfile.EnsureDirExists(invalidDirPath)
	if err == nil {
		t.Fatalf("Expected error for invalid directory path, got nil")
	}
}
