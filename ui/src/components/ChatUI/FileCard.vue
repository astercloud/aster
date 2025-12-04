<template>
  <div class="file-card">
    <div class="file-icon">
      <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M7 21h10a2 2 0 002-2V9.414a1 1 0 00-.293-.707l-5.414-5.414A1 1 0 0012.586 3H7a2 2 0 00-2 2v14a2 2 0 002 2z"></path>
      </svg>
    </div>
    <div class="file-info">
      <div class="file-name">{{ file.name }}</div>
      <div class="file-size">{{ formatSize(file.size) }}</div>
    </div>
    <a v-if="file.url" :href="file.url" target="_blank" class="file-download">
      <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4"></path>
      </svg>
    </a>
  </div>
</template>

<script setup lang="ts">
interface FileInfo {
  name: string;
  size: number;
  url?: string;
}

interface Props {
  file: FileInfo;
}

defineProps<Props>();

const formatSize = (bytes: number): string => {
  if (bytes === 0) return "0 B";
  const k = 1024;
  const sizes = ["B", "KB", "MB", "GB"];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return `${(bytes / Math.pow(k, i)).toFixed(1)} ${sizes[i]}`;
};
</script>

<style scoped>
.file-card {
  @apply flex items-center gap-3 p-4 bg-gray-50 dark:bg-gray-800 border border-gray-200 dark:border-gray-700 rounded-xl max-w-sm;
}

.file-icon {
  @apply text-blue-500 dark:text-blue-400;
}

.file-info {
  @apply flex-1 min-w-0;
}

.file-name {
  @apply text-sm font-medium text-gray-900 dark:text-gray-100 truncate;
}

.file-size {
  @apply text-xs text-gray-500 dark:text-gray-400;
}

.file-download {
  @apply text-blue-500 hover:text-blue-600 dark:text-blue-400 dark:hover:text-blue-300 transition-colors;
}
</style>
