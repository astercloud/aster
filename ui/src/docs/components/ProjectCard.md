# ProjectCard - 项目卡片

展示 AI Agent 项目/会话的卡片组件。

## 功能特性

- ✅ 项目名称和描述
- ✅ 工作空间类型（微信/视频/通用）
- ✅ 项目状态（草稿/进行中/已完成）
- ✅ 统计信息（字数、素材数、最后修改时间）
- ✅ 操作按钮（打开、编辑、删除）
- ✅ 悬停效果
- ✅ 深色模式支持

## 基础用法

\`\`\`vue
<template>
<ProjectCard
:project="project"
@open="handleOpen"
@edit="handleEdit"
@delete="handleDelete"
/>
</template>

<script setup>
import { ProjectCard } from '@/components/Project';

const project = {
  id: '1',
  name: '产品发布文章',
  description: '介绍新产品的特性和优势',
  workspace: 'wechat',
  status: 'in_progress',
  lastModified: '2024-11-22T10:30:00Z',
  stats: {
    words: 1500,
    materials: 5,
  },
};

const handleOpen = (project) => {
  console.log('打开项目:', project);
};

const handleEdit = (project) => {
  console.log('编辑项目:', project);
};

const handleDelete = (project) => {
  console.log('删除项目:', project);
};
</script>

\`\`\`

## 工作空间类型

### 微信公众号

\`\`\`typescript
{
workspace: 'wechat',
// 显示绿色图标 💬
}
\`\`\`

### 视频脚本

\`\`\`typescript
{
workspace: 'video',
// 显示紫色图标 🎬
}
\`\`\`

### 通用文档

\`\`\`typescript
{
workspace: 'general',
// 显示蓝色图标 📄
}
\`\`\`

## 项目状态

### 草稿

\`\`\`typescript
{
status: 'draft',
// 灰色标签
}
\`\`\`

### 进行中

\`\`\`typescript
{
status: 'in_progress',
// 蓝色标签
}
\`\`\`

### 已完成

\`\`\`typescript
{
status: 'completed',
// 绿色标签
}
\`\`\`

## 项目列表

使用 \`ProjectList\` 组件展示多个项目：

\`\`\`vue
<template>
<ProjectList
:projects="projects"
@create="handleCreate"
@open="handleOpen"
@edit="handleEdit"
@delete="handleDelete"
/>
</template>

<script setup>
import { ref } from 'vue';
import { ProjectList } from '@/components/Project';

const projects = ref([
  {
    id: '1',
    name: '产品发布文章',
    description: '介绍新产品的特性和优势',
    workspace: 'wechat',
    status: 'in_progress',
    lastModified: '2024-11-22T10:30:00Z',
    stats: { words: 1500, materials: 5 },
  },
  {
    id: '2',
    name: '教程视频脚本',
    description: '如何使用我们的产品',
    workspace: 'video',
    status: 'draft',
    lastModified: '2024-11-21T15:20:00Z',
    stats: { words: 800, materials: 3 },
  },
]);

const handleCreate = () => {
  console.log('创建新项目');
};

const handleOpen = (project) => {
  console.log('打开项目:', project);
};

const handleEdit = (project) => {
  console.log('编辑项目:', project);
};

const handleDelete = (project) => {
  console.log('删除项目:', project);
};
</script>

\`\`\`

## Props

### ProjectCard

| 属性    | 类型    | 必填 | 说明     |
| ------- | ------- | ---- | -------- |
| project | Project | 是   | 项目对象 |

### ProjectList

| 属性     | 类型      | 必填 | 说明     |
| -------- | --------- | ---- | -------- |
| projects | Project[] | 是   | 项目列表 |

## Events

### ProjectCard

| 事件   | 参数             | 说明     |
| ------ | ---------------- | -------- |
| open   | project: Project | 打开项目 |
| edit   | project: Project | 编辑项目 |
| delete | project: Project | 删除项目 |

### ProjectList

| 事件   | 参数             | 说明       |
| ------ | ---------------- | ---------- |
| create | -                | 创建新项目 |
| open   | project: Project | 打开项目   |
| edit   | project: Project | 编辑项目   |
| delete | project: Project | 删除项目   |

## 类型定义

\`\`\`typescript
export interface Project {
id: string;
name: string;
description?: string;
workspace: 'wechat' | 'video' | 'general';
lastModified: string;
status: 'draft' | 'in_progress' | 'completed';
stats: {
words: number;
materials: number;
};
}
\`\`\`

## 样式定制

### 卡片悬停效果

\`\`\`css
.project-card:hover {
box-shadow: 0 10px 15px -3px rgba(0, 0, 0, 0.1);
}
\`\`\`

### 自定义工作空间颜色

\`\`\`typescript
const workspaceConfig = {
wechat: {
icon: '💬',
label: '微信公众号',
class: 'bg-green-100 text-green-600',
},
// 可以自定义更多工作空间类型
};
\`\`\`

## 最佳实践

### 1. 日期格式化

组件自动将日期格式化为相对时间（今天、昨天、X天前等）。

### 2. 删除确认

删除操作会弹出确认对话框，防止误删。

### 3. 响应式布局

ProjectList 使用网格布局，自动适配不同屏幕尺寸：

- 移动端：1列
- 平板：2列
- 桌面：3列

### 4. 筛选功能

ProjectList 提供工作空间和状态筛选，方便用户查找项目。

## 使用场景

1. **项目管理页面** - 展示所有 AI 写作项目
2. **工作空间首页** - 显示最近的项目
3. **项目选择器** - 让用户选择要打开的项目
4. **项目归档** - 管理已完成的项目

## 注意事项

- 确保 Project 类型已在 \`@/types\` 中定义
- 删除操作需要在父组件中处理实际的删除逻辑
- 日期字符串应为 ISO 8601 格式
- 统计信息会实时更新，需要在父组件中维护
