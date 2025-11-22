<template>
  <div class="component-docs">
    <!-- 顶部导航 -->
    <Navbar title="ChatUI 组件文档">
      <template #menu>
        <Search
          v-model="searchQuery"
          placeholder="搜索组件..."
          @search="handleSearch"
        />
      </template>
      <template #actions>
        <Button variant="text" icon="github">GitHub</Button>
      </template>
    </Navbar>

    <div class="docs-container">
      <!-- 左侧导航 -->
      <Sidebar title="组件" :collapsible="false" class="docs-sidebar">
        <div class="component-nav">
          <div
            v-for="category in categories"
            :key="category.name"
            class="nav-category"
          >
            <div class="category-title">{{ category.name }}</div>
            <div
              v-for="comp in category.components"
              :key="comp.key"
              :class="['nav-item', { active: currentComponent === comp.key }]"
              @click="selectComponent(comp.key)"
            >
              {{ comp.name }}
            </div>
          </div>
        </div>
      </Sidebar>

      <!-- 主内容区域 -->
      <div class="docs-main">
        <ScrollView>
          <div class="docs-content">
            <!-- 通用组件文档 -->
            <div v-if="currentComponent && !['button', 'bubble', 'avatar'].includes(currentComponent)" class="component-doc-page">
              <h1 class="doc-title">{{ getComponentName(currentComponent) }}</h1>
              <p class="doc-subtitle">{{ getComponentDescription(currentComponent) }}</p>
              
              <div class="doc-section">
                <h2 class="section-title">基础用法</h2>
                <div class="code-block">
                  <pre><code>{{ getComponentCode(currentComponent) }}</code></pre>
                </div>
              </div>
              
              <div class="doc-section">
                <h2 class="section-title">Props</h2>
                <div class="props-table">
                  <p class="text-gray-600 dark:text-gray-400">
                    详细的 Props 文档正在完善中...
                  </p>
                </div>
              </div>
              
              <div class="doc-section">
                <h2 class="section-title">示例</h2>
                <div class="demo-area">
                  <p class="text-gray-600 dark:text-gray-400">
                    交互式示例正在开发中，请访问 <router-link to="/components" class="text-blue-600 hover:underline">组件展示页面</router-link> 查看实际效果。
                  </p>
                </div>
              </div>
            </div>

            <!-- Button 文档 -->
            <DocViewer
              v-else-if="currentComponent === 'button'"
              :content="buttonDoc"
              :code="buttonCode"
            >
              <template #demo>
                <DemoSection title="基础用法">
                  <Button>默认按钮</Button>
                </DemoSection>
                
                <DemoSection title="按钮类型">
                  <Flex gap="md">
                    <Button variant="primary">主要按钮</Button>
                    <Button variant="secondary">次要按钮</Button>
                    <Button variant="text">文本按钮</Button>
                  </Flex>
                </DemoSection>
                
                <DemoSection title="按钮尺寸">
                  <Flex gap="md" align="center">
                    <Button size="sm">小按钮</Button>
                    <Button size="md">中按钮</Button>
                    <Button size="lg">大按钮</Button>
                  </Flex>
                </DemoSection>
                
                <DemoSection title="带图标">
                  <Flex gap="md">
                    <Button icon="send">发送</Button>
                    <Button icon="image" variant="secondary">图片</Button>
                  </Flex>
                </DemoSection>
              </template>
            </DocViewer>

            <!-- Bubble 文档 -->
            <DocViewer
              v-else-if="currentComponent === 'bubble'"
              :content="bubbleDoc"
              :code="bubbleCode"
            >
              <template #demo>
                <DemoSection title="基础用法">
                  <Flex direction="column" gap="md">
                    <Bubble content="你好！" position="left" />
                    <Bubble content="很高兴认识你" position="right" />
                  </Flex>
                </DemoSection>
                
                <DemoSection title="消息状态">
                  <Flex direction="column" gap="md">
                    <Bubble content="发送中..." position="right" status="pending" />
                    <Bubble content="已发送" position="right" status="sent" />
                    <Bubble content="发送失败" position="right" status="error" />
                  </Flex>
                </DemoSection>
                
                <DemoSection title="Markdown 支持">
                  <Bubble 
                    content="这是 **粗体** 和 *斜体* 文本，还有 `代码`"
                    position="left"
                  />
                </DemoSection>
              </template>
            </DocViewer>

            <!-- Avatar 文档 -->
            <DocViewer
              v-else-if="currentComponent === 'avatar'"
              :content="avatarDoc"
              :code="avatarCode"
            >
              <template #demo>
                <DemoSection title="不同尺寸">
                  <Flex gap="md" align="center">
                    <Avatar alt="U" size="xs" />
                    <Avatar alt="S" size="sm" />
                    <Avatar alt="M" size="md" />
                    <Avatar alt="L" size="lg" />
                    <Avatar alt="X" size="xl" />
                  </Flex>
                </DemoSection>
                
                <DemoSection title="状态指示">
                  <Flex gap="md" align="center">
                    <Avatar alt="在线" size="md" status="online" />
                    <Avatar alt="忙碌" size="md" status="busy" />
                    <Avatar alt="离线" size="md" status="offline" />
                  </Flex>
                </DemoSection>
              </template>
            </DocViewer>

            <!-- 默认页面 -->
            <div v-else class="welcome-page">
              <h1 class="welcome-title">ChatUI 组件文档</h1>
              <p class="welcome-subtitle">
                选择左侧组件查看详细文档和实时演示
              </p>
              <div class="stats-grid">
                <div class="stat-card">
                  <div class="stat-value">67</div>
                  <div class="stat-label">组件总数</div>
                </div>
                <div class="stat-card">
                  <div class="stat-value">46</div>
                  <div class="stat-label">ChatUI 组件</div>
                </div>
                <div class="stat-card">
                  <div class="stat-value">100%</div>
                  <div class="stat-label">TypeScript</div>
                </div>
              </div>
            </div>
          </div>
        </ScrollView>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue';
