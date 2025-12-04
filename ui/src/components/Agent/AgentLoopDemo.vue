<script setup lang="ts">
/**
 * AgentLoopDemo - 演示完整 Agent Loop + HITL 集成
 *
 * 功能:
 * - 重试逻辑 (通过后端 ModelFallbackManager)
 * - Human-in-the-Loop 审批流程
 * - 真实工具执行
 * - 流式响应
 */

import { ref, computed } from "vue";
import { useAgentLoop } from "@/composables/useAgentLoop";
import type { ThinkAloudEvent, ApprovalRequest } from "@/composables/useAgentLoop";

// Props
const props = defineProps<{
  modelConfig?: {
    provider?: string;
    model?: string;
  };
}>();

// 思考事件列表
const thinkEvents = ref<ThinkAloudEvent[]>([]);

// Agent Loop
const { isRunning, isPaused, currentOutput, pendingApproval, isConnected, execute, approveAndResume, rejectTool, cancel } = useAgentLoop({
  modelConfig: props.modelConfig,
  sensitiveTools: ["Edit", "Write", "bash", "fs_write"],
  maxRetries: 3,
  maxLoops: 10,
  onThink: (event) => {
    thinkEvents.value.push(event);
  },
  onApprovalRequired: (request) => {
    console.log("Approval required:", request);
  },
  onToolStart: (toolName, args) => {
    console.log("Tool started:", toolName, args);
  },
  onToolEnd: (toolName, result) => {
    console.log("Tool ended:", toolName, result);
  },
  onTextDelta: (delta) => {
    // 已通过 currentOutput 响应式更新
  },
  onComplete: (result) => {
    console.log("Execution complete:", result.status);
  },
  onError: (error) => {
    console.error("Execution error:", error);
  },
});

// 用户输入
const userInput = ref("");
const rejectReason = ref("");

// 发送消息
const sendMessage = async () => {
  if (!userInput.value.trim() || isRunning.value) return;

  thinkEvents.value = [];
  const input = userInput.value;
  userInput.value = "";

  await execute(input);
};

// 批准工具
const handleApprove = async () => {
  if (!pendingApproval.value) return;
  await approveAndResume(pendingApproval.value.id);
};

// 拒绝工具
const handleReject = () => {
  if (!pendingApproval.value) return;
  rejectTool(pendingApproval.value.id, rejectReason.value || "用户拒绝");
  rejectReason.value = "";
};

// 取消执行
const handleCancel = () => {
  cancel();
};

// 格式化工具参数
const formatArgs = (args: Record<string, any>): string => {
  return JSON.stringify(args, null, 2);
};
</script>

<template>
  <div class="agent-loop-demo">
    <!-- 连接状态 -->
    <div class="connection-status" :class="{ connected: isConnected }">
      <span class="status-dot"></span>
      {{ isConnected ? "已连接" : "未连接" }}
    </div>

    <!-- 思考过程 -->
    <div class="thinking-panel" v-if="thinkEvents.length > 0">
      <h3>🧠 思考过程</h3>
      <div class="think-events">
        <div
          v-for="event in thinkEvents"
          :key="event.id"
          class="think-event"
          :class="{
            'is-approval': event.approvalRequest,
            'is-tool': event.toolCall || event.toolResult,
          }"
        >
          <div class="event-header">
            <span class="event-stage">{{ event.stage }}</span>
            <span class="event-time">{{ new Date(event.timestamp).toLocaleTimeString() }}</span>
          </div>
          <div class="event-reasoning">{{ event.reasoning }}</div>
          <div class="event-decision">→ {{ event.decision }}</div>

          <!-- 工具调用详情 -->
          <div v-if="event.toolCall" class="tool-details">
            <code>{{ event.toolCall.toolName }}({{ formatArgs(event.toolCall.args) }})</code>
          </div>

          <!-- 工具结果 -->
          <div v-if="event.toolResult" class="tool-result">
            <pre>{{ formatArgs(event.toolResult.result) }}</pre>
          </div>
        </div>
      </div>
    </div>

    <!-- 审批面板 -->
    <div class="approval-panel" v-if="pendingApproval">
      <div class="approval-header">
        <span class="approval-icon">⚠️</span>
        <h3>需要人工审批</h3>
      </div>
      <div class="approval-content">
        <p>
          工具 <strong>{{ pendingApproval.toolName }}</strong> 被标记为敏感操作
        </p>
        <div class="approval-args">
          <h4>参数:</h4>
          <pre>{{ formatArgs(pendingApproval.args) }}</pre>
        </div>
        <div class="approval-actions">
          <input v-model="rejectReason" placeholder="拒绝原因 (可选)" class="reject-reason-input" />
          <button class="btn btn-approve" @click="handleApprove" :disabled="isRunning && !isPaused">✓ 批准</button>
          <button class="btn btn-reject" @click="handleReject" :disabled="isRunning && !isPaused">✗ 拒绝</button>
        </div>
      </div>
    </div>

    <!-- 输出面板 -->
    <div class="output-panel">
      <h3>📝 输出</h3>
      <div class="output-content" v-html="currentOutput || '<em>等待输出...</em>'"></div>
    </div>

    <!-- 输入面板 -->
    <div class="input-panel">
      <textarea v-model="userInput" placeholder="输入你的请求..." :disabled="isRunning" @keydown.enter.ctrl="sendMessage" rows="3"></textarea>
      <div class="input-actions">
        <button class="btn btn-primary" @click="sendMessage" :disabled="!userInput.trim() || isRunning">
          {{ isRunning ? (isPaused ? "等待审批..." : "执行中...") : "发送" }}
        </button>
        <button class="btn btn-secondary" @click="handleCancel" :disabled="!isRunning">取消</button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.agent-loop-demo {
  display: flex;
  flex-direction: column;
  gap: 16px;
  padding: 16px;
  max-width: 800px;
  margin: 0 auto;
}

