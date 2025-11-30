<template>
  <div class="thinking-timeline">
    <ThinkingStep
      v-for="(step, index) in steps"
      :key="step.id || `step-${index}`"
      :step="step"
      :is-last="index === steps.length - 1"
    />

    <!-- 空状态 -->
    <div v-if="steps.length === 0" class="empty-state">
      <svg class="w-12 h-12 text-slate-300" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path
          stroke-linecap="round"
          stroke-linejoin="round"
          stroke-width="2"
          d="M9.663 17h4.673M12 3v1m6.364 1.636l-.707.707M21 12h-1M4 12H3m3.343-5.657l-.707-.707m2.828 9.9a5 5 0 117.072 0l-.548.547A3.374 3.374 0 0014 18.469V19a2 2 0 11-4 0v-.531c0-.895-.356-1.754-.988-2.386l-.548-.547z"
        />
      </svg>
      <p class="text-sm text-slate-500">暂无思考记录</p>
    </div>
  </div>
</template>

<script lang="ts">
import { defineComponent } from 'vue';
import ThinkingStep from './ThinkingStep.vue';
import type { ThinkingStep as ThinkingStepType } from '@/types/thinking';
import type { PropType } from 'vue';

export default defineComponent({
  name: 'ThinkingTimeline',
  components: {
    ThinkingStep,
  },
  props: {
    steps: {
      type: Array as PropType<ThinkingStepType[]>,
      required: true,
    },
  },
});
</script>

<style scoped>
.thinking-timeline {
  @apply px-4 py-3 space-y-0;
}

.empty-state {
  @apply flex flex-col items-center justify-center py-8 text-center;
}

.empty-state svg {
  @apply mb-2;
}
</style>
