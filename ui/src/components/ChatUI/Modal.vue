<template>
  <Teleport to="body">
    <div v-if="visible" class="modal-overlay" @click="handleOverlayClick">
      <div :class="['modal-container', sizeClass]" @click.stop>
        <div class="modal-header">
          <h3 class="modal-title">
            <slot name="title">{{ title }}</slot>
          </h3>
          <button class="modal-close" @click="close">
            <Icon type="close" />
          </button>
        </div>

        <div class="modal-body">
          <slot></slot>
        </div>

        <div v-if="$slots.footer" class="modal-footer">
          <slot name="footer"></slot>
        </div>
      </div>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import { computed } from "vue";
import Icon from "./Icon.vue";

interface Props {
  visible: boolean;
  title?: string;
  size?: "sm" | "md" | "lg" | "xl";
  closeOnOverlay?: boolean;
}

const props = withDefaults(defineProps<Props>(), {
  title: "",
  size: "md",
  closeOnOverlay: true,
});

const emit = defineEmits<{
  "update:visible": [value: boolean];
  close: [];
}>();

const sizeClass = computed(() => {
  const map = {
    sm: "max-w-sm",
    md: "max-w-md",
    lg: "max-w-lg",
    xl: "max-w-xl",
  };
  return map[props.size];
});

const close = () => {
  emit("update:visible", false);
  emit("close");
};

const handleOverlayClick = () => {
  if (props.closeOnOverlay) {
    close();
  }
};
</script>

<style scoped>
.modal-overlay {
  @apply fixed inset-0 z-50 flex items-center justify-center bg-black/50 backdrop-blur-sm p-4;
  animation: fadeIn 0.2s;
}

.modal-container {
  @apply w-full bg-white dark:bg-gray-800 rounded-xl shadow-2xl;
  animation: slideUp 0.3s;
}

.modal-header {
  @apply flex items-center justify-between px-6 py-4 border-b border-gray-200 dark:border-gray-700;
}

.modal-title {
  @apply text-lg font-semibold text-gray-900 dark:text-white;
}

.modal-close {
  @apply p-1 hover:bg-gray-100 dark:hover:bg-gray-700 rounded transition-colors;
}

.modal-body {
  @apply px-6 py-4 max-h-[70vh] overflow-y-auto;
}

.modal-footer {
  @apply px-6 py-4 border-t border-gray-200 dark:border-gray-700;
}

@keyframes fadeIn {
  from {
    opacity: 0;
  }
  to {
    opacity: 1;
  }
}

@keyframes slideUp {
  from {
    opacity: 0;
    transform: translateY(20px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}
</style>
