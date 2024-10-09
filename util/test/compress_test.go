package test

import (
	"os"
	"path/filepath"
	"testing"

	omeyang "github.com/omeyang/gokit/util/xfile"
)

func TestFileCompressor_Compress_Decompress(t *testing.T) {
	// 创建测试目录
	testDir := filepath.Clean("testdir")
	err := os.MkdirAll(testDir, 0755)
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(testDir)

	// 清理测试目录中的相关文件
	cleanupTestCompressFiles(testDir)

	// 创建测试文件
	testFile := filepath.Join(testDir, "test.txt")
	err = os.WriteFile(testFile, []byte("Hello, World!"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	compressor := &omeyang.FileCompressor{}

	// 测试压缩
	compressFormats := []omeyang.CompressType{omeyang.GzCompressType, omeyang.TarGzCompressType, omeyang.ZipCompressType}
	for _, format := range compressFormats {
		compressParam := &omeyang.CompressParam{
			Source:      testFile,
			Destination: testDir,
			Format:      format,
			BufferSize:  1024,
			PoolSize:    5,
		}

		err := compressor.Compress(compressParam)
		if err != nil {
			t.Errorf("Compress failed for format %s: %v", format, err)
		}

		// 检查压缩文件是否存在
		compressedFile := filepath.Join(testDir, "test.txt"+string(format))
		if _, err := os.Stat(compressedFile); os.IsNotExist(err) {
			t.Errorf("Compressed file does not exist for format %s", format)
		}

		// 测试解压缩
		var decompressedPath string
		if format == omeyang.GzCompressType {
			// 对于 .gz 格式，解压缩路径应为文件路径
			decompressedPath = filepath.Join(testDir, "decompressed_test.txt")
		} else {
			// 对于 .tar.gz 和 .zip 格式，解压缩路径应为目录路径
			decompressedPath = filepath.Join(testDir, "decompressed"+string(format))
			err = os.MkdirAll(decompressedPath, 0755)
			if err != nil {
				t.Errorf("Failed to create decompressed directory: %v", err)
			}
		}

		decompressParam := &omeyang.DecompressParam{
			Source:      compressedFile,
			Destination: decompressedPath,
			BufferSize:  1024,
			PoolSize:    5,
		}

		err = compressor.Decompress(decompressParam)
		if err != nil {
			t.Errorf("Decompress failed for format %s: %v", format, err)
		}

		// 检查解压缩文件是否存在
		var decompressedFile string
		if format == omeyang.GzCompressType {
			decompressedFile = decompressedPath
		} else {
			decompressedFile = filepath.Join(decompressedPath, "test.txt")
		}
		if _, err := os.Stat(decompressedFile); os.IsNotExist(err) {
			t.Errorf("Decompressed file does not exist for format %s", format)
		}

		// 检查解压缩文件内容是否正确
		content, err := os.ReadFile(decompressedFile)
		if err != nil {
			t.Errorf("Failed to read decompressed file for format %s: %v", format, err)
		}
		if string(content) != "Hello, World!" {
			t.Errorf("Decompressed file content mismatch for format %s", format)
		}
	}

	// 清理测试目录中的相关文件
	cleanupTestCompressFiles(testDir)
}

func TestFileCompressor_Compress_NonExistentSource(t *testing.T) {
	// 创建测试目录
	testDir := filepath.Clean("testdir")
	err := os.MkdirAll(testDir, 0755)
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(testDir)

	// 清理测试目录中的相关文件
	cleanupTestCompressFiles(testDir)

	compressor := &omeyang.FileCompressor{}

	compressParam := &omeyang.CompressParam{
		Source:      "nonexistent.txt",
		Destination: testDir,
		Format:      omeyang.GzCompressType,
	}

	err = compressor.Compress(compressParam)
	if err == nil {
		t.Error("Expected error for non-existent source file, but got nil")
	}

	// 清理测试目录中的相关文件
	cleanupTestCompressFiles(testDir)
}

func TestFileCompressor_Decompress_NonExistentSource(t *testing.T) {
	// 创建测试目录
	testDir := filepath.Clean("testdir")
	err := os.MkdirAll(testDir, 0755)
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(testDir)

	// 清理测试目录中的相关文件
	cleanupTestCompressFiles(testDir)

	compressor := &omeyang.FileCompressor{}

	decompressParam := &omeyang.DecompressParam{
		Source:      "nonexistent.gz",
		Destination: testDir,
	}

	err = compressor.Decompress(decompressParam)
	if err == nil {
		t.Error("Expected error for non-existent source file, but got nil")
	}

	// 清理测试目录中的相关文件
	cleanupTestCompressFiles(testDir)
}

func TestFileCompressor_Compress_UnsupportedFormat(t *testing.T) {
	// 创建测试目录
	testDir := filepath.Clean("testdir")
	err := os.MkdirAll(testDir, 0755)
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(testDir)

	// 清理测试目录中的相关文件
	cleanupTestCompressFiles(testDir)

	compressor := &omeyang.FileCompressor{}

	compressParam := &omeyang.CompressParam{
		Source:      filepath.Join(testDir, "test.txt"),
		Destination: testDir,
		Format:      omeyang.CompressType("unsupported"),
	}

	err = compressor.Compress(compressParam)
	if err == nil {
		t.Error("Expected error for unsupported format, but got nil")
	}

	// 清理测试目录中的相关文件
	cleanupTestCompressFiles(testDir)
}

func TestFileCompressor_Decompress_UnsupportedFormat(t *testing.T) {
	// 创建测试目录
	testDir := filepath.Clean("testdir")
	err := os.MkdirAll(testDir, 0755)
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(testDir)

	// 清理测试目录中的相关文件
	cleanupTestCompressFiles(testDir)

	compressor := &omeyang.FileCompressor{}

	decompressParam := &omeyang.DecompressParam{
		Source:      filepath.Join(testDir, "test.unsupported"),
		Destination: testDir,
	}

	err = compressor.Decompress(decompressParam)
	if err == nil {
		t.Error("Expected error for unsupported format, but got nil")
	}

	// 清理测试目录中的相关文件
	cleanupTestCompressFiles(testDir)
}

// cleanupTestCompressFiles 清理测试目录中的相关文件
func cleanupTestCompressFiles(dir string) {
	files := []string{
		"test.txt",
		"test.txt.gz",
		"test.txt.tar.gz",
		"test.txt.zip",
		"decompressed_test.txt",
		"decompressed.gz",
		"decompressed.tar.gz",
		"decompressed.zip",
		"test.unsupported",
	}
	for _, file := range files {
		os.Remove(filepath.Join(dir, file))
	}
}
