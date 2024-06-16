package pool_test

import (
	"testing"

	"github.com/omeyang/gokit/middleware/pool"
)

func TestBSONFactory(t *testing.T) {
	factory := pool.NewBSONFactory(10)
	batch := factory.New()

	if cap(*batch) != 10 {
		t.Errorf("Expected capacity 10, got %d", cap(*batch))
	}

	factory.Reset(batch)
	if len(*batch) != 0 {
		t.Errorf("Expected length 0, got %d", len(*batch))
	}
}

func TestBufferFactory(t *testing.T) {
	factory := pool.NewBufferFactory(1024)
	buffer := factory.New()

	if len(buffer) != 1024 {
		t.Errorf("Expected length 1024, got %d", len(buffer))
	}

	factory.Reset(buffer)
}

func TestBatchPool(t *testing.T) {
	factory := pool.NewBSONFactory(10)
	bp := pool.NewBatchPool[pool.BSONBatch](factory, 5)

	item := bp.Get()
	if cap(*item) != 10 {
		t.Errorf("Expected capacity 10, got %d", cap(*item))
	}

	bp.Put(item, factory)
	if len(*item) != 0 {
		t.Errorf("Expected length 0 after reset, got %d", len(*item))
	}

	bp.SetPoolSize(factory, 10)
	if bp.Get() == nil {
		t.Errorf("Expected non-nil item from pool")
	}
}
