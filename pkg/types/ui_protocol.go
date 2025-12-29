// Package types provides type definitions for the Aster framework.
package types

// ===================
// Aster UI Protocol Types
// ===================
// 声明式 UI 协议，借鉴 Google A2UI 设计理念
// 核心理念：Safe like data, but expressive like code

// MessageOperation 消息操作类型
type MessageOperation string

const (
	// MessageOperationCreateSurface 创建 Surface 操作
	MessageOperationCreateSurface MessageOperation = "createSurface"
	// MessageOperationSurfaceUpdate Surface 更新操作
	MessageOperationSurfaceUpdate MessageOperation = "surfaceUpdate"
	// MessageOperationDataModelUpdate 数据模型更新操作
	MessageOperationDataModelUpdate MessageOperation = "dataModelUpdate"
	// MessageOperationBeginRendering 开始渲染操作
	MessageOperationBeginRendering MessageOperation = "beginRendering"
	// MessageOperationDeleteSurface 删除 Surface 操作
	MessageOperationDeleteSurface MessageOperation = "deleteSurface"
)

// DataModelOperation 数据模型操作类型
type DataModelOperation string

const (
	// DataModelOperationAdd 添加操作（数组追加或对象合并）
	DataModelOperationAdd DataModelOperation = "add"
	// DataModelOperationReplace 替换操作（默认）
	DataModelOperationReplace DataModelOperation = "replace"
	// DataModelOperationRemove 删除操作
	DataModelOperationRemove DataModelOperation = "remove"
)

// AsterUIMessage Aster UI 协议主消息结构
// 支持五种操作类型，每次消息只包含一种操作
type AsterUIMessage struct {
	// CreateSurface 创建 Surface 消息
	CreateSurface *CreateSurfaceMessage `json:"createSurface,omitempty"`
	// SurfaceUpdate Surface 更新消息
	SurfaceUpdate *SurfaceUpdateMessage `json:"surfaceUpdate,omitempty"`
	// DataModelUpdate 数据模型更新消息
	DataModelUpdate *DataModelUpdateMessage `json:"dataModelUpdate,omitempty"`
	// BeginRendering 开始渲染消息
	BeginRendering *BeginRenderingMessage `json:"beginRendering,omitempty"`
	// DeleteSurface 删除 Surface 消息
	DeleteSurface *DeleteSurfaceMessage `json:"deleteSurface,omitempty"`
}

// CreateSurfaceMessage 创建 Surface 消息
// 用于创建新的 Surface 并指定组件目录
type CreateSurfaceMessage struct {
	// SurfaceID Surface 唯一标识符
	SurfaceID string `json:"surfaceId"`
	// CatalogID 组件目录标识符（推荐使用 URL 格式）
	CatalogID string `json:"catalogId,omitempty"`
}

// SurfaceUpdateMessage Surface 更新消息
// 用于更新指定 surface 的组件定义
type SurfaceUpdateMessage struct {
	// SurfaceID Surface 唯一标识符
	SurfaceID string `json:"surfaceId"`
	// Components 组件定义列表（邻接表模型）
	Components []ComponentDefinition `json:"components"`
}

// DataModelUpdateMessage 数据模型更新消息
// 用于更新数据模型并触发响应式 UI 更新
type DataModelUpdateMessage struct {
	// SurfaceID Surface 唯一标识符
	SurfaceID string `json:"surfaceId"`
	// Path JSON Pointer 路径，默认 "/" 表示根路径
	Path string `json:"path,omitempty"`
	// Op 操作类型：add/replace/remove，默认 replace
	Op DataModelOperation `json:"op,omitempty"`
	// Contents 数据内容（op 为 remove 时可选）
	Contents any `json:"contents,omitempty"`
}

