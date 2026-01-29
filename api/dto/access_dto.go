package dto

// HeartbeatCommand 客户端心跳命令对象
// @Description Heartbeat payload from client containing device and license info
type HeartbeatCommand struct {
	DeviceCode  string `json:"device_code" binding:"required"`  // 设备唯一识别码
	LicenseKey  string `json:"license_key" binding:"required"`  // 许可证
	ProductID   uint   `json:"product_id" binding:"required"`   // 产品ID
	VersionCode string `json:"version_code" binding:"required"` // 产品版本号
}
