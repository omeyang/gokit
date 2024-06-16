package test

import (
	"context"
	"log"
	"testing"
	"time"

	"github.com/omeyang/gokit/middleware/storage"
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
	mt := mtest.New(t, mtest.NewOptions().ClientOptions(options.Client().ApplyURI("mongodb://localhost:27017")))
	defer mt.Client.Disconnect(context.Background())

	mt.Run("simple connection test", func(mt *mtest.T) {
		ctx := context.TODO()

		// 尝试执行一个简单的数据库操作
		coll := mt.Client.Database("testdb").Collection("testcoll")
		_, err := coll.InsertOne(ctx, bson.M{"name": "test"})
		if err != nil {
			t.Fatalf("Failed to insert document: %v", err)
		}
	})
}

func TestReadByPage(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientOptions(options.Client().ApplyURI("mongodb://localhost:27017")))
	defer func() {
		if err := mt.Client.Disconnect(context.Background()); err != nil {
			t.Fatalf("Failed to disconnect client: %v", err)
		}
	}()

	mt.Run("successful pagination", func(mt *mtest.T) {
		ctx := context.TODO()

		// 分页参数
		params := storage.PaginatedQueryParams{
			DbName:   "testdb",
			CollName: "testcoll",
			Page:     0,
			PageSize: 2,
		}

		// 创建数据库和集合
		db := mt.Client.Database(params.DbName)
		coll := db.Collection(params.CollName)
		defer func() {
			if err := db.Drop(ctx); err != nil {
				t.Fatalf("Failed to drop database: %v", err)
			}
		}()

		// 插入测试数据
		docs := []interface{}{
			bson.D{{"name", "Alice"}},
			bson.D{{"name", "Bob"}},
		}
		if _, err := coll.InsertMany(ctx, docs); err != nil {
			t.Fatalf("InsertMany failed: %v", err)
		}

		// 初始化 MongoDBImpl 实例
		mongoDB := storage.GetMongoDBInstance(ctx, options.Client().ApplyURI("mongodb://localhost:27017"))

		// 执行分页读取
		results, err := mongoDB.ReadByPage(ctx, params)
		if err != nil {
			t.Fatalf("ReadByPage failed: %v", err)
		}

		if len(results) != 2 {
			t.Fatalf("Expected 2 documents, got %d", len(results))
		}
	})
}
