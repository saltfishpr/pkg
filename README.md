# pkg

A collection of reusable Go utility packages.

## Packages

| Package                      | Description                                                                                     |
| ---------------------------- | ----------------------------------------------------------------------------------------------- |
| [bizerrors](./bizerrors)     | Enhanced error handling with structured business error types, codes, stack traces, and metadata |
| [cache](./cache)             | Cache abstraction layer with fetch pattern, LRU implementation, and automatic cache-fill        |
| [consisthash](./consisthash) | Consistent hashing implementation for distributed systems with configurable virtual nodes       |
| [crypto](./crypto)           | AES-GCM encryption/decryption utilities with secure nonce handling                              |
| [future](./future)           | Promise-Future async primitives with executors, combinators, and Go 1.27 generic methods        |
| [gormx](./gormx)             | GORM extensions including transactions and encrypted string support                             |
| [retry](./retry)             | General-purpose retry mechanism with configurable strategies and backoff                        |
| [routine](./routine)         | Goroutine utilities for safe execution with panic recovery and timeout support                  |

## Installation

```bash
go get github.com/saltfishpr/pkg
```

## License

See [LICENSE](./LICENSE) file.
