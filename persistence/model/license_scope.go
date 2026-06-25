package model

// LicenseProductScope 表示 License 可授权的产品范围。
// 当前 License 表仍保留 ProductID 作为主产品字段；该表用于后续扩展多产品授权。
type LicenseProductScope struct {
	BaseModel
	LicenseID uint `gorm:"uniqueIndex:idx_license_product_scope;index;not null"`
	ProductID uint `gorm:"uniqueIndex:idx_license_product_scope;index;not null"`
	Status    int  `gorm:"type:int;index;not null;default:1"` // 1启用，2禁用
}

func (LicenseProductScope) TableName() string {
	return "license_product_scope"
}

// LicenseServiceScope 表示 License 可使用的控制服务范围。
type LicenseServiceScope struct {
	BaseModel
	LicenseID         uint   `gorm:"uniqueIndex:idx_license_service_scope;index;not null"`
	ServiceIdentifier string `gorm:"uniqueIndex:idx_license_service_scope;type:varchar(100);not null"`
	Status            int    `gorm:"type:int;index;not null;default:1"` // 1启用，2禁用
}

func (LicenseServiceScope) TableName() string {
	return "license_service_scope"
}
