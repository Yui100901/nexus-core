package service

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"nexus-core/domain/entity"
	"nexus-core/global"
	"nexus-core/monitor"
	"nexus-core/persistence/base"
	"nexus-core/persistence/model"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

type flowFixture struct {
	ctx            context.Context
	db             *gorm.DB
	productService *ProductService
	licenseService *LicenseService
	nodeService    *NodeService
	accessService  *AccessService
	product        *ProductData
	license        *LicenseData
}

func newFlowFixture(t *testing.T, maxNodes int, maxConcurrent int, validityHours int) *flowFixture {
	t.Helper()

	ctx := context.Background()
	oldDB := global.DB
	oldStat := monitor.GlobalStat
	oldMonitor := monitor.GlobalMonitor
	t.Cleanup(func() {
		global.DB = oldDB
		monitor.GlobalStat = oldStat
		monitor.GlobalMonitor = oldMonitor
	})

	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "flow.db")), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get sql db: %v", err)
	}
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = sqlDB.Close() })

	base.AutoMigrate(db)
	global.DB = db
	monitor.GlobalStat = monitor.NewOnlineStat()
	monitor.GlobalMonitor = monitor.NewMonitor(monitor.GlobalStat)

	productService := NewProductService()
	licenseService := NewLicenseService()
	nodeService := NewNodeService()
	accessService := NewAccessService(licenseService, nodeService, productService)

	product, err := productService.CreateProduct(ctx, CreateProductCommand{Name: "flow-product"})
	if err != nil {
		t.Fatalf("create product: %v", err)
	}

	version, err := productService.CreateProductVersion(ctx, CreateProductVersionCommand{
		ProductID:   product.ID,
		VersionCode: "1.0.0",
		Method:      ReleaseImmediate,
	})
	if err != nil {
		t.Fatalf("create version: %v", err)
	}
	if version.ID == 0 {
		t.Fatal("version id should be assigned")
	}

	license, err := licenseService.CreateLicense(ctx, CreateLicenseCommand{
		ProductID:     product.ID,
		ValidityHours: validityHours,
		MaxNodes:      maxNodes,
		MaxConcurrent: maxConcurrent,
	})
	if err != nil {
		t.Fatalf("create license: %v", err)
	}

	return &flowFixture{
		ctx:            ctx,
		db:             db,
		productService: productService,
		licenseService: licenseService,
		nodeService:    nodeService,
		accessService:  accessService,
		product:        product,
		license:        license,
	}
}

func (f *flowFixture) register(t *testing.T, deviceCode string) *RegisterResult {
	t.Helper()
	result, err := f.accessService.Register(f.ctx, AccessCommand{
		DeviceCode:  deviceCode,
		LicenseKey:  f.license.LicenseKey,
		ProductID:   f.product.ID,
		VersionCode: "1.0.0",
	})
	if err != nil {
		t.Fatalf("register %s: %v", deviceCode, err)
	}
	return result
}

func assertAppErrorKind(t *testing.T, err error, kind ErrorKind) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected %s error, got nil", kind)
	}
	var appErr *AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected app error %s, got %T: %v", kind, err, err)
	}
	if appErr.Kind != kind {
		t.Fatalf("expected error kind %s, got %s: %v", kind, appErr.Kind, err)
	}
}

func TestLicenseMainFlow(t *testing.T) {
	fixture := newFlowFixture(t, 1, 1, 24)

	registerResult := fixture.register(t, "device-a")
	if registerResult.NodeID == 0 {
		t.Fatal("register should create node")
	}
	if registerResult.LicenseStatus != int(entity.StatusActive) {
		t.Fatalf("license should be active, got %d", registerResult.LicenseStatus)
	}
	if registerResult.CurrentNodeCount != 1 {
		t.Fatalf("current node count should be 1, got %d", registerResult.CurrentNodeCount)
	}

	var storedLicense model.License
	if err := fixture.db.Where("id = ?", fixture.license.ID).First(&storedLicense).Error; err != nil {
		t.Fatalf("get stored license: %v", err)
	}
	if storedLicense.CurrentNodeCount != 1 {
		t.Fatalf("stored current node count should be 1, got %d", storedLicense.CurrentNodeCount)
	}

	heartbeatResult, err := fixture.accessService.Heartbeat(fixture.ctx, "device-a", fixture.product.ID, "1.0.0", fixture.license.LicenseKey)
	if err != nil {
		t.Fatalf("heartbeat: %v", err)
	}
	if !heartbeatResult.Online {
		t.Fatal("heartbeat should mark node online")
	}
}

