package service

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"nexus-core/global"
	"nexus-core/persistence/model"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

const (
	ControlServiceStatusEnabled  = 1
	ControlServiceStatusDisabled = 2
)

type ControlService struct {
}

func NewControlService() *ControlService {
	return &ControlService{}
}

func (s *ControlService) CreateControlService(ctx context.Context, cmd CreateControlServiceCommand) (*ControlServiceData, error) {
	if err := validateCreateControlServiceCommand(ctx, cmd); err != nil {
		return nil, err
	}

	control := &model.ControlService{
		ProductID:    cmd.ProductID,
		Identifier:   strings.TrimSpace(cmd.Identifier),
		Name:         strings.TrimSpace(cmd.Name),
		Description:  cmd.Description,
		ServiceType:  strings.TrimSpace(cmd.ServiceType),
		InputSchema:  normalizeJSON(cmd.InputSchema),
		OutputSchema: normalizeJSON(cmd.OutputSchema),
		Status:       ControlServiceStatusEnabled,
	}

	if err := global.DB.WithContext(ctx).Create(control).Error; err != nil {
		if isUniqueConstraintError(err) {
			return nil, ErrConflict("control service identifier already exists")
		}
		return nil, WrapInternal("create control service failed", err)
	}

	return toControlServiceData(control), nil
}

func (s *ControlService) GetControlServiceByID(ctx context.Context, id uint) (*ControlServiceData, error) {
	var control model.ControlService
	err := global.DB.WithContext(ctx).Where("id = ?", id).First(&control).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound("control service not found")
	}
	if err != nil {
		return nil, WrapInternal("get control service failed", err)
	}
	return toControlServiceData(&control), nil
}

func (s *ControlService) ListControlServices(ctx context.Context, productID *uint) ([]ControlServiceData, error) {
	var controls []model.ControlService
	query := global.DB.WithContext(ctx).Order("id DESC")
	if productID != nil {
		query = query.Where("product_id = ? OR product_id IS NULL", *productID)
	}
	if err := query.Find(&controls).Error; err != nil {
		return nil, WrapInternal("list control services failed", err)
	}

	data := make([]ControlServiceData, 0, len(controls))
	for i := range controls {
		data = append(data, *toControlServiceData(&controls[i]))
	}
	return data, nil
}

func (s *ControlService) UpdateControlService(ctx context.Context, cmd UpdateControlServiceCommand) (*ControlServiceData, error) {
	if cmd.ID == 0 {
		return nil, ErrBadRequest("id is required")
	}

	var existing model.ControlService
	err := global.DB.WithContext(ctx).Where("id = ?", cmd.ID).First(&existing).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound("control service not found")
	}
	if err != nil {
		return nil, WrapInternal("get control service failed", err)
	}

	updates := map[string]interface{}{}
	if cmd.ProductID != nil {
		product, err := productRepo.GetByID(ctx, global.DB.WithContext(ctx), *cmd.ProductID)
		if err != nil {
			return nil, WrapInternal("get product failed", err)
		}
		if product == nil {
			return nil, ErrNotFound("product not found")
		}
		updates["product_id"] = cmd.ProductID
	}
	if cmd.Name != nil {
		name := strings.TrimSpace(*cmd.Name)
		if name == "" {
			return nil, ErrBadRequest("name is required")
		}
		updates["name"] = name
	}
	if cmd.Description != nil {
		updates["description"] = cmd.Description
	}
	if cmd.ServiceType != nil {
		serviceType := strings.TrimSpace(*cmd.ServiceType)
		if !isValidControlServiceType(serviceType) {
			return nil, ErrBadRequest("invalid service_type")
		}
		updates["service_type"] = serviceType
	}
	if len(cmd.InputSchema) > 0 {
		if err := validateJSONSchema("input_schema", cmd.InputSchema); err != nil {
			return nil, err
		}
		updates["input_schema"] = normalizeJSON(cmd.InputSchema)
	}
	if len(cmd.OutputSchema) > 0 {
		if err := validateJSONSchema("output_schema", cmd.OutputSchema); err != nil {
			return nil, err
		}
		updates["output_schema"] = normalizeJSON(cmd.OutputSchema)
	}
	if len(updates) == 0 {
		return nil, ErrBadRequest("no control service fields to update")
	}

	if err := global.DB.WithContext(ctx).Model(&existing).Updates(updates).Error; err != nil {
		return nil, WrapInternal("update control service failed", err)
	}
	recordAuditLog(ctx, global.DB.WithContext(ctx), "control_service", existing.ID, "update", updates)
	return s.GetControlServiceByID(ctx, existing.ID)
}

