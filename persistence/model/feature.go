package model

//
// @Author yfy2001
// @Date 2026/3/19 11 06
//

//通用的功能表，存储产品侧实现，或可实现的功能

type Feature struct {
	BaseModel
	Name        string `gorm:"type:varchar(255);not null;unique"` // 功能名称
	Description string `gorm:"type:text"`                         // 功能描述
	FeatureMask string `gorm:"type:varchar(255)"`                 // 功能模块掩码
}

func (Feature) TableName() string {
	return "feature"
}
