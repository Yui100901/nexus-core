package service

import (
	"context"
	"strconv"
	"time"

	"nexus-core/global"
	"nexus-core/monitor"
	"nexus-core/persistence/model"
)

type OnlineNodeData struct {
	ProductID  uint   `json:"product_id"`
	DeviceCode string `json:"device_code"`
	LicenseKey string `json:"license_key"`
}

func uintToString(value uint) string {
	return strconv.FormatUint(uint64(value), 10)
}

type OnlineCountData struct {
	Key   string `json:"key"`
	Count int    `json:"count"`
}

type OnlineSummaryData struct {
	TotalOnline int               `json:"total_online"`
	Nodes       []OnlineNodeData  `json:"nodes"`
	ByProduct   []OnlineCountData `json:"by_product"`
	ByLicense   []OnlineCountData `json:"by_license"`
}

type NodeHeartbeatData struct {
	ID         uint       `json:"id"`
	DeviceCode string     `json:"device_code"`
	Status     int        `json:"status"`
	LastSeenAt *time.Time `json:"last_seen_at"`
	OnlineAt   *time.Time `json:"online_at"`
	OfflineAt  *time.Time `json:"offline_at"`
}

type MonitorService struct{}

func NewMonitorService() *MonitorService {
	return &MonitorService{}
}

func (s *MonitorService) GetOnlineSummary(ctx context.Context) (*OnlineSummaryData, error) {
	snapshot := monitor.GlobalStat.Snapshot()
	byProduct := map[uint]int{}
	byLicense := map[string]int{}

	nodes := make([]OnlineNodeData, 0, len(snapshot))
	for _, item := range snapshot {
		nodes = append(nodes, OnlineNodeData{
			ProductID:  item.ProductID,
			DeviceCode: item.DeviceCode,
			LicenseKey: item.LicenseKey,
		})
		byProduct[item.ProductID]++
		byLicense[item.LicenseKey]++
	}

	productCounts := make([]OnlineCountData, 0, len(byProduct))
	for productID, count := range byProduct {
		productCounts = append(productCounts, OnlineCountData{
			Key:   uintToString(productID),
			Count: count,
		})
	}
	licenseCounts := make([]OnlineCountData, 0, len(byLicense))
	for licenseKey, count := range byLicense {
		licenseCounts = append(licenseCounts, OnlineCountData{
			Key:   licenseKey,
			Count: count,
		})
	}

	return &OnlineSummaryData{
		TotalOnline: len(snapshot),
		Nodes:       nodes,
		ByProduct:   productCounts,
		ByLicense:   licenseCounts,
	}, nil
}

func (s *MonitorService) ListNodeHeartbeats(ctx context.Context, limit int) ([]NodeHeartbeatData, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}

	var nodes []model.Node
	if err := global.DB.WithContext(ctx).
		Order("last_seen_at IS NULL ASC, last_seen_at DESC, id DESC").
		Limit(limit).
		Find(&nodes).Error; err != nil {
		return nil, WrapInternal("list node heartbeats failed", err)
	}

	data := make([]NodeHeartbeatData, 0, len(nodes))
	for i := range nodes {
		data = append(data, NodeHeartbeatData{
			ID:         nodes[i].ID,
			DeviceCode: nodes[i].DeviceCode,
			Status:     nodes[i].Status,
			LastSeenAt: nodes[i].LastSeenAt,
			OnlineAt:   nodes[i].OnlineAt,
			OfflineAt:  nodes[i].OfflineAt,
		})
	}
	return data, nil
}
