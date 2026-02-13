# pkg

为微服务和分布式系统设计的可复用 Go 工具包集合。

## Packages

| Package                        | 说明                                                               |
| ------------------------------ | ------------------------------------------------------------------ |
| [bizerrors](./bizerrors)       | 增强的错误处理，包含结构化的业务错误类型、错误码、堆栈跟踪和元数据 |
| [cache](./cache)               | 缓存抽象层，支持获取模式、LRU 实现和自动缓存填充                   |
| [consisthash](./consisthash)   | 一致性哈希实现，用于分布式系统，支持可配置的虚拟节点               |
| [crypto](./crypto)             | AES-GCM 加密/解密工具，具有安全的 nonce 处理机制                   |
| [dag](./dag)                   | 有向无环图执行引擎，支持并发节点执行和拦截器                       |
| [future](./future)             | Promise-Future 异步编程模式                                        |
| [gormx](./gormx)               | GORM 扩展，包含事务和加密字符串支持                                |
| [microservice](./microservice) | 微服务基础设施组件                                                 |
| [retry](./retry)               | 通用重试机制，支持可配置的策略和退避策略                           |
| [routine](./routine)           | Goroutine 工具，支持安全执行、panic 恢复和超时控制                 |

## 安装

```bash
go get github.com/saltfishpr/pkg
```

## 许可证

参见 [LICENSE](./LICENSE) 文件。
