package service

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"sync"
	"time"

	"nexus-core/global"
	"nexus-core/persistence/model"

	"github.com/gorilla/websocket"
	"gorm.io/gorm"
)

type ControlCommandResponse struct {
	CommandID    uint            `json:"command_id"`
	Status       string          `json:"status"`
	Result       json.RawMessage `json:"result"`
	ErrorMessage *string         `json:"error_message,omitempty"`
}

type ControlWebSocketHub struct {
	mu          sync.RWMutex
	connections map[uint]*controlWebSocketConnection
	upgrader    websocket.Upgrader
}

type controlWebSocketConnection struct {
	nodeID  uint
	conn    *websocket.Conn
	writeMu sync.Mutex
	mu      sync.Mutex
	pending map[uint]chan ControlCommandResponse
}

var DefaultControlWebSocketHub = NewControlWebSocketHub()

func NewControlWebSocketHub() *ControlWebSocketHub {
	return &ControlWebSocketHub{
		connections: make(map[uint]*controlWebSocketConnection),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
	}
}

func (h *ControlWebSocketHub) ServeHTTP(w http.ResponseWriter, r *http.Request, nodeID uint) error {
	if nodeID == 0 {
		return ErrBadRequest("node_id is required")
	}

	wsConn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return WrapInternal("upgrade websocket failed", err)
	}

	conn := &controlWebSocketConnection{
		nodeID:  nodeID,
		conn:    wsConn,
		pending: make(map[uint]chan ControlCommandResponse),
	}
	h.register(conn)
	defer h.unregister(nodeID, conn)
	defer wsConn.Close()

	for {
		var response ControlCommandResponse
		if err := wsConn.ReadJSON(&response); err != nil {
			return nil
		}
		if response.CommandID == 0 {
			continue
		}
		if !conn.deliver(response) {
			_ = recordControlCommandResponse(context.Background(), response)
		}
	}
}

func (h *ControlWebSocketHub) Dispatch(ctx context.Context, command *model.ControlCommand) error {
	conn := h.get(command.NodeID)
	if conn == nil {
		return ErrConflict("node websocket connection is not active")
	}

	payload, err := marshalControlDispatchMessage(command)
	if err != nil {
		return err
	}

	responseCh := conn.registerPending(command.ID)
	defer conn.unregisterPending(command.ID)

	if err := markControlCommandSent(ctx, command); err != nil {
		return err
	}
	if err := conn.write(payload); err != nil {
		return WrapInternal("send websocket control command failed", err)
	}

	timer := time.NewTimer(controlDispatchTimeout())
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return WrapInternal("wait websocket control response canceled", ctx.Err())
	case <-timer.C:
		message := "websocket control command timeout"
		return completeControlCommand(ctx, command, ControlCommandStatusTimeout, "timeout", &message, []byte("{}"), nil)
	case response := <-responseCh:
		return applyControlCommandResponse(ctx, command, response)
	}
}

func (h *ControlWebSocketHub) IsOnline(nodeID uint) bool {
	return h.get(nodeID) != nil
}

func (h *ControlWebSocketHub) register(conn *controlWebSocketConnection) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if old := h.connections[conn.nodeID]; old != nil {
		_ = old.conn.Close()
	}
	h.connections[conn.nodeID] = conn
}

func (h *ControlWebSocketHub) unregister(nodeID uint, conn *controlWebSocketConnection) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.connections[nodeID] == conn {
		delete(h.connections, nodeID)
	}
}

func (h *ControlWebSocketHub) get(nodeID uint) *controlWebSocketConnection {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.connections[nodeID]
}

func (c *controlWebSocketConnection) registerPending(commandID uint) chan ControlCommandResponse {
	ch := make(chan ControlCommandResponse, 1)
	c.mu.Lock()
	c.pending[commandID] = ch
	c.mu.Unlock()
	return ch
}

func (c *controlWebSocketConnection) unregisterPending(commandID uint) {
	c.mu.Lock()
	delete(c.pending, commandID)
	c.mu.Unlock()
}

func (c *controlWebSocketConnection) deliver(response ControlCommandResponse) bool {
	c.mu.Lock()
	ch := c.pending[response.CommandID]
	if ch != nil {
		delete(c.pending, response.CommandID)
	}
	c.mu.Unlock()

	if ch == nil {
		return false
	}
	ch <- response
	return true
}

func (c *controlWebSocketConnection) write(payload []byte) error {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	return c.conn.WriteMessage(websocket.TextMessage, payload)
}

func recordControlCommandResponse(ctx context.Context, response ControlCommandResponse) error {
	var command model.ControlCommand
	err := global.DB.WithContext(ctx).Where("id = ?", response.CommandID).First(&command).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}
	if err != nil {
		return WrapInternal("get control command failed", err)
	}
	return applyControlCommandResponse(ctx, &command, response)
}

func applyControlCommandResponse(ctx context.Context, command *model.ControlCommand, response ControlCommandResponse) error {
	status, event, message := controlStatusFromResponse(response)
	return completeControlCommand(ctx, command, status, event, message, response.Result, nil)
}

func controlStatusFromResponse(response ControlCommandResponse) (int, string, *string) {
	status := strings.ToLower(strings.TrimSpace(response.Status))
	message := response.ErrorMessage

	switch status {
	case "", "success", "succeeded", "ok":
		return ControlCommandStatusSuccess, "succeeded", message
	case "running":
		return ControlCommandStatusRunning, "running", message
	case "timeout":
		if message == nil {
			text := "control command timeout"
			message = &text
		}
		return ControlCommandStatusTimeout, "timeout", message
	case "failed", "failure", "error":
		if message == nil {
			text := "control command failed"
			message = &text
		}
		return ControlCommandStatusFailed, "failed", message
	default:
		text := "invalid control command response status"
		return ControlCommandStatusFailed, "failed", &text
	}
}
