package service

import (
	"strings"
	"testing"
	"time"

	"nexus-core/domain/entity"
	"nexus-core/persistence/model"
)

func TestP2UpdateProduct(t *testing.T) {
	fixture := newFlowFixture(t, 1, 0, 24)

	description := "updated description"
	updated, err := fixture.productService.UpdateProduct(fixture.ctx, UpdateProductCommand{
		ID:          fixture.product.ID,
		Description: &description,
	})
	if err != nil {
		t.Fatalf("update product: %v", err)
	}
	if updated.Description == nil || *updated.Description != description {
		t.Fatalf("description mismatch: %#v", updated)
	}

	empty := " "
	_, err = fixture.productService.UpdateProduct(fixture.ctx, UpdateProductCommand{
		ID:   fixture.product.ID,
		Name: &empty,
	})
	assertAppErrorKind(t, err, ErrorKindBadRequest)
}

func TestP2BatchCreateAndRestoreLicense(t *testing.T) {
	fixture := newFlowFixture(t, 1, 0, 24)

	licenses, err := fixture.licenseService.BatchCreateLicenses(fixture.ctx, BatchCreateLicenseCommand{
		ProductID:     fixture.product.ID,
		ValidityHours: 24,
		MaxNodes:      2,
		Count:         3,
	})
	if err != nil {
		t.Fatalf("batch create licenses: %v", err)
	}
	if len(licenses) != 3 {
		t.Fatalf("expected 3 licenses, got %d", len(licenses))
	}
	keys := map[string]bool{}
	for _, license := range licenses {
		if license.ID == 0 || license.LicenseKey == "" {
			t.Fatalf("license should have id and key: %#v", license)
		}
		if keys[license.LicenseKey] {
			t.Fatalf("duplicate license key: %s", license.LicenseKey)
		}
		keys[license.LicenseKey] = true
	}

	register := fixture.register(t, "restore-node")
	if err := fixture.licenseService.RevokeLicense(fixture.ctx, register.LicenseID); err != nil {
		t.Fatalf("revoke license: %v", err)
	}
	restored, err := fixture.licenseService.RestoreLicense(fixture.ctx, RestoreLicenseCommand{ID: register.LicenseID})
	if err != nil {
		t.Fatalf("restore license: %v", err)
	}
	if restored.Status != int(entity.StatusActive) {
		t.Fatalf("restored license should be active, got %d", restored.Status)
	}
}

func TestP2UpdateNodeMetadata(t *testing.T) {
	fixture := newFlowFixture(t, 1, 0, 24)
	register := fixture.register(t, "metadata-node")

	metadata := `{"os":"windows","version":"1.0.1"}`
	updated, err := fixture.nodeService.UpdateNode(fixture.ctx, UpdateNodeCommand{
		ID:       register.NodeID,
		Metadata: &metadata,
	})
	if err != nil {
		t.Fatalf("update node metadata: %v", err)
	}
	if updated.Metadata == nil || !strings.Contains(*updated.Metadata, `"version":"1.0.1"`) {
		t.Fatalf("metadata mismatch: %#v", updated.Metadata)
	}

	invalid := `{bad json`
	_, err = fixture.nodeService.UpdateNode(fixture.ctx, UpdateNodeCommand{
		ID:       register.NodeID,
		Metadata: &invalid,
	})
	assertAppErrorKind(t, err, ErrorKindBadRequest)
}

func TestP2ProductDeletePolicy(t *testing.T) {
	fixture := newFlowFixture(t, 1, 0, 24)

	err := fixture.productService.DeleteProduct(fixture.ctx, fixture.product.ID)
	assertAppErrorKind(t, err, ErrorKindConflict)

	emptyProduct, err := fixture.productService.CreateProduct(fixture.ctx, CreateProductCommand{Name: "empty-product"})
	if err != nil {
		t.Fatalf("create empty product: %v", err)
	}
	if _, err := NewControlService().CreateControlService(fixture.ctx, CreateControlServiceCommand{
		ProductID:   &emptyProduct.ID,
		Identifier:  "delete_policy_service",
		Name:        "Delete Policy Service",
		ServiceType: "command",
	}); err != nil {
		t.Fatalf("create control service: %v", err)
	}
	err = fixture.productService.DeleteProduct(fixture.ctx, emptyProduct.ID)
	assertAppErrorKind(t, err, ErrorKindConflict)

	deletable, err := fixture.productService.CreateProduct(fixture.ctx, CreateProductCommand{Name: "deletable-product"})
	if err != nil {
		t.Fatalf("create deletable product: %v", err)
	}
	if _, err := fixture.productService.CreateProductVersion(fixture.ctx, CreateProductVersionCommand{
		ProductID:   deletable.ID,
		VersionCode: "0.1.0",
		Method:      ReleaseHold,
	}); err != nil {
		t.Fatalf("create deletable version: %v", err)
	}
	if err := fixture.productService.DeleteProduct(fixture.ctx, deletable.ID); err != nil {
		t.Fatalf("delete product without dependencies: %v", err)
	}
	_, err = fixture.productService.GetProductDataByID(fixture.ctx, deletable.ID)
	assertAppErrorKind(t, err, ErrorKindNotFound)
}

