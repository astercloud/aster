<template>
  <div :class="['thinking-block', { 'thinking-block-pending': pendingApproval }, { 'thinking-block-finished': isFinished }]">
    <!-- 头部 -->
    <button class="thinking-header" @click="toggleExpanded">
      <Icon :type="isExpanded ? 'chevron-down' : 'chevron-right'" size="sm" />
      <Icon type="brain" :class="['thinking-icon', { 'thinking-icon-active': !isFinished && !pendingApproval }, { 'thinking-icon-pending': pendingApproval }]" />
      <span class="thinking-title">思考中...</span>

      <!-- 状态标签 -->
      <span v-if="pendingApproval" class="status-badge status-pending">
        <Icon type="alert" size="sm" />
        等待审批
      </span>
      <span v-else-if="!isFinished" class="status-badge status-running">
        <span class="status-dot"></span>
        运行中
      </span>
    </button>

    <!-- 思考内容 -->
    <div v-if="isExpanded" class="thinking-content">
      <div v-for="(event, idx) in thoughts" :key="event.id || idx" :class="['thought-item', getEventClass(event)]">
        <!-- 事件头部 -->
        <div class="thought-header">
          <span :class="['thought-stage', getStageClass(event)]">
            <Icon :type="getStageIcon(event)" size="sm" />
            {{ event.stage }}
          </span>
          <span class="thought-time">
            {{ formatTime(event.timestamp) }}
          </span>
        </div>

        <!-- 推理过程 -->
        <div class="thought-body">
          <p v-if="event.reasoning" class="thought-reasoning">
            {{ event.reasoning }}
          </p>

          <div v-if="event.decision && !isToolEvent(event)" class="thought-decision">
            <Icon type="play" size="sm" />
            <span>{{ event.decision }}</span>
          </div>

          <!-- 工具调用 -->
          <div v-if="event.toolCall" class="tool-call">
            <div class="tool-call-header">
              <Icon type="terminal" size="sm" />
              <span>FUNCTION</span>
              <span class="tool-name">{{ event.toolCall.toolName }}</span>
            </div>
            <pre class="tool-code">{{ formatJSON(event.toolCall.args) }}</pre>
          </div>

          <!-- 工具结果 -->
          <div v-if="event.toolResult" class="tool-result">
            <div class="tool-result-header">
              <Icon type="check" size="sm" />
              <span>OUTPUT</span>
            </div>
            <pre class="tool-code">{{ formatJSON(event.toolResult.result) }}</pre>
          </div>
        </div>
      </div>

      <!-- 审批卡片 -->
      <div v-if="pendingApproval" class="approval-card">
        <div class="approval-header">
          <div class="approval-icon">
            <Icon type="alert" />
          </div>
          <div>
            <h4 class="approval-title">需要人工审批</h4>
            <p class="approval-desc">
              Agent 请求执行敏感操作
              <code class="approval-tool">{{ pendingApproval.toolName }}</code>
              。请确认是否允许。
            </p>
          </div>
        </div>

        <div class="approval-details">
          <div class="approval-details-header">
            <span>参数详情</span>
            <span class="approval-tool-name">{{ pendingApproval.toolName }}</span>
          </div>
          <pre class="approval-code">{{ formatJSON(pendingApproval.args) }}</pre>
        </div>

        <div class="approval-actions">
          <Button variant="primary" class="approval-btn-approve" @click="$emit('approve', pendingApproval)">
            <Icon type="check" size="sm" />
            批准执行
          </Button>
          <Button variant="secondary" class="approval-btn-reject" @click="$emit('reject')">
            <Icon type="close" size="sm" />
            拒绝
          </Button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch } from "vue";
import Icon from "../ChatUI/Icon.vue";
import Button from "../ChatUI/Button.vue";

interface ToolCallData {
  toolName: string;
  args: Record<string, any>;
}

interface ToolResultData {
  toolName: string;
  result: Record<string, any>;
}

interface ApprovalRequest {
  id: string;
  toolName: string;
  args: Record<string, any>;
}

