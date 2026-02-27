package ctx

import (
	"fmt"
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

//
// @Author yfy2001
// @Date 2026/2/27 14 22
//

type CommonContext interface {
	TraceID() string
	DB() *gorm.DB
	Logger() *log.Logger
}

type ServiceContext struct {
	*gin.Context
	traceID string
	db      *gorm.DB
	logger  *log.Logger
}

func NewServiceContext(c *gin.Context, traceID string, db *gorm.DB) *ServiceContext {
	// 从 gin.Context 获取方法和路径
	method := c.Request.Method
	path := c.Request.URL.Path

	// 日志前缀包含 TraceID、方法、路径
	prefix := fmt.Sprintf("[TraceID:%s] [%s %s] ", traceID, method, path)

	return &ServiceContext{
		Context: c,
		traceID: traceID,
		db:      db,
		logger:  log.New(os.Stdout, prefix, log.LstdFlags),
	}
}

func (s *ServiceContext) TraceID() string {
	return s.traceID
}

func (s *ServiceContext) DB() *gorm.DB {
	return s.db
}

func (s *ServiceContext) Logger() *log.Logger {
	return s.logger
}
