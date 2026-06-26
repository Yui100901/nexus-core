package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"nexus-core/domain/entity"
	"sync"
	"testing"

	"github.com/gin-gonic/gin"
)

type simpleProductScenario struct {
	router     http.Handler
	productID  uint
	licenseID  uint
	licenseKey string
	nodes      map[string]uint
}

func newSimpleProductScenario(t *testing.T, maxNodes int, maxConcurrent int) *simpleProductScenario {
	t.Helper()
	gin.SetMode(gin.TestMode)
	setupControlAPITest(t)

	router := NewServer()
	NewProductController().RegisterRoutes(router)
	NewLicenseController().RegisterRoutes(router)
	NewNodeController().RegisterRoutes(router)
	NewAccessController().RegisterRoutes(router)
	NewMonitorController().RegisterRoutes(router)

	product := doJSON(t, router, http.MethodPost, "/products", map[string]interface{}{
		"name":        "simple-product-api",
		"description": "simple product scenario test",
	})
	productID := uint(product.Data.(map[string]interface{})["id"].(float64))

	version := doJSON(t, router, http.MethodPost, "/products/versions", map[string]interface{}{
		"product_id":     productID,
		"version_code":   "1.0.0",
		"release_method": 0,
	})
	if version.Code != CodeOK {
		t.Fatalf("create version code = %d body = %#v", version.Code, version)
	}

	license := doJSON(t, router, http.MethodPost, "/licenses", map[string]interface{}{
		"product_id":     productID,
		"validity_hours": 24,
		"max_nodes":      maxNodes,
		"max_concurrent": maxConcurrent,
		"remark":         "simple product scenario license",
	})
	licenseData := license.Data.(map[string]interface{})

	return &simpleProductScenario{
		router:     router,
		productID:  productID,
		licenseID:  uint(licenseData["id"].(float64)),
		licenseKey: licenseData["license_key"].(string),
		nodes:      map[string]uint{},
	}
}

func (s *simpleProductScenario) register(t *testing.T, deviceCode string) map[string]interface{} {
	t.Helper()
	response := doJSON(t, s.router, http.MethodPost, "/access/register", map[string]interface{}{
		"device_code":  deviceCode,
		"license_key":  s.licenseKey,
		"product_id":   s.productID,
		"version_code": "1.0.0",
	})
	data := response.Data.(map[string]interface{})
	s.nodes[deviceCode] = uint(data["node_id"].(float64))
	return data
}

func (s *simpleProductScenario) heartbeatRaw(deviceCode string) (int, CommonResponse, error) {
	return doJSONRaw(s.router, http.MethodPost, "/access/heartbeat", map[string]interface{}{
		"device_code":  deviceCode,
		"license_key":  s.licenseKey,
		"product_id":   s.productID,
		"version_code": "1.0.0",
	})
}

func TestSimpleProductMultiBindingHeartbeatAPI(t *testing.T) {
	scenario := newSimpleProductScenario(t, 3, 3)

	devices := []string{"simple-node-a", "simple-node-b", "simple-node-c"}
	for i, device := range devices {
		register := scenario.register(t, device)
		if register["binding_established"] != true {
			t.Fatalf("register %s should establish binding: %#v", device, register)
		}
		if got := int(register["current_node_count"].(float64)); got != i+1 {
			t.Fatalf("current node count after %s = %d, want %d", device, got, i+1)
		}
	}

	const rounds = 4
	var wg sync.WaitGroup
	errCh := make(chan error, len(devices)*rounds)
	for round := 0; round < rounds; round++ {
		for _, device := range devices {
			wg.Add(1)
			go func(deviceCode string) {
				defer wg.Done()
				status, response, err := scenario.heartbeatRaw(deviceCode)
				if err != nil {
					errCh <- err
					return
				}
				if status != http.StatusOK || response.Code != CodeOK {
					errCh <- testingErrorf("heartbeat %s status=%d response=%#v", deviceCode, status, response)
					return
				}
				data := response.Data.(map[string]interface{})
				if data["online"] != true {
					errCh <- testingErrorf("heartbeat %s should be online: %#v", deviceCode, data)
				}
			}(device)
		}
	}
	wg.Wait()
	close(errCh)
	for err := range errCh {
		t.Fatal(err)
	}

	online := doJSON(t, scenario.router, http.MethodGet, "/monitor/online", nil)
	onlineData := online.Data.(map[string]interface{})
	if got := int(onlineData["total_online"].(float64)); got != 3 {
		t.Fatalf("total online = %d, want 3: %#v", got, onlineData)
	}

	heartbeats := doJSON(t, scenario.router, http.MethodGet, "/monitor/nodes/heartbeats?page=1&page_size=10", nil)
	if rows := heartbeats.Data.([]interface{}); len(rows) < 3 {
		t.Fatalf("heartbeat list should include at least 3 nodes, got %d", len(rows))
	}

	nodes := doJSON(t, scenario.router, http.MethodGet, "/nodes?page=1&page_size=10", nil)
	if rows := nodes.Data.([]interface{}); len(rows) != 3 {
		t.Fatalf("node list length = %d, want 3", len(rows))
	}
}

