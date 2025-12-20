package structured

import (
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"reflect"
	"strings"
)

// SchemaGenerator JSON Schema 生成器
// 用于从 Go struct 生成 JSON Schema，支持结构化输出
type SchemaGenerator struct{}

// NewSchemaGenerator 创建 Schema 生成器
func NewSchemaGenerator() *SchemaGenerator {
	return &SchemaGenerator{}
}

// FromStruct 从 Go struct 生成 JSON Schema
// 支持的 struct tags:
// - json: JSON 字段名
// - required: 是否必需 ("true"/"false")
// - description: 字段描述
// - enum: 枚举值（逗号分隔）
// - minimum: 最小值（数字类型）
// - maximum: 最大值（数字类型）
// - pattern: 正则模式（字符串类型）
func (g *SchemaGenerator) FromStruct(v any) (map[string]any, error) {
	typ := reflect.TypeOf(v)

	// 如果是指针，获取实际类型
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}

	if typ.Kind() != reflect.Struct {
		return nil, fmt.Errorf("expected struct type, got %v", typ.Kind())
	}

	return g.structToSchema(typ)
}

// structToSchema 将 struct 类型转换为 JSON Schema
func (g *SchemaGenerator) structToSchema(typ reflect.Type) (map[string]any, error) {
	schema := map[string]any{
		"type":       "object",
		"properties": make(map[string]any),
	}

	var required []string
	properties := schema["properties"].(map[string]any)

	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)

		// 跳过未导出的字段
		if !field.IsExported() {
			continue
		}

		// 获取 JSON 标签
		jsonTag := field.Tag.Get("json")
		if jsonTag == "-" {
			continue // 跳过标记为 "-" 的字段
		}

		// 解析 JSON 标签
		fieldName := field.Name
		omitEmpty := false
		if jsonTag != "" {
			parts := strings.Split(jsonTag, ",")
			if parts[0] != "" {
				fieldName = parts[0]
			}
			for _, part := range parts[1:] {
				if part == "omitempty" {
					omitEmpty = true
				}
			}
		}

		// 生成字段 Schema
		fieldSchema, err := g.typeToSchema(field.Type)
		if err != nil {
			return nil, fmt.Errorf("field %s: %w", fieldName, err)
		}

		// 添加描述
		if desc := field.Tag.Get("description"); desc != "" {
			fieldSchema["description"] = desc
		}

		// 添加枚举
		if enumTag := field.Tag.Get("enum"); enumTag != "" {
			values := strings.Split(enumTag, ",")
			fieldSchema["enum"] = values
		}

		// 添加数字范围约束
		if field.Type.Kind() == reflect.Int || field.Type.Kind() == reflect.Int64 ||
			field.Type.Kind() == reflect.Float64 {
			if minTag := field.Tag.Get("minimum"); minTag != "" {
				fieldSchema["minimum"] = minTag
			}
			if maxTag := field.Tag.Get("maximum"); maxTag != "" {
				fieldSchema["maximum"] = maxTag
			}
		}

		// 添加字符串模式约束
		if field.Type.Kind() == reflect.String {
			if pattern := field.Tag.Get("pattern"); pattern != "" {
				fieldSchema["pattern"] = pattern
			}
		}

		properties[fieldName] = fieldSchema

		// 检查是否必需
		isRequired := false
		if reqTag := field.Tag.Get("required"); reqTag == "true" {
			isRequired = true
		} else if !omitEmpty && field.Type.Kind() != reflect.Ptr {
			// 非指针且非 omitempty 默认为必需
			isRequired = true
		}

		if isRequired {
			required = append(required, fieldName)
		}
	}

	if len(required) > 0 {
		schema["required"] = required
	}

	return schema, nil
}

