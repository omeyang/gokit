package test

import (
	"context"
	"testing"

	"github.com/omeyang/gokit/storage"

	"github.com/omeyang/gokit/util/retry"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func FuzzMongoDBImpl_ReadByPage(f *testing.F) {
	ctx := context.Background()
	retryPolicy := &retry.NoRetryPolicy{}
	basic := options.Client().ApplyURI("mongodb://localhost:27017")

	instance, err := storage.GetMongoDBInstance(ctx, retryPolicy, basic)
	if err != nil {
		f.Fatalf("Expected no error, but got %v", err)
	}
	defer instance.GetClient().Disconnect(ctx)

	f.Add("testdb", "testcoll", int64(0), int64(10))
	f.Fuzz(func(t *testing.T, dbName string, collName string, page int64, pageSize int64) {
		params := storage.PaginatedQueryParams{
			DbName:   dbName,
			CollName: collName,
			Page:     page,
			PageSize: pageSize,
		}
		_, err := instance.ReadByPage(ctx, params)
		if err != nil {
			t.Logf("Error occurred: %v", err)
		}
	})
}

func FuzzMongoDBImpl_BulkWriteWithRetry(f *testing.F) {
	ctx := context.Background()
	retryPolicy := &retry.NoRetryPolicy{}
	basic := options.Client().ApplyURI("mongodb://localhost:27017")

	instance, err := storage.GetMongoDBInstance(ctx, retryPolicy, basic)
	if err != nil {
		f.Fatalf("Expected no error, but got %v", err)
	}
	defer instance.GetClient().Disconnect(ctx)

	f.Add("testdb", "testcoll", []*bson.M{{"name": "John", "age": 30}, {"name": "Alice", "age": 25}})
	f.Fuzz(func(t *testing.T, dbName string, collName string, docs []*bson.M) {
		opts := &storage.BulkWriteOptions{
			DbName:      dbName,
			CollName:    collName,
			Documents:   docs,
			RetryPolicy: retryPolicy,
		}
		err := instance.BulkWriteWithRetry(ctx, opts)
		if err != nil {
			t.Logf("Error occurred: %v", err)
		}
	})
}