.connection-status {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 14px;
  color: #666;
}

.connection-status.connected {
  color: #22c55e;
}

.status-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: #ef4444;
}

.connection-status.connected .status-dot {
  background: #22c55e;
}

.thinking-panel,
.approval-panel,
.output-panel {
  background: #f8f9fa;
  border-radius: 8px;
  padding: 16px;
}

.thinking-panel h3,
.approval-panel h3,
.output-panel h3 {
  margin: 0 0 12px 0;
  font-size: 16px;
}

.think-events {
  display: flex;
  flex-direction: column;
  gap: 12px;
  max-height: 300px;
  overflow-y: auto;
}

.think-event {
  background: white;
  border-radius: 6px;
  padding: 12px;
  border-left: 3px solid #3b82f6;
}

.think-event.is-approval {
  border-left-color: #f59e0b;
  background: #fffbeb;
}

.think-event.is-tool {
  border-left-color: #8b5cf6;
}

.event-header {
  display: flex;
  justify-content: space-between;
  margin-bottom: 8px;
}

.event-stage {
  font-weight: 600;
  color: #1f2937;
}

.event-time {
  font-size: 12px;
  color: #9ca3af;
}

.event-reasoning {
  color: #4b5563;
  margin-bottom: 4px;
}

.event-decision {
  color: #059669;
  font-style: italic;
}

.tool-details,
.tool-result {
  margin-top: 8px;
  background: #f3f4f6;
  padding: 8px;
  border-radius: 4px;
  font-size: 12px;
  overflow-x: auto;
}

.tool-details code,
.tool-result pre {
  margin: 0;
  white-space: pre-wrap;
  word-break: break-all;
}

.approval-panel {
  background: #fef3c7;
  border: 1px solid #f59e0b;
}

.approval-header {
  display: flex;
  align-items: center;
  gap: 8px;
}

.approval-icon {
  font-size: 24px;
}

.approval-args {
  background: white;
  padding: 12px;
  border-radius: 6px;
  margin: 12px 0;
}

.approval-args h4 {
  margin: 0 0 8px 0;
  font-size: 14px;
}

.approval-args pre {
  margin: 0;
  font-size: 12px;
  white-space: pre-wrap;
}

.approval-actions {
  display: flex;
  gap: 8px;
  align-items: center;
}

.reject-reason-input {
  flex: 1;
  padding: 8px 12px;
  border: 1px solid #d1d5db;
  border-radius: 6px;
  font-size: 14px;
}

.output-panel {
  min-height: 100px;
}

.output-content {
  white-space: pre-wrap;
  line-height: 1.6;
}

.input-panel {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.input-panel textarea {
  width: 100%;
  padding: 12px;
  border: 1px solid #d1d5db;
  border-radius: 8px;
  font-size: 14px;
  resize: vertical;
}

.input-panel textarea:focus {
  outline: none;
  border-color: #3b82f6;
  box-shadow: 0 0 0 3px rgba(59, 130, 246, 0.1);
}

.input-actions {
  display: flex;
  gap: 8px;
  justify-content: flex-end;
}

.btn {
  padding: 8px 16px;
  border-radius: 6px;
  font-size: 14px;
  font-weight: 500;
  cursor: pointer;
  border: none;
  transition: all 0.2s;
}

.btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.btn-primary {
  background: #3b82f6;
  color: white;
}

.btn-primary:hover:not(:disabled) {
  background: #2563eb;
}

.btn-secondary {
  background: #e5e7eb;
  color: #374151;
}

.btn-secondary:hover:not(:disabled) {
  background: #d1d5db;
}

.btn-approve {
  background: #22c55e;
  color: white;
}

.btn-approve:hover:not(:disabled) {
  background: #16a34a;
}

.btn-reject {
  background: #ef4444;
  color: white;
}

.btn-reject:hover:not(:disabled) {
  background: #dc2626;
}
</style>
