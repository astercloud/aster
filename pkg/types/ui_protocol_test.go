// Package types provides type definitions for the Aster framework.
package types

import (
	"encoding/json"
	"testing"
	"testing/quick"
)

// ===================
// Property 14: 跨语言序列化一致性
// Feature: aster-ui-protocol, Property 14: 跨语言序列化一致性
// 验证: 需求 8.5
//
// 对于任意有效的 TypeScript 消息对象，序列化为 JSON 并在 Go 中解析后，
// 应该产生等价的结构体；反之亦然。
// ===================

// TestAsterUIMessageRoundTrip 测试 AsterUIMessage 序列化往返
func TestAsterUIMessageRoundTrip(t *testing.T) {
	f := func(surfaceID string, root string) bool {
		if surfaceID == "" || root == "" {
			return true // Skip empty strings
		}

		original := AsterUIMessage{
			BeginRendering: &BeginRenderingMessage{
				SurfaceID: surfaceID,
				Root:      root,
				Styles:    map[string]string{"--primary-color": "#007bff"},
			},
		}

		// Serialize to JSON
		jsonBytes, err := json.Marshal(original)
		if err != nil {
			t.Logf("Marshal error: %v", err)
			return false
		}

		// Deserialize back
		var parsed AsterUIMessage
		if err := json.Unmarshal(jsonBytes, &parsed); err != nil {
			t.Logf("Unmarshal error: %v", err)
			return false
		}

		// Verify equality
		if parsed.BeginRendering == nil {
			return false
		}
		if parsed.BeginRendering.SurfaceID != original.BeginRendering.SurfaceID {
			return false
		}
		if parsed.BeginRendering.Root != original.BeginRendering.Root {
			return false
		}

		return true
	}

	if err := quick.Check(f, &quick.Config{MaxCount: 100}); err != nil {
		t.Error(err)
	}
}

// TestSurfaceUpdateMessageRoundTrip 测试 SurfaceUpdateMessage 序列化往返
func TestSurfaceUpdateMessageRoundTrip(t *testing.T) {
	f := func(surfaceID string, componentID string, text string) bool {
		if surfaceID == "" || componentID == "" {
			return true // Skip empty strings
		}

		original := SurfaceUpdateMessage{
			SurfaceID: surfaceID,
			Components: []ComponentDefinition{
				{
					ID:     componentID,
					Weight: ComponentWeightInitial,
					Component: ComponentSpec{
						Text: &TextProps{
							Text:      NewLiteralString(text),
							UsageHint: TextUsageHintBody,
						},
					},
				},
			},
		}

		// Serialize to JSON
		jsonBytes, err := json.Marshal(original)
		if err != nil {
			t.Logf("Marshal error: %v", err)
			return false
		}

		// Deserialize back
		var parsed SurfaceUpdateMessage
		if err := json.Unmarshal(jsonBytes, &parsed); err != nil {
			t.Logf("Unmarshal error: %v", err)
			return false
		}

		// Verify equality
		if parsed.SurfaceID != original.SurfaceID {
			return false
		}
		if len(parsed.Components) != len(original.Components) {
			return false
		}
		if parsed.Components[0].ID != original.Components[0].ID {
			return false
		}
		if parsed.Components[0].Component.Text == nil {
			return false
		}
		if !parsed.Components[0].Component.Text.Text.IsLiteralString() {
			return false
		}
		if *parsed.Components[0].Component.Text.Text.LiteralString != text {
			return false
		}

		return true
	}

	if err := quick.Check(f, &quick.Config{MaxCount: 100}); err != nil {
		t.Error(err)
	}
}

// TestDataModelUpdateMessageRoundTrip 测试 DataModelUpdateMessage 序列化往返
func TestDataModelUpdateMessageRoundTrip(t *testing.T) {
	f := func(surfaceID string, path string, value string) bool {
		if surfaceID == "" {
			return true // Skip empty strings
		}

		original := DataModelUpdateMessage{
			SurfaceID: surfaceID,
			Path:      "/" + path,
			Contents:  map[string]any{"key": value},
		}

		// Serialize to JSON
		jsonBytes, err := json.Marshal(original)
		if err != nil {
			t.Logf("Marshal error: %v", err)
			return false
		}

		// Deserialize back
		var parsed DataModelUpdateMessage
		if err := json.Unmarshal(jsonBytes, &parsed); err != nil {
			t.Logf("Unmarshal error: %v", err)
			return false
		}

		// Verify equality
		if parsed.SurfaceID != original.SurfaceID {
			return false
		}
		if parsed.Path != original.Path {
			return false
		}

		return true
	}

	if err := quick.Check(f, &quick.Config{MaxCount: 100}); err != nil {
		t.Error(err)
	}
}

