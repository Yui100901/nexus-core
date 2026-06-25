package service

import "time"

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

type LicenseData struct {
	ID            uint    `json:"id"`
	ProductID     uint    `json:"product_id"`
	LicenseKey    string  `json:"license_key"`
	ValidityHours int     `json:"validity_hours"`
	Status        int     `json:"status"`
	Remark        *string `json:"remark"`
}

type UpdateLicenseCommand struct {
	ID            uint
	MaxNodes      int
	MaxConcurrent int
	FeatureMask   string
	Remark        *string
}

type RenewLicenseCommand struct {
	ID         uint
	ExtraHours int
}

type CreateNodeCommand struct {
	DeviceCode string
	Metadata   *string
}

type NodeData struct {
	ID         uint    `json:"id"`
	DeviceCode string  `json:"device_code"`
	Status     int     `json:"status"`
	Metadata   *string `json:"metadata"`
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

type CreateProductCommand struct {
	Name        string
	Description *string
}

type ProductData struct {
	ID          uint    `json:"id"`
	Name        string  `json:"name"`
	Description *string `json:"description"`
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
