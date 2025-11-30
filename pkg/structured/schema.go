package structured

import (
	"encoding/json"
	"fmt"
)

// JSONSchema JSON Schema 定义
type JSONSchema struct {
	Type        string                 `json:"type,omitempty"`
	Properties  map[string]*JSONSchema `json:"properties,omitempty"`
	Items       *JSONSchema            `json:"items,omitempty"`
	Required    []string               `json:"required,omitempty"`
	Description string                 `json:"description,omitempty"`
	Enum        []any                  `json:"enum,omitempty"`
	Minimum     *float64               `json:"minimum,omitempty"`
	Maximum     *float64               `json:"maximum,omitempty"`
	MinLength   *int                   `json:"minLength,omitempty"`
	MaxLength   *int                   `json:"maxLength,omitempty"`
	Pattern     string                 `json:"pattern,omitempty"`
	Format      string                 `json:"format,omitempty"`
}

// SchemaValidator JSON Schema 验证器
type SchemaValidator struct {
	schema *JSONSchema
}

// NewSchemaValidator 创建 Schema 验证器
func NewSchemaValidator(schema *JSONSchema) *SchemaValidator {
	return &SchemaValidator{
		schema: schema,
	}
}

// Validate 验证 JSON 数据
func (sv *SchemaValidator) Validate(jsonData string) error {
	var data any
	if err := json.Unmarshal([]byte(jsonData), &data); err != nil {
		return fmt.Errorf("invalid json: %w", err)
	}

	return sv.validateValue(data, sv.schema, "root")
}

// validateValue 验证值
func (sv *SchemaValidator) validateValue(value any, schema *JSONSchema, path string) error {
	if schema == nil {
		return nil
	}

	// 类型验证
	if schema.Type != "" {
		if err := sv.validateType(value, schema.Type, path); err != nil {
			return err
		}
	}

	// 根据类型进行具体验证
	switch schema.Type {
	case "object":
		return sv.validateObject(value, schema, path)
	case "array":
		return sv.validateArray(value, schema, path)
	case "string":
		return sv.validateString(value, schema, path)
	case "number", "integer":
		return sv.validateNumber(value, schema, path)
	}

	return nil
}

// validateType 验证类型
func (sv *SchemaValidator) validateType(value any, expectedType string, path string) error {
	actualType := getJSONType(value)

	// integer 可以是 number
	if expectedType == "integer" && actualType == "number" {
		if _, ok := value.(float64); ok {
			if value.(float64) == float64(int(value.(float64))) {
				return nil
			}
		}
	}

	if actualType != expectedType {
		return fmt.Errorf("%s: expected type %s, got %s", path, expectedType, actualType)
	}

	return nil
}

// validateObject 验证对象
func (sv *SchemaValidator) validateObject(value any, schema *JSONSchema, path string) error {
	obj, ok := value.(map[string]any)
	if !ok {
		return fmt.Errorf("%s: expected object", path)
	}

	// 检查必填字段
	for _, required := range schema.Required {
		if _, exists := obj[required]; !exists {
			return fmt.Errorf("%s: missing required field '%s'", path, required)
		}
	}

	// 验证属性
	for key, val := range obj {
		if propSchema, exists := schema.Properties[key]; exists {
			propPath := fmt.Sprintf("%s.%s", path, key)
			if err := sv.validateValue(val, propSchema, propPath); err != nil {
				return err
			}
		}
	}

	return nil
}

// validateArray 验证数组
func (sv *SchemaValidator) validateArray(value any, schema *JSONSchema, path string) error {
	arr, ok := value.([]any)
	if !ok {
		return fmt.Errorf("%s: expected array", path)
	}

	// 验证数组项
	if schema.Items != nil {
		for i, item := range arr {
			itemPath := fmt.Sprintf("%s[%d]", path, i)
			if err := sv.validateValue(item, schema.Items, itemPath); err != nil {
				return err
			}
		}
	}

	return nil
}

// validateString 验证字符串
func (sv *SchemaValidator) validateString(value any, schema *JSONSchema, path string) error {
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("%s: expected string", path)
	}

	// 长度验证
	if schema.MinLength != nil && len(str) < *schema.MinLength {
		return fmt.Errorf("%s: string length %d is less than minimum %d", path, len(str), *schema.MinLength)
	}

	if schema.MaxLength != nil && len(str) > *schema.MaxLength {
		return fmt.Errorf("%s: string length %d exceeds maximum %d", path, len(str), *schema.MaxLength)
	}

	// 枚举验证
	if len(schema.Enum) > 0 {
		found := false
		for _, enumVal := range schema.Enum {
			if str == enumVal {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("%s: value '%s' is not in enum", path, str)
		}
	}

	return nil
}

// validateNumber 验证数字
func (sv *SchemaValidator) validateNumber(value any, schema *JSONSchema, path string) error {
	num, ok := value.(float64)
	if !ok {
		return fmt.Errorf("%s: expected number", path)
	}

	// 范围验证
	if schema.Minimum != nil && num < *schema.Minimum {
		return fmt.Errorf("%s: value %f is less than minimum %f", path, num, *schema.Minimum)
	}

	if schema.Maximum != nil && num > *schema.Maximum {
		return fmt.Errorf("%s: value %f exceeds maximum %f", path, num, *schema.Maximum)
	}

	return nil
}

// getJSONType 获取 JSON 类型
func getJSONType(value any) string {
	switch value.(type) {
	case map[string]any:
		return "object"
	case []any:
		return "array"
	case string:
		return "string"
	case float64:
		return "number"
	case bool:
		return "boolean"
	case nil:
		return "null"
	default:
		return "unknown"
	}
}

// ToJSON 将 Schema 转换为 JSON
func (s *JSONSchema) ToJSON() (string, error) {
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// FromJSON 从 JSON 解析 Schema
func FromJSON(jsonData string) (*JSONSchema, error) {
	var schema JSONSchema
	if err := json.Unmarshal([]byte(jsonData), &schema); err != nil {
		return nil, err
	}
	return &schema, nil
}
