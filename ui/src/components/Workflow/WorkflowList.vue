<template>
  <div class="workflow-list">
    <!-- Header -->
    <div class="list-header">
      <div class="header-left">
        <h2 class="list-title">Workflow 列表</h2>
        <span class="workflow-count">{{ workflows.length }} 个</span>
      </div>

      <div class="header-right">
        <button @click="$emit('create')" class="btn-create">
          <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4"></path>
          </svg>
          创建 Workflow
        </button>
      </div>
    </div>

    <!-- Filters -->
    <div class="list-filters">
      <div class="filter-search">
        <svg class="search-icon" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"></path>
        </svg>
        <input
          v-model="searchQuery"
          type="text"
          placeholder="搜索 Workflow..."
          class="search-input"
        />
      </div>

      <select v-model="statusFilter" class="filter-select">
        <option value="">全部状态</option>
        <option value="idle">空闲</option>
        <option value="running">运行中</option>
        <option value="paused">已暂停</option>
        <option value="completed">已完成</option>
        <option value="error">错误</option>
      </select>
    </div>

    <!-- Loading -->
    <LoadingSpinner v-if="loading" class="my-8" />

    <!-- Empty State -->
    <EmptyState
      v-else-if="filteredWorkflows.length === 0"
      title="暂无 Workflow"
      description="还没有创建任何 Workflow，点击上方按钮创建第一个"
    >
      <template #action>
        <button @click="$emit('create')" class="btn-create">
          创建 Workflow
        </button>
      </template>
    </EmptyState>

    <!-- Workflow Grid -->
    <div v-else class="workflow-grid">
      <WorkflowCard
        v-for="workflow in filteredWorkflows"
        :key="workflow.id"
        :workflow="workflow"
        @execute="$emit('execute', $event)"
        @edit="$emit('edit', $event)"
        @delete="$emit('delete', $event)"
      />
    </div>
  </div>
</template>

<script lang="ts">
import { defineComponent, ref, computed, type PropType } from 'vue';
import WorkflowCard from './WorkflowCard.vue';
import LoadingSpinner from '../Common/LoadingSpinner.vue';
import EmptyState from '../Common/EmptyState.vue';
import type { Workflow } from '@/types';

export default defineComponent({
  name: 'WorkflowList',
  components: {
    WorkflowCard,
    LoadingSpinner,
    EmptyState,
  },
  props: {
    workflows: {
      type: Array as PropType<Workflow[]>,
      required: true,
    },
    loading: {
      type: Boolean,
      default: false,
    },
  },
  emits: {
    create: () => true,
    execute: (workflow: Workflow) => true,
    edit: (workflow: Workflow) => true,
    delete: (workflow: Workflow) => true,
  },
  setup(props) {
    const searchQuery = ref('');
    const statusFilter = ref('');

    const filteredWorkflows = computed(() => {
      let result = props.workflows;

      // 搜索过滤
      if (searchQuery.value) {
        const query = searchQuery.value.toLowerCase();
        result = result.filter(workflow =>
          workflow.name.toLowerCase().includes(query) ||
          workflow.description?.toLowerCase().includes(query)
        );
      }

      // 状态过滤
      if (statusFilter.value) {
        result = result.filter(workflow => workflow.status === statusFilter.value);
      }

      return result;
    });

    return {
      searchQuery,
      statusFilter,
      filteredWorkflows,
    };
  },
});
</script>

<style scoped>
.workflow-list {
  @apply space-y-4;
}

.list-header {
  @apply flex items-center justify-between;
}

.header-left {
  @apply flex items-center gap-3;
}

.list-title {
  @apply text-2xl font-bold text-text dark:text-text-dark;
}

.workflow-count {
  @apply px-2 py-1 bg-primary/10 dark:bg-primary/20 text-primary dark:text-primary-light text-sm font-medium rounded-full;
}

.btn-create {
  @apply flex items-center gap-2 px-4 py-2 bg-primary hover:bg-primary-hover text-white rounded-lg font-medium transition-colors;
}

.list-filters {
  @apply flex gap-3;
}

.filter-search {
  @apply flex-1 relative;
}

.search-icon {
  @apply absolute left-3 top-1/2 -translate-y-1/2 w-5 h-5 text-secondary dark:text-secondary-dark;
}

.search-input {
  @apply w-full pl-10 pr-4 py-2 bg-surface dark:bg-surface-dark border border-border dark:border-border-dark rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-primary/20;
}

.filter-select {
  @apply px-4 py-2 bg-surface dark:bg-surface-dark border border-border dark:border-border-dark rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-primary/20;
}

.workflow-grid {
  @apply grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4;
}
</style>
