package ctx

import (
	"log"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

//
// @Author yfy2001
// @Date 2026/2/27 14 22
//

type CommonContext interface {
	TraceID() string
	RequestID() string
	DB() *gorm.DB
	Logger() *log.Logger
}

type ServiceContext struct {
	*gin.Context
	traceID   string
	requestID string
	db        *gorm.DB
	logger    *log.Logger
}

func NewServiceContext(c *gin.Context, traceID, requestID string, db *gorm.DB, logger *log.Logger) *ServiceContext {
	// 从 gin.Context 获取方法和路径

	return &ServiceContext{
		Context:   c,
		traceID:   traceID,
		requestID: requestID,
		db:        db,
		logger:    logger,
	}
}

func (s *ServiceContext) TraceID() string {
	return s.traceID
}

func (s *ServiceContext) RequestID() string {
	return s.requestID
}

func (s *ServiceContext) DB() *gorm.DB {
	return s.db
}

func (s *ServiceContext) Logger() *log.Logger {
	return s.logger
}
