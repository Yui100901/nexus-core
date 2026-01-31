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
// @Date 2026/1/31 12 08
//

// Node 表示一个节点，同时实现堆元素
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
func (h *NodeHeap) Less(i, j int) bool { return (*h)[i].expiresAt.Before((*h)[j].expiresAt) }
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

// Monitor 管理节点心跳和过期检查
type Monitor struct {
	mu      sync.Mutex
	nodeMap map[string]*Node
	heap    *NodeHeap
	once    sync.Once
	ctx     context.Context
	cancel  context.CancelFunc
}

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

// Start 启动监视器，使用结构体内的 context 控制退出
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
}

// Stop 优雅退出监视器
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