// BeginRenderingMessage 开始渲染消息
// 用于指定根组件开始渲染 surface
type BeginRenderingMessage struct {
	// SurfaceID Surface 唯一标识符
	SurfaceID string `json:"surfaceId"`
	// Root 根组件 ID
	Root string `json:"root"`
	// Styles CSS 自定义属性（主题化支持）
	Styles map[string]string `json:"styles,omitempty"`
	// CatalogID 组件目录标识符（可选，覆盖 createSurface 中的值）
	CatalogID string `json:"catalogId,omitempty"`
}

// DeleteSurfaceMessage 删除 Surface 消息
// 用于移除 surface 并清理相关资源
type DeleteSurfaceMessage struct {
	// SurfaceID Surface 唯一标识符
	SurfaceID string `json:"surfaceId"`
}

// ===================
// Component Definitions
// ===================

// ComponentWeight 组件权重（流式渲染）
type ComponentWeight string

const (
	// ComponentWeightInitial 初始权重
	ComponentWeightInitial ComponentWeight = "initial"
	// ComponentWeightFinal 最终权重
	ComponentWeightFinal ComponentWeight = "final"
)

// ComponentDefinition 组件定义（邻接表模型）
// 使用扁平列表 + ID 引用，而非嵌套树结构
type ComponentDefinition struct {
	// ID 组件唯一标识符
	ID string `json:"id"`
	// Weight 流式渲染权重
	Weight ComponentWeight `json:"weight,omitempty"`
	// Component 组件规格
	Component ComponentSpec `json:"component"`
}

// ComponentSpec 组件规格（联合类型）
// 每个组件类型对应一个 Props 结构
type ComponentSpec struct {
	Text           *TextProps           `json:"Text,omitempty"`
	Image          *ImageProps          `json:"Image,omitempty"`
	Icon           *IconProps           `json:"Icon,omitempty"`
	Video          *VideoProps          `json:"Video,omitempty"`
	AudioPlayer    *AudioPlayerProps    `json:"AudioPlayer,omitempty"`
	Button         *ButtonProps         `json:"Button,omitempty"`
	Row            *RowProps            `json:"Row,omitempty"`
	Column         *ColumnProps         `json:"Column,omitempty"`
	Card           *CardProps           `json:"Card,omitempty"`
	List           *ListProps           `json:"List,omitempty"`
	TextField      *TextFieldProps      `json:"TextField,omitempty"`
	Checkbox       *CheckboxProps       `json:"Checkbox,omitempty"`
	Select         *SelectProps         `json:"Select,omitempty"`
	DateTimeInput  *DateTimeInputProps  `json:"DateTimeInput,omitempty"`
	Slider         *SliderProps         `json:"Slider,omitempty"`
	MultipleChoice *MultipleChoiceProps `json:"MultipleChoice,omitempty"`
	Divider        *DividerProps        `json:"Divider,omitempty"`
	Modal          *ModalProps          `json:"Modal,omitempty"`
	Tabs           *TabsProps           `json:"Tabs,omitempty"`
	Custom         *CustomProps         `json:"Custom,omitempty"`
}

// GetTypeName 获取组件类型名称
func (c *ComponentSpec) GetTypeName() string {
	switch {
	case c.Text != nil:
		return "Text"
	case c.Image != nil:
		return "Image"
	case c.Icon != nil:
		return "Icon"
	case c.Video != nil:
		return "Video"
	case c.AudioPlayer != nil:
		return "AudioPlayer"
	case c.Button != nil:
		return "Button"
	case c.Row != nil:
		return "Row"
	case c.Column != nil:
		return "Column"
	case c.Card != nil:
		return "Card"
	case c.List != nil:
		return "List"
	case c.TextField != nil:
		return "TextField"
	case c.Checkbox != nil:
		return "Checkbox"
	case c.Select != nil:
		return "Select"
	case c.DateTimeInput != nil:
		return "DateTimeInput"
	case c.Slider != nil:
		return "Slider"
	case c.MultipleChoice != nil:
		return "MultipleChoice"
	case c.Divider != nil:
		return "Divider"
	case c.Modal != nil:
		return "Modal"
	case c.Tabs != nil:
		return "Tabs"
	case c.Custom != nil:
		return "Custom"
	default:
		return ""
	}
}

