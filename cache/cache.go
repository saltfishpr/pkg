package cache

import (
	"encoding/json"
	"time"

	"github.com/pkg/errors"
)

type Cache interface {
	Set(key string, value []byte) error
	Get(key string) ([]byte, error)
	Del(key string) error
}

type fetchOptions struct {
	expiration  time.Duration
	marshalFn   func(interface{}) ([]byte, error)
	unmarshalFn func([]byte, interface{}) error
	onSetError  func(key string, err error)
}

type FetchOption func(*fetchOptions)

func WithExpiration(d time.Duration) FetchOption {
	return func(opts *fetchOptions) {
		opts.expiration = d
	}
}

func WithMarshalFunc(marshalFn func(interface{}) ([]byte, error)) FetchOption {
	return func(opts *fetchOptions) {
		opts.marshalFn = marshalFn
	}
}

func WithUnmarshalFunc(unmarshalFn func([]byte, interface{}) error) FetchOption {
	return func(opts *fetchOptions) {
		opts.unmarshalFn = unmarshalFn
	}
}

func WithSetErrorCallback(fn func(key string, err error)) FetchOption {
	return func(opts *fetchOptions) {
		opts.onSetError = fn
	}
}

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
