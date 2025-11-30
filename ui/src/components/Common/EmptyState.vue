<template>
  <div class="empty-state">
    <div class="empty-icon">
      <slot name="icon">
        <svg class="w-16 h-16 text-secondary/30 dark:text-secondary-dark/30" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M20 13V6a2 2 0 00-2-2H6a2 2 0 00-2 2v7m16 0v5a2 2 0 01-2 2H6a2 2 0 01-2-2v-5m16 0h-2.586a1 1 0 00-.707.293l-2.414 2.414a1 1 0 01-.707.293h-3.172a1 1 0 01-.707-.293l-2.414-2.414A1 1 0 006.586 13H4"></path>
        </svg>
      </slot>
    </div>

    <div class="empty-content">
      <h3 v-if="title" class="empty-title">{{ title }}</h3>
      <p v-if="description" class="empty-description">{{ description }}</p>
    </div>

    <div v-if="$slots.action" class="empty-action">
      <slot name="action"></slot>
    </div>
  </div>
</template>

<script lang="ts">
import { defineComponent } from 'vue';

export default defineComponent({
  name: 'EmptyState',

  props: {
    title: {
      type: String,
      default: undefined,
    },
    description: {
      type: String,
      default: undefined,
    },
  },
});
</script>

<style scoped>
.empty-state {
  @apply flex flex-col items-center justify-center py-12 px-4 text-center;
}

.empty-icon {
  @apply mb-4;
}

.empty-content {
  @apply mb-6;
}

.empty-title {
  @apply text-lg font-semibold text-text dark:text-text-dark mb-2;
}

.empty-description {
  @apply text-sm text-secondary dark:text-secondary-dark max-w-md;
}

.empty-action {
  @apply flex gap-2;
}
</style>
