package test

import (
	"context"
	"errors"
	"log"
	"reflect"
	"testing"
	"time"

	"github.com/omeyang/gokit/storage"

	"github.com/omeyang/gokit/util/retry"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/integration/mtest"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func TestMongoDBConnection(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		t.Fatalf("Failed to connect to MongoDB: %v", err)
	}
	defer client.Disconnect(ctx)

	err = client.Ping(ctx, nil)
	if err != nil {
		t.Fatalf("Failed to ping MongoDB: %v", err)
	}
	log.Println("Successfully connected and pinged MongoDB.")
}

func TestSimpleMongoDBConnection(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))

	if mt.Client != nil {
		t.Errorf("client is nil : %v", mt.Client)
		defer mt.Client.Disconnect(context.Background())
	}

	mt.Run("simple connection test", func(mt *mtest.T) {
		mt.AddMockResponses(mtest.CreateSuccessResponse()) // 添加一个成功的 mock 响应

		ctx := context.TODO()

		// 尝试执行一个简单的数据库操作
		coll := mt.Client.Database("testdb").Collection("testcoll")
		_, err := coll.InsertOne(ctx, bson.M{"name": "test"})
		if err != nil {
			t.Fatalf("Failed to insert document: %v", err)
		}
	})
}
func TestGetMongoDBInstance(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))
	defer func() {
		if mt.Client != nil {
			mt.Client.Disconnect(context.Background())
		}
	}()

	mt.Run("GetMongoDBInstance", func(mt *mtest.T) {
		ctx := context.Background()
		retryPolicy := &retry.NoRetryPolicy{}
		basic := options.Client().ApplyURI("mongodb://localhost:27017")

		instance, err := storage.GetMongoDBInstance(ctx, retryPolicy, basic)
		if err != nil {
			t.Fatalf("Expected no error, but got %v", err)
		}
		if instance == nil {
			t.Fatal("Expected instance, but got nil")
		}
	})
}

func TestNewMongoDBImpl(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))
	defer func() {
		if mt.Client != nil {
			mt.Client.Disconnect(context.Background())
		}
	}()

	mt.Run("NewMongoDBImpl", func(mt *mtest.T) {
		ctx := context.Background()
		retryPolicy := &retry.NoRetryPolicy{}
		basic := options.Client().ApplyURI("mongodb://localhost:27017")

		instance, err := storage.GetMongoDBInstance(ctx, retryPolicy, basic)
		if err != nil {
			t.Fatalf("Expected no error, but got %v", err)
		}
		if instance == nil {
			t.Fatal("Expected instance, but got nil")
		}
	})
}

func TestMongoDBImpl_InitBatchPool(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))
	defer func() {
		if mt.Client != nil {
			mt.Client.Disconnect(context.Background())
		}
	}()

	mt.Run("InitBatchPool", func(mt *mtest.T) {
		instance := &storage.MongoDBImpl{}
		instance.InitBatchPool(10)

		if instance.GetBatchPool() == nil {
			t.Fatal("Expected batch pool, but got nil")
		}
	})
}

func TestMongoDBImpl_GetBatchPool(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))
	defer func() {
		if mt.Client != nil {
			mt.Client.Disconnect(context.Background())
		}
	}()

	mt.Run("GetBatchPool", func(mt *mtest.T) {
		instance := &storage.MongoDBImpl{}
		instance.InitBatchPool(10)

		batchPool := instance.GetBatchPool()
		if batchPool == nil {
			t.Fatal("Expected batch pool, but got nil")
		}
		if batchPool.PoolSize != 10 {
			t.Fatalf("Expected pool size 10, but got %d", batchPool.PoolSize)
		}
	})
}

func TestMongoDBImpl_SetPoolSize(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))
	defer func() {
		if mt.Client != nil {
			mt.Client.Disconnect(context.Background())
		}
	}()

	mt.Run("SetPoolSize", func(mt *mtest.T) {
		instance := &storage.MongoDBImpl{}
		instance.InitBatchPool(10)
		instance.SetPoolSize(20)

		if instance.GetBatchPool().PoolSize != 20 {
			t.Fatalf("Expected pool size 20, but got %d", instance.GetBatchPool().PoolSize)
		}
	})
}

func TestMongoDBImpl_GetClient(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))
	defer func() {
		if mt.Client != nil {
			mt.Client.Disconnect(context.Background())
		}
	}()

	mt.Run("GetClient", func(mt *mtest.T) {
		ctx := context.Background()
		retryPolicy := &retry.NoRetryPolicy{}
		basic := options.Client().ApplyURI("mongodb://localhost:27017")

		instance, err := storage.GetMongoDBInstance(ctx, retryPolicy, basic)
		if err != nil {
			t.Fatalf("Expected no error, but got %v", err)
		}

		client := instance.GetClient()
		if client == nil {
			t.Fatal("Expected client, but got nil")
		}
	})
}

