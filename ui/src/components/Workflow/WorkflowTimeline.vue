<template>
  <div class="workflow-timeline">
    <!-- 头部 -->
    <div class="timeline-header">
      <div class="header-content">
        <div class="header-label">PROJECT</div>
        <div class="header-title">{{ title }}</div>
      </div>
      <button v-if="showBack" class="back-button" @click="$emit('back')">
        <Icon type="arrow-left" size="sm" />
      </button>
    </div>

    <!-- 时间线 -->
    <div class="timeline-content" ref="timelineRef">
      <!-- 连接线 -->
      <div class="timeline-line"></div>

      <!-- 步骤列表 -->
      <div class="timeline-steps">
        <div
          v-for="(step, idx) in steps"
          :key="step.id"
          :data-active="idx === currentStep"
          :class="[
            'timeline-step',
            { 'step-active': idx === currentStep },
            { 'step-completed': idx < currentStep },
            { 'step-pending': idx > currentStep }
          ]"
        >
          <!-- 步骤节点 -->
          <button
            class="step-node"
            @click="handleStepClick(idx)"
          >
            <Icon
              v-if="idx < currentStep"
              type="check"
              size="sm"
            />
            <Icon
              v-else-if="idx === currentStep"
              type="disc"
              size="sm"
              class="animate-pulse"
            />
            <span v-else class="step-number">{{ idx + 1 }}</span>
          </button>

          <!-- 步骤内容 -->
          <div class="step-content">
            <!-- 激活状态 -->
            <div v-if="idx === currentStep" class="step-card step-card-active">
              <div class="step-card-header">
                <div>
                  <h3 class="step-name">{{ step.name }}</h3>
                  <p class="step-label">当前阶段</p>
                </div>
                <span class="step-icon">{{ step.icon }}</span>
              </div>
              
              <p class="step-description">{{ step.description }}</p>

              <!-- 快捷操作 -->
              <div v-if="step.actions && step.actions.length > 0" class="step-actions">
                <button
                  v-for="action in step.actions"
                  :key="action.id"
                  :class="['action-button', action.variant || 'primary']"
                  @click="$emit('action', action)"
                >
                  <Icon v-if="action.icon" :type="(action.icon as any)" size="sm" />
                  {{ action.label }}
                </button>
              </div>

              <!-- 下一步按钮 -->
              <button
                v-if="idx < steps.length - 1"
                class="next-button"
                @click="handleStepClick(idx + 1)"
              >
                下一步
                <Icon type="arrow-right" size="sm" />
              </button>
            </div>

            <!-- 非激活状态 -->
            <button
              v-else
              class="step-title"
              @click="handleStepClick(idx)"
            >
              <span :class="{ 'step-title-completed': idx < currentStep }">
                {{ step.name }}
              </span>
            </button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch, nextTick } from 'vue';
import Icon from '../ChatUI/Icon.vue';

interface StepAction {
  id: string;
  label: string;
  icon?: string;
  variant?: 'primary' | 'secondary';
}

interface WorkflowStep {
  id: string;
  name: string;
  icon: string;
  description: string;
  actions?: StepAction[];
}

interface Props {
  steps: WorkflowStep[];
  currentStep: number;
  title?: string;
  showBack?: boolean;
}

const props = withDefaults(defineProps<Props>(), {
  title: '工作流',
  showBack: false,
});

const emit = defineEmits<{
  'step-change': [step: number];
  action: [action: StepAction];
  back: [];
}>();

const timelineRef = ref<HTMLDivElement>();

const handleStepClick = (idx: number) => {
  emit('step-change', idx);
};

// 自动滚动到当前步骤
watch(() => props.currentStep, async () => {
  await nextTick();
  if (timelineRef.value) {
    const activeEl = timelineRef.value.querySelector('[data-active="true"]');
    if (activeEl) {
      activeEl.scrollIntoView({ behavior: 'smooth', block: 'center' });
    }
  }
});
</script>

<style scoped>
.workflow-timeline {
  @apply w-80 bg-white dark:bg-gray-800 border-r border-gray-200 dark:border-gray-700 flex flex-col flex-shrink-0;
}

.timeline-header {
  @apply p-5 border-b border-gray-200 dark:border-gray-700 flex items-center gap-3 hover:bg-gray-50 dark:hover:bg-gray-700/50 transition-colors cursor-pointer;
}

