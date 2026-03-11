package sc

import (
	"fmt"
	"nexus-core/persistence/base"
	"sync"

	"gorm.io/gorm"
)

//
// @Author yfy2001
// @Date 2026/3/10 15 23
//

// DBInfo wraps DB/Tx and transaction flag for a single datasource
type DBInfo struct {
	DB   *gorm.DB
	Tx   *gorm.DB
	InTx bool
}

func NewDBInfo(db *gorm.DB) *DBInfo {
	return &DBInfo{
		DB:   db,
		Tx:   nil,
		InTx: false,
	}
}

// DBHelper holds multiple dbInfo instances keyed by datasource name
type DBHelper struct {
	mu          sync.RWMutex
	infos       map[string]*DBInfo
	defaultName string //默认数据源
}

func NewDBHelper(dbNameList []string) *DBHelper {
	m := &DBHelper{
		infos:       make(map[string]*DBInfo),
		defaultName: base.DefaultDBName,
	}

	// 如果没有传入任何数据源名，就至少注册默认数据源
	if len(dbNameList) == 0 {
		defaultDB := base.MainDBManager.GetDefaultDB()
		m.infos[m.defaultName] = NewDBInfo(defaultDB)
		return m
	}

	// 遍历传入的数据源名，逐一注册
	for _, name := range dbNameList {
		db := base.MainDBManager.GetDB(name)
		if db == nil {
			panic(fmt.Sprintf("no base DB available for datasource '%s'", name))
		}
		m.infos[name] = NewDBInfo(db)
	}

	// 确保默认数据源一定存在
	if _, ok := m.infos[m.defaultName]; !ok {
		m.infos[m.defaultName] = NewDBInfo(base.MainDBManager.GetDefaultDB())
	}

	return m
}

func (m *DBHelper) MustGet(name string) *DBInfo {
	if name == "" {
		name = m.defaultName
	}
	// fast read path using RLock
	m.mu.RLock()
	info, ok := m.infos[name]
	m.mu.RUnlock()
	if ok && info != nil {
		return info
	}

	// if requested name is the default, try to create it under write lock
	if name == m.defaultName {
		m.mu.Lock()
		defer m.mu.Unlock()
		// double-check in write lock
		info, ok = m.infos[name]
		if !ok || info == nil {
			info = NewDBInfo(base.MainDBManager.GetDefaultDB())
			m.infos[name] = info
		}
		return info
	}

	// non-default datasource missing -> panic as before
	panic(fmt.Sprintf("no base DB available for datasource %s", name))
}

func (m *DBHelper) AddDB(name string, db *gorm.DB) {
	if name == "" {
		name = m.defaultName
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.infos[name] = NewDBInfo(db)
}

func (m *DBHelper) GetActive(name string) *gorm.DB {
	info := m.MustGet(name)
	if info.InTx && info.Tx != nil {
		return info.Tx
	}
	return info.DB
}

func (m *DBHelper) GetPlain(name string) *gorm.DB {
	info := m.MustGet(name)
	return info.DB
}

func (m *DBHelper) IsInTx(name string) bool {
	info := m.MustGet(name)
	return info.InTx
}

func (m *DBHelper) setTx(name string, tx *gorm.DB) {
	info := m.MustGet(name)
	m.mu.Lock()
	defer m.mu.Unlock()
	info.Tx = tx
	info.InTx = tx != nil
}

func (m *DBHelper) clearTx(name string) {
	m.setTx(name, nil)
}
