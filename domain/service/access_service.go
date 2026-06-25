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

// ServiceError 用于在 service 层携带 HTTP 状态用于 controller 映射
type ServiceError struct {
	HTTPStatus int
	Err        error
}

func (e *ServiceError) Error() string {
	return e.Err.Error()
}

func NewServiceError(status int, msg string) *ServiceError {
	return &ServiceError{HTTPStatus: status, Err: errors.New(msg)}
}

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
	Online bool
}

// Register 执行自动节点绑定注册逻辑
// 检查许可证，产品，版本之前的支持情况
// 绑定成功后激活许可证
func (s *AccessService) Register(ctx context.Context, cmd AccessCommand) error {
	deviceCode, licenseKey, productID, versionCode := cmd.DeviceCode, cmd.LicenseKey, cmd.ProductID, cmd.VersionCode
	return global.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		//验证许可证产品支持
		license, err := GetLicenseEntityByKey(ctx, tx, licenseKey)
		if err != nil || license == nil {
			return fmt.Errorf("invalid license")
		}
		if license.ProductID != productID {
			return fmt.Errorf("license not support for this product id %d", productID)
		}

		//验证产品版本支持
		product, err := GetProductEntityByID(ctx, tx, productID)
		if err != nil || product == nil {
			return fmt.Errorf("invalid product")
		}
		if !product.CheckVersionSupportedByCode(versionCode) {
			return fmt.Errorf("version not supported")
		}

		// 检查许可证状态
		toActivate := false
		currentStatus := license.CalculateStatus(time.Now())
		switch currentStatus {
		case entity.StatusInactive:
			//尝试激活许可证
			if !license.Activate(time.Now()) || !license.IsActive() {
				return fmt.Errorf("license activation failed")
			}
			toActivate = true
		case entity.StatusActive:
		case entity.StatusExpired, entity.StatusRevoked:
			return fmt.Errorf("license not avaliable")
		}

		// 检查当前绑定数量是否超过 MaxNodes
		if !license.ValidateNodeLimit() {
			return fmt.Errorf("license has reached max nodes ")
		}

		// 检查 Node 是否存在
		node, err := GetNodeEntityByCode(ctx, tx, deviceCode)
		if err != nil {
			return err
		}
		if node == nil {
			//创建节点
			newNode := &model.Node{
				DeviceCode: cmd.DeviceCode,
				Status:     0, //默认正常
			}
			err = nodeRepo.Create(ctx, tx, newNode)
			if err != nil {
				return err
			}
			metadata := string(newNode.Metadata)
			node = &entity.Node{
				ID:         newNode.ID,
				DeviceCode: newNode.DeviceCode,
				Status:     0,
				Metadata:   &metadata,
			}
			// 插入新绑定
			now := time.Now()
			newBinding := model.NodeLicenseBinding{
				NodeID:    node.ID,
				LicenseID: license.ID,
				ProductID: productID,
				Status:    int(entity.BindingStatusBound),
				BoundAt:   &now,
			}
			if err := tx.Create(&newBinding).Error; err != nil {
				return fmt.Errorf("create binding failed")
			}
		} else {
			// 查找是否已有绑定关系
			var binding model.NodeLicenseBinding
			err = tx.Where("node_id = ? AND license_id = ?", node.ID, license.ID).
				First(&binding).Error
			if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}
			if errors.Is(err, gorm.ErrRecordNotFound) {
				now := time.Now()
				newBinding := model.NodeLicenseBinding{
					NodeID:    node.ID,
					LicenseID: license.ID,
					ProductID: productID,
					Status:    int(entity.BindingStatusBound),
					BoundAt:   &now,
				}
				if err := tx.Create(&newBinding).Error; err != nil {
					return fmt.Errorf("create binding failed")
				}
			} else if binding.Status != int(entity.BindingStatusBound) {
				now := time.Now()
				if err := tx.Model(&binding).Updates(map[string]interface{}{
					"status":     entity.BindingStatusBound,
					"bound_at":   &now,
					"unbound_at": nil,
				}).Error; err != nil {
					return err
				}
			}
		}
		if toActivate {
			tx.Model(model.License{}).Where("id = ?", license.ID).
				Updates(model.License{
					ActivatedAt: license.ActivatedAt,
					ExpiredAt:   license.ExpiredAt,
					Status:      int(license.Status),
				})
		}

		return nil
	})
}

// Heartbeat 处理心跳逻辑
func (s *AccessService) Heartbeat(ctx context.Context, deviceCode string, productID uint, versionCode string, licenseKey string) (*HeartbeatResult, error) {
	_ = ctx

	product, err := GetProductEntityByID(ctx, global.DB.WithContext(ctx), productID)
	if err != nil || product == nil {
		return nil, NewServiceError(400, "invalid product")
	}
	if !product.CheckVersionSupportedByCode(versionCode) {
		return nil, NewServiceError(400, "product version not supported")
	}

	license, err := GetLicenseEntityByKey(ctx, global.DB.WithContext(ctx), licenseKey)
	if err != nil || license == nil {
		return nil, NewServiceError(400, "invalid license")
	}

	if license.ProductID != productID {
		return nil, NewServiceError(400, "product not supported")
	}

	currentStatus := license.CalculateStatus(time.Now())
	switch currentStatus {
	case entity.StatusInactive:
		return nil, NewServiceError(400, "license not active")
	case entity.StatusActive:
	case entity.StatusExpired:
		return nil, NewServiceError(400, "license expired")
	case entity.StatusRevoked:
		return nil, NewServiceError(400, "invalid license")
	}

	node, err := GetNodeEntityByCode(ctx, global.DB.WithContext(ctx), deviceCode)
	if err != nil {
		return nil, NewServiceError(500, "get node failed")
	}
	if node == nil {
		return nil, NewServiceError(500, "node not found")
	}
	if !node.IsValid() {
		return nil, NewServiceError(400, "invalid node")
	}

	var binding model.NodeLicenseBinding
	err = global.DB.WithContext(ctx).Where("node_id = ? AND license_id = ?", node.ID, license.ID).First(&binding).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, NewServiceError(400, "binding not found")
	}
	if err != nil {
		return nil, NewServiceError(500, "check binding failed")
	}
	if binding.Status != int(entity.BindingStatusBound) {
		return nil, NewServiceError(400, "binding not bound")
	}

	// 并发检查
	totalConcurrent := monitor.GlobalStat.GetConcurrentByLicenseForProduct(license.LicenseKey, productID)
	if !license.ValidateConcurrentLimit(totalConcurrent) {
		return nil, NewServiceError(400, "maximum concurrent exceeded")
	}

	monitor.GlobalMonitor.HeartBeat(fmt.Sprintf("%d|%s|%s", productID, node.DeviceCode, license.LicenseKey), time.Second*60)

	return &HeartbeatResult{Online: true}, nil
}
