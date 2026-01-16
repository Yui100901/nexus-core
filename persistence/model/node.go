package model

import "time"

//
// @Author yfy2001
// @Date 2026/1/16 10 16
//

// Node 节点信息
type Node struct {
	ID         uint      `gorm:"primaryKey;autoIncrement"`               // 节点ID
	DeviceIMEI string    `gorm:"type:varchar(100);uniqueIndex;not null"` // 设备唯一识别码
	MetaInfo   string    `gorm:"type:text"`                              // 其他信息（操作系统、版本等）
	CreatedAt  time.Time `gorm:"autoCreateTime"`
	UpdatedAt  time.Time `gorm:"autoUpdateTime"`
}

func (Node) TableName() string {
	return "node"
}

// NodeBinding 节点绑定关系
type NodeBinding struct {
	ID        uint       `gorm:"primaryKey;autoIncrement"`
	NodeID    uint       `gorm:"index;not null"` // 节点唯一标识 Node.ID
	LicenseID uint       `gorm:"index;not null"` // 对应 License.ID
	ProductID uint       `gorm:"index;not null"` // 产品 Product.ID
	BoundAt   time.Time  `gorm:"type:datetime;not null"`
	UnboundAt *time.Time `gorm:"type:datetime"`
}

func (NodeBinding) TableName() string {
	return "node_binding"
}
