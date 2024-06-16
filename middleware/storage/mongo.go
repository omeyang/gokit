package storage

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/omeyang/gokit/middleware/pool"
	"github.com/omeyang/gokit/util"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// PaginatedQueryParams 分页读取的选项
type PaginatedQueryParams struct {
	DbName   string
	CollName string
	Page     int64
	PageSize int64
}

// BulkWriteOptions 封装了BulkWriteWithRetry函数所需的参数
type BulkWriteOptions struct {
	DbName      string           // 数据库名称
	CollName    string           // 集合名称
	Documents   []*bson.M        // 要写入的文档指针数组
	RetryPolicy util.RetryPolicy // 重试策略接口
}

// MongoDB mongo接口定义
type MongoDB interface {
	// ReadByPage 读取mongo指定集合表中指定分页的数据
	ReadByPage(ctx context.Context, params PaginatedQueryParams) ([]bson.M, error)
	// BulkWriteWithRetry 带有重试策略的批量写操作
	BulkWriteWithRetry(ctx context.Context, opts *BulkWriteOptions) error
}

// dao层的实例变量
var (
	mongoDBInstance *MongoDBImpl
	mongoDBOnce     sync.Once
)

// MongoDBImpl mongo的具体实例
type MongoDBImpl struct {
	client     *mongo.Client
	once       sync.Once
	batchPool  *pool.BatchPool[pool.BSONBatch] // 临时对象池
	singleConn bool                            // 是否单例方式链接
}

// GetMongoDBInstance 单例方式初始化mongo实例
func GetMongoDBInstance(ctx context.Context, basic *options.ClientOptions,
	opts ...func(*options.ClientOptions)) *MongoDBImpl {
	mongoDBOnce.Do(func() {
		mongoDBInstance = newMongoDBImpl(ctx, basic, opts...)
	})
	return mongoDBInstance
}

// newMongoDBImpl 使用基本配置和可选的配置函数来创建一个新的 MongoDB 客户端实例
func newMongoDBImpl(ctx context.Context, basic *options.ClientOptions,
	opts ...func(*options.ClientOptions)) *MongoDBImpl {
	// 应用所有传入的配置选项到基本配置上 mongo-go-driver1.15以后发布的opts要注意顺序
	for _, opt := range opts {
		opt(basic) // 直接应用配置函数修改 basic 对象
	}
	mongoCli, err := mongo.Connect(ctx, basic)
	if err != nil {
		if mongoCli != nil {
			mongoCli.Disconnect(ctx)
		}
		return nil
	}
	// mongo-go-driver1.15 已经默认连接过了
	return &MongoDBImpl{
		client: mongoCli,
	}
}

// InitBatchPool 初始化对象池
func (m *MongoDBImpl) InitBatchPool(poolSize int) {
	bsonFactory := pool.NewBSONFactory(poolSize)
	m.batchPool = pool.NewBatchPool[pool.BSONBatch](bsonFactory, poolSize)
}

// SetPoolSize 设置连接池大小
func (m *MongoDBImpl) SetPoolSize(defaultPoolSize int) {
	if m.batchPool != nil {
		bsonFactory := pool.NewBSONFactory(defaultPoolSize)
		m.batchPool.SetPoolSize(bsonFactory, defaultPoolSize)
	}
}

// GetClient 返回mongo的client
func (m *MongoDBImpl) GetClient() *mongo.Client {
	return m.client
}

// SetClient 设置 MongoDB 客户端
func (m *MongoDBImpl) SetClient(client *mongo.Client) {
	m.client = client
}

// GetCollection 返回mongo的集合表
func (m *MongoDBImpl) GetCollection(dbName string, collName string) *mongo.Collection {
	return m.client.Database(dbName).Collection(collName)
}

// Validate 验证PaginatedQueryParams的字段是否已被正确设置
func (p *PaginatedQueryParams) Validate() error {
	if p.DbName == "" || p.CollName == "" || p.Page < 0 || p.PageSize <= 0 {
		return errors.New("invalid PaginatedQueryParams: missing required fields or values out of range")
	}
	return nil
}

// ReadByPage 读取指定页码的数据
func (m *MongoDBImpl) ReadByPage(ctx context.Context, params PaginatedQueryParams) ([]bson.M, error) {
	if err := params.Validate(); err != nil {
		return nil, err
	}
	coll := m.client.Database(params.DbName).Collection(params.CollName)
	findOpts := options.Find().SetSkip(params.Page * params.PageSize).SetLimit(params.PageSize)
	cursor, err := coll.Find(ctx, bson.M{}, findOpts)
	if err != nil {
		return nil, err
	}

	batchPtr := m.batchPool.Get()                                              // 从Pool中获取切片指针
	defer m.batchPool.Put(batchPtr, pool.NewBSONFactory(m.batchPool.PoolSize)) // 使用defer确保切片被正确回收
	batch := *batchPtr
	if err := cursor.All(ctx, &batch); err != nil {
		return nil, err
	}
	// 返回的是batch的副本，因此原始batch可以安全回收重用
	return append([]bson.M(nil), batch...), nil
}

// BulkWriteWithRetry 带有重试策略的批量写
func (m *MongoDBImpl) BulkWriteWithRetry(ctx context.Context, opts *BulkWriteOptions) error {
	if opts == nil {
		return fmt.Errorf("BulkWriteWithRetry param opt is nil :%v", opts)
	}
	coll := m.client.Database(opts.DbName).Collection(opts.CollName)
	writeModels := convertToWriteModels(opts.Documents)
	var err error
	for attempt := 1; ; attempt++ {
		// 使用随机写来提升性能
		_, err = coll.BulkWrite(ctx, writeModels, options.BulkWrite().SetOrdered(false))
		if err == nil {
			return nil // 成功写入
		}
		if !opts.RetryPolicy.ShouldRetry(attempt, err) {
			break
		}
		time.Sleep(time.Duration(opts.RetryPolicy.WaitDuration(attempt)) * time.Second)
	}
	return fmt.Errorf("bulk write failed: %w", err)
}

// convertToWriteModels 转换模型
func convertToWriteModels(docs []*bson.M) []mongo.WriteModel {
	models := make([]mongo.WriteModel, len(docs))
	for i, docPtr := range docs {
		if docPtr == nil {
			continue // 跳过nil指针，防止解引用nil指针导致运行时错误
		}
		models[i] = mongo.NewInsertOneModel().SetDocument(*docPtr) // 解引用
	}
	return models
}