.header-content {
  @apply flex-1 overflow-hidden;
}

.header-label {
  @apply text-xs font-bold text-gray-400 dark:text-gray-500 uppercase tracking-wider;
}

.header-title {
  @apply font-serif font-bold text-gray-900 dark:text-white truncate leading-tight;
}

.back-button {
  @apply p-2 rounded-lg border border-gray-200 dark:border-gray-700 text-gray-400 dark:text-gray-500 hover:text-gray-900 dark:hover:text-white hover:border-blue-300 dark:hover:border-blue-700 transition-all;
}

.timeline-content {
  @apply flex-1 overflow-y-auto p-5 relative;
}

.timeline-line {
  @apply absolute left-[2.65rem] top-0 bottom-0 w-px bg-gray-200 dark:bg-gray-700;
}

.timeline-steps {
  @apply space-y-6 relative z-10;
}

.timeline-step {
  @apply relative flex gap-4 transition-all duration-500;
}

.step-active {
  @apply scale-100;
}

.step-completed,
.step-pending {
  @apply scale-95 opacity-70 hover:opacity-100;
}

.step-node {
  @apply w-10 h-10 flex-shrink-0 rounded-full flex items-center justify-center border-2 transition-all duration-300 z-10;
}

.step-active .step-node {
  @apply bg-blue-500 dark:bg-blue-600 border-blue-500 dark:border-blue-600 text-white shadow-lg;
}

.step-completed .step-node {
  @apply bg-white dark:bg-gray-800 border-gray-300 dark:border-gray-600 text-gray-300 dark:text-gray-600;
}

.step-pending .step-node {
  @apply bg-white dark:bg-gray-800 border-gray-200 dark:border-gray-700 text-gray-300 dark:text-gray-600 hover:border-blue-300 dark:hover:border-blue-700 hover:text-blue-500 dark:hover:text-blue-400;
}

.step-number {
  @apply text-xs font-semibold;
}

.step-content {
  @apply flex-1 pt-0.5 transition-all duration-300;
}

.step-card {
  @apply bg-white dark:bg-gray-800 rounded-lg border p-4 shadow-sm;
}

.step-card-active {
  @apply border-blue-100 dark:border-blue-900 shadow-lg ring-1 ring-black/5 dark:ring-white/5;
  animation: slideIn 0.3s ease-out;
}

.step-card-header {
  @apply flex justify-between items-start mb-2;
}

.step-name {
  @apply text-sm font-bold text-blue-600 dark:text-blue-400;
}

.step-label {
  @apply text-xs text-blue-600/60 dark:text-blue-400/60 font-medium mt-0.5;
}

.step-icon {
  @apply text-lg grayscale opacity-80;
}

.step-description {
  @apply text-xs text-gray-600 dark:text-gray-400 leading-relaxed mb-4 pb-2 border-t border-gray-100 dark:border-gray-700 pt-2;
}

.step-actions {
  @apply flex flex-col gap-2 mb-2;
}

.action-button {
  @apply flex items-center justify-center gap-2 text-xs font-bold px-3 py-2.5 rounded border shadow-sm transition-all active:scale-95;
}

.action-button.primary {
  @apply bg-blue-500 hover:bg-blue-600 text-white border-blue-500;
}

.action-button.secondary {
  @apply bg-white dark:bg-gray-800 text-gray-900 dark:text-white border-gray-200 dark:border-gray-700 hover:bg-gray-50 dark:hover:bg-gray-700;
}

.next-button {
  @apply flex items-center justify-center gap-2 text-xs font-medium text-gray-600 dark:text-gray-400 bg-gray-50 dark:bg-gray-900 hover:bg-gray-100 dark:hover:bg-gray-800 border border-gray-200 dark:border-gray-700 px-3 py-2 rounded transition-colors mt-1;
}

.step-title {
  @apply text-left transition-transform w-full;
}

.step-item:hover .step-title {
  @apply translate-x-1;
}

.step-title span {
  @apply text-sm font-semibold text-gray-600 dark:text-gray-400;
}

.step-title-completed {
  @apply text-gray-400 dark:text-gray-600 line-through decoration-gray-300 dark:decoration-gray-700;
}

@keyframes slideIn {
  from {
    opacity: 0;
    transform: translateX(-10px);
  }
  to {
    opacity: 1;
    transform: translateX(0);
  }
}
</style>
