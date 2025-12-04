<template>
  <div class="plan-tool">
    <!-- 头部工具栏 -->
    <div class="plan-header">
      <div class="header-title">
        <Icon type="list" size="sm" />
        <span>计划管理</span>
      </div>
      <div class="header-actions">
        <button class="action-button" title="创建计划" @click="showCreateForm = !showCreateForm">
          <Icon type="plus" size="sm" />
        </button>
        <button class="action-button" title="刷新计划" @click="refreshPlans">
          <Icon type="refresh" size="sm" />
        </button>
        <button class="action-button" title="切换视图" @click="toggleViewMode">
          <Icon :type="viewMode === 'list' ? 'grid' : 'list'" size="sm" />
        </button>
      </div>
    </div>

    <!-- 创建计划表单 -->
    <div v-if="showCreateForm" class="create-form">
      <div class="form-content">
        <input v-model="newPlan.title" ref="titleInput" type="text" placeholder="计划标题..." class="plan-input" @keydown.enter="createPlan" @keydown.esc="showCreateForm = false" />
        <textarea v-model="newPlan.description" placeholder="计划描述..." class="plan-textarea" rows="3"></textarea>
        <div class="form-actions">
          <select v-model="newPlan.type" class="type-select">
            <option value="">计划类型</option>
            <option value="project">项目计划</option>
            <option value="development">开发计划</option>
            <option value="research">研究计划</option>
            <option value="deployment">部署计划</option>
            <option value="maintenance">维护计划</option>
          </select>
          <select v-model="newPlan.priority" class="priority-select">
            <option value="">优先级</option>
            <option value="low">低</option>
            <option value="medium">中</option>
            <option value="high">高</option>
          </select>
          <input v-model="newPlan.dueDate" type="date" class="date-input" title="截止日期" />
          <button class="create-button" :disabled="!newPlan.title.trim()" @click="createPlan">创建</button>
          <button
            class="cancel-button"
            @click="
              showCreateForm = false;
              newPlan.title = '';
              newPlan.description = '';
            "
          >
            取消
          </button>
        </div>
      </div>
    </div>

    <!-- 过滤器 -->
    <div class="plan-filters">
      <button v-for="filter in filters" :key="filter.key" :class="['filter-button', { active: currentFilter === filter.key }]" @click="currentFilter = filter.key">
        <Icon :type="filter.icon as any" size="sm" />
        {{ filter.label }}
        <span class="filter-count">{{ filter.count }}</span>
      </button>
    </div>

    <!-- 计划列表 -->
    <div :class="['plan-container', { 'plan-grid': viewMode === 'grid' }]">
      <div
        v-for="plan in filteredPlans"
        :key="plan.id"
        :class="[
          'plan-item',
          {
            'plan-priority-high': plan.priority === 'high',
            'plan-priority-medium': plan.priority === 'medium',
            'plan-priority-low': plan.priority === 'low',
          },
        ]"
      >
        <!-- 计划头部 -->
        <div class="plan-header-item">
          <div class="plan-title-section">
            <h3 class="plan-title">{{ plan.title }}</h3>
            <div class="plan-meta">
              <span v-if="plan.type" :class="`type-badge type-${plan.type}`">
                {{ getTypeText(plan.type) }}
              </span>
              <span v-if="plan.priority" :class="`priority-badge priority-${plan.priority}`">
                {{ getPriorityText(plan.priority) }}
              </span>
              <span :class="`status-badge status-${plan.status}`">
                {{ getStatusText(plan.status) }}
              </span>
            </div>
          </div>

          <div class="plan-actions">
            <button class="action-btn edit-btn" title="编辑计划" @click="editPlan(plan)">
              <Icon type="edit" size="xs" />
            </button>
            <button v-if="plan.status === 'draft'" class="action-btn approve-btn" title="批准计划" @click="approvePlan(plan.id)">
              <Icon type="check" size="xs" />
            </button>
            <button class="action-btn delete-btn" title="删除计划" @click="deletePlan(plan.id)">
              <Icon type="trash" size="xs" />
            </button>
          </div>
        </div>

        <!-- 计划内容 -->
        <div class="plan-content">
          <p v-if="plan.description" class="plan-description">{{ plan.description }}</p>

          <!-- 计划步骤 -->
          <div v-if="plan.steps && plan.steps.length > 0" class="plan-steps">
            <h4>执行步骤</h4>
            <div class="steps-list">
              <div
                v-for="(step, index) in plan.steps"
                :key="index"
                :class="[
                  'step-item',
                  {
                    'step-completed': step.completed,
                    'step-current': index === getCurrentStep(plan),
                  },
                ]"
              >
                <div class="step-number">{{ index + 1 }}</div>
                <div class="step-content">
                  <div class="step-title">{{ step.title }}</div>
                  <div v-if="step.description" class="step-description">{{ step.description }}</div>
                </div>
                <div class="step-status">
                  <Icon v-if="step.completed" type="check" size="sm" class="text-green-500" />
                  <Icon v-else type="clock" size="sm" class="text-gray-400" />
                </div>
              </div>
            </div>
          </div>

          <!-- 进度条 -->
          <div class="plan-progress">
            <div class="progress-info">
              <span>进度: {{ getProgress(plan) }}%</span>
              <span>{{ getCompletedSteps(plan) }}/{{ plan.steps?.length || 0 }} 步骤</span>
            </div>
            <div class="progress-bar">
              <div class="progress-fill" :style="{ width: getProgress(plan) + '%' }"></div>
            </div>
          </div>
        </div>

        <!-- 计划底部信息 -->
        <div class="plan-footer">
          <div class="plan-dates">
            <span v-if="plan.createdAt" class="created-date"> 创建于 {{ formatDate(plan.createdAt) }} </span>
            <span v-if="plan.dueDate" class="due-date" :class="{ overdue: isOverdue(plan.dueDate) }"> 截止 {{ formatDate(plan.dueDate) }} </span>
          </div>
          <div class="plan-tags" v-if="plan.tags && plan.tags.length > 0">
            <span v-for="tag in plan.tags" :key="tag" class="tag">{{ tag }}</span>
          </div>
        </div>
      </div>

      <!-- 空状态 -->
      <div v-if="filteredPlans.length === 0" class="empty-state">
        <Icon type="inbox" size="lg" />
        <p>{{ getEmptyMessage() }}</p>
      </div>
    </div>

    <!-- 统计信息 -->
    <div class="plan-stats">
      <div class="stat-item">
        <span class="stat-value">{{ plans.length }}</span>
        <span class="stat-label">总计</span>
      </div>
      <div class="stat-item">
        <span class="stat-value">{{ draftCount }}</span>
        <span class="stat-label">草稿</span>
      </div>
      <div class="stat-item">
        <span class="stat-value">{{ approvedCount }}</span>
        <span class="stat-label">已批准</span>
      </div>
      <div class="stat-item">
        <span class="stat-value">{{ completedCount }}</span>
        <span class="stat-label">已完成</span>
      </div>
    </div>

    <!-- 计划详情模态框 -->
    <div v-if="showDetailModal" class="modal-overlay" @click.self="showDetailModal = false">
      <div class="modal-content">
        <div class="modal-header">
          <h3>计划详情</h3>
          <button @click="showDetailModal = false">
            <Icon type="close" size="sm" />
          </button>
        </div>
        <div class="modal-body">
          <div v-if="selectedPlan" class="plan-detail-content">
            <div class="detail-header">
              <h2>{{ selectedPlan.title }}</h2>
              <div class="detail-meta">
                <span :class="`type-badge type-${selectedPlan.type}`">
                  {{ getTypeText(selectedPlan.type) }}
                </span>
                <span :class="`priority-badge priority-${selectedPlan.priority}`">
                  {{ getPriorityText(selectedPlan.priority) }}
                </span>
                <span :class="`status-badge status-${selectedPlan.status}`">
                  {{ getStatusText(selectedPlan.status) }}
                </span>
              </div>
            </div>

            <div v-if="selectedPlan.description" class="detail-description">
              <h4>描述</h4>
              <p>{{ selectedPlan.description }}</p>
            </div>

            <div v-if="selectedPlan.steps && selectedPlan.steps.length > 0" class="detail-steps">
              <h4>执行步骤</h4>
              <div class="detail-steps-list">
                <div
                  v-for="(step, index) in selectedPlan.steps"
                  :key="index"
                  :class="[
                    'detail-step-item',
                    {
                      'step-completed': step.completed,
                      'step-current': index === getCurrentStep(selectedPlan),
                    },
                  ]"
                >
                  <div class="step-header">
                    <div class="step-number">{{ index + 1 }}</div>
                    <div class="step-info">
                      <div class="step-title">{{ step.title }}</div>
                      <div v-if="step.description" class="step-description">{{ step.description }}</div>
                    </div>
                    <div class="step-actions">
                      <button v-if="!step.completed" class="step-complete-btn" @click="completeStep(selectedPlan.id, index)">
                        <Icon type="check" size="xs" />
                      </button>
                      <Icon v-else type="check" size="sm" class="text-green-500" />
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, nextTick, onMounted } from "vue";
import Icon from "../ChatUI/Icon.vue";

