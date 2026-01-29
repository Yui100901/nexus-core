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
	MetaInfo   string        // 设备元信息，包含操作系统、版本等信息
	Bindings   []NodeBinding // 与此节点关联的许可证绑定关系
}

// NodeBinding 定义节点与许可证之间的绑定关系
// 表示某个许可证在特定节点上的使用权
type NodeBinding struct {
	ID        uint // 绑定关系唯一标识符
	LicenseID uint // 关联的许可证ID，指向License实体
	IsBound   int  // 绑定状态，表示当前绑定的状态
}

func (n *Node) Unbind(licenseID uint) error {
	for _, b := range n.Bindings {
		if b.LicenseID == licenseID {
			b.IsBound = StatusInactive
			return nil
		}
	}
	return fmt.Errorf("binding for node %d not found", licenseID)
}

func (n *Node) Bind(binding NodeBinding) {
	for _, b := range n.Bindings {
		if b.LicenseID == binding.LicenseID {
			if b.IsBound != StatusActive {
				b.IsBound = StatusActive
			}
		}
	}
	n.Bindings = append(n.Bindings, binding)
}
