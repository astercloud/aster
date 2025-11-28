/**
 * 审批状态管理
 *
 * 管理 HITL (Human-in-the-Loop) 审批流程
 */

import { defineStore } from 'pinia';
import { ref } from 'vue';
import type { ApprovalRequest, ApprovalRecord } from '@/types/approval';
import { useWebSocket } from '@/composables/useWebSocket';

export const useApprovalStore = defineStore('approval', () => {
  // ==================
  // State
  // ==================

  // 待审批请求（使用 Map 提高查找性能）
  const pendingApprovals = ref<Map<string, ApprovalRequest>>(new Map());

  // 审批历史记录
  const approvalHistory = ref<ApprovalRecord[]>([]);

  // ==================
  // Actions
  // ==================

  /**
   * 添加审批请求
   */
  const addApprovalRequest = (request: ApprovalRequest) => {
    pendingApprovals.value.set(request.id, request);
  };

  /**
   * 批准请求
   */
  const approve = async (requestId: string) => {
    const request = pendingApprovals.value.get(requestId);
    if (!request) {
      console.warn(`Approval request ${requestId} not found`);
      return;
    }

    // 记录到历史
    approvalHistory.value.push({
      ...request,
      decision: 'approved',
      decidedAt: Date.now(),
    });

    // 发送到后端
    const { getInstance } = useWebSocket();
    const ws = getInstance();
    if (ws) {
      ws.send({
        type: 'permission_decision',
        payload: {
          request_id: requestId,
          decision: 'approve',
        },
      });
    } else {
      console.error('WebSocket not connected, cannot send approval decision');
    }

    // 从待审批列表中移除
    pendingApprovals.value.delete(requestId);
  };

  /**
   * 拒绝请求
   */
  const reject = async (requestId: string, reason?: string) => {
    const request = pendingApprovals.value.get(requestId);
    if (!request) {
      console.warn(`Approval request ${requestId} not found`);
      return;
    }

    // 记录到历史
    approvalHistory.value.push({
      ...request,
      decision: 'rejected',
      reason,
      decidedAt: Date.now(),
    });

    // 发送到后端
    const { getInstance } = useWebSocket();
    const ws = getInstance();
    if (ws) {
      ws.send({
        type: 'permission_decision',
        payload: {
          request_id: requestId,
          decision: 'reject',
          reason,
        },
      });
    } else {
      console.error('WebSocket not connected, cannot send rejection decision');
    }

    // 从待审批列表中移除
    pendingApprovals.value.delete(requestId);
  };

  /**
   * 获取指定消息的待审批请求
   */
  const getApprovalByMessage = (messageId: string): ApprovalRequest | null => {
    for (const [, request] of pendingApprovals.value) {
      if (request.messageId === messageId) {
        return request;
      }
    }
    return null;
  };

  /**
   * 检查是否有待审批请求
   */
  const hasPendingApprovals = (): boolean => {
    return pendingApprovals.value.size > 0;
  };

  /**
   * 清除所有待审批请求
   */
  const clearPendingApprovals = () => {
    pendingApprovals.value.clear();
  };

  /**
   * 清除审批历史
   */
  const clearHistory = () => {
    approvalHistory.value = [];
  };

  /**
   * 清除所有数据（用于测试）
   */
  const clearAll = () => {
    pendingApprovals.value.clear();
    approvalHistory.value = [];
  };

  // ==================
  // Return
  // ==================

  return {
    // State
    pendingApprovals,
    approvalHistory,

    // Actions
    addApprovalRequest,
    approve,
    reject,
    getApprovalByMessage,
    hasPendingApprovals,
    clearPendingApprovals,
    clearHistory,
    clearAll,
  };
});
