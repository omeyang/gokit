package pool_test

import (
	"testing"

	"github.com/omeyang/gokit/middleware/pool"
)

func BenchmarkBSONFactory(b *testing.B) {
	factory := pool.NewBSONFactory(10)
	for i := 0; i < b.N; i++ {
		batch := factory.New()
		factory.Reset(batch)
	}
}

func BenchmarkBufferFactory(b *testing.B) {
	factory := pool.NewBufferFactory(1024)
	for i := 0; i < b.N; i++ {
		buffer := factory.New()
		factory.Reset(buffer)
	}
}

func BenchmarkBatchPool(b *testing.B) {
	factory := pool.NewBSONFactory(10)
	bp := pool.NewBatchPool[pool.BSONBatch](factory, 5)
	for i := 0; i < b.N; i++ {
		item := bp.Get()
		bp.Put(item, factory)
	}
}
