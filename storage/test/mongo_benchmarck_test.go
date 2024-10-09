package test

import (
	"context"
	"testing"

	"github.com/omeyang/gokit/storage"

	"github.com/omeyang/gokit/util/retry"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func BenchmarkMongoDBImpl_ReadByPage(b *testing.B) {
	ctx := context.Background()
	retryPolicy := &retry.NoRetryPolicy{}
	basic := options.Client().ApplyURI("mongodb://localhost:27017")

	instance, err := storage.GetMongoDBInstance(ctx, retryPolicy, basic)
	if err != nil {
		b.Fatalf("Expected no error, but got %v", err)
	}
	defer instance.GetClient().Disconnect(ctx)

	params := storage.PaginatedQueryParams{
		DbName:   "testdb",
		CollName: "testcoll",
		Page:     0,
		PageSize: 10,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := instance.ReadByPage(ctx, params)
		if err != nil {
			b.Fatalf("Expected no error, but got %v", err)
		}
	}
}

func BenchmarkMongoDBImpl_BulkWriteWithRetry(b *testing.B) {
	ctx := context.Background()
	retryPolicy := &retry.NoRetryPolicy{}
	basic := options.Client().ApplyURI("mongodb://localhost:27017")

	instance, err := storage.GetMongoDBInstance(ctx, retryPolicy, basic)
	if err != nil {
		b.Fatalf("Expected no error, but got %v", err)
	}
	defer instance.GetClient().Disconnect(ctx)

	docs := []*bson.M{
		{
			"name": "John",
			"age":  30,
		},
		{
			"name": "Alice",
			"age":  25,
		},
	}
	opts := &storage.BulkWriteOptions{
		DbName:      "testdb",
		CollName:    "testcoll",
		Documents:   docs,
		RetryPolicy: retryPolicy,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := instance.BulkWriteWithRetry(ctx, opts)
		if err != nil {
			b.Fatalf("Expected no error, but got %v", err)
		}
	}
}
