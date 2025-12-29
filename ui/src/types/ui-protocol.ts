/**
 * Aster UI Protocol 类型定义
 *
 * 声明式 UI 协议，借鉴 Google A2UI 设计理念
 * 核心理念：Safe like data, but expressive like code
 *
 * @module ui-protocol
 */

// ==================
// 消息类型定义
// ==================

/**
 * 消息操作类型
 */
export type MessageOperation =
  | 'createSurface'
  | 'surfaceUpdate'
  | 'dataModelUpdate'
  | 'beginRendering'
  | 'deleteSurface';

/**
 * 数据模型操作类型
 */
export type DataModelOperation = 'add' | 'replace' | 'remove';

/**
 * Aster UI 协议主消息结构
 * 支持五种操作类型，每次消息只包含一种操作
 */
export interface AsterUIMessage {
  /** 创建 Surface 消息 */
  createSurface?: CreateSurfaceMessage;
  /** Surface 更新消息 */
  surfaceUpdate?: SurfaceUpdateMessage;
  /** 数据模型更新消息 */
  dataModelUpdate?: DataModelUpdateMessage;
  /** 开始渲染消息 */
  beginRendering?: BeginRenderingMessage;
  /** 删除 Surface 消息 */
  deleteSurface?: DeleteSurfaceMessage;
}

/**
 * 创建 Surface 消息
 * 用于创建新的 Surface 并指定组件目录
 */
export interface CreateSurfaceMessage {
  /** Surface 唯一标识符 */
  surfaceId: string;
  /** 组件目录标识符（推荐使用 URL 格式） */
  catalogId?: string;
}

/**
 * Surface 更新消息
 * 用于更新指定 surface 的组件定义
 */
export interface SurfaceUpdateMessage {
  /** Surface 唯一标识符 */
  surfaceId: string;
  /** 组件定义列表（邻接表模型） */
  components: ComponentDefinition[];
}

/**
 * 数据模型更新消息
 * 用于更新数据模型并触发响应式 UI 更新
 */
export interface DataModelUpdateMessage {
  /** Surface 唯一标识符 */
  surfaceId: string;
  /** JSON Pointer 路径，默认 "/" 表示根路径 */
  path?: string;
  /** 操作类型：add/replace/remove，默认 replace */
  op?: DataModelOperation;
  /** 数据内容（op 为 remove 时可选） */
  contents?: DataValue;
}

/**
 * 开始渲染消息
 * 用于指定根组件开始渲染 surface
 */
export interface BeginRenderingMessage {
  /** Surface 唯一标识符 */
  surfaceId: string;
  /** 根组件 ID */
  root: string;
  /** CSS 自定义属性（主题化支持） */
  styles?: Record<string, string>;
  /** 组件目录标识符（可选，覆盖 createSurface 中的值） */
  catalogId?: string;
}

/**
 * 删除 Surface 消息
 * 用于移除 surface 并清理相关资源
 */
export interface DeleteSurfaceMessage {
  /** Surface 唯一标识符 */
  surfaceId: string;
}

// ==================
// 组件定义
// ==================

/**
 * 组件定义（邻接表模型）
 * 使用扁平列表 + ID 引用，而非嵌套树结构
 */
export interface ComponentDefinition {
  /** 组件唯一标识符 */
  id: string;
  /** 流式渲染权重 */
  weight?: 'initial' | 'final';
  /** 组件规格 */
  component: ComponentSpec;
}

/**
 * 组件规格（联合类型）
 * 每个组件类型对应一个 Props 接口
 */
export type ComponentSpec =
  | { Text: TextProps }
  | { Image: ImageProps }
  | { Icon: IconProps }
  | { Video: VideoProps }
  | { AudioPlayer: AudioPlayerProps }
  | { Button: ButtonProps }
  | { Row: RowProps }
  | { Column: ColumnProps }
  | { Card: CardProps }
  | { List: ListProps }
  | { TextField: TextFieldProps }
  | { Checkbox: CheckboxProps }
  | { Select: SelectProps }
  | { DateTimeInput: DateTimeInputProps }
  | { Slider: SliderProps }
  | { MultipleChoice: MultipleChoiceProps }
  | { Divider: DividerProps }
  | { Modal: ModalProps }
  | { Tabs: TabsProps }
  | { Custom: CustomProps };

/**
 * 获取组件类型名称
 */
export type ComponentTypeName = keyof ComponentSpec extends infer K
  ? K extends string ? K : never
  : never;

// ==================
// 属性值类型
// ==================

/**
 * 属性值类型
 * 支持字面值和路径引用（数据绑定）
 * 同时支持 A2UI 简化格式（直接字面值）
 */
