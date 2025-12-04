/**
 * 工具调用状态管理
 *
 * 管理所有工具的调用状态、进度、结果等
 */

import { defineStore } from "pinia";
import { ref, computed } from "vue";
import type { ToolCallSnapshot } from "@/types/message";

export const useToolsStore = defineStore("tools", () => {
  // ==================
  // State
  // ==================

  // 工具调用状态（使用 Map 提高查找性能）
  const toolRuns = ref<Map<string, ToolCallSnapshot>>(new Map());

  // ==================
  // Getters
  // ==================

  // 工具调用列表（用于 UI 渲染）
  const toolRunsList = computed(() => Array.from(toolRuns.value.values()));

  // 运行中的工具数量
  const runningToolsCount = computed(() => {
    return Array.from(toolRuns.value.values()).filter((tool) => tool.state === "executing").length;
  });

  // 是否有工具正在运行
  const hasRunningTools = computed(() => runningToolsCount.value > 0);

  // ==================
  // Actions
  // ==================

  /**
   * 更新工具状态（细粒度更新）
   */
  const updateTool = (id: string, updates: Partial<ToolCallSnapshot>) => {
    const prev =
      toolRuns.value.get(id) ||
      ({
        id,
        name: "",
        state: "pending",
      } as ToolCallSnapshot);

    toolRuns.value.set(id, {
      ...prev,
      ...updates,
      updated_at: new Date().toISOString(),
    });
  };

  /**
   * 处理工具开始事件
   */
  const handleToolStart = (call: ToolCallSnapshot) => {
    toolRuns.value.set(call.id, {
      ...call,
      state: "executing",
      started_at: call.started_at || new Date().toISOString(),
      updated_at: new Date().toISOString(),
    });
  };

  /**
   * 处理工具进度事件
   */
  const handleToolProgress = (id: string, progress: number, message?: string, metadata?: Record<string, any>) => {
    updateTool(id, {
      progress,
      state: "executing",
      intermediate: metadata ? { ...metadata, message } : undefined,
    });
  };

  /**
   * 处理工具中间结果事件
   */
  const handleToolIntermediate = (id: string, label: string, data: any) => {
    const prev = toolRuns.value.get(id);
    if (prev) {
      updateTool(id, {
        intermediate: {
          ...(prev.intermediate || {}),
          [label]: data,
        },
      });
    }
  };

  /**
   * 处理工具结束事件
   */
  const handleToolEnd = (call: ToolCallSnapshot) => {
    toolRuns.value.set(call.id, {
      ...call,
      state: "completed",
      progress: 1,
      updated_at: new Date().toISOString(),
    });
  };

  /**
   * 处理工具错误事件
   */
  const handleToolError = (id: string, error: string) => {
    updateTool(id, {
      state: "failed",
      error,
    });
  };

  /**
   * 处理工具取消事件
   */
  const handleToolCancelled = (id: string, reason?: string) => {
    updateTool(id, {
      state: "cancelled",
      error: reason,
    });
  };

  /**
   * 获取指定工具的状态
   */
  const getTool = (id: string): ToolCallSnapshot | undefined => {
    return toolRuns.value.get(id);
  };

  /**
   * 移除工具记录
   */
  const removeTool = (id: string) => {
    toolRuns.value.delete(id);
  };

  /**
   * 清除所有工具记录
   */
  const clearAllTools = () => {
    toolRuns.value.clear();
  };

  /**
   * 清除已完成的工具记录
   */
  const clearCompletedTools = () => {
    for (const [id, tool] of toolRuns.value) {
      if (tool.state === "completed" || tool.state === "failed" || tool.state === "cancelled") {
        toolRuns.value.delete(id);
      }
    }
  };

  // ==================
  // Return
  // ==================

  return {
    // State
    toolRuns,

    // Getters
    toolRunsList,
    runningToolsCount,
    hasRunningTools,

    // Actions
    updateTool,
    handleToolStart,
    handleToolProgress,
    handleToolIntermediate,
    handleToolEnd,
    handleToolError,
    handleToolCancelled,
    getTool,
    removeTool,
    clearAllTools,
    clearCompletedTools,
  };
});