func TestPaginatedQueryParams_Validate(t *testing.T) {
	testCases := []struct {
		name    string
		params  storage.PaginatedQueryParams
		wantErr bool
	}{
		{
			name: "Valid",
			params: storage.PaginatedQueryParams{
				DbName:   "testdb",
				CollName: "testcoll",
				Page:     0,
				PageSize: 10,
			},
			wantErr: false,
		},
		{
			name: "MissingDbName",
			params: storage.PaginatedQueryParams{
				CollName: "testcoll",
				Page:     0,
				PageSize: 10,
			},
			wantErr: true,
		},
		{
			name: "MissingCollName",
			params: storage.PaginatedQueryParams{
				DbName:   "testdb",
				Page:     0,
				PageSize: 10,
			},
			wantErr: true,
		},
		{
			name: "NegativePage",
			params: storage.PaginatedQueryParams{
				DbName:   "testdb",
				CollName: "testcoll",
				Page:     -1,
				PageSize: 10,
			},
			wantErr: true,
		},
		{
			name: "ZeroPageSize",
			params: storage.PaginatedQueryParams{
				DbName:   "testdb",
				CollName: "testcoll",
				Page:     0,
				PageSize: 0,
			},
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.params.Validate()
			if (err != nil) != tc.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}

func TestMongoDBImpl_ReadByPage(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))
	defer func() {
		if mt.Client != nil {
			mt.Client.Disconnect(context.Background())
		}
	}()

	testCases := []struct {
		name           string
		params         storage.PaginatedQueryParams
		mockResponses  []bson.D
		expectedResult []bson.M
		expectedErr    error
	}{
		{
			name: "NoData",
			params: storage.PaginatedQueryParams{
				DbName:   "testdb",
				CollName: "testcoll",
				Page:     0,
				PageSize: 10,
			},
			mockResponses:  []bson.D{},
			expectedResult: []bson.M{},
			expectedErr:    nil,
		},
		{
			name: "SinglePage",
			params: storage.PaginatedQueryParams{
				DbName:   "testdb",
				CollName: "testcoll",
				Page:     0,
				PageSize: 10,
			},
			mockResponses: []bson.D{
				{
					{Key: "name", Value: "John"}, {Key: "age", Value: 30},
				},
			},
			expectedResult: []bson.M{
				{"name": "John", "age": 30},
			},
			expectedErr: nil,
		},
		{
			name: "MultiplePages",
			params: storage.PaginatedQueryParams{
				DbName:   "testdb",
				CollName: "testcoll",
				Page:     1,
				PageSize: 2,
			},
			mockResponses: []bson.D{
				{
					{Key: "name", Value: "John"}, {Key: "age", Value: 30},
					{Key: "name", Value: "Alice"}, {Key: "age", Value: 25},
					{Key: "name", Value: "Bob"}, {Key: "age", Value: 35},
					{Key: "name", Value: "Charlie"}, {Key: "age", Value: 40},
				},
			},
			expectedResult: []bson.M{
				{"name": "Bob", "age": 35},
				{"name": "Charlie", "age": 40},
			},
			expectedErr: nil,
		},
	}

	for _, tc := range testCases {
		mt.Run(tc.name, func(mt *mtest.T) {
			ctx := context.Background()
			retryPolicy := &retry.NoRetryPolicy{}
			basic := options.Client().ApplyURI("mongodb://localhost:27017")

			instance, err := storage.GetMongoDBInstance(ctx, retryPolicy, basic)
			if err != nil {
				t.Fatalf("Expected no error, but got %v", err)
			}
			instance.InitBatchPool(10)

			mt.ClearMockResponses() // 清除之前的模拟响应
			mt.AddMockResponses(tc.mockResponses...)

			result, err := instance.ReadByPage(ctx, tc.params)
			if !errors.Is(err, tc.expectedErr) {
				t.Fatalf("Expected error %v, but got %v", tc.expectedErr, err)
			}
			if !reflect.DeepEqual(result, tc.expectedResult) {
				t.Fatalf("Expected result %v, but got %v", tc.expectedResult, result)
			}
		})
	}
}

