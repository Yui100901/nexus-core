package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type commonResponse struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

type productData struct {
	ID   uint   `json:"id"`
	Name string `json:"name"`
}

type versionData struct {
	ID          uint   `json:"id"`
	VersionCode string `json:"version_code"`
}

type licenseData struct {
	ID         uint   `json:"id"`
	LicenseKey string `json:"license_key"`
}

type registerResult struct {
	NodeID uint `json:"node_id"`
}

type controlServiceData struct {
	ID         uint   `json:"id"`
	Identifier string `json:"identifier"`
}

type controlCommandData struct {
	ID               uint            `json:"id"`
	Status           int             `json:"status"`
	ConvertedPayload json.RawMessage `json:"converted_payload"`
	Result           json.RawMessage `json:"result"`
	ErrorMessage     *string         `json:"error_message"`
}

type dispatchMessage struct {
	CommandID         uint            `json:"command_id"`
	NodeID            uint            `json:"node_id"`
	ServiceIdentifier string          `json:"service_identifier"`
	Payload           json.RawMessage `json:"payload"`
}

type client struct {
	baseURL string
	http    *http.Client
}

func main() {
	baseURL := flag.String("server", "http://localhost:8080", "nexus-core server base URL")
	deviceCode := flag.String("device", "protocol-demo-node-001", "demo node device code")
	httpListen := flag.String("http-listen", "127.0.0.1:0", "local HTTP node listen address")
	flag.Parse()

	c := client{
		baseURL: strings.TrimRight(*baseURL, "/"),
		http:    &http.Client{Timeout: 10 * time.Second},
	}

	if err := run(c, *deviceCode, *httpListen); err != nil {
		fmt.Printf("protocol demo failed: %v\n", err)
		return
	}
	fmt.Println("protocol demo completed")
}

func run(c client, deviceCode string, httpListen string) error {
	suffix := time.Now().Format("20060102150405")
	productName := "protocol-demo-product-" + suffix
	versionCode := "1.0.0"

	product, err := postJSON[productData](c, "/products", map[string]interface{}{
		"name":        productName,
		"description": "protocol conversion demo product",
	})
	if err != nil {
		return fmt.Errorf("create product: %w", err)
	}
	fmt.Printf("product: id=%d name=%s\n", product.ID, product.Name)

	version, err := postJSON[versionData](c, "/products/versions", map[string]interface{}{
		"product_id":     product.ID,
		"version_code":   versionCode,
		"release_method": 0,
	})
	if err != nil {
		return fmt.Errorf("create version: %w", err)
	}
	fmt.Printf("version: id=%d code=%s\n", version.ID, version.VersionCode)

	license, err := postJSON[licenseData](c, "/licenses", map[string]interface{}{
		"product_id":     product.ID,
		"validity_hours": 24,
		"max_nodes":      1,
		"max_concurrent": 0,
		"remark":         "protocol conversion demo license",
	})
	if err != nil {
		return fmt.Errorf("create license: %w", err)
	}
	fmt.Printf("license: id=%d key=%s\n", license.ID, license.LicenseKey)

	register, err := postJSON[registerResult](c, "/access/register", map[string]interface{}{
		"device_code":  deviceCode,
		"license_key":  license.LicenseKey,
		"product_id":   product.ID,
		"version_code": versionCode,
	})
	if err != nil {
		return fmt.Errorf("register node: %w", err)
	}
	if _, err := postJSON[json.RawMessage](c, "/access/heartbeat", map[string]interface{}{
		"device_code":  deviceCode,
		"license_key":  license.LicenseKey,
		"product_id":   product.ID,
		"version_code": versionCode,
	}); err != nil {
		return fmt.Errorf("heartbeat: %w", err)
	}
	fmt.Printf("node: id=%d device=%s\n", register.NodeID, deviceCode)

	httpIdentifier := "demo_http_config_" + suffix
	if err := runHTTPConversion(c, product.ID, register.NodeID, httpIdentifier, httpListen); err != nil {
		return err
	}

	wsIdentifier := "demo_ws_config_" + suffix
	if err := runWebSocketConversion(c, product.ID, register.NodeID, wsIdentifier); err != nil {
		return err
	}

	return nil
}

