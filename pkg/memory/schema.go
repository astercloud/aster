package memory

import (
	"encoding/json"
	"errors"
	"fmt"
	"slices"
)

// JSONSchema JSON Schema 定义
// 简化版的 JSON Schema，支持基本的类型验证
type JSONSchema struct {
	Type       string                 `json:"type,omitempty"`       // object, string, number, array, boolean
	Properties map[string]*JSONSchema `json:"properties,omitempty"` // 对象属性（仅 type=object 时有效）
	Items      *JSONSchema            `json:"items,omitempty"`      // 数组元素（仅 type=array 时有效）
	Required   []string               `json:"required,omitempty"`   // 必需字段（仅 type=object 时有效）
	Enum       []any                  `json:"enum,omitempty"`       // 枚举值
	MinLength  *int                   `json:"minLength,omitempty"`  // 最小长度（仅 type=string 时有效）
	MaxLength  *int                   `json:"maxLength,omitempty"`  // 最大长度（仅 type=string 时有效）
	Pattern    string                 `json:"pattern,omitempty"`    // 正则模式（仅 type=string 时有效）
}

// Validate 验证 Schema 本身是否合法
func (s *JSONSchema) Validate() error {
	if s == nil {
		return errors.New("schema cannot be nil")
	}

	validTypes := map[string]bool{
		"object": true, "string": true, "number": true,
		"integer": true, "boolean": true, "array": true,
	}

	if s.Type != "" && !validTypes[s.Type] {
		return fmt.Errorf("invalid type: %s", s.Type)
	}

	// 验证 object 类型的 properties
	if s.Type == "object" && s.Properties != nil {
		for name, prop := range s.Properties {
			if err := prop.Validate(); err != nil {
				return fmt.Errorf("invalid property %s: %w", name, err)
			}
		}
	}

	// 验证 array 类型的 items
	if s.Type == "array" && s.Items != nil {
		if err := s.Items.Validate(); err != nil {
			return fmt.Errorf("invalid items: %w", err)
		}
	}

	return nil
}

// ValidateContent 验证内容是否符合 Schema
// content: JSON 字符串或 Markdown 字符串
// 如果 Schema.Type 为空或 "string"，直接验证字符串
// 如果 Schema.Type 为 "object" 等，先解析 JSON 再验证
func (s *JSONSchema) ValidateContent(content string) error {
	if s == nil {
		return nil // 无 Schema 时不验证
	}

	// 如果 Schema 类型为空或 string，直接验证字符串长度等
	if s.Type == "" || s.Type == "string" {
		return s.validateString(content)
	}

	// 否则，尝试解析为 JSON 并验证
	var data any
	if err := json.Unmarshal([]byte(content), &data); err != nil {
		return fmt.Errorf("content is not valid JSON: %w", err)
	}

	return s.validateValue(data)
}

// validateString 验证字符串
func (s *JSONSchema) validateString(value string) error {
	if s.MinLength != nil && len(value) < *s.MinLength {
		return fmt.Errorf("string length %d is less than minLength %d", len(value), *s.MinLength)
	}

	if s.MaxLength != nil && len(value) > *s.MaxLength {
		return fmt.Errorf("string length %d exceeds maxLength %d", len(value), *s.MaxLength)
	}

	// TODO: 支持 Pattern 正则验证

	return nil
}

// validateValue 验证任意类型的值
func (s *JSONSchema) validateValue(value any) error {
	if s.Type != "" {
		if err := s.validateType(value); err != nil {
			return err
		}
	}

	// 验证枚举
	if len(s.Enum) > 0 {
		found := slices.Contains(s.Enum, value)
		if !found {
			return fmt.Errorf("value %v is not in enum: %v", value, s.Enum)
		}
	}

	// 验证对象属性
	if s.Type == "object" {
		return s.validateObject(value)
	}

	// 验证数组
	if s.Type == "array" {
		return s.validateArray(value)
	}

	return nil
}

