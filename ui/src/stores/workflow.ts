/**
 * 工作流状态管理
 *
 * 管理通用工作流的步骤和进度
 */

import { defineStore } from 'pinia';
import { ref, computed } from 'vue';
import type { WorkflowStep, WorkflowConfig } from '@/types/workflow';

export const useWorkflowStore = defineStore('workflow', () => {
  // ==================
  // State
  // ==================

  // 工作流步骤列表
  const steps = ref<WorkflowStep[]>([]);

  // 当前步骤索引
  const currentStepIndex = ref(0);

  // 工作流 ID
  const workflowId = ref<string | null>(null);

  // ==================
  // Getters
  // ==================

  // 当前步骤
  const currentStep = computed(() => steps.value[currentStepIndex.value] || null);

  // 总步骤数
  const totalSteps = computed(() => steps.value.length);

  // 已完成步骤数
  const completedSteps = computed(() =>
    steps.value.filter(s => s.status === 'completed').length
  );

  // 进度（0-1）
  const progress = computed(() => {
    if (totalSteps.value === 0) return 0;
    return completedSteps.value / totalSteps.value;
  });

  // 是否有激活的工作流
  const hasActiveWorkflow = computed(() => workflowId.value !== null);

  // 是否是第一步
  const isFirstStep = computed(() => currentStepIndex.value === 0);

  // 是否是最后一步
  const isLastStep = computed(() =>
    currentStepIndex.value === totalSteps.value - 1
  );

  // 是否已完成所有步骤
  const isCompleted = computed(() =>
    steps.value.length > 0 && steps.value.every(s => s.status === 'completed')
  );

  // ==================
  // Actions
  // ==================

  /**
   * 加载工作流配置
   */
  const loadWorkflow = (config: WorkflowConfig) => {
    workflowId.value = config.id;
    steps.value = config.steps.map((step, index) => ({
      ...step,
      status: index === 0 ? 'active' : 'pending',
    }));
    currentStepIndex.value = 0;
  };

  /**
   * 完成当前步骤并激活下一步
   */
  const completeCurrentStep = () => {
    if (currentStep.value) {
      steps.value[currentStepIndex.value].status = 'completed';

      // 激活下一步
      if (!isLastStep.value) {
        currentStepIndex.value++;
        steps.value[currentStepIndex.value].status = 'active';
      }
    }
  };

  /**
   * 完成指定步骤
   */
  const completeStep = (stepId: string) => {
    const index = steps.value.findIndex(s => s.id === stepId);
    if (index !== -1) {
      steps.value[index].status = 'completed';

      // 如果是当前步骤，激活下一步
      if (index === currentStepIndex.value && index < steps.value.length - 1) {
        currentStepIndex.value = index + 1;
        steps.value[currentStepIndex.value].status = 'active';
      }
    }
  };

  /**
   * 标记步骤为失败
   */
  const failStep = (stepId: string) => {
    const index = steps.value.findIndex(s => s.id === stepId);
    if (index !== -1) {
      steps.value[index].status = 'failed';
    }
  };

  /**
   * 跳转到指定步骤
   */
  const goToStep = (stepId: string) => {
    const index = steps.value.findIndex(s => s.id === stepId);
    if (index !== -1) {
      // 将当前激活步骤设为 pending
      if (currentStep.value && currentStep.value.status === 'active') {
        steps.value[currentStepIndex.value].status = 'pending';
      }

      // 激活目标步骤
      currentStepIndex.value = index;
      steps.value[index].status = 'active';
    }
  };

  /**
   * 下一步
   */
  const nextStep = () => {
    if (!isLastStep.value) {
      completeCurrentStep();
    }
  };

  /**
   * 上一步
   */
  const previousStep = () => {
    if (!isFirstStep.value) {
      // 将当前步骤设为 pending
      if (currentStep.value) {
        steps.value[currentStepIndex.value].status = 'pending';
      }

      // 回到上一步
      currentStepIndex.value--;
      steps.value[currentStepIndex.value].status = 'active';
    }
  };

  /**
   * 重置工作流
   */
  const resetWorkflow = () => {
    steps.value.forEach((step, index) => {
      step.status = index === 0 ? 'active' : 'pending';
    });
    currentStepIndex.value = 0;
  };

  /**
   * 清除工作流
   */
  const clearWorkflow = () => {
    steps.value = [];
    currentStepIndex.value = 0;
    workflowId.value = null;
  };

  /**
   * 更新步骤元数据
   */
  const updateStepMetadata = (stepId: string, metadata: Record<string, any>) => {
    const index = steps.value.findIndex(s => s.id === stepId);
    if (index !== -1) {
      steps.value[index].metadata = {
        ...(steps.value[index].metadata || {}),
        ...metadata,
      };
    }
  };

  // ==================
  // Return
  // ==================

  return {
    // State
    steps,
    currentStepIndex,
    workflowId,

    // Getters
    currentStep,
    totalSteps,
    completedSteps,
    progress,
    hasActiveWorkflow,
    isFirstStep,
    isLastStep,
    isCompleted,

    // Actions
    loadWorkflow,
    completeCurrentStep,
    completeStep,
    failStep,
    goToStep,
    nextStep,
    previousStep,
    resetWorkflow,
    clearWorkflow,
    updateStepMetadata,
  };
});
