package service

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"nexus-core/global"
	"nexus-core/monitor"
	"nexus-core/persistence/base"
	"nexus-core/persistence/model"

	"github.com/glebarez/sqlite"
	"github.com/gorilla/websocket"
	"gorm.io/gorm"
)

func setupControlFlowTest(t *testing.T) context.Context {
	t.Helper()
	oldDB := global.DB
	oldStat := monitor.GlobalStat
	oldMonitor := monitor.GlobalMonitor
	t.Cleanup(func() {
		global.DB = oldDB
		monitor.GlobalStat = oldStat
		monitor.GlobalMonitor = oldMonitor
	})

	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "control-flow.db")), &gorm.Config{})
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

func TestControlFlowHTTPDispatch(t *testing.T) {
	ctx := setupControlFlowTest(t)

	productService := NewProductService()
	licenseService := NewLicenseService()
	nodeService := NewNodeService()
	accessService := NewAccessService(licenseService, nodeService, productService)
	controlService := NewControlService()

	product, err := productService.CreateProduct(ctx, CreateProductCommand{Name: "control-flow-product"})
	if err != nil {
		t.Fatalf("create product: %v", err)
	}
	if _, err := productService.CreateProductVersion(ctx, CreateProductVersionCommand{
		ProductID:   product.ID,
		VersionCode: "1.0.0",
		Method:      ReleaseImmediate,
	}); err != nil {
		t.Fatalf("create product version: %v", err)
	}
	license, err := licenseService.CreateLicense(ctx, CreateLicenseCommand{
		ProductID:     product.ID,
		ValidityHours: 24,
		MaxNodes:      1,
	})
	if err != nil {
		t.Fatalf("create license: %v", err)
	}
	register, err := accessService.Register(ctx, AccessCommand{
		DeviceCode:  "control-node",
		LicenseKey:  license.LicenseKey,
		ProductID:   product.ID,
		VersionCode: "1.0.0",
	})
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	if _, err := accessService.Heartbeat(ctx, "control-node", product.ID, "1.0.0", license.LicenseKey); err != nil {
		t.Fatalf("heartbeat: %v", err)
	}

	_, err = controlService.CreateControlService(ctx, CreateControlServiceCommand{
		ProductID:    &product.ID,
		Identifier:   "restart_process",
		Name:         "Restart Process",
		ServiceType:  "command",
		InputSchema:  json.RawMessage(`{"type":"object"}`),
		OutputSchema: json.RawMessage(`{"type":"object"}`),
	})
	if err != nil {
		t.Fatalf("create control service: %v", err)
	}

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

	_, err = controlService.ReportNodeCapability(ctx, ReportNodeCapabilityCommand{
		NodeID:            register.NodeID,
		ServiceIdentifier: "restart_process",
		Protocol:          "http",
		Endpoint:          &nodeServer.URL,
		Schema: json.RawMessage(`{
			"fields": {
				"proc": {"source": "process_name", "type": "string", "required": true},
				"delay": {"source": "delay_seconds", "type": "integer", "default": 0}
			}
		}`),
	})
	if err != nil {
		t.Fatalf("report node capability: %v", err)
	}

	command, err := controlService.CreateControlCommand(ctx, CreateControlCommand{
		NodeID:            register.NodeID,
		ServiceIdentifier: "restart_process",
		Payload:           json.RawMessage(`{"process_name":"worker","delay_seconds":"3"}`),
	})
	if err != nil {
		t.Fatalf("create control command: %v", err)
	}
	if command.Status != ControlCommandStatusSuccess {
		t.Fatalf("command should succeed, got status %d error %v", command.Status, command.ErrorMessage)
	}
	if received["proc"] != "worker" {
		t.Fatalf("converted payload proc mismatch: %#v", received)
	}
	if received["delay"].(float64) != 3 {
		t.Fatalf("converted payload delay mismatch: %#v", received)
	}

	got, err := controlService.GetControlCommandByID(ctx, command.ID)
	if err != nil {
		t.Fatalf("get command: %v", err)
	}
	if got.Status != ControlCommandStatusSuccess {
		t.Fatalf("stored command should succeed, got %d", got.Status)
	}
}

