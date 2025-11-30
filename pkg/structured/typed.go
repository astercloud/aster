package structured

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
)

// TypedParser 类型化解析器，支持直接绑定到 Go struct
type TypedParser struct {
	schema    *JSONSchema
	validator *SchemaValidator
}

// NewTypedParser 创建类型化解析器
func NewTypedParser(schema *JSONSchema) *TypedParser {
	var validator *SchemaValidator
	if schema != nil {
		validator = NewSchemaValidator(schema)
	}

	return &TypedParser{
		schema:    schema,
		validator: validator,
	}
}

// ParseInto 解析并绑定到目标 struct
func (tp *TypedParser) ParseInto(ctx context.Context, text string, target any) error {
	if target == nil {
		return fmt.Errorf("target cannot be nil")
	}

	// 检查 target 是否为指针
	rv := reflect.ValueOf(target)
	if rv.Kind() != reflect.Ptr {
		return fmt.Errorf("target must be a pointer")
	}

	// 提取 JSON
	rawJSON, err := extractJSONSegment(text)
	if err != nil {
		return fmt.Errorf("extract json: %w", err)
	}

	// Schema 验证（如果配置）
	if tp.validator != nil {
		if err := tp.validator.Validate(rawJSON); err != nil {
			return fmt.Errorf("schema validation failed: %w", err)
		}
	}

	// 绑定到 struct
	if err := json.Unmarshal([]byte(rawJSON), target); err != nil {
		return fmt.Errorf("unmarshal to struct: %w", err)
	}

	return nil
}

// ParseIntoWithValidation 解析并进行自定义验证
func (tp *TypedParser) ParseIntoWithValidation(ctx context.Context, text string, target any, validator func(any) error) error {
	if err := tp.ParseInto(ctx, text, target); err != nil {
		return err
	}

	if validator != nil {
		if err := validator(target); err != nil {
			return fmt.Errorf("custom validation failed: %w", err)
		}
	}

	return nil
}

// ExtractAndParse 提取 JSON 并解析为通用 map
func (tp *TypedParser) ExtractAndParse(ctx context.Context, text string) (map[string]any, error) {
	rawJSON, err := extractJSONSegment(text)
	if err != nil {
		return nil, fmt.Errorf("extract json: %w", err)
	}

	var result map[string]any
	if err := json.Unmarshal([]byte(rawJSON), &result); err != nil {
		return nil, fmt.Errorf("unmarshal json: %w", err)
	}

	return result, nil
}

// TypedOutputSpec 类型化输出规范
type TypedOutputSpec struct {
	StructType       any             // Go struct 类型（用于反射）
	Schema           *JSONSchema     // JSON Schema 验证
	RequiredFields   []string        // 必填字段
	Strict           bool            // 严格模式（验证失败则报错）
	AllowTextBackup  bool            // 解析失败时是否允许保留原始文本
	CustomValidation func(any) error // 自定义验证函数
}

// TypedParseResult 类型化解析结果
type TypedParseResult struct {
	RawText          string   // 原始文本
	RawJSON          string   // 提取的 JSON
	Data             any      // 解析后的数据（绑定到 struct）
	MissingFields    []string // 缺失的必填字段
	ValidationErrors []string // 验证错误
	Success          bool     // 是否成功
}

// ParseTyped 执行类型化解析
func ParseTyped(ctx context.Context, text string, spec TypedOutputSpec) (*TypedParseResult, error) {
	result := &TypedParseResult{
		RawText: text,
		Success: false,
	}

	// 创建目标实例
	if spec.StructType == nil {
		return nil, fmt.Errorf("struct type is required")
	}

	targetType := reflect.TypeOf(spec.StructType)
	if targetType.Kind() == reflect.Ptr {
		targetType = targetType.Elem()
	}

	target := reflect.New(targetType).Interface()

	// 创建解析器
	parser := NewTypedParser(spec.Schema)

	// 解析
	if err := parser.ParseInto(ctx, text, target); err != nil {
		if !spec.AllowTextBackup {
			return result, fmt.Errorf("parse failed: %w", err)
		}
		result.ValidationErrors = append(result.ValidationErrors, err.Error())
		return result, nil
	}

	result.Data = target
	result.Success = true

	// 提取 JSON（用于记录）
	rawJSON, _ := extractJSONSegment(text)
	result.RawJSON = rawJSON

	// 检查必填字段
	if len(spec.RequiredFields) > 0 {
		var dataMap map[string]any
		if err := json.Unmarshal([]byte(rawJSON), &dataMap); err == nil {
			result.MissingFields = checkRequiredFields(dataMap, spec.RequiredFields)
		}
	}

	// 自定义验证
	if spec.CustomValidation != nil {
		if err := spec.CustomValidation(target); err != nil {
			result.ValidationErrors = append(result.ValidationErrors, err.Error())
			if spec.Strict {
				result.Success = false
				return result, fmt.Errorf("custom validation failed: %w", err)
			}
		}
	}

	return result, nil
}

// GenerateSchema 从 Go struct 生成 JSON Schema
func GenerateSchema(structType any) (*JSONSchema, error) {
	t := reflect.TypeOf(structType)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	if t.Kind() != reflect.Struct {
		return nil, fmt.Errorf("expected struct type, got %s", t.Kind())
	}

	schema := &JSONSchema{
		Type:       "object",
		Properties: make(map[string]*JSONSchema),
		Required:   []string{},
	}

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		jsonTag := field.Tag.Get("json")
		if jsonTag == "" || jsonTag == "-" {
			continue
		}

		// 解析 json tag
		fieldName := jsonTag
		for idx := 0; idx < len(jsonTag); idx++ {
			if jsonTag[idx] == ',' {
				fieldName = jsonTag[:idx]
				break
			}
		}

		// 生成字段 schema
		fieldSchema := &JSONSchema{}
		switch field.Type.Kind() {
		case reflect.String:
			fieldSchema.Type = "string"
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			fieldSchema.Type = "integer"
		case reflect.Float32, reflect.Float64:
			fieldSchema.Type = "number"
		case reflect.Bool:
			fieldSchema.Type = "boolean"
		case reflect.Slice, reflect.Array:
			fieldSchema.Type = "array"
		case reflect.Map, reflect.Struct:
			fieldSchema.Type = "object"
		default:
			fieldSchema.Type = "string"
		}

		// 从 struct tag 读取描述
		if desc := field.Tag.Get("description"); desc != "" {
			fieldSchema.Description = desc
		}

		schema.Properties[fieldName] = fieldSchema

		// 检查是否必填
		if required := field.Tag.Get("required"); required == "true" {
			schema.Required = append(schema.Required, fieldName)
		}
	}

	return schema, nil
}

// MustGenerateSchema 生成 Schema，失败时 panic
func MustGenerateSchema(structType any) *JSONSchema {
	schema, err := GenerateSchema(structType)
	if err != nil {
		panic(fmt.Sprintf("generate schema failed: %v", err))
	}
	return schema
}
