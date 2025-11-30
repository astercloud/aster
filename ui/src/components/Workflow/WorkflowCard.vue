<template>
  <div :class="['workflow-card', statusClass]">
    <!-- Header -->
    <div class="workflow-header">
      <div class="workflow-icon">
        <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2m-3 7h3m-3 4h3m-6-4h.01M9 16h.01"></path>
        </svg>
      </div>

      <div class="workflow-info">
        <h3 class="workflow-name">{{ workflow.name }}</h3>
        <p v-if="workflow.description" class="workflow-description">
          {{ workflow.description }}
        </p>
      </div>

      <div class="workflow-status">
        <span :class="['status-badge', statusClass]">
          <span class="status-dot"></span>
          {{ statusText }}
        </span>
      </div>
    </div>

    <!-- Steps -->
    <div class="workflow-steps">
      <div class="steps-header">
        <span class="steps-label">步骤</span>
        <span class="steps-count">{{ workflow.steps.length }} 个</span>
      </div>

      <div class="steps-progress">
        <div class="progress-bar">
          <div
            class="progress-fill"
            :style="{ width: `${progress}%` }"
          ></div>
        </div>
        <span class="progress-text">{{ progress }}%</span>
      </div>

      <div class="steps-list">
        <div
          v-for="(step, index) in workflow.steps.slice(0, 3)"
          :key="step.id"
          :class="['step-item', `step-${step.status}`]"
        >
          <span class="step-number">{{ index + 1 }}</span>
          <span class="step-name">{{ step.name }}</span>
          <span :class="['step-status', `status-${step.status}`]">
            {{ getStepStatusText(step.status) }}
          </span>
        </div>
        <div v-if="workflow.steps.length > 3" class="step-more">
          +{{ workflow.steps.length - 3 }} 更多
        </div>
      </div>
    </div>

    <!-- Actions -->
    <div class="workflow-actions">
      <button
        @click="$emit('execute', workflow)"
        :disabled="workflow.status === 'running'"
        class="action-btn btn-primary"
        title="执行"
      >
        <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M14.752 11.168l-3.197-2.132A1 1 0 0010 9.87v4.263a1 1 0 001.555.832l3.197-2.132a1 1 0 000-1.664z"></path>
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 12a9 9 0 11-18 0 9 9 0 0118 0z"></path>
        </svg>
        执行
      </button>

      <button
        @click="$emit('edit', workflow)"
        class="action-btn btn-secondary"
        title="编辑"
      >
        <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z"></path>
        </svg>
      </button>

      <button
        @click="$emit('delete', workflow)"
        class="action-btn btn-danger"
        title="删除"
      >
        <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"></path>
        </svg>
      </button>
    </div>
  </div>
</template>

<script lang="ts">
import { defineComponent, computed, type PropType } from 'vue';
import type { Workflow } from '@/types';

export default defineComponent({
  name: 'WorkflowCard',
  props: {
    workflow: {
      type: Object as PropType<Workflow>,
      required: true,
    },
  },
  emits: {
    execute: (workflow: Workflow) => true,
    edit: (workflow: Workflow) => true,
    delete: (workflow: Workflow) => true,
  },
  setup(props) {
    const statusClass = computed(() => {
      const classes: Record<string, string> = {
        idle: 'status-idle',
        running: 'status-running',
        paused: 'status-paused',
        completed: 'status-completed',
        error: 'status-error',
      };
      return classes[props.workflow.status] || 'status-idle';
    });

    const statusText = computed(() => {
      const texts: Record<string, string> = {
        idle: '空闲',
        running: '运行中',
        paused: '已暂停',
        completed: '已完成',
        error: '错误',
      };
      return texts[props.workflow.status] || '未知';
    });

    const progress = computed(() => {
      const total = props.workflow.steps.length;
      if (total === 0) return 0;

      const completed = props.workflow.steps.filter(
        s => s.status === 'completed'
      ).length;

      return Math.round((completed / total) * 100);
    });

    function getStepStatusText(status: string): string {
      const texts: Record<string, string> = {
        pending: '待执行',
        running: '运行中',
        completed: '已完成',
        error: '错误',
        skipped: '已跳过',
      };
      return texts[status] || status;
    }

    return {
      statusClass,
      statusText,
      progress,
      getStepStatusText,
    };
  },
});
</script>

<style scoped>
.workflow-card {
  @apply bg-surface dark:bg-surface-dark border border-border dark:border-border-dark rounded-lg p-4 hover:shadow-md transition-shadow;
}

