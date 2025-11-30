<template>
  <div :class="['system-message', `type-${message.content.type || 'info'}`]">
    <div class="system-icon">
      <svg v-if="message.content.type === 'success'" class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z"></path>
      </svg>
      <svg v-else-if="message.content.type === 'error'" class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 14l2-2m0 0l2-2m-2 2l-2-2m2 2l2 2m7-2a9 9 0 11-18 0 9 9 0 0118 0z"></path>
      </svg>
      <svg v-else-if="message.content.type === 'warning'" class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"></path>
      </svg>
      <svg v-else class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"></path>
      </svg>
    </div>

    <div class="system-content">
      <p class="system-text">{{ message.content.text }}</p>
      <span v-if="message.createdAt" class="system-time">
        {{ formatTime(message.createdAt) }}
      </span>
    </div>
  </div>
</template>

<script lang="ts">
import { defineComponent } from 'vue';
import type { SystemMessage as SystemMessageType } from '@/types';

export default defineComponent({
  name: 'SystemMessage',

  props: {
    message: {
      type: Object as () => SystemMessageType,
      required: true,
    },
  },

  setup() {
    function formatTime(timestamp: number): string {
      const date = new Date(timestamp);
      return date.toLocaleTimeString('zh-CN', {
        hour: '2-digit',
        minute: '2-digit',
      });
    }

    return {
      formatTime,
    };
  },
});
</script>

<style scoped>
.system-message {
  @apply flex items-center gap-2 px-3 py-2 rounded-lg text-sm my-4 mx-auto max-w-md;
}

.type-info {
  @apply bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-800;
}

.type-success {
  @apply bg-emerald-50 dark:bg-emerald-900/20 border border-emerald-200 dark:border-emerald-800;
}

.type-warning {
  @apply bg-amber-50 dark:bg-amber-900/20 border border-amber-200 dark:border-amber-800;
}

.type-error {
  @apply bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800;
}

.system-icon {
  @apply flex-shrink-0;
}

.type-info .system-icon {
  @apply text-blue-600 dark:text-blue-400;
}

.type-success .system-icon {
  @apply text-emerald-600 dark:text-emerald-400;
}

.type-warning .system-icon {
  @apply text-amber-600 dark:text-amber-400;
}

.type-error .system-icon {
  @apply text-red-600 dark:text-red-400;
}

.system-content {
  @apply flex-1 flex items-center justify-between gap-2;
}

.system-text {
  @apply text-text dark:text-text-dark;
}

.type-info .system-text {
  @apply text-blue-900 dark:text-blue-100;
}

.type-success .system-text {
  @apply text-emerald-900 dark:text-emerald-100;
}

.type-warning .system-text {
  @apply text-amber-900 dark:text-amber-100;
}

.type-error .system-text {
  @apply text-red-900 dark:text-red-100;
}

.system-time {
  @apply text-xs text-secondary dark:text-secondary-dark;
}
</style>