// ===================
// Property Value Types
// ===================

// PropertyValue 属性值类型
// 支持字面值和路径引用（数据绑定）
type PropertyValue struct {
	// LiteralString 字面字符串值
	LiteralString *string `json:"literalString,omitempty"`
	// LiteralNumber 字面数字值
	LiteralNumber *float64 `json:"literalNumber,omitempty"`
	// LiteralBoolean 字面布尔值
	LiteralBoolean *bool `json:"literalBoolean,omitempty"`
	// Path JSON Pointer 路径引用
	Path *string `json:"path,omitempty"`
}

// IsLiteralString 判断是否为字面字符串
func (p *PropertyValue) IsLiteralString() bool {
	return p.LiteralString != nil
}

// IsLiteralNumber 判断是否为字面数字
func (p *PropertyValue) IsLiteralNumber() bool {
	return p.LiteralNumber != nil
}

// IsLiteralBoolean 判断是否为字面布尔值
func (p *PropertyValue) IsLiteralBoolean() bool {
	return p.LiteralBoolean != nil
}

// IsPathReference 判断是否为路径引用
func (p *PropertyValue) IsPathReference() bool {
	return p.Path != nil
}

// NewLiteralString 创建字面字符串 PropertyValue
func NewLiteralString(s string) PropertyValue {
	return PropertyValue{LiteralString: &s}
}

// NewLiteralNumber 创建字面数字 PropertyValue
func NewLiteralNumber(n float64) PropertyValue {
	return PropertyValue{LiteralNumber: &n}
}

// NewLiteralBoolean 创建字面布尔值 PropertyValue
func NewLiteralBoolean(b bool) PropertyValue {
	return PropertyValue{LiteralBoolean: &b}
}

// NewPathReference 创建路径引用 PropertyValue
func NewPathReference(path string) PropertyValue {
	return PropertyValue{Path: &path}
}

// ComponentArrayReference 子组件引用
// 支持显式列表和模板两种方式
type ComponentArrayReference struct {
	// ExplicitList 显式 ID 列表
	ExplicitList []string `json:"explicitList,omitempty"`
	// Template 模板方式（用于动态列表）
	Template *ComponentTemplate `json:"template,omitempty"`
}

// ComponentTemplate 组件模板
type ComponentTemplate struct {
	// ComponentID 模板组件 ID
	ComponentID string `json:"componentId"`
	// DataBinding 数据源路径（JSON Pointer）
	DataBinding string `json:"dataBinding"`
}

// ===================
// Standard Component Props
// ===================

// TextUsageHint 文本组件使用提示
type TextUsageHint string

const (
	TextUsageHintH1      TextUsageHint = "h1"
	TextUsageHintH2      TextUsageHint = "h2"
	TextUsageHintH3      TextUsageHint = "h3"
	TextUsageHintH4      TextUsageHint = "h4"
	TextUsageHintH5      TextUsageHint = "h5"
	TextUsageHintCaption TextUsageHint = "caption"
	TextUsageHintBody    TextUsageHint = "body"
)

// TextProps 文本组件 Props
type TextProps struct {
	// Text 文本内容
	Text PropertyValue `json:"text"`
	// UsageHint 使用提示（语义化样式）
	UsageHint TextUsageHint `json:"usageHint,omitempty"`
}

// ImageUsageHint 图片组件使用提示
type ImageUsageHint string

const (
	ImageUsageHintIcon          ImageUsageHint = "icon"
	ImageUsageHintAvatar        ImageUsageHint = "avatar"
	ImageUsageHintSmallFeature  ImageUsageHint = "smallFeature"
	ImageUsageHintMediumFeature ImageUsageHint = "mediumFeature"
	ImageUsageHintLargeFeature  ImageUsageHint = "largeFeature"
	ImageUsageHintHeader        ImageUsageHint = "header"
)

