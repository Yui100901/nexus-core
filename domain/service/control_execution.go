package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"nexus-core/domain/entity"
	"nexus-core/global"
	"nexus-core/persistence/model"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

const (
	NodeCapabilityStatusEnabled  = 1
	NodeCapabilityStatusDisabled = 2

	ControlCommandStatusPending = iota
	ControlCommandStatusSent
	ControlCommandStatusRunning
	ControlCommandStatusSuccess
	ControlCommandStatusFailed
	ControlCommandStatusTimeout
)

func (s *ControlService) ReportNodeCapability(ctx context.Context, cmd ReportNodeCapabilityCommand) (*NodeCapabilityData, error) {
	if err := validateNodeCapabilityCommand(ctx, cmd); err != nil {
		return nil, err
	}

	capability := model.NodeServiceCapability{
		NodeID:            cmd.NodeID,
		ServiceIdentifier: strings.TrimSpace(cmd.ServiceIdentifier),
		Schema:            normalizeJSON(cmd.Schema),
		Protocol:          strings.TrimSpace(cmd.Protocol),
		Endpoint:          cmd.Endpoint,
		Status:            NodeCapabilityStatusEnabled,
	}

	var existing model.NodeServiceCapability
	err := global.DB.WithContext(ctx).
		Where("node_id = ? AND service_identifier = ?", cmd.NodeID, capability.ServiceIdentifier).
		First(&existing).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, WrapInternal("get node capability failed", err)
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		if err := global.DB.WithContext(ctx).Create(&capability).Error; err != nil {
			return nil, WrapInternal("create node capability failed", err)
		}
		return toNodeCapabilityData(&capability), nil
	}

	if err := global.DB.WithContext(ctx).Model(&existing).Updates(map[string]interface{}{
		"schema":     capability.Schema,
		"protocol":   capability.Protocol,
		"endpoint":   capability.Endpoint,
		"status":     NodeCapabilityStatusEnabled,
		"updated_at": time.Now(),
	}).Error; err != nil {
		return nil, WrapInternal("update node capability failed", err)
	}
	existing.Schema = capability.Schema
	existing.Protocol = capability.Protocol
	existing.Endpoint = capability.Endpoint
	existing.Status = NodeCapabilityStatusEnabled
	return toNodeCapabilityData(&existing), nil
}

func (s *ControlService) ListNodeCapabilities(ctx context.Context, nodeID uint) ([]NodeCapabilityData, error) {
	var capabilities []model.NodeServiceCapability
	query := global.DB.WithContext(ctx).Order("id DESC")
	if nodeID != 0 {
		query = query.Where("node_id = ?", nodeID)
	}
	if err := query.Find(&capabilities).Error; err != nil {
		return nil, WrapInternal("list node capabilities failed", err)
	}

	data := make([]NodeCapabilityData, 0, len(capabilities))
	for i := range capabilities {
		data = append(data, *toNodeCapabilityData(&capabilities[i]))
	}
	return data, nil
}

