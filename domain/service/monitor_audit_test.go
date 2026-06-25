package service

import (
	"testing"

	"nexus-core/persistence/model"
)

func TestP3MonitorSummaryAndNodeHeartbeat(t *testing.T) {
	fixture := newFlowFixture(t, 2, 0, 24)
	fixture.register(t, "monitor-node-a")
	fixture.register(t, "monitor-node-b")

	if _, err := fixture.accessService.Heartbeat(fixture.ctx, "monitor-node-a", fixture.product.ID, "1.0.0", fixture.license.LicenseKey); err != nil {
		t.Fatalf("heartbeat node a: %v", err)
	}
	if _, err := fixture.accessService.Heartbeat(fixture.ctx, "monitor-node-b", fixture.product.ID, "1.0.0", fixture.license.LicenseKey); err != nil {
		t.Fatalf("heartbeat node b: %v", err)
	}

	monitorService := NewMonitorService()
	summary, err := monitorService.GetOnlineSummary(fixture.ctx)
	if err != nil {
		t.Fatalf("get online summary: %v", err)
	}
	if summary.TotalOnline != 2 {
		t.Fatalf("expected 2 online nodes, got %d: %#v", summary.TotalOnline, summary)
	}
	if len(summary.ByProduct) != 1 || summary.ByProduct[0].Count != 2 {
		t.Fatalf("product online count mismatch: %#v", summary.ByProduct)
	}
	if len(summary.ByLicense) != 1 || summary.ByLicense[0].Count != 2 {
		t.Fatalf("license online count mismatch: %#v", summary.ByLicense)
	}

	heartbeats, err := monitorService.ListNodeHeartbeats(fixture.ctx, 10)
	if err != nil {
		t.Fatalf("list node heartbeats: %v", err)
	}
	if len(heartbeats) < 2 {
		t.Fatalf("expected heartbeat rows, got %d", len(heartbeats))
	}
	if heartbeats[0].LastSeenAt == nil {
		t.Fatalf("expected latest node heartbeat to have last_seen_at: %#v", heartbeats[0])
	}
}

func TestP3AuditLogs(t *testing.T) {
	fixture := newFlowFixture(t, 1, 0, 24)

	if err := fixture.licenseService.RevokeLicense(fixture.ctx, fixture.license.ID); err != nil {
		t.Fatalf("revoke license: %v", err)
	}

	auditService := NewAuditService()
	logs, err := auditService.ListAuditLogs(fixture.ctx, ListAuditLogsCommand{
		ResourceType: ptrString("license"),
		ResourceID:   &fixture.license.ID,
		Limit:        20,
	})
	if err != nil {
		t.Fatalf("list audit logs: %v", err)
	}

	foundRevoke := false
	for _, log := range logs {
		if log.Action == "revoke" {
			foundRevoke = true
			break
		}
	}
	if !foundRevoke {
		var stored []model.AuditLog
		_ = fixture.db.Find(&stored).Error
		t.Fatalf("expected revoke audit log, got %#v", stored)
	}
}

func ptrString(value string) *string {
	return &value
}
