<template>
  <div class="input-wrapper">
    <label v-if="label" class="input-label">{{ label }}</label>
    <div class="input-container">
      <input :type="type" :value="modelValue" :placeholder="placeholder" :disabled="disabled" :class="['input-field', { 'input-error': error }]" @input="handleInput" @blur="$emit('blur')" @focus="$emit('focus')" />
      <span v-if="error" class="error-message">{{ error }}</span>
    </div>
  </div>
</template>

<script setup lang="ts">
interface Props {
  modelValue: string | number;
  type?: "text" | "password" | "email" | "number";
  label?: string;
  placeholder?: string;
  disabled?: boolean;
  error?: string;
}

withDefaults(defineProps<Props>(), {
  type: "text",
});

const emit = defineEmits<{
  "update:modelValue": [value: string | number];
  blur: [];
  focus: [];
}>();

const handleInput = (e: Event) => {
  const target = e.target as HTMLInputElement;
  emit("update:modelValue", target.value);
};
</script>

<style scoped>
.input-wrapper {
  @apply space-y-1;
}

.input-label {
  @apply block text-sm font-medium text-gray-700 dark:text-gray-300;
}

.input-container {
  @apply relative;
}

.input-field {
  @apply w-full px-4 py-2 bg-white dark:bg-gray-800 border border-gray-300 dark:border-gray-600 rounded-lg text-gray-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-blue-500 dark:focus:ring-blue-600 transition-colors;
}

.input-field:disabled {
  @apply opacity-50 cursor-not-allowed bg-gray-100 dark:bg-gray-900;
}

.input-error {
  @apply border-red-500 dark:border-red-600 focus:ring-red-500 dark:focus:ring-red-600;
}

.error-message {
  @apply block text-xs text-red-500 dark:text-red-400 mt-1;
}
</style>