func TestBindingLimitAndUnbindRecovery(t *testing.T) {
	fixture := newFlowFixture(t, 1, 0, 24)

	first := fixture.register(t, "device-a")
	_, err := fixture.accessService.Register(fixture.ctx, AccessCommand{
		DeviceCode:  "device-b",
		LicenseKey:  fixture.license.LicenseKey,
		ProductID:   fixture.product.ID,
		VersionCode: "1.0.0",
	})
	assertAppErrorKind(t, err, ErrorKindConflict)

	if err := fixture.nodeService.UnbindByID(fixture.ctx, UnbindCommand{
		NodeID:    first.NodeID,
		LicenseID: fixture.license.ID,
	}); err != nil {
		t.Fatalf("unbind: %v", err)
	}

	var storedLicense model.License
	if err := fixture.db.Where("id = ?", fixture.license.ID).First(&storedLicense).Error; err != nil {
		t.Fatalf("get stored license: %v", err)
	}
	if storedLicense.CurrentNodeCount != 0 {
		t.Fatalf("current node count should be recovered to 0, got %d", storedLicense.CurrentNodeCount)
	}

	second := fixture.register(t, "device-b")
	if second.NodeID == 0 {
		t.Fatal("second register should create node after unbind")
	}
}

func TestConcurrentLimit(t *testing.T) {
	fixture := newFlowFixture(t, 2, 1, 24)

	fixture.register(t, "device-a")
	fixture.register(t, "device-b")

	if _, err := fixture.accessService.Heartbeat(fixture.ctx, "device-a", fixture.product.ID, "1.0.0", fixture.license.LicenseKey); err != nil {
		t.Fatalf("first heartbeat: %v", err)
	}
	_, err := fixture.accessService.Heartbeat(fixture.ctx, "device-b", fixture.product.ID, "1.0.0", fixture.license.LicenseKey)
	assertAppErrorKind(t, err, ErrorKindConflict)
}

func TestLicenseExpiredAndRevokedBlockHeartbeat(t *testing.T) {
	fixture := newFlowFixture(t, 1, 0, 24)
	fixture.register(t, "device-a")

	expiredAt := time.Now().Add(-time.Hour)
	if err := fixture.db.Model(&model.License{}).Where("id = ?", fixture.license.ID).Updates(map[string]interface{}{
		"expired_at": expiredAt,
		"status":     entity.StatusActive,
	}).Error; err != nil {
		t.Fatalf("expire license: %v", err)
	}
	_, err := fixture.accessService.Heartbeat(fixture.ctx, "device-a", fixture.product.ID, "1.0.0", fixture.license.LicenseKey)
	assertAppErrorKind(t, err, ErrorKindConflict)

	if err := fixture.db.Model(&model.License{}).Where("id = ?", fixture.license.ID).Updates(map[string]interface{}{
		"expired_at": nil,
		"status":     entity.StatusActive,
	}).Error; err != nil {
		t.Fatalf("reactivate license for revoke test: %v", err)
	}
	if err := fixture.licenseService.RevokeLicense(fixture.ctx, fixture.license.ID); err != nil {
		t.Fatalf("revoke license: %v", err)
	}
	_, err = fixture.accessService.Heartbeat(fixture.ctx, "device-a", fixture.product.ID, "1.0.0", fixture.license.LicenseKey)
	assertAppErrorKind(t, err, ErrorKindForbidden)
}

func TestBannedNodeBlocksRegisterBindingAndHeartbeat(t *testing.T) {
	fixture := newFlowFixture(t, 2, 0, 24)
	first := fixture.register(t, "device-a")

	if err := fixture.nodeService.BanNode(fixture.ctx, UpdateNodeStatusCommand{NodeID: first.NodeID}); err != nil {
		t.Fatalf("ban node: %v", err)
	}

	_, err := fixture.accessService.Heartbeat(fixture.ctx, "device-a", fixture.product.ID, "1.0.0", fixture.license.LicenseKey)
	assertAppErrorKind(t, err, ErrorKindForbidden)

	_, err = fixture.accessService.Register(fixture.ctx, AccessCommand{
		DeviceCode:  "device-a",
		LicenseKey:  fixture.license.LicenseKey,
		ProductID:   fixture.product.ID,
		VersionCode: "1.0.0",
	})
	assertAppErrorKind(t, err, ErrorKindForbidden)

	if err := fixture.nodeService.UnbindByID(fixture.ctx, UnbindCommand{
		NodeID:    first.NodeID,
		LicenseID: fixture.license.ID,
	}); err != nil {
		t.Fatalf("unbind banned node: %v", err)
	}
	err = fixture.nodeService.AddBinding(fixture.ctx, AddBindingCommand{
		NodeID:    first.NodeID,
		LicenseID: fixture.license.ID,
	})
	assertAppErrorKind(t, err, ErrorKindForbidden)

	if err := fixture.nodeService.UnbanNode(fixture.ctx, UpdateNodeStatusCommand{NodeID: first.NodeID}); err != nil {
		t.Fatalf("unban node: %v", err)
	}
	if err := fixture.nodeService.AddBinding(fixture.ctx, AddBindingCommand{
		NodeID:    first.NodeID,
		LicenseID: fixture.license.ID,
	}); err != nil {
		t.Fatalf("rebind unbanned node: %v", err)
	}
}
