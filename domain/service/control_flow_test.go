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
	"nexus-core/persistence/base"

	"github.com/glebarez/sqlite"
	"github.com/gorilla/websocket"
	"gorm.io/gorm"
)

func setupControlFlowTest(t *testing.T) context.Context {
	t.Helper()
	oldDB := global.DB
	t.Cleanup(func() { global.DB = oldDB })

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
