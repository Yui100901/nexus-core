package entity

import "time"

//
// @Author yfy2001
// @Date 2026/1/20 10 44
//

// Product 代表系统中的一个产品
// 包含产品基本信息、版本列表和支持的最低版本
type Product struct {
	ID                    uint      // 产品唯一标识符
	Name                  string    // 产品名称，具有唯一性
	Description           string    // 产品详细描述
	MinSupportedVersionID uint      // 最低支持的版本ID，用于版本兼容性检查
	VersionList           []Version // 产品版本列表
}

// Version 表示产品的具体版本信息
type Version struct {
	ID          uint
	VersionCode string    // 版本号，遵循语义化版本规范
	ReleaseDate time.Time // 版本发布时间
	Description string    // 版本详细说明
	IsEnabled   int       // 版本状态，用于标识版本是否可用
}

// IsVersionSupported 判断某个产品是否支持某个版本
func (p *Product) IsVersionSupported(targetVersion Version) bool {
	if targetVersion.IsEnabled != 1 {
		return false
	}
	var minSupportVersion Version
	for _, v := range p.VersionList {
		if v.ID == p.MinSupportedVersionID {
			minSupportVersion = v
		}
	}
	if targetVersion.ReleaseDate.After(minSupportVersion.ReleaseDate) {
		return true
	} else {
		return false
	}
}
