/**
 * AsterUI Types
 * 导出所有类型定义
 */

// Message Types
export * from './message';

// Chat Types
export * from './chat';

// Thinking Types
export * from './thinking';

// Approval Types
export * from './approval';

// Workflow Types (new)
export * from './workflow';

// Agent Types
export interface Agent {
  id: string;
  name: string;
  description?: string;
  avatar?: string;
  status: 'idle' | 'thinking' | 'busy' | 'error';
  metadata?: Record<string, any>;
}

// Room Types
export interface Room {
  id: string;
  name: string;
  members: RoomMember[];
  createdAt: number;
  metadata?: Record<string, any>;
}

export interface RoomMember {
  name: string;
  agentId: string;
  avatar?: string;
  status?: string;
}

// Workflow Types
export interface Workflow {
  id: string;
  name: string;
  description?: string;
  steps: WorkflowStep[];
  status: 'idle' | 'running' | 'paused' | 'completed' | 'error';
  currentStep?: number;
}

export interface WorkflowStep {
  id: string;
  name: string;
  icon: string;
  description: string;
  type: 'agent' | 'tool' | 'condition' | 'loop';
  status: 'pending' | 'running' | 'completed' | 'error' | 'skipped';
  config?: Record<string, any>;
  actions?: StepAction[];
}

export interface StepAction {
  id: string;
  label: string;
  icon?: string;
  variant?: 'primary' | 'secondary';
}

// Think-Aloud Types
export interface ThinkAloudEvent {
  id: string;
  stage: string;
  reasoning: string;
  decision: string;
  timestamp: string;
  context?: Record<string, any>;
  toolCall?: ToolCallData;
  toolResult?: ToolResultData;
  approvalRequest?: ApprovalRequest;
}

export interface ToolCallData {
  toolName: string;
  args: Record<string, any>;
}

export interface ToolResultData {
  toolName: string;
  result: Record<string, any>;
}

// ApprovalRequest 已移至 approval.ts
// export interface ApprovalRequest {
//   id: string;
//   toolName: string;
//   args: Record<string, any>;
// }

// Project Types
export interface Project {
  id: string;
  name: string;
  description?: string;
  workspace: 'wechat' | 'video' | 'general';
  lastModified: string;
  status: 'draft' | 'in_progress' | 'completed';
  stats: {
    words: number;
    materials: number;
  };
}

// Material Types
export interface Material {
  id: string;
  type: 'text' | 'image' | 'video' | 'link' | 'template';
  category: string;
  content: string;
  title?: string;
  tags: string[];
  createdAt: string;
  thumbnail?: string;
  metadata?: Record<string, any>;
}

// API Types
export interface ApiResponse<T = any> {
  success: boolean;
  data?: T;
  error?: {
    code: string;
    message: string;
  };
}
