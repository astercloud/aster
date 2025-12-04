<template>
  <div class="agent-dashboard">
    <!-- Header -->
    <div class="dashboard-header">
      <div>
        <h1 class="dashboard-title">Agent 管理</h1>
        <p class="dashboard-subtitle">创建和管理您的 AI Agents</p>
      </div>
      <button @click="showCreateForm = true" class="btn-create">
        <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4"></path>
        </svg>
        创建 Agent
      </button>
    </div>

    <!-- Stats -->
    <div class="stats-grid">
      <div class="stat-card">
        <div class="stat-icon bg-blue-100 dark:bg-blue-900/30 text-blue-600 dark:text-blue-400">
          <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path
              stroke-linecap="round"
              stroke-linejoin="round"
              stroke-width="2"
              d="M9.663 17h4.673M12 3v1m6.364 1.636l-.707.707M21 12h-1M4 12H3m3.343-5.657l-.707-.707m2.828 9.9a5 5 0 117.072 0l-.548.547A3.374 3.374 0 0014 18.469V19a2 2 0 11-4 0v-.531c0-.895-.356-1.754-.988-2.386l-.548-.547z"
            ></path>
          </svg>
        </div>
        <div class="stat-content">
          <div class="stat-value">{{ agents.length }}</div>
          <div class="stat-label">总 Agents</div>
        </div>
      </div>

      <div class="stat-card">
        <div class="stat-icon bg-green-100 dark:bg-green-900/30 text-green-600 dark:text-green-400">
          <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7"></path>
          </svg>
        </div>
        <div class="stat-content">
          <div class="stat-value">{{ activeAgents }}</div>
          <div class="stat-label">活跃中</div>
        </div>
      </div>

      <div class="stat-card">
        <div class="stat-icon bg-amber-100 dark:bg-amber-900/30 text-amber-600 dark:text-amber-400">
          <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 12h.01M12 12h.01M16 12h.01M21 12c0 4.418-4.03 8-9 8a9.863 9.863 0 01-4.255-.949L3 20l1.395-3.72C3.512 15.042 3 13.574 3 12c0-4.418 4.03-8 9-8s9 3.582 9 8z"></path>
          </svg>
        </div>
        <div class="stat-content">
          <div class="stat-value">{{ totalSessions }}</div>
          <div class="stat-label">总会话</div>
        </div>
      </div>
    </div>

    <!-- Agent List -->
    <AgentList :agents="agents" :loading="loading" @create="showCreateForm = true" @chat="handleChat" @edit="handleEdit" @delete="handleDelete" />

    <!-- Create/Edit Modal -->
    <Modal v-if="showCreateForm" @close="showCreateForm = false">
      <template #header>
        {{ editingAgent ? "编辑 Agent" : "创建新 Agent" }}
      </template>
      <template #default>
        <AgentForm :agent="editingAgent ?? undefined" :loading="formLoading" @submit="handleSubmit" @cancel="handleCancel" />
      </template>
    </Modal>

    <!-- Delete Confirmation -->
    <Modal v-if="deletingAgent" @close="deletingAgent = null">
      <template #header>确认删除</template>
      <template #default>
        <div class="delete-confirm">
          <p class="delete-message">
            确定要删除 Agent "<strong>{{ deletingAgent.name }}</strong
            >" 吗？
          </p>
          <p class="delete-warning">此操作无法撤销。</p>
          <div class="delete-actions">
            <button @click="deletingAgent = null" class="btn-secondary">取消</button>
            <button @click="confirmDelete" class="btn-danger">删除</button>
          </div>
        </div>
      </template>
    </Modal>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from "vue";
import AgentList from "./AgentList.vue";
import AgentForm from "./AgentForm.vue";
import Modal from "../Common/Modal.vue";
import { useAsterClient } from "@/composables/useAsterClient";
import type { Agent } from "@/types";

const emit = defineEmits<{
  chat: [agent: Agent];
}>();

const { client } = useAsterClient();

const agents = ref<Agent[]>([]);
const loading = ref(false);
const formLoading = ref(false);
const showCreateForm = ref(false);
const editingAgent = ref<Agent | null>(null);
const deletingAgent = ref<Agent | null>(null);

