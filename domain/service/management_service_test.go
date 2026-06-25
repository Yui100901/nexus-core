package service

import (
	"strings"
	"testing"

	"nexus-core/domain/entity"
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