// TestDeleteSurfaceMessageRoundTrip 测试 DeleteSurfaceMessage 序列化往返
func TestDeleteSurfaceMessageRoundTrip(t *testing.T) {
	f := func(surfaceID string) bool {
		if surfaceID == "" {
			return true // Skip empty strings
		}

		original := DeleteSurfaceMessage{
			SurfaceID: surfaceID,
		}

		// Serialize to JSON
		jsonBytes, err := json.Marshal(original)
		if err != nil {
			t.Logf("Marshal error: %v", err)
			return false
		}

		// Deserialize back
		var parsed DeleteSurfaceMessage
		if err := json.Unmarshal(jsonBytes, &parsed); err != nil {
			t.Logf("Unmarshal error: %v", err)
			return false
		}

		// Verify equality
		return parsed.SurfaceID == original.SurfaceID
	}

	if err := quick.Check(f, &quick.Config{MaxCount: 100}); err != nil {
		t.Error(err)
	}
}

// TestPropertyValueRoundTrip 测试 PropertyValue 序列化往返
func TestPropertyValueRoundTrip(t *testing.T) {
	testCases := []struct {
		name  string
		value PropertyValue
	}{
		{
			name:  "LiteralString",
			value: NewLiteralString("hello world"),
		},
		{
			name:  "LiteralNumber",
			value: NewLiteralNumber(42.5),
		},
		{
			name:  "LiteralBoolean",
			value: NewLiteralBoolean(true),
		},
		{
			name:  "PathReference",
			value: NewPathReference("/user/name"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Serialize to JSON
			jsonBytes, err := json.Marshal(tc.value)
			if err != nil {
				t.Fatalf("Marshal error: %v", err)
			}

			// Deserialize back
			var parsed PropertyValue
			if err := json.Unmarshal(jsonBytes, &parsed); err != nil {
				t.Fatalf("Unmarshal error: %v", err)
			}

			// Verify equality based on type
			switch {
			case tc.value.IsLiteralString():
				if !parsed.IsLiteralString() || *parsed.LiteralString != *tc.value.LiteralString {
					t.Errorf("LiteralString mismatch: got %v, want %v", parsed, tc.value)
				}
			case tc.value.IsLiteralNumber():
				if !parsed.IsLiteralNumber() || *parsed.LiteralNumber != *tc.value.LiteralNumber {
					t.Errorf("LiteralNumber mismatch: got %v, want %v", parsed, tc.value)
				}
			case tc.value.IsLiteralBoolean():
				if !parsed.IsLiteralBoolean() || *parsed.LiteralBoolean != *tc.value.LiteralBoolean {
					t.Errorf("LiteralBoolean mismatch: got %v, want %v", parsed, tc.value)
				}
			case tc.value.IsPathReference():
				if !parsed.IsPathReference() || *parsed.Path != *tc.value.Path {
					t.Errorf("PathReference mismatch: got %v, want %v", parsed, tc.value)
				}
			}
		})
	}
}

