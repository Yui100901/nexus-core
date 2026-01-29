package entity

import "fmt"

//
// @Author yfy2001
// @Date 2026/1/16 16 46
//

// Node 表示用户环境的抽象节点
// 代表一台物理设备、虚拟设备或其他运行环境
type Node struct {
	ID         uint          // 节点唯一标识符
	DeviceCode string        // 设备唯一识别码，用于区分不同设备
	MetaInfo   *string       // 设备元信息，包含操作系统、版本等信息
	Bindings   []NodeBinding // 与此节点关联的许可证绑定关系
}

// NewNode 工厂方法
// 创建一个新的节点对象，默认没有绑定关系
func NewNode(deviceCode string, metaInfo *string, bindings []NodeBinding) (*Node, error) {
	if deviceCode == "" {
		return nil, fmt.Errorf("device code cannot be empty")
	}

	node := &Node{
		DeviceCode: deviceCode,
		MetaInfo:   metaInfo,
		Bindings:   []NodeBinding{}, // 初始化为空列表
	}
	if len(bindings) > 0 {
		for _, b := range bindings {
			node.Bind(b)
		}
	}

	return node, nil
}

// NodeBinding 定义节点与许可证之间的绑定关系
// 表示某个许可证在特定节点上的使用权
type NodeBinding struct {
	ID        uint // 绑定关系唯一标识符
	LicenseID uint // 关联的许可证ID，指向License实体
	IsBound   int  // 绑定状态，表示当前绑定的状态
}

// NewNodeBinding 工厂方法
// 创建一个新的节点绑定关系，默认状态为未激活
func NewNodeBinding(licenseID uint) (*NodeBinding, error) {
	if licenseID == 0 {
		return nil, fmt.Errorf("licenseID must be positive")
	}

	binding := &NodeBinding{
		LicenseID: licenseID,
		IsBound:   StatusInactive,
	}

	return binding, nil
}

func (n *Node) Unbind(licenseID uint) bool {
	for i := range n.Bindings {
		if n.Bindings[i].LicenseID == licenseID {
			if n.Bindings[i].IsBound == StatusInactive {
				return false // 已经解绑，无需操作
			}
			n.Bindings[i].IsBound = StatusInactive
			return true // 成功解绑
		}
	}
	// 不存在绑定关系，目标状态已满足，视为幂等成功
	return false
}

func (n *Node) Bind(binding NodeBinding) bool {
	for i := range n.Bindings {
		if n.Bindings[i].LicenseID == binding.LicenseID {
			if n.Bindings[i].IsBound == StatusActive {
				return false // 已经绑定，无需操作
			}
			n.Bindings[i].IsBound = StatusActive
			return true // 状态更新为绑定
		}
	}
	// 不存在绑定关系，新增绑定
	binding.IsBound = StatusActive
	n.Bindings = append(n.Bindings, binding)
	return true
}
