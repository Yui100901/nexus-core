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

// -----------------------------
// Node 状态机
// -----------------------------

type NodeState int

func (s NodeState) StateCode() int {
	return int(s)
}

const (
	StateInit NodeState = iota
	StateOnline
	StateOffline
	StateRemoved
)

func (s NodeState) String() string {
	switch s {
	case StateInit:
		return "Init"
	case StateOnline:
		return "Online"
	case StateOffline:
		return "Offline"
	case StateRemoved:
		return "Removed"
	default:
		return "Unknown"
	}
}

// Node 表示一个被监控的节点。
type Node struct {
	ID            string
	Timeout       time.Duration
	lastHeartbeat time.Time
	expiresAt     time.Time

	index int // heap 内部使用
	state NodeState
}

func NewNode(id string, timeout time.Duration) *Node {
	now := time.Now()
	return &Node{
		ID:            id,
		Timeout:       timeout,
		lastHeartbeat: now,
		expiresAt:     now.Add(timeout),
		index:         -1,
		state:         StateInit,
	}
}

func (n *Node) Heartbeat() {
	n.lastHeartbeat = time.Now()
	n.expiresAt = n.lastHeartbeat.Add(n.Timeout)
}

func (n *Node) ExpireTime() time.Time {
	return n.expiresAt
}

func (n *Node) IsOnline() bool {
	return time.Now().Before(n.expiresAt)
}

func (n *Node) Remaining() time.Duration {
	return time.Until(n.expiresAt)
}

// -----------------------------
// NodeHeap 实现最小堆
// -----------------------------

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

// -----------------------------
// Collector 接口
// -----------------------------

type StatCollector interface {
	OnNodeStateChange(node *Node, from, to NodeState)
}

// -----------------------------
// 事件系统
// -----------------------------

type eventType int

const (
	eventStateChange eventType = iota
)

type event struct {
	t    eventType
	node *Node
	from NodeState
	to   NodeState
}

// -----------------------------
// Monitor 主结构
// -----------------------------

type Monitor struct {
	mu      sync.Mutex
	nodeMap map[string]*Node
	heap    *NodeHeap

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	cleanupInterval time.Duration
	maxOffline      time.Duration

	Stat StatCollector

	eventCh chan event // Stat 异步事件队列
}

func NewMonitor() *Monitor {
	h := &NodeHeap{}
	heap.Init(h)
	ctx, cancel := context.WithCancel(context.Background())

	return &Monitor{
		nodeMap:         make(map[string]*Node),
		heap:            h,
		ctx:             ctx,
		cancel:          cancel,
		cleanupInterval: 10 * time.Minute,
		maxOffline:      1 * time.Hour,
		eventCh:         make(chan event, 1024),
	}
}

func (m *Monitor) SetCollector(c StatCollector) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Stat = c
}

// -----------------------------
// 状态变更（核心）
// -----------------------------

func (m *Monitor) changeState(node *Node, newState NodeState) {
	if node.state == newState {
		return
	}

	old := node.state
	node.state = newState

	select {
	case m.eventCh <- event{
		t:    eventStateChange,
		node: node,
		from: old,
		to:   newState,
	}:
	default:
		// 队列满了可选择丢弃
	}
}

// -----------------------------
// 公共逻辑
// -----------------------------

func (m *Monitor) createNode(id string, timeout time.Duration) {
	node := NewNode(id, timeout)
	m.nodeMap[node.ID] = node
	heap.Push(m.heap, node)
	m.changeState(node, StateOnline)
}

func (m *Monitor) updateNode(node *Node) {
	wasOnline := node.IsOnline()

	node.Heartbeat()
	heap.Fix(m.heap, node.index)

	if !wasOnline && node.IsOnline() {
		m.changeState(node, StateOnline)
	}
}

// HeartBeat 节点上报心跳
func (m *Monitor) HeartBeat(id string, timeout time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if node, ok := m.nodeMap[id]; ok {
		node.Timeout = timeout
		m.updateNode(node)
		return
	}

	m.createNode(id, timeout)
}

// -----------------------------
// 过期检测
// -----------------------------

func (m *Monitor) offlineCheck() {
	node := (*m.heap)[0]

	if !node.IsOnline() {
		heap.Pop(m.heap)
		m.changeState(node, StateOffline)
	} else {
		heap.Fix(m.heap, node.index)
	}
}

// -----------------------------
// 清理死亡节点
// -----------------------------

func (m *Monitor) deathCheck(id string, n *Node, now time.Time) {
	if !n.IsOnline() && now.After(n.expiresAt.Add(m.maxOffline)) {
		heap.Remove(m.heap, n.index)
		delete(m.nodeMap, id)
		m.changeState(n, StateRemoved)
	}
}

// -----------------------------
// 主循环
// -----------------------------

func (m *Monitor) Start() {
	m.wg.Add(3)

	go m.expiryLoop()
	go m.cleanupLoop()
	go m.eventLoop()
}

func (m *Monitor) expiryLoop() {
	defer m.wg.Done()

	timer := time.NewTimer(time.Hour)
	defer timer.Stop()

	for {
		m.mu.Lock()
		if m.heap.Len() == 0 {
			m.mu.Unlock()
			timer.Reset(time.Second)
			select {
			case <-m.ctx.Done():
				return
			case <-timer.C:
				continue
			}
		}

		node := (*m.heap)[0]
		wait := node.Remaining()
		m.mu.Unlock()

		if wait < 0 {
			wait = 0
		}

		if !timer.Stop() {
			select {
			case <-timer.C:
			default:
			}
		}
		timer.Reset(wait)

		select {
		case <-m.ctx.Done():
			return
		case <-timer.C:
		}

		m.mu.Lock()
		m.offlineCheck()
		m.mu.Unlock()
	}
}

func (m *Monitor) cleanupLoop() {
	defer m.wg.Done()

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
				m.deathCheck(id, node, now)
			}
			m.mu.Unlock()
		}
	}
}

func (m *Monitor) eventLoop() {
	defer m.wg.Done()

	for {
		select {
		case <-m.ctx.Done():
			return
		case ev := <-m.eventCh:
			if m.Stat == nil {
				continue
			}

			if ev.t == eventStateChange {
				m.Stat.OnNodeStateChange(ev.node, ev.from, ev.to)
			}
		}
	}
}

// -----------------------------
// 停止
// -----------------------------

func (m *Monitor) Stop() {
	m.cancel()
	m.wg.Wait()
	fmt.Println("Monitor stopped gracefully")
}

// -----------------------------
// 查询接口
// -----------------------------

func (m *Monitor) GetOnlineNodes() []Node {
	m.mu.Lock()
	defer m.mu.Unlock()

	nodes := make([]Node, 0, len(m.nodeMap))
	for _, node := range m.nodeMap {
		if node.state == StateOnline {
			nodes = append(nodes, *node)
		}
	}
	return nodes
}

func (m *Monitor) GetOfflineNodes() []Node {
	m.mu.Lock()
	defer m.mu.Unlock()

	nodes := make([]Node, 0, len(m.nodeMap))
	for _, node := range m.nodeMap {
		if node.state == StateOffline {
			nodes = append(nodes, *node)
		}
	}
	return nodes
}

var GlobalMonitor = NewMonitor()