// ImageProps 图片组件 Props
type ImageProps struct {
	// Src 图片 URL
	Src PropertyValue `json:"src"`
	// Alt 替代文本
	Alt *PropertyValue `json:"alt,omitempty"`
	// UsageHint 使用提示（尺寸和样式）
	UsageHint ImageUsageHint `json:"usageHint,omitempty"`
}

// IconProps 图标组件 Props
type IconProps struct {
	// Name 图标名称
	Name PropertyValue `json:"name"`
	// Size 图标大小
	Size *PropertyValue `json:"size,omitempty"`
	// Color 图标颜色
	Color *PropertyValue `json:"color,omitempty"`
}

// VideoProps 视频组件 Props
type VideoProps struct {
	// Src 视频 URL
	Src PropertyValue `json:"src"`
	// Poster 封面图 URL
	Poster *PropertyValue `json:"poster,omitempty"`
	// Autoplay 是否自动播放
	Autoplay *PropertyValue `json:"autoplay,omitempty"`
	// Controls 是否显示控制条
	Controls *PropertyValue `json:"controls,omitempty"`
	// Loop 是否循环播放
	Loop *PropertyValue `json:"loop,omitempty"`
	// Muted 是否静音
	Muted *PropertyValue `json:"muted,omitempty"`
}

// AudioPlayerProps 音频播放器组件 Props
type AudioPlayerProps struct {
	// Src 音频 URL
	Src PropertyValue `json:"src"`
	// Title 标题
	Title *PropertyValue `json:"title,omitempty"`
	// Autoplay 是否自动播放
	Autoplay *PropertyValue `json:"autoplay,omitempty"`
	// Loop 是否循环播放
	Loop *PropertyValue `json:"loop,omitempty"`
}

// ButtonVariant 按钮变体
type ButtonVariant string

const (
	ButtonVariantPrimary   ButtonVariant = "primary"
	ButtonVariantSecondary ButtonVariant = "secondary"
	ButtonVariantText      ButtonVariant = "text"
)

// ButtonProps 按钮组件 Props
type ButtonProps struct {
	// Label 按钮标签
	Label PropertyValue `json:"label"`
	// Action 动作标识符（用于事件回调）
	Action string `json:"action"`
	// Variant 按钮变体
	Variant ButtonVariant `json:"variant,omitempty"`
	// Disabled 是否禁用
	Disabled *PropertyValue `json:"disabled,omitempty"`
	// Icon 图标名称
	Icon *PropertyValue `json:"icon,omitempty"`
	// ActionContext 动作上下文（点击时自动解析路径引用）
	ActionContext map[string]PropertyValue `json:"actionContext,omitempty"`
}

// Alignment 对齐方式
type Alignment string

const (
	AlignmentStart   Alignment = "start"
	AlignmentCenter  Alignment = "center"
	AlignmentEnd     Alignment = "end"
	AlignmentStretch Alignment = "stretch"
)

// RowProps 行布局组件 Props
type RowProps struct {
	// Children 子组件引用
	Children ComponentArrayReference `json:"children"`
	// Gap 间距（像素）
	Gap *int `json:"gap,omitempty"`
	// Align 对齐方式
	Align Alignment `json:"align,omitempty"`
	// Wrap 是否换行
	Wrap *bool `json:"wrap,omitempty"`
}

// ColumnProps 列布局组件 Props
type ColumnProps struct {
	// Children 子组件引用
	Children ComponentArrayReference `json:"children"`
	// Gap 间距（像素）
	Gap *int `json:"gap,omitempty"`
	// Align 对齐方式
	Align Alignment `json:"align,omitempty"`
}

