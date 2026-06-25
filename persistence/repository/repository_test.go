package repository

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"nexus-core/persistence/base"
	"nexus-core/persistence/model"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupRepositoryTestDB(t *testing.T) (context.Context, *gorm.DB) {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "repository.db")), &gorm.Config{})
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
	return context.Background(), db
}

func TestRepositoryGenericHelpers(t *testing.T) {
	ctx, db := setupRepositoryTestDB(t)

	node := &model.Node{
		DeviceCode: "repo-node",
		Status:     1,
	}
	if err := NewNodeRepository().Create(ctx, db, node); err != nil {
		t.Fatalf("create node: %v", err)
	}

	got, err := GetOneByUniqueColumn[model.Node](ctx, db, "device_code", "repo-node")
	if err != nil {
		t.Fatalf("get by unique column: %v", err)
	}
	if got == nil || got.ID != node.ID {
		t.Fatalf("node mismatch: %#v", got)
	}

	rows, err := FindByColumn[model.Node](ctx, db, "status", 1)
	if err != nil {
		t.Fatalf("find by column: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected one row, got %d", len(rows))
	}

	count, err := CountWhere(ctx, db, &model.Node{}, "status = ?", 1)
	if err != nil {
		t.Fatalf("count where: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected count 1, got %d", count)
	}

	affected, err := UpdateByColumn[model.Node](ctx, db, "id", node.ID, map[string]interface{}{
		"status": 2,
	})
	if err != nil {
		t.Fatalf("update by column: %v", err)
	}
	if affected != 1 {
		t.Fatalf("expected one row affected, got %d", affected)
	}

	updated, err := NewNodeRepository().GetByID(ctx, db, node.ID)
	if err != nil {
		t.Fatalf("get updated node: %v", err)
	}
	if updated.Status != 2 {
		t.Fatalf("status should be updated, got %d", updated.Status)
	}

	deleted, err := DeleteByUniqueColumn[model.Node](ctx, db, "id", node.ID)
	if err != nil {
		t.Fatalf("delete by unique column: %v", err)
	}
	if deleted != 1 {
		t.Fatalf("expected one row deleted, got %d", deleted)
	}
}

func TestNodeAndLicenseRepositories(t *testing.T) {
	ctx, db := setupRepositoryTestDB(t)

	nodeRepo := NewNodeRepository()
	licenseRepo := NewLicenseRepository()

	node := &model.Node{DeviceCode: "repo-node-a", Status: 1}
	if err := nodeRepo.Create(ctx, db, node); err != nil {
		t.Fatalf("create node: %v", err)
	}
	byDevice, err := nodeRepo.GetByDeviceCode(ctx, db, "repo-node-a")
	if err != nil {
		t.Fatalf("get node by device: %v", err)
	}
	if byDevice == nil || byDevice.ID != node.ID {
		t.Fatalf("node by device mismatch: %#v", byDevice)
	}

	license := &model.License{
		ProductID:     1,
		LicenseKey:    "repo-license-key",
		ValidityHours: 24,
		Status:        1,
		MaxNodes:      2,
		ActivatedAt:   ptrTime(time.Now()),
	}
	if err := licenseRepo.Create(ctx, db, license); err != nil {
		t.Fatalf("create license: %v", err)
	}
	byKey, err := licenseRepo.GetByKey(ctx, db, "repo-license-key")
	if err != nil {
		t.Fatalf("get license by key: %v", err)
	}
	if byKey == nil || byKey.ID != license.ID {
		t.Fatalf("license by key mismatch: %#v", byKey)
	}

	if err := licenseRepo.UpdateLicenseStatus(ctx, db, license.ID, 3); err != nil {
		t.Fatalf("update license status: %v", err)
	}
	ids, err := licenseRepo.GetIdListByStatus(ctx, db, 3)
	if err != nil {
		t.Fatalf("get ids by status: %v", err)
	}
	if len(ids) != 1 || ids[0] != license.ID {
		t.Fatalf("ids by status mismatch: %#v", ids)
	}

	if err := licenseRepo.BatchDeleteByIdList(ctx, db, ids); err != nil {
		t.Fatalf("batch delete licenses: %v", err)
	}
	deletedLicense, err := licenseRepo.GetByID(ctx, db, license.ID)
	if err != nil {
		t.Fatalf("get deleted license: %v", err)
	}
	if deletedLicense != nil {
		t.Fatalf("license should be deleted: %#v", deletedLicense)
	}
}

func ptrTime(value time.Time) *time.Time {
	return &value
}
