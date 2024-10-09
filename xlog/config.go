package xlog

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/omeyang/gokit/metrics/sample"
)

// EncoderType 定义编码器类型
type EncoderType string

const (
	// TextEncoder 文本编码器类型
	TextEncoder EncoderType = "text"
	// JSONEncoder json编码器类型是默认编码类型
	JSONEncoder EncoderType = "json"
	// ProtoEncoder proto类型的暂不做实现
	ProtoEncoder EncoderType = "proto"
)

// LogConfig 定义日志配置
type LogConfig struct {
	// 日志级别
	Level LogLevel
	// 日志编码器类型
	Encoder EncoderType
	// 输出写入器
	Writer io.Writer
	// 异步缓冲区大小
	AsyncBufferSize int
	// 异步刷新间隔
	FlushInterval time.Duration
	// 是否启用调用者信息
	EnableCaller bool
	// 调用栈跳过的帧数
	CallerSkip int
	// 是否启用追踪
	EnableTracing bool
	// 是否启用 Kubernetes 集成
	EnableKubernetes bool
	// 采样配置
	Sampling struct {
		// 采样器类型
		Type sample.SamplerType
		// 采样率（0.0-1.0）
		Rate float64
		// 抖动时间（仅用于 JitterSampler）
		Jitter time.Duration
	}
	// 其他特定于实现的配置选项
	ExtraOptions map[string]any
}

// LoadConfig 从环境变量和配置文件加载配置
func LoadConfig(configPath string) (LogConfig, error) {
	config := LogConfig{
		Level:           LogLevel(os.Getenv("LOG_LEVEL")),
		Encoder:         EncoderType(os.Getenv("LOG_ENCODER")),
		AsyncBufferSize: getEnvInt("LOG_ASYNC_BUFFER_SIZE", 1000),
		FlushInterval:   time.Duration(getEnvInt("LOG_FLUSH_INTERVAL", 5)) * time.Second,
		EnableCaller:    getEnvBool("LOG_ENABLE_CALLER", false),
		EnableTracing:   getEnvBool("LOG_ENABLE_TRACING", false),
	}

	// 如果提供了配置文件路径，从文件中读取配置
	if configPath != "" {
		data, err := os.ReadFile(configPath)
		if err != nil {
			return config, fmt.Errorf("failed to read config file: %w", err)
		}

		ext := strings.ToLower(filepath.Ext(configPath))
		switch ext {
		case ".json":
			if err := json.Unmarshal(data, &config); err != nil {
				return config, fmt.Errorf("failed to parse JSON config file: %w", err)
			}
		case ".yaml", ".yml":
			if err := yaml.Unmarshal(data, &config); err != nil {
				return config, fmt.Errorf("failed to parse YAML config file: %w", err)
			}
		default:
			return config, fmt.Errorf("unsupported config file format: %s", ext)
		}
	}

	// 验证配置
	if err := validateConfig(&config); err != nil {
		return config, err
	}
	return config, nil
}

// 辅助函数，从环境变量中读取整数值
func getEnvInt(key string, defaultValue int) int {
	if value, exists := os.LookupEnv(key); exists {
		if i, err := strconv.Atoi(value); err == nil {
			return i
		}
	}
	return defaultValue
}

// 辅助函数，从环境变量中读取布尔值
func getEnvBool(key string, defaultValue bool) bool {
	if value, exists := os.LookupEnv(key); exists {
		if b, err := strconv.ParseBool(value); err == nil {
			return b
		}
	}
	return defaultValue
}