import {
  Navbar, Sidebar, ScrollView, Search, Button, Flex,
  Bubble, Avatar
} from '@/components/ChatUI';
import DocViewer from '@/components/DocViewer.vue';
import DemoSection from '@/components/DemoSection.vue';

// 导入文档内容
import buttonDoc from '@/docs/components/Button.md?raw';
import bubbleDoc from '@/docs/components/Bubble.md?raw';
import avatarDoc from '@/docs/components/Avatar.md?raw';

const searchQuery = ref('');
const currentComponent = ref('');

const categories = [
  {
    name: 'Chat 对话组件',
    tag: 'chat',
    components: [
      { key: 'chat', name: 'Chat 聊天容器' },
      { key: 'bubble', name: 'Bubble 消息气泡' },
      { key: 'think-bubble', name: 'ThinkBubble 思考气泡' },
      { key: 'typing-bubble', name: 'TypingBubble 输入中' },
      { key: 'card', name: 'Card 卡片' },
      { key: 'file-card', name: 'FileCard 文件卡片' },
      { key: 'message-status', name: 'MessageStatus 消息状态' },
      { key: 'system-message', name: 'SystemMessage 系统消息' },
    ],
  },
  {
    name: 'Basic 基础组件',
    tag: 'basic',
    components: [
      { key: 'button', name: 'Button 按钮' },
      { key: 'icon', name: 'Icon 图标' },
      { key: 'avatar', name: 'Avatar 头像' },
      { key: 'tag', name: 'Tag 标签' },
      { key: 'image', name: 'Image 图片' },
      { key: 'divider', name: 'Divider 分割线' },
    ],
  },
  {
    name: 'Form 表单组件',
    tag: 'form',
    components: [
      { key: 'input', name: 'Input 输入框' },
      { key: 'search', name: 'Search 搜索框' },
      { key: 'checkbox', name: 'Checkbox 复选框' },
      { key: 'radio', name: 'Radio 单选框' },
      { key: 'dropdown', name: 'Dropdown 下拉菜单' },
    ],
  },
  {
    name: 'Layout 布局组件',
    tag: 'layout',
    components: [
      { key: 'flex', name: 'Flex 弹性布局' },
      { key: 'navbar', name: 'Navbar 导航栏' },
      { key: 'sidebar', name: 'Sidebar 侧边栏' },
      { key: 'scroll-view', name: 'ScrollView 滚动视图' },
      { key: 'tabs', name: 'Tabs 标签页' },
      { key: 'carousel', name: 'Carousel 轮播图' },
    ],
  },
  {
    name: 'Feedback 反馈组件',
    tag: 'feedback',
    components: [
      { key: 'modal', name: 'Modal 对话框' },
      { key: 'notice', name: 'Notice 通知' },
      { key: 'tooltip', name: 'Tooltip 提示' },
      { key: 'popover', name: 'Popover 气泡卡片' },
      { key: 'progress', name: 'Progress 进度条' },
      { key: 'typing', name: 'Typing 打字效果' },
    ],
  },
  {
    name: 'Data 数据展示',
    tag: 'data',
    components: [
      { key: 'list', name: 'List 列表' },
      { key: 'rich-text', name: 'RichText 富文本' },
    ],
  },
  {
    name: 'Agent 专属组件',
    tag: 'agent',
    components: [
      { key: 'agent-card', name: 'AgentCard Agent卡片' },
      { key: 'thinking-block', name: 'ThinkingBlock 思考块' },
      { key: 'workflow-timeline', name: 'WorkflowTimeline 工作流' },
      { key: 'project-card', name: 'ProjectCard 项目卡片' },
      { key: 'room-card', name: 'RoomCard 房间卡片' },
      { key: 'workflow-card', name: 'WorkflowCard 工作流卡片' },
    ],
  },
];

