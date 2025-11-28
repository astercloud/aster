<template>
  <div class="min-h-screen bg-gray-50 dark:bg-gray-900 p-8">
    <div class="max-w-7xl mx-auto">
      <ProjectList
        :projects="projects"
        @create="handleCreate"
        @open="handleOpen"
        @edit="handleEdit"
        @delete="handleDelete"
      />
    </div>

    <!-- 创建/编辑项目对话框 -->
    <div
      v-if="showDialog"
      class="fixed inset-0 bg-black/50 flex items-center justify-center z-50"
      @click.self="showDialog = false"
    >
      <div class="bg-white dark:bg-gray-800 rounded-lg p-6 w-full max-w-md">
        <h3 class="text-xl font-bold text-gray-900 dark:text-white mb-4">
          {{ editingProject ? '编辑项目' : '创建项目' }}
        </h3>

        <form @submit.prevent="handleSubmit" class="space-y-4">
          <!-- 项目名称 -->
          <div>
            <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
              项目名称
            </label>
            <input
              v-model="formData.name"
              type="text"
              required
              class="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white"
              placeholder="输入项目名称"
            />
          </div>

          <!-- 项目描述 -->
          <div>
            <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
              项目描述
            </label>
            <textarea
              v-model="formData.description"
              rows="3"
              class="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white"
              placeholder="输入项目描述（可选）"
            />
          </div>

          <!-- 工作空间类型 -->
          <div>
            <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
              工作空间类型
            </label>
            <select
              v-model="formData.workspace"
              required
              class="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white"
            >
              <option value="wechat">💬 微信公众号</option>
              <option value="video">🎬 视频脚本</option>
              <option value="general">📄 通用文档</option>
            </select>
          </div>

          <!-- 项目状态 -->
          <div>
            <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
              项目状态
            </label>
            <select
              v-model="formData.status"
              required
              class="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white"
            >
              <option value="draft">草稿</option>
              <option value="in_progress">进行中</option>
              <option value="completed">已完成</option>
            </select>
          </div>

          <!-- 按钮 -->
          <div class="flex items-center space-x-3 pt-4">
            <button
              type="submit"
              class="flex-1 px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors font-medium"
            >
              {{ editingProject ? '保存' : '创建' }}
            </button>
            <button
              type="button"
              class="flex-1 px-4 py-2 border border-gray-300 dark:border-gray-600 text-gray-700 dark:text-gray-300 rounded-lg hover:bg-gray-50 dark:hover:bg-gray-700 transition-colors font-medium"
              @click="showDialog = false"
            >
              取消
            </button>
          </div>
        </form>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue';
import { ProjectList } from '@/components/Project';
import type { Project } from '@/types';

// 示例项目数据
const projects = ref<Project[]>([
  {
    id: '1',
    name: '产品发布文章',
    description: '介绍新产品的特性和优势，面向潜在客户',
    workspace: 'wechat',
    status: 'in_progress',
    lastModified: new Date().toISOString(),
    stats: {
      words: 1500,
      materials: 5,
    },
  },
  {
    id: '2',
    name: '教程视频脚本',
    description: '如何使用我们的产品，适合新手用户',
    workspace: 'video',
    status: 'draft',
    lastModified: new Date(Date.now() - 86400000).toISOString(), // 昨天
    stats: {
      words: 800,
      materials: 3,
    },
  },
  {
    id: '3',
    name: '技术文档',
    description: 'API 接口文档和使用说明',
    workspace: 'general',
    status: 'completed',
    lastModified: new Date(Date.now() - 86400000 * 7).toISOString(), // 7天前
    stats: {
      words: 3200,
      materials: 12,
    },
  },
]);

const showDialog = ref(false);
const editingProject = ref<Project | null>(null);
const formData = ref({
  name: '',
  description: '',
  workspace: 'general' as 'wechat' | 'video' | 'general',
  status: 'draft' as 'draft' | 'in_progress' | 'completed',
});

const handleCreate = () => {
  editingProject.value = null;
  formData.value = {
    name: '',
    description: '',
    workspace: 'general',
    status: 'draft',
  };
  showDialog.value = true;
};

const handleOpen = (project: Project) => {
  console.log('打开项目:', project);
  // TODO: 导航到项目详情页
  alert(`打开项目: ${project.name}`);
};

const handleEdit = (project: Project) => {
  editingProject.value = project;
  formData.value = {
    name: project.name,
    description: project.description || '',
    workspace: project.workspace,
    status: project.status,
  };
  showDialog.value = true;
};

const handleDelete = (project: Project) => {
  const index = projects.value.findIndex((p) => p.id === project.id);
  if (index !== -1) {
    projects.value.splice(index, 1);
  }
};

const handleSubmit = () => {
  if (editingProject.value) {
    // 更新现有项目
    const index = projects.value.findIndex((p) => p.id === editingProject.value!.id);
    if (index !== -1) {
      const existingProject = projects.value[index];
      if (existingProject) {
        projects.value[index] = {
          ...existingProject,
          ...formData.value,
          lastModified: new Date().toISOString(),
        };
      }
    }
  } else {
    // 创建新项目
    const newProject: Project = {
      id: Date.now().toString(),
      ...formData.value,
      lastModified: new Date().toISOString(),
      stats: {
        words: 0,
        materials: 0,
      },
    };
    projects.value.unshift(newProject);
  }
  showDialog.value = false;
};
</script>
