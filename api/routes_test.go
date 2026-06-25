package api

import (
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRegisterDefaultRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	WebEngine = NewServer()

	RegisterDefaultRoutes()
}
