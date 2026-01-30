package entity

import "fmt"

//
// @Author yfy2001
// @Date 2026/1/16 16 46
//

// Node 表示用户环境的抽象节点
// 代表一台物理设备、虚拟设备或其他运行环境
type Node struct {
	ID         uint    // 节点唯一标识符
	DeviceCode string  // 设备唯一识别码，用于区分不同设备
	MetaInfo   *string // 设备元信息，包含操作系统、版本等信息
}

// NewNode 工厂方法
// 创建一个新的节点对象，默认没有绑定关系
func NewNode(deviceCode string, metaInfo *string) (*Node, error) {
	if deviceCode == "" {
		return nil, fmt.Errorf("device code cannot be empty")
	}

	node := &Node{
		DeviceCode: deviceCode,
		MetaInfo:   metaInfo,
	}

	return node, nil
}