func TestControlFlowMQTTDispatch(t *testing.T) {
	ctx := setupControlFlowTest(t)
	_, nodeID := prepareControlFlowTarget(t, ctx)

	fakePublisher := &fakeMQTTPublisher{}
	oldPublisher := DefaultMQTTPublisher
	DefaultMQTTPublisher = fakePublisher
	t.Cleanup(func() { DefaultMQTTPublisher = oldPublisher })

	controlService := NewControlService()
	topic := "nodes/control-node/restart"
	_, err := controlService.ReportNodeCapability(ctx, ReportNodeCapabilityCommand{
		NodeID:            nodeID,
		ServiceIdentifier: "restart_process",
		Protocol:          "mqtt",
		Endpoint:          &topic,
		Schema: json.RawMessage(`{
			"fields": {
				"proc": {"source": "process_name", "type": "string", "required": true}
			}
		}`),
	})
	if err != nil {
		t.Fatalf("report mqtt node capability: %v", err)
	}

	command, err := controlService.CreateControlCommand(ctx, CreateControlCommand{
		NodeID:            nodeID,
		ServiceIdentifier: "restart_process",
		Payload:           json.RawMessage(`{"process_name":"worker"}`),
	})
	if err != nil {
		t.Fatalf("create mqtt control command: %v", err)
	}
	if command.Status != ControlCommandStatusSent {
		t.Fatalf("mqtt command should be sent, got status %d error %v", command.Status, command.ErrorMessage)
	}
	if fakePublisher.topic != topic {
		t.Fatalf("mqtt topic mismatch: %s", fakePublisher.topic)
	}

	var message ControlDispatchMessage
	if err := json.Unmarshal(fakePublisher.payload, &message); err != nil {
		t.Fatalf("decode mqtt message: %v", err)
	}
	if message.CommandID != command.ID || message.NodeID != nodeID || message.ServiceIdentifier != "restart_process" {
		t.Fatalf("mqtt message mismatch: %#v", message)
	}
	var payload map[string]string
	if err := json.Unmarshal(message.Payload, &payload); err != nil {
		t.Fatalf("decode mqtt payload: %v", err)
	}
	if payload["proc"] != "worker" {
		t.Fatalf("converted mqtt payload mismatch: %#v", payload)
	}
}

func TestControlFlowWebSocketDispatch(t *testing.T) {
	ctx := setupControlFlowTest(t)
	_, nodeID := prepareControlFlowTarget(t, ctx)

	hub := NewControlWebSocketHub()
	oldHub := DefaultControlWebSocketHub
	DefaultControlWebSocketHub = hub
	t.Cleanup(func() { DefaultControlWebSocketHub = oldHub })

	wsServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := hub.ServeHTTP(w, r, nodeID); err != nil {
			t.Logf("serve websocket: %v", err)
		}
	}))
	defer wsServer.Close()

	wsURL := "ws" + strings.TrimPrefix(wsServer.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial websocket: %v", err)
	}
	defer conn.Close()

	nodeDone := make(chan error, 1)
	go func() {
		var message ControlDispatchMessage
		if err := conn.ReadJSON(&message); err != nil {
			nodeDone <- err
			return
		}
		nodeDone <- conn.WriteJSON(ControlCommandResponse{
			CommandID: message.CommandID,
			Status:    "success",
			Result:    json.RawMessage(`{"applied":true}`),
		})
	}()

	controlService := NewControlService()
	_, err = controlService.ReportNodeCapability(ctx, ReportNodeCapabilityCommand{
		NodeID:            nodeID,
		ServiceIdentifier: "restart_process",
		Protocol:          "websocket",
		Schema: json.RawMessage(`{
			"fields": {
				"proc": {"source": "process_name", "type": "string", "required": true}
			}
		}`),
	})
	if err != nil {
		t.Fatalf("report websocket node capability: %v", err)
	}

	command, err := controlService.CreateControlCommand(ctx, CreateControlCommand{
		NodeID:            nodeID,
		ServiceIdentifier: "restart_process",
		Payload:           json.RawMessage(`{"process_name":"worker"}`),
	})
	if err != nil {
		t.Fatalf("create websocket control command: %v", err)
	}
	if err := <-nodeDone; err != nil {
		t.Fatalf("node websocket exchange: %v", err)
	}
	if command.Status != ControlCommandStatusSuccess {
		t.Fatalf("websocket command should succeed, got status %d error %v", command.Status, command.ErrorMessage)
	}
}

