<template>
  <div v-if="hasError" class="error-boundary">
    <div class="error-content">
      <svg class="error-icon" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path
          stroke-linecap="round"
          stroke-linejoin="round"
          stroke-width="2"
          d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"
        />
      </svg>

      <h3 class="error-title">{{ errorTitle }}</h3>
      <p v-if="errorMessage" class="error-message">{{ errorMessage }}</p>

      <!-- 错误详情 (仅开发模式) -->
      <details v-if="showDetails && error" class="error-details">
        <summary class="error-details-summary">查看错误详情</summary>
        <pre class="error-stack">{{ error.stack || error.message }}</pre>
      </details>

      <div class="error-actions">
        <button @click="handleRetry" class="error-button error-button-primary">
          <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path
              stroke-linecap="round"
              stroke-linejoin="round"
              stroke-width="2"
              d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"
            />
          </svg>
          重试
        </button>

        <button v-if="onReset" @click="handleReset" class="error-button error-button-secondary">
          重置
        </button>
      </div>
    </div>
  </div>

  <!-- 正常渲染子组件 -->
  <slot v-else></slot>
</template>

<script lang="ts">
import { defineComponent, ref, onErrorCaptured, provide } from 'vue';

export default defineComponent({
  name: 'ErrorBoundary',

  props: {
    errorTitle: {
      type: String,
      default: '出错了',
    },
    errorMessage: {
      type: String,
      default: '渲染组件时发生错误',
    },
    showDetails: {
      type: Boolean,
      default: import.meta.env.DEV,
    },
    onRetry: {
      type: Function as unknown as () => (() => void) | undefined,
      default: undefined,
    },
    onReset: {
      type: Function as unknown as () => (() => void) | undefined,
      default: undefined,
    },
    onError: {
      type: Function as unknown as () => ((error: Error) => void) | undefined,
      default: undefined,
    },
  },

  setup(props) {
    const hasError = ref(false);
    const error = ref<Error | null>(null);

    // 捕获子组件错误
    onErrorCaptured((err: Error) => {
      console.error('ErrorBoundary caught error:', err);

      hasError.value = true;
      error.value = err;

      // 调用外部错误处理函数
      if (props.onError) {
        props.onError(err);
      }

      // 阻止错误继续向上传播
      return false;
    });

    // 提供重置错误状态的方法给子组件
    provide('resetError', () => {
      hasError.value = false;
      error.value = null;
    });

    const handleRetry = () => {
      hasError.value = false;
      error.value = null;

      if (props.onRetry) {
        props.onRetry();
      }
    };

    const handleReset = () => {
      if (props.onReset) {
        props.onReset();
      }

      hasError.value = false;
      error.value = null;
    };

    return {
      hasError,
      error,
      handleRetry,
      handleReset,
    };
  },
});
</script>

<style scoped>
.error-boundary {
  @apply flex items-center justify-center min-h-[200px] p-6;
}

.error-content {
  @apply max-w-md w-full text-center;
}

.error-icon {
  @apply w-16 h-16 mx-auto mb-4 text-red-500 dark:text-red-400;
}

.error-title {
  @apply text-xl font-semibold text-slate-900 dark:text-slate-100 mb-2;
}

.error-message {
  @apply text-sm text-slate-600 dark:text-slate-400 mb-4;
}

.error-details {
  @apply mt-4 text-left bg-slate-50 dark:bg-slate-900 border border-slate-200 dark:border-slate-700 rounded-lg overflow-hidden;
}

.error-details-summary {
  @apply px-4 py-2 text-sm font-medium text-slate-700 dark:text-slate-300 cursor-pointer hover:bg-slate-100 dark:hover:bg-slate-800 transition-colors;
}

.error-stack {
  @apply px-4 py-3 text-xs font-mono text-red-600 dark:text-red-400 overflow-x-auto;
}

.error-actions {
  @apply flex gap-2 justify-center mt-6;
}

.error-button {
  @apply flex items-center gap-2 px-4 py-2 text-sm font-medium rounded-lg transition-colors;
}

.error-button-primary {
  @apply bg-blue-600 hover:bg-blue-700 text-white;
}

.error-button-secondary {
  @apply bg-slate-200 dark:bg-slate-700 hover:bg-slate-300 dark:hover:bg-slate-600 text-slate-700 dark:text-slate-300;
}
</style>
