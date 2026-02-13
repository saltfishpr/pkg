// Package retry 提供了一个灵活的重试机制，支持多种退避策略。
//
// 基本用法：
//
//	result, err := retry.Do(ctx, func() (string, error) {
//	    return apiCall()
//	})
//
// 配置选项：
//
//	result, err := retry.Do(ctx, f,
//	    retry.WithMaxAttempts(5),
//	    retry.WithRetryStrategy(retry.ExponentialBackoff(100*time.Millisecond, time.Second)),
//	    retry.WithShouldRetryFunc(func(err error) bool {
//	        return isTransientError(err)
//	    }),
//	)
//
// 支持的退避策略：
//   - FixedBackoff: 固定间隔重试
//   - LinearBackoff: 线性增长间隔
//   - ExponentialBackoff: 指数退避，可设置最大间隔
package retry
