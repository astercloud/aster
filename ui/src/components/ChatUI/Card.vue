<template>
  <div class="card">
    <div v-if="title" class="card-header">
      <h3 class="card-title">{{ title }}</h3>
    </div>
    <div class="card-body">
      <div class="card-content" v-html="content"></div>
    </div>
    <div v-if="actions && actions.length > 0" class="card-actions">
      <button v-for="action in actions" :key="action.value" class="card-action-btn" @click="$emit('action', action)">
        {{ action.text }}
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
interface Action {
  text: string;
  value: string;
}

interface Props {
  title?: string;
  content: string;
  actions?: Action[];
}

defineProps<Props>();

defineEmits<{
  action: [action: Action];
}>();
</script>

<style scoped>
.card {
  @apply bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-700 rounded-xl overflow-hidden shadow-sm max-w-sm;
}

.card-header {
  @apply px-4 py-3 border-b border-gray-200 dark:border-gray-700;
}

.card-title {
  @apply text-base font-semibold text-gray-900 dark:text-gray-100;
}

.card-body {
  @apply px-4 py-3;
}

.card-content {
  @apply text-sm text-gray-700 dark:text-gray-300 leading-relaxed;
}

.card-actions {
  @apply flex gap-2 px-4 py-3 border-t border-gray-200 dark:border-gray-700;
}

.card-action-btn {
  @apply flex-1 px-4 py-2 bg-blue-500 hover:bg-blue-600 text-white text-sm font-medium rounded-lg transition-colors;
}
</style>
