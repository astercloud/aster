/**
 * 消息类型定义
 * 参考阿里云 ChatUI 设计
 */

export type MessageType = 
  | 'text'
  | 'image'
  | 'card'
  | 'list'
  | 'system'
  | 'thinking'
  | 'tool-call'
  | 'tool-result'
  | 'ask-user';

export type MessageRole = 'user' | 'assistant' | 'system';

export type MessageStatus = 'pending' | 'sent' | 'delivered' | 'read' | 'error';

// ==================
// Agent Event Types (对应后端 pkg/types/events.go)
// ==================

export type AgentChannel = 'progress' | 'control' | 'monitor';

export type ToolCallState = 
  | 'pending'
  | 'executing'
  | 'paused'
  | 'completed'
  | 'failed'
  | 'cancelled';

// ToolCallSnapshot 工具调用快照 (对应后端 ToolCallSnapshot)
export interface ToolCallSnapshot {
  id: string;
  name: string;
  state?: ToolCallState;
  progress?: number; // 0-1
  arguments?: Record<string, any>;
  result?: any;
  error?: string;
  intermediate?: Record<string, any>;
  started_at?: string;
  updated_at?: string;
  cancelable?: boolean;
  pausable?: boolean;
}

// Agent 事件封装
export interface AgentEventEnvelope {
  cursor: number;
  bookmark: any;
  event: AgentEvent;
}

// Agent 事件类型
export type AgentEvent = 
  | ProgressThinkChunkStartEvent
  | ProgressThinkChunkEvent
  | ProgressThinkChunkEndEvent
  | ProgressTextChunkStartEvent
  | ProgressTextChunkEvent
  | ProgressTextChunkEndEvent
  | ProgressToolStartEvent
  | ProgressToolEndEvent
  | ProgressToolProgressEvent
  | ProgressToolIntermediateEvent
  | ProgressToolCancelledEvent
  | ProgressToolErrorEvent
  | ProgressDoneEvent
  | ProgressTodoUpdateEvent
  | ControlPermissionRequiredEvent
  | ControlPermissionDecidedEvent
  | ControlAskUserEvent
  | ControlUserAnswerEvent
  | MonitorStateChangedEvent
  | MonitorErrorEvent
  | MonitorTokenUsageEvent;

// Progress Channel Events
export interface ProgressThinkChunkStartEvent {
  type: 'think_chunk_start';
  step: number;
}

export interface ProgressThinkChunkEvent {
  type: 'think_chunk';
  step: number;
  delta: string;
}

export interface ProgressThinkChunkEndEvent {
  type: 'think_chunk_end';
  step: number;
}

export interface ProgressTextChunkStartEvent {
  type: 'text_chunk_start';
  step: number;
}

export interface ProgressTextChunkEvent {
  type: 'text_chunk';
  step: number;
  delta: string;
}

export interface ProgressTextChunkEndEvent {
  type: 'text_chunk_end';
  step: number;
  text: string;
}

export interface ProgressToolStartEvent {
  type: 'tool:start';
  call: ToolCallSnapshot;
}

export interface ProgressToolEndEvent {
  type: 'tool:end';
  call: ToolCallSnapshot;
}

export interface ProgressToolProgressEvent {
  type: 'tool:progress';
  call: ToolCallSnapshot;
  progress: number;
  message?: string;
  step?: number;
  total?: number;
  metadata?: Record<string, any>;
  eta_ms?: number;
}

export interface ProgressToolIntermediateEvent {
  type: 'tool:intermediate';
  call: ToolCallSnapshot;
  label?: string;
  data?: any;
}

export interface ProgressToolCancelledEvent {
  type: 'tool:cancelled';
  call: ToolCallSnapshot;
  reason?: string;
}

export interface ProgressToolErrorEvent {
  type: 'tool:error';
  call: ToolCallSnapshot;
  error: string;
}

export interface ProgressDoneEvent {
  type: 'done';
  step: number;
  reason: 'completed' | 'interrupted';
}

// Todo Events
export interface TodoItemData {
  id: string;
  content: string;
  active_form: string;
  status: 'pending' | 'in_progress' | 'completed';
  priority?: number;
  created_at: string;
  updated_at: string;
}

export interface ProgressTodoUpdateEvent {
  type: 'todo_update';
  todos: TodoItemData[];
}

// Control Channel Events
export interface ControlPermissionRequiredEvent {
  type: 'permission_required';
  call: ToolCallSnapshot;
}

export interface ControlPermissionDecidedEvent {
  type: 'permission_decided';
  call_id: string;
  decision: 'allow' | 'deny';
  decided_by: string;
  note?: string;
}

// AskUser Events (新增)
export interface QuestionOption {
  label: string;
  description: string;
}

export interface Question {
  question: string;
  header: string;
  options: QuestionOption[];
  multi_select?: boolean;
}