const selectComponent = (key: string) => {
  currentComponent.value = key;
};

const handleSearch = (query: string) => {
  console.log('Search:', query);
};

// 获取组件名称
const getComponentName = (key: string) => {
  for (const category of categories) {
    const comp = category.components.find(c => c.key === key);
    if (comp) return comp.name;
  }
  return key;
};

// 获取组件描述
const getComponentDescription = (key: string) => {
  const descriptions: Record<string, string> = {
    'chat': '完整的聊天容器组件，包含消息列表、输入框等',
    'bubble': '消息气泡组件，支持左右位置、状态显示、Markdown 渲染',
    'think-bubble': '思考气泡组件，用于显示 AI 的思考过程',
    'typing-bubble': '输入中气泡组件，显示对方正在输入',
    'card': '卡片消息组件，支持标题、内容、操作按钮',
    'file-card': '文件卡片组件，用于显示文件信息',
    'message-status': '消息状态组件，显示发送、已读等状态',
    'system-message': '系统消息组件，用于显示系统通知',
    'button': '按钮组件，支持多种样式、尺寸、图标',
    'icon': '图标组件，内置常用图标',
    'avatar': '头像组件，支持多种尺寸、状态指示',
    'tag': '标签组件，用于分类和标记',
    'image': '图片组件，支持懒加载、预览',
    'divider': '分割线组件，支持文字分割',
    'input': '输入框组件，支持多种类型、验证',
    'search': '搜索框组件，带搜索图标和清除按钮',
    'checkbox': '复选框组件，支持单选和多选',
    'radio': '单选框组件，支持分组',
    'dropdown': '下拉菜单组件，支持多级菜单',
    'modal': '对话框组件，支持自定义内容',
    'notice': '通知组件，支持多种类型',
    'tooltip': '提示组件，鼠标悬停显示',
    'popover': '气泡卡片组件，点击显示',
    'progress': '进度条组件，显示任务进度',
    'typing': '打字效果组件，逐字显示文本',
    'flex': '弹性布局组件，快速实现 Flexbox 布局',
    'navbar': '导航栏组件，顶部导航',
    'sidebar': '侧边栏组件，支持折叠',
    'scroll-view': '滚动视图组件，优化滚动性能',
    'tabs': '标签页组件，支持多标签切换',
    'carousel': '轮播图组件，支持自动播放',
    'list': '列表组件，支持虚拟滚动',
    'rich-text': '富文本组件，支持 HTML 渲染',
    'agent-card': 'Agent 卡片组件，显示 Agent 信息',
    'thinking-block': '思考块组件，可视化 AI 推理过程',
    'workflow-timeline': '工作流时间线组件，显示执行步骤',
    'project-card': '项目卡片组件，项目管理',
    'room-card': '房间卡片组件，协作房间',
    'workflow-card': '工作流卡片组件，工作流管理',
  };
  return descriptions[key] || '暂无描述';
};

