/**
 * 审批相关类型定义
 */

export interface ApprovalRequest {
  id: string;
  messageId?: string;         // 关联的消息 ID
  toolName: string;
  args: Record<string, any>;
  reason?: string;
  timestamp: number;
}

export type ApprovalDecision = 'approved' | 'rejected';

export interface ApprovalRecord extends ApprovalRequest {
  decision: ApprovalDecision;
  reason?: string;
  decidedAt: number;
}

export interface ApprovalState {
  // 待审批请求（使用 Map 提高查找性能）
  pendingApprovals: Map<string, ApprovalRequest>;
  // 审批历史记录
  approvalHistory: ApprovalRecord[];
}
