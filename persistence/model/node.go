package model

//
// @Author yfy2001
// @Date 2026/1/16 10 16
//

// Node 节点信息
type Node struct {
	BaseModel
	DeviceCode string  `gorm:"type:varchar(100);uniqueIndex;not null"` // 设备唯一识别码
	Status     int     `gorm:"type:int;not null;default:0"`            // 状态（0=正常，1=封禁）
	MetaInfo   *string `gorm:"type:text"`                              // 其他信息（操作系统、版本等）
}

func (Node) TableName() string {
	return "node"
}
