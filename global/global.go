package global

import (
	"fmt"
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

//
// @Author yfy2001
// @Date 2026/3/26 09 21
//

var (
	DB *gorm.DB
)

func GetLogger(sc *gin.Context) *log.Logger {
	traceID := sc.GetHeader("X-Trace-ID")
	if traceID == "" {
		traceID = uuid.New().String()
	}
	requestID := sc.GetHeader("X-Request-ID")
	if requestID == "" {
		requestID = uuid.New().String()
	}
	method := sc.Request.Method
	path := sc.Request.URL.Path
	prefix := fmt.Sprintf("[TraceID:%s] [RequestID:%s] [%s %s] ", traceID, requestID, method, path)
	logger := log.New(os.Stdout, prefix, log.LstdFlags)
	return logger
}