func (s *ControlService) CreateControlCommand(ctx context.Context, cmd CreateControlCommand) (*ControlCommandData, error) {
	if cmd.NodeID == 0 {
		return nil, ErrBadRequest("node_id is required")
	}
	if strings.TrimSpace(cmd.ServiceIdentifier) == "" {
		return nil, ErrBadRequest("service_identifier is required")
	}
	if len(cmd.Payload) == 0 || !json.Valid(cmd.Payload) {
		return nil, ErrBadRequest("payload must be valid json")
	}

	node, err := GetNodeEntityByID(ctx, global.DB.WithContext(ctx), cmd.NodeID)
	if err != nil {
		return nil, WrapInternal("get node failed", err)
	}
	if node == nil {
		return nil, ErrNotFound("node not found")
	}
	if !node.IsValid() {
		return nil, ErrForbidden("invalid node")
	}

	serviceDef, err := s.getEnabledControlService(ctx, strings.TrimSpace(cmd.ServiceIdentifier))
	if err != nil {
		return nil, err
	}
	capability, err := s.getEnabledNodeCapability(ctx, cmd.NodeID, serviceDef.Identifier)
	if err != nil {
		return nil, err
	}
	if err := validateControlNodeOnline(ctx, cmd.NodeID, capability.Protocol); err != nil {
		return nil, err
	}
	if err := validateNodeHasValidBindingForControl(ctx, cmd.NodeID, serviceDef); err != nil {
		return nil, err
	}

	convertedPayload, err := ConvertPayload(cmd.Payload, json.RawMessage(capability.Schema))
	if err != nil {
		return nil, err
	}

	command := &model.ControlCommand{
		NodeID:            cmd.NodeID,
		ServiceIdentifier: serviceDef.Identifier,
		Payload:           normalizeJSON(cmd.Payload),
		ConvertedPayload:  normalizeJSON(convertedPayload),
		Status:            ControlCommandStatusPending,
	}
	if err := global.DB.WithContext(ctx).Create(command).Error; err != nil {
		return nil, WrapInternal("create control command failed", err)
	}
	_ = createControlCommandLog(ctx, command.ID, command.NodeID, "created", command.Status, nil, nil)

	if err := s.dispatchControlCommand(ctx, command, capability); err != nil {
		message := err.Error()
		now := time.Now()
		_ = global.DB.WithContext(ctx).Model(command).Updates(map[string]interface{}{
			"status":        ControlCommandStatusFailed,
			"error_message": &message,
			"completed_at":  &now,
		}).Error
		command.Status = ControlCommandStatusFailed
		command.ErrorMessage = &message
		command.CompletedAt = &now
		_ = createControlCommandLog(ctx, command.ID, command.NodeID, "failed", command.Status, &message, nil)
		return toControlCommandData(command), nil
	}

	return toControlCommandData(command), nil
}

func (s *ControlService) GetControlCommandByID(ctx context.Context, id uint) (*ControlCommandData, error) {
	var command model.ControlCommand
	err := global.DB.WithContext(ctx).Where("id = ?", id).First(&command).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound("control command not found")
	}
	if err != nil {
		return nil, WrapInternal("get control command failed", err)
	}
	return toControlCommandData(&command), nil
}

func (s *ControlService) CompleteControlCommand(ctx context.Context, cmd CompleteControlCommandCommand) (*ControlCommandData, error) {
	if cmd.CommandID == 0 {
		return nil, ErrBadRequest("command_id is required")
	}

	var command model.ControlCommand
	err := global.DB.WithContext(ctx).Where("id = ?", cmd.CommandID).First(&command).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound("control command not found")
	}
	if err != nil {
		return nil, WrapInternal("get control command failed", err)
	}

	response := ControlCommandResponse{
		CommandID:    cmd.CommandID,
		Status:       cmd.Status,
		Result:       cmd.Result,
		ErrorMessage: cmd.ErrorMessage,
	}
	if err := applyControlCommandResponse(ctx, &command, response); err != nil {
		return nil, err
	}
	return toControlCommandData(&command), nil
}

func (s *ControlService) getEnabledControlService(ctx context.Context, identifier string) (*model.ControlService, error) {
	var serviceDef model.ControlService
	err := global.DB.WithContext(ctx).
		Where("identifier = ? AND status = ?", identifier, ControlServiceStatusEnabled).
		First(&serviceDef).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound("control service not found")
	}
	if err != nil {
		return nil, WrapInternal("get control service failed", err)
	}
	return &serviceDef, nil
}