export type PropertyValue =
  | { literalString: string }
  | { literalNumber: number }
  | { literalBoolean: boolean }
  | { path: string };

/**
 * 简化属性值类型（A2UI 兼容）
 * 支持直接字面值或路径引用对象
 */
export type SimplePropertyValue =
  | string
  | number
  | boolean
  | { path: string };

/**
 * 扩展属性值类型（同时支持两种格式）
 */
export type ExtendedPropertyValue = PropertyValue | SimplePropertyValue;

/**
 * 子组件引用
 * 支持显式列表和模板两种方式
 */
export interface ComponentArrayReference {
  /** 显式 ID 列表 */
  explicitList?: string[];
  /** 模板方式（用于动态列表） */
  template?: {
    /** 模板组件 ID */
    componentId: string;
    /** 数据源路径（JSON Pointer） */
    dataBinding: string;
  };
}

// ==================
// 数据模型
// ==================

/**
 * 数据值类型
 * 支持 JSON 兼容的所有基本类型
 */
export type DataValue =
  | string
  | number
  | boolean
  | null
  | DataValue[]
  | DataMap;

/**
 * 数据映射类型
 */
export type DataMap = { [key: string]: DataValue };

/**
 * Surface 状态
 */
export interface Surface {
  /** 根组件 ID */
  rootComponentId: string | null;
  /** 组件树（构建后） */
  componentTree: AnyComponentNode | null;
  /** 数据模型 */
  dataModel: DataMap;
  /** 组件定义映射（ID -> 定义） */
  components: Map<string, ComponentDefinition>;
  /** CSS 自定义属性 */
  styles: Record<string, string>;
  /** 组件目录标识符 */
  catalogId?: string;
}

/**
 * 组件节点（渲染后）
 */
export interface AnyComponentNode {
  /** 组件 ID */
  id: string;
  /** 组件类型 */
  type: string;
  /** 组件属性 */
  props: Record<string, unknown>;
  /** 子组件节点 */
  children?: AnyComponentNode[];
}

// ==================
// 标准组件 Props
// ==================

/**
 * 文本组件使用提示
 */
export type TextUsageHint = 'h1' | 'h2' | 'h3' | 'h4' | 'h5' | 'caption' | 'body';

/**
 * 文本组件 Props
 */
export interface TextProps {
  /** 文本内容 */
  text: PropertyValue;
  /** 使用提示（语义化样式） */
  usageHint?: TextUsageHint;
}

/**
 * 图片组件使用提示
 */
export type ImageUsageHint = 'icon' | 'avatar' | 'smallFeature' | 'mediumFeature' | 'largeFeature' | 'header';

/**
 * 图片组件 Props
 */
export interface ImageProps {
  /** 图片 URL */
  src: PropertyValue;
  /** 替代文本 */
  alt?: PropertyValue;
  /** 使用提示（尺寸和样式） */
  usageHint?: ImageUsageHint;
}

/**
 * 图标组件 Props
 */
export interface IconProps {
  /** 图标名称 */
  name: PropertyValue;
  /** 图标大小 */
  size?: PropertyValue;
  /** 图标颜色 */
  color?: PropertyValue;
}

/**
 * 视频组件 Props
 */
export interface VideoProps {
  /** 视频 URL */
  src: PropertyValue;
  /** 封面图 URL */
  poster?: PropertyValue;
  /** 是否自动播放 */
  autoplay?: PropertyValue;
  /** 是否显示控制条 */
  controls?: PropertyValue;
  /** 是否循环播放 */
  loop?: PropertyValue;
  /** 是否静音 */
  muted?: PropertyValue;
}

/**
 * 音频播放器组件 Props
 */
export interface AudioPlayerProps {
  /** 音频 URL */
  src: PropertyValue;
  /** 标题 */
  title?: PropertyValue;
  /** 是否自动播放 */
  autoplay?: PropertyValue;
  /** 是否循环播放 */
  loop?: PropertyValue;
}

/**
 * 按钮变体
 */
export type ButtonVariant = 'primary' | 'secondary' | 'text';

/**
 * 按钮组件 Props
 */
export interface ButtonProps {
  /** 按钮标签 */
  label: PropertyValue;
  /** 动作标识符（用于事件回调） */
  action: string;
  /** 按钮变体 */
  variant?: ButtonVariant;
  /** 是否禁用 */
  disabled?: PropertyValue;
  /** 图标名称 */
  icon?: PropertyValue;
  /** 动作上下文（点击时自动解析路径引用） */
  actionContext?: Record<string, PropertyValue>;
}

