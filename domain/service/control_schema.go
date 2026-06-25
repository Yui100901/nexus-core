package service

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

type nodeSchema struct {
	Fields map[string]nodeSchemaField `json:"fields"`
}

type nodeSchemaField struct {
	Source    string            `json:"source"`
	Type      string            `json:"type"`
	Required  bool              `json:"required"`
	Default   json.RawMessage   `json:"default"`
	Enum      []json.RawMessage `json:"enum"`
	Minimum   *float64          `json:"minimum"`
	Maximum   *float64          `json:"maximum"`
	MinLength *int              `json:"min_length"`
	MaxLength *int              `json:"max_length"`
	Pattern   string            `json:"pattern"`
	Format    string            `json:"format"`
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
		if err := validateConvertedValue(targetName, typedValue, field); err != nil {
			return nil, err
		}
		converted[targetName] = typedValue
	}

	data, err := json.Marshal(converted)
	if err != nil {
		return nil, WrapInternal("marshal converted payload failed", err)
	}
	return data, nil
}

func validateConvertedValue(fieldName string, value interface{}, field nodeSchemaField) error {
	if len(field.Enum) > 0 {
		if !valueInEnum(value, field.Enum) {
			return ErrBadRequest(fmt.Sprintf("%s must be one of enum values", fieldName))
		}
	}

	switch v := value.(type) {
	case int:
		if err := validateNumberRange(fieldName, float64(v), field); err != nil {
			return err
		}
	case float64:
		if err := validateNumberRange(fieldName, v, field); err != nil {
			return err
		}
	case string:
		if field.MinLength != nil && len(v) < *field.MinLength {
			return ErrBadRequest(fmt.Sprintf("%s length must be at least %d", fieldName, *field.MinLength))
		}
		if field.MaxLength != nil && len(v) > *field.MaxLength {
			return ErrBadRequest(fmt.Sprintf("%s length must be less than or equal to %d", fieldName, *field.MaxLength))
		}
		if field.Pattern != "" {
			ok, err := regexp.MatchString(field.Pattern, v)
			if err != nil {
				return ErrBadRequest(fmt.Sprintf("%s pattern is invalid", fieldName))
			}
			if !ok {
				return ErrBadRequest(fmt.Sprintf("%s format is invalid", fieldName))
			}
		}
		if err := validateStringFormat(fieldName, v, field.Format); err != nil {
			return err
		}
	}
	return nil
}

func validateNumberRange(fieldName string, value float64, field nodeSchemaField) error {
	if field.Minimum != nil && value < *field.Minimum {
		return ErrBadRequest(fmt.Sprintf("%s must be greater than or equal to %v", fieldName, *field.Minimum))
	}
	if field.Maximum != nil && value > *field.Maximum {
		return ErrBadRequest(fmt.Sprintf("%s must be less than or equal to %v", fieldName, *field.Maximum))
	}
	return nil
}

func valueInEnum(value interface{}, enum []json.RawMessage) bool {
	valueRaw, err := json.Marshal(value)
	if err != nil {
		return false
	}
	var valueAny interface{}
	if err := json.Unmarshal(valueRaw, &valueAny); err != nil {
		return false
	}
	for _, enumRaw := range enum {
		var enumValue interface{}
		if err := json.Unmarshal(enumRaw, &enumValue); err != nil {
			continue
		}
		if fmt.Sprintf("%v", enumValue) == fmt.Sprintf("%v", valueAny) {
			return true
		}
	}
	return false
}

func validateStringFormat(fieldName string, value string, format string) error {
	switch strings.TrimSpace(format) {
	case "":
		return nil
	case "email":
		if !strings.Contains(value, "@") {
			return ErrBadRequest(fmt.Sprintf("%s must be email", fieldName))
		}
	case "uuid":
		if ok, _ := regexp.MatchString(`^[0-9a-fA-F-]{36}$`, value); !ok {
			return ErrBadRequest(fmt.Sprintf("%s must be uuid", fieldName))
		}
	default:
		return ErrBadRequest(fmt.Sprintf("%s has unsupported format %s", fieldName, format))
	}
	return nil
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
