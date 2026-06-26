package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os/exec"
	"runtime"
	"strings"
	"time"
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

type shellRequest struct {
	Command string   `json:"cmd"`
	Args    []string `json:"args"`
}

type shellResult struct {
	OK       bool   `json:"ok"`
	Command  string `json:"command"`
	Output   string `json:"output"`
	Error    string `json:"error,omitempty"`
	ExitCode int    `json:"exit_code"`
}

type client struct {
	baseURL string
	http    *http.Client
}

func main() {
	baseURL := flag.String("server", "http://localhost:8080", "nexus-core server base URL")
	deviceCode := flag.String("device", "shell-demo-node-001", "demo node device code")
	httpListen := flag.String("http-listen", "127.0.0.1:0", "local HTTP shell node listen address")
	command := flag.String("command", "echo", "read-only command to test: echo, dir, pwd, whoami")
	args := flag.String("args", "hello from nexus shell demo", "comma-separated command arguments")
	flag.Parse()

	c := client{
		baseURL: strings.TrimRight(*baseURL, "/"),
		http:    &http.Client{Timeout: 10 * time.Second},
	}

	argList := splitArgs(*args)
	if err := run(c, *deviceCode, *httpListen, *command, argList); err != nil {
		fmt.Printf("shell demo failed: %v\n", err)
		return
	}
	fmt.Println("shell demo completed")
}

func run(c client, deviceCode string, httpListen string, testCommand string, testArgs []string) error {
	suffix := time.Now().Format("20060102150405")
	productName := "shell-demo-product-" + suffix
	versionCode := "1.0.0"
	identifier := "run_shell_" + suffix

	product, err := postJSON[productData](c, "/products", map[string]interface{}{
		"name":        productName,
		"description": "read-only shell command demo product",
	})
	if err != nil {
		return fmt.Errorf("create product: %w", err)
	}
	fmt.Printf("product: id=%d name=%s\n", product.ID, product.Name)

	version, err := postJSON[versionData](c, "/products/versions", map[string]interface{}{
		"product_id":     product.ID,
		"version_code":   versionCode,
		"release_method": 0,
		"description":    "shell demo version",
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
		"feature_mask":   identifier,
		"remark":         "shell demo license",
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

	server, endpoint, err := startShellHTTPNode(httpListen)
	if err != nil {
		return err
	}
	defer server.Close()
	fmt.Printf("shell endpoint: %s\n", endpoint)

	serviceDef, err := createShellControlService(c, product.ID, identifier)
	if err != nil {
		return err
	}
	if _, err := postJSON[json.RawMessage](c, "/node-capabilities", map[string]interface{}{
		"node_id":            register.NodeID,
		"service_identifier": serviceDef.Identifier,
		"protocol":           "http",
		"endpoint":           endpoint,
		"schema": map[string]interface{}{
			"fields": map[string]interface{}{
				"cmd": map[string]interface{}{
					"source":   "command",
					"type":     "string",
					"required": true,
					"enum":     []string{"echo", "dir", "pwd", "whoami"},
				},
				"args": map[string]interface{}{
					"source":  "args",
					"type":    "array",
					"default": []string{},
				},
			},
		},
	}); err != nil {
		return fmt.Errorf("report shell capability: %w", err)
	}

	command, err := postJSON[controlCommandData](c, "/control-commands", map[string]interface{}{
		"node_id":            register.NodeID,
		"service_identifier": serviceDef.Identifier,
		"payload": map[string]interface{}{
			"command": strings.TrimSpace(testCommand),
			"args":    testArgs,
		},
	})
	if err != nil {
		return fmt.Errorf("create shell command: %w", err)
	}
	if command.Status != 3 {
		dump, _ := json.Marshal(command)
		return fmt.Errorf("shell command status=%d error=%v command=%s", command.Status, command.ErrorMessage, dump)
	}
	fmt.Printf("command: id=%d status=%d\n", command.ID, command.Status)
	fmt.Printf("result: %s\n", strings.TrimSpace(string(command.Result)))

	return nil
}

func createShellControlService(c client, productID uint, identifier string) (*controlServiceData, error) {
	serviceDef, err := postJSON[controlServiceData](c, "/control-services", map[string]interface{}{
		"product_id":   productID,
		"identifier":   identifier,
		"name":         "Run Read-only Shell Command",
		"description":  "Runs a small whitelist of read-only shell commands.",
		"service_type": "command",
		"input_schema": map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"command": map[string]interface{}{"type": "string"},
				"args":    map[string]interface{}{"type": "array"},
			},
			"required": []string{"command"},
		},
		"output_schema": map[string]interface{}{"type": "object"},
	})
	if err != nil {
		return nil, fmt.Errorf("create shell control service: %w", err)
	}
	return &serviceDef, nil
}