func (s *ControlService) getEnabledNodeCapability(ctx context.Context, nodeID uint, identifier string) (*model.NodeServiceCapability, error) {
	var capability model.NodeServiceCapability
	err := global.DB.WithContext(ctx).
		Where("node_id = ? AND service_identifier = ? AND status = ?", nodeID, identifier, NodeCapabilityStatusEnabled).
		First(&capability).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound("node capability not found")
	}
	if err != nil {
		return nil, WrapInternal("get node capability failed", err)
	}
	return &capability, nil
}

func validateNodeCapabilityCommand(ctx context.Context, cmd ReportNodeCapabilityCommand) error {
	if cmd.NodeID == 0 {
		return ErrBadRequest("node_id is required")
	}
	if strings.TrimSpace(cmd.ServiceIdentifier) == "" {
		return ErrBadRequest("service_identifier is required")
	}
	if len(cmd.Schema) == 0 || !json.Valid(cmd.Schema) {
		return ErrBadRequest("schema must be valid json")
	}
	if !isSupportedControlProtocol(strings.TrimSpace(cmd.Protocol)) {
		return ErrBadRequest("invalid protocol")
	}
	switch strings.TrimSpace(cmd.Protocol) {
	case "http":
		if cmd.Endpoint == nil || strings.TrimSpace(*cmd.Endpoint) == "" {
			return ErrBadRequest("endpoint is required for http protocol")
		}
	case "mqtt":
		if cmd.Endpoint == nil || strings.TrimSpace(*cmd.Endpoint) == "" {
			return ErrBadRequest("endpoint topic is required for mqtt protocol")
		}
	}
	node, err := GetNodeEntityByID(ctx, global.DB.WithContext(ctx), cmd.NodeID)
	if err != nil {
		return WrapInternal("get node failed", err)
	}
	if node == nil {
		return ErrNotFound("node not found")
	}
	if !node.IsValid() {
		return ErrForbidden("invalid node")
	}
	var serviceDef model.ControlService
	err = global.DB.WithContext(ctx).
		Where("identifier = ? AND status = ?", strings.TrimSpace(cmd.ServiceIdentifier), ControlServiceStatusEnabled).
		First(&serviceDef).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrNotFound("control service not found")
	}
	if err != nil {
		return WrapInternal("get control service failed", err)
	}
	return nil
}

func isSupportedControlProtocol(protocol string) bool {
	switch protocol {
	case "http", "mqtt", "websocket":
		return true
	default:
		return false
	}
}

func validateNodeHasValidBindingForControl(ctx context.Context, nodeID uint, serviceDef *model.ControlService) error {
	if serviceDef.ProductID == nil {
		return nil
	}

	var bindings []model.NodeLicenseBinding
	if err := global.DB.WithContext(ctx).
		Where("node_id = ? AND product_id = ? AND status = ?", nodeID, *serviceDef.ProductID, entity.BindingStatusBound).
		Find(&bindings).Error; err != nil {
		return WrapInternal("get node bindings failed", err)
	}

	for _, binding := range bindings {
		license, err := GetLicenseEntityByID(ctx, global.DB.WithContext(ctx), binding.LicenseID)
		if err != nil {
			return WrapInternal("get license failed", err)
		}
		if license != nil && license.CalculateStatus(time.Now()) == entity.StatusActive {
			if err := validateLicenseControlServiceScope(ctx, license.ID, serviceDef.Identifier); err != nil {
				return err
			}
			return nil
		}
	}

	return ErrForbidden("node has no valid license for control service")
}

func validateLicenseControlServiceScope(ctx context.Context, licenseID uint, serviceIdentifier string) error {
	var scopeCount int64
	if err := global.DB.WithContext(ctx).Model(&model.LicenseServiceScope{}).
		Where("license_id = ?", licenseID).
		Count(&scopeCount).Error; err != nil {
		return WrapInternal("count license service scopes failed", err)
	}
	if scopeCount == 0 {
		return nil
	}

	var allowedCount int64
	if err := global.DB.WithContext(ctx).Model(&model.LicenseServiceScope{}).
		Where("license_id = ? AND service_identifier = ? AND status = ?", licenseID, serviceIdentifier, 1).
		Count(&allowedCount).Error; err != nil {
		return WrapInternal("count license service scope failed", err)
	}
	if allowedCount == 0 {
		return ErrForbidden("license does not allow control service")
	}
	return nil
}

