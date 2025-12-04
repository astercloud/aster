<template>
  <div class="tooltip-container" @mouseenter="show = true" @mouseleave="show = false">
    <slot></slot>
    <div v-if="show" :class="['tooltip', positionClass]">
      {{ content }}
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from "vue";

interface Props {
  content: string;
  position?: "top" | "bottom" | "left" | "right";
}

const props = withDefaults(defineProps<Props>(), {
  position: "top",
});

const show = ref(false);

const positionClass = computed(() => {
  const map = {
    top: "tooltip-top",
    bottom: "tooltip-bottom",
    left: "tooltip-left",
    right: "tooltip-right",
  };
  return map[props.position];
});
</script>

<style scoped>
.tooltip-container {
  @apply relative inline-block;
}

.tooltip {
  @apply absolute z-50 px-2 py-1 bg-gray-900 dark:bg-gray-700 text-white text-xs rounded whitespace-nowrap;
  animation: fadeIn 0.2s;
}

.tooltip-top {
  @apply bottom-full left-1/2 -translate-x-1/2 mb-2;
}

.tooltip-bottom {
  @apply top-full left-1/2 -translate-x-1/2 mt-2;
}

.tooltip-left {
  @apply right-full top-1/2 -translate-y-1/2 mr-2;
}

.tooltip-right {
  @apply left-full top-1/2 -translate-y-1/2 ml-2;
}

@keyframes fadeIn {
  from {
    opacity: 0;
  }
  to {
    opacity: 1;
  }
}
</style>
