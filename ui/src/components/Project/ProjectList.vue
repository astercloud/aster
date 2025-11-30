<template>
  <div class="space-y-4">
    <!-- 头部：标题和筛选 -->
    <div class="flex items-center justify-between">
      <h2 class="text-2xl font-bold text-gray-900 dark:text-white">
        我的项目
      </h2>
      <div class="flex items-center space-x-4">
        <!-- 工作空间筛选 -->
        <select
          v-model="selectedWorkspace"
          class="px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-800 text-gray-900 dark:text-white"
        >
          <option value="all">所有工作空间</option>
          <option value="wechat">微信公众号</option>
          <option value="video">视频脚本</option>
          <option value="general">通用文档</option>
        </select>

        <!-- 状态筛选 -->
        <select
          v-model="selectedStatus"
          class="px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-800 text-gray-900 dark:text-white"
        >
          <option value="all">所有状态</option>
          <option value="draft">草稿</option>
          <option value="in_progress">进行中</option>
          <option value="completed">已完成</option>
        </select>

        <!-- 新建按钮 -->
        <button
          class="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors font-medium"
          @click="$emit('create')"
        >
          + 新建项目
        </button>
      </div>
    </div>

    <!-- 项目网格 -->
    <div
      v-if="filteredProjects.length > 0"
      class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6"
    >
      <ProjectCard
        v-for="project in filteredProjects"
        :key="project.id"
        :project="project"
        @open="$emit('open', project)"
        @edit="$emit('edit', project)"
        @delete="$emit('delete', project)"
      />
    </div>

    <!-- 空状态 -->
    <div
      v-else
      class="text-center py-16"
    >
      <div class="text-6xl mb-4">📁</div>
      <h3 class="text-xl font-semibold text-gray-900 dark:text-white mb-2">
        暂无项目
      </h3>
      <p class="text-gray-500 dark:text-gray-400 mb-6">
        {{ emptyMessage }}
      </p>
      <button
        class="px-6 py-3 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors font-medium"
        @click="$emit('create')"
      >
        创建第一个项目
      </button>
    </div>
  </div>
</template>

<script lang="ts">
import { defineComponent, ref, computed, type PropType } from 'vue';
import type { Project } from '@/types';
import ProjectCard from './ProjectCard.vue';

export default defineComponent({
  name: 'ProjectList',
  components: {
    ProjectCard,
  },
  props: {
    projects: {
      type: Array as PropType<Project[]>,
      required: true,
    },
  },
  emits: {
    create: () => true,
    open: (project: Project) => true,
    edit: (project: Project) => true,
    delete: (project: Project) => true,
  },
  setup(props) {
    const selectedWorkspace = ref<string>('all');
    const selectedStatus = ref<string>('all');

    // 筛选项目
    const filteredProjects = computed(() => {
      return props.projects.filter((project) => {
        const workspaceMatch =
          selectedWorkspace.value === 'all' || project.workspace === selectedWorkspace.value;
        const statusMatch =
          selectedStatus.value === 'all' || project.status === selectedStatus.value;
        return workspaceMatch && statusMatch;
      });
    });

    // 空状态消息
    const emptyMessage = computed(() => {
      if (selectedWorkspace.value !== 'all' || selectedStatus.value !== 'all') {
        return '没有符合筛选条件的项目';
      }
      return '开始创建你的第一个 AI 写作项目';
    });

    return {
      selectedWorkspace,
      selectedStatus,
      filteredProjects,
      emptyMessage,
    };
  },
});
</script>
