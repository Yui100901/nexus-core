package sc

import (
	"context"
	"fmt"
	"log"
	"os"

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

//type CommonContext interface {
//	context.Context
//	TraceID() string
//	RequestID() string
//	DB() *gorm.DB
//	Logger() *log.Logger
//	Error(err error)
//}

// dbInfo wraps DB/Tx and transaction flag for ServiceContext
type dbInfo struct {
	DB   *gorm.DB
	Tx   *gorm.DB
	InTx bool
}

type ServiceContext struct {
	context.Context // 标准库 context
	GinContext      *gin.Context
	Metadata        map[string]any
	Logger          *log.Logger

	// DB/Tx propagation fields (managed at service layer)
	db *dbInfo
}

// NewServiceContext 构造函数
func NewServiceContext(ctx context.Context, c *gin.Context, metadata map[string]any, logger *log.Logger) *ServiceContext {
	return &ServiceContext{
		Context:    ctx,
		GinContext: c,
		Metadata:   metadata,
		Logger:     logger,
		db:         &dbInfo{},
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

	return NewServiceContext(stdCtx, c, metaData, logger)
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
func (s *ServiceContext) ensureDB() {
	if s.db == nil {
		s.db = &dbInfo{}
	}
}

func (s *ServiceContext) SetDB(db *gorm.DB) {
	s.ensureDB()
	s.db.DB = db
}

func (s *ServiceContext) SetTx(tx *gorm.DB) {
	s.ensureDB()
	s.db.Tx = tx
	if tx != nil {
		s.db.InTx = true
	} else {
		s.db.InTx = false
	}
}

// DefaultDB returns tx if in transaction else base db (convenience for use in services)
func (s *ServiceContext) DefaultDB() *gorm.DB {
	if s.db == nil {
		return nil
	}
	if s.db.InTx && s.db.Tx != nil {
		return s.db.Tx
	}
	return s.db.DB
}

// PlainDB returns the underlying base DB (not the tx). May be nil.
func (s *ServiceContext) PlainDB() *gorm.DB {
	if s.db == nil {
		return nil
	}
	return s.db.DB
}

// GetDB kept for backward compatibility: alias for DefaultDB
func (s *ServiceContext) GetDB() *gorm.DB {
	return s.DefaultDB()
}

func (s *ServiceContext) IsInTransaction() bool {
	if s.db == nil {
		return false
	}
	return s.db.InTx
}

// WithTransaction starts a transaction on baseDB (or s.PlainDB() if baseDB is nil), sets Tx/InTx on a copied ServiceContext
// and calls fn with the new ServiceContext; handles commit/rollback and panic.
func (s *ServiceContext) WithTransaction(baseDB *gorm.DB, fn func(txCtx *ServiceContext) error) error {
	var dbToUse *gorm.DB
	if baseDB != nil {
		dbToUse = baseDB
	} else if s.db != nil {
		dbToUse = s.db.DB
	}
	if dbToUse == nil {
		return fmt.Errorf("no base DB available for WithTransaction")
	}

	tx := dbToUse.Begin()
	if tx.Error != nil {
		return tx.Error
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			panic(r)
		}
	}()

	// copy service context and attach tx
	txCtx := *s
	txCtx.ensureDB()
	txCtx.db.Tx = tx
	txCtx.db.InTx = true

	if err := fn(&txCtx); err != nil {
		_ = tx.Rollback().Error
		return err
	}
	return tx.Commit().Error
}

// WithTransactionUsingDB convenience: use given db and start transaction, map result back to s if needed
func (s *ServiceContext) WithTransactionUsingDB(db *gorm.DB, fn func(txCtx *ServiceContext) error) error {
	return s.WithTransaction(db, fn)
}
