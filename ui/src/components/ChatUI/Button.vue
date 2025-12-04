<template>
  <button :class="['chatui-button', variantClass, sizeClass]" :disabled="disabled" @click="$emit('click')">
    <svg v-if="icon" class="button-icon" fill="none" stroke="currentColor" viewBox="0 0 24 24">
      <path v-if="icon === 'send'" stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 19l9 2-9-18-9 18 9-2zm0 0v-8"></path>
      <path
        v-else-if="icon === 'image'"
        stroke-linecap="round"
        stroke-linejoin="round"
        stroke-width="2"
        d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z"
      ></path>
      <path v-else-if="icon === 'mic'" stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 11a7 7 0 01-7 7m0 0a7 7 0 01-7-7m7 7v4m0 0H8m4 0h4m-4-8a3 3 0 01-3-3V5a3 3 0 116 0v6a3 3 0 01-3 3z"></path>
      <path v-else-if="icon === 'attach'" stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15.172 7l-6.586 6.586a2 2 0 102.828 2.828l6.414-6.586a4 4 0 00-5.656-5.656l-6.415 6.585a6 6 0 108.486 8.486L20.5 13"></path>
    </svg>
    <slot></slot>
  </button>
</template>

<script setup lang="ts">
import { computed } from "vue";

interface Props {
  icon?: string;
  variant?: "primary" | "secondary" | "text";
  size?: "sm" | "md" | "lg";
  disabled?: boolean;
}

const props = withDefaults(defineProps<Props>(), {
  variant: "primary",
  size: "md",
  disabled: false,
});

defineEmits<{
  click: [];
}>();

const variantClass = computed(() => {
  const variants = {
    primary: "button-primary",
    secondary: "button-secondary",
    text: "button-text",
  };
  return variants[props.variant];
});

const sizeClass = computed(() => {
  const sizes = {
    sm: "button-sm",
    md: "button-md",
    lg: "button-lg",
  };
  return sizes[props.size];
});
</script>

<style scoped>
.chatui-button {
  @apply inline-flex items-center justify-center gap-2 font-medium rounded-lg transition-colors disabled:opacity-50 disabled:cursor-not-allowed;
}

.button-icon {
  @apply w-5 h-5;
}

.button-primary {
  @apply bg-blue-500 hover:bg-blue-600 text-white;
}

.button-secondary {
  @apply bg-gray-200 hover:bg-gray-300 dark:bg-gray-700 dark:hover:bg-gray-600 text-gray-900 dark:text-gray-100;
}

.button-text {
  @apply text-gray-600 hover:text-gray-900 dark:text-gray-400 dark:hover:text-gray-100 hover:bg-gray-100 dark:hover:bg-gray-800;
}

.button-sm {
  @apply px-2 py-1 text-sm;
}

.button-md {
  @apply px-4 py-2 text-base;
}

.button-lg {
  @apply px-6 py-3 text-lg;
}
</style>
