package gormx

import (
	"context"
	"errors"

	"gorm.io/gorm"
)

type BaseRepo struct{}

func NewBaseRepo() *BaseRepo {
	return &BaseRepo{}
}

func (r *BaseRepo) IsNotFoundError(err error) bool {
	return errors.Is(err, gorm.ErrRecordNotFound)
}

type ctxKey string

const (
	CtxKeyMySQLTransaction ctxKey = "mysql_transaction"
)

type OnceTransactionRepo struct {
	db  *gorm.DB
	key ctxKey
}

func NewOnceTransactionRepo(db *gorm.DB, key ctxKey) *OnceTransactionRepo {
	return &OnceTransactionRepo{
		db:  db,
		key: key,
	}
}

func (r *OnceTransactionRepo) DB(ctx context.Context) *gorm.DB {
	if db, ok := ctx.Value(r.key).(*gorm.DB); ok {
		return db
	}
	return r.db.WithContext(ctx)
}

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