func TestControlServiceManagementLifecycle(t *testing.T) {
	ctx := setupControlFlowTest(t)
	productID, nodeID := prepareControlFlowTarget(t, ctx)
	controlService := NewControlService()

	services, err := controlService.ListControlServices(ctx, &productID)
	if err != nil {
		t.Fatalf("list control services: %v", err)
	}
	if len(services) != 1 {
		t.Fatalf("expected one control service, got %d", len(services))
	}

	name := "Restart Worker"
	updated, err := controlService.UpdateControlService(ctx, UpdateControlServiceCommand{
		ID:   services[0].ID,
		Name: &name,
		InputSchema: json.RawMessage(`{
			"fields": {
				"process_name": {"type": "string", "required": true, "min_length": 3}
			}
		}`),
	})
	if err != nil {
		t.Fatalf("update control service: %v", err)
	}
	if updated.Name != name {
		t.Fatalf("updated name mismatch: %s", updated.Name)
	}

	disabled, err := controlService.UpdateControlServiceStatus(ctx, UpdateControlServiceStatusCommand{
		ID:     services[0].ID,
		Status: ControlServiceStatusDisabled,
	})
	if err != nil {
		t.Fatalf("disable control service: %v", err)
	}
	if disabled.Status != ControlServiceStatusDisabled {
		t.Fatalf("service should be disabled, got %d", disabled.Status)
	}

	topic := "nodes/control-node/service-management"
	_, err = controlService.ReportNodeCapability(ctx, ReportNodeCapabilityCommand{
		NodeID:            nodeID,
		ServiceIdentifier: "restart_process",
		Protocol:          "mqtt",
		Endpoint:          &topic,
		Schema:            json.RawMessage(`{"fields":{"proc":{"source":"process_name","type":"string","required":true}}}`),
	})
	assertAppError(t, err, ErrorKindNotFound)

	if _, err := controlService.UpdateControlServiceStatus(ctx, UpdateControlServiceStatusCommand{
		ID:     services[0].ID,
		Status: ControlServiceStatusEnabled,
	}); err != nil {
		t.Fatalf("enable control service: %v", err)
	}

	if _, err := controlService.ReportNodeCapability(ctx, ReportNodeCapabilityCommand{
		NodeID:            nodeID,
		ServiceIdentifier: "restart_process",
		Protocol:          "mqtt",
		Endpoint:          &topic,
		Schema:            json.RawMessage(`{"fields":{"proc":{"source":"process_name","type":"string","required":true}}}`),
	}); err != nil {
		t.Fatalf("report node capability: %v", err)
	}

	err = controlService.DeleteControlService(ctx, services[0].ID)
	assertAppError(t, err, ErrorKindConflict)

	unused, err := controlService.CreateControlService(ctx, CreateControlServiceCommand{
		Identifier:  "unused_service",
		Name:        "Unused Service",
		ServiceType: "query",
	})
	if err != nil {
		t.Fatalf("create unused service: %v", err)
	}
	if err := controlService.DeleteControlService(ctx, unused.ID); err != nil {
		t.Fatalf("delete unused service: %v", err)
	}
}

func TestControlSchemaConstraints(t *testing.T) {
	schema := json.RawMessage(`{
		"fields": {
			"mode": {"type": "string", "required": true, "enum": ["safe", "fast"]},
			"delay": {"source": "delay_seconds", "type": "integer", "minimum": 1, "maximum": 10},
			"email": {"type": "string", "format": "email"},
			"trace": {"type": "string", "pattern": "^[a-z]{3}-[0-9]{3}$"}
		}
	}`)

	converted, err := ConvertPayload(json.RawMessage(`{
		"mode": "safe",
		"delay_seconds": "3",
		"email": "ops@example.com",
		"trace": "abc-123"
	}`), schema)
	if err != nil {
		t.Fatalf("convert constrained payload: %v", err)
	}
	var data map[string]interface{}
	if err := json.Unmarshal(converted, &data); err != nil {
		t.Fatalf("decode converted payload: %v", err)
	}
	if data["mode"] != "safe" || int(data["delay"].(float64)) != 3 {
		t.Fatalf("converted constrained payload mismatch: %#v", data)
	}

	_, err = ConvertPayload(json.RawMessage(`{
		"mode": "turbo",
		"delay_seconds": "12",
		"email": "invalid",
		"trace": "bad"
	}`), schema)
	assertAppError(t, err, ErrorKindBadRequest)
}

