package service

import (
	"context"
	"errors"
	"fmt"
	"nexus-core/domain/entity"
	"nexus-core/global"
	"nexus-core/monitor"
	"nexus-core/persistence/model"
	"time"

	"gorm.io/gorm"
)

// AccessService 负责 Access 相关的业务逻辑（自动绑定、心跳）
type AccessService struct {
	ls *LicenseService
	ns *NodeService
	ps *ProductService
}

func NewAccessService(ls *LicenseService, ns *NodeService, ps *ProductService) *AccessService {
	return &AccessService{ls: ls, ns: ns, ps: ps}
}

// AutoBindResult 返回自动绑定的结果（简化）
type AutoBindResult struct {
	NodeID    uint
	BindingOK bool
}

// HeartbeatResult 返回心跳结果（简化）
type HeartbeatResult struct {
	Online         bool                   `json:"online"`
	PendingControl *PendingControlSummary `json:"pending_control,omitempty"`
}

// Register 执行自动节点绑定注册逻辑
// 检查许可证，产品，版本之前的支持情况
// 绑定成功后激活许可证
func (s *AccessService) Register(ctx context.Context, cmd AccessCommand) (*RegisterResult, error) {
	deviceCode, licenseKey, productID, versionCode := cmd.DeviceCode, cmd.LicenseKey, cmd.ProductID, cmd.VersionCode
	var result *RegisterResult
	if err := global.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		//验证许可证产品支持
		license, err := GetLicenseEntityByKey(ctx, tx, licenseKey)
		if err != nil {
			return WrapInternal("get license failed", err)
		}
		if license == nil {
			return ErrBadRequest("invalid license")
		}
		if license.ProductID != productID {
			return Forbiddenf("license does not support product id %d", productID)
		}

		//验证产品版本支持
		product, err := GetProductEntityByID(ctx, tx, productID)
		if err != nil {
			return WrapInternal("get product failed", err)
		}
		if product == nil {
			return ErrBadRequest("invalid product")
		}
		if !product.CheckVersionSupportedByCode(versionCode) {
			return ErrBadRequest("version not supported")
		}

		// 检查许可证状态
		toActivate := false
		currentStatus := license.CalculateStatus(time.Now())
		switch currentStatus {
		case entity.StatusInactive:
			//尝试激活许可证
			if !license.Activate(time.Now()) || !license.IsActive() {
				return ErrConflict("license activation failed")
			}
			toActivate = true
		case entity.StatusActive:
		case entity.StatusExpired, entity.StatusRevoked:
			return ErrConflict("license not available")
		}

		// 检查当前绑定数量是否超过 MaxNodes
		if !license.ValidateNodeLimit() {
			return ErrConflict("license has reached max nodes")
		}

		// 检查 Node 是否存在
		node, err := GetNodeEntityByCode(ctx, tx, deviceCode)
		if err != nil {
			return WrapInternal("get node failed", err)
		}
		if node == nil {
			//创建节点
			newNode := &model.Node{
				DeviceCode: cmd.DeviceCode,
				Status:     entity.NodeStatusNormal,
			}
			err = nodeRepo.Create(ctx, tx, newNode)
			if err != nil {
				return WrapInternal("create node failed", err)
			}
			metadata := string(newNode.Metadata)
			node = &entity.Node{
				ID:         newNode.ID,
				DeviceCode: newNode.DeviceCode,
				Status:     0,
				Metadata:   &metadata,
			}
		} else if !node.IsValid() {
			return ErrForbidden("invalid node")
		}

		bound, err := bindNodeToLicense(ctx, tx, node.ID, license, productID)
		if err != nil {
			return err
		}

		if toActivate {
			if err := tx.Model(&model.License{}).Where("id = ?", license.ID).
				Updates(map[string]interface{}{
					"activated_at": license.ActivatedAt,
					"expired_at":   license.ExpiredAt,
					"status":       int(license.Status),
				}).Error; err != nil {
				return WrapInternal("update license activation failed", err)
			}
		}

		result = &RegisterResult{
			NodeID:             node.ID,
			LicenseID:          license.ID,
			ProductID:          productID,
			LicenseKey:         license.LicenseKey,
			LicenseStatus:      int(license.Status),
			FeatureMask:        license.FeatureMask,
			MaxNodes:           license.MaxNodes,
			CurrentNodeCount:   license.CurrentNodeCount,
			MaxConcurrent:      license.MaxConcurrent,
			HeartbeatInterval:  60,
			BindingEstablished: bound,
		}
		recordAuditLog(ctx, tx, "node", node.ID, "register", map[string]interface{}{
			"license_id": license.ID,
			"product_id": productID,
		})

		return nil
	}); err != nil {
		return nil, err
	}
	return result, nil
}