// TestComponentSpecRoundTrip 测试 ComponentSpec 序列化往返
func TestComponentSpecRoundTrip(t *testing.T) {
	testCases := []struct {
		name string
		spec ComponentSpec
	}{
		{
			name: "Text",
			spec: ComponentSpec{
				Text: &TextProps{
					Text:      NewLiteralString("Hello"),
					UsageHint: TextUsageHintH1,
				},
			},
		},
		{
			name: "Button",
			spec: ComponentSpec{
				Button: &ButtonProps{
					Label:   NewLiteralString("Click me"),
					Action:  "submit",
					Variant: ButtonVariantPrimary,
				},
			},
		},
		{
			name: "Row",
			spec: ComponentSpec{
				Row: &RowProps{
					Children: ComponentArrayReference{
						ExplicitList: []string{"child1", "child2"},
					},
					Gap:   intPtr(8),
					Align: AlignmentCenter,
				},
			},
		},
		{
			name: "TextField",
			spec: ComponentSpec{
				TextField: &TextFieldProps{
					Value:       NewPathReference("/form/name"),
					Label:       propValuePtr(NewLiteralString("Name")),
					Placeholder: propValuePtr(NewLiteralString("Enter your name")),
					Multiline:   boolPtr(false),
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Serialize to JSON
			jsonBytes, err := json.Marshal(tc.spec)
			if err != nil {
				t.Fatalf("Marshal error: %v", err)
			}

			// Deserialize back
			var parsed ComponentSpec
			if err := json.Unmarshal(jsonBytes, &parsed); err != nil {
				t.Fatalf("Unmarshal error: %v", err)
			}

			// Verify type name matches
			if parsed.GetTypeName() != tc.spec.GetTypeName() {
				t.Errorf("Type name mismatch: got %v, want %v", parsed.GetTypeName(), tc.spec.GetTypeName())
			}
		})
	}
}

// TestCrossLanguageJSONFormat 测试跨语言 JSON 格式一致性
// 验证 Go 生成的 JSON 格式与 TypeScript 期望的格式一致
func TestCrossLanguageJSONFormat(t *testing.T) {
	// Test AsterUIMessage with surfaceUpdate
	msg := AsterUIMessage{
		SurfaceUpdate: &SurfaceUpdateMessage{
			SurfaceID: "surface-1",
			Components: []ComponentDefinition{
				{
					ID:     "text-1",
					Weight: ComponentWeightInitial,
					Component: ComponentSpec{
						Text: &TextProps{
							Text:      NewLiteralString("Hello World"),
							UsageHint: TextUsageHintBody,
						},
					},
				},
			},
		},
	}

	jsonBytes, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	// Verify JSON structure
	var jsonMap map[string]any
	if err := json.Unmarshal(jsonBytes, &jsonMap); err != nil {
		t.Fatalf("Unmarshal to map error: %v", err)
	}

	// Check surfaceUpdate exists
	surfaceUpdate, ok := jsonMap["surfaceUpdate"].(map[string]any)
	if !ok {
		t.Fatal("surfaceUpdate not found or wrong type")
	}

	// Check surfaceId
	if surfaceUpdate["surfaceId"] != "surface-1" {
		t.Errorf("surfaceId mismatch: got %v", surfaceUpdate["surfaceId"])
	}

	// Check components
	components, ok := surfaceUpdate["components"].([]any)
	if !ok || len(components) != 1 {
		t.Fatal("components not found or wrong length")
	}

	comp := components[0].(map[string]any)
	if comp["id"] != "text-1" {
		t.Errorf("component id mismatch: got %v", comp["id"])
	}

	// Check component spec has Text
	compSpec := comp["component"].(map[string]any)
	textProps, ok := compSpec["Text"].(map[string]any)
	if !ok {
		t.Fatal("Text props not found")
	}

	// Check text property value
	textValue := textProps["text"].(map[string]any)
	if textValue["literalString"] != "Hello World" {
		t.Errorf("text literalString mismatch: got %v", textValue["literalString"])
	}
}

// TestIsStandardComponentType 测试标准组件类型判断
func TestIsStandardComponentType(t *testing.T) {
	// Test valid types
	validTypes := []string{"Text", "Image", "Button", "Row", "Column", "Card", "Custom"}
	for _, typeName := range validTypes {
		if !IsStandardComponentType(typeName) {
			t.Errorf("Expected %s to be a standard component type", typeName)
		}
	}

	// Test invalid types
	invalidTypes := []string{"Unknown", "InvalidType", "text", "BUTTON"}
	for _, typeName := range invalidTypes {
		if IsStandardComponentType(typeName) {
			t.Errorf("Expected %s to NOT be a standard component type", typeName)
		}
	}
}

// TestComponentSpecGetTypeName 测试组件类型名称获取
func TestComponentSpecGetTypeName(t *testing.T) {
	testCases := []struct {
		spec     ComponentSpec
		expected string
	}{
		{ComponentSpec{Text: &TextProps{}}, "Text"},
		{ComponentSpec{Image: &ImageProps{}}, "Image"},
		{ComponentSpec{Button: &ButtonProps{}}, "Button"},
		{ComponentSpec{Row: &RowProps{}}, "Row"},
		{ComponentSpec{Column: &ColumnProps{}}, "Column"},
		{ComponentSpec{Card: &CardProps{}}, "Card"},
		{ComponentSpec{List: &ListProps{}}, "List"},
		{ComponentSpec{TextField: &TextFieldProps{}}, "TextField"},
		{ComponentSpec{Checkbox: &CheckboxProps{}}, "Checkbox"},
		{ComponentSpec{Select: &SelectProps{}}, "Select"},
		{ComponentSpec{Divider: &DividerProps{}}, "Divider"},
		{ComponentSpec{Modal: &ModalProps{}}, "Modal"},
		{ComponentSpec{Tabs: &TabsProps{}}, "Tabs"},
		{ComponentSpec{Custom: &CustomProps{}}, "Custom"},
		{ComponentSpec{}, ""}, // Empty spec
	}

	for _, tc := range testCases {
		if got := tc.spec.GetTypeName(); got != tc.expected {
			t.Errorf("GetTypeName() = %v, want %v", got, tc.expected)
		}
	}
}

// Helper functions
func intPtr(i int) *int {
	return &i
}

func boolPtr(b bool) *bool {
	return &b
}

func propValuePtr(p PropertyValue) *PropertyValue {
	return &p
}


// ===================
// A2UI Alignment Tests
// ===================

// TestDataModelOperationRoundTrip 测试 DataModelOperation 序列化往返
func TestDataModelOperationRoundTrip(t *testing.T) {
	testCases := []struct {
		name string
		op   DataModelOperation
	}{
		{"Add", DataModelOperationAdd},
		{"Replace", DataModelOperationReplace},
		{"Remove", DataModelOperationRemove},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			msg := DataModelUpdateMessage{
				SurfaceID: "test-surface",
				Path:      "/items",
				Op:        tc.op,
				Contents:  map[string]any{"value": "test"},
			}

			jsonBytes, err := json.Marshal(msg)
			if err != nil {
				t.Fatalf("Marshal error: %v", err)
			}

			var parsed DataModelUpdateMessage
			if err := json.Unmarshal(jsonBytes, &parsed); err != nil {
				t.Fatalf("Unmarshal error: %v", err)
			}

			if parsed.Op != tc.op {
				t.Errorf("Op mismatch: got %v, want %v", parsed.Op, tc.op)
			}
		})
	}
}

