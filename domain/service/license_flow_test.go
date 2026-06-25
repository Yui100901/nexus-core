package service

import (
	"context"
	"path/filepath"
	"testing"

	"nexus-core/domain/entity"
	"nexus-core/global"
	"nexus-core/persistence/base"
	"nexus-core/persistence/model"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestLicenseMainFlow(t *testing.T) {
	ctx := context.Background()
	oldDB := global.DB
	t.Cleanup(func() {
		global.DB = oldDB
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
		ValidityHours: 24,
		MaxNodes:      1,
		MaxConcurrent: 1,
	})
	if err != nil {
		t.Fatalf("create license: %v", err)
	}

	registerResult, err := accessService.Register(ctx, AccessCommand{
		DeviceCode:  "device-a",
		LicenseKey:  license.LicenseKey,
		ProductID:   product.ID,
		VersionCode: "1.0.0",
	})
	if err != nil {
		t.Fatalf("register: %v", err)
	}
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
	if err := db.Where("id = ?", license.ID).First(&storedLicense).Error; err != nil {
		t.Fatalf("get stored license: %v", err)
	}
	if storedLicense.CurrentNodeCount != 1 {
		t.Fatalf("stored current node count should be 1, got %d", storedLicense.CurrentNodeCount)
	}

	heartbeatResult, err := accessService.Heartbeat(ctx, "device-a", product.ID, "1.0.0", license.LicenseKey)
	if err != nil {
		t.Fatalf("heartbeat: %v", err)
	}
	if !heartbeatResult.Online {
		t.Fatal("heartbeat should mark node online")
	}
}
