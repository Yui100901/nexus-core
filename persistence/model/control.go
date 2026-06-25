package model

import (
	"time"

	"gorm.io/datatypes"
)

const (
	ControlServiceTypeCommand = "command"
	ControlServiceTypeConfig  = "config"
	ControlServiceTypeQuery   = "query"
	ControlServiceTypeAction  = "action"
)

// ControlService 定义服务端可下发给节点的控制服务。
type ControlService struct {
	BaseModel
	ProductID    *uint          `gorm:"index"`                                  // 为空表示通用服务
	Identifier   string         `gorm:"uniqueIndex;type:varchar(100);not null"` // 服务唯一标识
	Name         string         `gorm:"type:varchar(100);not null"`             // 服务名称
	Description  *string        `gorm:"type:text"`                              // 服务描述
	ServiceType  string         `gorm:"type:varchar(50);index;not null"`        // command/config/query/action
	InputSchema  datatypes.JSON `gorm:"type:json"`                              // 标准输入 Schema
	OutputSchema datatypes.JSON `gorm:"type:json"`                              // 标准输出 Schema
	Status       int            `gorm:"type:int;index;not null;default:1"`      // 1启用，2禁用
}

func (ControlService) TableName() string {
	return "control_service"
}

// NodeServiceCapability 记录某个节点实际支持的控制服务及其节点侧 Schema。
type NodeServiceCapability struct {
	BaseModel
	NodeID            uint           `gorm:"uniqueIndex:idx_node_service_capability;index;not null"`
	ServiceIdentifier string         `gorm:"uniqueIndex:idx_node_service_capability;type:varchar(100);not null"`
	Schema            datatypes.JSON `gorm:"type:json"`                         // 节点侧字段 Schema
	Protocol          string         `gorm:"type:varchar(50);not null"`         // http/mqtt/websocket
	Endpoint          *string        `gorm:"type:varchar(255)"`                 // 节点接收地址或主题
	Status            int            `gorm:"type:int;index;not null;default:1"` // 1启用，2禁用
}

func (NodeServiceCapability) TableName() string {
	return "node_service_capability"
}

// ControlCommand 记录服务端对节点发起的一次控制调用。
type ControlCommand struct {
	BaseModel
	NodeID            uint           `gorm:"index;not null"`
	ServiceIdentifier string         `gorm:"type:varchar(100);index;not null"`
	Payload           datatypes.JSON `gorm:"type:json"`                         // 服务端标准请求
	ConvertedPayload  datatypes.JSON `gorm:"type:json"`                         // 节点侧请求
	Status            int            `gorm:"type:int;index;not null;default:0"` // 0待发送，1已发送，2执行中，3成功，4失败，5超时
	Result            datatypes.JSON `gorm:"type:json"`                         // 节点返回结果
	ErrorMessage      *string        `gorm:"type:text"`
	SentAt            *time.Time     `gorm:"type:datetime"`
	CompletedAt       *time.Time     `gorm:"type:datetime"`
}

func (ControlCommand) TableName() string {
	return "control_command"
}

// ControlCommandLog 记录控制指令的状态变化和关键事件。
type ControlCommandLog struct {
	BaseModel
	CommandID uint           `gorm:"index;not null"`
	NodeID    uint           `gorm:"index;not null"`
	Event     string         `gorm:"type:varchar(100);index;not null"`
	Status    int            `gorm:"type:int;index;not null;default:0"`
	Message   *string        `gorm:"type:text"`
	Data      datatypes.JSON `gorm:"type:json"`
}

func (ControlCommandLog) TableName() string {
	return "control_command_log"
}
