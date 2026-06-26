package service

import (
	"encoding/json"
	"time"
)

type AccessCommand struct {
	DeviceCode  string
	LicenseKey  string
	ProductID   uint
	VersionCode string
}

type RegisterResult struct {
	NodeID             uint   `json:"node_id"`
	LicenseID          uint   `json:"license_id"`
	ProductID          uint   `json:"product_id"`
	LicenseKey         string `json:"license_key"`
	LicenseStatus      int    `json:"license_status"`
	FeatureMask        string `json:"feature_mask"`
	MaxNodes           int    `json:"max_nodes"`
	CurrentNodeCount   int    `json:"current_node_count"`
	MaxConcurrent      int    `json:"max_concurrent"`
	HeartbeatInterval  int    `json:"heartbeat_interval"`
	BindingEstablished bool   `json:"binding_established"`
}

type CreateLicenseCommand struct {
	ProductID     uint
	ValidityHours int
	MaxNodes      int
	MaxConcurrent int
	Remark        *string
}

type BatchCreateLicenseCommand struct {
	ProductID     uint
	ValidityHours int
	MaxNodes      int
	MaxConcurrent int
	Remark        *string
	Count         int
}

type LicenseData struct {
	ID            uint    `json:"id"`
	ProductID     uint    `json:"product_id"`
	LicenseKey    string  `json:"license_key"`
	ValidityHours int     `json:"validity_hours"`
	Status        int     `json:"status"`
	Remark        *string `json:"remark"`
	MaxNodes      int     `json:"max_nodes"`
	MaxConcurrent int     `json:"max_concurrent"`
	FeatureMask   string  `json:"feature_mask"`
}

type UpdateLicenseCommand struct {
	ID            uint
	MaxNodes      int
	MaxConcurrent int
	FeatureMask   string
	Remark        *string
}

type RestoreLicenseCommand struct {
	ID uint
}

type RenewLicenseCommand struct {
	ID         uint
	ExtraHours int
}

type CreateNodeCommand struct {
	DeviceCode string
	Metadata   *string
}

type UpdateNodeCommand struct {
	ID         uint
	DeviceCode *string
	Metadata   *string
}

type NodeData struct {
	ID         uint    `json:"id"`
	DeviceCode string  `json:"device_code"`
	Status     int     `json:"status"`
	Metadata   *string `json:"metadata"`
}

type ListProductsCommand struct {
	Name   *string
	Status *int
	Limit  int
	Offset int
}

type ListLicensesCommand struct {
	ProductID  *uint
	Status     *int
	LicenseKey *string
	Limit      int
	Offset     int
}

type ListNodesCommand struct {
	DeviceCode *string
	Status     *int
	Limit      int
	Offset     int
}

type AddBindingCommand struct {
	NodeID    uint
	LicenseID uint
}

type AutoBindCommand struct {
	DeviceCode string
	LicenseID  uint
}

type UnbindCommand struct {
	NodeID    uint
	LicenseID uint
}

type UpdateNodeStatusCommand struct {
	NodeID uint
	Reason *string
}

type CreateProductCommand struct {
	Name        string
	Description *string
}

type ProductData struct {
	ID          uint    `json:"id"`
	Name        string  `json:"name"`
	Description *string `json:"description"`
}

type UpdateProductCommand struct {
	ID          uint
	Name        *string
	Description *string
}

type ReleaseMethod int

const (
	ReleaseImmediate ReleaseMethod = iota
	ReleaseScheduled
	ReleaseHold
)

type CreateProductVersionCommand struct {
	ProductID   uint
	VersionCode string
	ReleaseDate *time.Time
	Description *string
	Method      ReleaseMethod
}

type ProductVersionData struct {
	ID          uint       `json:"id"`
	ProductID   uint       `json:"product_id"`
	VersionCode string     `json:"version_code"`
	ReleaseDate *time.Time `json:"release_date"`
}

type ReleaseNewVersionCommand struct {
	ProductID   uint
	VersionID   uint
	ReleaseDate *time.Time
}

type UpdateMinVersionCommand struct {
	ProductID uint
	VersionID uint
}

type DeprecateVersionCommand struct {
	ProductID uint
	VersionID uint
}

type CreateControlServiceCommand struct {
	ProductID    *uint
	Identifier   string
	Name         string
	Description  *string
	ServiceType  string
	InputSchema  json.RawMessage
	OutputSchema json.RawMessage
}

type UpdateControlServiceCommand struct {
	ID           uint
	ProductID    *uint
	Name         *string
	Description  *string
	ServiceType  *string
	InputSchema  json.RawMessage
	OutputSchema json.RawMessage
}

type UpdateControlServiceStatusCommand struct {
	ID     uint
	Status int
}

type ControlServiceData struct {
	ID           uint            `json:"id"`
	ProductID    *uint           `json:"product_id,omitempty"`
	Identifier   string          `json:"identifier"`
	Name         string          `json:"name"`
	Description  *string         `json:"description,omitempty"`
	ServiceType  string          `json:"service_type"`
	InputSchema  json.RawMessage `json:"input_schema"`
	OutputSchema json.RawMessage `json:"output_schema"`
	Status       int             `json:"status"`
}

type ListControlServicesCommand struct {
	ProductID *uint
	Limit     int
	Offset    int
}

type ListControlCommandsCommand struct {
	NodeID            *uint
	ServiceIdentifier *string
	Status            *int
	Limit             int
	Offset            int
}

type ReportNodeCapabilityCommand struct {
	NodeID            uint
	ServiceIdentifier string
	Schema            json.RawMessage
	Protocol          string
	Endpoint          *string
}

type ListNodeCapabilitiesCommand struct {
	NodeID uint
	Limit  int
	Offset int
}

type NodeCapabilityData struct {
	ID                uint            `json:"id"`
	NodeID            uint            `json:"node_id"`
	ServiceIdentifier string          `json:"service_identifier"`
	Schema            json.RawMessage `json:"schema"`
	Protocol          string          `json:"protocol"`
	Endpoint          *string         `json:"endpoint,omitempty"`
	Status            int             `json:"status"`
}

type CreateControlCommand struct {
	NodeID            uint
	ServiceIdentifier string
	Payload           json.RawMessage
}

type ControlCommandData struct {
	ID                uint            `json:"id"`
	NodeID            uint            `json:"node_id"`
	ServiceIdentifier string          `json:"service_identifier"`
	Payload           json.RawMessage `json:"payload"`
	ConvertedPayload  json.RawMessage `json:"converted_payload"`
	Status            int             `json:"status"`
	Result            json.RawMessage `json:"result"`
	ErrorMessage      *string         `json:"error_message,omitempty"`
}

type CompleteControlCommandCommand struct {
	CommandID    uint
	Status       string
	Result       json.RawMessage
	ErrorMessage *string
}

type PendingControlSummary struct {
	Count      int    `json:"count"`
	CommandIDs []uint `json:"command_ids"`
}
