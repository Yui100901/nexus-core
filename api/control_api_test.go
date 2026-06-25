package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"testing"

	"nexus-core/domain/service"
	"nexus-core/global"
	"nexus-core/monitor"
	"nexus-core/persistence/base"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupControlAPITest(t *testing.T) context.Context {
	t.Helper()
	oldDB := global.DB
	oldStat := monitor.GlobalStat
	oldMonitor := monitor.GlobalMonitor
	t.Cleanup(func() {
		global.DB = oldDB
		monitor.GlobalStat = oldStat
		monitor.GlobalMonitor = oldMonitor
	})

	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "control-api.db")), &gorm.Config{})
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
	return context.Background()
}

func TestControlAPIHTTPFlow(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := setupControlAPITest(t)
	nodeID, productID := seedControlAPITarget(t, ctx)

	router := NewServer()
	NewControlController().RegisterRoutes(router)

	var received map[string]interface{}
	nodeServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
			t.Fatalf("decode node request: %v", err)
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"ok": true,
		})
	}))
	defer nodeServer.Close()

	control := doJSON(t, router, http.MethodPost, "/control-services", map[string]interface{}{
		"product_id":    productID,
		"identifier":    "restart_process",
		"name":          "Restart Process",
		"service_type":  "command",
		"input_schema":  map[string]interface{}{"type": "object"},
		"output_schema": map[string]interface{}{"type": "object"},
	})
	if control.Code != CodeOK {
		t.Fatalf("create control service code = %d body = %#v", control.Code, control)
	}

	capability := doJSON(t, router, http.MethodPost, "/node-capabilities", map[string]interface{}{
		"node_id":            nodeID,
		"service_identifier": "restart_process",
		"protocol":           "http",
		"endpoint":           nodeServer.URL,
		"schema": map[string]interface{}{
			"fields": map[string]interface{}{
				"proc":  map[string]interface{}{"source": "process_name", "type": "string", "required": true},
				"delay": map[string]interface{}{"source": "delay_seconds", "type": "integer", "default": 0},
			},
		},
	})
	if capability.Code != CodeOK {
		t.Fatalf("report capability code = %d body = %#v", capability.Code, capability)
	}

	command := doJSON(t, router, http.MethodPost, "/control-commands", map[string]interface{}{
		"node_id":            nodeID,
		"service_identifier": "restart_process",
		"payload": map[string]interface{}{
			"process_name":  "worker",
			"delay_seconds": "3",
		},
	})
	if command.Code != CodeOK {
		t.Fatalf("create command code = %d body = %#v", command.Code, command)
	}
	data := command.Data.(map[string]interface{})
	if int(data["status"].(float64)) != service.ControlCommandStatusSuccess {
		t.Fatalf("command status mismatch: %#v", data)
	}
	if received["proc"] != "worker" || int(received["delay"].(float64)) != 3 {
		t.Fatalf("converted node payload mismatch: %#v", received)
	}

	commandID := uint(data["id"].(float64))
	got := doJSON(t, router, http.MethodGet, "/control-commands/"+uintString(commandID), nil)
	if got.Code != CodeOK {
		t.Fatalf("get command code = %d body = %#v", got.Code, got)
	}
	gotData := got.Data.(map[string]interface{})
	if int(gotData["status"].(float64)) != service.ControlCommandStatusSuccess {
		t.Fatalf("stored command status mismatch: %#v", gotData)
	}
}