func TestP2VersionScheduleAndVersionGuards(t *testing.T) {
	fixture := newFlowFixture(t, 1, 0, 24)

	_, err := fixture.productService.CreateProductVersion(fixture.ctx, CreateProductVersionCommand{
		ProductID:   fixture.product.ID,
		VersionCode: "scheduled-without-date",
		Method:      ReleaseScheduled,
	})
	assertAppErrorKind(t, err, ErrorKindBadRequest)

	dueAt := time.Now().Add(-time.Minute)
	scheduled, err := fixture.productService.CreateProductVersion(fixture.ctx, CreateProductVersionCommand{
		ProductID:   fixture.product.ID,
		VersionCode: "1.1.0",
		ReleaseDate: &dueAt,
		Method:      ReleaseScheduled,
	})
	if err != nil {
		t.Fatalf("create scheduled version: %v", err)
	}

	var stored model.ProductVersion
	if err := fixture.db.Where("id = ?", scheduled.ID).First(&stored).Error; err != nil {
		t.Fatalf("get scheduled version: %v", err)
	}
	if stored.Status != int(entity.VersionStatusAvailable) {
		t.Fatalf("due scheduled version should be released, got %d", stored.Status)
	}
	err = fixture.productService.ReleaseVersion(fixture.ctx, ReleaseNewVersionCommand{
		ProductID: fixture.product.ID,
		VersionID: scheduled.ID,
	})
	assertAppErrorKind(t, err, ErrorKindConflict)

	futureAt := time.Now().Add(time.Hour)
	future, err := fixture.productService.CreateProductVersion(fixture.ctx, CreateProductVersionCommand{
		ProductID:   fixture.product.ID,
		VersionCode: "2.0.0",
		ReleaseDate: &futureAt,
		Method:      ReleaseScheduled,
	})
	if err != nil {
		t.Fatalf("create future version: %v", err)
	}
	err = fixture.productService.SetMinSupportedVersion(fixture.ctx, UpdateMinVersionCommand{
		ProductID: fixture.product.ID,
		VersionID: future.ID,
	})
	assertAppErrorKind(t, err, ErrorKindConflict)

	var product model.Product
	if err := fixture.db.Where("id = ?", fixture.product.ID).First(&product).Error; err != nil {
		t.Fatalf("get product: %v", err)
	}
	if product.MinSupportedVersionID == nil {
		t.Fatal("expected min supported version to be set")
	}
	err = fixture.productService.DeprecateVersion(fixture.ctx, DeprecateVersionCommand{
		ProductID: fixture.product.ID,
		VersionID: *product.MinSupportedVersionID,
	})
	assertAppErrorKind(t, err, ErrorKindConflict)

	err = fixture.productService.DeprecateVersion(fixture.ctx, DeprecateVersionCommand{
		ProductID: fixture.product.ID,
		VersionID: scheduled.ID,
	})
	if err != nil {
		t.Fatalf("deprecate non-min version: %v", err)
	}
	data, err := fixture.productService.GetProductDataByID(fixture.ctx, fixture.product.ID)
	if err != nil {
		t.Fatalf("get product data after deprecate: %v", err)
	}
	foundDeprecated := false
	for _, version := range data.Versions {
		if version.ID == scheduled.ID && version.Status == int(entity.VersionStatusDeprecated) {
			foundDeprecated = true
			break
		}
	}
	if !foundDeprecated {
		t.Fatalf("deprecated version should still appear in product versions: %#v", data.Versions)
	}

	_, err = fixture.accessService.Register(fixture.ctx, AccessCommand{
		DeviceCode:  "deprecated-version-node",
		LicenseKey:  fixture.license.LicenseKey,
		ProductID:   fixture.product.ID,
		VersionCode: scheduled.VersionCode,
	})
	assertAppErrorKind(t, err, ErrorKindBadRequest)
	bound := fixture.register(t, "deprecated-version-heartbeat-node")
	_, err = fixture.accessService.Heartbeat(
		fixture.ctx,
		"deprecated-version-heartbeat-node",
		fixture.product.ID,
		scheduled.VersionCode,
		fixture.license.LicenseKey,
	)
	assertAppErrorKind(t, err, ErrorKindBadRequest)
	if bound.NodeID == 0 {
		t.Fatal("registered node should have id")
	}
	err = fixture.productService.ReleaseVersion(fixture.ctx, ReleaseNewVersionCommand{
		ProductID: fixture.product.ID,
		VersionID: scheduled.ID,
	})
	assertAppErrorKind(t, err, ErrorKindConflict)

	err = fixture.productService.DeleteProductVersion(fixture.ctx, DeleteProductVersionCommand{
		ProductID: fixture.product.ID,
		VersionID: scheduled.ID,
	})
	if err != nil {
		t.Fatalf("delete deprecated version: %v", err)
	}
	data, err = fixture.productService.GetProductDataByID(fixture.ctx, fixture.product.ID)
	if err != nil {
		t.Fatalf("get product data after delete version: %v", err)
	}
	for _, version := range data.Versions {
		if version.ID == scheduled.ID {
			t.Fatalf("deleted version should not appear in product versions: %#v", data.Versions)
		}
	}
}