func startShellHTTPNode(listenAddr string) (*http.Server, string, error) {
	mux := http.NewServeMux()
	mux.HandleFunc("/control/shell", func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var req shellRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		result, status := runReadOnlyShellCommand(r.Context(), req)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(result)
	})

	listener, err := net.Listen("tcp", listenAddr)
	if err != nil {
		return nil, "", fmt.Errorf("listen shell node: %w", err)
	}
	server := &http.Server{Handler: mux}
	go func() {
		_ = server.Serve(listener)
	}()
	return server, "http://" + listener.Addr().String() + "/control/shell", nil
}

func runReadOnlyShellCommand(ctx context.Context, req shellRequest) (shellResult, int) {
	command := strings.ToLower(strings.TrimSpace(req.Command))
	if !isAllowedCommand(command) {
		return shellResult{
			OK:       false,
			Command:  command,
			Error:    "command is not allowed; allowed commands: echo, dir, pwd, whoami",
			ExitCode: 1,
		}, http.StatusBadRequest
	}
	if err := validateArgs(command, req.Args); err != nil {
		return shellResult{OK: false, Command: command, Error: err.Error(), ExitCode: 1}, http.StatusBadRequest
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	cmd := buildReadOnlyCommand(timeoutCtx, command, req.Args)
	output, err := cmd.CombinedOutput()
	result := shellResult{
		OK:      err == nil,
		Command: command,
		Output:  string(output),
	}
	if err != nil {
		result.Error = err.Error()
		result.ExitCode = 1
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
		}
		return result, http.StatusInternalServerError
	}
	return result, http.StatusOK
}

func buildReadOnlyCommand(ctx context.Context, command string, args []string) *exec.Cmd {
	switch runtime.GOOS {
	case "windows":
		switch command {
		case "dir":
			return exec.CommandContext(ctx, "cmd", "/C", "dir")
		case "pwd":
			return exec.CommandContext(ctx, "cmd", "/C", "cd")
		case "echo":
			return exec.CommandContext(ctx, "cmd", "/C", "echo "+strings.Join(args, " "))
		default:
			return exec.CommandContext(ctx, command)
		}
	default:
		switch command {
		case "dir":
			return exec.CommandContext(ctx, "ls", "-la", ".")
		case "echo":
			return exec.CommandContext(ctx, "echo", strings.Join(args, " "))
		default:
			return exec.CommandContext(ctx, command)
		}
	}
}

func isAllowedCommand(command string) bool {
	switch command {
	case "echo", "dir", "pwd", "whoami":
		return true
	default:
		return false
	}
}

func validateArgs(command string, args []string) error {
	if command != "echo" && len(args) > 0 {
		return fmt.Errorf("%s does not accept arguments in this demo", command)
	}
	if len(args) > 4 {
		return fmt.Errorf("too many arguments")
	}
	for _, arg := range args {
		if len(arg) > 80 {
			return fmt.Errorf("argument is too long")
		}
		for _, r := range arg {
			if strings.ContainsRune("&|<>^`$\\\n\r", r) {
				return fmt.Errorf("argument contains unsupported shell character")
			}
		}
	}
	return nil
}

func splitArgs(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return []string{}
	}
	parts := strings.Split(raw, ",")
	args := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			args = append(args, trimmed)
		}
	}
	return args
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