// TestCreateSurfaceMessageRoundTrip 测试 CreateSurfaceMessage 序列化往返
func TestCreateSurfaceMessageRoundTrip(t *testing.T) {
	testCases := []struct {
		name      string
		surfaceID string
		catalogID string
	}{
		{"WithCatalogID", "surface-1", "https://example.com/catalog/v1"},
		{"WithoutCatalogID", "surface-2", ""},
		{"URLCatalogID", "surface-3", "https://components.example.com/catalog/v2.0"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			original := CreateSurfaceMessage{
				SurfaceID: tc.surfaceID,
				CatalogID: tc.catalogID,
			}

			jsonBytes, err := json.Marshal(original)
			if err != nil {
				t.Fatalf("Marshal error: %v", err)
			}

			var parsed CreateSurfaceMessage
			if err := json.Unmarshal(jsonBytes, &parsed); err != nil {
				t.Fatalf("Unmarshal error: %v", err)
			}

			if parsed.SurfaceID != original.SurfaceID {
				t.Errorf("SurfaceID mismatch: got %v, want %v", parsed.SurfaceID, original.SurfaceID)
			}
			if parsed.CatalogID != original.CatalogID {
				t.Errorf("CatalogID mismatch: got %v, want %v", parsed.CatalogID, original.CatalogID)
			}
		})
	}
}

// TestAsterUIMessageWithCreateSurface 测试 AsterUIMessage 包含 CreateSurface
func TestAsterUIMessageWithCreateSurface(t *testing.T) {
	msg := AsterUIMessage{
		CreateSurface: &CreateSurfaceMessage{
			SurfaceID: "new-surface",
			CatalogID: "https://example.com/catalog",
		},
	}

	jsonBytes, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var parsed AsterUIMessage
	if err := json.Unmarshal(jsonBytes, &parsed); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if parsed.CreateSurface == nil {
		t.Fatal("CreateSurface should not be nil")
	}
	if parsed.CreateSurface.SurfaceID != "new-surface" {
		t.Errorf("SurfaceID mismatch: got %v", parsed.CreateSurface.SurfaceID)
	}
	if parsed.CreateSurface.CatalogID != "https://example.com/catalog" {
		t.Errorf("CatalogID mismatch: got %v", parsed.CreateSurface.CatalogID)
	}
}

