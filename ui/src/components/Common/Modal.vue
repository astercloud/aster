<template>
  <Teleport to="body">
    <div class="modal-overlay" @click="handleOverlayClick">
      <div class="modal-container" @click.stop>
        <div class="modal-header">
          <slot name="header">Modal</slot>
          <button @click="$emit('close')" class="modal-close">
            <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"></path>
            </svg>
          </button>
        </div>
        <div class="modal-body">
          <slot></slot>
        </div>
      </div>
    </div>
  </Teleport>
</template>

<script lang="ts">
import { defineComponent } from 'vue';

export default defineComponent({
  name: 'Modal',

  emits: {
    close: () => true,
  },

  setup(props, { emit }) {
    const handleOverlayClick = () => {
      emit('close');
    };

    return {
      handleOverlayClick,
    };
  },
});
</script>

<style scoped>
.modal-overlay {
  @apply fixed inset-0 bg-black/50 backdrop-blur-sm flex items-center justify-center z-50 p-4;
  animation: fadeIn 0.2s ease-out;
}

.modal-container {
  @apply bg-surface dark:bg-surface-dark rounded-xl shadow-2xl max-w-2xl w-full max-h-[90vh] overflow-hidden flex flex-col;
  animation: slideUp 0.3s ease-out;
}

.modal-header {
  @apply flex items-center justify-between p-6 border-b border-border dark:border-border-dark;
}

.modal-header slot {
  @apply text-xl font-bold text-text dark:text-text-dark;
}

.modal-close {
  @apply p-1 hover:bg-border dark:hover:bg-border-dark rounded-lg transition-colors text-secondary dark:text-secondary-dark hover:text-text dark:hover:text-text-dark;
}

.modal-body {
  @apply p-6 overflow-y-auto;
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
    transform: translateY(20px);
    opacity: 0;
  }
  to {
    transform: translateY(0);
    opacity: 1;
  }
}
</style>