func validateControlNodeOnline(ctx context.Context, nodeID uint, protocol string) error {
	if protocol == "websocket" && DefaultControlWebSocketHub.IsOnline(nodeID) {
		return nil
	}

	var node model.Node
	err := global.DB.WithContext(ctx).Where("id = ?", nodeID).First(&node).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrNotFound("node not found")
	}
	if err != nil {
		return WrapInternal("get node failed", err)
	}
	if node.LastSeenAt == nil {
		return ErrConflict("node is not online")
	}
	ttl := time.Duration(global.GetConfig().Control.NodeOnlineTTLSeconds) * time.Second
	if ttl <= 0 {
		ttl = 120 * time.Second
	}
	if time.Since(*node.LastSeenAt) > ttl {
		return ErrConflict("node is not online")
	}
	return nil
}

func (s *ControlService) dispatchControlCommand(ctx context.Context, command *model.ControlCommand, capability *model.NodeServiceCapability) error {
	attempts := global.GetConfig().Control.DispatchMaxRetries + 1
	if attempts <= 0 {
		attempts = 1
	}
	var lastErr error
	for i := 0; i < attempts; i++ {
		if i > 0 {
			_ = createControlCommandLog(ctx, command.ID, command.NodeID, "retry", command.Status, nil, mustMarshalJSON(map[string]interface{}{
				"attempt": i + 1,
			}))
		}
		switch capability.Protocol {
		case "http":
			lastErr = dispatchHTTPControlCommand(ctx, command, capability)
		case "mqtt":
			lastErr = dispatchMQTTControlCommand(ctx, command, capability)
		case "websocket":
			lastErr = DefaultControlWebSocketHub.Dispatch(ctx, command)
		default:
			return ErrBadRequest("invalid protocol")
		}
		if lastErr == nil {
			return nil
		}
	}
	return lastErr
}

func dispatchHTTPControlCommand(ctx context.Context, command *model.ControlCommand, capability *model.NodeServiceCapability) error {
	if capability.Endpoint == nil || strings.TrimSpace(*capability.Endpoint) == "" {
		return ErrBadRequest("endpoint is required for http protocol")
	}

	if err := markControlCommandSent(ctx, command); err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, *capability.Endpoint, bytes.NewReader(command.ConvertedPayload))
	if err != nil {
		return WrapInternal("create control request failed", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: controlDispatchTimeout()}
	resp, err := client.Do(req)
	if err != nil {
		return WrapInternal("send control command failed", err)
	}
	defer resp.Body.Close()

	result, err := io.ReadAll(resp.Body)
	if err != nil {
		return WrapInternal("read control response failed", err)
	}
	if len(result) == 0 {
		result = []byte("{}")
	}
	if !json.Valid(result) {
		result = mustMarshalJSON(map[string]interface{}{
			"raw": string(result),
		})
	}

	completedAt := time.Now()
	status := ControlCommandStatusSuccess
	var errorMessage *string
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		status = ControlCommandStatusFailed
		message := resp.Status
		errorMessage = &message
	}

	event := "succeeded"
	if status == ControlCommandStatusFailed {
		event = "failed"
	}
	return completeControlCommand(ctx, command, status, event, errorMessage, result, &completedAt)
}

func controlDispatchTimeout() time.Duration {
	timeout := time.Duration(global.GetConfig().Control.DispatchTimeoutSeconds) * time.Second
	if timeout <= 0 {
		return 5 * time.Second
	}
	return timeout
}

