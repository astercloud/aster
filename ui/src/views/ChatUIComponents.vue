<template>
  <div class="chatui-components-page">
    <Navbar title="ChatUI 组件库">
      <template #menu>
        <a
          v-for="section in sections"
          :key="section.id"
          :href="`#${section.id}`"
          class="nav-link"
        >
          {{ section.name }}
        </a>
      </template>
      <template #actions>
        <Button variant="primary">GitHub</Button>
      </template>
    </Navbar>

    <div class="page-container">
      <Sidebar title="组件导航" collapsible>
        <List :items="allComponents" @select="scrollToComponent">
          <template #default="{ item }">
            <div class="component-item">
              <span>{{ item.name }}</span>
              <Tag size="sm" :color="item.category === 'chat' ? 'primary' : 'default'">
                {{ item.category }}
              </Tag>
            </div>
          </template>
        </List>
      </Sidebar>

      <ScrollView class="main-content">
        <!-- Hero -->
        <section class="hero-section">
          <h1 class="hero-title">ChatUI 组件库</h1>
          <p class="hero-subtitle">
            参考 ChatUI 设计的完整对话界面组件库，专为 Aster Agent 打造
          </p>
          <Flex justify="center" gap="md">
            <Button variant="primary" size="lg">快速开始</Button>
            <Button variant="secondary" size="lg">查看文档</Button>
          </Flex>
        </section>

        <Divider>组件展示</Divider>

        <!-- 对话组件 -->
        <section id="chat" class="component-section">
          <h2 class="section-title">对话组件</h2>
          
          <div class="demo-card">
            <h3 class="demo-title">Bubble - 消息气泡</h3>
            <div class="demo-content">
              <Flex direction="column" gap="md">
                <Bubble content="你好！我是 Aster Agent" position="left" />
                <Bubble content="很高兴认识你" position="right" status="sent" />
              </Flex>
            </div>
          </div>

          <div class="demo-card">
            <h3 class="demo-title">ThinkBubble - 思考气泡</h3>
            <div class="demo-content">
              <ThinkBubble content="正在分析你的问题..." />
            </div>
          </div>

          <div class="demo-card">
            <h3 class="demo-title">Card - 卡片消息</h3>
            <div class="demo-content">
              <Card
                title="推荐文章"
                content="这是一篇关于 AI Agent 的深度文章"
                :actions="[
                  { text: '查看详情', value: 'view' },
                  { text: '分享', value: 'share' }
                ]"
              />
            </div>
          </div>

          <div class="demo-card">
            <h3 class="demo-title">SystemMessage - 系统消息</h3>
            <div class="demo-content">
              <SystemMessage content="Agent 已加入对话" />
            </div>
          </div>
        </section>

        <!-- 基础组件 -->
        <section id="basic" class="component-section">
          <h2 class="section-title">基础组件</h2>
          
          <div class="demo-card">
            <h3 class="demo-title">Button - 按钮</h3>
            <div class="demo-content">
              <Flex gap="md" wrap>
                <Button variant="primary">主要按钮</Button>
                <Button variant="secondary">次要按钮</Button>
                <Button variant="text">文本按钮</Button>
                <Button variant="primary" icon="send">发送</Button>
              </Flex>
            </div>
          </div>

          <div class="demo-card">
            <h3 class="demo-title">Avatar - 头像</h3>
            <div class="demo-content">
              <Flex gap="md" align="center">
                <Avatar alt="User" size="xs" />
                <Avatar alt="Agent" size="sm" status="online" />
                <Avatar alt="Bot" size="md" status="busy" />
                <Avatar alt="AI" size="lg" />
                <Avatar alt="System" size="xl" status="offline" />
              </Flex>
            </div>
          </div>

          <div class="demo-card">
            <h3 class="demo-title">Tag - 标签</h3>
            <div class="demo-content">
              <Flex gap="sm" wrap>
                <Tag>默认</Tag>
                <Tag color="primary">主要</Tag>
                <Tag color="success">成功</Tag>
                <Tag color="warning">警告</Tag>
                <Tag color="error">错误</Tag>
                <Tag closable @close="console.log('closed')">可关闭</Tag>
              </Flex>
            </div>
          </div>
        </section>

        <!-- 表单组件 -->
        <section id="form" class="component-section">
          <h2 class="section-title">表单组件</h2>
          
          <div class="demo-card">
            <h3 class="demo-title">Input - 输入框</h3>
            <div class="demo-content">
              <Input
                v-model="inputValue"
                label="用户名"
                placeholder="请输入用户名"
              />
            </div>
          </div>

          <div class="demo-card">
            <h3 class="demo-title">Search - 搜索框</h3>
            <div class="demo-content">
              <Search
                v-model="searchValue"
                placeholder="搜索组件..."
                @search="handleSearch"
              />
            </div>
          </div>

          <div class="demo-card">
            <h3 class="demo-title">Checkbox & Radio</h3>
            <div class="demo-content">
              <Flex direction="column" gap="md">
                <Checkbox v-model="checked">同意用户协议</Checkbox>
                <Flex gap="md">
                  <Radio v-model="radioValue" value="a" name="demo">选项 A</Radio>
                  <Radio v-model="radioValue" value="b" name="demo">选项 B</Radio>
                </Flex>
              </Flex>
            </div>
          </div>
        </section>

        <!-- 反馈组件 -->
        <section id="feedback" class="component-section">
          <h2 class="section-title">反馈组件</h2>
          
          <div class="demo-card">
            <h3 class="demo-title">Notice - 通知提示</h3>
            <div class="demo-content">
              <Flex direction="column" gap="md">
                <Notice type="info" content="这是一条信息提示" />
                <Notice type="success" title="成功" content="操作已成功完成" closable />
                <Notice type="warning" content="请注意检查输入内容" />
                <Notice type="error" content="发生了一个错误" />
              </Flex>
            </div>
          </div>

          <div class="demo-card">
            <h3 class="demo-title">Progress - 进度条</h3>
            <div class="demo-content">
              <Flex direction="column" gap="md">
                <Progress :percent="30" label="上传中" />
                <Progress :percent="100" status="success" label="已完成" />
                <Progress :percent="50" status="error" label="上传失败" />
              </Flex>
            </div>
          </div>

          <div class="demo-card">
            <h3 class="demo-title">Tooltip - 工具提示</h3>
            <div class="demo-content">
              <Flex gap="md">
                <Tooltip content="顶部提示" position="top">
                  <Button>上</Button>
                </Tooltip>
                <Tooltip content="右侧提示" position="right">
                  <Button>右</Button>
                </Tooltip>
                <Tooltip content="底部提示" position="bottom">
                  <Button>下</Button>
                </Tooltip>
                <Tooltip content="左侧提示" position="left">
                  <Button>左</Button>
                </Tooltip>
              </Flex>
            </div>
          </div>
        </section>

        <!-- 布局组件 -->
        <section id="layout" class="component-section">
          <h2 class="section-title">布局组件</h2>
          
          <div class="demo-card">
            <h3 class="demo-title">Tabs - 标签页</h3>
            <div class="demo-content">
              <Tabs
                :tabs="[
                  { key: 'tab1', label: '标签一' },
                  { key: 'tab2', label: '标签二' },
                  { key: 'tab3', label: '标签三' }
                ]"
                v-model="activeTab"
              >
                <div v-if="activeTab === 'tab1'">标签一的内容</div>
                <div v-if="activeTab === 'tab2'">标签二的内容</div>
                <div v-if="activeTab === 'tab3'">标签三的内容</div>
              </Tabs>
            </div>
          </div>

          <div class="demo-card">
            <h3 class="demo-title">Divider - 分割线</h3>
            <div class="demo-content">
              <Flex direction="column" gap="md">
                <div>内容上方</div>
                <Divider />
                <div>内容下方</div>
                <Divider>带文字的分割线</Divider>
                <div>更多内容</div>
              </Flex>
            </div>
          </div>
        </section>
      </ScrollView>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue';
