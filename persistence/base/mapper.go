package base

import (
	"fmt"
	"gorm.io/gorm"
)

//
// @Author yfy2001
// @Date 2025/7/21 15 32
//

type Mapper[T Model] struct {
	model T
	db    *gorm.DB
}

// SaveOrUpdate 插入或更新
func (m *Mapper[T]) SaveOrUpdate(t *T) (int64, error) {
	result := m.db.Save(t)
	return result.RowsAffected, result.Error
}

// BatchInsert 批量插入
func (m *Mapper[T]) BatchInsert(records []*T) (int64, error) {
	result := m.db.Create(&records)
	return result.RowsAffected, result.Error
}

// Update 更新部分字段
func (m *Mapper[T]) Update(id string, updates map[string]interface{}) (int64, error) {
	var t T
	result := m.db.Model(&t).Where("id = ?", id).Updates(updates)
	return result.RowsAffected, result.Error
}

// GetList 批量查询
func (m *Mapper[T]) GetList() ([]*T, error) {
	var list []*T
	result := m.db.Find(&list)
	return list, result.Error
}

// GetByID 根据id查询
func (m *Mapper[T]) GetByID(id string) (*T, error) {
	var t T
	result := m.db.First(&t, "id = ?", id)
	return &t, result.Error
}

// GetPaginatedList 分页查询
func (m *Mapper[T]) GetPaginatedList(page, pageSize int) ([]*T, error) {
	var list []*T
	offset := (page - 1) * pageSize
	result := m.db.Limit(pageSize).Offset(offset).Find(&list)
	return list, result.Error
}

// GetByCondition 条件查询
func (m *Mapper[T]) GetByCondition(conditions map[string]interface{}) ([]*T, error) {
	var list []*T
	result := m.db.Where(conditions).Find(&list)
	return list, result.Error
}

// GetSortedList 查询并排序
func (m *Mapper[T]) GetSortedList(orderBy string) ([]*T, error) {
	var list []*T
	result := m.db.Order(orderBy).Find(&list)
	return list, result.Error
}

// DeleteByID 根据id删除
func (m *Mapper[T]) DeleteByID(id string) (int64, error) {
	var t T // 创建类型实例
	result := m.db.Delete(&t, "id = ?", id)
	return result.RowsAffected, result.Error
}

// BatchDelete 批量删除
func (m *Mapper[T]) BatchDelete(conditions map[string]interface{}) (int64, error) {
	var t T
	result := m.db.Where(conditions).Delete(&t)
	return result.RowsAffected, result.Error
}

// BatchDeleteByIdList 批量删除使用id列表
func (m *Mapper[T]) BatchDeleteByIdList(idList []any) (int64, error) {
	var t T
	if len(idList) == 0 {
		return 0, fmt.Errorf("idList 不能为空")
	}
	result := m.db.Where("id IN ?", idList).Delete(&t)
	return result.RowsAffected, result.Error
}

// Exists 检查记录是否存在
func (m *Mapper[T]) Exists(conditions map[string]interface{}) (bool, error) {
	var t T
	var count int64
	result := m.db.Model(&t).Where(conditions).Count(&count)
	return count > 0, result.Error
}

// Count 计数
func (m *Mapper[T]) Count() (int64, error) {
	var t T
	var count int64
	result := m.db.Model(&t).Count(&count)
	return count, result.Error
}

// CountByCondition 条件计数
func (m *Mapper[T]) CountByCondition(conditions map[string]interface{}) (int64, error) {
	var t T
	var count int64
	result := m.db.Model(&t).Where(conditions).Count(&count)
	return count, result.Error
}
