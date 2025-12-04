<template>
  <div class="progress-container">
    <div v-if="label" class="progress-label">
      <span>{{ label }}</span>
      <span v-if="showPercent" class="progress-percent">{{ percent }}%</span>
    </div>
    <div class="progress-bar">
      <div :class="['progress-fill', statusClass]" :style="{ width: `${percent}%` }"></div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from "vue";

interface Props {
  percent: number;
  label?: string;
  showPercent?: boolean;
  status?: "normal" | "success" | "error";
}

const props = withDefaults(defineProps<Props>(), {
  showPercent: true,
  status: "normal",
});

const statusClass = computed(() => {
  const map = {
    normal: "progress-normal",
    success: "progress-success",
    error: "progress-error",
  };
  return map[props.status];
});
</script>

<style scoped>
.progress-container {
  @apply space-y-2;
}

.progress-label {
  @apply flex justify-between text-sm text-gray-700 dark:text-gray-300;
}

.progress-percent {
  @apply font-semibold;
}

.progress-bar {
  @apply w-full h-2 bg-gray-200 dark:bg-gray-700 rounded-full overflow-hidden;
}

.progress-fill {
  @apply h-full transition-all duration-300 ease-out;
}

.progress-normal {
  @apply bg-blue-500;
}

.progress-success {
  @apply bg-green-500;
}

.progress-error {
  @apply bg-red-500;
}
</style>