// Heartbeat 处理心跳逻辑
func (s *AccessService) Heartbeat(ctx context.Context, deviceCode string, productID uint, versionCode string, licenseKey string) (*HeartbeatResult, error) {
	product, err := GetProductEntityByID(ctx, global.DB.WithContext(ctx), productID)
	if err != nil {
		return nil, WrapInternal("get product failed", err)
	}
	if product == nil {
		return nil, ErrBadRequest("invalid product")
	}
	if !product.CheckVersionSupportedByCode(versionCode) {
		return nil, ErrBadRequest("product version not supported")
	}

	license, err := GetLicenseEntityByKey(ctx, global.DB.WithContext(ctx), licenseKey)
	if err != nil {
		return nil, WrapInternal("get license failed", err)
	}
	if license == nil {
		return nil, ErrBadRequest("invalid license")
	}

	if license.ProductID != productID {
		return nil, ErrForbidden("product not supported")
	}

	currentStatus := license.CalculateStatus(time.Now())
	switch currentStatus {
	case entity.StatusInactive:
		return nil, ErrConflict("license not active")
	case entity.StatusActive:
	case entity.StatusExpired:
		return nil, ErrConflict("license expired")
	case entity.StatusRevoked:
		return nil, ErrForbidden("invalid license")
	}

	node, err := GetNodeEntityByCode(ctx, global.DB.WithContext(ctx), deviceCode)
	if err != nil {
		return nil, WrapInternal("get node failed", err)
	}
	if node == nil {
		return nil, ErrNotFound("node not found")
	}
	if !node.IsValid() {
		return nil, ErrForbidden("invalid node")
	}

	var binding model.NodeLicenseBinding
	err = global.DB.WithContext(ctx).Where("node_id = ? AND license_id = ?", node.ID, license.ID).First(&binding).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound("binding not found")
	}
	if err != nil {
		return nil, WrapInternal("check binding failed", err)
	}
	if binding.Status != int(entity.BindingStatusBound) {
		return nil, ErrConflict("binding not bound")
	}

	onlineKey := fmt.Sprintf("%d|%s|%s", productID, node.DeviceCode, license.LicenseKey)

	// 并发检查，同一个节点刷新心跳不占用新的并发名额。
	totalConcurrent := monitor.GlobalStat.GetConcurrentByLicenseForProduct(license.LicenseKey, productID)
	if monitor.GlobalStat.HasOnlineNode(onlineKey) {
		totalConcurrent--
	}
	if !license.ValidateConcurrentLimit(totalConcurrent) {
		return nil, ErrConflict("maximum concurrent exceeded")
	}

	monitor.GlobalMonitor.HeartBeat(onlineKey, time.Second*60)
	monitor.GlobalStat.AddOnlineNode(onlineKey)
	now := time.Now()
	if err := global.DB.WithContext(ctx).Model(&model.Node{}).
		Where("id = ?", node.ID).
		Updates(map[string]interface{}{
			"last_seen_at": now,
			"online_at":    gorm.Expr("COALESCE(online_at, ?)", now),
		}).Error; err != nil {
		return nil, WrapInternal("update node heartbeat failed", err)
	}

	pendingControl, err := getPendingControlSummary(ctx, node.ID)
	if err != nil {
		return nil, err
	}

	return &HeartbeatResult{Online: true, PendingControl: pendingControl}, nil
}

func getPendingControlSummary(ctx context.Context, nodeID uint) (*PendingControlSummary, error) {
	statuses := []int{
		ControlCommandStatusPending,
		ControlCommandStatusSent,
		ControlCommandStatusRunning,
	}

	var count int64
	if err := global.DB.WithContext(ctx).Model(&model.ControlCommand{}).
		Where("node_id = ? AND status IN ?", nodeID, statuses).
		Count(&count).Error; err != nil {
		return nil, WrapInternal("count pending control commands failed", err)
	}

	var commands []model.ControlCommand
	if err := global.DB.WithContext(ctx).
		Select("id").
		Where("node_id = ? AND status IN ?", nodeID, statuses).
		Order("id DESC").
		Limit(10).
		Find(&commands).Error; err != nil {
		return nil, WrapInternal("list pending control commands failed", err)
	}

	commandIDs := make([]uint, 0, len(commands))
	for _, command := range commands {
		commandIDs = append(commandIDs, command.ID)
	}

	return &PendingControlSummary{
		Count:      int(count),
		CommandIDs: commandIDs,
	}, nil
}
