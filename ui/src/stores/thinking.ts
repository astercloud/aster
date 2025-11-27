/**
 * 思维过程状态管理
 *
 * 管理 Agent 的思维过程，包括推理、工具调用决策等
 */

import { defineStore } from 'pinia';
import { ref } from 'vue';
import type { ThinkingStep } from '@/types/thinking';

export const useThinkingStore = defineStore('thinking', () => {
  // ==================
  // State
  // ==================

  // 思维步骤（按消息 ID 分组，使用 Map 提高性能）
  const stepsByMessage = ref<Map<string, ThinkingStep[]>>(new Map());

  // 当前思维内容（流式累积）
  const currentThought = ref<string>('');

  // 当前思维所属的消息 ID
  const currentMessageId = ref<string | null>(null);

  // 是否正在思考
  const isThinking = ref(false);

  // ==================
  // Actions
  // ==================

  /**
   * 开始思考（对应 think_chunk_start 事件）
   */
  const startThinking = (messageId: string) => {
    currentMessageId.value = messageId;
    currentThought.value = '';
    isThinking.value = true;
  };

  /**
   * 处理思考块（对应 think_chunk 事件）
   */
  const handleThinkChunk = (delta: string) => {
    currentThought.value += delta;
  };

  /**
   * 结束思考（对应 think_chunk_end 事件）
   */
  const endThinking = () => {
    if (currentMessageId.value && currentThought.value) {
      addStep(currentMessageId.value, {
        type: 'reasoning',
        content: currentThought.value,
        timestamp: Date.now(),
      });
    }
    currentThought.value = '';
    currentMessageId.value = null;
    isThinking.value = false;
  };

  /**
   * 添加思维步骤
   */
  const addStep = (messageId: string, step: ThinkingStep) => {
    const steps = stepsByMessage.value.get(messageId) || [];
    steps.push({
      ...step,
      id: step.id || `step-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`,
      messageId,
    });
    stepsByMessage.value.set(messageId, steps);
  };

  /**
   * 添加工具调用步骤
   */
  const addToolCallStep = (messageId: string, toolName: string, args: any) => {
    addStep(messageId, {
      type: 'tool_call',
      tool: { name: toolName, args },
      timestamp: Date.now(),
    });
  };

  /**
   * 添加工具结果步骤
   */
  const addToolResultStep = (messageId: string, result: any) => {
    addStep(messageId, {
      type: 'tool_result',
      result,
      timestamp: Date.now(),
    });
  };

  /**
   * 添加决策步骤
   */
  const addDecisionStep = (messageId: string, decision: string) => {
    addStep(messageId, {
      type: 'decision',
      content: decision,
      timestamp: Date.now(),
    });
  };

  /**
   * 添加审批步骤
   */
  const addApprovalStep = (messageId: string, toolName: string, args: any) => {
    addStep(messageId, {
      type: 'approval',
      tool: { name: toolName, args },
      timestamp: Date.now(),
    });
  };

  /**
   * 获取指定消息的思维步骤
   */
  const getSteps = (messageId: string): ThinkingStep[] => {
    return stepsByMessage.value.get(messageId) || [];
  };

  /**
   * 清除指定消息的思维步骤
   */
  const clearSteps = (messageId: string) => {
    stepsByMessage.value.delete(messageId);
  };

  /**
   * 清除所有思维步骤
   */
  const clearAllSteps = () => {
    stepsByMessage.value.clear();
    currentThought.value = '';
    currentMessageId.value = null;
    isThinking.value = false;
  };

  // ==================
  // Return
  // ==================

  return {
    // State
    stepsByMessage,
    currentThought,
    currentMessageId,
    isThinking,

    // Actions
    startThinking,
    handleThinkChunk,
    endThinking,
    addStep,
    addToolCallStep,
    addToolResultStep,
    addDecisionStep,
    addApprovalStep,
    getSteps,
    clearSteps,
    clearAllSteps,
  };
});