export interface ControlAskUserEvent {
  type: 'ask_user';
  request_id: string;
  questions: Question[];
}

export interface ControlUserAnswerEvent {
  type: 'user_answer';
  request_id: string;
  answers: Record<string, any>;
}

// Monitor Channel Events
export type AgentRuntimeState = 
  | 'ready'
  | 'working'
  | 'idle'
  | 'running'
  | 'paused'
  | 'completed'
  | 'failed';

export interface MonitorStateChangedEvent {
  type: 'state_changed';
  state: AgentRuntimeState;
}

export interface MonitorErrorEvent {
  type: 'error';
  severity: 'info' | 'warn' | 'error';
  phase: 'model' | 'tool' | 'system' | 'lifecycle';
  message: string;
  detail?: Record<string, any>;
}

export interface MonitorTokenUsageEvent {
  type: 'token_usage';
  input_tokens: number;
  output_tokens: number;
  total_tokens: number;
}

/**
 * 基础消息接口
 */
export interface Attachment {
  id: string;
  type: 'image' | 'file' | 'link';
  name?: string;
  url?: string;
  preview?: string;
  size?: number;
  metadata?: Record<string, any>;
}

export interface BaseMessage {
  id: string;
  type: MessageType;
  role: MessageRole;
  createdAt: number;
  status?: MessageStatus;
  user?: User;
  attachments?: Attachment[];
  metadata?: Record<string, any>;
  thoughts?: ThinkingEvent[];
}

/**
 * 用户信息
 */
export interface User {
  id: string;
  name: string;
  avatar?: string;
}

/**
 * 文本消息
 */
export interface TextMessage extends BaseMessage {
  type: 'text';
  content: {
    text: string;
  };
}

/**
 * 图片消息
 */
export interface ImageMessage extends BaseMessage {
  type: 'image';
  content: {
    url: string;
    alt?: string;
    caption?: string;
    metadata?: {
      size?: number;
      dimensions?: {
        width: number;
        height: number;
      };
    };
  };
}

/**
 * 卡片消息
 */
export interface CardMessage extends BaseMessage {
  type: 'card';
  content: {
    title?: string;
    subtitle?: string;
    description?: string;
    image?: {
      url: string;
      alt?: string;
    };
    fields?: Array<{
      label: string;
      value: string;
    }>;
    actions?: CardAction[];
    footer?: string;
  };
}

export interface CardAction {
  label: string;
  action: string;
  style?: 'primary' | 'secondary';
  icon?: string;
}

/**
 * 列表消息
 */
export interface ListMessage extends BaseMessage {
  type: 'list';
  content: {
    title?: string;
    items: ListItem[];
    footer?: {
      text: string;
      action?: string;
    };
  };
}

export interface ListItem {
  title: string;
  description?: string;
  icon?: string;
  image?: string;
  metadata?: Record<string, string>;
  action?: string;
}

/**
 * 系统消息
 */
export interface SystemMessage extends BaseMessage {
  type: 'system';
  role: 'system';
  content: {
    text: string;
    type?: 'info' | 'success' | 'warning' | 'error';
  };
}

/**
 * 思考消息（Think-Aloud）
 */
export interface ThinkingMessage extends BaseMessage {
  type: 'thinking';
  role: 'assistant';
  content: {
    steps: ThinkingStep[];
    isActive?: boolean;
    summary?: string;
  };
}

export interface ThinkingStep {
  id?: string;
  type: 'reasoning' | 'tool_call' | 'tool_result' | 'decision';
  content?: string;
  tool?: {
    name: string;
    args: any;
  };
  result?: any;
  timestamp: number;
}

export interface ToolCall {
  id: string;
  name: string;
  args: Record<string, any>;
}

export interface ToolResult {
  id: string;
  toolCallId: string;
  result: any;
}

export interface ThinkingEvent {
  id?: string;
  stage?: string;
  timestamp: number;
  reasoning?: string;
  decision?: string;
  toolCall?: ToolCall;
  toolResult?: { result: any };
  approvalRequest?: ApprovalRequest;
}

export interface ApprovalRequest {
  id: string;
  toolName: string;
  args: Record<string, any>;
  reason: string;
}

/**
 * AskUser 消息 (新增)
 */
export interface AskUserMessage extends BaseMessage {
  type: 'ask-user';
  role: 'assistant';
  content: {
    request_id: string;
    questions: Question[];
    answered?: boolean;
    answers?: Record<string, any>;
  };
}

/**
 * 消息联合类型
 */
export type Message =
  | TextMessage
  | ImageMessage
  | CardMessage
  | ListMessage
  | SystemMessage
  | ThinkingMessage
  | AskUserMessage;

/**
 * 快捷回复
 */
export interface QuickReply {
  name: string;
  text: string;
  icon?: string;
  isNew?: boolean;
  isHighlight?: boolean;
}

/**
 * 输入建议
 */
export interface Suggestion {
  text: string;
  icon?: string;
}
