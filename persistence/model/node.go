package model

//
// @Author yfy2001
// @Date 2026/1/16 10 16
//

import "time"

const (
	BoundStatusActive  = iota // 0 已绑定
	BoundStatusUnbound        // 1 已解绑
)

// Node 节点信息
type Node struct {
	BaseModel
	DeviceCode string `gorm:"type:varchar(100);uniqueIndex;not null"` // 设备唯一识别码
	MetaInfo   string `gorm:"type:text"`                              // 其他信息（操作系统、版本等）
}

func (Node) TableName() string {
	return "node"
}

// NodeBinding 节点绑定关系
type NodeBinding struct {
	BaseModel
	NodeID      uint       `gorm:"index;not null"` // 节点唯一标识 Node.ID
	LicenseID   uint       `gorm:"index;not null"` // 对应 License.ID
	BoundAt     time.Time  `gorm:"not null"`       // 绑定时间
	UnboundAt   *time.Time // 解绑时间，nil表示未解绑
	BoundStatus int        `gorm:"type:int;index;not null"` // 绑定状态枚举
}

func (NodeBinding) TableName() string {
	return "node_binding"
}
