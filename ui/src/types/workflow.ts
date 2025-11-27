/**
 * 工作流相关类型定义
 *
 * 注意：这些类型用于通用工作流组件 (WorkflowTimeline/WorkflowStep/WorkflowProgress)
 * 与 types/index.ts 中的 Workflow/WorkflowStep 类型不同 - 那些用于专门的工作流执行
 */

export type WorkflowStepStatus = 'pending' | 'active' | 'completed' | 'failed';

export type WorkflowActionType = 'primary' | 'secondary' | 'danger';

export interface WorkflowAction {
  type: WorkflowActionType;
  label: string;
  icon?: string;
}

export interface WorkflowStep {
  id: string;
  title: string;
  description?: string;
  icon?: string;
  status: WorkflowStepStatus;
  actions?: WorkflowAction[];
  metadata?: Record<string, any>;
}

export interface WorkflowConfig {
  id: string;
  name: string;
  steps: Omit<WorkflowStep, 'status'>[];
}

export interface WorkflowState {
  steps: WorkflowStep[];
  currentStepIndex: number;
  workflowId: string | null;
}