/**
 * 对齐方式
 */
export type Alignment = 'start' | 'center' | 'end' | 'stretch';

/**
 * 行布局组件 Props
 */
export interface RowProps {
  /** 子组件引用 */
  children: ComponentArrayReference;
  /** 间距（像素） */
  gap?: number;
  /** 对齐方式 */
  align?: Alignment;
  /** 是否换行 */
  wrap?: boolean;
}

/**
 * 列布局组件 Props
 */
export interface ColumnProps {
  /** 子组件引用 */
  children: ComponentArrayReference;
  /** 间距（像素） */
  gap?: number;
  /** 对齐方式 */
  align?: Alignment;
}

/**
 * 卡片组件 Props
 */
export interface CardProps {
  /** 子组件引用 */
  children: ComponentArrayReference;
  /** 标题 */
  title?: PropertyValue;
  /** 副标题 */
  subtitle?: PropertyValue;
  /** 是否可点击 */
  clickable?: boolean;
  /** 点击动作标识符 */
  action?: string;
}

/**
 * 列表组件 Props
 */
export interface ListProps {
  /** 子组件引用 */
  children: ComponentArrayReference;
  /** 是否显示分隔线 */
  dividers?: boolean;
}

/**
 * 文本输入组件 Props
 */
export interface TextFieldProps {
  /** 值（必须是 path 类型，用于双向绑定） */
  value: PropertyValue;
  /** 标签 */
  label?: PropertyValue;
  /** 占位符 */
  placeholder?: PropertyValue;
  /** 是否多行 */
  multiline?: boolean;
  /** 是否禁用 */
  disabled?: PropertyValue;
  /** 最大长度 */
  maxLength?: number;
  /** 输入类型 */
  inputType?: 'text' | 'password' | 'email' | 'number' | 'tel' | 'url';
}

/**
 * 复选框组件 Props
 */
export interface CheckboxProps {
  /** 选中状态（必须是 path 类型，用于双向绑定） */
  checked: PropertyValue;
  /** 标签 */
  label?: PropertyValue;
  /** 是否禁用 */
  disabled?: PropertyValue;
}

/**
 * 选择选项
 */
export interface SelectOption {
  /** 选项值 */
  value: string;
  /** 显示标签 */
  label: string;
  /** 是否禁用 */
  disabled?: boolean;
}

/**
 * 下拉选择组件 Props
 */
export interface SelectProps {
  /** 值（必须是 path 类型，用于双向绑定） */
  value: PropertyValue;
  /** 选项数组路径或字面值 */
  options: PropertyValue;
  /** 标签 */
  label?: PropertyValue;
  /** 占位符 */
  placeholder?: PropertyValue;
  /** 是否禁用 */
  disabled?: PropertyValue;
  /** 是否支持多选 */
  multiple?: boolean;
}

/**
 * 日期时间输入类型
 */
export type DateTimeInputType = 'date' | 'time' | 'datetime';

/**
 * 日期时间输入组件 Props
 */
export interface DateTimeInputProps {
  /** 值（必须是 path 类型，用于双向绑定） */
  value: PropertyValue;
  /** 标签 */
  label?: PropertyValue;
  /** 输入类型 */
  type?: DateTimeInputType;
  /** 是否禁用 */
  disabled?: PropertyValue;
  /** 最小值 */
  min?: PropertyValue;
  /** 最大值 */
  max?: PropertyValue;
}

/**
 * 滑块组件 Props
 */
export interface SliderProps {
  /** 值（必须是 path 类型，用于双向绑定） */
  value: PropertyValue;
  /** 标签 */
  label?: PropertyValue;
  /** 最小值 */
  min?: number;
  /** 最大值 */
  max?: number;
  /** 步长 */
  step?: number;
  /** 是否禁用 */
  disabled?: PropertyValue;
  /** 是否显示值 */
  showValue?: boolean;
}

/**
 * 多选选项
 */
export interface MultipleChoiceOption {
  /** 选项值 */
  value: string;
  /** 显示标签 */
  label: string;
  /** 描述 */
  description?: string;
  /** 是否禁用 */
  disabled?: boolean;
}

/**
 * 多选组件 Props
 */
export interface MultipleChoiceProps {
  /** 值（必须是 path 类型，用于双向绑定） */
  value: PropertyValue;
  /** 选项数组路径或字面值 */
  options: PropertyValue;
  /** 标签 */
  label?: PropertyValue;
  /** 是否禁用 */
  disabled?: PropertyValue;
  /** 布局方向 */
  direction?: 'horizontal' | 'vertical';
}

/**
 * 分隔线方向
 */