// TestValidationErrorRoundTrip 测试 ValidationError 序列化往返
func TestValidationErrorRoundTrip(t *testing.T) {
	original := NewValidationError("surface-1", "/dataModelUpdate/contents", "contents is required")

	jsonBytes, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var parsed ProtocolError
	if err := json.Unmarshal(jsonBytes, &parsed); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if parsed.Code != string(ValidationErrorCodeValidationFailed) {
		t.Errorf("Code mismatch: got %v, want %v", parsed.Code, ValidationErrorCodeValidationFailed)
	}
	if parsed.SurfaceID != original.SurfaceID {
		t.Errorf("SurfaceID mismatch: got %v, want %v", parsed.SurfaceID, original.SurfaceID)
	}
	if parsed.Path != original.Path {
		t.Errorf("Path mismatch: got %v, want %v", parsed.Path, original.Path)
	}
	if parsed.Message != original.Message {
		t.Errorf("Message mismatch: got %v, want %v", parsed.Message, original.Message)
	}
	if !parsed.IsValidationError() {
		t.Error("IsValidationError should return true")
	}
}

// TestGenericErrorRoundTrip 测试 GenericError 序列化往返
func TestGenericErrorRoundTrip(t *testing.T) {
	original := NewGenericError(
		"UNKNOWN_COMPONENT",
		"surface-1",
		"Component type not found",
		map[string]any{"componentType": "CustomWidget"},
	)

	jsonBytes, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var parsed ProtocolError
	if err := json.Unmarshal(jsonBytes, &parsed); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if parsed.Code != "UNKNOWN_COMPONENT" {
		t.Errorf("Code mismatch: got %v", parsed.Code)
	}
	if parsed.SurfaceID != original.SurfaceID {
		t.Errorf("SurfaceID mismatch: got %v", parsed.SurfaceID)
	}
	if parsed.Message != original.Message {
		t.Errorf("Message mismatch: got %v", parsed.Message)
	}
	if parsed.Details == nil {
		t.Error("Details should not be nil")
	}
	if parsed.IsValidationError() {
		t.Error("IsValidationError should return false for generic error")
	}
}

// TestClientMessageRoundTrip 测试 ClientMessage 序列化往返
func TestClientMessageRoundTrip(t *testing.T) {
	t.Run("UserAction", func(t *testing.T) {
		original := ClientMessage{
			UserAction: &UserActionMessage{
				Name:              "submit",
				SurfaceID:         "form-surface",
				SourceComponentID: "submit-btn",
				Timestamp:         "2024-01-15T10:30:00.000Z",
				Context:           map[string]any{"formData": "test"},
			},
		}

		jsonBytes, err := json.Marshal(original)
		if err != nil {
			t.Fatalf("Marshal error: %v", err)
		}

		var parsed ClientMessage
		if err := json.Unmarshal(jsonBytes, &parsed); err != nil {
			t.Fatalf("Unmarshal error: %v", err)
		}

		if parsed.UserAction == nil {
			t.Fatal("UserAction should not be nil")
		}
		if parsed.UserAction.Name != original.UserAction.Name {
			t.Errorf("Name mismatch: got %v", parsed.UserAction.Name)
		}
		if parsed.UserAction.SurfaceID != original.UserAction.SurfaceID {
			t.Errorf("SurfaceID mismatch: got %v", parsed.UserAction.SurfaceID)
		}
		if parsed.UserAction.SourceComponentID != original.UserAction.SourceComponentID {
			t.Errorf("SourceComponentID mismatch: got %v", parsed.UserAction.SourceComponentID)
		}
		if parsed.UserAction.Timestamp != original.UserAction.Timestamp {
			t.Errorf("Timestamp mismatch: got %v", parsed.UserAction.Timestamp)
		}
	})

	t.Run("Error", func(t *testing.T) {
		original := ClientMessage{
			Error: NewValidationError("surface-1", "/path", "error message"),
		}

		jsonBytes, err := json.Marshal(original)
		if err != nil {
			t.Fatalf("Marshal error: %v", err)
		}

		var parsed ClientMessage
		if err := json.Unmarshal(jsonBytes, &parsed); err != nil {
			t.Fatalf("Unmarshal error: %v", err)
		}

		if parsed.Error == nil {
			t.Fatal("Error should not be nil")
		}
		if parsed.Error.Code != string(ValidationErrorCodeValidationFailed) {
			t.Errorf("Code mismatch: got %v", parsed.Error.Code)
		}
	})
}

