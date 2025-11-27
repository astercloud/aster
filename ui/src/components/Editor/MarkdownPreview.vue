<template>
  <div class="markdown-preview">
    <!-- 内容区域 -->
    <div
      v-if="renderedContent"
      class="preview-content"
      v-html="renderedContent"
    ></div>

    <!-- 空状态 -->
    <div v-else class="preview-empty">
      <svg class="w-12 h-12 text-slate-300 dark:text-slate-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path
          stroke-linecap="round"
          stroke-linejoin="round"
          stroke-width="2"
          d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"
        />
      </svg>
      <p class="text-sm text-slate-500 dark:text-slate-400 mt-2">{{ emptyText }}</p>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import { renderMarkdown } from '@/utils/markdown';

interface Props {
  content: string;
  emptyText?: string;
}

const props = withDefaults(defineProps<Props>(), {
  emptyText: '暂无内容',
});

const renderedContent = computed(() => {
  if (!props.content) return '';
  return renderMarkdown(props.content);
});
</script>

<style scoped>
.markdown-preview {
  @apply min-h-[200px] p-4 border border-slate-200 dark:border-slate-700 rounded-lg bg-white dark:bg-slate-800;
}

.preview-content {
  @apply text-slate-900 dark:text-slate-100 prose prose-slate dark:prose-invert max-w-none;
}

/* 覆盖 prose 样式 */
.preview-content :deep(h1) {
  @apply text-2xl font-bold mb-4 text-slate-900 dark:text-slate-100;
}

.preview-content :deep(h2) {
  @apply text-xl font-bold mb-3 text-slate-900 dark:text-slate-100;
}

.preview-content :deep(h3) {
  @apply text-lg font-semibold mb-2 text-slate-900 dark:text-slate-100;
}

.preview-content :deep(p) {
  @apply mb-4 leading-relaxed;
}

.preview-content :deep(ul),
.preview-content :deep(ol) {
  @apply mb-4 pl-6;
}

.preview-content :deep(li) {
  @apply mb-1;
}

.preview-content :deep(code) {
  @apply bg-slate-100 dark:bg-slate-900 px-1.5 py-0.5 rounded text-sm font-mono text-blue-600 dark:text-blue-400;
}

.preview-content :deep(pre) {
  @apply bg-slate-900 dark:bg-slate-950 text-slate-100 p-4 rounded-lg my-4 overflow-x-auto;
}

.preview-content :deep(pre code) {
  @apply bg-transparent text-slate-100 p-0;
}

.preview-content :deep(blockquote) {
  @apply border-l-4 border-slate-300 dark:border-slate-600 pl-4 py-2 my-4 italic text-slate-700 dark:text-slate-300;
}

.preview-content :deep(a) {
  @apply text-blue-600 dark:text-blue-400 hover:underline;
}

.preview-content :deep(hr) {
  @apply my-6 border-slate-200 dark:border-slate-700;
}

.preview-content :deep(table) {
  @apply w-full my-4 border-collapse;
}

.preview-content :deep(th) {
  @apply bg-slate-100 dark:bg-slate-900 border border-slate-200 dark:border-slate-700 px-4 py-2 text-left font-semibold;
}

.preview-content :deep(td) {
  @apply border border-slate-200 dark:border-slate-700 px-4 py-2;
}

.preview-empty {
  @apply flex flex-col items-center justify-center py-12 text-center;
}
</style>
