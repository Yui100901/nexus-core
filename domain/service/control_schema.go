package service

import (
	"encoding/json"
	"fmt"
	"strconv"
)

type nodeSchema struct {
	Fields map[string]nodeSchemaField `json:"fields"`
}

type nodeSchemaField struct {
	Source   string          `json:"source"`
	Type     string          `json:"type"`
	Required bool            `json:"required"`
	Default  json.RawMessage `json:"default"`
}

func ConvertPayload(payload json.RawMessage, schemaRaw json.RawMessage) (json.RawMessage, error) {
	if len(payload) == 0 || !json.Valid(payload) {
		return nil, ErrBadRequest("payload must be valid json")
	}
	if len(schemaRaw) == 0 || !json.Valid(schemaRaw) {
		return nil, ErrBadRequest("node schema must be valid json")
	}

	var source map[string]interface{}
	if err := json.Unmarshal(payload, &source); err != nil {
		return nil, ErrBadRequest("payload must be a json object")
	}

	var schema nodeSchema
	if err := json.Unmarshal(schemaRaw, &schema); err != nil {
		return nil, ErrBadRequest("node schema format is invalid")
	}
	if len(schema.Fields) == 0 {
		return payload, nil
	}

	converted := make(map[string]interface{}, len(schema.Fields))
	for targetName, field := range schema.Fields {
		sourceName := field.Source
		if sourceName == "" {
			sourceName = targetName
		}

		value, ok := source[sourceName]
		if !ok {
			if len(field.Default) > 0 {
				var defaultValue interface{}
				if err := json.Unmarshal(field.Default, &defaultValue); err != nil {
					return nil, ErrBadRequest(fmt.Sprintf("default for %s is invalid", targetName))
				}
				value = defaultValue
				ok = true
			}
		}
		if !ok {
			if field.Required {
				return nil, ErrBadRequest(fmt.Sprintf("%s is required", sourceName))
			}
			continue
		}

		typedValue, err := convertScalarValue(value, field.Type)
		if err != nil {
			return nil, ErrBadRequest(fmt.Sprintf("%s %s", targetName, err.Error()))
		}
		converted[targetName] = typedValue
	}

	data, err := json.Marshal(converted)
	if err != nil {
		return nil, WrapInternal("marshal converted payload failed", err)
	}
	return data, nil
}

func convertScalarValue(value interface{}, targetType string) (interface{}, error) {
	switch targetType {
	case "", "any":
		return value, nil
	case "string":
		switch v := value.(type) {
		case string:
			return v, nil
		default:
			return fmt.Sprintf("%v", value), nil
		}
	case "integer":
		switch v := value.(type) {
		case float64:
			return int(v), nil
		case int:
			return v, nil
		case string:
			n, err := strconv.Atoi(v)
			if err != nil {
				return nil, fmt.Errorf("must be integer")
			}
			return n, nil
		default:
			return nil, fmt.Errorf("must be integer")
		}
	case "number":
		switch v := value.(type) {
		case float64:
			return v, nil
		case int:
			return float64(v), nil
		case string:
			n, err := strconv.ParseFloat(v, 64)
			if err != nil {
				return nil, fmt.Errorf("must be number")
			}
			return n, nil
		default:
			return nil, fmt.Errorf("must be number")
		}
	case "boolean":
		switch v := value.(type) {
		case bool:
			return v, nil
		case string:
			b, err := strconv.ParseBool(v)
			if err != nil {
				return nil, fmt.Errorf("must be boolean")
			}
			return b, nil
		default:
			return nil, fmt.Errorf("must be boolean")
		}
	case "object":
		if _, ok := value.(map[string]interface{}); !ok {
			return nil, fmt.Errorf("must be object")
		}
		return value, nil
	case "array":
		if _, ok := value.([]interface{}); !ok {
			return nil, fmt.Errorf("must be array")
		}
		return value, nil
	default:
		return nil, fmt.Errorf("has unsupported type %s", targetType)
	}
}
