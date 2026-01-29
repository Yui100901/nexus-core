package dto

import (
	"nexus-core/domain/entity"
)

// CreateNodeCommand 创建节点的命令对象
// @Description Command to create a node
// @Tags Node
type CreateNodeCommand struct {
	DeviceCode string  `json:"device_code" binding:"required"` // 设备唯一识别码
	MetaInfo   *string `json:"meta_info"`                      // 设备元信息
}

// AddBindingCommand 添加绑定的命令对象
// @Description Command to add a binding to a node
type AddBindingCommand struct {
	NodeID    uint `json:"node_id" binding:"required"`    // 节点ID
	LicenseID uint `json:"license_id" binding:"required"` // 许可证ID
}

// ToEntityNode 将创建节点命令转换为实体对象
func ToEntityNode(cmd CreateNodeCommand) *entity.Node {
	return &entity.Node{
		DeviceCode: cmd.DeviceCode,
		MetaInfo:   cmd.MetaInfo,
	}
}

// ToEntityBinding 将添加绑定命令转换为实体对象
func ToEntityBinding(cmd AddBindingCommand) *entity.NodeBinding {
	return &entity.NodeBinding{
		LicenseID: cmd.LicenseID,
		IsBound:   0,
	}
}

// Query DTOs
// GetNodeByIDQuery 根据ID查询节点的查询对象
type GetNodeByIDQuery struct {
	ID uint `form:"id" binding:"required"` // 节点ID
}

// GetNodeByDeviceCodeQuery 根据设备码查询节点的查询对象
type GetNodeByDeviceCodeQuery struct {
	DeviceCode string `form:"device_code" binding:"required"` // 设备码
}

// UpdateBindingStatusCommand 更新绑定状态的命令对象
// @Description Command to update binding status
type UpdateBindingStatusCommand struct {
	ID     uint `json:"id" binding:"required"`     // 绑定ID
	Status int  `json:"status" binding:"required"` // 新状态
}

// ForceUnbindCommand
// @Description Command to force unbind a node binding using node and license IDs
// @Tags Node
type ForceUnbindCommand struct {
	NodeID    uint `json:"node_id" binding:"required"`    // 节点ID
	LicenseID uint `json:"license_id" binding:"required"` // 许可证ID
}
