<template>
  <div class="list-message">
    <!-- List Header -->
    <div v-if="message.content.title" class="list-header">
      <h3 class="list-title">{{ message.content.title }}</h3>
      <span v-if="message.content.items" class="list-count">
        {{ message.content.items.length }} 项
      </span>
    </div>

    <!-- List Items -->
    <div class="list-items">
      <div
        v-for="(item, index) in message.content.items"
        :key="index"
        :class="['list-item', { 'clickable': item.action }]"
        @click="handleItemClick(item)"
      >
        <!-- Item Icon/Image -->
        <div v-if="item.icon || item.image" class="item-media">
          <img
            v-if="item.image"
            :src="item.image"
            :alt="item.title"
            class="item-image"
          />
          <div v-else-if="item.icon" class="item-icon">
            {{ item.icon }}
          </div>
        </div>

        <!-- Item Content -->
        <div class="item-content">
          <h4 class="item-title">{{ item.title }}</h4>
          <p v-if="item.description" class="item-description">
            {{ item.description }}
          </p>
          <div v-if="item.metadata" class="item-metadata">
            <span
              v-for="(meta, key) in item.metadata"
              :key="key"
              class="metadata-tag"
            >
              {{ meta }}
            </span>
          </div>
        </div>

        <!-- Item Action -->
        <div v-if="item.action" class="item-action">
          <svg class="w-5 h-5 text-secondary dark:text-secondary-dark" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7"></path>
          </svg>
        </div>
      </div>
    </div>

    <!-- List Footer -->
    <div v-if="message.content.footer" class="list-footer">
      <button
        v-if="message.content.footer.action"
        @click="handleFooterAction"
        class="footer-action"
      >
        {{ message.content.footer.text }}
      </button>
      <span v-else class="footer-text">
        {{ message.content.footer.text }}
      </span>
    </div>
  </div>
</template>

<script lang="ts">
import { defineComponent } from 'vue';
import type { ListMessage as ListMessageType } from '@/types';

export default defineComponent({
  name: 'ListMessage',

  props: {
    message: {
      type: Object as () => ListMessageType,
      required: true,
    },
  },

  emits: {
    itemClick: (item: unknown) => true,
    footerAction: () => true,
  },

  setup(props, { emit }) {
    function handleItemClick(item: unknown & { action?: unknown }) {
      if ((item as { action?: unknown }).action) {
        emit('itemClick', item);
      }
    }

    function handleFooterAction() {
      emit('footerAction');
    }

    return {
      handleItemClick,
      handleFooterAction,
    };
  },
});
</script>

<style scoped>
.list-message {
  @apply bg-surface dark:bg-surface-dark border border-border dark:border-border-dark rounded-lg overflow-hidden;
  max-width: 500px;
}

.list-header {
  @apply flex items-center justify-between px-4 py-3 border-b border-border dark:border-border-dark;
}

.list-title {
  @apply text-base font-semibold text-text dark:text-text-dark;
}

.list-count {
  @apply text-xs text-secondary dark:text-secondary-dark bg-background dark:bg-background-dark px-2 py-1 rounded-full;
}

.list-items {
  @apply divide-y divide-border dark:divide-border-dark;
}

.list-item {
  @apply flex items-center gap-3 px-4 py-3 transition-colors;
}

.list-item.clickable {
  @apply cursor-pointer hover:bg-background dark:hover:bg-background-dark;
}

.item-media {
  @apply flex-shrink-0;
}

.item-image {
  @apply w-12 h-12 rounded-lg object-cover;
}

.item-icon {
  @apply w-12 h-12 rounded-lg bg-primary/10 dark:bg-primary/20 flex items-center justify-center text-2xl;
}

.item-content {
  @apply flex-1 min-w-0;
}

.item-title {
  @apply text-sm font-medium text-text dark:text-text-dark truncate;
}

.item-description {
  @apply text-xs text-secondary dark:text-secondary-dark mt-1 line-clamp-2;
}

.item-metadata {
  @apply flex gap-2 mt-2;
}

.metadata-tag {
  @apply text-xs px-2 py-0.5 bg-background dark:bg-background-dark text-secondary dark:text-secondary-dark rounded-full;
}

.item-action {
  @apply flex-shrink-0;
}

.list-footer {
  @apply px-4 py-3 border-t border-border dark:border-border-dark bg-background dark:bg-background-dark;
}

.footer-action {
  @apply w-full text-sm font-medium text-primary hover:text-primary-hover dark:text-primary-light transition-colors;
}

.footer-text {
  @apply text-xs text-secondary dark:text-secondary-dark;
}
</style>
