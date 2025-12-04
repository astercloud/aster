<template>
  <div class="think-bubble">
    <div class="think-icon">
      <svg class="w-5 h-5 animate-spin" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path
          stroke-linecap="round"
          stroke-linejoin="round"
          stroke-width="2"
          d="M9.663 17h4.673M12 3v1m6.364 1.636l-.707.707M21 12h-1M4 12H3m3.343-5.657l-.707-.707m2.828 9.9a5 5 0 117.072 0l-.548.547A3.374 3.374 0 0014 18.469V19a2 2 0 11-4 0v-.531c0-.895-.356-1.754-.988-2.386l-.548-.547z"
        ></path>
      </svg>
    </div>
    <div class="think-content">
      <div class="think-title">思考中</div>
    </div>
  </div>
</template>

<script setup lang="ts">
// ThinkBubble组件不再需要content属性，只显示思考状态
</script>

<style scoped>
.think-bubble {
  @apply flex items-start gap-3 p-4 bg-amber-50 dark:bg-amber-900/20 border border-amber-200 dark:border-amber-800 rounded-xl max-w-md;
}

.think-icon {
  @apply text-amber-600 dark:text-amber-400;
}

.think-content {
  @apply flex-1;
}

.think-title {
  @apply text-sm font-semibold text-amber-900 dark:text-amber-100;
}

.think-text {
  @apply text-xs text-amber-700 dark:text-amber-300 mt-1;
}
</style>
