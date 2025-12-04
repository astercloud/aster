<template>
  <div class="popover-container">
    <div @click="toggle">
      <slot name="trigger"></slot>
    </div>
    <Teleport to="body">
      <div v-if="visible" class="popover-overlay" @click="close">
        <div :class="['popover-content', positionClass]" @click.stop>
          <slot></slot>
        </div>
      </div>
    </Teleport>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from "vue";

interface Props {
  position?: "top" | "bottom" | "left" | "right";
}

const props = withDefaults(defineProps<Props>(), {
  position: "bottom",
});

const visible = ref(false);

const positionClass = computed(() => {
  const map = {
    top: "popover-top",
    bottom: "popover-bottom",
    left: "popover-left",
    right: "popover-right",
  };
  return map[props.position];
});

const toggle = () => {
  visible.value = !visible.value;
};

const close = () => {
  visible.value = false;
};
</script>

<style scoped>
.popover-container {
  @apply relative inline-block;
}

.popover-overlay {
  @apply fixed inset-0 z-40;
}

.popover-content {
  @apply absolute z-50 bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-700 rounded-lg shadow-lg p-4;
  animation: slideIn 0.2s;
}

.popover-bottom {
  @apply top-full left-0 mt-2;
}

.popover-top {
  @apply bottom-full left-0 mb-2;
}

.popover-left {
  @apply right-full top-0 mr-2;
}

.popover-right {
  @apply left-full top-0 ml-2;
}

@keyframes slideIn {
  from {
    opacity: 0;
    transform: translateY(-10px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}
</style>
