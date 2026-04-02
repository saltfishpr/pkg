// Package cache defines a minimal [Cache] interface and a generic [Fetch]
// helper that implements the cache-aside (read-through) pattern.
//
// Fetch first looks up the key in the cache. On a hit it deserializes and
// returns the cached value; on a miss it calls the provided function, caches
// the result, and returns it. Serialization, expiration, and error handling
// are configurable through [FetchOption] functions.
package cache

import (
	"encoding/json"
	"time"

	"github.com/pkg/errors"
)

// Cache is a minimal byte-level key-value cache abstraction.
type Cache interface {
	Set(key string, value []byte) error
	Get(key string) ([]byte, error)
	Del(key string) error
}

// fetchOptions holds the resolved configuration for a single [Fetch] call.
type fetchOptions struct {
	expiration  time.Duration
	marshalFn   func(interface{}) ([]byte, error)
	unmarshalFn func([]byte, interface{}) error
	onSetError  func(key string, err error)
}

// FetchOption configures the behavior of [Fetch].
type FetchOption func(*fetchOptions)

// WithExpiration sets the cache entry TTL. The default is 5 minutes.
func WithExpiration(d time.Duration) FetchOption {
	return func(opts *fetchOptions) {
		opts.expiration = d
	}
}

// WithMarshalFunc overrides the serialization function used before writing
// to the cache. The default is [json.Marshal].
func WithMarshalFunc(marshalFn func(interface{}) ([]byte, error)) FetchOption {
	return func(opts *fetchOptions) {
		opts.marshalFn = marshalFn
	}
}

// WithUnmarshalFunc overrides the deserialization function used when reading
// from the cache. The default is [json.Unmarshal].
func WithUnmarshalFunc(unmarshalFn func([]byte, interface{}) error) FetchOption {
	return func(opts *fetchOptions) {
		opts.unmarshalFn = unmarshalFn
	}
}

// WithSetErrorCallback registers a callback that is invoked when storing
// the computed value in the cache fails. The callback receives the key
// and the error. By default cache-write errors are silently ignored.
func WithSetErrorCallback(fn func(key string, err error)) FetchOption {
	return func(opts *fetchOptions) {
		opts.onSetError = fn
	}
}

// Fetch implements the cache-aside pattern for an arbitrary value type T.
//
// It first attempts to read key from c. On a cache hit the value is
// deserialized and returned. On a miss fn is called to produce the value,
// which is then serialized and stored in c before being returned.
//
// If c is nil, fn is called directly (no caching). Cache-write errors do
// not affect the return value; they are reported via [WithSetErrorCallback]
// if configured.
func Fetch[T any](c Cache, key string, fn func() (T, error), options ...FetchOption) (T, error) {
	if c == nil {
		return fn()
	}

	opts := fetchOptions{
		expiration:  5 * time.Minute,
		marshalFn:   json.Marshal,
		unmarshalFn: json.Unmarshal,
	}
	for _, option := range options {
		option(&opts)
	}

	data, err := c.Get(key)
	if err == nil {
		var v T
		if err := opts.unmarshalFn(data, &v); err == nil {
			return v, nil
		}
	}

	result, err := fn()
	if err != nil {
		return result, err
	}

	setValue := func() error {
		data, err := opts.marshalFn(result)
		if err != nil {
			return errors.WithStack(err)
		}
		if err := c.Set(key, data); err != nil {
			return errors.WithStack(err)
		}
		return nil
	}

	if err := setValue(); err != nil {
		if opts.onSetError != nil {
			opts.onSetError(key, err)
		}
	}

	return result, nil
}
