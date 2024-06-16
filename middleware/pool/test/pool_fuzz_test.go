package pool_test

import (
	"testing"

	"github.com/omeyang/gokit/middleware/pool"

	"go.mongodb.org/mongo-driver/bson"
)

func FuzzBSONFactory(f *testing.F) {
	factory := pool.NewBSONFactory(10)

	f.Fuzz(func(t *testing.T, data []byte) {
		// 模拟使用 BSON 数据
		var doc bson.M
		if err := bson.Unmarshal(data, &doc); err == nil {
			batch := factory.New()
			*batch = append(*batch, doc)
			factory.Reset(batch)
		}
	})
}

func FuzzBufferFactory(f *testing.F) {
	factory := pool.NewBufferFactory(1024)

	f.Fuzz(func(t *testing.T, data []byte) {
		buffer := factory.New()
		copy(buffer, data)
		factory.Reset(buffer)
	})
}

func FuzzBatchPool(f *testing.F) {
	factory := pool.NewBSONFactory(10)
	bp := pool.NewBatchPool[pool.BSONBatch](factory, 5)

	f.Fuzz(func(t *testing.T, data []byte) {
		item := bp.Get()
		var doc bson.M
		if err := bson.Unmarshal(data, &doc); err == nil {
			*item = append(*item, doc)
		}
		bp.Put(item, factory)
	})
}
