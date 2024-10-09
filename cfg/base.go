package cfg

import (
	"context"
	"log"
	"sync"
	"sync/atomic"
	"time"
)

// DefaultNotifyTimeout 是通知观察者的默认超时时间
const DefaultNotifyTimeout = 3000 * time.Millisecond

// AtomicValue 是一个线程安全的值容器
type AtomicValue[T any] struct {
	value atomic.Value
}

// Store 原子地存储值
func (av *AtomicValue[T]) Store(val T) {
	av.value.Store(val)
}

// Load 原子地加载值
func (av *AtomicValue[T]) Load() T {
	return av.value.Load().(T)
}

// baseConfigInternal 存储 BaseConfig 的内部共享状态
type baseConfigInternal struct {
	mu            sync.RWMutex
	stopCh        chan struct{}
	notifyTimeout time.Duration
}

// BaseConfig 是 Config 接口的基础实现
type BaseConfig[T any] struct {
	value    AtomicValue[T] // 存储当前配置值 类型安全
	source   Source         // 配置数据源
	parser   Parser[T]      // 配置解析器
	watchers []chan<- T     // 配置变更观察者列表
	internal *baseConfigInternal
}

// BaseConfigOption 定义了 BaseConfig 的可选配置函数
type BaseConfigOption func(*baseConfigInternal)

// WithNotifyTimeout 设置通知超时时间
func WithNotifyTimeout(timeout time.Duration) BaseConfigOption {
	return func(bci *baseConfigInternal) {
		bci.notifyTimeout = timeout
	}
}

// NewBaseConfig 创建一个新的 BaseConfig 实例
func NewBaseConfig[T any](ctx context.Context, source Source,
	parser Parser[T], opts ...BaseConfigOption) (*BaseConfig[T], error) {
	internal := &baseConfigInternal{
		stopCh:        make(chan struct{}),
		notifyTimeout: DefaultNotifyTimeout,
	}

	// 应用可选配置
	for _, opt := range opts {
		opt(internal)
	}

	bc := &BaseConfig[T]{
		source:   source,
		parser:   parser,
		internal: internal,
	}

	// 初始加载配置
	err := bc.Load(ctx)
	if err != nil {
		return nil, err
	}

	// 开始监控配置变更
	err = bc.Start(ctx)
	if err != nil {
		return nil, err
	}

	return bc, nil
}

// Load 从源加载并解析配置
func (bc *BaseConfig[T]) Load(ctx context.Context) error {
	data, err := bc.source.Read(ctx)
	if err != nil {
		return err
	}

	config, err := bc.parser.Parse(data)
	if err != nil {
		return err
	}

	bc.value.Store(config)
	bc.notifyWatchers(config)
	return nil
}

// Get 返回当前配置
func (bc *BaseConfig[T]) Get() T {
	return bc.value.Load()
}

// Watch 监视配置变化并返回一个通道
func (bc *BaseConfig[T]) Watch(ctx context.Context) (<-chan T, error) {
	ch := make(chan T, 1)

	bc.internal.mu.Lock()
	bc.watchers = append(bc.watchers, ch)
	bc.internal.mu.Unlock()

	go func() {
		<-ctx.Done()
		bc.removeWatcher(ch)
	}()

	// 立即发送当前配置
	ch <- bc.Get()

	return ch, nil
}

// removeWatcher 从观察者列表中移除指定的观察者
func (bc *BaseConfig[T]) removeWatcher(ch chan<- T) {
	bc.internal.mu.Lock()
	defer bc.internal.mu.Unlock()
	for i, watcher := range bc.watchers {
		if watcher == ch {
			bc.watchers = append(bc.watchers[:i], bc.watchers[i+1:]...)
			close(ch)
			break
		}
	}
}

// notifyWatchers 通知所有观察者配置已更新
func (bc *BaseConfig[T]) notifyWatchers(config T) {
	bc.internal.mu.RLock()
	defer bc.internal.mu.RUnlock()

	for _, watcher := range bc.watchers {
		select {
		case watcher <- config:
		case <-time.After(bc.internal.notifyTimeout):
			log.Printf("警告: 观察者通道已满，更新超时")
		}
	}
}

// Start 开始监控配置源的变化
func (bc *BaseConfig[T]) Start(ctx context.Context) error {
	sourceChan, err := bc.source.Watch(ctx)
	if err != nil {
		return err
	}

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-bc.internal.stopCh:
				return
			case data := <-sourceChan:
				config, err := bc.parser.Parse(data)
				if err == nil {
					bc.value.Store(config)
					bc.notifyWatchers(config)
				} else {
					log.Printf("错误: 解析配置失败: %v", err)
				}
			}
		}
	}()

	return nil
}

// Stop 停止配置监控和所有观察者
func (bc *BaseConfig[T]) Stop() {
	close(bc.internal.stopCh)
	bc.internal.mu.Lock()
	defer bc.internal.mu.Unlock()
	for _, watcher := range bc.watchers {
		close(watcher)
	}
	bc.watchers = nil
}
