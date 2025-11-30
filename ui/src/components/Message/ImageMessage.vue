<template>
  <div class="image-message">
    <div v-if="message.content.caption" class="image-caption">
      {{ message.content.caption }}
    </div>

    <div class="image-wrapper">
      <img
        :src="message.content.url"
        :alt="message.content.alt || '图片'"
        :class="['image-content', { 'loading': isLoading }]"
        @load="handleLoad"
        @error="handleError"
        loading="lazy"
      />

      <!-- Loading State -->
      <div v-if="isLoading" class="image-loading">
        <svg class="w-8 h-8 animate-spin text-primary" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"></path>
        </svg>
      </div>

      <!-- Error State -->
      <div v-if="hasError" class="image-error">
        <svg class="w-8 h-8 text-red-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"></path>
        </svg>
        <p class="text-xs text-red-600 mt-2">图片加载失败</p>
      </div>

      <!-- Preview Button -->
      <button
        v-if="!isLoading && !hasError"
        @click="openPreview"
        class="image-preview-btn"
        title="预览图片"
      >
        <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0zM10 7v3m0 0v3m0-3h3m-3 0H7"></path>
        </svg>
      </button>
    </div>

    <!-- Metadata -->
    <div v-if="message.content.metadata" class="image-metadata">
      <span v-if="message.content.metadata.size" class="metadata-item">
        {{ formatFileSize(message.content.metadata.size) }}
      </span>
      <span v-if="message.content.metadata.dimensions" class="metadata-item">
        {{ message.content.metadata.dimensions.width }} × {{ message.content.metadata.dimensions.height }}
      </span>
    </div>
  </div>
</template>

<script lang="ts">
import { defineComponent, ref } from 'vue';
import type { ImageMessage as ImageMessageType } from '@/types';
import { formatFileSize } from '@/utils/format';

export default defineComponent({
  name: 'ImageMessage',

  props: {
    message: {
      type: Object as () => ImageMessageType,
      required: true,
    },
  },

  emits: {
    preview: (url: string) => true,
  },

  setup(props, { emit }) {
    const isLoading = ref(true);
    const hasError = ref(false);

    function handleLoad() {
      isLoading.value = false;
    }

    function handleError() {
      isLoading.value = false;
      hasError.value = true;
    }

    function openPreview() {
      emit('preview', props.message.content.url);
    }

    return {
      isLoading,
      hasError,
      handleLoad,
      handleError,
      openPreview,
      formatFileSize,
    };
  },
});
</script>

<style scoped>
.image-message {
  @apply space-y-2;
}

.image-caption {
  @apply text-sm text-secondary dark:text-secondary-dark;
}

.image-wrapper {
  @apply relative rounded-lg overflow-hidden bg-background dark:bg-background-dark;
  max-width: 400px;
  max-height: 400px;
}

.image-content {
  @apply w-full h-full object-contain;
  transition: opacity 0.3s;
}

.image-content.loading {
  @apply opacity-0;
}

.image-loading,
.image-error {
  @apply absolute inset-0 flex flex-col items-center justify-center bg-background dark:bg-background-dark;
}

.image-preview-btn {
  @apply absolute top-2 right-2 p-2 bg-black/50 hover:bg-black/70 text-white rounded-lg opacity-0 transition-opacity;
}

.image-container:hover .image-preview-btn {
  @apply opacity-100;
}

.image-wrapper:hover .image-preview-btn {
  @apply opacity-100;
}

.image-metadata {
  @apply flex gap-2 text-xs text-secondary dark:text-secondary-dark;
}

.metadata-item {
  @apply px-2 py-1 bg-surface dark:bg-surface-dark rounded border border-border dark:border-border-dark;
}
</style>
