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

func newDBInfo() *dbInfo {
	return &dbInfo{
		DB:   base.Connect(),
		Tx:   nil,
		InTx: false,
	}
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
func NewServiceContext(ctx context.Context, c *gin.Context, metadata map[string]any, logger *log.Logger, info *dbInfo) *ServiceContext {
	return &ServiceContext{
		Context:    ctx,
		GinContext: c,
		Metadata:   metadata,
		Logger:     logger,
		db:         info,
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

	return NewServiceContext(stdCtx, c, metaData, logger, newDBInfo())
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
		s.db = newDBInfo()
	}
}

// MustDefaultDB returns tx if in transaction else base db (convenience for use in services)
func (s *ServiceContext) MustDefaultDB() *gorm.DB {
	s.ensureDB()
	if s.db.InTx && s.db.Tx != nil {
		return s.db.Tx
	}
	return s.db.DB
}

// MustPlainDB returns the underlying base DB (not the tx). May be nil.
func (s *ServiceContext) MustPlainDB() *gorm.DB {
	s.ensureDB()
	return s.db.DB
}

func (s *ServiceContext) IsInTransaction() bool {
	if s.db == nil {
		return false
	}
	return s.db.InTx
}

// WithTransaction starts a transaction on baseDB (or s.MustPlainDB() if baseDB is nil), sets Tx/InTx on a copied ServiceContext
// and calls fn with the new ServiceContext; handles commit/rollback and panic.
// If the caller is already in a transaction and baseDB is nil (or equals the caller's base DB), the existing transaction will be reused
// and no new begin/commit/rollback will be performed by this function. In that case the outer caller manages commit/rollback.
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

	// If already in transaction and baseDB is nil or equals current base DB, use a savepoint to emulate nested transaction
	if s.db != nil && s.db.InTx && s.db.Tx != nil && (baseDB == nil || baseDB == s.db.DB) {
		// copy context and dbInfo
		txCtx := *s
		if s.db != nil {
			copied := *s.db
			txCtx.db = &copied
		} else {
			txCtx.ensureDB()
		}
		// reuse existing tx
		tx := s.db.Tx
		txCtx.db.Tx = tx
		txCtx.db.InTx = true

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

	// copy service context and attach tx (copy dbInfo to avoid mutating outer context)
	txCtx := *s
	if s.db != nil {
		copied := *s.db
		txCtx.db = &copied
	} else {
		txCtx.ensureDB()
	}
	// attach the new tx
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

// WithSavepoint starts a savepoint on the current transaction when available and executes fn within that context.
// Behavior:
// - If the caller is already in a transaction and baseDB is nil (or equals current base DB), create a savepoint on the existing tx:
//   - On fn error -> ROLLBACK TO SAVEPOINT
//   - On fn success -> RELEASE SAVEPOINT
//
// - If no transaction is active, begin a new transaction (Begin/Commit/Rollback) similar to WithTransaction.
func (s *ServiceContext) WithSavepoint(baseDB *gorm.DB, fn func(txCtx *ServiceContext) error) error {
	var dbToUse *gorm.DB
	if baseDB != nil {
		dbToUse = baseDB
	} else if s.db != nil {
		dbToUse = s.db.DB
	}
	if dbToUse == nil {
		return fmt.Errorf("no base DB available for WithSavepoint")
	}

	// If already in transaction and baseDB is nil or equals current base DB, use savepoint
	if s.db != nil && s.db.InTx && s.db.Tx != nil && (baseDB == nil || baseDB == s.db.DB) {
		// copy context and dbInfo
		txCtx := *s
		if s.db != nil {
			copied := *s.db
			txCtx.db = &copied
		} else {
			txCtx.ensureDB()
		}
		// reuse existing tx
		tx := s.db.Tx
		txCtx.db.Tx = tx
		txCtx.db.InTx = true

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

	// Not in a transaction -> behave like WithTransaction (start a new tx)
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

	// copy service context and attach tx
	txCtx := *s
	if s.db != nil {
		copied := *s.db
		txCtx.db = &copied
	} else {
		txCtx.ensureDB()
	}
	txCtx.db.Tx = tx
	txCtx.db.InTx = true

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
