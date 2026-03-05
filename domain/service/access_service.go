package service

import (
	"fmt"
	"nexus-core/domain/entity"
	"nexus-core/monitor"
	"nexus-core/persistence/base"
	"nexus-core/sc"
	"time"
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
	return &ServiceError{HTTPStatus: status, Err: fmt.Errorf(msg)}
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

// AutoBind 执行自动绑定逻辑，返回 ServiceError 以便 controller 映射HTTP状态
func (s *AccessService) AutoBind(ctx *sc.ServiceContext, deviceCode string, productID uint, versionCode string, licenseKey string) (*AutoBindResult, error) {
	// 验证产品和版本是否支持
	ok, err := s.ps.CheckProductVersionSupported(ctx, productID, nil, &versionCode)
	if err != nil {
		return nil, NewServiceError(500, "internal error")
	}
	if !ok {
		return nil, NewServiceError(400, "product version not supported")
	}

	// 找到 license
	license, err := s.ls.GetLicenseByKey(ctx, licenseKey)
	if err != nil || license == nil {
		return nil, NewServiceError(400, "invalid license")
	}

	// 验证许可证是否对产品有效
	scope := license.GetScope(productID)
	if scope == nil {
		return nil, NewServiceError(400, "product not supported")
	}

	// 检查许可证状态
	currentStatus := license.CheckStatus(time.Now())
	toActivate := false
	switch currentStatus {
	case entity.StatusInactive:
		toActivate = true
	case entity.StatusActive:
	case entity.StatusExpired:
		return nil, NewServiceError(400, "license expired")
	case entity.StatusRevoked:
		return nil, NewServiceError(400, "invalid license")
	}

	// Wrap the critical section in a transaction and propagate tx via ServiceContext
	var res *AutoBindResult
	err = ctx.WithTransactionUsingDB(base.Connect(), func(txCtx *sc.ServiceContext) error {
		// txCtx already has Tx set and InTx true

		// 查找或创建节点 using context-aware node service (no nested tx)
		node, err := s.ns.AutoCreateNodeWithContext(txCtx, deviceCode, nil)
		if err != nil {
			return fmt.Errorf("create node failed")
		}
		if !node.IsValid() {
			return fmt.Errorf("invalid node")
		}

		// 检查绑定
		binding, err := s.ns.GetBindingByNodeAndLicense(txCtx, node.ID, license.ID)
		if err != nil {
			return fmt.Errorf("check binding failed")
		}

		if binding == nil {
			if err := s.ns.AutoCreateBindWithContext(txCtx, node.ID, productID, license); err != nil {
				return fmt.Errorf("auto bind failed")
			}
		} else {
			if binding.IsBound == 0 {
				if err := s.ns.UpdateBindingStatus(txCtx, binding.ID, 1); err != nil {
					return fmt.Errorf("update binding status failed")
				}
			}
		}

		if toActivate {
			// call license activation with tx (txCtx.GetDB() will return tx)
			txDB := txCtx.GetDB()
			if txDB == nil {
				return fmt.Errorf("transaction not available for activation")
			}
			if err := s.ls.ActivateLicenseIfNeededWithTx(txCtx, txDB, license); err != nil {
				return fmt.Errorf("activate license failed")
			}
		}

		res = &AutoBindResult{NodeID: node.ID, BindingOK: true}
		return nil
	})

	if err != nil {
		// map error to ServiceError
		return nil, NewServiceError(500, err.Error())
	}

	return res, nil
}

// Heartbeat 处理心跳逻辑
func (s *AccessService) Heartbeat(ctx *sc.ServiceContext, deviceCode string, productID uint, versionCode string, licenseKey string) (*HeartbeatResult, error) {
	ok, err := s.ps.CheckProductVersionSupported(ctx, productID, nil, &versionCode)
	if err != nil {
		return nil, NewServiceError(500, "internal error")
	}
	if !ok {
		return nil, NewServiceError(400, "product version not supported")
	}

	license, err := s.ls.GetLicenseByKey(ctx, licenseKey)
	if err != nil || license == nil {
		return nil, NewServiceError(400, "invalid license")
	}

	scope := license.GetScope(productID)
	if scope == nil {
		return nil, NewServiceError(400, "product not supported")
	}

	currentStatus := license.CheckStatus(time.Now())
	switch currentStatus {
	case entity.StatusInactive:
		return nil, NewServiceError(400, "license not active")
	case entity.StatusActive:
	case entity.StatusExpired:
		return nil, NewServiceError(400, "license expired")
	case entity.StatusRevoked:
		return nil, NewServiceError(400, "invalid license")
	}

	node, err := s.ns.GetByDeviceCode(ctx, deviceCode)
	if err != nil {
		return nil, NewServiceError(500, "get node failed")
	}
	if node == nil {
		return nil, NewServiceError(500, "node not found")
	}
	if !node.IsValid() {
		return nil, NewServiceError(400, "invalid node")
	}

	binding, err := s.ns.GetBindingByNodeAndLicense(ctx, node.ID, license.ID)
	if err != nil {
		return nil, NewServiceError(500, "check binding failed")
	}
	if binding == nil {
		return nil, NewServiceError(400, "binding not found")
	}
	if binding.IsBound == 0 {
		return nil, NewServiceError(400, "binding not bound")
	}

	// 并发检查
	totalConcurrent := monitor.GlobalStat.GetConcurrentByLicenseForProduct(license.LicenseKey, productID)
	if !license.ValidateMaxConcurrentForProduct(productID, totalConcurrent) {
		return nil, NewServiceError(400, "maximum concurrent exceeded")
	}

	monitor.GlobalMonitor.HeartBeat(fmt.Sprintf("%d|%s|%s", productID, node.DeviceCode, license.LicenseKey), time.Second*60)

	return &HeartbeatResult{Online: true}, nil
}
