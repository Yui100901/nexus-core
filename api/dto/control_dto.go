package dto

import "encoding/json"

type CreateControlServiceCommand struct {
	ProductID    *uint           `json:"product_id"`
	Identifier   string          `json:"identifier" binding:"required"`
	Name         string          `json:"name" binding:"required"`
	Description  *string         `json:"description"`
	ServiceType  string          `json:"service_type" binding:"required"`
	InputSchema  json.RawMessage `json:"input_schema" swaggertype:"object"`
	OutputSchema json.RawMessage `json:"output_schema" swaggertype:"object"`
}

type ReportNodeCapabilityCommand struct {
	NodeID            uint            `json:"node_id" binding:"required"`
	ServiceIdentifier string          `json:"service_identifier" binding:"required"`
	Schema            json.RawMessage `json:"schema" binding:"required" swaggertype:"object"`
	Protocol          string          `json:"protocol" binding:"required"`
	Endpoint          *string         `json:"endpoint"`
}

type CreateControlCommand struct {
	NodeID            uint            `json:"node_id" binding:"required"`
	ServiceIdentifier string          `json:"service_identifier" binding:"required"`
	Payload           json.RawMessage `json:"payload" binding:"required" swaggertype:"object"`
}
