package model

//
// @Author yfy2001
// @Date 2026/3/19 11 06
//

//通用的功能表，存储产品功能

const (
	FeatureTypeAbility = "ability" //产品能力
	FeatureTypeCommand = "command" //下发命令

)

type Feature struct {
	Identifier  string  `json:"identifier" gorm:"type:varchar(255);unique"` // 功能标识符（通用表内唯一，产品内唯一）
	Type        string  `json:"type" gorm:"type:varchar(255);not null"`     // 功能类型
	Name        string  `json:"name" gorm:"type:varchar(255);not null"`     // 功能名称
	Description *string `json:"description" gorm:"type:text"`               // 功能描述
	Data        *string `json:"data" gorm:"type:text"`                      // 相关数据
}

type CommonFeature struct {
	BaseModel
	Feature
}

func (CommonFeature) TableName() string {
	return "common_feature"
}