interface PlanStep {
  title: string;
  description?: string;
  completed: boolean;
  completedAt?: number;
}

interface Plan {
  id: string;
  title: string;
  description?: string;
  type: "project" | "development" | "research" | "deployment" | "maintenance" | "";
  priority: "low" | "medium" | "high" | "";
  status: "draft" | "pending" | "approved" | "in_progress" | "completed" | "cancelled";
  steps: PlanStep[];
  tags?: string[];
  createdAt: number;
  updatedAt: number;
  dueDate?: string;
  approvedAt?: number;
  completedAt?: number;
}

interface Props {
  wsUrl?: string;
  sessionId?: string;
}

const props = withDefaults(defineProps<Props>(), {
  wsUrl: "ws://localhost:8080/ws",
  sessionId: "default",
});

const emit = defineEmits<{
  planCreated: [plan: Plan];
  planUpdated: [plan: Plan];
  planApproved: [planId: string];
  planCompleted: [planId: string];
}>();

// 响应式数据
const plans = ref<Plan[]>([]);
const showCreateForm = ref(false);
const showDetailModal = ref(false);
const selectedPlan = ref<Plan | null>(null);
const currentFilter = ref("all");
const viewMode = ref<"list" | "grid">("list");
const titleInput = ref<HTMLInputElement>();
const websocket = ref<WebSocket | null>(null);