func TestControlCommandRequiresOnlineNode(t *testing.T) {
	ctx := setupControlFlowTest(t)
	productService := NewProductService()
	licenseService := NewLicenseService()
	nodeService := NewNodeService()
	accessService := NewAccessService(licenseService, nodeService, productService)
	controlService := NewControlService()

	product, err := productService.CreateProduct(ctx, CreateProductCommand{Name: "offline-control-product"})
	if err != nil {
		t.Fatalf("create product: %v", err)
	}
	if _, err := productService.CreateProductVersion(ctx, CreateProductVersionCommand{
		ProductID:   product.ID,
		VersionCode: "1.0.0",
		Method:      ReleaseImmediate,
	}); err != nil {
		t.Fatalf("create product version: %v", err)
	}
	license, err := licenseService.CreateLicense(ctx, CreateLicenseCommand{
		ProductID:     product.ID,
		ValidityHours: 24,
		MaxNodes:      1,
	})
	if err != nil {
		t.Fatalf("create license: %v", err)
	}
	register, err := accessService.Register(ctx, AccessCommand{
		DeviceCode:  "offline-control-node",
		LicenseKey:  license.LicenseKey,
		ProductID:   product.ID,
		VersionCode: "1.0.0",
	})
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	_, err = controlService.CreateControlService(ctx, CreateControlServiceCommand{
		ProductID:   &product.ID,
		Identifier:  "restart_process",
		Name:        "Restart Process",
		ServiceType: "command",
	})
	if err != nil {
		t.Fatalf("create control service: %v", err)
	}
	topic := "nodes/offline-control-node/restart"
	if _, err := controlService.ReportNodeCapability(ctx, ReportNodeCapabilityCommand{
		NodeID:            register.NodeID,
		ServiceIdentifier: "restart_process",
		Protocol:          "mqtt",
		Endpoint:          &topic,
		Schema:            json.RawMessage(`{"fields":{"proc":{"source":"process_name","type":"string","required":true}}}`),
	}); err != nil {
		t.Fatalf("report node capability: %v", err)
	}

	_, err = controlService.CreateControlCommand(ctx, CreateControlCommand{
		NodeID:            register.NodeID,
		ServiceIdentifier: "restart_process",
		Payload:           json.RawMessage(`{"process_name":"worker"}`),
	})
	assertAppError(t, err, ErrorKindConflict)
}

