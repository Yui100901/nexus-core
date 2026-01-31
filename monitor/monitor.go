package monitor

import (
	"container/heap"
	"context"
	"fmt"
	"sync"
	"time"
)

//
// @Author yfy2001
// @Date 2026/1/31 12:08
//

// Node 表示一个被监控的节点。
// 每个节点维护心跳时间、过期时间和在线状态。
// 同时实现了堆元素所需的字段 index，用于在 NodeHeap 中定位。
type Node struct {
	ID            string        // 节点唯一标识
	Timeout       time.Duration // 心跳超时时间
	lastHeartbeat time.Time     // 最近一次心跳时间
	expiresAt     time.Time     // 节点过期时间（lastHeartbeat + Timeout）
	isOnline      bool          // 节点是否在线
	index         int           // 堆中的索引位置（由 NodeHeap 管理）
}

// NewNode 创建一个新的节点，并初始化心跳和过期时间。
func NewNode(id string, timeout time.Duration) *Node {
	now := time.Now()
	return &Node{
		ID:            id,
		Timeout:       timeout,
		lastHeartbeat: now,
		expiresAt:     now.Add(timeout),
		isOnline:      true,
	}
}

// Heartbeat 更新节点的心跳时间，刷新过期时间，并标记为在线。
func (n *Node) Heartbeat() {
	n.lastHeartbeat = time.Now()
	n.expiresAt = n.lastHeartbeat.Add(n.Timeout)
	n.isOnline = true
}

// ExpireTime 返回节点的过期时间。
func (n *Node) ExpireTime() time.Time {
	return n.expiresAt
}

// IsOnline 判断节点是否在线：
// 1. isOnline 标志为 true
// 2. 当前时间未超过过期时间
func (n *Node) IsOnline() bool {
	return n.isOnline && time.Now().Before(n.expiresAt)
}

// Remaining 返回节点距离过期的剩余时间。
func (n *Node) Remaining() time.Duration {
	return time.Until(n.expiresAt)
}

// NodeHeap 实现一个最小堆，按照节点过期时间排序。
// 堆顶元素是最早过期的节点。
type NodeHeap []*Node

func (h *NodeHeap) Len() int           { return len(*h) }
func (h *NodeHeap) Less(i, j int) bool { return (*h)[i].ExpireTime().Before((*h)[j].ExpireTime()) }
func (h *NodeHeap) Swap(i, j int) {
	(*h)[i], (*h)[j] = (*h)[j], (*h)[i]
	(*h)[i].index = i
	(*h)[j].index = j
}
func (h *NodeHeap) Push(x interface{}) {
	n := x.(*Node)
	n.index = len(*h)
	*h = append(*h, n)
}
func (h *NodeHeap) Pop() interface{} {
	old := *h
	n := len(old)
	node := old[n-1]
	node.index = -1
	*h = old[:n-1]
	return node
}

// Monitor 管理节点的注册、心跳更新和过期检查。
// 内部使用 NodeHeap 来快速找到最早过期的节点。
// 通过 context 控制监控协程的启动和停止。
type Monitor struct {
	mu      sync.Mutex         // 保护共享数据的互斥锁
	nodeMap map[string]*Node   // 节点映射表，便于快速查找
	heap    *NodeHeap          // 最小堆，按过期时间排序
	once    sync.Once          // 确保 Stop 只执行一次
	ctx     context.Context    // 控制监控协程退出的上下文
	cancel  context.CancelFunc // 用于取消 ctx 的函数
}

// NewMonitor 创建一个新的监控器实例。
func NewMonitor() *Monitor {
	h := &NodeHeap{}
	heap.Init(h)
	ctx, cancel := context.WithCancel(context.Background())
	return &Monitor{
		nodeMap: make(map[string]*Node),
		heap:    h,
		once:    sync.Once{},
		ctx:     ctx,
		cancel:  cancel,
	}
}

// Register 注册一个节点。
// 如果节点已存在，则更新其超时时间并刷新心跳。
// 如果是新节点，则加入堆和映射表。
func (m *Monitor) Register(id string, timeout time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if node, ok := m.nodeMap[id]; ok {
		node.Timeout = timeout
		node.Heartbeat()
		heap.Fix(m.heap, node.index)
		return
	}

	n := NewNode(id, timeout)
	m.nodeMap[n.ID] = n
	heap.Push(m.heap, n)
}

// UpdateHeartbeat 更新指定节点的心跳。
// 如果节点存在，则刷新其过期时间并调整堆位置。
func (m *Monitor) UpdateHeartbeat(nodeID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if node, ok := m.nodeMap[nodeID]; ok {
		node.Heartbeat()
		heap.Fix(m.heap, node.index)
	}
}

// Start 启动监控器。
// 在后台协程中循环检查堆顶节点是否过期。
// 如果节点过期，则标记为离线并移出堆。
func (m *Monitor) Start() {
	go func() {
		for {
			select {
			case <-m.ctx.Done():
				return
			default:
				m.mu.Lock()
				if m.heap.Len() == 0 {
					m.mu.Unlock()
					time.Sleep(time.Second)
					continue
				}
				node := (*m.heap)[0]
				wait := node.Remaining()
				m.mu.Unlock()

				// 等待节点过期或上下文取消
				if wait > 0 {
					select {
					case <-m.ctx.Done():
						return
					case <-time.After(wait):
					}
				}

				m.mu.Lock()
				if !node.IsOnline() {
					heap.Pop(m.heap)
					node.isOnline = false
					fmt.Printf("Node %s went offline at %v\n", node.ID, time.Now())
				} else {
					// 节点仍在线，重新调整堆位置
					heap.Fix(m.heap, node.index)
				}
				m.mu.Unlock()
			}
		}
	}()
}

// Stop 优雅退出监控器。
// 使用 sync.Once 确保只调用一次。
func (m *Monitor) Stop() {
	m.once.Do(func() {
		m.cancel()
		fmt.Println("Stopping monitor...")
	})
}

// GetOnlineNodes 返回当前在线的所有节点。
func (m *Monitor) GetOnlineNodes() []*Node {
	m.mu.Lock()
	defer m.mu.Unlock()
	var nodes []*Node
	for _, node := range m.nodeMap {
		if node.IsOnline() {
			nodes = append(nodes, node)
		}
	}
	return nodes
}

// GetOfflineNodes 返回当前离线的所有节点。
func (m *Monitor) GetOfflineNodes() []*Node {
	m.mu.Lock()
	defer m.mu.Unlock()
	var nodes []*Node
	for _, node := range m.nodeMap {
		if !node.IsOnline() {
			nodes = append(nodes, node)
		}
	}
	return nodes
}

// GlobalMonitor 提供一个全局默认的监控器实例。
// 可直接使用 GlobalMonitor.Register / UpdateHeartbeat / Start 等方法。
var GlobalMonitor = NewMonitor()
