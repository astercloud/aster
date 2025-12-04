<template>
  <div :class="['flex-container', directionClass, wrapClass, justifyClass, alignClass, gapClass]">
    <slot></slot>
  </div>
</template>

<script setup lang="ts">
import { computed } from "vue";

interface Props {
  direction?: "row" | "column";
  wrap?: boolean;
  justify?: "start" | "end" | "center" | "between" | "around";
  align?: "start" | "end" | "center" | "stretch";
  gap?: "none" | "sm" | "md" | "lg";
}

const props = withDefaults(defineProps<Props>(), {
  direction: "row",
  wrap: false,
  justify: "start",
  align: "start",
  gap: "md",
});

const directionClass = computed(() => {
  return props.direction === "column" ? "flex-col" : "flex-row";
});

const wrapClass = computed(() => {
  return props.wrap ? "flex-wrap" : "";
});

const justifyClass = computed(() => {
  const map = {
    start: "justify-start",
    end: "justify-end",
    center: "justify-center",
    between: "justify-between",
    around: "justify-around",
  };
  return map[props.justify];
});

const alignClass = computed(() => {
  const map = {
    start: "items-start",
    end: "items-end",
    center: "items-center",
    stretch: "items-stretch",
  };
  return map[props.align];
});

const gapClass = computed(() => {
  const map = {
    none: "gap-0",
    sm: "gap-2",
    md: "gap-4",
    lg: "gap-6",
  };
  return map[props.gap];
});
</script>

<style scoped>
.flex-container {
  @apply flex;
}
</style>
