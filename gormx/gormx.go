// Package gormx extends the GORM ORM framework with:
//   - Transparent field-level encryption via [SecureString].
//   - Context-propagated transaction management via [OnceTransactionRepo].
//   - A minimal [BaseRepo] helper for common repository operations.
package gormx

import (
	"context"

	"github.com/pkg/errors"
	"gorm.io/gorm"
)

// BaseRepo provides convenience helpers common to all repositories.
type BaseRepo struct{}

// NewBaseRepo creates a new [BaseRepo].
func NewBaseRepo() *BaseRepo {
	return &BaseRepo{}
}

// IsNotFoundError reports whether err is [gorm.ErrRecordNotFound].
func (r *BaseRepo) IsNotFoundError(err error) bool {
	return errors.Is(err, gorm.ErrRecordNotFound)
}

// TransactionKey identifies a transaction scope in a context. Repositories
// with the same key share transactions.
type TransactionKey string

// DefaultTransactionKey is the conventional key for the default scope.
const DefaultTransactionKey TransactionKey = "default"

// OnceTransactionRepo propagates a transaction through context. Nested calls
// to [OnceTransactionRepo.Transaction] reuse the transaction already stored in
// ctx under the repository's key. Repositories with the same key share
// transactions; use different keys for independent scopes or databases.
type OnceTransactionRepo struct {
	db  *gorm.DB
	key TransactionKey
}

// NewOnceTransactionRepo creates a repository that stores its transaction
// under key. db is the underlying connection.
func NewOnceTransactionRepo(db *gorm.DB, key TransactionKey) *OnceTransactionRepo {
	return &OnceTransactionRepo{
		db:  db,
		key: key,
	}
}

// DB returns a session bound to ctx. If ctx carries a transaction, the session
// runs inside it.
func (r *OnceTransactionRepo) DB(ctx context.Context) *gorm.DB {
	if db, ok := ctx.Value(r.key).(*gorm.DB); ok {
		return db.WithContext(ctx)
	}
	return r.db.WithContext(ctx)
}

// Transaction runs fn in a transaction. If ctx already carries one, fn runs in
// that transaction; otherwise a new transaction is started. The transaction
// commits when fn returns nil and rolls back otherwise. Use
// [OnceTransactionRepo.DB] inside fn to execute queries in the transaction.
// Transaction panics if fn is nil.
func (r *OnceTransactionRepo) Transaction(ctx context.Context, fn func(ctx context.Context) error) error {
	if fn == nil {
		panic("gormx: nil transaction function")
	}

	_, err := TransactionResult(r, ctx, func(ctx context.Context) (struct{}, error) {
		return struct{}{}, fn(ctx)
	})
	return err
}

// TransactionResult is like [OnceTransactionRepo.Transaction] but returns a
// typed result. On error it returns the zero value of T. It panics if fn is
// nil.
func TransactionResult[T any](
	r *OnceTransactionRepo,
	ctx context.Context,
	fn func(ctx context.Context) (T, error),
) (T, error) {
	if fn == nil {
		panic("gormx: nil transaction function")
	}

	if _, ok := ctx.Value(r.key).(*gorm.DB); ok {
		return fn(ctx)
	}

	var res T
	err := r.db.WithContext(ctx).Transaction(func(db *gorm.DB) error {
		var err error
		res, err = fn(context.WithValue(ctx, r.key, db))
		return err
	})
	if err != nil {
		var zero T
		return zero, err
	}
	return res, nil
}
