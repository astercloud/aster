# ChatUI 组件库

参考 [ChatUI](https://chatui.io/) 设计的完整对话界面组件库，专为 Aster Agent 打造。

## 📦 组件总览

### 对话组件 (9个)

- **Chat** - 聊天容器，完整的对话界面
- **Bubble** - 消息气泡，支持 Markdown
- **ThinkBubble** - 思考气泡，显示 Agent 推理
- **TypingBubble** - 打字指示器，三点动画
- **Typing** - 输入状态提示
- **Card** - 卡片消息，支持操作按钮
- **FileCard** - 文件卡片，显示文件信息
- **SystemMessage** - 系统消息提示
- **MessageStatus** - 消息状态指示器

### 基础组件 (4个)

- **Button** - 按钮，多种样式和尺寸
- **Icon** - 图标，内置常用图标
- **Avatar** - 头像，支持状态指示
- **Image** - 图片，自动加载和错误处理

### 表单组件 (4个)

- **Input** - 输入框，支持标签和错误提示
- **Search** - 搜索框，带清除按钮
- **Checkbox** - 复选框
- **Radio** - 单选框

### 布局组件 (8个)

- **Flex** - 弹性布局容器
- **Divider** - 分割线，支持文字
- **List** - 列表，支持自定义项
- **Navbar** - 导航栏
- **Sidebar** - 侧边栏，可折叠
- **Tabs** - 标签页
- **ScrollView** - 滚动视图，支持回到顶部
- **Carousel** - 轮播图

### 反馈组件 (6个)

- **Notice** - 通知提示，多种类型
- **Progress** - 进度条，支持状态
- **Tooltip** - 工具提示，四个方向
- **Popover** - 气泡卡片
- **Modal** - 模态框
- **Dropdown** - 下拉菜单

### 数据展示 (2个)

- **Tag** - 标签，多种颜色
- **RichText** - 富文本，Markdown 渲染

**总计：33 个组件** ✨

## 快速开始

### 安装依赖

```bash
npm install marked
```

### 基础用法

```vue
<template>
  <Chat :messages="messages" placeholder="输入消息..." :quick-replies="quickReplies" @send="handleSend" />
</template>

<script setup>
import { ref } from "vue";
import { Chat } from "@/components/ChatUI";

const messages = ref([
  {
    id: "1",
    type: "text",
    content: "你好！",
    position: "left",
  },
]);

const quickReplies = [
  { name: "帮我写文章", value: "write" },
  { name: "分析代码", value: "analyze" },
];

const handleSend = (message) => {
  messages.value.push({
    id: Date.now().toString(),
    type: "text",
    content: message.content,
    position: "right",
  });
};
</script>
```

## 组件列表

### Chat - 聊天容器

主聊天组件，包含消息列表和输入区域。

**Props:**

- `messages` - 消息列表
- `placeholder` - 输入框占位符
- `disabled` - 是否禁用输入
- `quickReplies` - 快捷回复列表
- `toolbar` - 工具栏按钮

**Events:**

- `send` - 发送消息
- `quickReply` - 点击快捷回复
- `cardAction` - 卡片操作

### Bubble - 消息气泡

显示文本消息的气泡组件。

**Props:**

- `content` - 消息内容（支持 Markdown）
- `position` - 位置 `'left' | 'right'`
- `status` - 状态 `'pending' | 'sent' | 'error'`
- `avatar` - 头像 URL

**特性:**

- 自动渲染 Markdown
- 支持代码高亮
- 消息状态指示器

### ThinkBubble - 思考气泡

显示 Agent 思考状态的组件。

**Props:**

- `content` - 思考内容

**使用场景:**

- Agent 正在处理请求
- 显示推理过程
- 工具调用状态

### TypingBubble - 打字指示器

显示对方正在输入的动画。

**特性:**

- 三点动画效果
- 自动循环播放

### Card - 卡片消息

显示结构化内容的卡片组件。

**Props:**

- `title` - 卡片标题
- `content` - 卡片内容
- `actions` - 操作按钮列表

**Events:**

- `action` - 点击操作按钮

**示例:**

```vue
<Card
  title="推荐文章"
  content="这是一篇关于 AI 的文章..."
  :actions="[
    { text: '查看详情', value: 'view' },
    { text: '分享', value: 'share' },
  ]"
  @action="handleAction"
/>
```

### FileCard - 文件卡片

显示文件信息和下载链接。

**Props:**

- `file` - 文件对象
  - `name` - 文件名
  - `size` - 文件大小（字节）
  - `url` - 下载链接

### Button - 按钮

通用按钮组件。

**Props:**

- `icon` - 图标名称 `'send' | 'image' | 'mic' | 'attach'`
- `variant` - 样式变体 `'primary' | 'secondary' | 'text'`
- `size` - 尺寸 `'sm' | 'md' | 'lg'`
- `disabled` - 是否禁用

## 消息类型

### 文本消息

```javascript
{
  id: '1',
  type: 'text',
  content: '你好！',
  position: 'left',
  status: 'sent',
  user: {
    avatar: 'https://...',
    name: 'Agent'
  }
}
```

### 思考消息

```javascript
{
  id: '2',
  type: 'thinking',
  content: '正在分析你的问题...',
  position: 'left'
}
```

### 打字中

```javascript
{
  id: '3',
  type: 'typing',
  position: 'left'
}
```

### 卡片消息

```javascript
{
  id: '4',
  type: 'card',
  position: 'left',
  card: {
    title: '推荐内容',
    content: '这是内容...',
    actions: [
      { text: '查看', value: 'view' },
      { text: '分享', value: 'share' }
    ]
  }
}
```

### 文件消息

```javascript
{
  id: '5',
  type: 'file',
  position: 'left',
  file: {
    name: 'document.pdf',
    size: 1024000,
    url: 'https://...'
  }
}
```

## 高级用法

### 自定义工具栏

```vue
<Chat
  :toolbar="[
    { icon: 'image', onClick: handleImageUpload },
    { icon: 'attach', onClick: handleFileUpload },
    { icon: 'mic', onClick: handleVoiceInput },
  ]"
/>
```

### 快捷回复

```vue
<Chat
  :quick-replies="[
    { name: '帮我写文章', value: 'write', icon: '✍️' },
    { name: '分析代码', value: 'analyze', icon: '🔍' },
    { name: '生成工作流', value: 'workflow', icon: '⚙️' },
  ]"
  @quick-reply="handleQuickReply"
/>
```

### 流式响应

```javascript
const handleStreamResponse = async (message) => {
  // 添加思考消息
  const thinkingId = addThinkingMessage();

  try {
    // 流式接收响应
    for await (const chunk of streamChat(message)) {
      updateMessage(thinkingId, chunk);
    }
  } finally {
    removeThinkingMessage(thinkingId);
  }
};
```

## 样式定制

所有组件使用 Tailwind CSS，支持深色模式。

### 自定义主题

在 `tailwind.config.js` 中配置：

```javascript
module.exports = {
  theme: {
    extend: {
      colors: {
        primary: "#3b82f6",
        secondary: "#64748b",
      },
    },
  },
};
```

## 与 Aster Agent 集成

```vue
<template>
  <Chat :messages="messages" @send="handleSend" />
</template>

<script setup>
import { useAsterClient } from "@/composables/useAsterClient";
import { Chat } from "@/components/ChatUI";

const { client } = useAsterClient();

const handleSend = async (message) => {
  // 添加用户消息
  addUserMessage(message);

  // 显示思考状态
  const thinkingId = addThinkingMessage();

  try {
    // 调用 Agent
    const response = await client.agents.chat(agentId, message.content);

    // 移除思考消息
    removeMessage(thinkingId);

    // 添加 Agent 回复
    addAgentMessage(response.data.text);
  } catch (error) {
    showError(error);
  }
};
</script>
```

## 最佳实践

1. **消息 ID** - 使用唯一 ID 标识每条消息
2. **状态管理** - 使用 Vue 的响应式系统管理消息列表
3. **错误处理** - 显示友好的错误消息
4. **加载状态** - 使用思考气泡或打字指示器
5. **自动滚动** - 新消息自动滚动到底部
6. **快捷回复** - 提供常用操作的快捷入口
7. **无障碍** - 支持键盘导航和屏幕阅读器

## 示例项目

查看 `ui/src/views/AgentChatUIDemo.vue` 获取完整示例。