// TestUIActionEventRoundTrip 测试 UIActionEvent 序列化往返
func TestUIActionEventRoundTrip(t *testing.T) {
	original := UIActionEvent{
		SurfaceID:   "surface-1",
		ComponentID: "button-1",
		Action:      "click",
		Timestamp:   "2024-01-15T10:30:00.000Z",
		Context:     map[string]any{"key": "value"},
		Payload:     map[string]any{"extra": "data"},
	}

	jsonBytes, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var parsed UIActionEvent
	if err := json.Unmarshal(jsonBytes, &parsed); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if parsed.SurfaceID != original.SurfaceID {
		t.Errorf("SurfaceID mismatch: got %v", parsed.SurfaceID)
	}
	if parsed.ComponentID != original.ComponentID {
		t.Errorf("ComponentID mismatch: got %v", parsed.ComponentID)
	}
	if parsed.Action != original.Action {
		t.Errorf("Action mismatch: got %v", parsed.Action)
	}
	if parsed.Timestamp != original.Timestamp {
		t.Errorf("Timestamp mismatch: got %v", parsed.Timestamp)
	}
	if parsed.Context == nil {
		t.Error("Context should not be nil")
	}
	if parsed.Payload == nil {
		t.Error("Payload should not be nil")
	}
}

// TestButtonPropsWithActionContext 测试 ButtonProps 包含 ActionContext
func TestButtonPropsWithActionContext(t *testing.T) {
	original := ButtonProps{
		Label:   NewLiteralString("Submit"),
		Action:  "submit",
		Variant: ButtonVariantPrimary,
		ActionContext: map[string]PropertyValue{
			"userName": NewPathReference("/user/name"),
			"formId":   NewLiteralString("form-1"),
		},
	}

	jsonBytes, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var parsed ButtonProps
	if err := json.Unmarshal(jsonBytes, &parsed); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if parsed.ActionContext == nil {
		t.Fatal("ActionContext should not be nil")
	}
	if len(parsed.ActionContext) != 2 {
		t.Errorf("ActionContext length mismatch: got %v, want 2", len(parsed.ActionContext))
	}

	userName, ok := parsed.ActionContext["userName"]
	if !ok {
		t.Error("userName not found in ActionContext")
	}
	if !userName.IsPathReference() {
		t.Error("userName should be a path reference")
	}
	if *userName.Path != "/user/name" {
		t.Errorf("userName path mismatch: got %v", *userName.Path)
	}

	formId, ok := parsed.ActionContext["formId"]
	if !ok {
		t.Error("formId not found in ActionContext")
	}
	if !formId.IsLiteralString() {
		t.Error("formId should be a literal string")
	}
	if *formId.LiteralString != "form-1" {
		t.Errorf("formId value mismatch: got %v", *formId.LiteralString)
	}
}

// TestBeginRenderingWithCatalogID 测试 BeginRenderingMessage 包含 CatalogID
func TestBeginRenderingWithCatalogID(t *testing.T) {
	original := BeginRenderingMessage{
		SurfaceID: "surface-1",
		Root:      "root-component",
		Styles:    map[string]string{"--primary": "#007bff"},
		CatalogID: "https://example.com/catalog/v2",
	}

	jsonBytes, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var parsed BeginRenderingMessage
	if err := json.Unmarshal(jsonBytes, &parsed); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if parsed.CatalogID != original.CatalogID {
		t.Errorf("CatalogID mismatch: got %v, want %v", parsed.CatalogID, original.CatalogID)
	}
}

// TestDataModelUpdateWithOp 测试 DataModelUpdateMessage 包含 Op 字段
func TestDataModelUpdateWithOp(t *testing.T) {
	testCases := []struct {
		name     string
		op       DataModelOperation
		contents any
	}{
		{"AddToArray", DataModelOperationAdd, []string{"new-item"}},
		{"ReplaceValue", DataModelOperationReplace, map[string]any{"key": "value"}},
		{"RemoveValue", DataModelOperationRemove, nil},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			original := DataModelUpdateMessage{
				SurfaceID: "surface-1",
				Path:      "/items",
				Op:        tc.op,
				Contents:  tc.contents,
			}

			jsonBytes, err := json.Marshal(original)
			if err != nil {
				t.Fatalf("Marshal error: %v", err)
			}

			var parsed DataModelUpdateMessage
			if err := json.Unmarshal(jsonBytes, &parsed); err != nil {
				t.Fatalf("Unmarshal error: %v", err)
			}

			if parsed.Op != tc.op {
				t.Errorf("Op mismatch: got %v, want %v", parsed.Op, tc.op)
			}
		})
	}
}
