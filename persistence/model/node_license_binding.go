package model

//
// @Author yfy2001
// @Date 2026/1/30 13 57
//

// NodeLicenseBinding 节点绑定关系
type NodeLicenseBinding struct {
	BaseModel
	NodeID    uint `gorm:"index;not null"`          // 节点唯一标识 Node.ID
	LicenseID uint `gorm:"index;not null"`          // 对应 License.ID
	ProductID uint `gorm:"index;not null"`          // 对应 Product.ID
	IsBound   int  `gorm:"type:int;index;not null"` // 是否绑定
}

func (NodeLicenseBinding) TableName() string {
	return "node_license_binding"
}
