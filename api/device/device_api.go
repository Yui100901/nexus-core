package device

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"nexus-core/service"
)

//
// @Author yfy2001
// @Date 2025/7/22 09 57
//

type deviceController struct {
	ds *service.DeviceService
}

func (dc *deviceController) RouteRegister(r *gin.RouterGroup) {
	users := r.Group("/device")
	{
		users.GET("/create", dc.create)
	}
}

func (dc *deviceController) create(ctx *gin.Context) {

	var cmd CreateDeviceCommand
	if err := ctx.ShouldBindJSON(&cmd); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	}
	e := cmd.ToDomain()
	dc.ds.CreateDevice(e)
}

func (dc *deviceController) createWithAuth(c *gin.Context) {

}
