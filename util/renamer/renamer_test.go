package renamer

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/omeyang/gokit/util/xfile"
)

// testEnvironment 用于管理测试环境
type testEnvironment struct {
	dir     string
	oldPath string
	t       testing.TB
}

// setupTestEnvironment 创建测试环境并返回一个清理函数
func setupTestEnvironment(t testing.TB, filename string) *testEnvironment {
	t.Helper()
	dir := t.TempDir()
	oldPath := filepath.Join(dir, filename)

	return &testEnvironment{
		dir:     dir,
		oldPath: oldPath,
		t:       t,
	}
}

// cleanup 清理测试环境
func (te *testEnvironment) cleanup() {
	te.t.Helper()
	if err := os.RemoveAll(te.dir); err != nil {
		te.t.Fatalf("Failed to clean up test directory: %v", err)
	}
}

// createFile 在测试环境中创建一个新文件
func (te *testEnvironment) createFile(path string) {
	te.t.Helper()
	if err := os.WriteFile(path, []byte("test content"), 0644); err != nil {
		te.t.Fatalf("Failed to create test file: %v", err)
	}
}

func TestSequentialRenamer(t *testing.T) {
	testCases := []struct {
		name      string
		separator string
		files     []string
		setup     func(*testEnvironment)
	}{
		{
			name:      "Default separator",
			separator: "",
			files:     []string{"test.txt", "test-1.txt", "test-2.txt"},
			setup:     func(te *testEnvironment) { te.createFile(te.oldPath) },
		},
		{
			name:      "Custom separator",
			separator: "_",
			files:     []string{"test.txt", "test_1.txt", "test_2.txt"},
			setup:     func(te *testEnvironment) { te.createFile(te.oldPath) },
		},
		{
			name:      "Filename with existing separator",
			separator: "-",
			files:     []string{"test-file.txt", "test-file-1.txt", "test-file-2.txt"},
			setup:     func(te *testEnvironment) { te.createFile(te.oldPath) },
		},
		{
			name:      "Filename with existing number",
			separator: "-",
			files:     []string{"test-123.txt", "test-124.txt", "test-125.txt"},
			setup:     func(te *testEnvironment) { te.createFile(te.oldPath) },
		},
		{
			name:      "Non-existent file",
			separator: "-",
			files:     []string{"non-existent.txt"},
			setup:     func(te *testEnvironment) {},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			env := setupTestEnvironment(t, tc.files[0])
			defer env.cleanup()

			tc.setup(env)

			renamer := NewSequentialRenamer(tc.separator)

			for i, expectedFile := range tc.files {
				newPath, err := renamer.Rename(env.oldPath)
				if err != nil {
					t.Fatalf("Failed to rename file: %v", err)
				}

				if filepath.Base(newPath) != expectedFile {
					t.Errorf("Expected %s, got %s", expectedFile, filepath.Base(newPath))
				}

				if _, err := os.Stat(newPath); os.IsNotExist(err) {
					t.Errorf("File %s does not exist after renaming", newPath)
				}

				// For next iteration, update oldPath
				env.oldPath = filepath.Join(filepath.Dir(env.oldPath), tc.files[0])
				if i < len(tc.files)-1 {
					env.createFile(env.oldPath)
				}
			}
		})
	}
}

func TestTimestampRenamer(t *testing.T) {
	testCases := []struct {
		name          string
		separator     string
		input         string
		expectRemoval bool
		setup         func(*testEnvironment)
	}{
		{
			name:          "Default separator",
			separator:     "",
			input:         "test.txt",
			expectRemoval: false,
			setup:         func(te *testEnvironment) { te.createFile(te.oldPath) },
		},
		{
			name:          "Custom separator",
			separator:     "_",
			input:         "test.txt",
			expectRemoval: false,
			setup:         func(te *testEnvironment) { te.createFile(te.oldPath) },
		},
		{
			name:          "Filename with existing separator",
			separator:     "-",
			input:         "test-file.txt",
			expectRemoval: false,
			setup:         func(te *testEnvironment) { te.createFile(te.oldPath) },
		},
		{
			name:          "Filename with existing timestamp",
			separator:     "-",
			input:         "test-1234567890123.txt",
			expectRemoval: true,
			setup:         func(te *testEnvironment) { te.createFile(te.oldPath) },
		},
		{
			name:          "Non-existent file",
			separator:     "-",
			input:         "non-existent.txt",
			expectRemoval: false,
			setup:         func(te *testEnvironment) {},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			env := setupTestEnvironment(t, tc.input)
			defer env.cleanup()

			tc.setup(env)

			renamer := NewTimestampRenamer(tc.separator)

			newPath, err := renamer.Rename(env.oldPath)
			if err != nil {
				t.Fatalf("Failed to rename file: %v", err)
			}

			// Verify new file name format
			newFile := filepath.Base(newPath)
			parts := strings.SplitN(newFile, tc.separator, 2)
			if len(parts) != 2 {
				t.Fatalf("Expected 2 parts in filename, got %d", len(parts))
			}

			baseName := strings.TrimSuffix(parts[0], filepath.Ext(parts[0]))
			expectedBaseName := strings.TrimSuffix(tc.input, filepath.Ext(tc.input))
			if tc.expectRemoval {
				expectedBaseName = strings.TrimSuffix(expectedBaseName, "-1234567890123")
			}
			if baseName != expectedBaseName {
				t.Errorf("Expected base name %s, got %s", expectedBaseName, baseName)
			}

			timestamp := strings.TrimSuffix(parts[1], filepath.Ext(parts[1]))
			if _, err := strconv.ParseInt(timestamp, 10, 64); err != nil {
				t.Errorf("Expected timestamp, got %s", timestamp)
			}

			if _, err := os.Stat(newPath); os.IsNotExist(err) {
				t.Errorf("File %s does not exist after renaming", newPath)
			}
		})
	}
}