func TestControlCommandLicenseServiceScope(t *testing.T) {
	ctx := setupControlFlowTest(t)
	_, nodeID := prepareControlFlowTarget(t, ctx)
	controlService := NewControlService()

	var binding model.NodeLicenseBinding
	if err := global.DB.WithContext(ctx).Where("node_id = ?", nodeID).First(&binding).Error; err != nil {
		t.Fatalf("get node binding: %v", err)
	}
	if err := global.DB.WithContext(ctx).Create(&model.LicenseServiceScope{
		LicenseID:         binding.LicenseID,
		ServiceIdentifier: "other_service",
		Status:            1,
	}).Error; err != nil {
		t.Fatalf("create disallowed scope: %v", err)
	}

	topic := "nodes/control-node/scope"
	if _, err := controlService.ReportNodeCapability(ctx, ReportNodeCapabilityCommand{
		NodeID:            nodeID,
		ServiceIdentifier: "restart_process",
		Protocol:          "mqtt",
		Endpoint:          &topic,
		Schema:            json.RawMessage(`{"fields":{"proc":{"source":"process_name","type":"string","required":true}}}`),
	}); err != nil {
		t.Fatalf("report node capability: %v", err)
	}
	_, err := controlService.CreateControlCommand(ctx, CreateControlCommand{
		NodeID:            nodeID,
		ServiceIdentifier: "restart_process",
		Payload:           json.RawMessage(`{"process_name":"worker"}`),
	})
	assertAppError(t, err, ErrorKindForbidden)

	if err := global.DB.WithContext(ctx).Create(&model.LicenseServiceScope{
		LicenseID:         binding.LicenseID,
		ServiceIdentifier: "restart_process",
		Status:            1,
	}).Error; err != nil {
		t.Fatalf("create allowed scope: %v", err)
	}

	fakePublisher := &fakeMQTTPublisher{}
	oldPublisher := DefaultMQTTPublisher
	DefaultMQTTPublisher = fakePublisher
	t.Cleanup(func() { DefaultMQTTPublisher = oldPublisher })

	command, err := controlService.CreateControlCommand(ctx, CreateControlCommand{
		NodeID:            nodeID,
		ServiceIdentifier: "restart_process",
		Payload:           json.RawMessage(`{"process_name":"worker"}`),
	})
	if err != nil {
		t.Fatalf("create scoped control command: %v", err)
	}
	if command.Status != ControlCommandStatusSent {
		t.Fatalf("scoped command should be sent, got %d", command.Status)
	}
}

func TestCompleteControlCommandAndHeartbeatPendingSummary(t *testing.T) {
	ctx := setupControlFlowTest(t)
	productID, nodeID := prepareControlFlowTarget(t, ctx)
	controlService := NewControlService()

	services, err := controlService.ListControlServices(ctx, &productID)
	if err != nil {
		t.Fatalf("list control services: %v", err)
	}
	if len(services) != 1 {
		t.Fatalf("expected one control service, got %d", len(services))
	}
	if _, err := controlService.UpdateControlService(ctx, UpdateControlServiceCommand{
		ID: services[0].ID,
		OutputSchema: json.RawMessage(`{
			"fields": {
				"done": {"source": "applied", "type": "boolean", "required": true}
			}
		}`),
	}); err != nil {
		t.Fatalf("update output schema: %v", err)
	}

	topic := "nodes/control-node/async"
	if _, err := controlService.ReportNodeCapability(ctx, ReportNodeCapabilityCommand{
		NodeID:            nodeID,
		ServiceIdentifier: "restart_process",
		Protocol:          "mqtt",
		Endpoint:          &topic,
		Schema:            json.RawMessage(`{"fields":{"proc":{"source":"process_name","type":"string","required":true}}}`),
	}); err != nil {
		t.Fatalf("report node capability: %v", err)
	}

	fakePublisher := &fakeMQTTPublisher{}
	oldPublisher := DefaultMQTTPublisher
	DefaultMQTTPublisher = fakePublisher
	t.Cleanup(func() { DefaultMQTTPublisher = oldPublisher })

	command, err := controlService.CreateControlCommand(ctx, CreateControlCommand{
		NodeID:            nodeID,
		ServiceIdentifier: "restart_process",
		Payload:           json.RawMessage(`{"process_name":"worker"}`),
	})
	if err != nil {
		t.Fatalf("create async command: %v", err)
	}
	if command.Status != ControlCommandStatusSent {
		t.Fatalf("async command should be sent, got %d", command.Status)
	}

	var node model.Node
	if err := global.DB.WithContext(ctx).Where("id = ?", nodeID).First(&node).Error; err != nil {
		t.Fatalf("get node: %v", err)
	}
	var binding model.NodeLicenseBinding
	if err := global.DB.WithContext(ctx).Where("node_id = ?", nodeID).First(&binding).Error; err != nil {
		t.Fatalf("get binding: %v", err)
	}
	var license model.License
	if err := global.DB.WithContext(ctx).Where("id = ?", binding.LicenseID).First(&license).Error; err != nil {
		t.Fatalf("get license: %v", err)
	}
	pending, err := NewAccessService(NewLicenseService(), NewNodeService(), NewProductService()).
		Heartbeat(ctx, node.DeviceCode, productID, "1.0.0", license.LicenseKey)
	if err != nil {
		t.Fatalf("heartbeat pending summary: %v", err)
	}
	if pending.PendingControl == nil || pending.PendingControl.Count != 1 || len(pending.PendingControl.CommandIDs) != 1 || pending.PendingControl.CommandIDs[0] != command.ID {
		t.Fatalf("pending summary mismatch: %#v", pending.PendingControl)
	}

	completed, err := controlService.CompleteControlCommand(ctx, CompleteControlCommandCommand{
		CommandID: command.ID,
		Status:    "success",
		Result:    json.RawMessage(`{"applied":true}`),
	})
	if err != nil {
		t.Fatalf("complete command: %v", err)
	}
	if completed.Status != ControlCommandStatusSuccess {
		t.Fatalf("completed command should succeed, got %d", completed.Status)
	}
	var result map[string]bool
	if err := json.Unmarshal(completed.Result, &result); err != nil {
		t.Fatalf("decode converted result: %v", err)
	}
	if !result["done"] {
		t.Fatalf("converted result mismatch: %#v", result)
	}
}

