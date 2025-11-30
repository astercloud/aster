<template>
  <div
    :class="[
      'workflow-step',
      `workflow-step--${step.status}`,
      { 'workflow-step--clickable': isClickable }
    ]"
    @click="handleClick"
  >
    <!-- 步骤指示器 -->
    <div class="step-indicator">
      <!-- 状态图标 -->
      <div :class="['step-icon', stepIconClass]">
        <!-- 自定义图标 -->
        <component v-if="step.icon" :is="step.icon" class="w-5 h-5" />

        <!-- 默认状态图标 -->
        <template v-else>
          <!-- Pending -->
          <svg
            v-if="step.status === 'pending'"
            class="w-5 h-5"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
          >
            <path
              stroke-linecap="round"
              stroke-linejoin="round"
              stroke-width="2"
              d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z"
            />
          </svg>

          <!-- Active -->
          <svg
            v-else-if="step.status === 'active'"
            class="w-5 h-5 animate-pulse"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
          >
            <path
              stroke-linecap="round"
              stroke-linejoin="round"
              stroke-width="2"
              d="M13 10V3L4 14h7v7l9-11h-7z"
            />
          </svg>

          <!-- Completed -->
          <svg
            v-else-if="step.status === 'completed'"
            class="w-5 h-5"
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

          <!-- Failed -->
          <svg
            v-else-if="step.status === 'failed'"
            class="w-5 h-5"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
          >
            <path
              stroke-linecap="round"
              stroke-linejoin="round"
              stroke-width="2"
              d="M10 14l2-2m0 0l2-2m-2 2l-2-2m2 2l2 2m7-2a9 9 0 11-18 0 9 9 0 0118 0z"
            />
          </svg>
        </template>
      </div>

      <!-- 连接线 -->
      <div v-if="!isLast" :class="['step-connector', stepConnectorClass]"></div>
    </div>

    <!-- 步骤内容 -->
    <div class="step-content">
      <div class="step-header">
        <h4 class="step-title">{{ step.title }}</h4>
        <span :class="['step-badge', stepBadgeClass]">
          {{ stepStatusLabel }}
        </span>
      </div>

      <p v-if="step.description" class="step-description">
        {{ step.description }}
      </p>

      <!-- 步骤操作 -->
      <div v-if="step.actions && step.actions.length > 0" class="step-actions">
        <button
          v-for="action in step.actions"
          :key="action.label"
          @click.stop="handleAction(action)"
          :class="['action-button', `action-button--${action.type}`]"
        >
          {{ action.label }}
        </button>
      </div>

      <!-- 元数据 -->
      <div v-if="step.metadata && showMetadata" class="step-metadata">
        <div
          v-for="(value, key) in step.metadata"
          :key="key"
          class="metadata-item"
        >
          <span class="metadata-key">{{ key }}:</span>
          <span class="metadata-value">{{ formatMetadataValue(value) }}</span>
        </div>
      </div>
    </div>
  </div>
</template>

<script lang="ts">
import { defineComponent, computed, type PropType } from 'vue';
import type { WorkflowStep, WorkflowAction } from '@/types/workflow';

export default defineComponent({
  name: 'WorkflowStep',
  props: {
    step: {
      type: Object as PropType<WorkflowStep>,
      required: true,
    },
    isLast: {
      type: Boolean,
      default: false,
    },
    isClickable: {
      type: Boolean,
      default: false,
    },
    showMetadata: {
      type: Boolean,
      default: false,
    },
  },
  emits: {
    click: (step: WorkflowStep) => true,
    action: (action: WorkflowAction, step: WorkflowStep) => true,
  },
  setup(props, { emit }) {
    const stepIconClass = computed(() => {
      const classes: Record<string, string> = {
        pending: 'step-icon--pending',
        active: 'step-icon--active',
        completed: 'step-icon--completed',
        failed: 'step-icon--failed',
      };
      return classes[props.step.status] || 'step-icon--pending';
    });

    const stepConnectorClass = computed(() => {
      // 如果当前步骤已完成,连接线也显示为完成状态
      if (props.step.status === 'completed') {
        return 'step-connector--completed';
      }
      return 'step-connector--default';
    });

    const stepBadgeClass = computed(() => {
      const classes: Record<string, string> = {
        pending: 'step-badge--pending',
        active: 'step-badge--active',
        completed: 'step-badge--completed',
        failed: 'step-badge--failed',
      };
      return classes[props.step.status] || 'step-badge--pending';
    });

    const stepStatusLabel = computed(() => {
      const labels: Record<string, string> = {
        pending: '待执行',
        active: '进行中',
        completed: '已完成',
        failed: '失败',
      };
      return labels[props.step.status] || props.step.status;
    });

    const handleClick = () => {
      if (props.isClickable) {
        emit('click', props.step);
      }
    };

    const handleAction = (action: WorkflowAction) => {
      emit('action', action, props.step);
    };

    const formatMetadataValue = (value: any): string => {
      if (typeof value === 'object') {
        return JSON.stringify(value);
      }
      return String(value);
    };

    return {
      stepIconClass,
      stepConnectorClass,
      stepBadgeClass,
      stepStatusLabel,
      handleClick,
      handleAction,
      formatMetadataValue,
    };
  },
});
</script>