// CardProps 卡片组件 Props
type CardProps struct {
	// Children 子组件引用
	Children ComponentArrayReference `json:"children"`
	// Title 标题
	Title *PropertyValue `json:"title,omitempty"`
	// Subtitle 副标题
	Subtitle *PropertyValue `json:"subtitle,omitempty"`
	// Clickable 是否可点击
	Clickable *bool `json:"clickable,omitempty"`
	// Action 点击动作标识符
	Action string `json:"action,omitempty"`
}

// ListProps 列表组件 Props
type ListProps struct {
	// Children 子组件引用
	Children ComponentArrayReference `json:"children"`
	// Dividers 是否显示分隔线
	Dividers *bool `json:"dividers,omitempty"`
}

// TextFieldInputType 文本输入类型
type TextFieldInputType string

const (
	TextFieldInputTypeText     TextFieldInputType = "text"
	TextFieldInputTypePassword TextFieldInputType = "password"
	TextFieldInputTypeEmail    TextFieldInputType = "email"
	TextFieldInputTypeNumber   TextFieldInputType = "number"
	TextFieldInputTypeTel      TextFieldInputType = "tel"
	TextFieldInputTypeURL      TextFieldInputType = "url"
)

// TextFieldProps 文本输入组件 Props
type TextFieldProps struct {
	// Value 值（必须是 path 类型，用于双向绑定）
	Value PropertyValue `json:"value"`
	// Label 标签
	Label *PropertyValue `json:"label,omitempty"`
	// Placeholder 占位符
	Placeholder *PropertyValue `json:"placeholder,omitempty"`
	// Multiline 是否多行
	Multiline *bool `json:"multiline,omitempty"`
	// Disabled 是否禁用
	Disabled *PropertyValue `json:"disabled,omitempty"`
	// MaxLength 最大长度
	MaxLength *int `json:"maxLength,omitempty"`
	// InputType 输入类型
	InputType TextFieldInputType `json:"inputType,omitempty"`
}

// CheckboxProps 复选框组件 Props
type CheckboxProps struct {
	// Checked 选中状态（必须是 path 类型，用于双向绑定）
	Checked PropertyValue `json:"checked"`
	// Label 标签
	Label *PropertyValue `json:"label,omitempty"`
	// Disabled 是否禁用
	Disabled *PropertyValue `json:"disabled,omitempty"`
}

// SelectOption 选择选项
type SelectOption struct {
	// Value 选项值
	Value string `json:"value"`
	// Label 显示标签
	Label string `json:"label"`
	// Disabled 是否禁用
	Disabled bool `json:"disabled,omitempty"`
}

// SelectProps 下拉选择组件 Props
type SelectProps struct {
	// Value 值（必须是 path 类型，用于双向绑定）
	Value PropertyValue `json:"value"`
	// Options 选项数组路径或字面值
	Options PropertyValue `json:"options"`
	// Label 标签
	Label *PropertyValue `json:"label,omitempty"`
	// Placeholder 占位符
	Placeholder *PropertyValue `json:"placeholder,omitempty"`
	// Disabled 是否禁用
	Disabled *PropertyValue `json:"disabled,omitempty"`
	// Multiple 是否支持多选
	Multiple *bool `json:"multiple,omitempty"`
}

// DateTimeInputType 日期时间输入类型
type DateTimeInputType string

const (
	DateTimeInputTypeDate     DateTimeInputType = "date"
	DateTimeInputTypeTime     DateTimeInputType = "time"
	DateTimeInputTypeDatetime DateTimeInputType = "datetime"
)

// DateTimeInputProps 日期时间输入组件 Props
type DateTimeInputProps struct {
	// Value 值（必须是 path 类型，用于双向绑定）
	Value PropertyValue `json:"value"`
	// Label 标签
	Label *PropertyValue `json:"label,omitempty"`
	// Type 输入类型
	Type DateTimeInputType `json:"type,omitempty"`
	// Disabled 是否禁用
	Disabled *PropertyValue `json:"disabled,omitempty"`
	// Min 最小值
	Min *PropertyValue `json:"min,omitempty"`
	// Max 最大值
	Max *PropertyValue `json:"max,omitempty"`
}

