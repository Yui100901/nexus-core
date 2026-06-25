package service

import (
	"context"
	"encoding/json"
	"time"

	"nexus-core/global"
	"nexus-core/persistence/model"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type AuditLogData struct {
	ID           uint            `json:"id"`
	ResourceType string          `json:"resource_type"`
	ResourceID   uint            `json:"resource_id"`
	Action       string          `json:"action"`
	Operator     string          `json:"operator"`
	Data         json.RawMessage `json:"data"`
	CreatedAt    time.Time       `json:"created_at"`
}

type ListAuditLogsCommand struct {
	ResourceType *string
	ResourceID   *uint
	Action       *string
	Limit        int
}

type AuditService struct{}

func NewAuditService() *AuditService {
	return &AuditService{}
}

func (s *AuditService) ListAuditLogs(ctx context.Context, cmd ListAuditLogsCommand) ([]AuditLogData, error) {
	limit := cmd.Limit
	if limit <= 0 || limit > 200 {
		limit = 50
	}

	query := global.DB.WithContext(ctx).Model(&model.AuditLog{}).Order("id DESC").Limit(limit)
	if cmd.ResourceType != nil && *cmd.ResourceType != "" {
		query = query.Where("resource_type = ?", *cmd.ResourceType)
	}
	if cmd.ResourceID != nil {
		query = query.Where("resource_id = ?", *cmd.ResourceID)
	}
	if cmd.Action != nil && *cmd.Action != "" {
		query = query.Where("action = ?", *cmd.Action)
	}

	var logs []model.AuditLog
	if err := query.Find(&logs).Error; err != nil {
		return nil, WrapInternal("list audit logs failed", err)
	}

	data := make([]AuditLogData, 0, len(logs))
	for i := range logs {
		data = append(data, toAuditLogData(&logs[i]))
	}
	return data, nil
}

func recordAuditLog(ctx context.Context, db *gorm.DB, resourceType string, resourceID uint, action string, data interface{}) {
	if db == nil {
		db = global.DB
	}
	payload := datatypes.JSON([]byte("{}"))
	if data != nil {
		raw, err := json.Marshal(data)
		if err == nil && json.Valid(raw) {
			payload = datatypes.JSON(raw)
		}
	}

	log := model.AuditLog{
		ResourceType: resourceType,
		ResourceID:   resourceID,
		Action:       action,
		Operator:     "system",
		Data:         payload,
	}
	_ = db.WithContext(ctx).Create(&log).Error
}

func toAuditLogData(log *model.AuditLog) AuditLogData {
	data := json.RawMessage(log.Data)
	if len(data) == 0 || !json.Valid(data) {
		data = json.RawMessage(`{}`)
	}
	return AuditLogData{
		ID:           log.ID,
		ResourceType: log.ResourceType,
		ResourceID:   log.ResourceID,
		Action:       log.Action,
		Operator:     log.Operator,
		Data:         data,
		CreatedAt:    log.CreatedAt,
	}
}
