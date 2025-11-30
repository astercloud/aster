<template>
  <div class="workflow-progress">
    <!-- 进度条 -->
    <div class="progress-bar-container">
      <div
        class="progress-bar-fill"
        :style="{ width: `${progressPercentage}%` }"
      >
        <div class="progress-bar-shimmer"></div>
      </div>
    </div>

    <!-- 进度文本 -->
    <div class="progress-text">
      <div class="progress-stats">
        <span class="progress-fraction">
          {{ completed }} / {{ total }}
        </span>
        <span class="progress-percentage">
          {{ progressPercentage }}%
        </span>
      </div>

      <!-- 当前步骤 -->
      <div v-if="current >= 0 && current < total" class="progress-current">
        <svg class="w-3 h-3 text-blue-600 dark:text-blue-400" fill="currentColor" viewBox="0 0 20 20">
          <path
            fill-rule="evenodd"
            d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z"
            clip-rule="evenodd"
          />
        </svg>
        <span class="text-sm text-slate-600 dark:text-slate-400">
          步骤 {{ current + 1 }} / {{ total }}
        </span>
      </div>
    </div>

    <!-- 步骤指示点 -->
    <div v-if="showDots" class="progress-dots">
      <div
        v-for="(_, index) in total"
        :key="index"
        :class="[
          'progress-dot',
          {
            'progress-dot--completed': index < completed,
            'progress-dot--active': index === current,
            'progress-dot--pending': index > current,
          }
        ]"
        :title="`步骤 ${index + 1}`"
      ></div>
    </div>
  </div>
</template>

<script lang="ts">
import { defineComponent, computed } from 'vue';

export default defineComponent({
  name: 'WorkflowProgress',
  props: {
    total: {
      type: Number,
      required: true,
    },
    completed: {
      type: Number,
      required: true,
    },
    current: {
      type: Number,
      default: -1,
    },
    showDots: {
      type: Boolean,
      default: true,
    },
  },
  setup(props) {
    const progressPercentage = computed(() => {
      if (props.total === 0) return 0;
      return Math.round((props.completed / props.total) * 100);
    });

    return {
      progressPercentage,
    };
  },
});
</script>

<style scoped>
.workflow-progress {
  @apply space-y-3;
}

.progress-bar-container {
  @apply relative h-2 bg-slate-200 dark:bg-slate-700 rounded-full overflow-hidden;
}

.progress-bar-fill {
  @apply h-full bg-gradient-to-r from-blue-500 to-blue-600 dark:from-blue-600 dark:to-blue-700 rounded-full transition-all duration-500 ease-out relative;
}

.progress-bar-shimmer {
  @apply absolute inset-0 bg-gradient-to-r from-transparent via-white/30 to-transparent animate-shimmer;
}

@keyframes shimmer {
  0% {
    transform: translateX(-100%);
  }
  100% {
    transform: translateX(100%);
  }
}

.progress-text {
  @apply flex items-center justify-between;
}

.progress-stats {
  @apply flex items-center gap-3;
}

.progress-fraction {
  @apply text-sm font-semibold text-slate-700 dark:text-slate-300;
}

.progress-percentage {
  @apply text-xs text-slate-500 dark:text-slate-400 font-medium;
}

.progress-current {
  @apply flex items-center gap-1.5;
}

.progress-dots {
  @apply flex items-center gap-2;
}

.progress-dot {
  @apply w-2 h-2 rounded-full transition-all duration-300;
}

.progress-dot--completed {
  @apply bg-emerald-500 dark:bg-emerald-600 scale-110;
}

.progress-dot--active {
  @apply bg-blue-500 dark:bg-blue-600 scale-125 animate-pulse ring-4 ring-blue-200 dark:ring-blue-900/50;
}

.progress-dot--pending {
  @apply bg-slate-300 dark:bg-slate-600;
}

/* Shimmer animation */
.animate-shimmer {
  animation: shimmer 2s infinite;
}
</style>
