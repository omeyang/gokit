package cfg

import "context"

// Config 定义了配置接口
// 代表了配置本身,面向配置的使用者，它抽象了配置的管理和访问方式,使用者不需要关心配置的具体来源或存储方式
type Config[T any] interface {
	// Load 加载配置
	Load(ctx context.Context) error
	// Get 获取配置
	Get() T
	// Watch 监听配置变更
	Watch(ctx context.Context) (<-chan T, error)
}

// Source 定义了配置源接口
// 代表了配置的来源。它定义了如何读取和监视原始配置数据
// 面向配置的提供者，它抽象了配置数据的获取方式，使得我们可以轻松地支持不同的配置源（如文件、环境变量、远程服务等）
type Source interface {
	// Read 阅读配置源信息
	Read(ctx context.Context) ([]byte, error)
	// Watch 监听配置源信息
	Watch(ctx context.Context) (<-chan []byte, error)
}

// Parser 定义了配置解析器接口
type Parser[T any] interface {
	// Parse 解析配置
	Parse(data []byte) (T, error)
}