func createControlCommandLog(ctx context.Context, commandID uint, nodeID uint, event string, status int, message *string, data []byte) error {
	log := model.ControlCommandLog{
		CommandID: commandID,
		NodeID:    nodeID,
		Event:     event,
		Status:    status,
		Message:   message,
		Data:      normalizeJSON(data),
	}
	return global.DB.WithContext(ctx).Create(&log).Error
}

func markControlCommandSent(ctx context.Context, command *model.ControlCommand) error {
	now := time.Now()
	if err := global.DB.WithContext(ctx).Model(command).Updates(map[string]interface{}{
		"status":  ControlCommandStatusSent,
		"sent_at": &now,
	}).Error; err != nil {
		return WrapInternal("mark command sent failed", err)
	}
	command.Status = ControlCommandStatusSent
	command.SentAt = &now
	_ = createControlCommandLog(ctx, command.ID, command.NodeID, "sent", command.Status, nil, nil)
	return nil
}

func completeControlCommand(ctx context.Context, command *model.ControlCommand, status int, event string, message *string, data []byte, completedAt *time.Time) error {
	if len(data) == 0 {
		data = []byte("{}")
	}
	if !json.Valid(data) {
		data = mustMarshalJSON(map[string]interface{}{
			"raw": string(data),
		})
	}
	if completedAt == nil {
		now := time.Now()
		completedAt = &now
	}
	converted, err := convertControlCommandResult(ctx, command.ServiceIdentifier, data)
	if err != nil {
		status = ControlCommandStatusFailed
		event = "failed"
		errMessage := err.Error()
		message = &errMessage
	} else {
		data = converted
	}

	if err := global.DB.WithContext(ctx).Model(command).Updates(map[string]interface{}{
		"status":        status,
		"result":        datatypes.JSON(data),
		"error_message": message,
		"completed_at":  completedAt,
	}).Error; err != nil {
		return WrapInternal("update control command result failed", err)
	}

	command.Status = status
	command.Result = datatypes.JSON(data)
	command.ErrorMessage = message
	command.CompletedAt = completedAt
	_ = createControlCommandLog(ctx, command.ID, command.NodeID, event, status, message, data)
	return nil
}

func convertControlCommandResult(ctx context.Context, serviceIdentifier string, data []byte) ([]byte, error) {
	if len(data) == 0 || !json.Valid(data) {
		return data, nil
	}
	var serviceDef model.ControlService
	err := global.DB.WithContext(ctx).
		Where("identifier = ?", serviceIdentifier).
		First(&serviceDef).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return data, nil
	}
	if err != nil {
		return nil, WrapInternal("get control service failed", err)
	}
	if len(serviceDef.OutputSchema) == 0 {
		return data, nil
	}
	converted, err := ConvertPayload(json.RawMessage(data), json.RawMessage(serviceDef.OutputSchema))
	if err != nil {
		return nil, err
	}
	return converted, nil
}

func toNodeCapabilityData(capability *model.NodeServiceCapability) *NodeCapabilityData {
	return &NodeCapabilityData{
		ID:                capability.ID,
		NodeID:            capability.NodeID,
		ServiceIdentifier: capability.ServiceIdentifier,
		Schema:            json.RawMessage(capability.Schema),
		Protocol:          capability.Protocol,
		Endpoint:          capability.Endpoint,
		Status:            capability.Status,
	}
}

func toControlCommandData(command *model.ControlCommand) *ControlCommandData {
	return &ControlCommandData{
		ID:                command.ID,
		NodeID:            command.NodeID,
		ServiceIdentifier: command.ServiceIdentifier,
		Payload:           json.RawMessage(command.Payload),
		ConvertedPayload:  json.RawMessage(command.ConvertedPayload),
		Status:            command.Status,
		Result:            json.RawMessage(command.Result),
		ErrorMessage:      command.ErrorMessage,
	}
}

func mustMarshalJSON(value interface{}) []byte {
	data, err := json.Marshal(value)
	if err != nil {
		return []byte("{}")
	}
	return data
}
