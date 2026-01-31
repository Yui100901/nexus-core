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
type Node struct {
	ID            string
	Timeout       time.Duration
	lastHeartbeat time.Time
	expiresAt     time.Time
	isOnline      bool
	index         int
}

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

func (n *Node) Heartbeat() {
	n.lastHeartbeat = time.Now()
	n.expiresAt = n.lastHeartbeat.Add(n.Timeout)
	n.isOnline = true
}

func (n *Node) ExpireTime() time.Time {
	return n.expiresAt
}

func (n *Node) IsOnline() bool {
	return n.isOnline && time.Now().Before(n.expiresAt)
}

func (n *Node) Remaining() time.Duration {
	return time.Until(n.expiresAt)
}

// NodeHeap 实现最小堆
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

// Monitor 管理节点
type Monitor struct {
	mu      sync.Mutex
	nodeMap map[string]*Node
	heap    *NodeHeap

	// 生命周期管理
	once   sync.Once
	ctx    context.Context
	cancel context.CancelFunc

	// 清理参数
	cleanupInterval time.Duration
	maxOffline      time.Duration
}

func NewMonitor() *Monitor {
	h := &NodeHeap{}
	heap.Init(h)
	ctx, cancel := context.WithCancel(context.Background())
	return &Monitor{
		nodeMap:         make(map[string]*Node),
		heap:            h,
		once:            sync.Once{},
		ctx:             ctx,
		cancel:          cancel,
		cleanupInterval: 10 * time.Minute, // 默认每 10 分钟清理一次
		maxOffline:      1 * time.Hour,    // 默认离线超过 1 小时销毁
	}
}

// SetCleanupPolicy 设置清理策略
func (m *Monitor) SetCleanupPolicy(interval, maxOffline time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.cleanupInterval = interval
	m.maxOffline = maxOffline
}

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

func (m *Monitor) UpdateHeartbeat(nodeID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if node, ok := m.nodeMap[nodeID]; ok {
		node.Heartbeat()
		heap.Fix(m.heap, node.index)
	}
}

// Start 启动监控器，包含过期检测和僵尸清理
func (m *Monitor) Start() {
	// 过期检测协程
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
					heap.Fix(m.heap, node.index)
				}
				m.mu.Unlock()
			}
		}
	}()

	// 清理协程，清理太久不活跃的节点
	go func() {
		ticker := time.NewTicker(m.cleanupInterval)
		defer ticker.Stop()

		for {
			select {
			case <-m.ctx.Done():
				return
			case <-ticker.C:
				m.mu.Lock()
				now := time.Now()
				for id, node := range m.nodeMap {
					if !node.IsOnline() && now.Sub(node.expiresAt) > m.maxOffline {
						delete(m.nodeMap, id)
						fmt.Printf("Node %s removed after being offline for %v\n", id, m.maxOffline)
					}
				}
				m.mu.Unlock()
			}
		}
	}()
}

func (m *Monitor) Stop() {
	m.once.Do(func() {
		m.cancel()
		fmt.Println("Stopping monitor...")
	})
}

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

var GlobalMonitor = NewMonitor()
