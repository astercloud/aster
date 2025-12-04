/**
 * 思维过程相关类型定义
 */

export type ThinkingStepType = "reasoning" | "tool_call" | "tool_result" | "decision" | "approval";

export interface ThinkingStep {
  id?: string;
  type: ThinkingStepType;
  content?: string; // 推理内容
  tool?: {
    name: string;
    args: any;
  };
  result?: any; // 工具执行结果
  timestamp: number;
  messageId?: string; // 关联的消息 ID
}

export interface ThinkingState {
  // 思维步骤（按消息 ID 分组）
  stepsByMessage: Map<string, ThinkingStep[]>;
  // 当前思维内容（流式累积）
  currentThought: string;
  // 当前思维所属的消息 ID
  currentMessageId: string | null;
  // 是否正在思考
  isThinking: boolean;
}
