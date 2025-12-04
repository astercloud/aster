<template>
  <label class="checkbox-container">
    <input type="checkbox" :checked="modelValue" :disabled="disabled" class="checkbox-input" @change="handleChange" />
    <span class="checkbox-label">
      <slot></slot>
    </span>
  </label>
</template>

<script setup lang="ts">
interface Props {
  modelValue: boolean;
  disabled?: boolean;
}

defineProps<Props>();

const emit = defineEmits<{
  "update:modelValue": [value: boolean];
}>();

const handleChange = (e: Event) => {
  const target = e.target as HTMLInputElement;
  emit("update:modelValue", target.checked);
};
</script>

<style scoped>
.checkbox-container {
  @apply inline-flex items-center gap-2 cursor-pointer;
}

.checkbox-input {
  @apply w-4 h-4 text-blue-600 bg-gray-100 border-gray-300 rounded focus:ring-blue-500 dark:focus:ring-blue-600 dark:ring-offset-gray-800 focus:ring-2 dark:bg-gray-700 dark:border-gray-600 cursor-pointer;
}

.checkbox-input:disabled {
  @apply opacity-50 cursor-not-allowed;
}

.checkbox-label {
  @apply text-sm text-gray-700 dark:text-gray-300;
}
</style>