func (s *ControlService) UpdateControlServiceStatus(ctx context.Context, cmd UpdateControlServiceStatusCommand) (*ControlServiceData, error) {
	if cmd.ID == 0 {
		return nil, ErrBadRequest("id is required")
	}
	if cmd.Status != ControlServiceStatusEnabled && cmd.Status != ControlServiceStatusDisabled {
		return nil, ErrBadRequest("invalid status")
	}

	result := global.DB.WithContext(ctx).Model(&model.ControlService{}).
		Where("id = ?", cmd.ID).
		Update("status", cmd.Status)
	if result.Error != nil {
		return nil, WrapInternal("update control service status failed", result.Error)
	}
	if result.RowsAffected == 0 {
		return nil, ErrNotFound("control service not found")
	}
	recordAuditLog(ctx, global.DB.WithContext(ctx), "control_service", cmd.ID, "status_update", map[string]interface{}{
		"status": cmd.Status,
	})
	return s.GetControlServiceByID(ctx, cmd.ID)
}

func (s *ControlService) DeleteControlService(ctx context.Context, id uint) error {
	if id == 0 {
		return ErrBadRequest("id is required")
	}

	var serviceDef model.ControlService
	err := global.DB.WithContext(ctx).Where("id = ?", id).First(&serviceDef).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrNotFound("control service not found")
	}
	if err != nil {
		return WrapInternal("get control service failed", err)
	}

	var capabilityCount int64
	if err := global.DB.WithContext(ctx).Model(&model.NodeServiceCapability{}).
		Where("service_identifier = ?", serviceDef.Identifier).
		Count(&capabilityCount).Error; err != nil {
		return WrapInternal("count node capabilities failed", err)
	}
	if capabilityCount > 0 {
		return ErrConflict("control service has node capabilities")
	}

	var commandCount int64
	if err := global.DB.WithContext(ctx).Model(&model.ControlCommand{}).
		Where("service_identifier = ?", serviceDef.Identifier).
		Count(&commandCount).Error; err != nil {
		return WrapInternal("count control commands failed", err)
	}
	if commandCount > 0 {
		return ErrConflict("control service has control commands")
	}

	if err := global.DB.WithContext(ctx).Delete(&serviceDef).Error; err != nil {
		return WrapInternal("delete control service failed", err)
	}
	recordAuditLog(ctx, global.DB.WithContext(ctx), "control_service", id, "delete", map[string]interface{}{
		"identifier": serviceDef.Identifier,
	})
	return nil
}

func validateCreateControlServiceCommand(ctx context.Context, cmd CreateControlServiceCommand) error {
	if strings.TrimSpace(cmd.Identifier) == "" {
		return ErrBadRequest("identifier is required")
	}
	if strings.TrimSpace(cmd.Name) == "" {
		return ErrBadRequest("name is required")
	}
	if !isValidControlServiceType(strings.TrimSpace(cmd.ServiceType)) {
		return ErrBadRequest("invalid service_type")
	}
	if err := validateJSONSchema("input_schema", cmd.InputSchema); err != nil {
		return err
	}
	if err := validateJSONSchema("output_schema", cmd.OutputSchema); err != nil {
		return err
	}
	if cmd.ProductID != nil {
		product, err := productRepo.GetByID(ctx, global.DB.WithContext(ctx), *cmd.ProductID)
		if err != nil {
			return WrapInternal("get product failed", err)
		}
		if product == nil {
			return ErrNotFound("product not found")
		}
	}
	return nil
}

func isValidControlServiceType(serviceType string) bool {
	switch serviceType {
	case model.ControlServiceTypeCommand,
		model.ControlServiceTypeConfig,
		model.ControlServiceTypeQuery,
		model.ControlServiceTypeAction:
		return true
	default:
		return false
	}
}

func validateJSONSchema(field string, raw json.RawMessage) error {
	if len(raw) == 0 {
		return nil
	}
	if !json.Valid(raw) {
		return ErrBadRequest(field + " must be valid json")
	}
	return nil
}

func normalizeJSON(raw json.RawMessage) datatypes.JSON {
	if len(raw) == 0 {
		return datatypes.JSON([]byte("{}"))
	}
	return datatypes.JSON(raw)
}

func toControlServiceData(control *model.ControlService) *ControlServiceData {
	return &ControlServiceData{
		ID:           control.ID,
		ProductID:    control.ProductID,
		Identifier:   control.Identifier,
		Name:         control.Name,
		Description:  control.Description,
		ServiceType:  control.ServiceType,
		InputSchema:  json.RawMessage(control.InputSchema),
		OutputSchema: json.RawMessage(control.OutputSchema),
		Status:       control.Status,
	}
}

func isUniqueConstraintError(err error) bool {
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "unique") || strings.Contains(message, "duplicate")
}
