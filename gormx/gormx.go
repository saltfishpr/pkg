package gormx

import (
	"context"
	"errors"

	"gorm.io/gorm"
)

// BaseRepo 是 Repository 的基础类型,提供通用的仓储操作。
type BaseRepo struct{}

// NewBaseRepo 创建一个新的 BaseRepo 实例。
func NewBaseRepo() *BaseRepo {
	return &BaseRepo{}
}

// IsNotFoundError 判断 err 是否为 gorm.ErrRecordNotFound 错误。
func (r *BaseRepo) IsNotFoundError(err error) bool {
	return errors.Is(err, gorm.ErrRecordNotFound)
}

type ctxKey string

const (
	// CtxKeyMySQLTransaction 是 context 中存储 MySQL 事务的键。
	CtxKeyMySQLTransaction ctxKey = "mysql_transaction"
)

// OnceTransactionRepo 提供基于 context 的事务管理。
// 它确保事务在同一个 context 中只创建一次,实现事务传播。
// 如果 context 中已存在事务,则复用该事务;否则创建新事务。
type OnceTransactionRepo struct {
	db  *gorm.DB
	key ctxKey
}

// NewOnceTransactionRepo 创建一个新的 OnceTransactionRepo 实例。
// db 是底层数据库连接,key 是用于在 context 中存储事务的键。
func NewOnceTransactionRepo(db *gorm.DB, key ctxKey) *OnceTransactionRepo {
	return &OnceTransactionRepo{
		db:  db,
		key: key,
	}
}

// DB 返回一个与 context 绑定的 gorm.DB 实例。
// 如果 context 中已存在事务,返回事务 DB;否则返回普通 DB。
func (r *OnceTransactionRepo) DB(ctx context.Context) *gorm.DB {
	if db, ok := ctx.Value(r.key).(*gorm.DB); ok {
		return db
	}
	return r.db.WithContext(ctx)
}

// Transaction 在事务中执行 fn 函数。
// 如果 context 中已存在事务,则在现有事务中执行;否则创建新事务。
// fn 返回 error 时事务回滚,否则提交。
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

// TransactionResult 在事务中执行 fn 函数并返回结果。
// 如果 context 中已存在事务,则在现有事务中执行;否则创建新事务。
// fn 返回 error 时事务回滚,否则提交。
// 返回 fn 的执行结果和可能的错误。
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
