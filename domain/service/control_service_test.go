package service

import (
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"testing"

	"nexus-core/global"
	"nexus-core/persistence/base"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupControlServiceTestDB(t *testing.T) context.Context {
	t.Helper()

	oldDB := global.DB
	t.Cleanup(func() {
		global.DB = oldDB
	})

	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "control.db")), &gorm.Config{})
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
	return context.Background()
}

func TestCreateControlService(t *testing.T) {
	ctx := setupControlServiceTestDB(t)

	product, err := NewProductService().CreateProduct(ctx, CreateProductCommand{Name: "control-product"})
	if err != nil {
		t.Fatalf("create product: %v", err)
	}

	control, err := NewControlService().CreateControlService(ctx, CreateControlServiceCommand{
		ProductID:   &product.ID,
		Identifier:  "restart_process",
		Name:        "Restart Process",
		ServiceType: "command",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"required": ["process_name"],
			"properties": {
				"process_name": {"type": "string"},
				"delay_seconds": {"type": "integer", "default": 0}
			}
		}`),
		OutputSchema: json.RawMessage(`{"type":"object","properties":{"ok":{"type":"boolean"}}}`),
	})
	if err != nil {
		t.Fatalf("create control service: %v", err)
	}
	if control.ID == 0 {
		t.Fatal("control service id should be assigned")
	}
	if control.ProductID == nil || *control.ProductID != product.ID {
		t.Fatalf("control service product id mismatch: %#v", control.ProductID)
	}

	got, err := NewControlService().GetControlServiceByID(ctx, control.ID)
	if err != nil {
		t.Fatalf("get control service: %v", err)
	}
	if got.Identifier != "restart_process" {
		t.Fatalf("unexpected identifier: %s", got.Identifier)
	}

	list, err := NewControlService().ListControlServices(ctx, &product.ID)
	if err != nil {
		t.Fatalf("list control services: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 control service, got %d", len(list))
	}
}

func TestCreateControlServiceValidation(t *testing.T) {
	ctx := setupControlServiceTestDB(t)
	controlService := NewControlService()

	_, err := controlService.CreateControlService(ctx, CreateControlServiceCommand{
		Identifier:  "bad_schema",
		Name:        "Bad Schema",
		ServiceType: "command",
		InputSchema: json.RawMessage(`{`),
	})
	assertAppError(t, err, ErrorKindBadRequest)

	_, err = controlService.CreateControlService(ctx, CreateControlServiceCommand{
		Identifier:  "bad_type",
		Name:        "Bad Type",
		ServiceType: "unknown",
	})
	assertAppError(t, err, ErrorKindBadRequest)

	missingProductID := uint(404)
	_, err = controlService.CreateControlService(ctx, CreateControlServiceCommand{
		ProductID:   &missingProductID,
		Identifier:  "missing_product",
		Name:        "Missing Product",
		ServiceType: "command",
	})
	assertAppError(t, err, ErrorKindNotFound)

	_, err = controlService.CreateControlService(ctx, CreateControlServiceCommand{
		Identifier:  "duplicate",
		Name:        "Duplicate",
		ServiceType: "command",
	})
	if err != nil {
		t.Fatalf("create first duplicate service: %v", err)
	}
	_, err = controlService.CreateControlService(ctx, CreateControlServiceCommand{
		Identifier:  "duplicate",
		Name:        "Duplicate Again",
		ServiceType: "command",
	})
	assertAppError(t, err, ErrorKindConflict)
}

func assertAppError(t *testing.T, err error, kind ErrorKind) {
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