<style scoped>
.workflow-step {
  @apply flex gap-4 mb-6;
}

.workflow-step--clickable {
  @apply cursor-pointer transition-all;
}

.workflow-step--clickable:hover {
  @apply scale-[1.02];
}

.step-indicator {
  @apply flex flex-col items-center flex-shrink-0;
}

.step-icon {
  @apply w-12 h-12 rounded-full flex items-center justify-center flex-shrink-0 transition-all;
}

.step-icon--pending {
  @apply bg-slate-100 dark:bg-slate-800 text-slate-400 dark:text-slate-500;
}

.step-icon--active {
  @apply bg-blue-100 dark:bg-blue-900/30 text-blue-600 dark:text-blue-400 shadow-lg ring-4 ring-blue-100 dark:ring-blue-900/20;
}

.step-icon--completed {
  @apply bg-emerald-100 dark:bg-emerald-900/30 text-emerald-600 dark:text-emerald-400;
}

.step-icon--failed {
  @apply bg-red-100 dark:bg-red-900/30 text-red-600 dark:text-red-400;
}

.step-connector {
  @apply w-0.5 flex-1 mt-2 transition-colors;
}

.step-connector--default {
  @apply bg-slate-200 dark:bg-slate-700;
}

.step-connector--completed {
  @apply bg-emerald-400 dark:bg-emerald-600;
}

.step-content {
  @apply flex-1 pt-1;
}

.step-header {
  @apply flex items-start justify-between gap-3 mb-2;
}

.step-title {
  @apply text-base font-semibold text-slate-800 dark:text-slate-200;
}

.step-badge {
  @apply text-xs px-2 py-1 rounded-full font-medium flex-shrink-0;
}

.step-badge--pending {
  @apply bg-slate-100 dark:bg-slate-800 text-slate-600 dark:text-slate-400;
}

.step-badge--active {
  @apply bg-blue-100 dark:bg-blue-900/30 text-blue-700 dark:text-blue-300 animate-pulse;
}

.step-badge--completed {
  @apply bg-emerald-100 dark:bg-emerald-900/30 text-emerald-700 dark:text-emerald-300;
}

.step-badge--failed {
  @apply bg-red-100 dark:bg-red-900/30 text-red-700 dark:text-red-300;
}

.step-description {
  @apply text-sm text-slate-600 dark:text-slate-400 leading-relaxed mb-3;
}

.step-actions {
  @apply flex gap-2 flex-wrap mt-3;
}

.action-button {
  @apply px-3 py-1.5 text-sm font-medium rounded-lg transition-colors;
}

.action-button--primary {
  @apply bg-blue-600 text-white hover:bg-blue-700;
}

.action-button--secondary {
  @apply bg-slate-200 dark:bg-slate-700 text-slate-700 dark:text-slate-300 hover:bg-slate-300 dark:hover:bg-slate-600;
}

.action-button--danger {
  @apply bg-red-600 text-white hover:bg-red-700;
}

.step-metadata {
  @apply mt-3 p-3 bg-slate-50 dark:bg-slate-800 rounded-lg border border-slate-200 dark:border-slate-700;
}

.metadata-item {
  @apply text-xs flex gap-2 mb-1 last:mb-0;
}

.metadata-key {
  @apply font-semibold text-slate-600 dark:text-slate-400;
}

.metadata-value {
  @apply text-slate-700 dark:text-slate-300 font-mono;
}
</style>
