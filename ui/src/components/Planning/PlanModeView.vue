<template>
  <Transition name="slide-fade">
    <div v-if="active" class="plan-mode-overlay" @click.self="handleClose">
      <div class="plan-mode-view" @click.stop>
        <div class="plan-header">
          <div class="header-title">
            <span class="plan-badge">🗂️ Plan Mode</span>
            <h3>实施计划审批</h3>
          </div>
          <button @click="handleClose" class="btn-close" title="关闭">
            <svg xmlns="http://www.w3.org/2000/svg" class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>

        <div class="plan-content">
          <div class="content-header">
            <h4>计划内容</h4>
            <span class="content-hint">请仔细审阅实施方案</span>
          </div>
          <pre class="plan-text">{{ content }}</pre>
        </div>

        <div class="plan-actions">
          <button @click="handleApprove" class="btn-action btn-approve">
            <svg xmlns="http://www.w3.org/2000/svg" class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
            </svg>
            批准计划
          </button>
          <button @click="handleReject" class="btn-action btn-reject">
            <svg xmlns="http://www.w3.org/2000/svg" class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
            </svg>
            拒绝计划
          </button>
        </div>
      </div>
    </div>
  </Transition>
</template>

<script setup lang="ts">
interface Props {
  active: boolean;
  content: string;
  planId: string | null;
}

defineProps<Props>();

const emit = defineEmits<{
  approve: [];
  reject: [];
  close: [];
}>();

const handleApprove = () => emit("approve");
const handleReject = () => emit("reject");
const handleClose = () => emit("close");
</script>

<style scoped>
/* 过渡动画 */
.slide-fade-enter-active {
  transition: all 0.3s ease-out;
}

.slide-fade-leave-active {
  transition: all 0.2s cubic-bezier(1, 0.5, 0.8, 1);
}

.slide-fade-enter-from,
.slide-fade-leave-to {
  transform: translateY(-20px);
  opacity: 0;
}

/* 遮罩层 */
.plan-mode-overlay {
  @apply fixed inset-0 bg-black/50 dark:bg-black/70 z-50 flex items-start justify-center pt-20 px-4
         backdrop-blur-sm;
}

/* 主容器 */
.plan-mode-view {
  @apply w-full max-w-3xl bg-white dark:bg-gray-800 rounded-2xl shadow-2xl overflow-hidden
         border border-purple-200 dark:border-purple-700;
  max-height: calc(100vh - 10rem);
  display: flex;
  flex-direction: column;
}

/* 头部 */
.plan-header {
  @apply flex items-center justify-between px-6 py-4 bg-gradient-to-r from-purple-100 to-indigo-100
         dark:from-purple-900/30 dark:to-indigo-900/30 border-b border-purple-200 dark:border-purple-700;
}

.header-title {
  @apply flex items-center gap-3;
}

.plan-badge {
  @apply px-3 py-1 bg-purple-200 dark:bg-purple-800 text-purple-800 dark:text-purple-200
         font-bold rounded-full text-sm;
}

.plan-header h3 {
  @apply text-lg font-bold text-gray-900 dark:text-gray-100;
}

.btn-close {
  @apply p-2 text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200
         rounded-lg hover:bg-gray-200 dark:hover:bg-gray-700 transition-colors;
}

/* 内容区域 */
.plan-content {
  @apply flex-1 overflow-y-auto p-6;
}

.content-header {
  @apply mb-4 flex items-center justify-between;
}

.content-header h4 {
  @apply text-base font-bold text-gray-900 dark:text-gray-100;
}

.content-hint {
  @apply text-xs text-gray-500 dark:text-gray-400 italic;
}

.plan-text {
  @apply text-sm bg-gray-50 dark:bg-gray-900 p-4 rounded-lg whitespace-pre-wrap
         text-gray-800 dark:text-gray-200 font-mono leading-relaxed
         border border-gray-200 dark:border-gray-700;
  max-height: 60vh;
  overflow-y: auto;
}

/* 操作按钮 */
.plan-actions {
  @apply flex gap-3 px-6 py-4 bg-gray-50 dark:bg-gray-900/50 border-t border-gray-200 dark:border-gray-700;
}

.btn-action {
  @apply flex-1 flex items-center justify-center gap-2 px-4 py-3 font-bold rounded-lg
         transition-all duration-200 shadow-md hover:shadow-lg active:scale-95;
}

.btn-approve {
  @apply bg-gradient-to-r from-green-500 to-emerald-600 text-white
         hover:from-green-600 hover:to-emerald-700;
}

.btn-reject {
  @apply bg-gradient-to-r from-red-500 to-rose-600 text-white
         hover:from-red-600 hover:to-rose-700;
}

/* 滚动条样式 */
.plan-text::-webkit-scrollbar {
  @apply w-2;
}

.plan-text::-webkit-scrollbar-track {
  @apply bg-gray-200 dark:bg-gray-800 rounded;
}

.plan-text::-webkit-scrollbar-thumb {
  @apply bg-gray-400 dark:bg-gray-600 rounded hover:bg-gray-500 dark:hover:bg-gray-500;
}
</style>