func TestSimpleProductBindingAndConcurrentLimitsAPI(t *testing.T) {
	scenario := newSimpleProductScenario(t, 3, 2)

	for _, device := range []string{"limit-node-a", "limit-node-b", "limit-node-c"} {
		scenario.register(t, device)
	}

	status, response, err := doJSONRaw(scenario.router, http.MethodPost, "/access/register", map[string]interface{}{
		"device_code":  "limit-node-d",
		"license_key":  scenario.licenseKey,
		"product_id":   scenario.productID,
		"version_code": "1.0.0",
	})
	if err != nil {
		t.Fatalf("register over max nodes response decode failed: %v", err)
	}
	if status != http.StatusConflict || response.Code != CodeConflict {
		t.Fatalf("fourth binding should be rejected, status=%d response=%#v", status, response)
	}

	for _, device := range []string{"limit-node-a", "limit-node-b"} {
		status, response, err := scenario.heartbeatRaw(device)
		if err != nil {
			t.Fatalf("heartbeat %s decode failed: %v", device, err)
		}
		if status != http.StatusOK || response.Code != CodeOK {
			t.Fatalf("heartbeat %s should succeed, status=%d response=%#v", device, status, response)
		}
	}

	status, response, err = scenario.heartbeatRaw("limit-node-c")
	if err != nil {
		t.Fatalf("third concurrent heartbeat response decode failed: %v", err)
	}
	if status != http.StatusConflict || response.Code != CodeConflict {
		t.Fatalf("third concurrent heartbeat should be rejected, status=%d response=%#v", status, response)
	}

	unbind := doJSON(t, scenario.router, http.MethodDelete, "/node-bindings", map[string]interface{}{
		"node_id":    scenario.nodes["limit-node-c"],
		"license_id": scenario.licenseID,
	})
	if unbind.Code != CodeOK {
		t.Fatalf("unbind should succeed: %#v", unbind)
	}

	registerAfterUnbind := doJSON(t, scenario.router, http.MethodPost, "/access/register", map[string]interface{}{
		"device_code":  "limit-node-d",
		"license_key":  scenario.licenseKey,
		"product_id":   scenario.productID,
		"version_code": "1.0.0",
	})
	if registerAfterUnbind.Code != CodeOK {
		t.Fatalf("register after unbind should succeed: %#v", registerAfterUnbind)
	}

	licenses := doJSON(t, scenario.router, http.MethodGet, "/licenses?product_id="+uintString(scenario.productID)+"&page=1&page_size=10", nil)
	if rows := licenses.Data.([]interface{}); len(rows) != 1 {
		t.Fatalf("license list length = %d, want 1", len(rows))
	}
}

func TestForceOfflineRejectsHeartbeatAndRegisterRestoresNodeAPI(t *testing.T) {
	scenario := newSimpleProductScenario(t, 1, 1)
	const deviceCode = "force-offline-node"

	register := scenario.register(t, deviceCode)
	nodeID := uint(register["node_id"].(float64))

	status, response, err := scenario.heartbeatRaw(deviceCode)
	if err != nil {
		t.Fatalf("heartbeat response decode failed: %v", err)
	}
	if status != http.StatusOK || response.Code != CodeOK {
		t.Fatalf("heartbeat before force offline should succeed, status=%d response=%#v", status, response)
	}

	forced := doJSON(t, scenario.router, http.MethodPost, "/nodes/"+uintString(nodeID)+"/force-offline", map[string]interface{}{
		"reason": "operator test",
	})
	if forced.Code != CodeOK {
		t.Fatalf("force offline should succeed: %#v", forced)
	}

	node := doJSON(t, scenario.router, http.MethodGet, "/nodes/"+uintString(nodeID), nil)
	nodeData := node.Data.(map[string]interface{})
	if got := int(nodeData["status"].(float64)); got != entity.NodeStatusForcedOffline {
		t.Fatalf("node status = %d, want forced offline: %#v", got, nodeData)
	}

	online := doJSON(t, scenario.router, http.MethodGet, "/monitor/online", nil)
	onlineData := online.Data.(map[string]interface{})
	if got := int(onlineData["total_online"].(float64)); got != 0 {
		t.Fatalf("total online after force offline = %d, want 0: %#v", got, onlineData)
	}

	status, response, err = scenario.heartbeatRaw(deviceCode)
	if err != nil {
		t.Fatalf("heartbeat after force offline response decode failed: %v", err)
	}
	if status != http.StatusForbidden || response.Code != CodeForbidden {
		t.Fatalf("heartbeat after force offline should be rejected, status=%d response=%#v", status, response)
	}

	restored := doJSON(t, scenario.router, http.MethodPost, "/access/register", map[string]interface{}{
		"device_code":  deviceCode,
		"license_key":  scenario.licenseKey,
		"product_id":   scenario.productID,
		"version_code": "1.0.0",
	})
	restoredData := restored.Data.(map[string]interface{})
	if got := uint(restoredData["node_id"].(float64)); got != nodeID {
		t.Fatalf("register should restore same node id = %d, want %d: %#v", got, nodeID, restoredData)
	}

	node = doJSON(t, scenario.router, http.MethodGet, "/nodes/"+uintString(nodeID), nil)
	nodeData = node.Data.(map[string]interface{})
	if got := int(nodeData["status"].(float64)); got != entity.NodeStatusNormal {
		t.Fatalf("node status after register = %d, want normal: %#v", got, nodeData)
	}

	status, response, err = scenario.heartbeatRaw(deviceCode)
	if err != nil {
		t.Fatalf("heartbeat after register response decode failed: %v", err)
	}
	if status != http.StatusOK || response.Code != CodeOK {
		t.Fatalf("heartbeat after register should succeed, status=%d response=%#v", status, response)
	}
}

func doJSONRaw(router http.Handler, method string, path string, payload interface{}) (int, CommonResponse, error) {
	var body []byte
	if payload != nil {
		var err error
		body, err = json.Marshal(payload)
		if err != nil {
			return 0, CommonResponse{}, err
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
		return recorder.Code, CommonResponse{}, err
	}
	return recorder.Code, response, nil
}

type testingError string

func (e testingError) Error() string {
	return string(e)
}

func testingErrorf(format string, args ...interface{}) error {
	return testingError(fmt.Sprintf(format, args...))
}
