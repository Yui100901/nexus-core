package api

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"nexus-core/domain/entity"
	"nexus-core/service"
	"time"
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

type CreateDeviceCommand struct {
	ID            string `json:"id"`            //id
	Name          string `json:"name"`          //名称
	DeviceType    string `json:"deviceType"`    //设备类型
	Model         string `json:"model"`         //设备型号
	Description   string `json:"description"`   //备注
	ValidDuration int    `json:"validDuration"` //有效时长（小时）
}

func (c *CreateDeviceCommand) ToDomain() *entity.Device {
	return &entity.Device{
		ID:          c.ID,
		Name:        c.Name,
		DeviceType:  c.DeviceType,
		Model:       c.Model,
		Description: c.Description,
		Protocol:    "",
		IP:          "",
		Auth: &entity.Auth{
			CreatedAt:     time.Now(),
			ActivatedAt:   nil,
			ValidDuration: c.ValidDuration,
			ExpiredAt:     nil,
			Status:        0,
		},
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