const newPlan = ref({
  title: "",
  description: "",
  type: "" as "project" | "development" | "research" | "deployment" | "maintenance" | "",
  priority: "" as "low" | "medium" | "high" | "",
  dueDate: "",
});

// 过滤器选项
const filters = computed(() => [
  { key: "all", label: "全部", icon: "list", count: plans.value.length },
  { key: "draft", label: "草稿", icon: "edit", count: plans.value.filter((p) => p.status === "draft").length },
  { key: "approved", label: "已批准", icon: "check", count: plans.value.filter((p) => p.status === "approved").length },
  { key: "in_progress", label: "进行中", icon: "play", count: plans.value.filter((p) => p.status === "in_progress").length },
  { key: "completed", label: "已完成", icon: "check-circle", count: plans.value.filter((p) => p.status === "completed").length },
]);

// 计算属性
const filteredPlans = computed(() => {
  let filtered = plans.value;

  switch (currentFilter.value) {
    case "draft":
      filtered = filtered.filter((p) => p.status === "draft");
      break;
    case "approved":
      filtered = filtered.filter((p) => p.status === "approved");
      break;
    case "in_progress":
      filtered = filtered.filter((p) => p.status === "in_progress");
      break;
    case "completed":
      filtered = filtered.filter((p) => p.status === "completed");
      break;
  }

  return filtered.sort((a, b) => {
    // 优先级排序
    const priorityOrder = { high: 0, medium: 1, low: 2, "": 3 };
    if (priorityOrder[a.priority] !== priorityOrder[b.priority]) {
      return priorityOrder[a.priority] - priorityOrder[b.priority];
    }

    // 时间排序（最新的在前）
    return b.updatedAt - a.updatedAt;
  });
});

const draftCount = computed(() => plans.value.filter((p) => p.status === "draft").length);
const approvedCount = computed(() => plans.value.filter((p) => p.status === "approved").length);
const completedCount = computed(() => plans.value.filter((p) => p.status === "completed").length);