import {
  Navbar, Sidebar, ScrollView, Tabs, Divider, Flex, List,
  Button, Avatar, Tag, Input, Search, Checkbox, Radio,
  Bubble, ThinkBubble, Card, SystemMessage,
  Notice, Progress, Tooltip
} from '@/components/ChatUI';

const sections = [
  { id: 'chat', name: '对话组件' },
  { id: 'basic', name: '基础组件' },
  { id: 'form', name: '表单组件' },
  { id: 'feedback', name: '反馈组件' },
  { id: 'layout', name: '布局组件' },
];

const allComponents = [
  // Chat 对话组件
  { name: 'Chat 聊天容器', category: 'chat' },
  { name: 'Bubble 消息气泡', category: 'chat' },
  { name: 'ThinkBubble 思考气泡', category: 'chat' },
  { name: 'TypingBubble 输入中', category: 'chat' },
  { name: 'Card 卡片', category: 'chat' },
  { name: 'FileCard 文件卡片', category: 'chat' },
  { name: 'MessageStatus 消息状态', category: 'chat' },
  { name: 'SystemMessage 系统消息', category: 'chat' },
  
  // Basic 基础组件
  { name: 'Button 按钮', category: 'basic' },
  { name: 'Icon 图标', category: 'basic' },
  { name: 'Avatar 头像', category: 'basic' },
  { name: 'Tag 标签', category: 'basic' },
  { name: 'Image 图片', category: 'basic' },
  { name: 'Divider 分割线', category: 'basic' },
  
  // Form 表单组件
  { name: 'Input 输入框', category: 'form' },
  { name: 'Search 搜索框', category: 'form' },
  { name: 'Checkbox 复选框', category: 'form' },
  { name: 'Radio 单选框', category: 'form' },
  { name: 'Dropdown 下拉菜单', category: 'form' },
  { name: 'MultimodalInput 多模态输入', category: 'form' },
  
  // Feedback 反馈组件
  { name: 'Modal 对话框', category: 'feedback' },
  { name: 'Notice 通知', category: 'feedback' },
  { name: 'Tooltip 提示', category: 'feedback' },
  { name: 'Popover 气泡卡片', category: 'feedback' },
  { name: 'Progress 进度条', category: 'feedback' },
  { name: 'Typing 打字效果', category: 'feedback' },
  
  // Layout 布局组件
  { name: 'Flex 弹性布局', category: 'layout' },
  { name: 'Navbar 导航栏', category: 'layout' },
  { name: 'Sidebar 侧边栏', category: 'layout' },
  { name: 'ScrollView 滚动视图', category: 'layout' },
  { name: 'Tabs 标签页', category: 'layout' },
  { name: 'Carousel 轮播图', category: 'layout' },
  
  // Data 数据展示
  { name: 'List 列表', category: 'data' },
  { name: 'RichText 富文本', category: 'data' },
  
  // Agent 专属组件
  { name: 'AgentCard Agent卡片', category: 'agent' },
  { name: 'ThinkingBlock 思考块', category: 'agent' },
  { name: 'WorkflowTimeline 工作流', category: 'agent' },
  { name: 'EditorPanel 编辑器', category: 'agent' },
  { name: 'ProjectCard 项目卡片', category: 'agent' },
  { name: 'RoomCard 房间卡片', category: 'agent' },
  { name: 'WorkflowCard 工作流卡片', category: 'agent' },
];

