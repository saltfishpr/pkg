# AGENTS.md

## 项目概述

这是一个 Go 语言的通用工具包集合，包含多个独立的实用模块，适用于构建高性能、可扩展的微服务应用。

## 发布

项目使用 release-please 自动管理发布。当向 master 分支推送包含特定 Conventional Commits 格式的提交时，会自动创建 release PR：

- `feat:`: 新功能
- `fix:`: bug 修复
- `chore:`: 其他更改

合并 release PR 后会自动创建 GitHub Release 并生成 tag。

## 核心模块架构

### future: Promise/Future 模式实现

这是一个**无锁**、高性能的 Promise-Future 模式实现。

**核心设计：**
- 使用 `atomic.Uint32` 和 `unsafe.Pointer` 实现无锁状态管理
- 状态机：Free → Doing → Done
- 使用 lock-free stack 管理回调函数
- 支持泛型 `Future[T]`

**关键组件：**
- `Promise[T]`：生产者端，用于设置结果
- `Future[T]`：消费者端，用于获取结果或订阅回调
- `Executor` 接口：可自定义任务执行器（默认使用 goroutine）

**使用场景：**
```go
// 创建 Promise-Future 对
promise := future.NewPromise[string]()
fut := promise.Future()

// 异步设置值
promise.Set("hello", nil)

// 获取结果（阻塞）
val, err := fut.Get()

// 或订阅回调
fut.Subscribe(func(val string, err error) {
    // 处理结果
})
```

**组合函数：**
- `Async()` / `Submit()`：异步执行函数返回 Future
- `Then()`：链式调用
- `AllOf()`：等待多个 Future 全部完成
- `WithContext()`：关联 Context 取消
- `Timeout()`：超时控制

**注意事项：**
- Promise 只能设置一次（SetSafety 可安全设置）
- Subscribe 回调在设置 Future 状态的 goroutine 中执行，不应包含阻塞操作
- 可通过 `SetExecutor()` 自定义执行器（如使用 goroutine 池）

### dag: 有向无环图执行引擎

支持并发执行、子图嵌套和拦截器的 DAG 实现。

**使用流程：**
1. 创建 DAG：`NewDAG(entryNodeID)`
2. 添加节点：`AddNode(id, deps, fn)` 或 `AddSubGraph()`
3. 冻结 DAG：`Freeze()` - 验证完整性和循环检测
4. 创建实例：`Instantiate(input, options...)`
5. 运行实例：`Run(ctx)` 或 `RunAsync(ctx)`

**核心特性：**
- **并发执行**：无依赖关系的节点自动并发执行
- **子图嵌套**：通过 `AddSubGraph()` 支持 DAG 模块化组合
- **拦截器**：`WithNodeFuncInterceptor()` 可添加日志、监控等横切关注点
- **预设结果**：`WithNodeResults()` 可预设节点结果（用于缓存/短路）
- **可视化**：`ToMermaid()` 生成 Mermaid 图表

**节点类型：**
- `EntryNode`：入口节点（无依赖）
- `SimpleNode`：普通执行节点
- `SubDAGNode`：子图节点

**注意事项：**
- 必须调用 `Freeeze()` 后才能 `Instantiate()`
- 子图必须先 `Freeeze()` 才能被添加到父 DAG
- 节点执行失败会中断整个 DAG
- `ErrNodeSkipped` 表示节点被跳过（依赖失败）

### bizerrors: 业务错误处理

带错误码、堆栈跟踪和详情的结构化错误类型。

**核心功能：**
- 错误码和消息：`New(code, message)`
- 堆栈跟踪：自动在创建点捕获堆栈
- 链式包装：`WithCause()` / `WithMessage()` / `WithDetails()`
- 格式化输出：支持 `%+v` 打印完整堆栈

**设计要点：**
- `WithCause()` 包装时，如果 cause 已经是 `*Error`，会直接返回（避免重复包装）
- 使用 `errors.As()` 检查错误类型
- `FromError()` 从标准 error 提取 `*Error`

### retry: 重试机制

支持自定义退避策略和重试条件的泛型重试库。

**退避策略（`RetryStrategy`）：**
- `FixedBackoff(duration)`：固定间隔
- `LinearBackoff(duration)`：线性增长
- `ExponentialBackoff(duration, max)`：指数退避

**配置选项：**
- `WithMaxAttempts(n)`：最大重试次数
- `WithRetryStrategy(strategy)`：退避策略
- `WithShouldRetryFunc(fn)`：自定义是否重试

**注意事项：**
- 支持Context取消
- 默认重试 3 次，固定间隔 100ms
- 最后一次尝试失败后不再等待

### cache: 缓存抽象

泛型缓存获取工具，支持缓存穿透保护。

**核心函数：**
```go
cache.Fetch[MyType](cache, "key", func() (MyType, error) {
    // 缓存未命中时的数据加载逻辑
    return loadData()
}, cache.WithExpiration(5*time.Minute))
```

**设计：**
- 默认使用 JSON 序列化
- 可自定义 marshal/unmarshal 函数
- Cache 接口：`Set()` / `Get()` / `Del()`
- 设置失败时不影响结果返回（TODO: 需要更好的错误处理）

### consisthash: 一致性哈希

线程安全的一致性哈希环实现。

**核心特性：**
- 泛型支持：`Ring[T]` 可存储任意类型节点
- 虚拟节点：减少数据倾斜
- 线程安全：使用 `sync.RWMutex` 保护
- 可定制：自定义哈希函数和 Key 函数

**使用示例：**
```go
ring := consisthash.NewRing(100, func(node string) string {
    return node
})
ring.Add("node1", "node2", "node3")
node, ok := ring.Get("some-key")
```

### routine: Goroutine 安全工具

- `RunSafe()`：带 panic 恢复的同步执行
- `GoSafe()`：带 panic 恢复的异步执行
- `RunWithTimeout()`：带超时的执行

### gormx: GORM 扩展

- `BaseRepo`：基础 Repository，提供 `IsNotFoundError()`
- `OnceTransactionRepo`：确保事务只执行一次的 Repository
  - `DB(ctx)`：获取数据库连接（自动检测事务）
  - `Transaction(ctx, fn)`：执行事务
  - `TransactionResult(ctx, fn)`：执行事务并返回值

**事务设计：**
使用 context 传递事务状态，支持嵌套调用时复用外层事务。

### 其他模块

- `crypto`：加密工具
- `daemon`：守护进程
- `i18n`：国际化
- `logs`：日志
- `microservice`：微服务相关（discovery, shard）
- `ptr`：指针工具