// SliderProps 滑块组件 Props
type SliderProps struct {
	// Value 值（必须是 path 类型，用于双向绑定）
	Value PropertyValue `json:"value"`
	// Label 标签
	Label *PropertyValue `json:"label,omitempty"`
	// Min 最小值
	Min *float64 `json:"min,omitempty"`
	// Max 最大值
	Max *float64 `json:"max,omitempty"`
	// Step 步长
	Step *float64 `json:"step,omitempty"`
	// Disabled 是否禁用
	Disabled *PropertyValue `json:"disabled,omitempty"`
	// ShowValue 是否显示值
	ShowValue *bool `json:"showValue,omitempty"`
}

// MultipleChoiceOption 多选选项
type MultipleChoiceOption struct {
	// Value 选项值
	Value string `json:"value"`
	// Label 显示标签
	Label string `json:"label"`
	// Description 描述
	Description string `json:"description,omitempty"`
	// Disabled 是否禁用
	Disabled bool `json:"disabled,omitempty"`
}

// MultipleChoiceDirection 多选布局方向
type MultipleChoiceDirection string

const (
	MultipleChoiceDirectionHorizontal MultipleChoiceDirection = "horizontal"
	MultipleChoiceDirectionVertical   MultipleChoiceDirection = "vertical"
)

// MultipleChoiceProps 多选组件 Props
type MultipleChoiceProps struct {
	// Value 值（必须是 path 类型，用于双向绑定）
	Value PropertyValue `json:"value"`
	// Options 选项数组路径或字面值
	Options PropertyValue `json:"options"`
	// Label 标签
	Label *PropertyValue `json:"label,omitempty"`
	// Disabled 是否禁用
	Disabled *PropertyValue `json:"disabled,omitempty"`
	// Direction 布局方向
	Direction MultipleChoiceDirection `json:"direction,omitempty"`
}

// DividerOrientation 分隔线方向
type DividerOrientation string

const (
	DividerOrientationHorizontal DividerOrientation = "horizontal"
	DividerOrientationVertical   DividerOrientation = "vertical"
)

// DividerProps 分隔线组件 Props
type DividerProps struct {
	// Orientation 方向
	Orientation DividerOrientation `json:"orientation,omitempty"`
}

// ModalProps 模态框组件 Props
type ModalProps struct {
	// Open 打开状态（必须是 path 类型，用于双向绑定）
	Open PropertyValue `json:"open"`
	// Title 标题
	Title *PropertyValue `json:"title,omitempty"`
	// Children 子组件引用
	Children ComponentArrayReference `json:"children"`
	// Closable 是否可关闭
	Closable *bool `json:"closable,omitempty"`
	// CloseAction 关闭动作标识符
	CloseAction string `json:"closeAction,omitempty"`
}

// TabDefinition 标签页定义
type TabDefinition struct {
	// ID 标签页 ID
	ID string `json:"id"`
	// Label 标签页标签
	Label PropertyValue `json:"label"`
	// Content 标签页内容（子组件引用）
	Content ComponentArrayReference `json:"content"`
	// Icon 图标
	Icon *PropertyValue `json:"icon,omitempty"`
	// Disabled 是否禁用
	Disabled bool `json:"disabled,omitempty"`
}

// TabsProps 标签页组件 Props
type TabsProps struct {
	// ActiveTab 当前激活的标签页（必须是 path 类型，用于双向绑定）
	ActiveTab PropertyValue `json:"activeTab"`
	// Tabs 标签页定义列表
	Tabs []TabDefinition `json:"tabs"`
}