// WebSocket 连接
const connectWebSocket = () => {
  try {
    websocket.value = new WebSocket(`${props.wsUrl}?session=${props.sessionId}`);

    websocket.value.onopen = () => {
      console.log("PlanTool WebSocket connected");
      requestPlans();
    };

    websocket.value.onmessage = (event) => {
      try {
        const message = JSON.parse(event.data);
        handleWebSocketMessage(message);
      } catch (error) {
        console.error("Failed to parse WebSocket message:", error);
      }
    };

    websocket.value.onclose = () => {
      console.log("PlanTool WebSocket disconnected");
      setTimeout(connectWebSocket, 5000);
    };

    websocket.value.onerror = (error) => {
      console.error("PlanTool WebSocket error:", error);
    };
  } catch (error) {
    console.error("Failed to connect WebSocket:", error);
  }
};

const handleWebSocketMessage = (message: any) => {
  switch (message.type) {
    case "plan_list_response":
      plans.value = message.plans || [];
      break;
    case "plan_created":
      const createdPlan = message.plan;
      const existingIndex = plans.value.findIndex((p) => p.id === createdPlan.id);
      if (existingIndex === -1) {
        plans.value.push(createdPlan);
      }
      emit("planCreated", createdPlan);
      break;
    case "plan_updated":
      const updatedPlan = message.plan;
      const index = plans.value.findIndex((p) => p.id === updatedPlan.id);
      if (index !== -1) {
        plans.value[index] = updatedPlan;
      }
      emit("planUpdated", updatedPlan);
      break;
    case "plan_approved": {
      const approvedPlanId = message.plan_id;
      const approvedIndex = plans.value.findIndex((p) => p.id === approvedPlanId);
      const approvedPlan = plans.value[approvedIndex];
      if (approvedIndex !== -1 && approvedPlan) {
        approvedPlan.status = "approved";
        approvedPlan.approvedAt = Date.now();
      }
      emit("planApproved", approvedPlanId);
      break;
    }
    case "plan_completed": {
      const completedPlanId = message.plan_id;
      const completedIndex = plans.value.findIndex((p) => p.id === completedPlanId);
      const completedPlan = plans.value[completedIndex];
      if (completedIndex !== -1 && completedPlan) {
        completedPlan.status = "completed";
        completedPlan.completedAt = Date.now();
      }
      emit("planCompleted", completedPlanId);
      break;
    }
    case "plan_deleted":
      const deletedId = message.id;
      plans.value = plans.value.filter((p) => p.id !== deletedId);
      break;
  }
};

const sendWebSocketMessage = (message: any) => {
  if (websocket.value && websocket.value.readyState === WebSocket.OPEN) {
    websocket.value.send(JSON.stringify(message));
  }
};

// 计划操作方法
const requestPlans = () => {
  sendWebSocketMessage({ type: "plan_list_request" });
};

const createPlan = () => {
  if (!newPlan.value.title.trim()) return;

  const plan: Partial<Plan> = {
    title: newPlan.value.title.trim(),
    description: newPlan.value.description.trim(),
    type: newPlan.value.type,
    priority: newPlan.value.priority,
    status: "draft",
    steps: [],
  };

  if (newPlan.value.dueDate) {
    plan.dueDate = newPlan.value.dueDate;
  }

  sendWebSocketMessage({
    type: "plan_create",
    plan,
  });

  // 重置表单
  newPlan.value = { title: "", description: "", type: "", priority: "", dueDate: "" };
  showCreateForm.value = false;

  nextTick(() => {
    titleInput.value?.focus();
  });
};

const editPlan = (plan: Plan) => {
  selectedPlan.value = plan;
  showDetailModal.value = true;
};

const approvePlan = (planId: string) => {
  if (confirm("确定要批准这个计划吗？")) {
    sendWebSocketMessage({
      type: "plan_approve",
      id: planId,
    });
  }
};

const deletePlan = (planId: string) => {
  if (confirm("确定要删除这个计划吗？")) {
    sendWebSocketMessage({
      type: "plan_delete",
      id: planId,
    });
  }
};

const refreshPlans = () => {
  requestPlans();
};

const toggleViewMode = () => {
  viewMode.value = viewMode.value === "list" ? "grid" : "list";
};

const completeStep = (planId: string, stepIndex: number) => {
  sendWebSocketMessage({
    type: "plan_step_complete",
    plan_id: planId,
    step_index: stepIndex,
  });
};

// 工具方法
const getTypeText = (type: string) => {
  const map = {
    project: "项目计划",
    development: "开发计划",
    research: "研究计划",
    deployment: "部署计划",
    maintenance: "维护计划",
  };
  return map[type as keyof typeof map] || type;
};

