<template>
  <div :class="['agent-card', statusClass]">
    <!-- Header -->
    <div class="agent-header">
      <div class="agent-avatar">
        <img v-if="agent.avatar" :src="agent.avatar" :alt="agent.name" />
        <div v-else class="avatar-placeholder">
          <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9.663 17h4.673M12 3v1m6.364 1.636l-.707.707M21 12h-1M4 12H3m3.343-5.657l-.707-.707m2.828 9.9a5 5 0 117.072 0l-.548.547A3.374 3.374 0 0014 18.469V19a2 2 0 11-4 0v-.531c0-.895-.356-1.754-.988-2.386l-.548-.547z"></path>
          </svg>
        </div>
      </div>

      <div class="agent-info">
        <h3 class="agent-name">{{ agent.name }}</h3>
        <p v-if="agent.description" class="agent-description">{{ agent.description }}</p>
      </div>

      <div class="agent-status">
        <span :class="['status-badge', statusClass]">
          <span class="status-dot"></span>
          {{ statusText }}
        </span>
      </div>
    </div>

    <!-- Metadata -->
    <div v-if="agent.metadata" class="agent-metadata">
      <div
        v-for="(value, key) in displayMetadata"
        :key="key"
        class="metadata-item"
      >
        <span class="metadata-label">{{ key }}</span>
        <span class="metadata-value">{{ value }}</span>
      </div>
    </div>

    <!-- Actions -->
    <div class="agent-actions">
      <button
        @click="$emit('chat', agent)"
        class="action-btn btn-primary"
        title="开始对话"
      >
        <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 12h.01M12 12h.01M16 12h.01M21 12c0 4.418-4.03 8-9 8a9.863 9.863 0 01-4.255-.949L3 20l1.395-3.72C3.512 15.042 3 13.574 3 12c0-4.418 4.03-8 9-8s9 3.582 9 8z"></path>
        </svg>
        对话
      </button>

      <button
        @click="$emit('edit', agent)"
        class="action-btn btn-secondary"
        title="编辑"
      >
        <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z"></path>
        </svg>
      </button>

      <button
        @click="$emit('delete', agent)"
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
import { computed, defineComponent } from 'vue';
import type { Agent } from '@/types';
import type { PropType } from 'vue';

export default defineComponent({
  name: 'AgentCard',
  props: {
    agent: {
      type: Object as PropType<Agent>,
      required: true
    }
  },
  emits: {
    chat: (agent: Agent) => true,
    edit: (agent: Agent) => true,
    delete: (agent: Agent) => true
  },
  setup(props) {
    const statusClass = computed(() => {
      const classes: Record<string, string> = {
        idle: 'status-idle',
        thinking: 'status-thinking',
        busy: 'status-busy',
        error: 'status-error',
      };
      return classes[props.agent.status] || 'status-idle';
    });

    const statusText = computed(() => {
      const texts: Record<string, string> = {
        idle: '空闲',
        thinking: '思考中',
        busy: '忙碌',
        error: '错误',
      };
      return texts[props.agent.status] || '未知';
    });

    const displayMetadata = computed(() => {
      if (!props.agent.metadata) return {};

      // 只显示部分元数据
      const { model, provider, version } = props.agent.metadata;
      return {
        ...(model && { 模型: model }),
        ...(provider && { 提供商: provider }),
        ...(version && { 版本: version }),
      };
    });

    return {
      statusClass,
      statusText,
      displayMetadata
    };
  }
});
</script>

<style scoped>
.agent-card {
  @apply bg-surface dark:bg-surface-dark border border-border dark:border-border-dark rounded-lg p-4 hover:shadow-md transition-shadow;
}

.agent-header {
  @apply flex items-start gap-3 mb-3;
}

.agent-avatar {
  @apply w-12 h-12 rounded-full overflow-hidden flex-shrink-0;
}

.agent-avatar img {
  @apply w-full h-full object-cover;
}

.avatar-placeholder {
  @apply w-full h-full bg-primary/10 dark:bg-primary/20 flex items-center justify-center text-primary dark:text-primary-light;
}

.agent-info {
  @apply flex-1 min-w-0;
}

.agent-name {
  @apply text-base font-semibold text-text dark:text-text-dark truncate;
}

.agent-description {
  @apply text-sm text-secondary dark:text-secondary-dark mt-1 line-clamp-2;
}

.agent-status {
  @apply flex-shrink-0;
}

.status-badge {
  @apply flex items-center gap-1.5 px-2 py-1 rounded-full text-xs font-medium;
}

.status-badge.status-idle {
  @apply bg-gray-100 dark:bg-gray-800 text-gray-700 dark:text-gray-300;
}

.status-badge.status-thinking {
  @apply bg-blue-100 dark:bg-blue-900/30 text-blue-700 dark:text-blue-300;
}

.status-badge.status-busy {
  @apply bg-amber-100 dark:bg-amber-900/30 text-amber-700 dark:text-amber-300;
}

.status-badge.status-error {
  @apply bg-red-100 dark:bg-red-900/30 text-red-700 dark:text-red-300;
}

.status-dot {
  @apply w-2 h-2 rounded-full;
}

.status-idle .status-dot {
  @apply bg-gray-500 dark:bg-gray-400;
}

.status-thinking .status-dot {
  @apply bg-blue-500 dark:bg-blue-400 animate-pulse;
}

.status-busy .status-dot {
  @apply bg-amber-500 dark:bg-amber-400 animate-pulse;
}

.status-error .status-dot {
  @apply bg-red-500 dark:bg-red-400;
}

.agent-metadata {
  @apply grid grid-cols-2 gap-2 mb-3 pb-3 border-b border-border dark:border-border-dark;
}

.metadata-item {
  @apply flex flex-col;
}

.metadata-label {
  @apply text-xs text-secondary dark:text-secondary-dark;
}

.metadata-value {
  @apply text-sm text-text dark:text-text-dark font-medium;
}

.agent-actions {
  @apply flex gap-2;
}

.action-btn {
  @apply flex items-center gap-1.5 px-3 py-2 rounded-lg text-sm font-medium transition-colors;
}

.btn-primary {
  @apply flex-1 bg-primary hover:bg-primary-hover text-white;
}

.btn-secondary {
  @apply bg-background dark:bg-background-dark hover:bg-border dark:hover:bg-border-dark text-text dark:text-text-dark border border-border dark:border-border-dark;
}

.btn-danger {
  @apply bg-background dark:bg-background-dark hover:bg-red-50 dark:hover:bg-red-900/20 text-red-600 dark:text-red-400 border border-border dark:border-border-dark hover:border-red-300 dark:hover:border-red-800;
}
</style>
