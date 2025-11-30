/**
 * Chat 配置类型
 */

import type { Message, QuickReply, Suggestion } from './message';

export interface AgentProfile {
  name: string;
  description?: string;
  avatar?: string;
  status?: 'idle' | 'thinking' | 'busy' | 'error';
}

export interface ChatConfig {
  // Agent 配置
  agentId?: string;
  roomId?: string;
  workflowId?: string;
  title?: string;
  subtitle?: string;
  showHeader?: boolean;
  emptyText?: string;

  // 连接配置
  apiUrl?: string;
  wsUrl?: string;
  apiKey?: string;

  // 模型配置
  modelConfig?: {
    provider?: string;
    model?: string;
    api_key?: string;
  };

  // 功能开关
  enableThinking?: boolean;
  enableApproval?: boolean;
  enableVoice?: boolean;
  enableImage?: boolean;
  enableFile?: boolean;
  enableQuickReplies?: boolean;
  enableSuggestions?: boolean;
  enableTodos?: boolean;

  // UI 配置
  placeholder?: string;
  welcomeMessage?: string | Message;
  quickReplies?: QuickReply[];
  suggestions?: Suggestion[];
  agentProfile?: AgentProfile;
  demoMode?: boolean;
  demoResponses?: string[];
  demoDelay?: number;

  // 主题配置
  theme?: 'light' | 'dark' | 'auto';
  primaryColor?: string;

  // 行为配置
  autoScroll?: boolean;
  showTimestamp?: boolean;
  showAvatar?: boolean;
  showUsername?: boolean;

  // 回调函数
  onSend?: (message: Message) => void;
  onReceive?: (message: Message) => void;
  onError?: (error: Error) => void;
  onQuickReplyClick?: (reply: QuickReply) => void;
  onSuggestionClick?: (suggestion: Suggestion) => void;
  onApproveAction?: (requestId: string) => void;
  onRejectAction?: (requestId: string) => void;
}

export interface ChatState {
  messages: Message[];
  isTyping: boolean;
  isConnected: boolean;
  currentInput: string;
  quickReplies: QuickReply[];
  suggestions: Suggestion[];
}

export interface ChatActions {
  sendMessage: (content: string | Message) => Promise<void>;
  sendImage: (file: File) => Promise<void>;
  sendFile: (file: File) => Promise<void>;
  sendVoice: (blob: Blob) => Promise<void>;
  clearMessages: () => void;
  deleteMessage: (id: string) => void;
  resendMessage: (id: string) => Promise<void>;
  approveAction: (requestId: string) => Promise<void>;
  rejectAction: (requestId: string) => Promise<void>;
}

/**
 * Chat 组件 Props
 */
export interface ChatProps {
  /** Chat 配置 */
  config: ChatConfig;
}
