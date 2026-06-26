package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"nexus-core/api"
	"nexus-core/global"
	"nexus-core/monitor"
	"nexus-core/persistence/base"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestShellDemoProductEndToEnd(t *testing.T) {
	server := startTestNexusServer(t)
	defer server.Close()

	c := client{
		baseURL: server.URL,
		http:    server.Client(),
	}

	if err := run(c, "shell-demo-test-node", "127.0.0.1:0", "echo", []string{"nexus", "shell", "demo"}); err != nil {
		t.Fatalf("run shell demo: %v", err)
	}
}

func TestReadOnlyShellCommandGuard(t *testing.T) {
	result, status := runReadOnlyShellCommand(context.Background(), shellRequest{
		Command: "del",
		Args:    []string{"important.txt"},
	})
	if status != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", status)
	}
	if result.OK {
		t.Fatalf("disallowed command should not be ok: %#v", result)
	}

	echo, status := runReadOnlyShellCommand(context.Background(), shellRequest{
		Command: "echo",
		Args:    []string{"readonly"},
	})
	if status != http.StatusOK {
		t.Fatalf("echo status = %d, result=%#v", status, echo)
	}
	if !strings.Contains(echo.Output, "readonly") {
		t.Fatalf("echo output should contain readonly, got %#v", echo.Output)
	}
}

func TestShellDemoDirCommandThroughServer(t *testing.T) {
	server := startTestNexusServer(t)
	defer server.Close()

	c := client{
		baseURL: server.URL,
		http:    server.Client(),
	}

	suffix := "test-dir"
	product, err := postJSON[productData](c, "/products", map[string]interface{}{
		"name": "shell-demo-product-" + suffix,
	})
	if err != nil {
		t.Fatalf("create product: %v", err)
	}
	if _, err := postJSON[versionData](c, "/products/versions", map[string]interface{}{
		"product_id":     product.ID,
		"version_code":   "1.0.0",
		"release_method": 0,
	}); err != nil {
		t.Fatalf("create version: %v", err)
	}
	license, err := postJSON[licenseData](c, "/licenses", map[string]interface{}{
		"product_id":     product.ID,
		"validity_hours": 24,
		"max_nodes":      1,
		"max_concurrent": 0,
		"feature_mask":   "run_shell_dir_test",
	})
	if err != nil {
		t.Fatalf("create license: %v", err)
	}
	register, err := postJSON[registerResult](c, "/access/register", map[string]interface{}{
		"device_code":  "shell-demo-dir-node",
		"license_key":  license.LicenseKey,
		"product_id":   product.ID,
		"version_code": "1.0.0",
	})
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	if _, err := postJSON[json.RawMessage](c, "/access/heartbeat", map[string]interface{}{
		"device_code":  "shell-demo-dir-node",
		"license_key":  license.LicenseKey,
		"product_id":   product.ID,
		"version_code": "1.0.0",
	}); err != nil {
		t.Fatalf("heartbeat: %v", err)
	}

	nodeServer, endpoint, err := startShellHTTPNode("127.0.0.1:0")
	if err != nil {
		t.Fatalf("start shell node: %v", err)
	}
	defer nodeServer.Close()

	serviceDef, err := createShellControlService(c, product.ID, "run_shell_dir_test")
	if err != nil {
		t.Fatalf("create shell service: %v", err)
	}
	if _, err := postJSON[json.RawMessage](c, "/node-capabilities", map[string]interface{}{
		"node_id":            register.NodeID,
		"service_identifier": serviceDef.Identifier,
		"protocol":           "http",
		"endpoint":           endpoint,
		"schema": map[string]interface{}{
			"fields": map[string]interface{}{
				"cmd":  map[string]interface{}{"source": "command", "type": "string", "required": true, "enum": []string{"echo", "dir", "pwd", "whoami"}},
				"args": map[string]interface{}{"source": "args", "type": "array", "default": []string{}},
			},
		},
	}); err != nil {
		t.Fatalf("report capability: %v", err)
	}

	command, err := postJSON[controlCommandData](c, "/control-commands", map[string]interface{}{
		"node_id":            register.NodeID,
		"service_identifier": serviceDef.Identifier,
		"payload": map[string]interface{}{
			"command": "dir",
			"args":    []string{},
		},
	})
	if err != nil {
		t.Fatalf("create command: %v", err)
	}
	if command.Status != 3 {
		t.Fatalf("dir command status=%d error=%v result=%s", command.Status, command.ErrorMessage, command.Result)
	}
	if !strings.Contains(strings.ToLower(string(command.Result)), "output") {
		t.Fatalf("dir command result should include output, got %s", command.Result)
	}
}

func startTestNexusServer(t *testing.T) *httptest.Server {
	t.Helper()
	gin.SetMode(gin.TestMode)

	oldDB := global.DB
	oldStat := monitor.GlobalStat
	oldMonitor := monitor.GlobalMonitor
	t.Cleanup(func() {
		global.DB = oldDB
		monitor.GlobalStat = oldStat
		monitor.GlobalMonitor = oldMonitor
	})

	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "shell-demo.db")), &gorm.Config{})
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

	router := api.NewServer()
	api.NewProductController().RegisterRoutes(router)
	api.NewLicenseController().RegisterRoutes(router)
	api.NewNodeController().RegisterRoutes(router)
	api.NewAccessController().RegisterRoutes(router)
	api.NewControlController().RegisterRoutes(router)
	api.NewMonitorController().RegisterRoutes(router)
	return httptest.NewServer(router)
}
