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
