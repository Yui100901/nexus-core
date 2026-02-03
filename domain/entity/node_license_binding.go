package entity

import "fmt"

//
// @Author yfy2001
// @Date 2026/1/30 13 53
//

// NodeLicenseBinding 定义节点与许可证之间的绑定关系
// 表示某个许可证在特定节点上的使用权
type NodeLicenseBinding struct {
	ID        uint // 绑定关系唯一标识符
	NodeID    uint // 绑定的节点ID，指向Node实体
	LicenseID uint // 关联的许可证ID，指向License实体
	ProductID uint // 关联的产品ID，指向Product实体
	IsBound   int  // 绑定状态，表示当前绑定的状态
}

// NewNodeLicenseBinding 工厂方法
// 创建一个新的节点绑定关系，默认状态为未激活
func NewNodeLicenseBinding(nodeID, licenseID, productID uint) (*NodeLicenseBinding, error) {
	if licenseID == 0 {
		return nil, fmt.Errorf("licenseID must be positive")
	}

	binding := &NodeLicenseBinding{
		NodeID:    nodeID,
		LicenseID: licenseID,
		ProductID: productID,
		IsBound:   0,
	}

	return binding, nil
}