const inputValue = ref('');
const searchValue = ref('');
const checked = ref(false);
const radioValue = ref('a');
const activeTab = ref('tab1');

const scrollToComponent = (item: any) => {
  console.log('Scroll to:', item.name);
};

const handleSearch = (value: string) => {
  console.log('Search:', value);
};
</script>

<style scoped>
.chatui-components-page {
  @apply min-h-screen bg-gray-50 dark:bg-gray-900;
}

.page-container {
  @apply flex h-[calc(100vh-64px)];
}

.main-content {
  @apply flex-1 p-8;
}

.hero-section {
  @apply text-center py-16 space-y-6;
}

.hero-title {
  @apply text-5xl font-bold text-gray-900 dark:text-white;
}

.hero-subtitle {
  @apply text-xl text-gray-600 dark:text-gray-400 max-w-2xl mx-auto;
}

.component-section {
  @apply py-12 space-y-8;
}

.section-title {
  @apply text-3xl font-bold text-gray-900 dark:text-white mb-8;
}

.demo-card {
  @apply bg-white dark:bg-gray-800 rounded-xl p-6 shadow-sm border border-gray-200 dark:border-gray-700;
}

.demo-title {
  @apply text-lg font-semibold text-gray-900 dark:text-white mb-4;
}

.demo-content {
  @apply p-4 bg-gray-50 dark:bg-gray-900 rounded-lg;
}

.nav-link {
  @apply text-sm font-medium text-gray-600 dark:text-gray-400 hover:text-gray-900 dark:hover:text-white transition-colors;
}

.component-item {
  @apply flex items-center justify-between;
}
</style>
