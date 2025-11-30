<template>
  <div class="thinking-message">
    <!-- Compact View -->
    <button
      v-if="!isExpanded"
      @click="isExpanded = true"
      class="thinking-compact"
    >
      <div class="flex items-center gap-2">
        <svg
          :class="['w-4 h-4', isActive ? 'text-primary animate-pulse' : 'text-secondary']"
          fill="none"
          stroke="currentColor"
          viewBox="0 0 24 24"
        >
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9.663 17h4.673M12 3v1m6.364 1.636l-.707.707M21 12h-1M4 12H3m3.343-5.657l-.707-.707m2.828 9.9a5 5 0 117.072 0l-.548.547A3.374 3.374 0 0014 18.469V19a2 2 0 11-4 0v-.531c0-.895-.356-1.754-.988-2.386l-.548-.547z"></path>
        </svg>
        <span class="text-sm font-medium">思考过程</span>
        <span class="text-xs text-secondary">({{ steps.length }} 步)</span>
      </div>
      <svg class="w-4 h-4 text-secondary" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7"></path>
      </svg>
    </button>

    <!-- Expanded View -->
    <div v-else class="thinking-expanded">
      <!-- Header -->
      <div class="thinking-header">
        <div class="flex items-center gap-2">
          <svg
            :class="['w-5 h-5', isActive ? 'text-primary animate-pulse' : 'text-secondary']"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
          >
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9.663 17h4.673M12 3v1m6.364 1.636l-.707.707M21 12h-1M4 12H3m3.343-5.657l-.707-.707m2.828 9.9a5 5 0 117.072 0l-.548.547A3.374 3.374 0 0014 18.469V19a2 2 0 11-4 0v-.531c0-.895-.356-1.754-.988-2.386l-.548-.547z"></path>
          </svg>
          <h3 class="text-base font-semibold">思考过程</h3>
          <span v-if="isActive" class="status-badge active">运行中</span>
          <span v-else class="status-badge completed">已完成</span>
        </div>
        <button @click="isExpanded = false" class="collapse-btn">
          <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 15l7-7 7 7"></path>
          </svg>
        </button>
      </div>

      <!-- Timeline -->
      <div class="thinking-timeline">
        <div
          v-for="(step, index) in steps"
          :key="step.id || index"
          class="timeline-step"
        >
          <!-- Step Indicator -->
          <div class="step-indicator">
            <div :class="['step-dot', getStepClass(step)]">
              <svg v-if="step.type === 'reasoning'" class="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9.663 17h4.673M12 3v1m6.364 1.636l-.707.707M21 12h-1M4 12H3m3.343-5.657l-.707-.707m2.828 9.9a5 5 0 117.072 0l-.548.547A3.374 3.374 0 0014 18.469V19a2 2 0 11-4 0v-.531c0-.895-.356-1.754-.988-2.386l-.548-.547z"></path>
              </svg>
              <svg v-else-if="step.type === 'tool_call'" class="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 20l4-16m4 4l4 4-4 4M6 16l-4-4 4-4"></path>
              </svg>
              <svg v-else-if="step.type === 'tool_result'" class="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7"></path>
              </svg>
              <svg v-else class="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"></path>
              </svg>
            </div>
            <div v-if="index < steps.length - 1" class="step-line"></div>
          </div>

          <!-- Step Content -->
          <div class="step-content">
            <div class="step-header">
              <span :class="['step-type', getStepClass(step)]">
                {{ getStepLabel(step.type) }}
              </span>
              <span class="step-time">{{ formatTime(step.timestamp) }}</span>
            </div>

            <!-- Reasoning -->
            <div v-if="step.type === 'reasoning' && step.content" class="step-body">
              <p class="text-sm text-text dark:text-text-dark">{{ step.content }}</p>
            </div>

            <!-- Tool Call -->
            <div v-if="step.type === 'tool_call' && step.tool" class="step-body">
              <div class="tool-call">
                <div class="tool-header">
                  <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 20l4-16m4 4l4 4-4 4M6 16l-4-4 4-4"></path>
                  </svg>
                  <span class="font-mono font-semibold">{{ step.tool.name }}</span>
                </div>
                <pre class="tool-args">{{ JSON.stringify(step.tool.args, null, 2) }}</pre>
              </div>
            </div>

            <!-- Tool Result -->
            <div v-if="step.type === 'tool_result' && step.result" class="step-body">
              <div class="tool-result">
                <div class="result-header">
                  <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7"></path>
                  </svg>
                  <span class="font-semibold">执行结果</span>
                </div>
                <pre class="result-content">{{ JSON.stringify(step.result, null, 2) }}</pre>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- Summary -->
      <div v-if="!isActive && summary" class="thinking-summary">
        <svg class="w-4 h-4 text-primary" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z"></path>
        </svg>
        <span class="text-sm text-text dark:text-text-dark">{{ summary }}</span>
      </div>
    </div>
  </div>
</template>

<script lang="ts">
import { defineComponent, ref } from 'vue';

interface ThinkingStep {
  id?: string;
  type: 'reasoning' | 'tool_call' | 'tool_result' | 'decision';
  content?: string;
  tool?: {
    name: string;
    args: unknown;
  };
  result?: unknown;
  timestamp: number;
}

