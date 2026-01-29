package entity

import (
	"fmt"
	"time"
)

//
// @Author yfy2001
// @Date 2026/1/20 10 44
//

// Product 代表系统中的一个产品
// 包含产品基本信息、版本列表和支持的最低版本
type Product struct {
	ID                    uint      // 产品唯一标识符
	Name                  string    // 产品名称，具有唯一性
	Description           *string   // 产品详细描述
	MinSupportedVersionID *uint     // 最低支持的版本ID，用于版本兼容性检查
	VersionList           []Version // 产品版本列表
}

// NewProduct 工厂方法
// 创建一个新的产品对象，默认版本列表为空
func NewProduct(name string, description *string, minSupportedVersionID *uint) (*Product, error) {
	if name == "" {
		return nil, fmt.Errorf("product name cannot be empty")
	}

	product := &Product{
		Name:                  name,
		Description:           description,
		MinSupportedVersionID: minSupportedVersionID,
		VersionList:           []Version{},
	}

	return product, nil
}

// Version 表示产品的具体版本信息
type Version struct {
	ID          uint
	VersionCode string    // 版本号，遵循语义化版本规范
	ReleaseDate time.Time // 版本发布时间
	Description *string   // 版本详细说明
	IsEnabled   int       // 版本状态，用于标识版本是否可用
}

// NewVersion 工厂方法
// 创建一个新的版本对象，默认状态为启用
func NewVersion(versionCode string, releaseDate time.Time, description *string) (*Version, error) {
	if versionCode == "" {
		return nil, fmt.Errorf("version code cannot be empty")
	}

	version := &Version{
		VersionCode: versionCode,
		ReleaseDate: releaseDate,
		Description: description,
		IsEnabled:   1, // 默认启用
	}

	return version, nil
}

func (p *Product) SetMinSupportedVersion(versionID uint) error {
	var targetVersion *Version
	for _, v := range p.VersionList {
		if v.ID == versionID {
			targetVersion = &v
			break
		}
	}

	if targetVersion == nil {
		return fmt.Errorf("version with ID %d not found", versionID)
	}

	if targetVersion.IsEnabled != 1 {
		return fmt.Errorf("version with ID %d is not enabled", versionID)
	}

	p.MinSupportedVersionID = &targetVersion.ID
	return nil
}

// ReleaseVersion 发布新版本
// 为产品添加一个新的版本，默认启用
func (p *Product) ReleaseVersion(newVersion Version) error {

	// 检查版本号是否已存在
	for _, v := range p.VersionList {
		if v.VersionCode == newVersion.VersionCode {
			return fmt.Errorf("newVersion %s already exists", newVersion.VersionCode)
		}
	}

	// 添加到版本列表
	p.VersionList = append(p.VersionList, newVersion)

	// 如果产品还没有最低支持版本，则设置为当前版本
	if p.MinSupportedVersionID == nil {
		p.MinSupportedVersionID = &newVersion.ID
	}

	return nil
}

// IsVersionSupported 判断某个产品是否支持某个版本
func (p *Product) IsVersionSupported(targetVersion Version) bool {
	if targetVersion.IsEnabled != 1 {
		return false
	}

	var minSupportVersion *Version
	for _, v := range p.VersionList {
		if p.MinSupportedVersionID != nil {
			if v.ID == *p.MinSupportedVersionID {
				minSupportVersion = &v
				break
			}
		}
	}

	if minSupportVersion == nil {
		return false // 没找到最低支持版本，视为不支持
	}

	// 等于或晚于最低支持版本的发布时间才算支持
	return !targetVersion.ReleaseDate.Before(minSupportVersion.ReleaseDate)
}