export type DividerOrientation = 'horizontal' | 'vertical';

/**
 * 分隔线组件 Props
 */
export interface DividerProps {
  /** 方向 */
  orientation?: DividerOrientation;
}

/**
 * 模态框组件 Props
 */
export interface ModalProps {
  /** 打开状态（必须是 path 类型，用于双向绑定） */
  open: PropertyValue;
  /** 标题 */
  title?: PropertyValue;
  /** 子组件引用 */
  children: ComponentArrayReference;
  /** 是否可关闭 */
  closable?: boolean;
  /** 关闭动作标识符 */
  closeAction?: string;
}

/**
 * 标签页定义
 */
export interface TabDefinition {
  /** 标签页 ID */
  id: string;
  /** 标签页标签 */
  label: PropertyValue;
  /** 标签页内容（子组件引用） */
  content: ComponentArrayReference;
  /** 图标 */
  icon?: PropertyValue;
  /** 是否禁用 */
  disabled?: boolean;
}

/**
 * 标签页组件 Props
 */
export interface TabsProps {
  /** 当前激活的标签页（必须是 path 类型，用于双向绑定） */
  activeTab: PropertyValue;
  /** 标签页定义列表 */
  tabs: TabDefinition[];
}

/**
 * 自定义组件 Props
 */
export interface CustomProps {
  /** 已注册的自定义组件类型名称 */
  type: string;
  /** 自定义属性 */
  props: Record<string, PropertyValue>;
}

// ==================
// 事件类型
// ==================

/**
 * UI 动作事件（用户交互）
 */
export interface UIActionEvent {
  /** Surface ID */
  surfaceId: string;
  /** 组件 ID */
  componentId: string;
  /** 动作标识符 */
  action: string;
  /** ISO 8601 时间戳 */
  timestamp: string;
  /** 解析后的上下文数据 */
  context: Record<string, unknown>;
  /** 附加数据（向后兼容） */
  payload?: Record<string, unknown>;
}

// ==================
// Client-to-Server 消息
// ==================

/**
 * 客户端到服务端消息
 */
export interface ClientMessage {
  /** 用户动作消息 */
  userAction?: UserActionMessage;
  /** 错误消息 */
  error?: ProtocolError;
}

/**
 * 用户动作消息
 */
export interface UserActionMessage {
  /** 动作名称 */
  name: string;
  /** Surface ID */
  surfaceId: string;
  /** 触发动作的组件 ID */
  sourceComponentId: string;
  /** ISO 8601 时间戳 */
  timestamp: string;
  /** 解析后的上下文数据 */
  context: Record<string, unknown>;
}

// ==================
// 验证错误类型
// ==================

/**
 * 验证错误代码
 */
export type ValidationErrorCode = 'VALIDATION_FAILED';

/**
 * 协议错误（联合类型）
 */
export interface ProtocolError {
  /** 错误代码 */
  code: string;
  /** Surface ID */
  surfaceId: string;
  /** JSON Pointer 路径（验证错误时使用） */
  path?: string;
  /** 错误消息 */
  message: string;
  /** 错误详情（通用错误时使用） */
  details?: Record<string, unknown>;
}

/**
 * 验证错误
 */
export interface ValidationError extends ProtocolError {
  code: ValidationErrorCode;
  path: string;
}

/**
 * 判断是否为验证错误
 */
export function isValidationError(error: ProtocolError): error is ValidationError {
  return error.code === 'VALIDATION_FAILED' && error.path !== undefined;
}

/**
 * 创建验证错误
 */
export function createValidationError(surfaceId: string, path: string, message: string): ValidationError {
  return {
    code: 'VALIDATION_FAILED',
    surfaceId,
    path,
    message,
  };
}

/**
 * 创建通用错误
 */
export function createGenericError(
  code: string,
  surfaceId: string,
  message: string,
  details?: Record<string, unknown>,
): ProtocolError {
  return {
    code,
    surfaceId,
    message,
    details,
  };
}

// ==================
// 工具函数类型
// ==================

/**
 * 判断 PropertyValue 是否为字面字符串
 */
export function isLiteralString(value: PropertyValue): value is { literalString: string } {
  return 'literalString' in value;
}

/**
 * 判断 PropertyValue 是否为字面数字
 */
export function isLiteralNumber(value: PropertyValue): value is { literalNumber: number } {
  return 'literalNumber' in value;
}

/**
 * 判断 PropertyValue 是否为字面布尔值
 */
export function isLiteralBoolean(value: PropertyValue): value is { literalBoolean: boolean } {
  return 'literalBoolean' in value;
}

