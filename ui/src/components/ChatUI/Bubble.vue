<template>
  <div :class="['bubble-wrapper', `bubble-${position}`]">
    <!-- 头像 -->
    <div v-if="avatar && position === 'left'" class="bubble-avatar">
      <img :src="avatar" alt="avatar" />
    </div>

    <!-- 气泡内容 -->
    <div :class="['bubble', `bubble-${position}`, statusClass]">
      <div class="bubble-content" v-html="renderedContent"></div>

      <!-- 状态指示器 -->
      <div v-if="status && position === 'right'" class="bubble-status">
        <svg v-if="status === 'pending'" class="w-3 h-3 text-gray-400" fill="currentColor" viewBox="0 0 20 20">
          <path d="M10 18a8 8 0 100-16 8 8 0 000 16zm1-12a1 1 0 10-2 0v4a1 1 0 00.293.707l2.828 2.829a1 1 0 101.415-1.415L11 9.586V6z" />
        </svg>
        <svg v-else-if="status === 'sent'" class="w-3 h-3 text-blue-500" fill="currentColor" viewBox="0 0 20 20">
          <path fill-rule="evenodd" d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z" />
        </svg>
        <svg v-else-if="status === 'error'" class="w-3 h-3 text-red-500" fill="currentColor" viewBox="0 0 20 20">
          <path
            fill-rule="evenodd"
            d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z"
          />
        </svg>
      </div>
    </div>

    <!-- 头像（右侧） -->
    <div v-if="avatar && position === 'right'" class="bubble-avatar">
      <img :src="avatar" alt="avatar" />
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { marked } from "marked";

interface Props {
  content: string;
  position?: "left" | "right";
  status?: "pending" | "sent" | "error";
  avatar?: string;
}

const props = withDefaults(defineProps<Props>(), {
  position: "left",
});

const statusClass = computed(() => {
  if (!props.status) return "";
  return `bubble-status-${props.status}`;
});

const renderedContent = computed(() => {
  try {
    return marked(props.content);
  } catch {
    return props.content;
  }
});
</script>

<style scoped>
.bubble-wrapper {
  @apply flex gap-2 max-w-[80%];
}

.bubble-left {
  @apply self-start;
}

.bubble-right {
  @apply self-end flex-row-reverse;
}

.bubble-avatar {
  @apply w-8 h-8 rounded-full overflow-hidden flex-shrink-0 bg-gray-200 dark:bg-surface-dark;
}

.bubble-avatar img {
  @apply w-full h-full object-cover;
}

.bubble {
  @apply relative px-4 py-2 rounded-2xl break-words;
}

.bubble-left {
  @apply bg-gray-100 dark:bg-surface-dark text-gray-900 dark:text-text-dark;
  border-bottom-left-radius: 4px;
}

.bubble-right {
  @apply bg-blue-500 text-white;
  border-bottom-right-radius: 4px;
}

.bubble-content {
  @apply text-sm leading-relaxed;
}

.bubble-content :deep(p) {
  @apply my-1;
}

.bubble-content :deep(code) {
  @apply px-1 py-0.5 bg-black/10 rounded text-xs font-mono;
}

.bubble-content :deep(pre) {
  @apply my-2 p-3 bg-black/20 rounded-lg overflow-x-auto;
}

.bubble-content :deep(pre code) {
  @apply p-0 bg-transparent;
}

.bubble-status {
  @apply absolute -bottom-4 right-0 flex items-center gap-1 text-xs;
}
</style>