func runHTTPConversion(c client, productID uint, nodeID uint, identifier string, listenAddr string) error {
	serviceDef, err := createControlService(c, productID, identifier, "HTTP Config Conversion")
	if err != nil {
		return err
	}

	received := make(chan map[string]interface{}, 1)
	server, endpoint, err := startHTTPNode(listenAddr, received)
	if err != nil {
		return err
	}
	defer server.Close()
	fmt.Printf("HTTP node endpoint: %s\n", endpoint)

	if _, err := postJSON[json.RawMessage](c, "/node-capabilities", map[string]interface{}{
		"node_id":            nodeID,
		"service_identifier": serviceDef.Identifier,
		"protocol":           "http",
		"endpoint":           endpoint,
		"schema": map[string]interface{}{
			"fields": httpNodeFields(),
		},
	}); err != nil {
		return fmt.Errorf("report HTTP capability: %w", err)
	}

	command, err := postJSON[controlCommandData](c, "/control-commands", map[string]interface{}{
		"node_id":            nodeID,
		"service_identifier": serviceDef.Identifier,
		"payload": map[string]interface{}{
			"process_name":  "worker_api",
			"delay_seconds": "7",
			"dry_run":       "true",
			"mode":          "fast",
			"email":         "ops@example.com",
			"tags":          []string{"blue", "canary"},
			"metadata":      map[string]interface{}{"region": "cn"},
		},
	})
	if err != nil {
		return fmt.Errorf("create HTTP command: %w", err)
	}
	if command.Status != 3 {
		dump, _ := json.Marshal(command)
		return fmt.Errorf("HTTP command status=%d error=%v command=%s", command.Status, command.ErrorMessage, dump)
	}

	select {
	case payload := <-received:
		if err := assertHTTPConvertedPayload(payload); err != nil {
			return err
		}
	case <-time.After(5 * time.Second):
		return fmt.Errorf("HTTP node did not receive command")
	}
	if err := assertConvertedResult(command.Result, "http-trace"); err != nil {
		return fmt.Errorf("HTTP output conversion: %w", err)
	}

	_, err = postJSON[controlCommandData](c, "/control-commands", map[string]interface{}{
		"node_id":            nodeID,
		"service_identifier": serviceDef.Identifier,
		"payload": map[string]interface{}{
			"process_name":  "x",
			"delay_seconds": 100,
			"mode":          "turbo",
			"email":         "not-email",
		},
	})
	if err == nil {
		return fmt.Errorf("expected invalid HTTP payload conversion to fail")
	}

	fmt.Println("HTTP conversion: ok")
	return nil
}

