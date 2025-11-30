<template>
  <div
    class="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 p-6 hover:shadow-lg transition-shadow cursor-pointer"
    @click="$emit('open', project)"
  >
    <!-- 头部：图标和状态 -->
    <div class="flex items-start justify-between mb-4">
      <div class="flex items-center space-x-3">
        <!-- 工作空间图标 -->
        <div
          class="w-12 h-12 rounded-lg flex items-center justify-center text-2xl"
          :class="workspaceIconClass"
        >
          {{ workspaceIcon }}
        </div>
        <div>
          <h3 class="text-lg font-semibold text-gray-900 dark:text-white">
            {{ project.name }}
          </h3>
          <p class="text-sm text-gray-500 dark:text-gray-400">
            {{ workspaceLabel }}
          </p>
        </div>
      </div>

      <!-- 状态标签 -->
      <span
        class="px-3 py-1 rounded-full text-xs font-medium"
        :class="statusClass"
      >
        {{ statusLabel }}
      </span>
    </div>

    <!-- 描述 -->
    <p
      v-if="project.description"
      class="text-sm text-gray-600 dark:text-gray-300 mb-4 line-clamp-2"
    >
      {{ project.description }}
    </p>

    <!-- 统计信息 -->
    <div class="flex items-center space-x-6 mb-4 text-sm text-gray-500 dark:text-gray-400">
      <div class="flex items-center space-x-1">
        <span>📝</span>
        <span>{{ project.stats.words }} 字</span>
      </div>
      <div class="flex items-center space-x-1">
        <span>📎</span>
        <span>{{ project.stats.materials }} 素材</span>
      </div>
      <div class="flex items-center space-x-1">
        <span>🕒</span>
        <span>{{ formattedDate }}</span>
      </div>
    </div>

    <!-- 操作按钮 -->
    <div class="flex items-center space-x-2 pt-4 border-t border-gray-100 dark:border-gray-700">
      <button
        class="flex-1 px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors text-sm font-medium"
        @click.stop="$emit('open', project)"
      >
        打开
      </button>
      <button
        class="px-4 py-2 border border-gray-300 dark:border-gray-600 text-gray-700 dark:text-gray-300 rounded-lg hover:bg-gray-50 dark:hover:bg-gray-700 transition-colors text-sm font-medium"
        @click.stop="$emit('edit', project)"
      >
        编辑
      </button>
      <button
        class="px-4 py-2 border border-red-300 dark:border-red-600 text-red-600 dark:text-red-400 rounded-lg hover:bg-red-50 dark:hover:bg-red-900/20 transition-colors text-sm font-medium"
        @click.stop="handleDelete"
      >
        删除
      </button>
    </div>
  </div>
</template>

<script lang="ts">
import { defineComponent, computed, type PropType } from 'vue';
import type { Project } from '@/types';

export default defineComponent({
  name: 'ProjectCard',
  props: {
    project: {
      type: Object as PropType<Project>,
      required: true,
    },
  },
  emits: {
    open: (project: Project) => true,
    edit: (project: Project) => true,
    delete: (project: Project) => true,
  },
  setup(props, { emit }) {
    // 工作空间配置
    const workspaceConfig = {
      wechat: {
        icon: '💬',
        label: '微信公众号',
        class: 'bg-green-100 dark:bg-green-900/30 text-green-600 dark:text-green-400',
      },
      video: {
        icon: '🎬',
        label: '视频脚本',
        class: 'bg-purple-100 dark:bg-purple-900/30 text-purple-600 dark:text-purple-400',
      },
      general: {
        icon: '📄',
        label: '通用文档',
        class: 'bg-blue-100 dark:bg-blue-900/30 text-blue-600 dark:text-blue-400',
      },
    };

    const workspaceIcon = computed(() => workspaceConfig[props.project.workspace].icon);
    const workspaceLabel = computed(() => workspaceConfig[props.project.workspace].label);
    const workspaceIconClass = computed(() => workspaceConfig[props.project.workspace].class);

    // 状态配置
    const statusConfig = {
      draft: {
        label: '草稿',
        class: 'bg-gray-100 dark:bg-gray-700 text-gray-600 dark:text-gray-300',
      },
      in_progress: {
        label: '进行中',
        class: 'bg-blue-100 dark:bg-blue-900/30 text-blue-600 dark:text-blue-400',
      },
      completed: {
        label: '已完成',
        class: 'bg-green-100 dark:bg-green-900/30 text-green-600 dark:text-green-400',
      },
    };

    const statusLabel = computed(() => statusConfig[props.project.status].label);
    const statusClass = computed(() => statusConfig[props.project.status].class);

    // 格式化日期
    const formattedDate = computed(() => {
      const date = new Date(props.project.lastModified);
      const now = new Date();
      const diff = now.getTime() - date.getTime();
      const days = Math.floor(diff / (1000 * 60 * 60 * 24));

      if (days === 0) return '今天';
      if (days === 1) return '昨天';
      if (days < 7) return `${days} 天前`;
      if (days < 30) return `${Math.floor(days / 7)} 周前`;
      if (days < 365) return `${Math.floor(days / 30)} 月前`;
      return `${Math.floor(days / 365)} 年前`;
    });

    const handleDelete = () => {
      if (confirm(`确定要删除项目 "${props.project.name}" 吗？`)) {
        emit('delete', props.project);
      }
    };

    return {
      workspaceIcon,
      workspaceLabel,
      workspaceIconClass,
      statusLabel,
      statusClass,
      formattedDate,
      handleDelete,
    };
  },
});
</script>

<style scoped>
.line-clamp-2 {
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}
</style>
