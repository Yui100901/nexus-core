package sc

import (
	"context"
	"fmt"
	"log"
	"nexus-core/persistence/base"
	"os"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ServiceContextKey is the key used to store ServiceContext in gin.Context
const ServiceContextKey = "ServiceContext"

//
// @Author yfy2001
// @Date 2026/2/27 14 22
//

// dbInfo wraps DB/Tx and transaction flag for a single datasource
type dbInfo struct {
	DB   *gorm.DB
	Tx   *gorm.DB
	InTx bool
}

func newDBInfo(db *gorm.DB) *dbInfo {
	return &dbInfo{
		DB:   db,
		Tx:   nil,
		InTx: false,
	}
}

// DBManager holds multiple dbInfo instances keyed by datasource name
type DBManager struct {
	mu          sync.RWMutex
	infos       map[string]*dbInfo
	defaultName string
}

func NewDBManager(defaultDB *gorm.DB) *DBManager {
	if defaultDB == nil {
		defaultDB = base.DefaultDBManager.GetDefaultDB()
	}
	m := &DBManager{
		infos:       make(map[string]*dbInfo),
		defaultName: "default",
	}
	m.infos[m.defaultName] = newDBInfo(defaultDB)
	return m
}

func (m *DBManager) ensureInfo(name string) *dbInfo {
	if name == "" {
		name = m.defaultName
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	info, ok := m.infos[name]
	if !ok || info == nil {
		info = newDBInfo(base.DefaultDBManager.GetDefaultDB())
		m.infos[name] = info
	}
	return info
}

// AddDB registers a datasource under the given name (overwrites if exists)
func (m *DBManager) AddDB(name string, db *gorm.DB) {
	if name == "" {
		name = m.defaultName
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.infos[name] = newDBInfo(db)
}

// GetActive returns the active DB for the given datasource name (tx if in tx else base DB)
func (m *DBManager) GetActive(name string) *gorm.DB {
	info := m.ensureInfo(name)
	if info.InTx && info.Tx != nil {
		return info.Tx
	}
	return info.DB
}

// GetPlain returns the base DB (not tx) for the datasource
func (m *DBManager) GetPlain(name string) *gorm.DB {
	info := m.ensureInfo(name)
	return info.DB
}

// IsInTx reports whether given datasource is in transaction
func (m *DBManager) IsInTx(name string) bool {
	info := m.ensureInfo(name)
	return info.InTx
}

// setTx attaches tx to the dbInfo for given name
func (m *DBManager) setTx(name string, tx *gorm.DB) {
	info := m.ensureInfo(name)
	m.mu.Lock()
	defer m.mu.Unlock()
	info.Tx = tx
	info.InTx = tx != nil
}

// clearTx clears tx on given datasource
func (m *DBManager) clearTx(name string) {
	m.setTx(name, nil)
}

// ServiceContext now holds a DBManager to support multiple datasources
type ServiceContext struct {
	context.Context // 标准库 context
	GinContext      *gin.Context
	Metadata        map[string]any
	Logger          *log.Logger

	// DB/Tx propagation fields (managed at service layer)
	dbMgr *DBManager
}

// NewServiceContext 构造函数
func NewServiceContext(ctx context.Context, c *gin.Context, metadata map[string]any, logger *log.Logger, mgr *DBManager) *ServiceContext {
	return &ServiceContext{
		Context:    ctx,
		GinContext: c,
		Metadata:   metadata,
		Logger:     logger,
		dbMgr:      mgr,
	}
}

// InitContext 从 gin.Context 初始化 ServiceContext
func InitContext(c *gin.Context) *ServiceContext {
	traceID := c.GetHeader("X-Trace-ID")
	if traceID == "" {
		traceID = uuid.New().String()
	}
	requestID := c.GetHeader("X-Request-ID")
	if requestID == "" {
		requestID = uuid.New().String()
	}
	method := c.Request.Method
	path := c.Request.URL.Path
	prefix := fmt.Sprintf("[TraceID:%s] [RequestID:%s] [%s %s] ", traceID, requestID, method, path)
	logger := log.New(os.Stdout, prefix, log.LstdFlags)

	metaData := map[string]any{
		"TraceID":   traceID,
		"RequestID": requestID,
	}

	// 使用标准库 context，优先取 request.Context()
	stdCtx := c.Request.Context()

	// create DBManager with default DB
	dbMgr := NewDBManager(base.DefaultDBManager.GetDefaultDB())

	return NewServiceContext(stdCtx, c, metaData, logger, dbMgr)
}

func (s *ServiceContext) SetMetadata(key string, value any) {
	s.Metadata[key] = value
}

func (s *ServiceContext) GetMetadata(key string) (any, bool) {
	v, ok := s.Metadata[key]
	return v, ok
}

func (s *ServiceContext) DeleteMetadata(key string) {
	delete(s.Metadata, key)
}

func (s *ServiceContext) Error(err error) {
	s.Logger.Println(err)
}

// DB/Transaction helpers on ServiceContext
func (s *ServiceContext) ensureDBMgr() {
	if s.dbMgr == nil {
		s.dbMgr = NewDBManager(base.DefaultDBManager.GetDefaultDB())
	}
}

// MustDefaultDB returns the active DB (tx if in transaction) for default datasource
// Guaranteed to return a non-nil *gorm.DB by creating a default DB if necessary
func (s *ServiceContext) MustDefaultDB() *gorm.DB {
	s.ensureDBMgr()
	db := s.dbMgr.GetActive("")
	if db == nil {
		// ensure non-nil by connecting
		db = base.DefaultDBManager.GetDefaultDB()
		s.dbMgr.AddDB("", db)
	}
	return db
}

// MustPlainDB returns the underlying base DB (not the tx) for the default datasource (or given name - kept for compatibility)
func (s *ServiceContext) MustPlainDB() *gorm.DB {
	s.ensureDBMgr()
	db := s.dbMgr.GetPlain("")
	if db == nil {
		db = base.DefaultDBManager.GetDefaultDB()
		s.dbMgr.AddDB("", db)
	}
	return db
}

// GetActiveDB returns active DB for named datasource (tx if present)
func (s *ServiceContext) GetActiveDB(name string) *gorm.DB {
	s.ensureDBMgr()
	return s.dbMgr.GetActive(name)
}

// IsInTransaction reports whether default datasource is in transaction
func (s *ServiceContext) IsInTransaction() bool {
	if s.dbMgr == nil {
		return false
	}
	return s.dbMgr.IsInTx("")
}

// WithTransaction starts a transaction on default DB, sets Tx/InTx on a copied ServiceContext
// and calls fn with the new ServiceContext; handles commit/rollback and panic.
// If the caller is already in a transaction for the same datasource, a savepoint is used to emulate nested transaction
func (s *ServiceContext) WithTransaction(baseDB *gorm.DB, fn func(txCtx *ServiceContext) error) error {
	return s.WithTransactionFor("", baseDB, fn)
}

// WithTransactionFor starts a transaction for a named datasource (name=="" => default)
func (s *ServiceContext) WithTransactionFor(name string, baseDB *gorm.DB, fn func(txCtx *ServiceContext) error) error {
	s.ensureDBMgr()

	// decide dbToUse: provided baseDB overrides stored base
	var dbToUse *gorm.DB
	if baseDB != nil {
		dbToUse = baseDB
	} else {
		dbToUse = s.dbMgr.GetPlain(name)
	}
	if dbToUse == nil {
		return fmt.Errorf("no base DB available for WithTransaction for datasource '%s'", name)
	}

	// If already in transaction on this datasource and baseDB is nil or equals current base DB, use savepoint to emulate nested transaction
	info := s.dbMgr.ensureInfo(name)
	if info != nil && info.InTx && info.Tx != nil && (baseDB == nil || baseDB == info.DB) {
		// copy context and manager (shallow copy of manager pointer is fine; we'll copy dbInfo)
		txCtx := *s
		// create a shallow copy of DBManager but deep-copy the specific dbInfo to avoid mutating outer info
		copiedMgr := &DBManager{}
		// copy minimal fields
		copiedMgr.defaultName = s.dbMgr.defaultName
		copiedMgr.infos = make(map[string]*dbInfo)
		// copy all existing infos pointers but replace the one for 'name' with a copied value
		s.dbMgr.mu.RLock()
		for k, v := range s.dbMgr.infos {
			copiedMgr.infos[k] = v
		}
		s.dbMgr.mu.RUnlock()
		// deep-copy the targeted info
		copied := *info
		copiedMgr.infos[name] = &copied
		txCtx.dbMgr = copiedMgr

		// reuse existing tx
		tx := info.Tx
		copiedMgr.infos[name].Tx = tx
		copiedMgr.infos[name].InTx = true

		// create unique savepoint name
		sp := "sp_" + strings.ReplaceAll(uuid.New().String(), "-", "_")
		if err := tx.Exec("SAVEPOINT " + sp).Error; err != nil {
			return err
		}

		// ensure we attempt to rollback to savepoint on panic
		defer func() {
			if r := recover(); r != nil {
				_ = tx.Exec("ROLLBACK TO SAVEPOINT " + sp).Error
				panic(r)
			}
		}()

		if err := fn(&txCtx); err != nil {
			// rollback to savepoint to undo inner changes
			_ = tx.Exec("ROLLBACK TO SAVEPOINT " + sp).Error
			return err
		}

		// attempt to release savepoint; ignore release error but log if occurs
		_ = tx.Exec("RELEASE SAVEPOINT " + sp).Error
		return nil
	}

	// Otherwise begin a new transaction on dbToUse
	tx := dbToUse.Begin()
	if tx.Error != nil {
		return tx.Error
	}
	// defer panic handling to rollback the started tx
	defer func() {
		if r := recover(); r != nil {
			_ = tx.Rollback().Error
			panic(r)
		}
	}()

	// copy service context and attach tx (copy DBManager and targeted dbInfo)
	txCtx := *s
	// deep copy DBManager structure to avoid mutating outer context
	copiedMgr := &DBManager{}
	copiedMgr.defaultName = s.dbMgr.defaultName
	copiedMgr.infos = make(map[string]*dbInfo)
	s.dbMgr.mu.RLock()
	for k, v := range s.dbMgr.infos {
		copiedMgr.infos[k] = v
	}
	s.dbMgr.mu.RUnlock()
	// ensure targeted info exists and copy it
	targetInfo := copiedMgr.ensureInfo(name)
	copied := *targetInfo
	copiedMgr.infos[name] = &copied

	// attach the new tx
	copiedMgr.infos[name].Tx = tx
	copiedMgr.infos[name].InTx = true
	txCtx.dbMgr = copiedMgr

	if err := fn(&txCtx); err != nil {
		_ = tx.Rollback().Error
		return err
	}
	// commit and clear tx on original manager as needed
	if err := tx.Commit().Error; err != nil {
		return err
	}
	return nil
}

// WithTransactionUsingDB convenience: use given db and start transaction for default datasource
func (s *ServiceContext) WithTransactionUsingDB(db *gorm.DB, fn func(txCtx *ServiceContext) error) error {
	return s.WithTransaction(db, fn)
}

// WithSavepoint is a convenience wrapper that behaves like WithTransaction but prefers savepoint when already in tx
func (s *ServiceContext) WithSavepoint(baseDB *gorm.DB, fn func(txCtx *ServiceContext) error) error {
	return s.WithSavepointFor("", baseDB, fn)
}

// WithSavepointFor operates on named datasource
func (s *ServiceContext) WithSavepointFor(name string, baseDB *gorm.DB, fn func(txCtx *ServiceContext) error) error {
	s.ensureDBMgr()

	var dbToUse *gorm.DB
	if baseDB != nil {
		dbToUse = baseDB
	} else {
		dbToUse = s.dbMgr.GetPlain(name)
	}
	if dbToUse == nil {
		return fmt.Errorf("no base DB available for WithSavepoint for datasource '%s'", name)
	}

	info := s.dbMgr.ensureInfo(name)
	if info != nil && info.InTx && info.Tx != nil && (baseDB == nil || baseDB == info.DB) {
		// nested: use savepoint similar to WithTransactionFor
		txCtx := *s
		copiedMgr := &DBManager{}
		copiedMgr.defaultName = s.dbMgr.defaultName
		copiedMgr.infos = make(map[string]*dbInfo)
		s.dbMgr.mu.RLock()
		for k, v := range s.dbMgr.infos {
			copiedMgr.infos[k] = v
		}
		s.dbMgr.mu.RUnlock()
		copied := *info
		copiedMgr.infos[name] = &copied
		txCtx.dbMgr = copiedMgr

		tx := info.Tx
		copiedMgr.infos[name].Tx = tx
		copiedMgr.infos[name].InTx = true

		sp := "sp_" + strings.ReplaceAll(uuid.New().String(), "-", "_")
		if err := tx.Exec("SAVEPOINT " + sp).Error; err != nil {
			return err
		}
		defer func() {
			if r := recover(); r != nil {
				_ = tx.Exec("ROLLBACK TO SAVEPOINT " + sp).Error
				panic(r)
			}
		}()

		if err := fn(&txCtx); err != nil {
			_ = tx.Exec("ROLLBACK TO SAVEPOINT " + sp).Error
			return err
		}
		_ = tx.Exec("RELEASE SAVEPOINT " + sp).Error
		return nil
	}

	// Not in transaction -> behave like WithTransactionFor (start a new tx)
	tx := dbToUse.Begin()
	if tx.Error != nil {
		return tx.Error
	}
	defer func() {
		if r := recover(); r != nil {
			_ = tx.Rollback().Error
			panic(r)
		}
	}()

	txCtx := *s
	copiedMgr := &DBManager{}
	copiedMgr.defaultName = s.dbMgr.defaultName
	copiedMgr.infos = make(map[string]*dbInfo)
	s.dbMgr.mu.RLock()
	for k, v := range s.dbMgr.infos {
		copiedMgr.infos[k] = v
	}
	s.dbMgr.mu.RUnlock()
	targetInfo := copiedMgr.ensureInfo(name)
	copied := *targetInfo
	copiedMgr.infos[name] = &copied

	copiedMgr.infos[name].Tx = tx
	copiedMgr.infos[name].InTx = true
	txCtx.dbMgr = copiedMgr

	if err := fn(&txCtx); err != nil {
		_ = tx.Rollback().Error
		return err
	}
	return tx.Commit().Error
}

// WithSavepointUsingDB convenience wrapper
func (s *ServiceContext) WithSavepointUsingDB(db *gorm.DB, fn func(txCtx *ServiceContext) error) error {
	return s.WithSavepoint(db, fn)
}