func TestMongoDBImpl_BulkWriteWithRetry(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))
	defer func() {
		if mt.Client != nil {
			mt.Client.Disconnect(context.Background())
		}
	}()

	mt.Run("BulkWriteWithRetry", func(mt *mtest.T) {
		ctx := context.Background()
		retryPolicy := &retry.SimpleRetryPolicy{
			MaxAttempts: 3,
			WaitTime:    time.Second,
		}
		basic := options.Client().ApplyURI("mongodb://localhost:27017")

		instance, err := storage.GetMongoDBInstance(ctx, retryPolicy, basic)
		if err != nil {
			t.Fatalf("Expected no error, but got %v", err)
		}

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

		// Add mock response
		mt.AddMockResponses(mtest.CreateSuccessResponse())

		err = instance.BulkWriteWithRetry(ctx, opts)
		if err != nil {
			t.Fatalf("Expected no error, but got %v", err)
		}
	})

	mt.Run("BulkWriteWithRetry_NilOptions", func(mt *mtest.T) {
		ctx := context.Background()
		retryPolicy := &retry.NoRetryPolicy{}
		basic := options.Client().ApplyURI("mongodb://localhost:27017")

		instance, err := storage.GetMongoDBInstance(ctx, retryPolicy, basic)
		if err != nil {
			t.Fatalf("Expected no error, but got %v", err)
		}

		err = instance.BulkWriteWithRetry(ctx, nil)
		if err == nil {
			t.Fatal("Expected error, but got nil")
		}
	})

	mt.Run("BulkWriteWithRetry_NoRetry", func(mt *mtest.T) {
		ctx := context.Background()
		retryPolicy := &retry.NoRetryPolicy{}
		basic := options.Client().ApplyURI("mongodb://localhost:27017")

		instance, err := storage.GetMongoDBInstance(ctx, retryPolicy, basic)
		if err != nil {
			t.Fatalf("Expected no error, but got %v", err)
		}

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

		// Add mock response
		mt.AddMockResponses(mtest.CreateWriteErrorsResponse(mtest.WriteError{
			Index:   1,
			Code:    11000,
			Message: "duplicate key error",
		}))

		err = instance.BulkWriteWithRetry(ctx, opts)
		if err == nil {
			t.Fatal("Expected error, but got nil")
		}
	})

	mt.Run("BulkWriteWithRetry_RetrySuccess", func(mt *mtest.T) {
		ctx := context.Background()
		retryPolicy := &retry.ExponentialBackoffRetryPolicy{
			MaxAttempts:  3,
			BaseWaitTime: time.Millisecond,
		}
		basic := options.Client().ApplyURI("mongodb://localhost:27017")

		instance, err := storage.GetMongoDBInstance(ctx, retryPolicy, basic)
		if err != nil {
			t.Fatalf("Expected no error, but got %v", err)
		}

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

		// Add mock responses
		mt.AddMockResponses(
			mtest.CreateWriteErrorsResponse(mtest.WriteError{
				Index:   1,
				Code:    11000,
				Message: "duplicate key error",
			}),
			mtest.CreateSuccessResponse(),
		)

		err = instance.BulkWriteWithRetry(ctx, opts)
		if err != nil {
			t.Fatalf("Expected no error, but got %v", err)
		}
	})

	mt.Run("BulkWriteWithRetry_RetryFailure", func(mt *mtest.T) {
		ctx := context.Background()
		retryPolicy := &retry.ExponentialBackoffRetryPolicy{
			MaxAttempts:  2,
			BaseWaitTime: time.Millisecond,
		}
		basic := options.Client().ApplyURI("mongodb://localhost:27017")

		instance, err := storage.GetMongoDBInstance(ctx, retryPolicy, basic)
		if err != nil {
			t.Fatalf("Expected no error, but got %v", err)
		}

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

		// Add mock responses
		mt.AddMockResponses(
			mtest.CreateWriteErrorsResponse(mtest.WriteError{
				Index:   1,
				Code:    11000,
				Message: "duplicate key error",
			}),
			mtest.CreateWriteErrorsResponse(mtest.WriteError{
				Index:   1,
				Code:    11000,
				Message: "duplicate key error",
			}),
		)

		err = instance.BulkWriteWithRetry(ctx, opts)
		if err == nil {
			t.Fatal("Expected error, but got nil")
		}
	})
	mt.Run("BulkWriteWithRetry_ConvertToWriteModels", func(mt *mtest.T) {
		ctx := context.Background()
		retryPolicy := &retry.NoRetryPolicy{}
		basic := options.Client().ApplyURI("mongodb://localhost:27017")

		instance, err := storage.GetMongoDBInstance(ctx, retryPolicy, basic)
		if err != nil {
			t.Fatalf("Expected no error, but got %v", err)
		}

		docs := []*bson.M{
			{
				"name": "John",
				"age":  30,
			},
			nil,
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

		// Add mock response
		mt.AddMockResponses(mtest.CreateSuccessResponse())

		err = instance.BulkWriteWithRetry(ctx, opts)
		if err != nil {
			t.Fatalf("Expected no error, but got %v", err)
		}
	})
}