interface ThinkAloudEvent {
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

interface Props {
  thoughts: ThinkAloudEvent[];
  isFinished: boolean;
  pendingApproval?: ApprovalRequest;
  defaultExpanded?: boolean;
}

const props = withDefaults(defineProps<Props>(), {
  defaultExpanded: false,
});

const emit = defineEmits<{
  approve: [request: ApprovalRequest];
  reject: [];
}>();

const isExpanded = ref(props.defaultExpanded);

// 自动展开仅当有审批请求时
watch([() => props.pendingApproval], () => {
  if (props.pendingApproval) {
    isExpanded.value = true;
  }
});

const toggleExpanded = () => {
  isExpanded.value = !isExpanded.value;
};

const isToolEvent = (event: ThinkAloudEvent) => {
  return !!event.toolCall || !!event.toolResult;
};

const getEventClass = (event: ThinkAloudEvent) => {
  if (event.toolCall) return "thought-item-tool-call";
  if (event.toolResult) return "thought-item-tool-result";
  if (event.stage === "Human in the Loop") return "thought-item-hitl";
  return "thought-item-thinking";
};

const getStageClass = (event: ThinkAloudEvent) => {
  if (event.toolCall) return "stage-tool-call";
  if (event.toolResult) return "stage-tool-result";
  if (event.stage === "Human in the Loop") return "stage-hitl";
  return "stage-thinking";
};

const getStageIcon = (event: ThinkAloudEvent) => {
  if (event.toolCall) return "terminal";
  if (event.toolResult) return "check";
  if (event.stage === "Human in the Loop") return "alert";
  return "brain";
};

const formatTime = (timestamp: string) => {
  const date = new Date(timestamp);
  return date.toLocaleTimeString("zh-CN", {
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
  });
};

const formatJSON = (obj: any) => {
  return JSON.stringify(obj, null, 2);
};
</script>

<style scoped>
.thinking-block {
  @apply rounded-lg border border-gray-200 dark:border-gray-700 overflow-hidden transition-all;
}

.thinking-block-pending {
  @apply border-amber-300 dark:border-amber-700 bg-amber-50 dark:bg-amber-900/20;
}

.thinking-block-finished {
  @apply opacity-90;
}

.thinking-header {
  @apply w-full flex items-center gap-2 px-3 py-2 text-xs font-medium text-gray-600 dark:text-gray-400 hover:bg-gray-100 dark:hover:bg-gray-800 transition-colors;
}

.thinking-icon {
  @apply text-gray-400 dark:text-gray-500;
}

.thinking-icon-active {
  @apply text-blue-500 dark:text-blue-400 animate-pulse;
}

.thinking-icon-pending {
  @apply text-amber-500 dark:text-amber-400 animate-pulse;
}

.thinking-title {
  @apply flex-1;
}

.status-badge {
  @apply flex items-center gap-1.5 px-2 py-1 rounded-full text-xs font-semibold;
}

.status-pending {
  @apply bg-amber-100 dark:bg-amber-900/30 text-amber-700 dark:text-amber-300 border border-amber-200 dark:border-amber-800 animate-pulse;
}

.status-running {
  @apply bg-blue-50 dark:bg-blue-900/30 text-blue-600 dark:text-blue-400 border border-blue-100 dark:border-blue-800;
}

.status-dot {
  @apply w-1.5 h-1.5 rounded-full bg-blue-500 dark:bg-blue-400 animate-pulse;
}

.thinking-content {
  @apply px-3 pb-3 pt-1 space-y-3 border-t border-gray-100 dark:border-gray-800 bg-white dark:bg-gray-900;
}

.thought-item {
  @apply relative pl-4 border-l-2 ml-1 transition-all;
}

.thought-item-thinking {
  @apply border-gray-200 dark:border-gray-700;
}

.thought-item-tool-call {
  @apply border-indigo-300 dark:border-indigo-700;
}

.thought-item-tool-result {
  @apply border-emerald-300 dark:border-emerald-700;
}

.thought-item-hitl {
  @apply border-amber-400 dark:border-amber-600;
}

.thought-header {
  @apply flex items-center justify-between mb-1;
}

.thought-stage {
  @apply flex items-center gap-1 text-xs font-bold px-1.5 py-0.5 rounded;
}

.stage-thinking {
  @apply bg-gray-100 dark:bg-gray-800 text-gray-600 dark:text-gray-400;
}

.stage-tool-call {
  @apply bg-indigo-100 dark:bg-indigo-900/30 text-indigo-700 dark:text-indigo-300;
}

.stage-tool-result {
  @apply bg-emerald-100 dark:bg-emerald-900/30 text-emerald-700 dark:text-emerald-300;
}

.stage-hitl {
  @apply bg-amber-100 dark:bg-amber-900/30 text-amber-700 dark:text-amber-300;
}

.thought-time {
  @apply text-xs text-gray-400 dark:text-gray-500 tabular-nums;
}

.thought-body {
  @apply space-y-2;
}

.thought-reasoning {
  @apply text-xs text-gray-600 dark:text-gray-400 leading-relaxed;
}

.thought-decision {
  @apply flex items-center gap-1.5 text-xs text-gray-800 dark:text-gray-200 font-medium;
}

.tool-call {
  @apply bg-gray-900 dark:bg-gray-950 rounded-md overflow-hidden shadow-sm;
}

.tool-call-header {
  @apply flex items-center gap-2 bg-gray-800 dark:bg-gray-900 px-2 py-1.5 border-b border-gray-700 dark:border-gray-800 text-xs font-bold text-indigo-300 dark:text-indigo-400;
}

.tool-name {
  @apply ml-auto bg-indigo-500/20 text-indigo-200 dark:text-indigo-300 px-1.5 py-0.5 rounded font-mono border border-indigo-500/30;
}

.tool-code {
  @apply p-2 font-mono text-xs text-indigo-100 dark:text-indigo-200 leading-relaxed overflow-x-auto;
}

.tool-result {
  @apply bg-emerald-50 dark:bg-emerald-900/20 border border-emerald-200 dark:border-emerald-800 rounded-md overflow-hidden;
}

.tool-result-header {
  @apply flex items-center gap-2 bg-emerald-100 dark:bg-emerald-900/30 px-2 py-1.5 border-b border-emerald-200 dark:border-emerald-800 text-xs font-bold text-emerald-700 dark:text-emerald-300;
}

.approval-card {
  @apply mt-4 bg-amber-50 dark:bg-amber-900/20 border border-amber-200 dark:border-amber-800 rounded-lg p-4 shadow-sm;
  animation: slideIn 0.3s ease-out;
}

.approval-header {
  @apply flex items-start gap-3 mb-3;
}

.approval-icon {
  @apply p-2 bg-amber-100 dark:bg-amber-900/30 rounded-full text-amber-600 dark:text-amber-400;
}

.approval-title {
  @apply text-sm font-bold text-amber-900 dark:text-amber-100;
}

.approval-desc {
  @apply text-xs text-amber-700 dark:text-amber-300 mt-0.5 leading-relaxed;
}

.approval-tool {
  @apply font-mono font-bold bg-amber-100 dark:bg-amber-900/30 px-1 rounded;
}

.approval-details {
  @apply bg-white dark:bg-gray-900 border border-amber-200 dark:border-amber-800 rounded-md mb-4 overflow-hidden;
}

.approval-details-header {
  @apply flex items-center justify-between bg-amber-50 dark:bg-amber-900/20 px-3 py-2 border-b border-amber-100 dark:border-amber-800 text-xs font-semibold text-amber-800 dark:text-amber-200;
}

.approval-tool-name {
  @apply font-mono opacity-70;
}

.approval-code {
  @apply p-3 font-mono text-xs text-gray-600 dark:text-gray-400 overflow-x-auto bg-gray-50 dark:bg-gray-950;
}

.approval-actions {
  @apply flex gap-3;
}

.approval-btn-approve {
  @apply flex-1 bg-emerald-600 hover:bg-emerald-700 text-white;
}

.approval-btn-reject {
  @apply flex-1 bg-white hover:bg-red-50 text-red-600 border-red-200 hover:border-red-300;
}

@keyframes slideIn {
  from {
    opacity: 0;
    transform: translateY(-10px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}
</style>