const getPriorityText = (priority: string) => {
  const map = { high: "高", medium: "中", low: "低" };
  return map[priority as keyof typeof map] || priority;
};

const getStatusText = (status: string) => {
  const map = {
    draft: "草稿",
    pending: "待批准",
    approved: "已批准",
    in_progress: "进行中",
    completed: "已完成",
    cancelled: "已取消",
  };
  return map[status as keyof typeof map] || status;
};

const formatDate = (timestamp: number | string) => {
  const date = new Date(timestamp);
  return date.toLocaleDateString("zh-CN", { month: "short", day: "numeric" });
};

const formatDateTime = (timestamp: number) => {
  const date = new Date(timestamp);
  return date.toLocaleDateString("zh-CN", {
    month: "short",
    day: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  });
};

const isOverdue = (dueDate: string) => {
  return new Date(dueDate) < new Date(new Date().toDateString());
};

const getProgress = (plan: Plan) => {
  if (!plan.steps || plan.steps.length === 0) return 0;
  const completedSteps = plan.steps.filter((step) => step.completed).length;
  return Math.round((completedSteps / plan.steps.length) * 100);
};

const getCompletedSteps = (plan: Plan) => {
  if (!plan.steps) return 0;
  return plan.steps.filter((step) => step.completed).length;
};

const getCurrentStep = (plan: Plan) => {
  if (!plan.steps) return -1;
  for (let i = 0; i < plan.steps.length; i++) {
    const step = plan.steps[i];
    if (step && !step.completed) {
      return i;
    }
  }
  return -1;
};

const getEmptyMessage = () => {
  const messages = {
    all: "暂无计划，点击 + 创建第一个计划",
    draft: "暂无草稿计划",
    approved: "暂无已批准的计划",
    in_progress: "暂无进行中的计划",
    completed: "暂无已完成的计划",
  };
  return messages[currentFilter.value as keyof typeof messages];
};

// 生命周期
onMounted(() => {
  connectWebSocket();

  // 自动聚焦到标题输入框
  watch(showCreateForm, (show) => {
    if (show) {
      nextTick(() => {
        titleInput.value?.focus();
      });
    }
  });
});
</script>

<style scoped>
.plan-tool {
  @apply flex flex-col h-full bg-surface dark:bg-surface-dark border border-border dark:border-border-dark rounded-lg;
  max-height: 600px;
}

.plan-header {
  @apply flex items-center justify-between px-4 py-3 border-b border-border dark:border-border-dark bg-surface dark:bg-surface-dark;
}

.header-title {
  @apply flex items-center gap-2 font-semibold text-text dark:text-text-dark;
}

.header-actions {
  @apply flex gap-1;
}

.action-button {
  @apply p-1.5 text-gray-500 dark:text-gray-400 hover:text-gray-700 dark:hover:text-gray-200 hover:bg-gray-100 dark:hover:bg-gray-700 rounded transition-colors;
}

.create-form {
  @apply px-4 py-3 bg-blue-50 dark:bg-blue-900/20 border-b border-border dark:border-border-dark;
}

.form-content {
  @apply space-y-2;
}

.plan-input {
  @apply w-full px-3 py-2 border border-border dark:border-border-dark rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500 dark:bg-gray-800 dark:text-white;
}

.plan-textarea {
  @apply w-full px-3 py-2 border border-border dark:border-border-dark rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500 dark:bg-gray-800 dark:text-white resize-none;
}

.form-actions {
  @apply flex gap-2 items-center;
}

.type-select,
.priority-select,
.date-input {
  @apply px-2 py-1 text-sm border border-border dark:border-border-dark rounded focus:outline-none focus:ring-2 focus:ring-blue-500 dark:bg-gray-800 dark:text-white;
}

.create-button {
  @apply px-3 py-1 bg-blue-500 hover:bg-blue-600 text-white text-sm rounded transition-colors disabled:opacity-50 disabled:cursor-not-allowed;
}

.cancel-button {
  @apply px-3 py-1 text-gray-500 dark:text-gray-400 hover:text-gray-700 dark:hover:text-gray-200 text-sm rounded transition-colors;
}

.plan-filters {
  @apply flex gap-1 px-4 py-2 border-b border-border dark:border-border-dark bg-surface dark:bg-surface-dark;
}