// validateType 验证值的类型
func (s *JSONSchema) validateType(value any) error {
	switch s.Type {
	case "object":
		if _, ok := value.(map[string]any); !ok {
			return fmt.Errorf("expected object, got %T", value)
		}
	case "array":
		if _, ok := value.([]any); !ok {
			return fmt.Errorf("expected array, got %T", value)
		}
	case "string":
		if _, ok := value.(string); !ok {
			return fmt.Errorf("expected string, got %T", value)
		}
	case "number":
		switch value.(type) {
		case float64, float32, int, int64, int32:
			// OK
		default:
			return fmt.Errorf("expected number, got %T", value)
		}
	case "integer":
		switch value.(type) {
		case int, int64, int32, float64:
			// Go JSON 解析会把整数解析为 float64，需要额外检查
			if f, ok := value.(float64); ok {
				if f != float64(int64(f)) {
					return fmt.Errorf("expected integer, got float %v", f)
				}
			}
		default:
			return fmt.Errorf("expected integer, got %T", value)
		}
	case "boolean":
		if _, ok := value.(bool); !ok {
			return fmt.Errorf("expected boolean, got %T", value)
		}
	}

	return nil
}

// validateObject 验证对象
func (s *JSONSchema) validateObject(value any) error {
	obj, ok := value.(map[string]any)
	if !ok {
		return fmt.Errorf("expected object, got %T", value)
	}

	// 验证必需字段
	for _, reqField := range s.Required {
		if _, exists := obj[reqField]; !exists {
			return fmt.Errorf("required field '%s' is missing", reqField)
		}
	}

	// 验证各个属性
	for name, propSchema := range s.Properties {
		if propVal, exists := obj[name]; exists {
			if err := propSchema.validateValue(propVal); err != nil {
				return fmt.Errorf("property '%s': %w", name, err)
			}
		}
	}

	return nil
}

// validateArray 验证数组
func (s *JSONSchema) validateArray(value any) error {
	arr, ok := value.([]any)
	if !ok {
		return fmt.Errorf("expected array, got %T", value)
	}

	// 验证数组元素
	if s.Items != nil {
		for i, item := range arr {
			if err := s.Items.validateValue(item); err != nil {
				return fmt.Errorf("array item [%d]: %w", i, err)
			}
		}
	}

	return nil
}

// NewJSONSchemaFromMap 从 map 创建 JSONSchema（方便从配置文件加载）
func NewJSONSchemaFromMap(m map[string]any) (*JSONSchema, error) {
	// 序列化再反序列化，简单实现类型转换
	jsonData, err := json.Marshal(m)
	if err != nil {
		return nil, fmt.Errorf("marshal schema map: %w", err)
	}

	var schema JSONSchema
	if err := json.Unmarshal(jsonData, &schema); err != nil {
		return nil, fmt.Errorf("unmarshal schema: %w", err)
	}

	if err := schema.Validate(); err != nil {
		return nil, err
	}

	return &schema, nil
}

// ToMap 将 JSONSchema 转换为 map（方便序列化）
func (s *JSONSchema) ToMap() map[string]any {
	if s == nil {
		return nil
	}

	result := make(map[string]any)

	if s.Type != "" {
		result["type"] = s.Type
	}
	if s.Properties != nil {
		props := make(map[string]any)
		for name, prop := range s.Properties {
			props[name] = prop.ToMap()
		}
		result["properties"] = props
	}
	if s.Items != nil {
		result["items"] = s.Items.ToMap()
	}
	if len(s.Required) > 0 {
		result["required"] = s.Required
	}
	if len(s.Enum) > 0 {
		result["enum"] = s.Enum
	}
	if s.MinLength != nil {
		result["minLength"] = *s.MinLength
	}
	if s.MaxLength != nil {
		result["maxLength"] = *s.MaxLength
	}
	if s.Pattern != "" {
		result["pattern"] = s.Pattern
	}

	return result
}
