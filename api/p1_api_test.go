package api

import (
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestP1AccessLicenseNodeAndMonitorAPI(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setupControlAPITest(t)

	router := NewServer()
	NewProductController().RegisterRoutes(router)
	NewLicenseController().RegisterRoutes(router)
	NewNodeController().RegisterRoutes(router)
	NewAccessController().RegisterRoutes(router)
	NewMonitorController().RegisterRoutes(router)

	product := doJSON(t, router, http.MethodPost, "/products", map[string]interface{}{
		"name": "p1-api-product",
	})
	productID := uint(product.Data.(map[string]interface{})["id"].(float64))

	version := doJSON(t, router, http.MethodPost, "/products/versions", map[string]interface{}{
		"product_id":   productID,
		"version_code": "1.0.0",
		"method":       0,
	})
	if version.Code != CodeOK {
		t.Fatalf("create version code = %d body = %#v", version.Code, version)
	}

	license := doJSON(t, router, http.MethodPost, "/licenses", map[string]interface{}{
		"product_id":     productID,
		"validity_hours": 24,
		"max_nodes":      1,
		"max_concurrent": 1,
	})
	licenseData := license.Data.(map[string]interface{})
	licenseID := uint(licenseData["id"].(float64))
	licenseKey := licenseData["license_key"].(string)

	register := doJSON(t, router, http.MethodPost, "/access/register", map[string]interface{}{
		"device_code":  "p1-api-node",
		"license_key":  licenseKey,
		"product_id":   productID,
		"version_code": "1.0.0",
	})
	registerData := register.Data.(map[string]interface{})
	nodeID := uint(registerData["node_id"].(float64))
	if nodeID == 0 {
		t.Fatalf("register should return node id: %#v", registerData)
	}

	heartbeat := doJSON(t, router, http.MethodPost, "/access/heartbeat", map[string]interface{}{
		"device_code":  "p1-api-node",
		"license_key":  licenseKey,
		"product_id":   productID,
		"version_code": "1.0.0",
	})
	heartbeatData := heartbeat.Data.(map[string]interface{})
	if heartbeatData["online"] != true {
		t.Fatalf("heartbeat should mark node online: %#v", heartbeatData)
	}
	if heartbeatData["pending_control"] == nil {
		t.Fatalf("heartbeat should include pending control summary: %#v", heartbeatData)
	}

	gotLicense := doJSON(t, router, http.MethodGet, "/license-keys/"+licenseKey, nil)
	if gotLicense.Code != CodeOK {
		t.Fatalf("get license by key code = %d body = %#v", gotLicense.Code, gotLicense)
	}

	updatedLicense := doJSON(t, router, http.MethodPatch, "/licenses/"+uintString(licenseID), map[string]interface{}{
		"max_nodes":      2,
		"max_concurrent": 2,
		"feature_mask":   "control",
	})
	if updatedLicense.Code != CodeOK {
		t.Fatalf("update license code = %d body = %#v", updatedLicense.Code, updatedLicense)
	}

	revoked := doJSON(t, router, http.MethodPost, "/licenses/"+uintString(licenseID)+"/revoke", map[string]interface{}{})
	if revoked.Code != CodeOK {
		t.Fatalf("revoke license code = %d body = %#v", revoked.Code, revoked)
	}
	restored := doJSON(t, router, http.MethodPost, "/licenses/"+uintString(licenseID)+"/restore", map[string]interface{}{})
	if restored.Code != CodeOK {
		t.Fatalf("restore license code = %d body = %#v", restored.Code, restored)
	}
	renewed := doJSON(t, router, http.MethodPost, "/licenses/"+uintString(licenseID)+"/renew", map[string]interface{}{
		"extra_hours": 1,
	})
	if renewed.Code != CodeOK {
		t.Fatalf("renew license code = %d body = %#v", renewed.Code, renewed)
	}

	node := doJSON(t, router, http.MethodGet, "/node-devices/p1-api-node", nil)
	if node.Code != CodeOK {
		t.Fatalf("get node by device code = %d body = %#v", node.Code, node)
	}
	updatedNode := doJSON(t, router, http.MethodPatch, "/nodes/"+uintString(nodeID), map[string]interface{}{
		"metadata": `{"os":"windows","channel":"p1"}`,
	})
	if updatedNode.Code != CodeOK {
		t.Fatalf("update node code = %d body = %#v", updatedNode.Code, updatedNode)
	}
	banned := doJSON(t, router, http.MethodPost, "/nodes/"+uintString(nodeID)+"/ban", map[string]interface{}{})
	if banned.Code != CodeOK {
		t.Fatalf("ban node code = %d body = %#v", banned.Code, banned)
	}
	unbanned := doJSON(t, router, http.MethodPost, "/nodes/"+uintString(nodeID)+"/unban", map[string]interface{}{})
	if unbanned.Code != CodeOK {
		t.Fatalf("unban node code = %d body = %#v", unbanned.Code, unbanned)
	}

	heartbeats := doJSON(t, router, http.MethodGet, "/monitor/nodes/heartbeats?page=1&page_size=1", nil)
	heartbeatRows := heartbeats.Data.([]interface{})
	if len(heartbeatRows) != 1 {
		t.Fatalf("heartbeat pagination should return one row, got %d", len(heartbeatRows))
	}

	auditLogs := doJSON(t, router, http.MethodGet, "/audit-logs?resource_type=license&resource_id="+uintString(licenseID)+"&page=1&page_size=1", nil)
	auditRows := auditLogs.Data.([]interface{})
	if len(auditRows) != 1 {
		t.Fatalf("audit pagination should return one row, got %d", len(auditRows))
	}
}
