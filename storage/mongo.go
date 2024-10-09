package storage

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/omeyang/gokit/util/retry"

	"github.com/omeyang/gokit/middleware/pool"
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
	DbName      string            // 数据库名称
	CollName    string            // 集合名称
	Documents   []*bson.M         // 要写入的文档指针数组
	RetryPolicy retry.RetryPolicy // 重试策略接口
}

// FieldMapping 定义字段映射
type FieldMapping map[string]string

// RenameOption 定义重命名选项
type RenameOption struct {
	Mapping      FieldMapping
	KeepOriginal bool // 是否保留原始字段
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

// GetMongoDBInstance 单例方式初始化 MongoDB 实例
func GetMongoDBInstance(ctx context.Context, retryPolicy retry.RetryPolicy, basic *options.ClientOptions,
	opts ...func(*options.ClientOptions),
) (*MongoDBImpl, error) {
	var err error
	mongoDBOnce.Do(func() {
		mongoDBInstance, err = newMongoDBImpl(ctx, retryPolicy, basic, opts...)
	})
	if err != nil {
		return nil, err
	}
	return mongoDBInstance, nil
}

// newMongoDBImpl 使用基本配置和可选的配置函数来创建一个新的 MongoDB 客户端实例
func newMongoDBImpl(ctx context.Context, retryPolicy retry.RetryPolicy, basic *options.ClientOptions,
	opts ...func(*options.ClientOptions),
) (*MongoDBImpl, error) {
	for _, opt := range opts {
		opt(basic)
	}

	var mongoCli *mongo.Client
	var err error

	// 检查重试策略是否允许重试
	if !retryPolicy.ShouldRetry(1, nil) {
		// 如果不允许重试，直接尝试连接
		mongoCli, err = mongo.Connect(ctx, basic)
		if err != nil {
			return nil, fmt.Errorf("failed to connect to MongoDB: %v", err)
		}
	} else {
		// 如果允许重试，进入重试循环
		for attempt := 1; ; attempt++ {
			mongoCli, err = mongo.Connect(ctx, basic)
			if err == nil {
				break
			}
			if !retryPolicy.ShouldRetry(attempt, err) {
				return nil, fmt.Errorf("failed to connect to MongoDB after %d attempts: %v", attempt, err)
			}
			time.Sleep(time.Duration(retryPolicy.WaitDuration(attempt)) * time.Second)
		}
	}

	return &MongoDBImpl{
		client: mongoCli,
	}, nil
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

// GetBatchPool 返回批处理对象池
func (m *MongoDBImpl) GetBatchPool() *pool.BatchPool[pool.BSONBatch] {
	return m.batchPool
}

// GetClient 返回mongo的client
func (m *MongoDBImpl) GetClient() *mongo.Client {
	return m.client
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
	defer cursor.Close(ctx)

	var results []bson.M
	if err := cursor.All(ctx, &results); err != nil {
		return nil, err
	}

	return results, nil
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

// renameFields 重命名文档中的字段
func (m *MongoDBImpl) renameFields(doc bson.M, opt *RenameOption) bson.M {
	if opt == nil || len(opt.Mapping) == 0 {
		return doc
	}

	result := make(bson.M, len(doc))
	for k, v := range doc {
		if newKey, exists := opt.Mapping[k]; exists {
			result[newKey] = v
			if opt.KeepOriginal {
				result[k] = v // 保留原始字段
			}
		} else {
			result[k] = v
		}
	}
	return result
}
