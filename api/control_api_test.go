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
	"nexus-core/persistence/base"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupControlAPITest(t *testing.T) context.Context {
	t.Helper()
	oldDB := global.DB
	t.Cleanup(func() { global.DB = oldDB })

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
