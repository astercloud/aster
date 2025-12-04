<template>
  <div class="rich-text" v-html="renderedContent"></div>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { marked } from "marked";

interface Props {
  content: string;
}

const props = defineProps<Props>();

const renderedContent = computed(() => {
  try {
    return marked(props.content);
  } catch {
    return props.content;
  }
});
</script>

<style scoped>
.rich-text {
  @apply text-gray-900 dark:text-gray-100;
}

.rich-text :deep(h1) {
  @apply text-2xl font-bold mt-6 mb-4;
}

.rich-text :deep(h2) {
  @apply text-xl font-bold mt-5 mb-3;
}

.rich-text :deep(h3) {
  @apply text-lg font-bold mt-4 mb-2;
}

.rich-text :deep(p) {
  @apply my-2 leading-relaxed;
}

.rich-text :deep(a) {
  @apply text-blue-600 dark:text-blue-400 hover:underline;
}

.rich-text :deep(code) {
  @apply px-1.5 py-0.5 bg-gray-100 dark:bg-gray-800 rounded text-sm font-mono;
}

.rich-text :deep(pre) {
  @apply my-4 p-4 bg-gray-900 dark:bg-gray-950 rounded-lg overflow-x-auto;
}

.rich-text :deep(pre code) {
  @apply p-0 bg-transparent text-gray-100;
}

.rich-text :deep(ul) {
  @apply my-2 ml-6 list-disc;
}

.rich-text :deep(ol) {
  @apply my-2 ml-6 list-decimal;
}

.rich-text :deep(li) {
  @apply my-1;
}

.rich-text :deep(blockquote) {
  @apply my-4 pl-4 border-l-4 border-gray-300 dark:border-gray-600 italic text-gray-600 dark:text-gray-400;
}

.rich-text :deep(table) {
  @apply my-4 w-full border-collapse;
}

.rich-text :deep(th) {
  @apply px-4 py-2 bg-gray-100 dark:bg-gray-800 border border-gray-300 dark:border-gray-600 font-semibold text-left;
}

.rich-text :deep(td) {
  @apply px-4 py-2 border border-gray-300 dark:border-gray-600;
}
</style>
