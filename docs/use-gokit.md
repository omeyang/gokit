# 使用文档

## Mongo的正确使用

### 已具备功能
使用官方的`go.mongodb.org/mongo-driver`库 `GetClient`暴漏了原有的Mongo客户端，原库拥有的方法不要重写
新的已经具备方法包括:
- ReadByPage 读取mongo指定集合表中指定分页的数据
- BulkWriteWithRetry 带有重试策略的批量写 重试策略具体查看`util`包下的实现

### 单元测试
**不得使用 ` gomonkey ` 这种的运行时stub行为的工具**(具体参考文档: 为什么要禁用 gomonkey)

gomock 已经在 23年 6 月移交给 `uber-go` 团队维护， 新增代码中不得再使用旧的 `golang/mock` , 必须使用`uber-go/mock`

安装 gomock: ` go install go.uber.org/mock/mockgen@latest `
检查版本: ` mockgen -version `

生成 mock 文件: 在项目根目录下运行 `mockgen -source=middleware/storage/mongo.go -destination=middleware/storage/test/mock_mongo.go -package=test`