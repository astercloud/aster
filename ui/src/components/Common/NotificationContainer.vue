<template>
  <Teleport to="body">
    <div class="notification-container">
      <TransitionGroup name="notification">
        <div
          v-for="notification in notifications"
          :key="notification.id"
          :class="['notification', `notification-${notification.type}`]"
        >
          <div class="notification-icon">
            <svg v-if="notification.type === 'success'" class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z"></path>
            </svg>
            <svg v-else-if="notification.type === 'error'" class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 14l2-2m0 0l2-2m-2 2l-2-2m2 2l2 2m7-2a9 9 0 11-18 0 9 9 0 0118 0z"></path>
            </svg>
            <svg v-else-if="notification.type === 'warning'" class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"></path>
            </svg>
            <svg v-else class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"></path>
            </svg>
          </div>

          <div class="notification-content">
            <h4 v-if="notification.title" class="notification-title">
              {{ notification.title }}
            </h4>
            <p class="notification-message">{{ notification.message }}</p>
          </div>

          <button
            v-if="notification.closable"
            @click="handleClose(notification.id)"
            class="notification-close"
          >
            <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"></path>
            </svg>
          </button>
        </div>
      </TransitionGroup>
    </div>
  </Teleport>
</template>

<script lang="ts">
import { defineComponent } from 'vue';
import { useNotification } from '@/composables/useNotification';

export default defineComponent({
  name: 'NotificationContainer',

  setup() {
    const { notifications, remove } = useNotification();

    function handleClose(id: string) {
      remove(id);
    }

    return {
      notifications,
      handleClose,
    };
  },
});
</script>

<style scoped>
.notification-container {
  @apply fixed top-4 right-4 z-50 flex flex-col gap-2 max-w-sm;
}

.notification {
  @apply flex items-start gap-3 p-4 rounded-lg shadow-lg border;
  @apply bg-surface dark:bg-surface-dark;
}

.notification-info {
  @apply border-blue-200 dark:border-blue-800;
}

.notification-success {
  @apply border-emerald-200 dark:border-emerald-800;
}

.notification-warning {
  @apply border-amber-200 dark:border-amber-800;
}

.notification-error {
  @apply border-red-200 dark:border-red-800;
}

.notification-icon {
  @apply flex-shrink-0;
}

.notification-info .notification-icon {
  @apply text-blue-600 dark:text-blue-400;
}

.notification-success .notification-icon {
  @apply text-emerald-600 dark:text-emerald-400;
}

.notification-warning .notification-icon {
  @apply text-amber-600 dark:text-amber-400;
}

.notification-error .notification-icon {
  @apply text-red-600 dark:text-red-400;
}

.notification-content {
  @apply flex-1 min-w-0;
}

.notification-title {
  @apply text-sm font-semibold text-text dark:text-text-dark mb-1;
}

.notification-message {
  @apply text-sm text-secondary dark:text-secondary-dark;
}

.notification-close {
  @apply flex-shrink-0 p-1 hover:bg-background dark:hover:bg-background-dark rounded transition-colors text-secondary dark:text-secondary-dark hover:text-text dark:hover:text-text-dark;
}

/* Transitions */
.notification-enter-active,
.notification-leave-active {
  @apply transition-all duration-300;
}

.notification-enter-from {
  @apply opacity-0 translate-x-full;
}

.notification-leave-to {
  @apply opacity-0 translate-x-full scale-95;
}

.notification-move {
  @apply transition-transform duration-300;
}
</style>
