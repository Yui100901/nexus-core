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
