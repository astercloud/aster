<template>
  <label class="radio-container">
    <input type="radio" :name="name" :value="value" :checked="modelValue === value" :disabled="disabled" class="radio-input" @change="handleChange" />
    <span class="radio-label">
      <slot></slot>
    </span>
  </label>
</template>

<script setup lang="ts">
interface Props {
  modelValue: any;
  value: any;
  name: string;
  disabled?: boolean;
}

const props = defineProps<Props>();

const emit = defineEmits<{
  "update:modelValue": [value: any];
}>();

const handleChange = () => {
  emit("update:modelValue", props.value);
};
</script>

<style scoped>
.radio-container {
  @apply inline-flex items-center gap-2 cursor-pointer;
}

.radio-input {
  @apply w-4 h-4 text-blue-600 bg-gray-100 border-gray-300 focus:ring-blue-500 dark:focus:ring-blue-600 dark:ring-offset-gray-800 focus:ring-2 dark:bg-gray-700 dark:border-gray-600 cursor-pointer;
}

.radio-input:disabled {
  @apply opacity-50 cursor-not-allowed;
}

.radio-label {
  @apply text-sm text-gray-700 dark:text-gray-300;
}
</style>
