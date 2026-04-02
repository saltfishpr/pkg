// Package gormx extends the GORM ORM framework with:
//   - Transparent field-level encryption via [SecureString].
//   - Context-propagated transaction management via [OnceTransactionRepo].
//   - A minimal [BaseRepo] helper for common repository operations.
package gormx

import (
	"context"
	"errors"

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

// ctxKey is a private type used as context-value keys to avoid collisions.
type ctxKey string

// CtxKeyMySQLTransaction is the default context key for storing a MySQL
// transaction. Pass a custom key to [NewOnceTransactionRepo] if you need
// multiple independent transaction scopes.
const CtxKeyMySQLTransaction ctxKey = "mysql_transaction"

// OnceTransactionRepo implements context-based transaction propagation.
//
// When [OnceTransactionRepo.Transaction] is called, it checks whether the
// context already carries a transaction (stored under key). If so, fn runs
// inside the existing transaction; otherwise a new transaction is started.
// This lets nested service calls share a single database transaction without
// explicit plumbing.
type OnceTransactionRepo struct {
	db  *gorm.DB
	key ctxKey
}

// NewOnceTransactionRepo creates a new [OnceTransactionRepo].
// db is the underlying database connection and key is the context key under
// which the active transaction is stored.
func NewOnceTransactionRepo(db *gorm.DB, key ctxKey) *OnceTransactionRepo {
	return &OnceTransactionRepo{
		db:  db,
		key: key,
	}
}

// DB returns a [gorm.DB] bound to ctx. If ctx carries an active transaction
// it is reused; otherwise a plain session is returned.
func (r *OnceTransactionRepo) DB(ctx context.Context) *gorm.DB {
	if db, ok := ctx.Value(r.key).(*gorm.DB); ok {
		return db
	}
	return r.db.WithContext(ctx)
}

// Transaction executes fn inside a database transaction. If a transaction
// is already present in ctx it is reused (nested call); otherwise a new one
// is created. The transaction commits when fn returns nil, or rolls back on
// error.
func (r *OnceTransactionRepo) Transaction(ctx context.Context, fn func(ctx context.Context) error) error {
	if fn == nil {
		panic("fn cannot be nil")
	}

	if _, ok := ctx.Value(r.key).(*gorm.DB); ok {
		return fn(ctx)
	}

	return r.db.Transaction(func(db *gorm.DB) error {
		tx := db.WithContext(ctx)
		ctx := context.WithValue(ctx, r.key, tx)
		return fn(ctx)
	})
}

// TransactionResult is like [OnceTransactionRepo.Transaction] but also
// returns a result value from fn.
func (r *OnceTransactionRepo) TransactionResult(ctx context.Context, fn func(ctx context.Context) (any, error)) (any, error) {
	if fn == nil {
		panic("fn cannot be nil")
	}

	if _, ok := ctx.Value(r.key).(*gorm.DB); ok {
		return fn(ctx)
	}

	var res any
	err := r.db.Transaction(func(db *gorm.DB) error {
		tx := db.WithContext(ctx)
		ctx := context.WithValue(ctx, r.key, tx)

		var err error
		res, err = fn(ctx)
		return err
	})
	return res, err
}
