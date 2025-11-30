<template>
  <div class="workflow-progress-view">
    <!-- 标题区域 -->
    <div v-if="title || showProgress" class="view-header">
      <h4 v-if="title" class="view-title">{{ title }}</h4>
      <WorkflowProgress
        v-if="showProgress && steps.length > 0"
        :total="steps.length"
        :completed="completedCount"
        :current="currentIndex"
        :show-dots="showDots"
      />
    </div>

    <!-- 步骤列表 -->
    <div v-if="showSteps" class="view-steps">
      <WorkflowStep
        v-for="(step, index) in visibleSteps"
        :key="step.id"
        :step="step"
        :is-last="index === visibleSteps.length - 1"
        :is-clickable="allowNavigation"
        :show-metadata="showMetadata"
        @click="handleStepClick"
        @action="handleStepAction"
      />

      <!-- 展开/收起按钮 -->
      <button
        v-if="hasMoreSteps"
        @click="toggleExpanded"
        class="toggle-button"
      >
        <span v-if="isExpanded">收起</span>
        <span v-else>显示全部 {{ steps.length }} 个步骤</span>
        <svg
          :class="['w-4 h-4 transition-transform', { 'rotate-180': isExpanded }]"
          fill="none"
          stroke="currentColor"
          viewBox="0 0 24 24"
        >
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7" />
        </svg>
      </button>
    </div>

    <!-- 空状态 -->
    <div v-if="steps.length === 0" class="view-empty">
      <p class="text-sm text-slate-500 dark:text-slate-400">暂无工作流步骤</p>
    </div>
  </div>
</template>

<script lang="ts">
import { defineComponent, ref, computed, type PropType } from 'vue';
import WorkflowStep from './WorkflowStep.vue';
import WorkflowProgress from './WorkflowProgress.vue';
import type { WorkflowStep as WorkflowStepType, WorkflowAction } from '@/types/workflow';

export default defineComponent({
  name: 'WorkflowProgressView',
  components: {
    WorkflowStep,
    WorkflowProgress,
  },
  props: {
    steps: {
      type: Array as PropType<WorkflowStepType[]>,
      required: true,
    },
    title: {
      type: String,
      default: undefined,
    },
    showProgress: {
      type: Boolean,
      default: true,
    },
    showSteps: {
      type: Boolean,
      default: true,
    },
    showDots: {
      type: Boolean,
      default: false,
    },
    showMetadata: {
      type: Boolean,
      default: false,
    },
    allowNavigation: {
      type: Boolean,
      default: false,
    },
    maxVisibleSteps: {
      type: Number,
      default: 5,
    },
  },
  emits: {
    stepClick: (step: WorkflowStepType) => true,
    stepAction: (action: WorkflowAction, step: WorkflowStepType) => true,
  },
  setup(props, { emit }) {
    const isExpanded = ref(false);

    // 已完成步骤数
    const completedCount = computed(() => {
      return props.steps.filter(s => s.status === 'completed').length;
    });

    // 当前激活步骤索引
    const currentIndex = computed(() => {
      return props.steps.findIndex(s => s.status === 'active');
    });

    // 是否有更多步骤
    const hasMoreSteps = computed(() => {
      return props.steps.length > props.maxVisibleSteps;
    });

    // 可见步骤
    const visibleSteps = computed(() => {
      if (isExpanded.value || !hasMoreSteps.value) {
        return props.steps;
      }
      return props.steps.slice(0, props.maxVisibleSteps);
    });

    const toggleExpanded = () => {
      isExpanded.value = !isExpanded.value;
    };

    const handleStepClick = (step: WorkflowStepType) => {
      emit('stepClick', step);
    };

    const handleStepAction = (action: WorkflowAction, step: WorkflowStepType) => {
      emit('stepAction', action, step);
    };

    return {
      isExpanded,
      completedCount,
      currentIndex,
      hasMoreSteps,
      visibleSteps,
      toggleExpanded,
      handleStepClick,
      handleStepAction,
    };
  },
});
</script>

<style scoped>
.workflow-progress-view {
  @apply bg-slate-50 dark:bg-slate-800 border border-slate-200 dark:border-slate-700 rounded-lg p-4;
}

.view-header {
  @apply mb-4 space-y-3;
}

.view-title {
  @apply text-base font-semibold text-slate-800 dark:text-slate-200;
}

.view-steps {
  @apply space-y-0;
}

.toggle-button {
  @apply flex items-center justify-center gap-2 w-full mt-4 px-4 py-2 text-sm font-medium text-slate-600 dark:text-slate-400 bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 rounded-lg hover:bg-slate-50 dark:hover:bg-slate-700 transition-colors;
}

.view-empty {
  @apply flex items-center justify-center py-8;
}
</style>