const activeAgents = computed(() => agents.value.filter((a) => a.status === "idle" || a.status === "thinking").length);

const totalSessions = computed(() => agents.value.reduce((sum, a) => sum + (a.metadata?.sessions || 0), 0));

const loadAgents = async () => {
  loading.value = true;
  try {
    const response = await client.agents.list();
    if (response.success && response.data) {
      agents.value = response.data.map((record: any) => ({
        id: record.ID,
        name: record.Metadata?.name || "Unnamed Agent",
        description: record.Metadata?.description,
        avatar: record.Metadata?.avatar,
        status: record.Status || "idle",
        metadata: {
          ...record.Metadata,
          template_id: record.Config?.TemplateID,
          provider: record.Config?.ModelConfig?.Provider,
          model: record.Config?.ModelConfig?.Model,
        },
      }));
    }
  } catch (error) {
    console.error("Failed to load agents:", error);
  } finally {
    loading.value = false;
  }
};

const handleSubmit = async (data: any) => {
  formLoading.value = true;
  try {
    if (editingAgent.value) {
      // Update existing agent
      await client.agents.update(editingAgent.value.id, {
        name: data.name,
        metadata: {
          ...data.metadata,
          description: data.description,
        },
      });
    } else {
      // Create new agent
      await client.agents.create({
        template_id: data.template_id,
        name: data.name,
        model_config: data.model_config,
        metadata: {
          ...data.metadata,
          description: data.description,
        },
      });
    }
    await loadAgents();
    handleCancel();
  } catch (error) {
    console.error("Failed to save agent:", error);
  } finally {
    formLoading.value = false;
  }
};

const handleCancel = () => {
  showCreateForm.value = false;
  editingAgent.value = null;
};

const handleChat = (agent: Agent) => {
  emit("chat", agent);
};

const handleEdit = (agent: Agent) => {
  editingAgent.value = agent;
  showCreateForm.value = true;
};

const handleDelete = (agent: Agent) => {
  deletingAgent.value = agent;
};

const confirmDelete = async () => {
  if (!deletingAgent.value) return;

  try {
    await client.agents.delete(deletingAgent.value.id);
    await loadAgents();
    deletingAgent.value = null;
  } catch (error) {
    console.error("Failed to delete agent:", error);
  }
};

onMounted(() => {
  loadAgents();
});
</script>

<style scoped>
.agent-dashboard {
  @apply space-y-6;
}

.dashboard-header {
  @apply flex items-center justify-between;
}

.dashboard-title {
  @apply text-3xl font-bold text-text dark:text-text-dark;
}

.dashboard-subtitle {
  @apply text-secondary dark:text-secondary-dark mt-1;
}

.btn-create {
  @apply flex items-center gap-2 px-4 py-2 bg-primary hover:bg-primary-hover text-white rounded-lg font-medium transition-colors;
}

.stats-grid {
  @apply grid grid-cols-1 md:grid-cols-3 gap-4;
}

.stat-card {
  @apply bg-surface dark:bg-surface-dark border border-border dark:border-border-dark rounded-lg p-4 flex items-center gap-4;
}

.stat-icon {
  @apply w-12 h-12 rounded-lg flex items-center justify-center;
}

.stat-content {
  @apply flex-1;
}

.stat-value {
  @apply text-2xl font-bold text-text dark:text-text-dark;
}

.stat-label {
  @apply text-sm text-secondary dark:text-secondary-dark;
}

.delete-confirm {
  @apply space-y-4;
}

.delete-message {
  @apply text-text dark:text-text-dark;
}

.delete-warning {
  @apply text-sm text-red-600 dark:text-red-400;
}

.delete-actions {
  @apply flex justify-end gap-3 pt-4;
}

.btn-secondary {
  @apply px-4 py-2 bg-background dark:bg-background-dark hover:bg-border dark:hover:bg-border-dark text-text dark:text-text-dark border border-border dark:border-border-dark rounded-lg font-medium transition-colors;
}

.btn-danger {
  @apply px-4 py-2 bg-red-600 hover:bg-red-700 text-white rounded-lg font-medium transition-colors;
}
</style>