.filter-button {
  @apply flex items-center gap-2 px-3 py-1.5 text-sm text-gray-600 dark:text-gray-400 hover:text-gray-900 dark:hover:text-white hover:bg-gray-100 dark:hover:bg-gray-700 rounded transition-colors;
}

.filter-button.active {
  @apply bg-blue-100 dark:bg-blue-900/30 text-blue-700 dark:text-blue-300;
}

.filter-count {
  @apply text-xs bg-gray-200 dark:bg-gray-600 px-1.5 py-0.5 rounded-full;
}

.plan-container {
  @apply flex-1 overflow-y-auto px-4 py-2 space-y-3;
}

.plan-container.plan-grid {
  @apply grid grid-cols-1 md:grid-cols-2 gap-4;
}

.plan-item {
  @apply border border-border dark:border-border-dark rounded-lg bg-surface dark:bg-surface-dark hover:bg-gray-50 dark:hover:bg-gray-700/50 transition-colors;
}

.plan-priority-high {
  @apply border-l-4 border-l-red-500;
}

.plan-priority-medium {
  @apply border-l-4 border-l-yellow-500;
}

.plan-priority-low {
  @apply border-l-4 border-l-green-500;
}

.plan-header-item {
  @apply flex items-center justify-between p-4 pb-2;
}

.plan-title-section {
  @apply flex-1 min-w-0;
}

.plan-title {
  @apply text-lg font-semibold text-text dark:text-text-dark truncate mb-2;
}

.plan-meta {
  @apply flex items-center gap-2 flex-wrap;
}

.type-badge {
  @apply text-xs px-2 py-1 rounded-full font-medium;
}

.type-project {
  @apply bg-purple-100 dark:bg-purple-900/30 text-purple-700 dark:text-purple-300;
}

.type-development {
  @apply bg-blue-100 dark:bg-blue-900/30 text-blue-700 dark:text-blue-300;
}

.type-research {
  @apply bg-green-100 dark:bg-green-900/30 text-green-700 dark:text-green-300;
}

.type-deployment {
  @apply bg-orange-100 dark:bg-orange-900/30 text-orange-700 dark:text-orange-300;
}

.type-maintenance {
  @apply bg-gray-100 dark:bg-gray-700 text-gray-700 dark:text-gray-300;
}

.priority-badge {
  @apply text-xs px-2 py-1 rounded-full font-medium;
}

.priority-high {
  @apply bg-red-100 dark:bg-red-900/30 text-red-700 dark:text-red-300;
}

.priority-medium {
  @apply bg-yellow-100 dark:bg-yellow-900/30 text-yellow-700 dark:text-yellow-300;
}

.priority-low {
  @apply bg-green-100 dark:bg-green-900/30 text-green-700 dark:text-green-300;
}

.status-badge {
  @apply text-xs px-2 py-1 rounded-full font-medium;
}

.status-draft {
  @apply bg-gray-100 dark:bg-gray-700 text-gray-700 dark:text-gray-300;
}

.status-pending {
  @apply bg-yellow-100 dark:bg-yellow-900/30 text-yellow-700 dark:text-yellow-300;
}

.status-approved {
  @apply bg-blue-100 dark:bg-blue-900/30 text-blue-700 dark:text-blue-300;
}

.status-in_progress {
  @apply bg-purple-100 dark:bg-purple-900/30 text-purple-700 dark:text-purple-300;
}

.status-completed {
  @apply bg-green-100 dark:bg-green-900/30 text-green-700 dark:text-green-300;
}

.status-cancelled {
  @apply bg-red-100 dark:bg-red-900/30 text-red-700 dark:text-red-300;
}

.plan-actions {
  @apply flex gap-1 opacity-0 hover:opacity-100 transition-opacity;
}

.plan-item:hover .plan-actions {
  @apply opacity-100;
}

.action-btn {
  @apply p-1 text-gray-400 dark:text-gray-500 hover:text-gray-600 dark:hover:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-600 rounded transition-colors;
}

.approve-btn:hover {
  @apply text-green-500 hover:bg-green-50 dark:hover:bg-green-900/20;
}

.delete-btn:hover {
  @apply text-red-500 hover:bg-red-50 dark:hover:bg-red-900/20;
}

.plan-content {
  @apply px-4 pb-2;
}

