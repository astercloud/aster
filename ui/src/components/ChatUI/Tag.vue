<template>
  <span :class="['tag', colorClass, sizeClass]">
    <slot></slot>
    <button v-if="closable" class="tag-close" @click="$emit('close')">
      <Icon type="close" size="sm" />
    </button>
  </span>
</template>

<script setup lang="ts">
import { computed } from "vue";
import Icon from "./Icon.vue";

interface Props {
  color?: "default" | "primary" | "success" | "warning" | "error";
  size?: "sm" | "md" | "lg";
  closable?: boolean;
}

const props = withDefaults(defineProps<Props>(), {
  color: "default",
  size: "md",
  closable: false,
});

defineEmits<{
  close: [];
}>();

const colorClass = computed(() => {
  const map = {
    default: "tag-default",
    primary: "tag-primary",
    success: "tag-success",
    warning: "tag-warning",
    error: "tag-error",
  };
  return map[props.color];
});

const sizeClass = computed(() => {
  const map = {
    sm: "tag-sm",
    md: "tag-md",
    lg: "tag-lg",
  };
  return map[props.size];
});
</script>

<style scoped>
.tag {
  @apply inline-flex items-center gap-1 rounded-full font-medium;
}

.tag-default {
  @apply bg-gray-100 dark:bg-gray-800 text-gray-700 dark:text-gray-300;
}

.tag-primary {
  @apply bg-blue-100 dark:bg-blue-900/30 text-blue-700 dark:text-blue-300;
}

.tag-success {
  @apply bg-green-100 dark:bg-green-900/30 text-green-700 dark:text-green-300;
}

.tag-warning {
  @apply bg-amber-100 dark:bg-amber-900/30 text-amber-700 dark:text-amber-300;
}

.tag-error {
  @apply bg-red-100 dark:bg-red-900/30 text-red-700 dark:text-red-300;
}

.tag-sm {
  @apply px-2 py-0.5 text-xs;
}

.tag-md {
  @apply px-3 py-1 text-sm;
}

.tag-lg {
  @apply px-4 py-1.5 text-base;
}

.tag-close {
  @apply hover:opacity-70 transition-opacity;
}
</style>