func TestConvertPayloadValidation(t *testing.T) {
	_, err := ConvertPayload(
		json.RawMessage(`{"delay_seconds": 1}`),
		json.RawMessage(`{"fields":{"proc":{"source":"process_name","type":"string","required":true}}}`),
	)
	assertAppError(t, err, ErrorKindBadRequest)

	converted, err := ConvertPayload(
		json.RawMessage(`{"delay_seconds":"5"}`),
		json.RawMessage(`{"fields":{"delay":{"source":"delay_seconds","type":"integer","default":0}}}`),
	)
	if err != nil {
		t.Fatalf("convert payload: %v", err)
	}
	var data map[string]int
	if err := json.Unmarshal(converted, &data); err != nil {
		t.Fatalf("decode converted: %v", err)
	}
	if data["delay"] != 5 {
		t.Fatalf("expected delay 5, got %d", data["delay"])
	}
}

type fakeMQTTPublisher struct {
	topic   string
	payload []byte
}

func (f *fakeMQTTPublisher) Publish(ctx context.Context, topic string, payload []byte) error {
	f.topic = topic
	f.payload = append([]byte(nil), payload...)
	return nil
}

func prepareControlFlowTarget(t *testing.T, ctx context.Context) (uint, uint) {
	t.Helper()

	productService := NewProductService()
	licenseService := NewLicenseService()
	nodeService := NewNodeService()
	accessService := NewAccessService(licenseService, nodeService, productService)
	controlService := NewControlService()

	product, err := productService.CreateProduct(ctx, CreateProductCommand{Name: "control-flow-product-shared"})
	if err != nil {
		t.Fatalf("create product: %v", err)
	}
	if _, err := productService.CreateProductVersion(ctx, CreateProductVersionCommand{
		ProductID:   product.ID,
		VersionCode: "1.0.0",
		Method:      ReleaseImmediate,
	}); err != nil {
		t.Fatalf("create product version: %v", err)
	}
	license, err := licenseService.CreateLicense(ctx, CreateLicenseCommand{
		ProductID:     product.ID,
		ValidityHours: 24,
		MaxNodes:      1,
	})
	if err != nil {
		t.Fatalf("create license: %v", err)
	}
	register, err := accessService.Register(ctx, AccessCommand{
		DeviceCode:  "control-node-shared",
		LicenseKey:  license.LicenseKey,
		ProductID:   product.ID,
		VersionCode: "1.0.0",
	})
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	if _, err := accessService.Heartbeat(ctx, "control-node-shared", product.ID, "1.0.0", license.LicenseKey); err != nil {
		t.Fatalf("heartbeat: %v", err)
	}

	_, err = controlService.CreateControlService(ctx, CreateControlServiceCommand{
		ProductID:    &product.ID,
		Identifier:   "restart_process",
		Name:         "Restart Process",
		ServiceType:  "command",
		InputSchema:  json.RawMessage(`{"type":"object"}`),
		OutputSchema: json.RawMessage(`{"type":"object"}`),
	})
	if err != nil {
		t.Fatalf("create control service: %v", err)
	}

	return product.ID, register.NodeID
}