// 获取组件示例代码
const getComponentCode = (key: string) => {
  const codes: Record<string, string> = {
    'chat': `<template>
  <Chat :messages="messages" @send="handleSend" />
</template>

<script setup>
import { Chat } from '@/components/ChatUI';
const messages = ref([]);
<\/script>`,
    'bubble': `<template>
  <Bubble content="你好！" position="left" />
</template>

<script setup>
import { Bubble } from '@/components/ChatUI';
<\/script>`,
    'button': `<template>
  <Button variant="primary">主要按钮</Button>
</template>

<script setup>
import { Button } from '@/components/ChatUI';
<\/script>`,
  };
  return codes[key] || `<template>
  <${getComponentName(key).split(' ')[0]} />
</template>

<script setup>
import { ${getComponentName(key).split(' ')[0]} } from '@/components/ChatUI';
<\/script>`;
};

// 示例代码
const buttonCode = `<template>
  <Button variant="primary">主要按钮</Button>
</template>

<script setup>
import { Button } from '@/components/ChatUI';
<\/script>`;

const bubbleCode = `<template>
  <Bubble content="你好！" position="left" />
</template>

<script setup>
import { Bubble } from '@/components/ChatUI';
<\/script>`;

const avatarCode = `<template>
  <Avatar alt="User" size="md" status="online" />
</template>

<script setup>
import { Avatar } from '@/components/ChatUI';
<\/script>`;
</script>

<style scoped>
.component-docs {
  @apply min-h-screen bg-gray-50 dark:bg-gray-900;
}

.docs-container {
  @apply flex h-[calc(100vh-64px)];
}

.docs-sidebar {
  @apply w-64;
}

.component-nav {
  @apply space-y-6;
}

.nav-category {
  @apply space-y-1;
}

.category-title {
  @apply text-xs font-semibold text-gray-500 dark:text-gray-400 uppercase tracking-wider mb-2;
}

.nav-item {
  @apply px-3 py-2 text-sm text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-lg cursor-pointer transition-colors;
}

.nav-item.active {
  @apply bg-blue-50 dark:bg-blue-900/30 text-blue-600 dark:text-blue-400 font-medium;
}

.docs-main {
  @apply flex-1 overflow-hidden;
}

.docs-content {
  @apply p-8;
}

.welcome-page {
  @apply text-center py-20;
}

.welcome-title {
  @apply text-4xl font-bold text-gray-900 dark:text-white mb-4;
}

.welcome-subtitle {
  @apply text-lg text-gray-600 dark:text-gray-400 mb-12;
}

.stats-grid {
  @apply grid grid-cols-3 gap-6 max-w-2xl mx-auto;
}

.stat-card {
  @apply bg-white dark:bg-gray-800 rounded-xl p-6 border border-gray-200 dark:border-gray-700;
}

.stat-value {
  @apply text-3xl font-bold text-gray-900 dark:text-white mb-2;
}

.stat-label {
  @apply text-sm text-gray-600 dark:text-gray-400;
}

.component-doc-page {
  @apply max-w-4xl;
}

.doc-title {
  @apply text-4xl font-bold text-gray-900 dark:text-white mb-4;
}

.doc-subtitle {
  @apply text-lg text-gray-600 dark:text-gray-400 mb-8;
}

.doc-section {
  @apply mb-12;
}

.section-title {
  @apply text-2xl font-bold text-gray-900 dark:text-white mb-4;
}

.code-block {
  @apply bg-gray-900 dark:bg-gray-950 rounded-lg p-4 overflow-x-auto;
}

.code-block pre {
  @apply text-sm text-gray-100;
}

.code-block code {
  @apply font-mono;
}

.props-table {
  @apply bg-white dark:bg-gray-800 rounded-lg p-6 border border-gray-200 dark:border-gray-700;
}

.demo-area {
  @apply bg-white dark:bg-gray-800 rounded-lg p-6 border border-gray-200 dark:border-gray-700;
}
</style>
