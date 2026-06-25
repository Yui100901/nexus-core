package entity

import "time"

type ServiceStatus int

const (
	ServiceStatusEnabled ServiceStatus = iota + 1
	ServiceStatusDisabled
)

type ControlCommandStatus int

const (
	ControlCommandPending ControlCommandStatus = iota
	ControlCommandSent
	ControlCommandRunning
	ControlCommandSucceeded
	ControlCommandFailed
	ControlCommandTimeout
)

type ControlService struct {
	ID           uint
	ProductID    *uint
	Identifier   string
	Name         string
	Description  *string
	ServiceType  string
	InputSchema  string
	OutputSchema string
	Status       ServiceStatus
}

type NodeServiceCapability struct {
	ID                uint
	NodeID            uint
	ServiceIdentifier string
	Schema            string
	Protocol          string
	Endpoint          *string
	Status            ServiceStatus
}

type ControlCommand struct {
	ID                uint
	NodeID            uint
	ServiceIdentifier string
	Payload           string
	ConvertedPayload  string
	Status            ControlCommandStatus
	Result            string
	ErrorMessage      *string
	SentAt            *time.Time
	CompletedAt       *time.Time
}

type ControlCommandLog struct {
	ID        uint
	CommandID uint
	NodeID    uint
	Event     string
	Status    ControlCommandStatus
	Message   *string
	Data      string
}

type AuditLog struct {
	ID           uint
	ResourceType string
	ResourceID   uint
	Action       string
	Operator     string
	Data         string
}