func runWebSocketConversion(c client, productID uint, nodeID uint, identifier string) error {
	serviceDef, err := createControlService(c, productID, identifier, "WebSocket Config Conversion")
	if err != nil {
		return err
	}
	if _, err := postJSON[json.RawMessage](c, "/node-capabilities", map[string]interface{}{
		"node_id":            nodeID,
		"service_identifier": serviceDef.Identifier,
		"protocol":           "websocket",
		"schema": map[string]interface{}{
			"fields": map[string]interface{}{
				"cfg_key":   map[string]interface{}{"source": "config_key", "type": "string", "required": true, "pattern": "^[a-z.]+$"},
				"cfg_value": map[string]interface{}{"source": "value", "type": "string", "required": true},
				"reload":    map[string]interface{}{"source": "reload", "type": "boolean", "default": true},
				"ratio":     map[string]interface{}{"source": "ratio", "type": "number", "minimum": 0, "maximum": 1},
			},
		},
	}); err != nil {
		return fmt.Errorf("report WebSocket capability: %w", err)
	}

	wsURL, err := nodeControlWebSocketURL(c.baseURL, nodeID)
	if err != nil {
		return err
	}
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		return fmt.Errorf("connect websocket node: %w", err)
	}
	defer conn.Close()

	var wg sync.WaitGroup
	errCh := make(chan error, 1)
	wg.Add(1)
	go func() {
		defer wg.Done()
		var message dispatchMessage
		if err := conn.ReadJSON(&message); err != nil {
			errCh <- fmt.Errorf("read websocket command: %w", err)
			return
		}
		if message.ServiceIdentifier != serviceDef.Identifier {
			errCh <- fmt.Errorf("unexpected websocket service: %s", message.ServiceIdentifier)
			return
		}
		var payload map[string]interface{}
		if err := json.Unmarshal(message.Payload, &payload); err != nil {
			errCh <- fmt.Errorf("decode websocket payload: %w", err)
			return
		}
		if err := assertWebSocketConvertedPayload(payload); err != nil {
			errCh <- err
			return
		}
		errCh <- conn.WriteJSON(map[string]interface{}{
			"command_id": message.CommandID,
			"status":     "success",
			"result": map[string]interface{}{
				"ok":       true,
				"delay_ms": 0,
				"trace_id": "ws-trace",
			},
		})
	}()

	command, err := postJSON[controlCommandData](c, "/control-commands", map[string]interface{}{
		"node_id":            nodeID,
		"service_identifier": serviceDef.Identifier,
		"payload": map[string]interface{}{
			"config_key": "runtime.mode",
			"value":      "safe",
			"reload":     "false",
			"ratio":      "0.75",
		},
	})
	if err != nil {
		return fmt.Errorf("create WebSocket command: %w", err)
	}
	if err := <-errCh; err != nil {
		return err
	}
	wg.Wait()
	if command.Status != 3 {
		return fmt.Errorf("WebSocket command status=%d error=%v", command.Status, command.ErrorMessage)
	}
	if err := assertConvertedResult(command.Result, "ws-trace"); err != nil {
		return fmt.Errorf("WebSocket output conversion: %w", err)
	}

	fmt.Println("WebSocket conversion: ok")
	return nil
}

func createControlService(c client, productID uint, identifier string, name string) (*controlServiceData, error) {
	serviceDef, err := postJSON[controlServiceData](c, "/control-services", map[string]interface{}{
		"product_id":   productID,
		"identifier":   identifier,
		"name":         name,
		"service_type": "command",
		"input_schema": map[string]interface{}{"type": "object"},
		"output_schema": map[string]interface{}{
			"fields": map[string]interface{}{
				"accepted":      map[string]interface{}{"source": "ok", "type": "boolean", "required": true},
				"applied_delay": map[string]interface{}{"source": "delay_ms", "type": "integer", "default": 0},
				"trace":         map[string]interface{}{"source": "trace_id", "type": "string", "required": true},
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("create control service %s: %w", identifier, err)
	}
	return &serviceDef, nil
}

func httpNodeFields() map[string]interface{} {
	return map[string]interface{}{
		"target_proc":  map[string]interface{}{"source": "process_name", "type": "string", "required": true, "min_length": 3, "max_length": 32, "pattern": "^[a-zA-Z0-9_-]+$"},
		"delay":        map[string]interface{}{"source": "delay_seconds", "type": "integer", "default": 5, "minimum": 0, "maximum": 60},
		"dry_run":      map[string]interface{}{"source": "dry_run", "type": "boolean", "default": false},
		"mode_code":    map[string]interface{}{"source": "mode", "type": "string", "enum": []string{"safe", "fast"}, "default": "safe"},
		"notify_email": map[string]interface{}{"source": "email", "type": "string", "format": "email"},
		"tags":         map[string]interface{}{"source": "tags", "type": "array", "default": []string{}},
		"metadata":     map[string]interface{}{"source": "metadata", "type": "object", "default": map[string]interface{}{}},
	}
}

func startHTTPNode(listenAddr string, received chan<- map[string]interface{}) (*http.Server, string, error) {
	mux := http.NewServeMux()
	mux.HandleFunc("/control/config", func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		var payload map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		received <- payload
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"ok":       true,
			"delay_ms": 250,
			"trace_id": "http-trace",
		})
	})

	listener, err := net.Listen("tcp", listenAddr)
	if err != nil {
		return nil, "", fmt.Errorf("listen HTTP node: %w", err)
	}
	server := &http.Server{Handler: mux}
	go func() {
		_ = server.Serve(listener)
	}()
	return server, "http://" + listener.Addr().String() + "/control/config", nil
}

