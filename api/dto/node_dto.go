package dto

type CreateNodeCommand struct {
	DeviceCode string  `json:"device_code" binding:"required"`
	Metadata   *string `json:"metadata"`
}

type UpdateNodeCommand struct {
	ID         uint    `json:"id"`
	DeviceCode *string `json:"device_code"`
	Metadata   *string `json:"metadata"`
}

type NodeData struct {
	ID         uint    `json:"id"`
	DeviceCode string  `json:"device_code"`
	Status     int     `json:"status"`
	Metadata   *string `json:"metadata"`
}

type AddBindingCommand struct {
	NodeID    uint `json:"node_id" binding:"required"`
	LicenseID uint `json:"license_id" binding:"required"`
}

type GetNodeByDeviceCodeQuery struct {
	DeviceCode string `form:"device_code" binding:"required"`
}

type UnbindCommand struct {
	NodeID    uint `json:"node_id" binding:"required"`
	LicenseID uint `json:"license_id" binding:"required"`
}

type UpdateNodeStatusCommand struct {
	NodeID uint `json:"node_id" binding:"required"`
}

type ForceUnbindCommand struct {
	NodeID    uint `json:"node_id" binding:"required"`
	LicenseID uint `json:"license_id" binding:"required"`
}
