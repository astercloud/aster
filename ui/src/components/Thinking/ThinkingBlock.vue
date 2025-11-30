<template>
  <div
    :class="[
      'thinking-block',
      {
        'thinking-block--pending-approval': hasPendingApproval,
        'thinking-block--active': isActive,
      },
    ]"
  >
    <!-- 折叠视图 -->
    <button v-if="!isExpanded" @click="expand" class="thinking-compact">
      <div class="flex items-center gap-2">
        <svg
          :class="['w-4 h-4', isActive ? 'text-blue-500 animate-pulse' : 'text-slate-500']"
          fill="none"
          stroke="currentColor"
          viewBox="0 0 24 24"
        >
          <path
            stroke-linecap="round"
            stroke-linejoin="round"
            stroke-width="2"
            d="M9.663 17h4.673M12 3v1m6.364 1.636l-.707.707M21 12h-1M4 12H3m3.343-5.657l-.707-.707m2.828 9.9a5 5 0 117.072 0l-.548.547A3.374 3.374 0 0014 18.469V19a2 2 0 11-4 0v-.531c0-.895-.356-1.754-.988-2.386l-.548-.547z"
          />
        </svg>
        <span class="text-sm font-medium text-slate-700">思考过程</span>
        <span class="text-xs text-slate-500">({{ steps.length }} 步)</span>
      </div>
      <svg class="w-4 h-4 text-slate-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7" />
      </svg>
    </button>

    <!-- 展开视图 -->
    <div v-else class="thinking-expanded">
      <!-- 头部 -->
      <div class="thinking-header">
        <div class="flex items-center gap-2">
          <svg
            :class="['w-5 h-5', isActive ? 'text-blue-500 animate-pulse' : 'text-slate-500']"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
          >
            <path
              stroke-linecap="round"
              stroke-linejoin="round"
              stroke-width="2"
              d="M9.663 17h4.673M12 3v1m6.364 1.636l-.707.707M21 12h-1M4 12H3m3.343-5.657l-.707-.707m2.828 9.9a5 5 0 117.072 0l-.548.547A3.374 3.374 0 0014 18.469V19a2 2 0 11-4 0v-.531c0-.895-.356-1.754-.988-2.386l-.548-.547z"
            />
          </svg>
          <h3 class="text-base font-semibold text-slate-800 dark:text-slate-200">思考过程</h3>

          <!-- 状态标签 -->
          <span v-if="isActive" class="status-badge status-badge--active">运行中</span>
          <span
            v-else-if="hasPendingApproval"
            class="status-badge status-badge--warning animate-pulse"
          >
            <svg class="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path
                stroke-linecap="round"
                stroke-linejoin="round"
                stroke-width="2"
                d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"
              />
            </svg>
            等待审批
          </span>
          <span v-else class="status-badge status-badge--completed">已完成</span>
        </div>

        <!-- 折叠按钮 -->
        <button @click="collapse" class="collapse-btn">
          <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 15l7-7 7 7" />
          </svg>
        </button>
      </div>

      <!-- 时间线 -->
      <ThinkingTimeline :steps="steps" />

      <!-- 审批卡片 -->
      <ApprovalCard
        v-if="pendingApproval"
        :request="pendingApproval"
        @approve="handleApprove"
        @reject="handleReject"
      />

      <!-- 摘要 -->
      <div v-if="!isActive && summary" class="thinking-summary">
        <svg
          class="w-4 h-4 text-blue-600"
          fill="none"
          stroke="currentColor"
          viewBox="0 0 24 24"
        >
          <path
            stroke-linecap="round"
            stroke-linejoin="round"
            stroke-width="2"
            d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z"
          />
        </svg>
        <span class="text-sm text-slate-700 dark:text-slate-300">{{ summary }}</span>
      </div>
    </div>
  </div>
</template>

<script lang="ts">
import { defineComponent, ref, computed, watch } from 'vue';
import { useApprovalStore } from '@/stores/approval';
import ThinkingTimeline from './ThinkingTimeline.vue';
import ApprovalCard from './ApprovalCard.vue';
import type { ThinkingStep } from '@/types/thinking';
import type { PropType } from 'vue';

export default defineComponent({
  name: 'ThinkingBlock',
  components: {
    ThinkingTimeline,
    ApprovalCard,
  },
  props: {
    messageId: {
      type: String,
      required: true,
    },
    steps: {
      type: Array as PropType<ThinkingStep[]>,
      required: true,
    },
    isActive: {
      type: Boolean,
      default: false,
    },
    summary: {
      type: String,
      default: undefined,
    },
  },
  setup(props) {
    const approvalStore = useApprovalStore();
    const isExpanded = ref(false);

    // 检查是否有待审批请求（关联此消息）
    const pendingApproval = computed(() => {
      return approvalStore.getApprovalByMessage(props.messageId);
    });

    const hasPendingApproval = computed(() => !!pendingApproval.value);

    // 自动展开逻辑：运行中或有审批请求时自动展开，完成后自动折叠
    watch(
      [() => props.isActive, hasPendingApproval],
      ([active, pending], [wasActive]) => {
        if (active || pending) {
          isExpanded.value = true;
        } else if (wasActive && !active && !pending) {
          // 思考刚完成，自动折叠
          isExpanded.value = false;
        }
      },
      { immediate: true }
    );

    const expand = () => {
      isExpanded.value = true;
    };

    const collapse = () => {
      isExpanded.value = false;
    };

    const handleApprove = () => {
      if (pendingApproval.value) {
        approvalStore.approve(pendingApproval.value.id);
      }
    };

    const handleReject = () => {
      if (pendingApproval.value) {
        approvalStore.reject(pendingApproval.value.id);
      }
    };

    return {
      isExpanded,
      pendingApproval,
      hasPendingApproval,
      expand,
      collapse,
      handleApprove,
      handleReject,
    };
  },
});
</script>

<style scoped>
.thinking-block {
  @apply my-2;
}

.thinking-block--pending-approval {
  @apply animate-pulse;
}

.thinking-compact {
  @apply w-full flex items-center justify-between px-4 py-2 bg-slate-50 dark:bg-slate-800 border border-slate-200 dark:border-slate-700 rounded-lg hover:bg-slate-100 dark:hover:bg-slate-700 transition-colors;
}

.thinking-expanded {
  @apply bg-slate-50 dark:bg-slate-800 border border-slate-200 dark:border-slate-700 rounded-lg overflow-hidden;
}

.thinking-block--pending-approval .thinking-expanded {
  @apply border-amber-300 bg-amber-50/30 shadow-lg;
}

.thinking-block--active .thinking-expanded {
  @apply border-blue-300;
}

.thinking-header {
  @apply flex items-center justify-between px-4 py-3 border-b border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-900;
}

.status-badge {
  @apply text-xs px-2 py-0.5 rounded-full font-medium flex items-center gap-1;
}

.status-badge--active {
  @apply bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-400;
}

.status-badge--warning {
  @apply bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-400;
}

.status-badge--completed {
  @apply bg-slate-200 text-slate-600 dark:bg-slate-700 dark:text-slate-400;
}

.collapse-btn {
  @apply p-1 hover:bg-slate-100 dark:hover:bg-slate-700 rounded transition-colors;
}

.thinking-summary {
  @apply flex items-center gap-2 px-4 py-3 bg-blue-50 dark:bg-blue-900/10 border-t border-slate-200 dark:border-slate-700;
}
</style>
