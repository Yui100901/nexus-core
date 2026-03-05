package repository

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"
)

//
// @Author yfy2001
// @Date 2026/3/5 09 53
//

// GetOneByUniqueColumn 根据唯一列查询一条记录
func GetOneByUniqueColumn[T any](ctx context.Context, db *gorm.DB, column string, value any) (*T, error) {
	result, err := gorm.G[T](db).
		Where(fmt.Sprintf("%s = ?", column), value).
		First(ctx)

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &result, nil
}

// DeleteByUniqueColumn 根据唯一列删除记录
func DeleteByUniqueColumn[T any](ctx context.Context, db *gorm.DB, column string, value any) (int, error) {
	rowsAffected, err := gorm.G[T](db).
		Where(fmt.Sprintf("%s = ?", column), value).
		Delete(ctx)
	return rowsAffected, err
}

// FindByColumn 查找符合单列条件的记录（返回多条）
func FindByColumn[T any](ctx context.Context, db *gorm.DB, column string, value any) ([]T, error) {
	results, err := gorm.G[T](db).
		Where(fmt.Sprintf("%s = ?", column), value).
		Find(ctx)
	if err != nil {
		return nil, err
	}
	return results, nil
}

// CountWhere 在给定条件下计数，query 是 SQL 条件字符串（例如 "license_id = ? AND is_bound = ?")
func CountWhere(ctx context.Context, db *gorm.DB, model any, query string, args ...any) (int64, error) {
	var cnt int64
	err := db.WithContext(ctx).
		Model(model).
		Where(query, args...).
		Count(&cnt).Error
	if err != nil {
		return 0, err
	}
	return cnt, nil
}

// UpdateByColumn 通用按列更新记录的函数，updates 可以是 struct 或 map[string]interface{}
func UpdateByColumn[T any](ctx context.Context, db *gorm.DB, column string, value any, updates any) (int, error) {
	rows, err := gorm.G[T](db).
		Where(fmt.Sprintf("%s = ?", column), value).
		Updates(ctx, updates)
	if err != nil {
		return 0, err
	}
	return rows, nil
}

// WithTransaction is a helper to run fn inside a database transaction (begin/commit/rollback)
func WithTransaction(db *gorm.DB, fn func(tx *gorm.DB) error) error {
	if db == nil {
		return fmt.Errorf("db is nil in WithTransaction")
	}
	tx := db.Begin()
	if tx.Error != nil {
		return tx.Error
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			panic(r)
		}
	}()

	if err := fn(tx); err != nil {
		_ = tx.Rollback().Error
		return err
	}
	return tx.Commit().Error
}

// db context propagation helpers
type dbCtxKeyType struct{}

type dbCtxVal struct {
	DB   *gorm.DB
	Tx   *gorm.DB
	InTx bool
}

// ContextWithDB returns a new context with base DB set
func ContextWithDB(ctx context.Context, db *gorm.DB) context.Context {
	return context.WithValue(ctx, dbCtxKeyType{}, &dbCtxVal{DB: db, Tx: nil, InTx: false})
}

// ContextWithTx returns a new context marked as in-transaction with tx set
func ContextWithTx(ctx context.Context, tx *gorm.DB) context.Context {
	return context.WithValue(ctx, dbCtxKeyType{}, &dbCtxVal{DB: nil, Tx: tx, InTx: true})
}

// GetDBFromContext returns tx if present, otherwise base db; nil if none set
func GetDBFromContext(ctx context.Context) *gorm.DB {
	if v := ctx.Value(dbCtxKeyType{}); v != nil {
		if d, ok := v.(*dbCtxVal); ok {
			if d.InTx && d.Tx != nil {
				return d.Tx
			}
			return d.DB
		}
	}
	return nil
}

// IsInTransaction reports whether the context carries an active tx
func IsInTransaction(ctx context.Context) bool {
	if v := ctx.Value(dbCtxKeyType{}); v != nil {
		if d, ok := v.(*dbCtxVal); ok {
			return d.InTx
		}
	}
	return false
}

// WithTransactionCtx starts a transaction using baseDB and passes a new ServiceContext-like context
// to fn where the context holds the tx (so services can fetch tx from ctx). fn receives a context.Context
// For convenience we accept a raw context and return error. Caller typically will use sc.ServiceContext.Context
func WithTransactionCtx(baseDB *gorm.DB, ctx context.Context, fn func(ctx context.Context) error) error {
	if baseDB == nil {
		return fmt.Errorf("baseDB is nil in WithTransactionCtx")
	}
	tx := baseDB.Begin()
	if tx.Error != nil {
		return tx.Error
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			panic(r)
		}
	}()

	ctxWithTx := ContextWithTx(ctx, tx)
	if err := fn(ctxWithTx); err != nil {
		_ = tx.Rollback().Error
		return err
	}
	return tx.Commit().Error
}