func TestP2LicenseRenewAndCleanupPolicy(t *testing.T) {
	fixture := newFlowFixture(t, 1, 0, 24)
	register := fixture.register(t, "license-policy-node")

	expiredAt := time.Now().Add(-time.Hour)
	if err := fixture.db.Model(&model.License{}).Where("id = ?", register.LicenseID).Updates(map[string]interface{}{
		"expired_at": expiredAt,
		"status":     int(entity.StatusExpired),
	}).Error; err != nil {
		t.Fatalf("expire license: %v", err)
	}
	if err := fixture.licenseService.RenewLicense(fixture.ctx, RenewLicenseCommand{
		ID:         register.LicenseID,
		ExtraHours: 2,
	}); err != nil {
		t.Fatalf("renew expired license: %v", err)
	}
	renewed, err := fixture.licenseService.GetLicenseDataByID(fixture.ctx, register.LicenseID)
	if err != nil {
		t.Fatalf("get renewed license: %v", err)
	}
	if renewed.Status != int(entity.StatusActive) {
		t.Fatalf("expired license should become active after positive renew, got %d", renewed.Status)
	}

	if err := fixture.licenseService.RevokeLicense(fixture.ctx, register.LicenseID); err != nil {
		t.Fatalf("revoke license: %v", err)
	}
	err = fixture.licenseService.RenewLicense(fixture.ctx, RenewLicenseCommand{
		ID:         register.LicenseID,
		ExtraHours: 1,
	})
	assertAppErrorKind(t, err, ErrorKindForbidden)

	if err := fixture.db.Create(&model.LicenseServiceScope{
		LicenseID:         register.LicenseID,
		ServiceIdentifier: "restart_process",
		Status:            1,
	}).Error; err != nil {
		t.Fatalf("create license service scope: %v", err)
	}
	if err := fixture.licenseService.DeleteLicense(fixture.ctx, register.LicenseID); err != nil {
		t.Fatalf("delete license: %v", err)
	}
	var scopeCount int64
	if err := fixture.db.Model(&model.LicenseServiceScope{}).
		Where("license_id = ?", register.LicenseID).
		Count(&scopeCount).Error; err != nil {
		t.Fatalf("count license scopes: %v", err)
	}
	if scopeCount != 0 {
		t.Fatalf("license scopes should be cleaned, got %d", scopeCount)
	}
}

func TestP2NodeBanReasonAndDeleteNotFound(t *testing.T) {
	fixture := newFlowFixture(t, 1, 0, 24)
	register := fixture.register(t, "ban-reason-node")

	reason := "abuse"
	if err := fixture.nodeService.BanNode(fixture.ctx, UpdateNodeStatusCommand{
		NodeID: register.NodeID,
		Reason: &reason,
	}); err != nil {
		t.Fatalf("ban node: %v", err)
	}

	var node model.Node
	if err := fixture.db.Where("id = ?", register.NodeID).First(&node).Error; err != nil {
		t.Fatalf("get banned node: %v", err)
	}
	if node.BanReason == nil || *node.BanReason != reason || node.BannedAt == nil || node.OfflineAt == nil {
		t.Fatalf("ban metadata mismatch: %#v", node)
	}

	if err := fixture.nodeService.UnbanNode(fixture.ctx, UpdateNodeStatusCommand{NodeID: register.NodeID}); err != nil {
		t.Fatalf("unban node: %v", err)
	}
	var unbanned model.Node
	if err := fixture.db.Where("id = ?", register.NodeID).First(&unbanned).Error; err != nil {
		t.Fatalf("get unbanned node: %v", err)
	}
	if unbanned.BanReason != nil || unbanned.BannedAt != nil {
		t.Fatalf("unban should clear ban metadata: %#v", unbanned)
	}

	err := fixture.nodeService.DeleteNode(fixture.ctx, 404)
	assertAppErrorKind(t, err, ErrorKindNotFound)

	logs, err := NewAuditService().ListAuditLogs(fixture.ctx, ListAuditLogsCommand{
		ResourceType: ptrString("node"),
		ResourceID:   &register.NodeID,
		Limit:        20,
	})
	if err != nil {
		t.Fatalf("list node audit logs: %v", err)
	}
	foundBan := false
	for _, log := range logs {
		if log.Action == "ban" && strings.Contains(string(log.Data), reason) {
			foundBan = true
			break
		}
	}
	if !foundBan {
		t.Fatalf("expected ban audit log with reason, got %#v", logs)
	}
}
