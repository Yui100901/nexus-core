package model

import "time"

//
// @Author yfy2001
// @Date 2026/1/30 13 57
//

// NodeLicenseBinding 节点绑定关系
type NodeLicenseBinding struct {
	BaseModel
	NodeID    uint       `gorm:"uniqueIndex:idx_node_license;index;not null"` // 节点唯一标识 Node.ID
	LicenseID uint       `gorm:"uniqueIndex:idx_node_license;index;not null"` // 对应 License.ID
	ProductID uint       `gorm:"index;not null;default:0"`                    // 关联产品
	Status    int        `gorm:"type:int;index;not null;default:0"`           // 状态：0未绑定，1已绑定
	BoundAt   *time.Time `gorm:"type:datetime"`                               // 绑定时间
	UnboundAt *time.Time `gorm:"type:datetime"`                               // 解绑时间
}

func (NodeLicenseBinding) TableName() string {
	return "node_license_binding"
}