.workflow-header {
  @apply flex items-start gap-3 mb-4;
}

.workflow-icon {
  @apply w-12 h-12 rounded-lg bg-primary/10 dark:bg-primary/20 flex items-center justify-center text-primary dark:text-primary-light flex-shrink-0;
}

.workflow-info {
  @apply flex-1 min-w-0;
}

.workflow-name {
  @apply text-base font-semibold text-text dark:text-text-dark truncate;
}

.workflow-description {
  @apply text-sm text-secondary dark:text-secondary-dark mt-1 line-clamp-2;
}

.workflow-status {
  @apply flex-shrink-0;
}

.status-badge {
  @apply flex items-center gap-1.5 px-2 py-1 rounded-full text-xs font-medium;
}

.status-badge.status-idle {
  @apply bg-gray-100 dark:bg-gray-800 text-gray-700 dark:text-gray-300;
}

.status-badge.status-running {
  @apply bg-blue-100 dark:bg-blue-900/30 text-blue-700 dark:text-blue-300;
}

.status-badge.status-paused {
  @apply bg-amber-100 dark:bg-amber-900/30 text-amber-700 dark:text-amber-300;
}

.status-badge.status-completed {
  @apply bg-emerald-100 dark:bg-emerald-900/30 text-emerald-700 dark:text-emerald-300;
}

.status-badge.status-error {
  @apply bg-red-100 dark:bg-red-900/30 text-red-700 dark:text-red-300;
}

.status-dot {
  @apply w-2 h-2 rounded-full;
}

.status-running .status-dot {
  @apply bg-blue-500 dark:bg-blue-400 animate-pulse;
}

.workflow-steps {
  @apply mb-4 pb-4 border-b border-border dark:border-border-dark;
}

.steps-header {
  @apply flex items-center justify-between mb-2;
}

.steps-label {
  @apply text-sm font-medium text-text dark:text-text-dark;
}

.steps-count {
  @apply text-xs text-secondary dark:text-secondary-dark;
}

.steps-progress {
  @apply flex items-center gap-2 mb-3;
}

.progress-bar {
  @apply flex-1 h-2 bg-background dark:bg-background-dark rounded-full overflow-hidden;
}

.progress-fill {
  @apply h-full bg-primary dark:bg-primary-light transition-all duration-300;
}

.progress-text {
  @apply text-xs font-medium text-text dark:text-text-dark min-w-[3rem] text-right;
}

.steps-list {
  @apply space-y-2;
}

.step-item {
  @apply flex items-center gap-2 text-sm;
}

.step-number {
  @apply w-5 h-5 rounded-full bg-background dark:bg-background-dark flex items-center justify-center text-xs font-medium text-secondary dark:text-secondary-dark flex-shrink-0;
}

.step-name {
  @apply flex-1 truncate text-text dark:text-text-dark;
}

.step-status {
  @apply text-xs px-2 py-0.5 rounded-full flex-shrink-0;
}

.status-pending {
  @apply bg-gray-100 dark:bg-gray-800 text-gray-600 dark:text-gray-400;
}

.status-running {
  @apply bg-blue-100 dark:bg-blue-900/30 text-blue-600 dark:text-blue-400;
}

.status-completed {
  @apply bg-emerald-100 dark:bg-emerald-900/30 text-emerald-600 dark:text-emerald-400;
}

.status-error {
  @apply bg-red-100 dark:bg-red-900/30 text-red-600 dark:text-red-400;
}

.step-more {
  @apply text-xs text-secondary dark:text-secondary-dark text-center py-1;
}

.workflow-actions {
  @apply flex gap-2;
}

.action-btn {
  @apply flex items-center gap-1.5 px-3 py-2 rounded-lg text-sm font-medium transition-colors;
}

.btn-primary {
  @apply flex-1 bg-primary hover:bg-primary-hover text-white disabled:opacity-50 disabled:cursor-not-allowed;
}

.btn-secondary {
  @apply bg-background dark:bg-background-dark hover:bg-border dark:hover:bg-border-dark text-text dark:text-text-dark border border-border dark:border-border-dark;
}

.btn-danger {
  @apply bg-background dark:bg-background-dark hover:bg-red-50 dark:hover:bg-red-900/20 text-red-600 dark:text-red-400 border border-border dark:border-border-dark hover:border-red-300 dark:hover:border-red-800;
}
</style>
