package dto

// CreateNodeCommand 创建节点的命令对象
// @Description Command to create a node
// @Tags Node
type CreateNodeCommand struct {
	DeviceCode string  `json:"device_code" binding:"required"` // 设备唯一识别码
	Metadata   *string `json:"metadata"`                       // 设备元信息
}

type NodeData struct {
	ID         uint    `json:"id"`          // 节点ID
	DeviceCode string  `json:"device_code"` // 设备唯一识别码
	Status     int     `json:"status"`      // 状态（0=正常，1=封禁）
	Metadata   *string `json:"metadata"`    // 设备元信息
}

// AddBindingCommand 添加绑定的命令对象
// @Description Command to add a binding to a node
type AddBindingCommand struct {
	NodeID    uint `json:"node_id" binding:"required"`    // 节点ID
	LicenseID uint `json:"license_id" binding:"required"` // 许可证ID
}

type AutoBindCommand struct {
	DeviceCode string `json:"device_code" binding:"required"`
	LicenseID  uint   `json:"license_id" binding:"required"`
}

// GetNodeByDeviceCodeQuery 根据设备码查询节点的查询对象
type GetNodeByDeviceCodeQuery struct {
	DeviceCode string `form:"device_code" binding:"required"` // 设备码
}

// UnbindCommand 更新绑定状态的命令对象
// @Description Command to update binding status
type UnbindCommand struct {
	NodeID    uint `json:"node_id" binding:"required"`    // 节点ID
	LicenseID uint `json:"license_id" binding:"required"` // 许可证ID
}

// ForceUnbindCommand
// @Description Command to force unbind a node binding using node and license IDs
// @Tags Node
type ForceUnbindCommand struct {
	NodeID    uint `json:"node_id" binding:"required"`    // 节点ID
	LicenseID uint `json:"license_id" binding:"required"` // 许可证ID
}