export default defineComponent({
  name: 'ThinkingMessage',

  props: {
    steps: {
      type: Array as () => ThinkingStep[],
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

  setup() {
    const isExpanded = ref(false);

    function getStepClass(step: ThinkingStep): string {
      const classes: Record<string, string> = {
        reasoning: 'step-reasoning',
        tool_call: 'step-tool-call',
        tool_result: 'step-tool-result',
        decision: 'step-decision',
      };
      return classes[step.type] || '';
    }

    function getStepLabel(type: string): string {
      const labels: Record<string, string> = {
        reasoning: '推理',
        tool_call: '工具调用',
        tool_result: '执行结果',
        decision: '决策',
      };
      return labels[type] || type;
    }

    function formatTime(timestamp: number): string {
      const date = new Date(timestamp);
      return date.toLocaleTimeString('zh-CN', {
        hour: '2-digit',
        minute: '2-digit',
        second: '2-digit',
      });
    }

    return {
      isExpanded,
      getStepClass,
      getStepLabel,
      formatTime,
    };
  },
});
</script>

<style scoped>
.thinking-message {
  @apply my-2;
}

.thinking-compact {
  @apply w-full flex items-center justify-between px-4 py-2 bg-surface dark:bg-surface-dark border border-border dark:border-border-dark rounded-lg hover:bg-background dark:hover:bg-background-dark transition-colors;
}

.thinking-expanded {
  @apply bg-surface dark:bg-surface-dark border border-border dark:border-border-dark rounded-lg overflow-hidden;
}

.thinking-header {
  @apply flex items-center justify-between px-4 py-3 border-b border-border dark:border-border-dark;
}

.status-badge {
  @apply text-xs px-2 py-0.5 rounded-full font-medium;
}

.status-badge.active {
  @apply bg-primary/10 text-primary;
}

.status-badge.completed {
  @apply bg-secondary/10 text-secondary dark:text-secondary-dark;
}

.collapse-btn {
  @apply p-1 hover:bg-background dark:hover:bg-background-dark rounded transition-colors;
}

.thinking-timeline {
  @apply px-4 py-3 space-y-4;
}

.timeline-step {
  @apply flex gap-3;
}

.step-indicator {
  @apply flex flex-col items-center;
}

.step-dot {
  @apply w-8 h-8 rounded-full flex items-center justify-center flex-shrink-0;
}

.step-dot.step-reasoning {
  @apply bg-blue-100 dark:bg-blue-900/30 text-blue-600 dark:text-blue-400;
}

.step-dot.step-tool-call {
  @apply bg-indigo-100 dark:bg-indigo-900/30 text-indigo-600 dark:text-indigo-400;
}

.step-dot.step-tool-result {
  @apply bg-emerald-100 dark:bg-emerald-900/30 text-emerald-600 dark:text-emerald-400;
}

.step-dot.step-decision {
  @apply bg-purple-100 dark:bg-purple-900/30 text-purple-600 dark:text-purple-400;
}

.step-line {
  @apply w-0.5 flex-1 bg-border dark:bg-border-dark mt-1;
}

.step-content {
  @apply flex-1 pb-2;
}

.step-header {
  @apply flex items-center justify-between mb-2;
}

.step-type {
  @apply text-xs font-semibold px-2 py-1 rounded;
}

.step-type.step-reasoning {
  @apply bg-blue-100 dark:bg-blue-900/30 text-blue-700 dark:text-blue-300;
}

.step-type.step-tool-call {
  @apply bg-indigo-100 dark:bg-indigo-900/30 text-indigo-700 dark:text-indigo-300;
}

.step-type.step-tool-result {
  @apply bg-emerald-100 dark:bg-emerald-900/30 text-emerald-700 dark:text-emerald-300;
}

.step-type.step-decision {
  @apply bg-purple-100 dark:bg-purple-900/30 text-purple-700 dark:text-purple-300;
}

.step-time {
  @apply text-xs text-secondary dark:text-secondary-dark;
}

.step-body {
  @apply text-sm;
}

.tool-call,
.tool-result {
  @apply rounded-lg overflow-hidden border;
}

.tool-call {
  @apply border-indigo-200 dark:border-indigo-800;
}

.tool-result {
  @apply border-emerald-200 dark:border-emerald-800;
}

.tool-header,
.result-header {
  @apply flex items-center gap-2 px-3 py-2 text-sm font-medium;
}

.tool-header {
  @apply bg-indigo-50 dark:bg-indigo-900/20 text-indigo-700 dark:text-indigo-300;
}

.result-header {
  @apply bg-emerald-50 dark:bg-emerald-900/20 text-emerald-700 dark:text-emerald-300;
}

.tool-args,
.result-content {
  @apply p-3 text-xs font-mono bg-background dark:bg-background-dark text-text dark:text-text-dark overflow-x-auto;
}

.thinking-summary {
  @apply flex items-center gap-2 px-4 py-3 bg-primary/5 dark:bg-primary/10 border-t border-border dark:border-border-dark;
}
</style>
