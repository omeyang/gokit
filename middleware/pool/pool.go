package pool

import (
	"sync"

	"go.mongodb.org/mongo-driver/bson"
)

// ObjectFactory 定义对象创建和重置方法的接口
type ObjectFactory[T any] interface {
	New() T
	Reset(T)
}

// BatchPool 泛型对象池
type BatchPool[T any] struct {
	Pool     sync.Pool
	PoolSize int
}

// NewBatchPool 初始化一个泛型对象池
func NewBatchPool[T any](factory ObjectFactory[T], poolSize int) *BatchPool[T] {
	bp := &BatchPool[T]{
		PoolSize: poolSize,
		Pool: sync.Pool{
			New: func() any {
				return factory.New()
			},
		},
	}
	// 预先填充对象池
	for i := 0; i < poolSize; i++ {
		bp.Pool.Put(factory.New())
	}
	return bp
}

// Get 获取对象
func (bp *BatchPool[T]) Get() T {
	return bp.Pool.Get().(T)
}

// Put 放置对象
func (bp *BatchPool[T]) Put(item T, factory ObjectFactory[T]) {
	factory.Reset(item)
	bp.Pool.Put(item)
}

// SetPoolSize 更新池子的大小
func (bp *BatchPool[T]) SetPoolSize(factory ObjectFactory[T], poolSize int) {
	// 更新池子大小
	bp.PoolSize = poolSize
	// 预先填充对象池
	for i := 0; i < poolSize; i++ {
		bp.Pool.Put(factory.New())
	}
}

// BSONBatch 是 *[]bson.M 类型的别名
type BSONBatch *[]bson.M

// BSONFactory 实现 ObjectFactory 接口，用于创建和重置 BSONBatch 对象
type BSONFactory struct {
	PoolSize int
}

// New 创建新的 BSON 对象
func (f *BSONFactory) New() BSONBatch {
	batch := make([]bson.M, 0, f.PoolSize)
	return &batch
}

// Reset 重置 BSON 对象
func (f *BSONFactory) Reset(batch BSONBatch) {
	*batch = (*batch)[:0] // 重置切片，避免持有过多内存
}

// BufferFactory 实现 ObjectFactory 接口，用于创建和重置缓冲区
type BufferFactory struct {
	BufferSize int
}

// New 创建新的缓冲区
func (f *BufferFactory) New() []byte {
	buffer := make([]byte, f.BufferSize)
	return buffer
}

// Reset 重置缓冲区
func (f *BufferFactory) Reset(buffer []byte) {
	// 缓冲区无需实际重置，只需清空内容
	for i := range buffer {
		buffer[i] = 0
	}
}

// NewBSONFactory 构造函数
func NewBSONFactory(poolSize int) *BSONFactory {
	return &BSONFactory{PoolSize: poolSize}
}

// NewBufferFactory 构造函数
func NewBufferFactory(bufferSize int) *BufferFactory {
	return &BufferFactory{BufferSize: bufferSize}
}
