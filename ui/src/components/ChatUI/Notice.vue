<template>
  <div :class="['notice', typeClass]">
    <Icon :type="iconType" class="notice-icon" />
    <div class="notice-content">
      <div v-if="title" class="notice-title">{{ title }}</div>
      <div class="notice-text">{{ content }}</div>
    </div>
    <button v-if="closable" class="notice-close" @click="$emit('close')">
      <Icon type="close" size="sm" />
    </button>
  </div>
</template>

<script setup lang="ts">
import { computed } from "vue";
import Icon from "./Icon.vue";

interface Props {
  type?: "info" | "success" | "warning" | "error";
  title?: string;
  content: string;
  closable?: boolean;
}

const props = withDefaults(defineProps<Props>(), {
  type: "info",
  closable: false,
});

defineEmits<{
  close: [];
}>();

const typeClass = computed(() => {
  const map = {
    info: "notice-info",
    success: "notice-success",
    warning: "notice-warning",
    error: "notice-error",
  };
  return map[props.type];
});

const iconType = computed(() => {
  const map = {
    info: "info",
    success: "check",
    warning: "warning",
    error: "error",
  };
  return map[props.type] as any;
});
</script>

<style scoped>
.notice {
  @apply flex items-start gap-3 p-4 rounded-lg;
}

.notice-info {
  @apply bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-800;
}

.notice-success {
  @apply bg-green-50 dark:bg-green-900/20 border border-green-200 dark:border-green-800;
}

.notice-warning {
  @apply bg-amber-50 dark:bg-amber-900/20 border border-amber-200 dark:border-amber-800;
}

.notice-error {
  @apply bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800;
}

.notice-icon {
  @apply flex-shrink-0 mt-0.5;
}

.notice-info .notice-icon {
  @apply text-blue-600 dark:text-blue-400;
}

.notice-success .notice-icon {
  @apply text-green-600 dark:text-green-400;
}

.notice-warning .notice-icon {
  @apply text-amber-600 dark:text-amber-400;
}

.notice-error .notice-icon {
  @apply text-red-600 dark:text-red-400;
}

.notice-content {
  @apply flex-1;
}

.notice-title {
  @apply font-semibold text-sm mb-1;
}

.notice-info .notice-title {
  @apply text-blue-900 dark:text-blue-100;
}

.notice-success .notice-title {
  @apply text-green-900 dark:text-green-100;
}

.notice-warning .notice-title {
  @apply text-amber-900 dark:text-amber-100;
}

.notice-error .notice-title {
  @apply text-red-900 dark:text-red-100;
}

.notice-text {
  @apply text-sm;
}

.notice-info .notice-text {
  @apply text-blue-700 dark:text-blue-300;
}

.notice-success .notice-text {
  @apply text-green-700 dark:text-green-300;
}

.notice-warning .notice-text {
  @apply text-amber-700 dark:text-amber-300;
}

.notice-error .notice-text {
  @apply text-red-700 dark:text-red-300;
}

.notice-close {
  @apply flex-shrink-0 text-gray-400 hover:text-gray-600 dark:hover:text-gray-300 transition-colors;
}
</style>
