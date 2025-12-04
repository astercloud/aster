<template>
  <div :class="['divider', directionClass]">
    <span v-if="$slots.default" class="divider-text">
      <slot></slot>
    </span>
  </div>
</template>

<script setup lang="ts">
import { computed } from "vue";

interface Props {
  direction?: "horizontal" | "vertical";
}

const props = withDefaults(defineProps<Props>(), {
  direction: "horizontal",
});

const directionClass = computed(() => {
  return props.direction === "vertical" ? "divider-vertical" : "divider-horizontal";
});
</script>

<style scoped>
.divider-horizontal {
  @apply flex items-center my-4;
}

.divider-horizontal::before,
.divider-horizontal::after {
  content: "";
  @apply flex-1 border-t border-gray-200 dark:border-gray-700;
}

.divider-horizontal .divider-text {
  @apply px-4 text-sm text-gray-500 dark:text-gray-400;
}

.divider-vertical {
  @apply inline-block w-px h-full bg-gray-200 dark:bg-gray-700 mx-2;
}
</style>