// typeToSchema 将 Go 类型转换为 JSON Schema 类型
func (g *SchemaGenerator) typeToSchema(typ reflect.Type) (map[string]any, error) {
	// 处理指针类型
	if typ.Kind() == reflect.Ptr {
		return g.typeToSchema(typ.Elem())
	}

	schema := make(map[string]any)

	switch typ.Kind() {
	case reflect.String:
		schema["type"] = "string"

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		schema["type"] = "integer"

	case reflect.Float32, reflect.Float64:
		schema["type"] = "number"

	case reflect.Bool:
		schema["type"] = "boolean"

	case reflect.Slice, reflect.Array:
		schema["type"] = "array"
		itemSchema, err := g.typeToSchema(typ.Elem())
		if err != nil {
			return nil, fmt.Errorf("array element: %w", err)
		}
		schema["items"] = itemSchema

	case reflect.Map:
		schema["type"] = "object"
		if typ.Key().Kind() != reflect.String {
			return nil, fmt.Errorf("map key must be string, got %v", typ.Key().Kind())
		}
		// 对于 map[string]any，使用 additionalProperties
		schema["additionalProperties"] = true

	case reflect.Struct:
		// 嵌套结构体
		return g.structToSchema(typ)

	case reflect.Interface:
		// 对于 interface{} 或 any，允许任何类型
		// 不指定 type 字段，表示可以是任何 JSON 类型
		return schema, nil

	default:
		return nil, fmt.Errorf("unsupported type: %v", typ.Kind())
	}

	return schema, nil
}

// Validate 验证 Schema 的基本有效性
func (g *SchemaGenerator) Validate(schema map[string]any) error {
	// 检查必需字段
	if _, ok := schema["type"]; !ok {
		return errors.New("schema must have 'type' field")
	}

	schemaType, ok := schema["type"].(string)
	if !ok {
		return errors.New("'type' must be a string")
	}

	switch schemaType {
	case "object":
		// 对象类型应有 properties
		if _, ok := schema["properties"]; !ok {
			return errors.New("object type must have 'properties'")
		}

	case "array":
		// 数组类型应有 items
		if _, ok := schema["items"]; !ok {
			return errors.New("array type must have 'items'")
		}
	}

	return nil
}

// MergeSchemas 合并多个 Schema（用于复杂场景）
func (g *SchemaGenerator) MergeSchemas(schemas ...map[string]any) (map[string]any, error) {
	if len(schemas) == 0 {
		return nil, errors.New("no schemas to merge")
	}

	result := make(map[string]any)
	properties := make(map[string]any)
	var required []string

	for _, schema := range schemas {
		// 合并 properties
		if props, ok := schema["properties"].(map[string]any); ok {
			maps.Copy(properties, props)
		}

		// 合并 required
		if req, ok := schema["required"].([]string); ok {
			required = append(required, req...)
		}
	}

	result["type"] = "object"
	result["properties"] = properties
	if len(required) > 0 {
		result["required"] = required
	}

	return result, nil
}

// JSONSchema JSON Schema 定义（结构化版本）
type JSONSchema struct {
	Type        string                 `json:"type,omitempty"`
	Description string                 `json:"description,omitempty"`
	Properties  map[string]*JSONSchema `json:"properties,omitempty"`
	Items       *JSONSchema            `json:"items,omitempty"`
	Required    []string               `json:"required,omitempty"`
	Enum        []any                  `json:"enum,omitempty"`
	Minimum     *float64               `json:"minimum,omitempty"`
	Maximum     *float64               `json:"maximum,omitempty"`
	Pattern     string                 `json:"pattern,omitempty"`
	Format      string                 `json:"format,omitempty"`
	Default     any                    `json:"default,omitempty"`
}

// ToJSON 将 JSONSchema 转换为 JSON 字符串
func (s *JSONSchema) ToJSON() (string, error) {
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshal schema to json: %w", err)
	}
	return string(data), nil
}

// ToMap 将 JSONSchema 转换为 map[string]any
func (s *JSONSchema) ToMap() (map[string]any, error) {
	data, err := json.Marshal(s)
	if err != nil {
		return nil, fmt.Errorf("marshal schema: %w", err)
	}
	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("unmarshal schema to map: %w", err)
	}
	return result, nil
}

// SchemaValidator Schema 验证器
type SchemaValidator struct {
	schema *JSONSchema
}

// NewSchemaValidator 创建 Schema 验证器
func NewSchemaValidator(schema *JSONSchema) *SchemaValidator {
	return &SchemaValidator{schema: schema}
}

// Validate 验证 JSON 数据是否符合 Schema
// 目前是简化实现，仅做基础类型检查
func (sv *SchemaValidator) Validate(jsonData string) error {
	if sv.schema == nil {
		return nil // 无 Schema，跳过验证
	}

	// TODO: 实现完整的 JSON Schema 验证逻辑
	// 目前只做基本的 JSON 格式检查
	var data any
	if err := json.Unmarshal([]byte(jsonData), &data); err != nil {
		return fmt.Errorf("invalid json: %w", err)
	}

	return nil
}