// CustomProps 自定义组件 Props
type CustomProps struct {
	// Type 已注册的自定义组件类型名称
	Type string `json:"type"`
	// Props 自定义属性
	Props map[string]PropertyValue `json:"props"`
}

// ===================
// UI Event Types
// ===================

// UIActionEvent UI 动作事件（用户交互）
type UIActionEvent struct {
	// SurfaceID Surface ID
	SurfaceID string `json:"surfaceId"`
	// ComponentID 组件 ID
	ComponentID string `json:"componentId"`
	// Action 动作标识符
	Action string `json:"action"`
	// Timestamp ISO 8601 时间戳
	Timestamp string `json:"timestamp"`
	// Context 解析后的上下文数据
	Context map[string]any `json:"context"`
	// Payload 附加数据（向后兼容）
	Payload map[string]any `json:"payload,omitempty"`
}

// ===================
// Client-to-Server Messages
// ===================

// ClientMessage 客户端到服务端消息
type ClientMessage struct {
	// UserAction 用户动作消息
	UserAction *UserActionMessage `json:"userAction,omitempty"`
	// Error 错误消息
	Error *ProtocolError `json:"error,omitempty"`
}

// UserActionMessage 用户动作消息
type UserActionMessage struct {
	// Name 动作名称
	Name string `json:"name"`
	// SurfaceID Surface ID
	SurfaceID string `json:"surfaceId"`
	// SourceComponentID 触发动作的组件 ID
	SourceComponentID string `json:"sourceComponentId"`
	// Timestamp ISO 8601 时间戳
	Timestamp string `json:"timestamp"`
	// Context 解析后的上下文数据
	Context map[string]any `json:"context"`
}

// ===================
// Validation Error Types
// ===================

// ValidationErrorCode 验证错误代码
type ValidationErrorCode string

const (
	// ValidationErrorCodeValidationFailed 验证失败
	ValidationErrorCodeValidationFailed ValidationErrorCode = "VALIDATION_FAILED"
)

// ProtocolError 协议错误（联合类型）
type ProtocolError struct {
	// Code 错误代码
	Code string `json:"code"`
	// SurfaceID Surface ID
	SurfaceID string `json:"surfaceId"`
	// Path JSON Pointer 路径（验证错误时使用）
	Path string `json:"path,omitempty"`
	// Message 错误消息
	Message string `json:"message"`
	// Details 错误详情（通用错误时使用）
	Details map[string]any `json:"details,omitempty"`
}

// IsValidationError 判断是否为验证错误
func (e *ProtocolError) IsValidationError() bool {
	return e.Code == string(ValidationErrorCodeValidationFailed)
}

// NewValidationError 创建验证错误
func NewValidationError(surfaceID, path, message string) *ProtocolError {
	return &ProtocolError{
		Code:      string(ValidationErrorCodeValidationFailed),
		SurfaceID: surfaceID,
		Path:      path,
		Message:   message,
	}
}

// NewGenericError 创建通用错误
func NewGenericError(code, surfaceID, message string, details map[string]any) *ProtocolError {
	return &ProtocolError{
		Code:      code,
		SurfaceID: surfaceID,
		Message:   message,
		Details:   details,
	}
}

// ===================
// Standard Component Types
// ===================

// StandardComponentTypes 标准组件类型白名单
var StandardComponentTypes = []string{
	"Text",
	"Image",
	"Icon",
	"Video",
	"AudioPlayer",
	"Row",
	"Column",
	"Card",
	"List",
	"Tabs",
	"Modal",
	"Divider",
	"Button",
	"TextField",
	"Checkbox",
	"Select",
	"DateTimeInput",
	"Slider",
	"MultipleChoice",
	"Custom",
}

// IsStandardComponentType 判断是否为标准组件类型
func IsStandardComponentType(typeName string) bool {
	for _, t := range StandardComponentTypes {
		if t == typeName {
			return true
		}
	}
	return false
}
