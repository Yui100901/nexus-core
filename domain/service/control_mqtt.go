package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"nexus-core/global"
	"nexus-core/persistence/model"

	paho "github.com/eclipse/paho.mqtt.golang"
)

type ControlDispatchMessage struct {
	CommandID         uint            `json:"command_id"`
	NodeID            uint            `json:"node_id"`
	ServiceIdentifier string          `json:"service_identifier"`
	Payload           json.RawMessage `json:"payload"`
}

type MQTTPublisher interface {
	Publish(ctx context.Context, topic string, payload []byte) error
}

type PahoMQTTPublisher struct{}

var DefaultMQTTPublisher MQTTPublisher = PahoMQTTPublisher{}

func dispatchMQTTControlCommand(ctx context.Context, command *model.ControlCommand, capability *model.NodeServiceCapability) error {
	if capability.Endpoint == nil || strings.TrimSpace(*capability.Endpoint) == "" {
		return ErrBadRequest("endpoint topic is required for mqtt protocol")
	}

	payload, err := marshalControlDispatchMessage(command)
	if err != nil {
		return err
	}
	if err := DefaultMQTTPublisher.Publish(ctx, strings.TrimSpace(*capability.Endpoint), payload); err != nil {
		return err
	}
	return markControlCommandSent(ctx, command)
}

func (PahoMQTTPublisher) Publish(ctx context.Context, topic string, payload []byte) error {
	cfg := global.GetConfig().MQTT
	if strings.TrimSpace(cfg.BrokerURL) == "" {
		return ErrInternal("mqtt broker is not configured")
	}

	timeout := time.Duration(cfg.PublishTimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = 5 * time.Second
	}

	clientID := strings.TrimSpace(cfg.ClientID)
	if clientID == "" {
		clientID = fmt.Sprintf("nexus-core-control-%d", time.Now().UnixNano())
	}

	opts := paho.NewClientOptions().
		AddBroker(cfg.BrokerURL).
		SetClientID(clientID).
		SetConnectTimeout(timeout).
		SetWriteTimeout(timeout)
	if cfg.Username != "" {
		opts.SetUsername(cfg.Username)
		opts.SetPassword(cfg.Password)
	}

	client := paho.NewClient(opts)
	if token := client.Connect(); !token.WaitTimeout(timeout) {
		return ErrInternal("connect mqtt broker timeout")
	} else if token.Error() != nil {
		return WrapInternal("connect mqtt broker failed", token.Error())
	}
	defer client.Disconnect(250)

	publishDone := make(chan error, 1)
	go func() {
		token := client.Publish(topic, 1, false, payload)
		if !token.WaitTimeout(timeout) {
			publishDone <- ErrInternal("publish mqtt command timeout")
			return
		}
		if token.Error() != nil {
			publishDone <- WrapInternal("publish mqtt command failed", token.Error())
			return
		}
		publishDone <- nil
	}()

	select {
	case <-ctx.Done():
		return WrapInternal("publish mqtt command canceled", ctx.Err())
	case err := <-publishDone:
		return err
	}
}

func marshalControlDispatchMessage(command *model.ControlCommand) ([]byte, error) {
	data, err := json.Marshal(ControlDispatchMessage{
		CommandID:         command.ID,
		NodeID:            command.NodeID,
		ServiceIdentifier: command.ServiceIdentifier,
		Payload:           json.RawMessage(command.ConvertedPayload),
	})
	if err != nil {
		return nil, WrapInternal("marshal control command failed", err)
	}
	return data, nil
}