func TestDateTimeRenamer(t *testing.T) {
	testCases := []struct {
		name          string
		separator     string
		input         string
		expectRemoval bool
		setup         func(*testEnvironment)
	}{
		{
			name:          "Default separator",
			separator:     "",
			input:         "test.txt",
			expectRemoval: false,
			setup:         func(te *testEnvironment) { te.createFile(te.oldPath) },
		},
		{
			name:          "Custom separator",
			separator:     "_",
			input:         "test.txt",
			expectRemoval: false,
			setup:         func(te *testEnvironment) { te.createFile(te.oldPath) },
		},
		{
			name:          "Filename with existing separator",
			separator:     "-",
			input:         "test-file.txt",
			expectRemoval: false,
			setup:         func(te *testEnvironment) { te.createFile(te.oldPath) },
		},
		{
			name:          "Filename with existing datetime",
			separator:     "-",
			input:         "test-20210101-120000.txt",
			expectRemoval: true,
			setup:         func(te *testEnvironment) { te.createFile(te.oldPath) },
		},
		{
			name:          "Non-existent file",
			separator:     "-",
			input:         "non-existent.txt",
			expectRemoval: false,
			setup:         func(te *testEnvironment) {},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			env := setupTestEnvironment(t, tc.input)
			defer env.cleanup()

			tc.setup(env)

			renamer := NewDateTimeRenamer(tc.separator)

			newPath, err := renamer.Rename(env.oldPath)
			if err != nil {
				t.Fatalf("Failed to rename file: %v", err)
			}

			// Verify new file name format
			newFile := filepath.Base(newPath)
			parts := strings.SplitN(newFile, tc.separator, 2)
			if len(parts) != 2 {
				t.Fatalf("Expected 2 parts in filename, got %d", len(parts))
			}

			baseName := strings.TrimSuffix(parts[0], filepath.Ext(parts[0]))
			expectedBaseName := strings.TrimSuffix(tc.input, filepath.Ext(tc.input))
			if tc.expectRemoval {
				expectedBaseName = strings.TrimSuffix(expectedBaseName, "-20210101-120000")
			}
			if baseName != expectedBaseName {
				t.Errorf("Expected base name %s, got %s", expectedBaseName, baseName)
			}

			datetime := strings.TrimSuffix(parts[1], filepath.Ext(parts[1]))
			if _, err := time.Parse("20060102-150405", datetime); err != nil {
				t.Errorf("Expected datetime format, got %s", datetime)
			}

			if _, err := os.Stat(newPath); os.IsNotExist(err) {
				t.Errorf("File %s does not exist after renaming", newPath)
			}
		})
	}
}

func TestNewRenamers(t *testing.T) {
	tests := []struct {
		name      string
		separator string
		factory   func(...string) xfile.FileRenamer
	}{
		{"SequentialRenamer with default separator", "", NewSequentialRenamer},
		{"SequentialRenamer with custom separator", "_", NewSequentialRenamer},
		{"TimestampRenamer with default separator", "", NewTimestampRenamer},
		{"TimestampRenamer with custom separator", "_", NewTimestampRenamer},
		{"DateTimeRenamer with default separator", "", NewDateTimeRenamer},
		{"DateTimeRenamer with custom separator", "_", NewDateTimeRenamer},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var renamer xfile.FileRenamer
			if tt.separator == "" {
				renamer = tt.factory()
			} else {
				renamer = tt.factory(tt.separator)
			}

			if renamer == nil {
				t.Errorf("Expected non-nil renamer, got nil")
			}
		})
	}
}