func assertHTTPConvertedPayload(payload map[string]interface{}) error {
	checks := map[string]interface{}{
		"target_proc":  "worker_api",
		"delay":        float64(7),
		"dry_run":      true,
		"mode_code":    "fast",
		"notify_email": "ops@example.com",
	}
	for key, expected := range checks {
		if payload[key] != expected {
			return fmt.Errorf("HTTP payload %s expected %#v got %#v full=%#v", key, expected, payload[key], payload)
		}
	}
	if _, ok := payload["tags"].([]interface{}); !ok {
		return fmt.Errorf("HTTP payload tags should be array: %#v", payload)
	}
	if _, ok := payload["metadata"].(map[string]interface{}); !ok {
		return fmt.Errorf("HTTP payload metadata should be object: %#v", payload)
	}
	return nil
}

func assertWebSocketConvertedPayload(payload map[string]interface{}) error {
	checks := map[string]interface{}{
		"cfg_key":   "runtime.mode",
		"cfg_value": "safe",
		"reload":    false,
		"ratio":     float64(0.75),
	}
	for key, expected := range checks {
		if payload[key] != expected {
			return fmt.Errorf("WebSocket payload %s expected %#v got %#v full=%#v", key, expected, payload[key], payload)
		}
	}
	return nil
}

func assertConvertedResult(raw json.RawMessage, expectedTrace string) error {
	var result map[string]interface{}
	if err := json.Unmarshal(raw, &result); err != nil {
		return err
	}
	if result["accepted"] != true {
		return fmt.Errorf("accepted should be true: %#v", result)
	}
	if result["trace"] != expectedTrace {
		return fmt.Errorf("trace expected %s got %#v", expectedTrace, result["trace"])
	}
	if _, ok := result["applied_delay"].(float64); !ok {
		return fmt.Errorf("applied_delay should be number: %#v", result)
	}
	return nil
}

func nodeControlWebSocketURL(baseURL string, nodeID uint) (string, error) {
	parsed, err := url.Parse(baseURL)
	if err != nil {
		return "", err
	}
	switch parsed.Scheme {
	case "http":
		parsed.Scheme = "ws"
	case "https":
		parsed.Scheme = "wss"
	default:
		return "", fmt.Errorf("unsupported server scheme %s", parsed.Scheme)
	}
	parsed.Path = "/node-control/ws"
	parsed.RawQuery = fmt.Sprintf("node_id=%d", nodeID)
	return parsed.String(), nil
}

func postJSON[T any](c client, path string, payload interface{}) (T, error) {
	var zero T
	body, err := json.Marshal(payload)
	if err != nil {
		return zero, err
	}
	req, err := http.NewRequest(http.MethodPost, c.baseURL+path, bytes.NewReader(body))
	if err != nil {
		return zero, err
	}
	req.Header.Set("Content-Type", "application/json")
	return doJSON[T](c, req)
}

func doJSON[T any](c client, req *http.Request) (T, error) {
	var zero T
	resp, err := c.http.Do(req)
	if err != nil {
		return zero, err
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return zero, err
	}
	var common commonResponse
	if err := json.Unmarshal(raw, &common); err != nil {
		return zero, fmt.Errorf("invalid response status=%d body=%s: %w", resp.StatusCode, string(raw), err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 || common.Code < 200 || common.Code >= 300 {
		return zero, fmt.Errorf("status=%d code=%d message=%s body=%s", resp.StatusCode, common.Code, common.Message, string(raw))
	}
	if len(common.Data) == 0 || string(common.Data) == "null" || string(common.Data) == `""` {
		return zero, nil
	}

	var result T
	if err := json.Unmarshal(common.Data, &result); err != nil {
		return zero, fmt.Errorf("decode data: %w", err)
	}
	return result, nil
}