func TestControlAPIManageAndCompleteCommand(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := setupControlAPITest(t)
	nodeID, productID := seedControlAPITarget(t, ctx)

	router := NewServer()
	NewControlController().RegisterRoutes(router)

	fakePublisher := &apiFakeMQTTPublisher{}
	oldPublisher := service.DefaultMQTTPublisher
	service.DefaultMQTTPublisher = fakePublisher
	t.Cleanup(func() { service.DefaultMQTTPublisher = oldPublisher })

	control := doJSON(t, router, http.MethodPost, "/control-services", map[string]interface{}{
		"product_id":   productID,
		"identifier":   "restart_process",
		"name":         "Restart Process",
		"service_type": "command",
		"output_schema": map[string]interface{}{
			"fields": map[string]interface{}{
				"done": map[string]interface{}{"source": "applied", "type": "boolean", "required": true},
			},
		},
	})
	if control.Code != CodeOK {
		t.Fatalf("create control service code = %d body = %#v", control.Code, control)
	}
	controlID := uint(control.Data.(map[string]interface{})["id"].(float64))

	updated := doJSON(t, router, http.MethodPatch, "/control-services/"+uintString(controlID), map[string]interface{}{
		"name": "Restart Worker",
	})
	if updated.Code != CodeOK {
		t.Fatalf("update control service code = %d body = %#v", updated.Code, updated)
	}
	if updated.Data.(map[string]interface{})["name"] != "Restart Worker" {
		t.Fatalf("updated name mismatch: %#v", updated.Data)
	}

	disabled := doJSON(t, router, http.MethodPost, "/control-services/"+uintString(controlID)+"/status", map[string]interface{}{
		"status": service.ControlServiceStatusDisabled,
	})
	if disabled.Code != CodeOK {
		t.Fatalf("disable control service code = %d body = %#v", disabled.Code, disabled)
	}
	enabled := doJSON(t, router, http.MethodPost, "/control-services/"+uintString(controlID)+"/status", map[string]interface{}{
		"status": service.ControlServiceStatusEnabled,
	})
	if enabled.Code != CodeOK {
		t.Fatalf("enable control service code = %d body = %#v", enabled.Code, enabled)
	}

	capability := doJSON(t, router, http.MethodPost, "/node-capabilities", map[string]interface{}{
		"node_id":            nodeID,
		"service_identifier": "restart_process",
		"protocol":           "mqtt",
		"endpoint":           "nodes/control-api-node/restart",
		"schema": map[string]interface{}{
			"fields": map[string]interface{}{
				"proc": map[string]interface{}{"source": "process_name", "type": "string", "required": true},
			},
		},
	})
	if capability.Code != CodeOK {
		t.Fatalf("report capability code = %d body = %#v", capability.Code, capability)
	}

	command := doJSON(t, router, http.MethodPost, "/control-commands", map[string]interface{}{
		"node_id":            nodeID,
		"service_identifier": "restart_process",
		"payload": map[string]interface{}{
			"process_name": "worker",
		},
	})
	if command.Code != CodeOK {
		t.Fatalf("create command code = %d body = %#v", command.Code, command)
	}
	commandID := uint(command.Data.(map[string]interface{})["id"].(float64))
	if fakePublisher.topic != "nodes/control-api-node/restart" {
		t.Fatalf("mqtt topic mismatch: %s", fakePublisher.topic)
	}

	completed := doJSON(t, router, http.MethodPost, "/control-commands/"+uintString(commandID)+"/complete", map[string]interface{}{
		"status": "success",
		"result": map[string]interface{}{"applied": true},
	})
	if completed.Code != CodeOK {
		t.Fatalf("complete command code = %d body = %#v", completed.Code, completed)
	}
	completedData := completed.Data.(map[string]interface{})
	if int(completedData["status"].(float64)) != service.ControlCommandStatusSuccess {
		t.Fatalf("completed status mismatch: %#v", completedData)
	}
	result := completedData["result"].(map[string]interface{})
	if result["done"] != true {
		t.Fatalf("converted result mismatch: %#v", result)
	}

	unused := doJSON(t, router, http.MethodPost, "/control-services", map[string]interface{}{
		"identifier":   "unused_service",
		"name":         "Unused Service",
		"service_type": "query",
	})
	if unused.Code != CodeOK {
		t.Fatalf("create unused service code = %d body = %#v", unused.Code, unused)
	}
	unusedID := uint(unused.Data.(map[string]interface{})["id"].(float64))
	deleted := doJSON(t, router, http.MethodDelete, "/control-services/"+uintString(unusedID), nil)
	if deleted.Code != CodeOK {
		t.Fatalf("delete unused service code = %d body = %#v", deleted.Code, deleted)
	}
}

func seedControlAPITarget(t *testing.T, ctx context.Context) (uint, uint) {
	t.Helper()

	productService := service.NewProductService()
	licenseService := service.NewLicenseService()
	nodeService := service.NewNodeService()
	accessService := service.NewAccessService(licenseService, nodeService, productService)

	product, err := productService.CreateProduct(ctx, service.CreateProductCommand{Name: "control-api-product"})
	if err != nil {
		t.Fatalf("create product: %v", err)
	}
	if _, err := productService.CreateProductVersion(ctx, service.CreateProductVersionCommand{
		ProductID:   product.ID,
		VersionCode: "1.0.0",
		Method:      service.ReleaseImmediate,
	}); err != nil {
		t.Fatalf("create product version: %v", err)
	}
	license, err := licenseService.CreateLicense(ctx, service.CreateLicenseCommand{
		ProductID:     product.ID,
		ValidityHours: 24,
		MaxNodes:      1,
	})
	if err != nil {
		t.Fatalf("create license: %v", err)
	}
	register, err := accessService.Register(ctx, service.AccessCommand{
		DeviceCode:  "control-api-node",
		LicenseKey:  license.LicenseKey,
		ProductID:   product.ID,
		VersionCode: "1.0.0",
	})
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	if _, err := accessService.Heartbeat(ctx, "control-api-node", product.ID, "1.0.0", license.LicenseKey); err != nil {
		t.Fatalf("heartbeat: %v", err)
	}
	return register.NodeID, product.ID
}

func doJSON(t *testing.T, router http.Handler, method string, path string, payload interface{}) CommonResponse {
	t.Helper()

	var body []byte
	if payload != nil {
		var err error
		body, err = json.Marshal(payload)
		if err != nil {
			t.Fatalf("marshal payload: %v", err)
		}
	}
	req := httptest.NewRequest(method, path, bytes.NewReader(body))
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	var response CommonResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response %s %s status %d body %s: %v", method, path, recorder.Code, recorder.Body.String(), err)
	}
	if recorder.Code < 200 || recorder.Code >= 300 {
		t.Fatalf("%s %s http status %d response %#v", method, path, recorder.Code, response)
	}
	return response
}

func uintString(value uint) string {
	return strconv.FormatUint(uint64(value), 10)
}

type apiFakeMQTTPublisher struct {
	topic   string
	payload []byte
}

func (f *apiFakeMQTTPublisher) Publish(ctx context.Context, topic string, payload []byte) error {
	f.topic = topic
	f.payload = append([]byte(nil), payload...)
	return nil
}
