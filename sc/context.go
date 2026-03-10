package sc

import (
	"context"
	"fmt"
	"log"
	"nexus-core/persistence/base"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

//
// @Author yfy2001
// @Date 2026/2/27 14 22
//

// ServiceContextKey is the key used to store ServiceContext in gin.Context
const ServiceContextKey = "ServiceContext"

// ServiceContext now holds a DBHelper to support multiple datasources
type ServiceContext struct {
	context.Context
	GinContext *gin.Context
	Metadata   map[string]any
	Logger     *log.Logger
	dbMgr      *DBHelper
}

func NewServiceContext(ctx context.Context, c *gin.Context, metadata map[string]any, logger *log.Logger, mgr *DBHelper) *ServiceContext {
	return &ServiceContext{
		Context:    ctx,
		GinContext: c,
		Metadata:   metadata,
		Logger:     logger,
		dbMgr:      mgr,
	}
}

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

	stdCtx := c.Request.Context()
	//这里获取默认数据库实例并创建DBHelper
	dbHelper := NewDBHelper([]string{base.DefaultDBName})

	return NewServiceContext(stdCtx, c, metaData, logger, dbHelper)
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

func (s *ServiceContext) ensureDBHelper() {
	if s.dbMgr == nil {
		s.dbMgr = NewDBHelper([]string{base.DefaultDBName})
	}
}

func (s *ServiceContext) MustDefaultDB() *gorm.DB {
	s.ensureDBHelper()
	db := s.dbMgr.GetActive("")
	if db == nil {
		db = base.DefaultDBManager.GetDefaultDB()
		s.dbMgr.AddDB("", db)
	}
	return db
}

func (s *ServiceContext) MustPlainDB() *gorm.DB {
	s.ensureDBHelper()
	db := s.dbMgr.GetPlain("")
	if db == nil {
		db = base.DefaultDBManager.GetDefaultDB()
		s.dbMgr.AddDB("", db)
	}
	return db
}

func (s *ServiceContext) GetActiveDB(name string) *gorm.DB {
	s.ensureDBHelper()
	return s.dbMgr.GetActive(name)
}

func (s *ServiceContext) IsInTransaction() bool {
	if s.dbMgr == nil {
		return false
	}
	return s.dbMgr.IsInTx("")
}

// ---------- 公共工具函数 ----------

func (s *ServiceContext) copyDBHelperWithInfo(name string) *DBHelper {
	copiedMgr := &DBHelper{
		defaultName: s.dbMgr.defaultName,
		infos:       make(map[string]*DBInfo),
	}
	s.dbMgr.mu.RLock()
	for k, v := range s.dbMgr.infos {
		copiedMgr.infos[k] = v
	}
	s.dbMgr.mu.RUnlock()
	target := copiedMgr.MustGet(name)
	copied := *target
	copiedMgr.infos[name] = &copied
	return copiedMgr
}

func newSavepointName() string {
	return "sp_" + strings.ReplaceAll(uuid.New().String(), "-", "_")
}

func rollbackOnPanic(tx *gorm.DB, rollbackSQL string) {
	if r := recover(); r != nil {
		_ = tx.Exec(rollbackSQL).Error
		panic(r)
	}
}

// RunInTransaction 在指定数据源上运行事务逻辑
func (s *ServiceContext) RunInTransaction(name string, fn func(txCtx *ServiceContext) error) error {
	s.ensureDBHelper()
	dbToUse := s.dbMgr.GetPlain(name)
	if dbToUse == nil {
		return fmt.Errorf("no base DB available for datasource '%s'", name)
	}

	info := s.dbMgr.MustGet(name)
	if info.InTx && info.Tx != nil {
		// 已在事务中 → 使用 savepoint 模拟嵌套
		txCtx := *s
		copiedMgr := s.copyDBHelperWithInfo(name)
		txCtx.dbMgr = copiedMgr
		tx := info.Tx
		sp := newSavepointName()
		if err := tx.Exec("SAVEPOINT " + sp).Error; err != nil {
			return err
		}
		defer rollbackOnPanic(tx, "ROLLBACK TO SAVEPOINT "+sp)

		if err := fn(&txCtx); err != nil {
			_ = tx.Exec("ROLLBACK TO SAVEPOINT " + sp).Error
			return err
		}
		_ = tx.Exec("RELEASE SAVEPOINT " + sp).Error
		return nil
	}

	// 不在事务中 → 开启新事务
	tx := dbToUse.Begin()
	if tx.Error != nil {
		return tx.Error
	}
	defer rollbackOnPanic(tx, "ROLLBACK")

	txCtx := *s
	copiedMgr := s.copyDBHelperWithInfo(name)
	copiedMgr.infos[name].Tx = tx
	copiedMgr.infos[name].InTx = true
	txCtx.dbMgr = copiedMgr

	if err := fn(&txCtx); err != nil {
		_ = tx.Rollback().Error
		return err
	}
	return tx.Commit().Error
}

// RunInSavepoint 在已有事务中使用 savepoint，或开启新事务
func (s *ServiceContext) RunInSavepoint(name string, fn func(txCtx *ServiceContext) error) error {
	return s.RunInTransaction(name, fn)
}
