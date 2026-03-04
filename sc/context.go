package sc

import (
	"fmt"
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

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

type ServiceContext struct {
	*gin.Context
	traceID   string
	requestID string
	logger    *log.Logger
}

func NewServiceContext(c *gin.Context, traceID, requestID string, logger *log.Logger) *ServiceContext {
	// 从 gin.Context 获取方法和路径

	return &ServiceContext{
		Context:   c,
		traceID:   traceID,
		requestID: requestID,
		logger:    logger,
	}
}

// InitContext 从 gin.Context 初始化 ServiceContext
func InitContext(c *gin.Context) *ServiceContext {
	traceID := c.GetHeader("X-Trace-ID")
	if traceID == "" {
		traceID = uuid.New().String()
	}
	requestID := c.Request.Header.Get("X-Request-ID")
	if requestID == "" {
		requestID = uuid.New().String()
	}
	method := c.Request.Method
	path := c.Request.URL.Path
	prefix := fmt.Sprintf("[TraceID:%s] [RequestID:%s] [%s %s] ", traceID, requestID, method, path)
	logger := log.New(os.Stdout, prefix, log.LstdFlags)

	return NewServiceContext(c, traceID, requestID, logger)
}

func (s *ServiceContext) TraceID() string {
	return s.traceID
}

func (s *ServiceContext) RequestID() string {
	return s.requestID
}

func (s *ServiceContext) Logger() *log.Logger {
	return s.logger
}

func (s *ServiceContext) Error(err error) {
	s.logger.Println(err)
}
