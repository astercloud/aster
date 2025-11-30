<template>
  <div class="card-message">
    <!-- Card Header -->
    <div v-if="message.content.title || message.content.subtitle" class="card-header">
      <h3 v-if="message.content.title" class="card-title">
        {{ message.content.title }}
      </h3>
      <p v-if="message.content.subtitle" class="card-subtitle">
        {{ message.content.subtitle }}
      </p>
    </div>

    <!-- Card Image -->
    <div v-if="message.content.image" class="card-image">
      <img
        :src="message.content.image.url"
        :alt="message.content.image.alt || '卡片图片'"
        loading="lazy"
      />
    </div>

    <!-- Card Body -->
    <div v-if="message.content.description" class="card-body">
      <p class="card-description">{{ message.content.description }}</p>
    </div>

    <!-- Card Fields -->
    <div v-if="message.content.fields && message.content.fields.length > 0" class="card-fields">
      <div
        v-for="(field, index) in message.content.fields"
        :key="index"
        class="card-field"
      >
        <span class="field-label">{{ field.label }}</span>
        <span class="field-value">{{ field.value }}</span>
      </div>
    </div>

    <!-- Card Actions -->
    <div v-if="message.content.actions && message.content.actions.length > 0" class="card-actions">
      <button
        v-for="(action, index) in message.content.actions"
        :key="index"
        @click="handleAction(action)"
        :class="[
          'card-action-btn',
          action.style === 'primary' ? 'btn-primary' : 'btn-secondary'
        ]"
      >
        <svg v-if="action.icon" class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" :d="getIconPath(action.icon)"></path>
        </svg>
        {{ action.label }}
      </button>
    </div>

    <!-- Card Footer -->
    <div v-if="message.content.footer" class="card-footer">
      <span class="footer-text">{{ message.content.footer }}</span>
    </div>
  </div>
</template>

<script lang="ts">
import { defineComponent } from 'vue';
import type { CardMessage as CardMessageType } from '@/types';

export default defineComponent({
  name: 'CardMessage',

  props: {
    message: {
      type: Object as () => CardMessageType,
      required: true,
    },
  },

  emits: {
    action: (action: unknown) => true,
  },

  setup(props, { emit }) {
    function handleAction(action: unknown) {
      emit('action', action);
    }

    function getIconPath(icon: string): string {
      const icons: Record<string, string> = {
        link: 'M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14',
        download: 'M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4',
        share: 'M8.684 13.342C8.886 12.938 9 12.482 9 12c0-.482-.114-.938-.316-1.342m0 2.684a3 3 0 110-2.684m0 2.684l6.632 3.316m-6.632-6l6.632-3.316m0 0a3 3 0 105.367-2.684 3 3 0 00-5.367 2.684zm0 9.316a3 3 0 105.368 2.684 3 3 0 00-5.368-2.684z',
      };
      return icons[icon] || '';
    }

    return {
      handleAction,
      getIconPath,
    };
  },
});
</script>

<style scoped>
.card-message {
  @apply bg-surface dark:bg-surface-dark border border-border dark:border-border-dark rounded-lg overflow-hidden shadow-sm;
  max-width: 400px;
}

.card-header {
  @apply px-4 pt-4 pb-2;
}

.card-title {
  @apply text-base font-semibold text-text dark:text-text-dark mb-1;
}

.card-subtitle {
  @apply text-sm text-secondary dark:text-secondary-dark;
}

.card-image {
  @apply w-full;
}

.card-image img {
  @apply w-full h-auto object-cover;
  max-height: 200px;
}

.card-body {
  @apply px-4 py-3;
}

.card-description {
  @apply text-sm text-text dark:text-text-dark leading-relaxed;
}

.card-fields {
  @apply px-4 py-2 space-y-2 border-t border-border dark:border-border-dark;
}

.card-field {
  @apply flex justify-between items-center text-sm;
}

.field-label {
  @apply text-secondary dark:text-secondary-dark font-medium;
}

.field-value {
  @apply text-text dark:text-text-dark;
}

.card-actions {
  @apply flex gap-2 px-4 py-3 border-t border-border dark:border-border-dark;
}

.card-action-btn {
  @apply flex items-center gap-2 px-4 py-2 rounded-lg text-sm font-medium transition-colors;
}

.btn-primary {
  @apply bg-primary hover:bg-primary-hover text-white;
}

.btn-secondary {
  @apply bg-background dark:bg-background-dark hover:bg-border dark:hover:bg-border-dark text-text dark:text-text-dark border border-border dark:border-border-dark;
}

.card-footer {
  @apply px-4 py-2 bg-background dark:bg-background-dark border-t border-border dark:border-border-dark;
}

.footer-text {
  @apply text-xs text-secondary dark:text-secondary-dark;
}
</style>