/**
 * 判断 PropertyValue 是否为路径引用
 */
export function isPathReference(value: PropertyValue): value is { path: string } {
  return 'path' in value;
}

/**
 * 获取 PropertyValue 的字面值
 */
export function getLiteralValue(value: PropertyValue): string | number | boolean | null {
  if (isLiteralString(value)) return value.literalString;
  if (isLiteralNumber(value)) return value.literalNumber;
  if (isLiteralBoolean(value)) return value.literalBoolean;
  return null;
}

/**
 * 获取组件类型名称
 */
export function getComponentTypeName(spec: ComponentSpec): string {
  const keys = Object.keys(spec);
  const firstKey = keys[0];
  if (!firstKey) {
    throw new Error('Invalid ComponentSpec: no type found');
  }
  return firstKey;
}

/**
 * 获取组件 Props
 */
export function getComponentProps<T = unknown>(spec: ComponentSpec): T {
  const typeName = getComponentTypeName(spec);
  const props = (spec as unknown as Record<string, T>)[typeName];
  if (props === undefined) {
    throw new Error(`Invalid ComponentSpec: no props found for type ${typeName}`);
  }
  return props;
}

/**
 * 标准组件类型白名单
 */
export const STANDARD_COMPONENT_TYPES = [
  'Text',
  'Image',
  'Icon',
  'Video',
  'AudioPlayer',
  'Row',
  'Column',
  'Card',
  'List',
  'Tabs',
  'Modal',
  'Divider',
  'Button',
  'TextField',
  'Checkbox',
  'Select',
  'DateTimeInput',
  'Slider',
  'MultipleChoice',
  'Custom',
] as const;

/**
 * 标准组件类型
 */
export type StandardComponentType = typeof STANDARD_COMPONENT_TYPES[number];

/**
 * 判断是否为标准组件类型
 */
export function isStandardComponentType(type: string): type is StandardComponentType {
  return STANDARD_COMPONENT_TYPES.includes(type as StandardComponentType);
}

/**
 * 解析后的属性值
 */
export interface ResolvedPropertyValue {
  type: 'literal' | 'path';
  value?: string | number | boolean;
  path?: string;
}

/**
 * 解析扩展属性值（支持两种格式）
 * - 直接字面值：string | number | boolean
 * - 路径引用：{ path: string }
 * - 现有格式：{ literalString } | { literalNumber } | { literalBoolean } | { path }
 */
export function parseExtendedPropertyValue(value: unknown): ResolvedPropertyValue {
  // 直接字面值
  if (typeof value === 'string') {
    return { type: 'literal', value };
  }
  if (typeof value === 'number') {
    return { type: 'literal', value };
  }
  if (typeof value === 'boolean') {
    return { type: 'literal', value };
  }

  // 对象格式
  if (typeof value === 'object' && value !== null) {
    const obj = value as Record<string, unknown>;

    // 现有格式
    if ('literalString' in obj && typeof obj.literalString === 'string') {
      return { type: 'literal', value: obj.literalString };
    }
    if ('literalNumber' in obj && typeof obj.literalNumber === 'number') {
      return { type: 'literal', value: obj.literalNumber };
    }
    if ('literalBoolean' in obj && typeof obj.literalBoolean === 'boolean') {
      return { type: 'literal', value: obj.literalBoolean };
    }

    // 路径引用（两种格式都支持）
    if ('path' in obj && typeof obj.path === 'string') {
      return { type: 'path', path: obj.path };
    }
  }

  return { type: 'literal', value: undefined };
}

/**
 * 将简化格式转换为标准 PropertyValue
 */
export function normalizePropertyValue(value: ExtendedPropertyValue): PropertyValue {
  if (typeof value === 'string') {
    return { literalString: value };
  }
  if (typeof value === 'number') {
    return { literalNumber: value };
  }
  if (typeof value === 'boolean') {
    return { literalBoolean: value };
  }
  // 已经是标准格式或路径引用
  return value as PropertyValue;
}

/**
 * 创建 UserActionMessage
 */
export function createUserActionMessage(
  name: string,
  surfaceId: string,
  sourceComponentId: string,
  context: Record<string, unknown>,
): UserActionMessage {
  return {
    name,
    surfaceId,
    sourceComponentId,
    timestamp: new Date().toISOString(),
    context,
  };
}

/**
 * 创建 UIActionEvent
 */
export function createUIActionEvent(
  surfaceId: string,
  componentId: string,
  action: string,
  context: Record<string, unknown>,
): UIActionEvent {
  return {
    surfaceId,
    componentId,
    action,
    timestamp: new Date().toISOString(),
    context,
  };
}
