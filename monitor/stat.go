package monitor

import (
	"fmt"
	"maps"
	"strconv"
	"strings"
	"sync"
)

//
// @Author yfy2001
// @Date 2026/1/31 16 52
//

type OnlineNodeKey struct {
	ProductID  uint
	DeviceCode string
	LicenseKey string
}

func NewOnlineNodeKey(productID uint, deviceCode string, licenseKey string) *OnlineNodeKey {
	return &OnlineNodeKey{
		ProductID:  productID,
		DeviceCode: deviceCode,
		LicenseKey: licenseKey,
	}
}

// From 将字符串解析为 OnlineNodeKey
// 格式: "ProductID|DeviceCode|LicenseKey"
func From(key string) (*OnlineNodeKey, error) {
	keys := strings.Split(key, "|")
	if len(keys) != 3 {
		return nil, fmt.Errorf("invalid key format: %s", key)
	}

	// 转换 ProductID
	id, err := strconv.ParseUint(keys[0], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid productID: %v", err)
	}

	return NewOnlineNodeKey(uint(id), keys[1], keys[2]), nil
}

// Key 将 OnlineNodeKey 序列化为字符串
func (k *OnlineNodeKey) Key() string {
	return fmt.Sprintf("%d|%s|%s", k.ProductID, k.DeviceCode, k.LicenseKey)
}

type OnlineStat struct {
	mu        sync.Mutex
	OnlineMap map[string]*OnlineNodeKey
}

func NewOnlineStat() *OnlineStat {
	return &OnlineStat{
		mu:        sync.Mutex{},
		OnlineMap: make(map[string]*OnlineNodeKey),
	}
}

func (s *OnlineStat) AddOnlineNode(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	onlineNodeKey, err := From(id)
	if err != nil {
		return
	}
	s.OnlineMap[id] = onlineNodeKey
}

func (s *OnlineStat) RemoveOnlineNode(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.OnlineMap, id)
}

// GetConcurrentByLicenseForProduct 获取许可证下某个产品并发使用情况
func (s *OnlineStat) GetConcurrentByLicenseForProduct(licenseKey string, productID uint) int {
	s.mu.Lock()
	onlineMapCopy := maps.Clone(s.OnlineMap)
	s.mu.Unlock()

	res := 0
	for _, onlineNodeKey := range onlineMapCopy {
		if onlineNodeKey.LicenseKey == licenseKey && onlineNodeKey.ProductID == productID {
			res++
		}
	}
	return res
}

// GetOnlineLicense 获取所有许可证的在线情况
func (s *OnlineStat) GetOnlineLicense(licenseKey string) map[string]struct{} {
	m := map[string]struct{}{}
	for _, onlineNodeKey := range s.OnlineMap {
		if onlineNodeKey.LicenseKey == licenseKey {
			m[onlineNodeKey.Key()] = struct{}{}
		}
	}
	return m
}

func (s *OnlineStat) OnNodeStateChange(node *Node, from, to NodeState) {

	// Init → Online（首次创建）
	if from == StateInit && to == StateOnline {
		fmt.Printf("Node %s: Init → Online (first seen)\n", node.ID)
		s.AddOnlineNode(node.ID)
		return
	}

	// Online → Offline（超时）
	if from == StateOnline && to == StateOffline {
		fmt.Printf("Node %s: Online → Offline (timeout)\n", node.ID)
		s.RemoveOnlineNode(node.ID)
		return
	}

	// Offline → Online（心跳恢复）
	if from == StateOffline && to == StateOnline {
		fmt.Printf("Node %s: Offline → Online (heartbeat recovery)\n", node.ID)
		s.AddOnlineNode(node.ID)
		return
	}

	//// Offline → Removed（离线太久被清理）
	//if from == StateOffline && to == StateRemoved {
	//	fmt.Printf("Node %s: Offline → Removed (cleanup)\n", node.ID)
	//	return
	//}

	//// Online → Removed（理论上不会发生）
	//if from == StateOnline && to == StateRemoved {
	//	fmt.Printf("Node %s: Online → Removed (unexpected)\n", node.ID)
	//	return
	//}
	//
	//// Init → Removed（理论上不会发生）
	//if from == StateInit && to == StateRemoved {
	//	fmt.Printf("Node %s: Init → Removed (unexpected)\n", node.ID)
	//	return
	//}
	//
	//// Offline → Offline（状态未变，不应触发事件）
	//if from == StateOffline && to == StateOffline {
	//	fmt.Printf("Node %s: Offline → Offline (should not happen)\n", node.ID)
	//	return
	//}
	//
	//// Online → Online（状态未变，不应触发事件）
	//if from == StateOnline && to == StateOnline {
	//	fmt.Printf("Node %s: Online → Online (should not happen)\n", node.ID)
	//	return
	//}
	//
	//// Removed → anything（Removed 是终态）
	//if from == StateRemoved {
	//	fmt.Printf("Node %s: Removed → %s (invalid transition)\n", node.ID, to)
	//	return
	//}
	//
	//// 兜底
	//fmt.Printf("Node %s: %s → %s (unknown transition)\n", node.ID, from, to)
}
