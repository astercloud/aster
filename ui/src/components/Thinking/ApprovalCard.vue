<template>
  <div class="approval-card">
    <div class="approval-header">
      <div class="approval-icon">
        <svg
          class="w-5 h-5 text-amber-600"
          fill="none"
          stroke="currentColor"
          viewBox="0 0 24 24"
        >
          <path
            stroke-linecap="round"
            stroke-linejoin="round"
            stroke-width="2"
            d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"
          />
        </svg>
      </div>
      <div>
        <h4 class="approval-title">需要人工审批</h4>
        <p class="approval-description">
          Agent 请求执行敏感操作
          <code class="tool-name">{{ request.toolName }}</code>
          ，请仔细确认后再批准。
        </p>
      </div>
    </div>

    <!-- 参数详情 -->
    <div class="approval-details">
      <div class="detail-header">
        <span>参数详情</span>
        <code class="detail-tool-name">{{ request.toolName }}</code>
      </div>
      <pre class="detail-content">{{ formatArgs(request.args) }}</pre>
    </div>

    <!-- 原因说明 -->
    <div v-if="request.reason" class="approval-reason">
      <div class="reason-label">原因</div>
      <p class="reason-text">{{ request.reason }}</p>
    </div>

    <!-- 操作按钮 -->
    <div class="approval-actions">
      <button
        @click="handleApprove"
        class="btn-approve"
        :disabled="isProcessing"
      >
        <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path
            stroke-linecap="round"
            stroke-linejoin="round"
            stroke-width="2"
            d="M5 13l4 4L19 7"
          />
        </svg>
        {{ isProcessing ? '处理中...' : '批准执行' }}
      </button>
      <button
        @click="handleReject"
        class="btn-reject"
        :disabled="isProcessing"
      >
        <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path
            stroke-linecap="round"
            stroke-linejoin="round"
            stroke-width="2"
            d="M6 18L18 6M6 6l12 12"
          />
        </svg>
        {{ isProcessing ? '处理中...' : '拒绝' }}
      </button>
    </div>
  </div>
</template>

<script lang="ts">
import { defineComponent, ref } from 'vue';
import type { ApprovalRequest } from '@/types/approval';
import type { PropType } from 'vue';

export default defineComponent({
  name: 'ApprovalCard',
  props: {
    request: {
      type: Object as PropType<ApprovalRequest>,
      required: true,
    },
  },
  emits: {
    approve: () => true,
    reject: () => true,
  },
  setup(props, { emit }) {
    const isProcessing = ref(false);

    const formatArgs = (args: Record<string, any>): string => {
      try {
        return JSON.stringify(args, null, 2);
      } catch (e) {
        return String(args);
      }
    };

    const handleApprove = async () => {
      if (isProcessing.value) return;
      isProcessing.value = true;
      try {
        emit('approve');
      } finally {
        // 延迟重置，避免按钮闪烁
        setTimeout(() => {
          isProcessing.value = false;
        }, 500);
      }
    };

    const handleReject = async () => {
      if (isProcessing.value) return;
      isProcessing.value = true;
      try {
        emit('reject');
      } finally {
        setTimeout(() => {
          isProcessing.value = false;
        }, 500);
      }
    };

    return {
      isProcessing,
      formatArgs,
      handleApprove,
      handleReject,
    };
  },
});
</script>

<style scoped>
.approval-card {
  @apply mt-4 bg-amber-50 border border-amber-200 rounded-lg p-4 shadow-sm;
  animation: fadeIn 0.3s ease-in-out, slideInFromTop 0.3s ease-out;
}

@keyframes fadeIn {
  from {
    opacity: 0;
  }
  to {
    opacity: 1;
  }
}

@keyframes slideInFromTop {
  from {
    transform: translateY(-8px);
  }
  to {
    transform: translateY(0);
  }
}

.approval-header {
  @apply flex items-start gap-3 mb-3;
}

.approval-icon {
  @apply p-2 bg-amber-100 rounded-full flex-shrink-0;
}

.approval-title {
  @apply text-sm font-bold text-amber-900;
}

.approval-description {
  @apply text-xs text-amber-700 mt-0.5 leading-relaxed;
}

.tool-name {
  @apply font-mono font-bold bg-amber-100 px-1 rounded text-amber-800;
}

.approval-details {
  @apply bg-white border border-amber-200 rounded-md mb-3 overflow-hidden;
}

.detail-header {
  @apply bg-amber-50/50 px-3 py-2 border-b border-amber-100 text-xs font-semibold text-amber-800 flex justify-between items-center;
}

.detail-tool-name {
  @apply font-mono opacity-70;
}

.detail-content {
  @apply p-3 font-mono text-xs text-slate-600 overflow-x-auto bg-slate-50;
}

.approval-reason {
  @apply mb-3 p-3 bg-amber-100/30 rounded border border-amber-200;
}

.reason-label {
  @apply text-xs font-semibold text-amber-800 mb-1;
}

.reason-text {
  @apply text-xs text-amber-700 leading-relaxed;
}

.approval-actions {
  @apply flex gap-3;
}

.btn-approve,
.btn-reject {
  @apply flex-1 flex items-center justify-center gap-2 text-xs font-bold py-2.5 rounded-md shadow-sm transition-all duration-200;
}

.btn-approve {
  @apply bg-emerald-600 hover:bg-emerald-700 text-white ring-1 ring-emerald-700;
}

.btn-approve:disabled {
  @apply bg-emerald-400 cursor-not-allowed;
}

.btn-approve:active:not(:disabled) {
  @apply scale-95;
}

.btn-reject {
  @apply bg-white hover:bg-red-50 text-red-600 border border-red-200 hover:border-red-300;
}

.btn-reject:disabled {
  @apply bg-gray-100 text-gray-400 cursor-not-allowed;
}

.btn-reject:active:not(:disabled) {
  @apply scale-95;
}
</style>