.plan-description {
  @apply text-sm text-gray-600 dark:text-gray-400 mb-3;
}

.plan-steps {
  @apply mb-3;
}

.plan-steps h4 {
  @apply text-sm font-semibold text-text dark:text-text-dark mb-2;
}

.steps-list {
  @apply space-y-1;
}

.step-item {
  @apply flex items-center gap-2 p-2 rounded border border-gray-200 dark:border-gray-600;
}

.step-current {
  @apply bg-blue-50 dark:bg-blue-900/20 border-blue-200 dark:border-blue-700;
}

.step-completed {
  @apply bg-green-50 dark:bg-green-900/20 border-green-200 dark:border-green-700;
}

.step-number {
  @apply w-6 h-6 rounded-full bg-gray-200 dark:bg-gray-600 flex items-center justify-center text-xs font-medium text-gray-600 dark:text-gray-300 flex-shrink-0;
}

.step-current .step-number {
  @apply bg-blue-500 text-white;
}

.step-completed .step-number {
  @apply bg-green-500 text-white;
}

.step-content {
  @apply flex-1 min-w-0;
}

.step-title {
  @apply text-sm font-medium text-text dark:text-text-dark;
}

.step-description {
  @apply text-xs text-gray-500 dark:text-gray-400;
}

.plan-progress {
  @apply mb-3;
}

.progress-info {
  @apply flex justify-between items-center mb-1 text-xs text-gray-600 dark:text-gray-400;
}

.progress-bar {
  @apply w-full bg-gray-200 dark:bg-gray-600 rounded-full h-2;
}

.progress-fill {
  @apply bg-blue-500 h-2 rounded-full transition-all duration-300;
}

.plan-footer {
  @apply px-4 py-3 border-t border-border dark:border-border-dark flex items-center justify-between;
}

.plan-dates {
  @apply flex items-center gap-3 text-xs text-gray-500 dark:text-gray-400;
}

.overdue {
  @apply text-red-500 dark:text-red-400;
}

.plan-tags {
  @apply flex gap-1;
}

.tag {
  @apply text-xs px-2 py-1 bg-gray-100 dark:bg-gray-700 text-gray-600 dark:text-gray-300 rounded;
}

.empty-state {
  @apply flex flex-col items-center justify-center py-8 text-gray-400 dark:text-gray-500;
}

.plan-stats {
  @apply flex items-center justify-around px-4 py-3 border-t border-border dark:border-border-dark bg-surface dark:bg-surface-dark;
}

.stat-item {
  @apply text-center;
}

.stat-value {
  @apply block text-lg font-semibold text-text dark:text-text-dark;
}

.stat-label {
  @apply block text-xs text-gray-500 dark:text-gray-400;
}

/* 模态框样式 */
.modal-overlay {
  @apply fixed inset-0 bg-black/50 flex items-center justify-center z-50;
}

.modal-content {
  @apply bg-white dark:bg-gray-800 rounded-lg shadow-xl w-full max-w-4xl max-h-[80vh] flex flex-col;
}

.modal-header {
  @apply flex items-center justify-between p-4 border-b border-gray-200 dark:border-gray-700;
}

.modal-header h3,
.modal-header h2 {
  @apply text-lg font-semibold text-gray-900 dark:text-white;
}

.modal-body {
  @apply flex-1 overflow-y-auto p-4;
}

.plan-detail-content {
  @apply space-y-4;
}

.detail-header {
  @apply pb-4 border-b border-gray-200 dark:border-gray-700;
}

.detail-header h2 {
  @apply text-xl font-bold text-gray-900 dark:text-white mb-2;
}

.detail-meta {
  @apply flex items-center gap-2 flex-wrap;
}

.detail-description {
  @apply space-y-2;
}

.detail-description h4,
.detail-steps h4 {
  @apply text-sm font-semibold text-gray-900 dark:text-white;
}

.detail-steps-list {
  @apply space-y-2;
}

.detail-step-item {
  @apply p-3 border border-gray-200 dark:border-gray-600 rounded;
}

.step-header {
  @apply flex items-center gap-3;
}

.step-info {
  @apply flex-1;
}

.step-actions {
  @apply flex gap-2;
}

.step-complete-btn {
  @apply p-1 text-gray-400 hover:text-green-500 hover:bg-green-50 dark:hover:bg-green-900/20 rounded transition-colors;
}
</style>
